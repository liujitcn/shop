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

// DashboardCase 首页业务实例
type DashboardCase struct {
	*biz.BaseCase
	baseUserCase      *BaseUserCase
	goodsCase         *GoodsCase
	goodsCategoryCase *GoodsCategoryCase
	orderCase         *OrderCase
	orderGoodsCase    *OrderGoodsCase
	baseDictCase      *BaseDictCase
	baseDictItemCase  *BaseDictItemCase
}

// NewDashboardCase 创建首页业务实例
func NewDashboardCase(baseCase *biz.BaseCase, baseUserCase *BaseUserCase, goodsCase *GoodsCase, goodsCategoryCase *GoodsCategoryCase, orderCase *OrderCase, orderGoodsCase *OrderGoodsCase, baseDictCase *BaseDictCase, baseDictItemCase *BaseDictItemCase) *DashboardCase {
	return &DashboardCase{
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

// DashboardCountUser 查询用户汇总
func (c *DashboardCase) DashboardCountUser(ctx context.Context, req *admin.DashboardCountRequest) (*admin.DashboardCountResponse, error) {
	startAt, endAt := getDashboardTimeRange(req.GetTimeType())
	query := c.baseUserCase.Query(ctx).BaseUser
	newNum, err := c.baseUserCase.Count(ctx,
		repo.Where(query.CreatedAt.Gte(startAt)),
		repo.Where(query.CreatedAt.Lt(endAt)),
	)
	if err != nil {
		return nil, err
	}
	var totalNum int64
	totalNum, err = c.baseUserCase.Count(ctx)
	if err != nil {
		return nil, err
	}
	return &admin.DashboardCountResponse{NewNum: newNum, TotalNum: totalNum}, nil
}

// DashboardCountGoods 查询商品汇总
func (c *DashboardCase) DashboardCountGoods(ctx context.Context, req *admin.DashboardCountRequest) (*admin.DashboardCountResponse, error) {
	startAt, endAt := getDashboardTimeRange(req.GetTimeType())
	query := c.goodsCase.Query(ctx).Goods
	newNum, err := c.goodsCase.Count(ctx,
		repo.Where(query.CreatedAt.Gte(startAt)),
		repo.Where(query.CreatedAt.Lt(endAt)),
	)
	if err != nil {
		return nil, err
	}
	var totalNum int64
	totalNum, err = c.goodsCase.Count(ctx)
	if err != nil {
		return nil, err
	}
	return &admin.DashboardCountResponse{NewNum: newNum, TotalNum: totalNum}, nil
}

// DashboardCountOrder 查询订单汇总
func (c *DashboardCase) DashboardCountOrder(ctx context.Context, req *admin.DashboardCountRequest) (*admin.DashboardCountResponse, error) {
	startAt, endAt := getDashboardTimeRange(req.GetTimeType())
	query := c.orderCase.Query(ctx).Order
	newNum, err := c.orderCase.Count(ctx,
		repo.Where(query.CreatedAt.Gte(startAt)),
		repo.Where(query.CreatedAt.Lt(endAt)),
	)
	if err != nil {
		return nil, err
	}
	var totalNum int64
	totalNum, err = c.orderCase.Count(ctx)
	if err != nil {
		return nil, err
	}
	return &admin.DashboardCountResponse{NewNum: newNum, TotalNum: totalNum}, nil
}

// DashboardCountSale 查询销售汇总
func (c *DashboardCase) DashboardCountSale(ctx context.Context, req *admin.DashboardCountRequest) (*admin.DashboardCountResponse, error) {
	startAt, endAt := getDashboardTimeRange(req.GetTimeType())
	var newNum struct {
		Num int64 `gorm:"column:num"`
	}
	err := c.orderCase.Query(ctx).Order.WithContext(ctx).UnderlyingDB().
		Model(&models.Order{}).
		Select("COALESCE(SUM(pay_money),0) AS num").
		Where("created_at >= ? AND created_at < ?", startAt, endAt).
		Scan(&newNum).Error
	if err != nil {
		return nil, err
	}

	var totalNum struct {
		Num int64 `gorm:"column:num"`
	}
	err = c.orderCase.Query(ctx).Order.WithContext(ctx).UnderlyingDB().
		Model(&models.Order{}).
		Select("COALESCE(SUM(pay_money),0) AS num").
		Scan(&totalNum).Error
	if err != nil {
		return nil, err
	}
	return &admin.DashboardCountResponse{NewNum: newNum.Num, TotalNum: totalNum.Num}, nil
}

// DashboardBarOrder 查询订单柱状图
func (c *DashboardCase) DashboardBarOrder(ctx context.Context, req *admin.DashboardBarOrderRequest) (*admin.DashboardBarResponse, error) {
	startAt, endAt := getDashboardTimeRange(req.GetTimeType())
	summary, axisData := c.queryOrderSummary(ctx, req.GetTimeType(), startAt, endAt)

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

	return &admin.DashboardBarResponse{
		AxisData: axisData,
		SeriesData: []*admin.DashboardBarResponse_SeriesData{
			{Value: orderCountRow},
			{Value: saleAmountRow},
			{Value: orderCountRateRow},
			{Value: saleAmountRateRow},
		},
	}, nil
}

// DashboardBarGoods 查询商品柱状图
func (c *DashboardCase) DashboardBarGoods(ctx context.Context, req *admin.DashboardBarGoodsRequest) (*admin.DashboardBarResponse, error) {
	startAt, endAt := getDashboardTimeRange(req.GetTimeType())
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
	return &admin.DashboardBarResponse{
		AxisData:   axisData,
		SeriesData: []*admin.DashboardBarResponse_SeriesData{{Value: goodsCountRow}},
	}, nil
}

// DashboardPieGoods 查询商品饼图
func (c *DashboardCase) DashboardPieGoods(ctx context.Context, req *admin.DashboardPieGoodsRequest) (*admin.DashboardPieResponse, error) {
	summary, err := c.queryGoodsCategorySummary(ctx)
	if err != nil {
		return nil, err
	}
	nameMap := c.goodsCategoryCase.NameMap(ctx, new(int64(0)))

	seriesData := make([]*admin.DashboardPieResponse_SeriesData, 0, len(summary))
	for _, item := range summary {
		seriesData = append(seriesData, &admin.DashboardPieResponse_SeriesData{
			Value: item.GoodsCount,
			Name:  nameMap[item.CategoryId],
		})
	}
	return &admin.DashboardPieResponse{SeriesData: seriesData}, nil
}

// DashboardRadarOrder 查询订单雷达图
func (c *DashboardCase) DashboardRadarOrder(ctx context.Context, req *admin.DashboardRadarOrderRequest) (*admin.DashboardRadarResponse, error) {
	startAt, endAt := getDashboardTimeRange(req.GetTimeType())
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
		return &admin.DashboardRadarResponse{}, nil
	}

	dictItemQuery := c.baseDictItemCase.Query(ctx).BaseDictItem
	var baseDictItemList []*models.BaseDictItem
	dictItemOpts := make([]repo.QueryOption, 0, 1)
	dictItemOpts = append(dictItemOpts, repo.Where(dictItemQuery.DictID.Eq(baseDict.ID)))
	baseDictItemList, err = c.baseDictItemCase.List(ctx, dictItemOpts...)
	if err != nil || len(baseDictItemList) == 0 || len(goodsCategoryNameMap) == 0 {
		return &admin.DashboardRadarResponse{}, nil
	}

	categoryIds := make([]int64, 0, len(goodsCategoryNameMap))
	for id := range goodsCategoryNameMap {
		categoryIds = append(categoryIds, id)
	}
	sort.Slice(categoryIds, func(i, j int) bool { return categoryIds[i] < categoryIds[j] })

	legendData := make([]string, 0, len(baseDictItemList))
	radarIndicator := make([]*admin.DashboardRadarResponse_RadarIndicator, 0, len(categoryIds))
	seriesData := make([]*admin.DashboardRadarResponse_SeriesData, 0, len(baseDictItemList))
	for idx, item := range baseDictItemList {
		legendData = append(legendData, item.Label)
		goodsNum := make([]int64, 0, len(categoryIds))
		for _, categoryId := range categoryIds {
			if idx == 0 {
				radarIndicator = append(radarIndicator, &admin.DashboardRadarResponse_RadarIndicator{
					Name: goodsCategoryNameMap[categoryId],
				})
			}
			key := fmt.Sprintf("%d_%s", categoryId, item.Value)
			goodsNum = append(goodsNum, summaryMap[key])
		}
		seriesData = append(seriesData, &admin.DashboardRadarResponse_SeriesData{
			Name:  item.Label,
			Value: goodsNum,
		})
	}

	return &admin.DashboardRadarResponse{
		LegendData:     legendData,
		RadarIndicator: radarIndicator,
		SeriesData:     seriesData,
	}, nil
}

// getDashboardTimeRange 获取统计时间范围
func getDashboardTimeRange(timeType admin.DashboardTimeType) (time.Time, time.Time) {
	now := time.Now()
	switch timeType {
	case admin.DashboardTimeType_MONTH:
		startAt := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		return startAt, startAt.AddDate(1, 0, 0)
	case admin.DashboardTimeType_WEEK:
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		startAt := time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, now.Location())
		return startAt, startAt.AddDate(0, 0, 7)
	default:
		startAt := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		return startAt, startAt.AddDate(0, 1, 0)
	}
}

