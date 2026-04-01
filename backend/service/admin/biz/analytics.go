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
// 返回 seriesData 顺序固定为：订单量、销售额、订单量增长率、销售额增长率。
func (c *AnalyticsCase) AnalyticsBarOrder(ctx context.Context, req *admin.AnalyticsBarOrderRequest) (*admin.AnalyticsBarResponse, error) {
	startAt, endAt := getAnalyticsTimeRange(req.GetTimeType())
	summary, axisData, err := c.queryOrderSummary(ctx, req.GetTimeType(), startAt, endAt)
	if err != nil {
		return nil, err
	}

	orderCountRow := make([]int64, 0, len(axisData))
	saleAmountRow := make([]int64, 0, len(axisData))
	orderCountRateRow := make([]int64, 0, len(axisData))
	saleAmountRateRow := make([]int64, 0, len(axisData))
	for i := range axisData {
		key := int64(i + 1)
		item, ok := summary[key]
		if ok {
			orderCountRow = append(orderCountRow, item.OrderCount)
			saleAmountRow = append(saleAmountRow, item.SaleAmount)
		} else {
			orderCountRow = append(orderCountRow, 0)
			saleAmountRow = append(saleAmountRow, 0)
		}
		if i == 0 {
			orderCountRateRow = append(orderCountRateRow, calcGrowthRate(0, orderCountRow[i]))
			saleAmountRateRow = append(saleAmountRateRow, calcGrowthRate(0, saleAmountRow[i]))
		} else {
			orderCountRateRow = append(orderCountRateRow, calcGrowthRate(orderCountRow[i-1], orderCountRow[i]))
			saleAmountRateRow = append(saleAmountRateRow, calcGrowthRate(saleAmountRow[i-1], saleAmountRow[i]))
		}
	}

	return &admin.AnalyticsBarResponse{
		AxisData: axisData,
		SeriesData: []*admin.AnalyticsBarResponse_SeriesData{
			{Value: orderCountRow},
			{Value: saleAmountRow},
			{Value: orderCountRateRow},
			{Value: saleAmountRateRow},
		},
	}, nil
}

// AnalyticsBarGoods 查询商品柱状图
func (c *AnalyticsCase) AnalyticsBarGoods(ctx context.Context, req *admin.AnalyticsBarGoodsRequest) (*admin.AnalyticsBarResponse, error) {
	startAt, endAt := getAnalyticsTimeRange(req.GetTimeType())
	summary, err := c.queryOrderGoodsSummary(ctx, req.GetTop(), startAt, endAt)
	if err != nil {
		return nil, err
	}

	goodsIds := make([]int64, 0, len(summary))
	for _, item := range summary {
		goodsIds = append(goodsIds, item.GoodsId)
	}

	goodsMap := make(map[int64]string)
	if len(goodsIds) > 0 {
		goodsList, listErr := c.goodsCase.ListByIds(ctx, goodsIds)
		if listErr != nil {
			return nil, listErr
		}
		for _, item := range goodsList {
			goodsMap[item.ID] = item.Name
		}
	}

	sort.Slice(summary, func(i, j int) bool { return summary[i].GoodsCount > summary[j].GoodsCount })
	axisData := make([]string, 0, len(summary))
	goodsCountRow := make([]int64, 0, len(summary))
	for _, item := range summary {
		axisData = append(axisData, goodsMap[item.GoodsId])
		goodsCountRow = append(goodsCountRow, item.GoodsCount)
	}
	return &admin.AnalyticsBarResponse{
		AxisData:   axisData,
		SeriesData: []*admin.AnalyticsBarResponse_SeriesData{{Value: goodsCountRow}},
	}, nil
}

// AnalyticsPieGoods 查询商品饼图
// 按时间范围统计已下单商品的分类销量占比，便于与顶部时间筛选保持一致。
func (c *AnalyticsCase) AnalyticsPieGoods(ctx context.Context, req *admin.AnalyticsPieGoodsRequest) (*admin.AnalyticsPieResponse, error) {
	startAt, endAt := getAnalyticsTimeRange(req.GetTimeType())
	summary, err := c.queryGoodsCategorySummary(ctx, startAt, endAt)
	if err != nil {
		return nil, err
	}
	nameMap := c.goodsCategoryCase.NameMap(ctx, new(int64(0)))

	seriesData := make([]*admin.AnalyticsPieResponse_SeriesData, 0, len(summary))
	for _, item := range summary {
		seriesData = append(seriesData, &admin.AnalyticsPieResponse_SeriesData{
			Value: item.GoodsCount,
			Name:  nameMap[item.CategoryId],
		})
	}
	return &admin.AnalyticsPieResponse{SeriesData: seriesData}, nil
}

