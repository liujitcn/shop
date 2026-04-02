package biz

import (
	"context"
	"fmt"
	"shop/service/admin/dto"
	"sort"
	"time"

	"shop/api/gen/go/admin"
	"shop/pkg/biz"
	"shop/pkg/gen/models"

	"github.com/liujitcn/gorm-kit/repo"
)

// AnalyticsCase 数据分析实例
type AnalyticsCase struct {
	*biz.BaseCase
	baseUserCase      *BaseUserCase
	goodsCase         *GoodsCase
	goodsCategoryCase *GoodsCategoryCase
	orderCase         *OrderCase
	orderGoodsCase    *OrderGoodsCase
	baseDictCase      *BaseDictCase
	baseDictItemCase  *BaseDictItemCase
}

// NewAnalyticsCase 创建数据分析实例
func NewAnalyticsCase(baseCase *biz.BaseCase, baseUserCase *BaseUserCase, goodsCase *GoodsCase, goodsCategoryCase *GoodsCategoryCase, orderCase *OrderCase, orderGoodsCase *OrderGoodsCase, baseDictCase *BaseDictCase, baseDictItemCase *BaseDictItemCase) *AnalyticsCase {
	return &AnalyticsCase{
		BaseCase:          baseCase,
		baseUserCase:      baseUserCase,
		goodsCase:         goodsCase,
		goodsCategoryCase: goodsCategoryCase,
		orderCase:         orderCase,
		orderGoodsCase:    orderGoodsCase,
		baseDictCase:      baseDictCase,
		baseDictItemCase:  baseDictItemCase,
	}
}

// AnalyticsCountUser 查询用户汇总
func (c *AnalyticsCase) AnalyticsCountUser(ctx context.Context, req *admin.AnalyticsCountRequest) (*admin.AnalyticsCountResponse, error) {
	startAt, endAt := getAnalyticsTimeRange(req.GetTimeType())
	var result dto.CountResult
	// 优化：将两次查询合并为一次，使用条件聚合减少数据库往返次数
	err := c.baseUserCase.Query(ctx).BaseUser.WithContext(ctx).UnderlyingDB().
		Model(&models.BaseUser{}).
		Select(`
			SUM(CASE WHEN created_at >= ? AND created_at < ? THEN 1 ELSE 0 END) AS new_num,
			COUNT(*) AS total_num
		`, startAt, endAt).
		Scan(&result).Error
	if err != nil {
		return nil, err
	}
	return &admin.AnalyticsCountResponse{NewNum: result.NewNum, TotalNum: result.TotalNum}, nil
}

// AnalyticsCountGoods 查询商品汇总
func (c *AnalyticsCase) AnalyticsCountGoods(ctx context.Context, req *admin.AnalyticsCountRequest) (*admin.AnalyticsCountResponse, error) {
	startAt, endAt := getAnalyticsTimeRange(req.GetTimeType())
	var result dto.CountResult
	// 优化：将两次查询合并为一次，使用条件聚合减少数据库往返次数
	err := c.goodsCase.Query(ctx).Goods.WithContext(ctx).UnderlyingDB().
		Model(&models.Goods{}).
		Select(`
			SUM(CASE WHEN created_at >= ? AND created_at < ? THEN 1 ELSE 0 END) AS new_num,
			COUNT(*) AS total_num
		`, startAt, endAt).
		Scan(&result).Error
	if err != nil {
		return nil, err
	}
	return &admin.AnalyticsCountResponse{NewNum: result.NewNum, TotalNum: result.TotalNum}, nil
}

// AnalyticsCountOrder 查询订单汇总
func (c *AnalyticsCase) AnalyticsCountOrder(ctx context.Context, req *admin.AnalyticsCountRequest) (*admin.AnalyticsCountResponse, error) {
	startAt, endAt := getAnalyticsTimeRange(req.GetTimeType())
	var result dto.CountResult
	// 优化：将两次查询合并为一次，使用条件聚合减少数据库往返次数
	err := c.orderCase.Query(ctx).Order.WithContext(ctx).UnderlyingDB().
		Model(&models.Order{}).
		Select(`
			SUM(CASE WHEN created_at >= ? AND created_at < ? THEN 1 ELSE 0 END) AS new_num,
			COUNT(*) AS total_num
		`, startAt, endAt).
		Scan(&result).Error
	if err != nil {
		return nil, err
	}
	return &admin.AnalyticsCountResponse{NewNum: result.NewNum, TotalNum: result.TotalNum}, nil
}