// formatDashboardAxis 格式化坐标轴
func formatDashboardAxis(timeType admin.DashboardTimeType, index int, startAt time.Time) string {
	switch timeType {
	case admin.DashboardTimeType_MONTH:
		return fmt.Sprintf("%d月", index+1)
	case admin.DashboardTimeType_WEEK:
		labels := []string{"一", "二", "三", "四", "五", "六", "日"}
		return "周" + labels[index]
	default:
		return startAt.AddDate(0, 0, index).Format("01-02")
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
func (c *DashboardCase) queryOrderSummary(ctx context.Context, timeType admin.DashboardTimeType, startAt, endAt time.Time) (map[int64]*dto.OrderSummary, []string) {
	summaryMap := make(map[int64]*dto.OrderSummary)
	axisData := make([]string, 0)
	db := c.orderCase.Query(ctx).Order.WithContext(ctx).UnderlyingDB()

	switch timeType {
	case admin.DashboardTimeType_MONTH:
		var rows []*dto.OrderSummary
		_ = db.Model(&models.Order{}).
			Select("MONTH(created_at) AS `key`, COUNT(*) AS order_count, COALESCE(SUM(pay_money),0) AS sale_amount").
			Where("created_at >= ? AND created_at < ?", startAt, endAt).
			Group("MONTH(created_at)").
			Scan(&rows).Error
		for _, item := range rows {
			summaryMap[item.Key] = item
		}
		for i := 0; i < 12; i++ {
			axisData = append(axisData, formatDashboardAxis(timeType, i, startAt))
		}
	case admin.DashboardTimeType_WEEK:
		var rows []*dto.OrderSummary
		_ = db.Model(&models.Order{}).
			Select("WEEKDAY(created_at)+1 AS `key`, COUNT(*) AS order_count, COALESCE(SUM(pay_money),0) AS sale_amount").
			Where("created_at >= ? AND created_at < ?", startAt, endAt).
			Group("WEEKDAY(created_at)").
			Scan(&rows).Error
		for _, item := range rows {
			summaryMap[item.Key] = item
		}
		for i := 0; i < 7; i++ {
			axisData = append(axisData, formatDashboardAxis(timeType, i, startAt))
		}
	default:
		var rows []*dto.OrderSummary
		_ = db.Model(&models.Order{}).
			Select("DAY(created_at) AS `key`, COUNT(*) AS order_count, COALESCE(SUM(pay_money),0) AS sale_amount").
			Where("created_at >= ? AND created_at < ?", startAt, endAt).
			Group("DAY(created_at)").
			Scan(&rows).Error
		for _, item := range rows {
			summaryMap[item.Key] = item
		}
		for i := 0; i < endAt.AddDate(0, 0, -1).Day(); i++ {
			axisData = append(axisData, formatDashboardAxis(timeType, i, startAt))
		}
	}
	return summaryMap, axisData
}

// queryOrderGoodsSummary 查询商品销量统计
func (c *DashboardCase) queryOrderGoodsSummary(ctx context.Context, top int64, startAt, endAt time.Time) ([]*dto.OrderGoodsSummary, error) {
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

// queryGoodsCategorySummary 查询商品分类统计
func (c *DashboardCase) queryGoodsCategorySummary(ctx context.Context) ([]*dto.GoodsCategorySummary, error) {
	res := make([]*dto.GoodsCategorySummary, 0)
	err := c.goodsCase.Query(ctx).Goods.WithContext(ctx).UnderlyingDB().
		Model(&models.Goods{}).
		Select("category_id, COUNT(*) AS goods_count").
		Group("category_id").
		Scan(&res).Error
	return res, err
}

// queryOrderGoodsStatusSummary 查询商品订单状态统计
func (c *DashboardCase) queryOrderGoodsStatusSummary(ctx context.Context, startAt, endAt time.Time) ([]*dto.OrderGoodsStatusSummary, error) {
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