// AnalyticsRadarOrder 查询订单雷达图
// 图例为订单状态，指示器为商品分类，数值为对应分类下该状态的商品销量。
func (c *AnalyticsCase) AnalyticsRadarOrder(ctx context.Context, req *admin.AnalyticsRadarOrderRequest) (*admin.AnalyticsRadarResponse, error) {
	startAt, endAt := getAnalyticsTimeRange(req.GetTimeType())
	summary, err := c.queryOrderGoodsStatusSummary(ctx, startAt, endAt)
	if err != nil {
		return nil, err
	}

	summaryMap := make(map[string]int64, len(summary))
	for _, item := range summary {
		summaryMap[fmt.Sprintf("%d_%d", item.CategoryId, item.Status)] = item.GoodsCount
	}

	parentId := int64(0)
	goodsCategoryNameMap := c.goodsCategoryCase.NameMap(ctx, &parentId)

	dictQuery := c.baseDictCase.Query(ctx).BaseDict
	var baseDict *models.BaseDict
	baseDict, err = c.baseDictCase.Find(ctx, repo.Where(dictQuery.Code.Eq("order_status")))
	if err != nil {
		return &admin.AnalyticsRadarResponse{}, nil
	}

	dictItemQuery := c.baseDictItemCase.Query(ctx).BaseDictItem
	var baseDictItemList []*models.BaseDictItem
	dictItemOpts := make([]repo.QueryOption, 0, 1)
	dictItemOpts = append(dictItemOpts, repo.Where(dictItemQuery.DictID.Eq(baseDict.ID)))
	baseDictItemList, err = c.baseDictItemCase.List(ctx, dictItemOpts...)
	if err != nil || len(baseDictItemList) == 0 || len(goodsCategoryNameMap) == 0 {
		return &admin.AnalyticsRadarResponse{}, nil
	}
	// 按字典排序值固定图例与雷达数据顺序，避免前端展示顺序漂移。
	sort.Slice(baseDictItemList, func(i, j int) bool {
		if baseDictItemList[i].Sort == baseDictItemList[j].Sort {
			return baseDictItemList[i].ID < baseDictItemList[j].ID
		}
		return baseDictItemList[i].Sort < baseDictItemList[j].Sort
	})

	categoryIds := make([]int64, 0, len(goodsCategoryNameMap))
	for id := range goodsCategoryNameMap {
		categoryIds = append(categoryIds, id)
	}
	sort.Slice(categoryIds, func(i, j int) bool { return categoryIds[i] < categoryIds[j] })

	legendData := make([]string, 0, len(baseDictItemList))
	radarIndicator := make([]*admin.AnalyticsRadarResponse_RadarIndicator, 0, len(categoryIds))
	seriesData := make([]*admin.AnalyticsRadarResponse_SeriesData, 0, len(baseDictItemList))
	for idx, item := range baseDictItemList {
		legendData = append(legendData, item.Label)
		goodsNum := make([]int64, 0, len(categoryIds))
		for _, categoryId := range categoryIds {
			if idx == 0 {
				radarIndicator = append(radarIndicator, &admin.AnalyticsRadarResponse_RadarIndicator{
					Name: goodsCategoryNameMap[categoryId],
				})
			}
			key := fmt.Sprintf("%d_%s", categoryId, item.Value)
			goodsNum = append(goodsNum, summaryMap[key])
		}
		seriesData = append(seriesData, &admin.AnalyticsRadarResponse_SeriesData{
			Name:  item.Label,
			Value: goodsNum,
		})
	}

	return &admin.AnalyticsRadarResponse{
		LegendData:     legendData,
		RadarIndicator: radarIndicator,
		SeriesData:     seriesData,
	}, nil
}

// getAnalyticsTimeRange 获取统计时间范围
// DAY=今日，WEEK=本周，MONTH=本月。
func getAnalyticsTimeRange(timeType admin.AnalyticsTimeType) (time.Time, time.Time) {
	now := time.Now()
	switch timeType {
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
		startAt := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		return startAt, startAt.AddDate(0, 0, 1)
	}
}

// formatAnalyticsAxis 格式化坐标轴
// DAY 返回小时，WEEK 返回星期，MONTH 返回当月日期。
func formatAnalyticsAxis(timeType admin.AnalyticsTimeType, index int, startAt time.Time) string {
	switch timeType {
	case admin.AnalyticsTimeType_MONTH:
		return startAt.AddDate(0, 0, index).Format("01-02")
	case admin.AnalyticsTimeType_WEEK:
		labels := []string{"一", "二", "三", "四", "五", "六", "日"}
		return "周" + labels[index]
	default:
		return fmt.Sprintf("%02d:00", index)
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
// DAY 按小时，WEEK 按星期，MONTH 按日期聚合订单量与销售额。
func (c *AnalyticsCase) queryOrderSummary(ctx context.Context, timeType admin.AnalyticsTimeType, startAt, endAt time.Time) (map[int64]*dto.OrderSummary, []string, error) {
	summaryMap := make(map[int64]*dto.OrderSummary)
	axisData := make([]string, 0)
	db := c.orderCase.Query(ctx).Order.WithContext(ctx).UnderlyingDB()

	var err error
	switch timeType {
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
		var rows []*dto.OrderSummary
		// 修复：正确处理查询错误
		err = db.Model(&models.Order{}).
			Select("HOUR(created_at)+1 AS `key`, COUNT(*) AS order_count, COALESCE(SUM(pay_money),0) AS sale_amount").
			Where("created_at >= ? AND created_at < ?", startAt, endAt).
			Group("HOUR(created_at)+1").
			Scan(&rows).Error
		if err != nil {
			return nil, nil, err
		}
		for _, item := range rows {
			summaryMap[item.Key] = item
		}
		for i := 0; i < 24; i++ {
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

// queryGoodsCategorySummary 查询指定时间范围内的商品分类销量统计
// 统计口径基于订单商品数量，而不是商品表中的累计库存或总商品数。
func (c *AnalyticsCase) queryGoodsCategorySummary(ctx context.Context, startAt, endAt time.Time) ([]*dto.GoodsCategorySummary, error) {
	res := make([]*dto.GoodsCategorySummary, 0)
	err := c.orderGoodsCase.Query(ctx).OrderGoods.WithContext(ctx).UnderlyingDB().
		Model(&models.OrderGoods{}).
		Select("goods.category_id, COALESCE(SUM(order_goods.num),0) AS goods_count").
		Joins("JOIN goods ON goods.id = order_goods.goods_id").
		Joins("JOIN `order` ON `order`.id = order_goods.order_id").
		Where("`order`.created_at >= ? AND `order`.created_at < ?", startAt, endAt).
		Group("goods.category_id").
		Order("goods_count DESC").
		Scan(&res).Error
	return res, err
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