// AnalyticsCountSale 查询销售汇总
func (c *AnalyticsCase) AnalyticsCountSale(ctx context.Context, req *admin.AnalyticsCountRequest) (*admin.AnalyticsCountResponse, error) {
	startAt, endAt := getAnalyticsTimeRange(req.GetTimeType())
	var result dto.CountResult
	// 优化：将两次查询合并为一次，使用条件聚合减少数据库往返次数
	err := c.orderCase.Query(ctx).Order.WithContext(ctx).UnderlyingDB().
		Model(&models.Order{}).
		Select(`
			COALESCE(SUM(CASE WHEN created_at >= ? AND created_at < ? THEN pay_money ELSE NULL END), 0) AS new_num,
			COALESCE(SUM(pay_money), 0) AS total_num
		`, startAt, endAt).
		Scan(&result).Error
	if err != nil {
		return nil, err
	}
	return &admin.AnalyticsCountResponse{NewNum: result.NewNum, TotalNum: result.TotalNum}, nil
}

// AnalyticsBarOrder 查询订单柱状图
// 返回 seriesData 顺序固定为：订单量、订单量增长率。
func (c *AnalyticsCase) AnalyticsBarOrder(ctx context.Context, req *admin.AnalyticsBarOrderRequest) (*admin.AnalyticsBarResponse, error) {
	startAt, endAt := getAnalyticsTimeRange(req.GetTimeType())
	summary, axisData, err := c.queryOrderSummary(ctx, req.GetTimeType(), startAt, endAt)
	if err != nil {
		return nil, err
	}

	orderCountRow := make([]int64, 0, len(axisData))
	orderCountRateRow := make([]int64, 0, len(axisData))
	for i := range axisData {
		key := int64(i + 1)
		item, ok := summary[key]
		if ok {
			orderCountRow = append(orderCountRow, item.OrderCount)
		} else {
			orderCountRow = append(orderCountRow, 0)
		}
		if i == 0 {
			orderCountRateRow = append(orderCountRateRow, calcGrowthRate(0, orderCountRow[i]))
		} else {
			orderCountRateRow = append(orderCountRateRow, calcGrowthRate(orderCountRow[i-1], orderCountRow[i]))
		}
	}

	return &admin.AnalyticsBarResponse{
		AxisData: axisData,
		SeriesData: []*admin.AnalyticsBarResponse_SeriesData{
			{Value: orderCountRow},
			{Value: orderCountRateRow},
		},
	}, nil
}

// AnalyticsBarSale 查询订单销售额柱状图
// 返回 seriesData 顺序固定为：销售额、销售额增长率。
func (c *AnalyticsCase) AnalyticsBarSale(ctx context.Context, req *admin.AnalyticsBarSaleRequest) (*admin.AnalyticsBarResponse, error) {
	startAt, endAt := getAnalyticsTimeRange(req.GetTimeType())
	summary, axisData, err := c.queryOrderSummary(ctx, req.GetTimeType(), startAt, endAt)
	if err != nil {
		return nil, err
	}

	saleAmountRow := make([]int64, 0, len(axisData))
	saleAmountRateRow := make([]int64, 0, len(axisData))
	for i := range axisData {
		key := int64(i + 1)
		item, ok := summary[key]
		if ok {
			saleAmountRow = append(saleAmountRow, item.SaleAmount)
		} else {
			saleAmountRow = append(saleAmountRow, 0)
		}
		if i == 0 {
			saleAmountRateRow = append(saleAmountRateRow, calcGrowthRate(0, saleAmountRow[i]))
		} else {
			saleAmountRateRow = append(saleAmountRateRow, calcGrowthRate(saleAmountRow[i-1], saleAmountRow[i]))
		}
	}

	return &admin.AnalyticsBarResponse{
		AxisData: axisData,
		SeriesData: []*admin.AnalyticsBarResponse_SeriesData{
			{Value: saleAmountRow},
			{Value: saleAmountRateRow},
		},
	}, nil
}

// AnalyticsPieGoods 查询商品饼图
// 按时间范围统计已下单商品的一级分类销量占比，便于与顶部时间筛选保持一致。
func (c *AnalyticsCase) AnalyticsPieGoods(ctx context.Context, req *admin.AnalyticsPieGoodsRequest) (*admin.AnalyticsPieResponse, error) {
	startAt, endAt := getAnalyticsTimeRange(req.GetTimeType())
	summary, err := c.queryGoodsCategorySummary(ctx, startAt, endAt)
	if err != nil {
		return nil, err
	}

	// 获取一级分类的名称映射
	parentId := int64(0)
	categoryNameMap := c.goodsCategoryCase.NameMap(ctx, &parentId)

	seriesData := make([]*admin.AnalyticsPieResponse_SeriesData, 0, len(summary))
	for _, item := range summary {
		seriesData = append(seriesData, &admin.AnalyticsPieResponse_SeriesData{
			Value: item.GoodsCount,
			Name:  categoryNameMap[item.CategoryId],
		})
	}
	return &admin.AnalyticsPieResponse{SeriesData: seriesData}, nil
}

// AnalyticsPieOrder 查询订单状态分布（饼状图）。
// 按订单状态统计订单数量，用于展示各状态订单的占比分布。
func (c *AnalyticsCase) AnalyticsPieOrder(ctx context.Context, req *admin.AnalyticsPieOrderRequest) (*admin.AnalyticsPieResponse, error) {
	startAt, endAt := getAnalyticsTimeRange(req.GetTimeType())

	// 查询订单状态统计数据
	summary, err := c.queryOrderStatusSummary(ctx, startAt, endAt)
	if err != nil {
		return nil, err
	}

	// 获取订单状态字典映射（value -> label）
	statusLabelMap, err := c.getOrderStatusLabelMap(ctx)
	if err != nil {
		return nil, err
	}

	// 组装饼图数据
	seriesData := make([]*admin.AnalyticsPieResponse_SeriesData, 0, len(summary))
	for _, item := range summary {
		label, ok := statusLabelMap[item.Status]
		if !ok {
			label = fmt.Sprintf("状态%d", item.Status)
		}
		seriesData = append(seriesData, &admin.AnalyticsPieResponse_SeriesData{
			Value: item.OrderCount,
			Name:  label,
		})
	}

	return &admin.AnalyticsPieResponse{SeriesData: seriesData}, nil
}

// getOrderStatusLabelMap 获取订单状态字典映射（value -> label）。
func (c *AnalyticsCase) getOrderStatusLabelMap(ctx context.Context) (map[int32]string, error) {
	dictQuery := c.baseDictCase.Query(ctx).BaseDict
	baseDict, err := c.baseDictCase.Find(ctx, repo.Where(dictQuery.Code.Eq("order_status")))
	if err != nil {
		return nil, err
	}

	dictItemQuery := c.baseDictItemCase.Query(ctx).BaseDictItem
	dictItemOpts := make([]repo.QueryOption, 0, 1)
	dictItemOpts = append(dictItemOpts, repo.Where(dictItemQuery.DictID.Eq(baseDict.ID)))
	baseDictItemList, err := c.baseDictItemCase.List(ctx, dictItemOpts...)
	if err != nil {
		return nil, err
	}

	statusLabelMap := make(map[int32]string, len(baseDictItemList))
	for _, item := range baseDictItemList {
		// 将字符串 value 转换为 int32
		var statusValue int32
		_, err = fmt.Sscanf(item.Value, "%d", &statusValue)
		if err != nil {
			continue
		}
		statusLabelMap[statusValue] = item.Label
	}

	return statusLabelMap, nil
}

// getAnalyticsTimeRange 获取统计时间范围
// WEEK=本周，MONTH=本月，YEAR=本年。
func getAnalyticsTimeRange(timeType admin.AnalyticsTimeType) (time.Time, time.Time) {
	now := time.Now()
	switch timeType {
	case admin.AnalyticsTimeType_YEAR:
		startAt := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		return startAt, startAt.AddDate(1, 0, 0)
	case admin.AnalyticsTimeType_MONTH:
		startAt := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		return startAt, startAt.AddDate(0, 1, 0)
	case admin.AnalyticsTimeType_WEEK:
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		startAt := time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, now.Location())
		return startAt, startAt.AddDate(0, 0, 7)
	default:
		// 默认本周
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		startAt := time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, now.Location())
		return startAt, startAt.AddDate(0, 0, 7)
	}
}

// formatAnalyticsAxis 格式化坐标轴
// WEEK 返回星期，MONTH 返回当月日期，YEAR 返回月份。
func formatAnalyticsAxis(timeType admin.AnalyticsTimeType, index int, startAt time.Time) string {
	switch timeType {
	case admin.AnalyticsTimeType_YEAR:
		months := []string{"1月", "2月", "3月", "4月", "5月", "6月", "7月", "8月", "9月", "10月", "11月", "12月"}
		if index < len(months) {
			return months[index]
		}
		return ""
	case admin.AnalyticsTimeType_MONTH:
		return startAt.AddDate(0, 0, index).Format("01-02")
	case admin.AnalyticsTimeType_WEEK:
		labels := []string{"一", "二", "三", "四", "五", "六", "日"}
		return "周" + labels[index]
	default:
		// 默认本周
		labels := []string{"一", "二", "三", "四", "五", "六", "日"}
		return "周" + labels[index]
	}
}

// calcGrowthRate 计算增长率
func calcGrowthRate(prev, curr int64) int64 {
	if prev == 0 {
		if curr == 0 {
			return 0
		}
		return 100
	}
	return (curr - prev) * 100 / prev
}

// queryOrderSummary 查询订单统计
// WEEK 按星期，MONTH 按日期，YEAR 按月份聚合订单量与销售额。
func (c *AnalyticsCase) queryOrderSummary(ctx context.Context, timeType admin.AnalyticsTimeType, startAt, endAt time.Time) (map[int64]*dto.OrderSummary, []string, error) {
	summaryMap := make(map[int64]*dto.OrderSummary)
	axisData := make([]string, 0)
	db := c.orderCase.Query(ctx).Order.WithContext(ctx).UnderlyingDB()

	var err error
	switch timeType {
	case admin.AnalyticsTimeType_YEAR:
		var rows []*dto.OrderSummary
		err = db.Model(&models.Order{}).
			Select("MONTH(created_at) AS `key`, COUNT(*) AS order_count, COALESCE(SUM(pay_money),0) AS sale_amount").
			Where("created_at >= ? AND created_at < ?", startAt, endAt).
			Group("MONTH(created_at)").
			Scan(&rows).Error
		if err != nil {
			return nil, nil, err
		}
		for _, item := range rows {
			summaryMap[item.Key] = item
		}
		for i := 0; i < 12; i++ {
			axisData = append(axisData, formatAnalyticsAxis(timeType, i, startAt))
		}
	case admin.AnalyticsTimeType_MONTH:
		var rows []*dto.OrderSummary
		// 修复：正确处理查询错误
		err = db.Model(&models.Order{}).
			Select("DAY(created_at) AS `key`, COUNT(*) AS order_count, COALESCE(SUM(pay_money),0) AS sale_amount").
			Where("created_at >= ? AND created_at < ?", startAt, endAt).
			Group("DAY(created_at)").
			Scan(&rows).Error
		if err != nil {
			return nil, nil, err
		}
		for _, item := range rows {
			summaryMap[item.Key] = item
		}
		monthDays := endAt.AddDate(0, 0, -1).Day()
		for i := 0; i < monthDays; i++ {
			axisData = append(axisData, formatAnalyticsAxis(timeType, i, startAt))
		}
	case admin.AnalyticsTimeType_WEEK:
		var rows []*dto.OrderSummary
		// 修复：正确处理查询错误
		err = db.Model(&models.Order{}).
			Select("WEEKDAY(created_at)+1 AS `key`, COUNT(*) AS order_count, COALESCE(SUM(pay_money),0) AS sale_amount").
			Where("created_at >= ? AND created_at < ?", startAt, endAt).
			Group("WEEKDAY(created_at)+1").
			Scan(&rows).Error
		if err != nil {
			return nil, nil, err
		}
		for _, item := range rows {
			summaryMap[item.Key] = item
		}
		for i := 0; i < 7; i++ {
			axisData = append(axisData, formatAnalyticsAxis(timeType, i, startAt))
		}
	default:
		// 默认本周
		var rows []*dto.OrderSummary
		err = db.Model(&models.Order{}).
			Select("WEEKDAY(created_at)+1 AS `key`, COUNT(*) AS order_count, COALESCE(SUM(pay_money),0) AS sale_amount").
			Where("created_at >= ? AND created_at < ?", startAt, endAt).
			Group("WEEKDAY(created_at)+1").
			Scan(&rows).Error
		if err != nil {
			return nil, nil, err
		}
		for _, item := range rows {
			summaryMap[item.Key] = item
		}
		for i := 0; i < 7; i++ {
			axisData = append(axisData, formatAnalyticsAxis(timeType, i, startAt))
		}
	}
	return summaryMap, axisData, nil
}

// queryOrderGoodsSummary 查询商品销量统计
func (c *AnalyticsCase) queryOrderGoodsSummary(ctx context.Context, top int64, startAt, endAt time.Time) ([]*dto.OrderGoodsSummary, error) {
	if top <= 0 {
		top = 10
	}
	res := make([]*dto.OrderGoodsSummary, 0)
	err := c.orderGoodsCase.Query(ctx).OrderGoods.WithContext(ctx).UnderlyingDB().
		Model(&models.OrderGoods{}).
		Select("goods_id, COALESCE(SUM(num),0) AS goods_count").
		Joins("JOIN `order` ON `order`.id = order_goods.order_id").
		Where("`order`.created_at >= ? AND `order`.created_at < ?", startAt, endAt).
		Group("goods_id").
		Order("goods_count DESC").
		Limit(int(top)).
		Scan(&res).Error
	return res, err
}

// queryGoodsCategorySummary 查询指定时间范围内的一级商品分类销量统计
// 统计口径基于订单商品数量，将子分类销量汇总到对应的一级分类下。
func (c *AnalyticsCase) queryGoodsCategorySummary(ctx context.Context, startAt, endAt time.Time) ([]*dto.GoodsCategorySummary, error) {
	// 查询所有分类，建立分类层级映射
	categoryList, err := c.goodsCategoryCase.List(ctx)
	if err != nil {
		return nil, err
	}

	// 建立分类ID到父级ID的映射，以及找到每个分类的根分类（一级分类）
	parentMap := make(map[int64]int64, len(categoryList))
	for _, category := range categoryList {
		parentMap[category.ID] = category.ParentID
	}

	// 获取一级分类ID的函数：递归查找根分类
	getRootCategoryId := func(categoryId int64) int64 {
		for {
			parentId, ok := parentMap[categoryId]
			if !ok || parentId == 0 {
				return categoryId
			}
			categoryId = parentId
		}
	}

	// 查询所有商品分类的销量数据（包含子分类）
	rows := make([]*dto.GoodsCategorySummary, 0)
	err = c.orderGoodsCase.Query(ctx).OrderGoods.WithContext(ctx).UnderlyingDB().
		Model(&models.OrderGoods{}).
		Select("goods.category_id, COALESCE(SUM(order_goods.num),0) AS goods_count").
		Joins("JOIN goods ON goods.id = order_goods.goods_id").
		Joins("JOIN `order` ON `order`.id = order_goods.order_id").
		Where("`order`.created_at >= ? AND `order`.created_at < ?", startAt, endAt).
		Group("goods.category_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	// 将子分类的销量汇总到一级分类
	rootCategoryCount := make(map[int64]int64)
	for _, row := range rows {
		rootId := getRootCategoryId(row.CategoryId)
		rootCategoryCount[rootId] += row.GoodsCount
	}

	// 转换为结果数组
	res := make([]*dto.GoodsCategorySummary, 0, len(rootCategoryCount))
	for categoryId, count := range rootCategoryCount {
		res = append(res, &dto.GoodsCategorySummary{
			CategoryId: categoryId,
			GoodsCount: count,
		})
	}

	// 按销量降序排序
	sort.Slice(res, func(i, j int) bool {
		return res[i].GoodsCount > res[j].GoodsCount
	})

	return res, nil
}

// queryOrderGoodsStatusSummary 查询商品订单状态统计
func (c *AnalyticsCase) queryOrderGoodsStatusSummary(ctx context.Context, startAt, endAt time.Time) ([]*dto.OrderGoodsStatusSummary, error) {
	res := make([]*dto.OrderGoodsStatusSummary, 0)
	err := c.orderGoodsCase.Query(ctx).OrderGoods.WithContext(ctx).UnderlyingDB().
		Model(&models.OrderGoods{}).
		Select("goods.category_id, `order`.status, COALESCE(SUM(order_goods.num),0) AS goods_count").
		Joins("JOIN goods ON goods.id = order_goods.goods_id").
		Joins("JOIN `order` ON `order`.id = order_goods.order_id").
		Where("`order`.created_at >= ? AND `order`.created_at < ?", startAt, endAt).
		Group("goods.category_id, `order`.status").
		Scan(&res).Error
	return res, err
}

// queryOrderStatusSummary 查询订单状态统计
func (c *AnalyticsCase) queryOrderStatusSummary(ctx context.Context, startAt, endAt time.Time) ([]*dto.OrderStatusSummary, error) {
	res := make([]*dto.OrderStatusSummary, 0)
	err := c.orderCase.Query(ctx).Order.WithContext(ctx).UnderlyingDB().
		Model(&models.Order{}).
		Select("status, COUNT(*) AS order_count").
		Where("created_at >= ? AND created_at < ?", startAt, endAt).
		Group("status").
		Scan(&res).Error
	return res, err
}
