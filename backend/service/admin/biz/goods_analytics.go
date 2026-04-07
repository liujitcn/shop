package biz

import (
	"context"
	"sort"
	"time"

	adminApi "shop/api/gen/go/admin"
	commonApi "shop/api/gen/go/common"
	"shop/pkg/gen/models"
	"shop/service/admin/utils"
)

// GoodsAnalyticsCase 商品分析业务
type GoodsAnalyticsCase struct {
	goodsInfoCase     *GoodsInfoCase
	goodsCategoryCase *GoodsCategoryCase
	orderGoodsCase    *OrderGoodsCase
}

// NewGoodsAnalyticsCase 创建商品分析业务
func NewGoodsAnalyticsCase(goodsInfoCase *GoodsInfoCase, goodsCategoryCase *GoodsCategoryCase, orderGoodsCase *OrderGoodsCase) *GoodsAnalyticsCase {
	return &GoodsAnalyticsCase{
		goodsInfoCase:     goodsInfoCase,
		goodsCategoryCase: goodsCategoryCase,
		orderGoodsCase:    orderGoodsCase,
	}
}

// GetGoodsAnalyticsSummary 查询商品摘要指标
func (c *GoodsAnalyticsCase) GetGoodsAnalyticsSummary(ctx context.Context, req *commonApi.AnalyticsTimeRequest) (*adminApi.GoodsAnalyticsSummaryResponse, error) {
	startAt, endAt := utils.GetAnalyticsTimeRange(req.GetTimeType())
	prevStartAt, prevEndAt := utils.GetPreviousAnalyticsTimeRange(req.GetTimeType(), startAt)

	newGoodsCount, err := c.countNewGoods(ctx, startAt, endAt)
	if err != nil {
		return nil, err
	}
	var totalGoodsCount int64
	totalGoodsCount, err = c.countTotalGoods(ctx)
	if err != nil {
		return nil, err
	}
	var putOnGoodsCount int64
	putOnGoodsCount, err = c.countPutOnGoods(ctx)
	if err != nil {
		return nil, err
	}
	var activeGoodsCount int64
	activeGoodsCount, err = c.countDistinctActiveGoods(ctx, startAt, endAt)
	if err != nil {
		return nil, err
	}
	var saleCount int64
	saleCount, err = c.countGoodsSaleNum(ctx, startAt, endAt)
	if err != nil {
		return nil, err
	}
	var prevSaleCount int64
	prevSaleCount, err = c.countGoodsSaleNum(ctx, prevStartAt, prevEndAt)
	if err != nil {
		return nil, err
	}

	return &adminApi.GoodsAnalyticsSummaryResponse{
		NewGoodsCount:    newGoodsCount,
		PutOnGoodsRate:   utils.CalcRatio(putOnGoodsCount, totalGoodsCount),
		ActiveGoodsCount: activeGoodsCount,
		ActiveGoodsRate:  utils.CalcRatio(activeGoodsCount, totalGoodsCount),
		SaleCount:        saleCount,
		SaleGrowthRate:   utils.CalcGrowthRate(prevSaleCount, saleCount),
	}, nil
}

// GetGoodsAnalyticsTrend 查询商品趋势
func (c *GoodsAnalyticsCase) GetGoodsAnalyticsTrend(ctx context.Context, req *commonApi.AnalyticsTimeRequest) (*commonApi.AnalyticsTrendResponse, error) {
	startAt, endAt := utils.GetAnalyticsTimeRange(req.GetTimeType())
	summary, axis, err := c.queryGoodsTrendSummary(ctx, req.GetTimeType(), startAt, endAt)
	if err != nil {
		return nil, err
	}

	saleRow := make([]int64, 0, len(axis))
	activeGoodsRow := make([]int64, 0, len(axis))
	for i := range axis {
		key := int64(i + 1)
		saleRow = append(saleRow, summary[key].saleCount)
		activeGoodsRow = append(activeGoodsRow, summary[key].activeGoodsCount)
	}

	return &commonApi.AnalyticsTrendResponse{
		Axis: axis,
		Series: []*commonApi.AnalyticsTrendSeries{
			{Name: "销量", Type: commonApi.AnalyticsSeriesType_BAR, Data: saleRow},
			{Name: "动销商品数", Type: commonApi.AnalyticsSeriesType_LINE, Data: activeGoodsRow},
		},
		YAxisNames: []string{"销量"},
	}, nil
}

// GetGoodsAnalyticsPie 查询商品分类分布
func (c *GoodsAnalyticsCase) GetGoodsAnalyticsPie(ctx context.Context, req *commonApi.AnalyticsTimeRequest) (*commonApi.AnalyticsPieResponse, error) {
	startAt, endAt := utils.GetAnalyticsTimeRange(req.GetTimeType())
	var summary []*struct {
		GoodsCount int64
		CategoryId int64
	}
	summary, err := c.queryGoodsCategorySummary(ctx, startAt, endAt)
	if err != nil {
		return nil, err
	}

	parentID := int64(0)
	categoryNameMap := c.goodsCategoryCase.NameMap(ctx, &parentID)
	items := make([]*commonApi.AnalyticsPieItem, 0, len(summary))
	for _, item := range summary {
		items = append(items, &commonApi.AnalyticsPieItem{
			Name:  categoryNameMap[item.CategoryId],
			Value: item.GoodsCount,
		})
	}
	return &commonApi.AnalyticsPieResponse{Items: items}, nil
}

func (c *GoodsAnalyticsCase) countNewGoods(ctx context.Context, startAt, endAt time.Time) (int64, error) {
	var count int64
	err := c.goodsInfoCase.Query(ctx).GoodsInfo.WithContext(ctx).UnderlyingDB().
		Model(&models.GoodsInfo{}).
		Where("created_at >= ? AND created_at < ?", startAt, endAt).
		Count(&count).Error
	return count, err
}

func (c *GoodsAnalyticsCase) countTotalGoods(ctx context.Context) (int64, error) {
	var count int64
	err := c.goodsInfoCase.Query(ctx).GoodsInfo.WithContext(ctx).UnderlyingDB().Model(&models.GoodsInfo{}).Count(&count).Error
	return count, err
}

func (c *GoodsAnalyticsCase) countPutOnGoods(ctx context.Context) (int64, error) {
	var count int64
	err := c.goodsInfoCase.Query(ctx).GoodsInfo.WithContext(ctx).UnderlyingDB().
		Model(&models.GoodsInfo{}).
		Where("status = ?", int32(commonApi.GoodsStatus_PUT_ON)).
		Count(&count).Error
	return count, err
}

func (c *GoodsAnalyticsCase) countDistinctActiveGoods(ctx context.Context, startAt, endAt time.Time) (int64, error) {
	var count int64
	err := c.orderGoodsCase.Query(ctx).OrderGoods.WithContext(ctx).UnderlyingDB().
		Model(&models.OrderGoods{}).
		Joins("JOIN `"+models.TableNameOrderInfo+"` ON `"+models.TableNameOrderInfo+"`.id = order_goods.order_id").
		Where("`"+models.TableNameOrderInfo+"`.created_at >= ? AND `"+models.TableNameOrderInfo+"`.created_at < ?", startAt, endAt).
		Distinct("order_goods.goods_id").
		Count(&count).Error
	return count, err
}

func (c *GoodsAnalyticsCase) countGoodsSaleNum(ctx context.Context, startAt, endAt time.Time) (int64, error) {
	type row struct {
		SaleCount int64 `gorm:"column:sale_count"`
	}
	var result row
	err := c.orderGoodsCase.Query(ctx).OrderGoods.WithContext(ctx).UnderlyingDB().
		Model(&models.OrderGoods{}).
		Select("COALESCE(SUM(order_goods.num),0) AS sale_count").
		Joins("JOIN `"+models.TableNameOrderInfo+"` ON `"+models.TableNameOrderInfo+"`.id = order_goods.order_id").
		Where("`"+models.TableNameOrderInfo+"`.created_at >= ? AND `"+models.TableNameOrderInfo+"`.created_at < ?", startAt, endAt).
		Scan(&result).Error
	return result.SaleCount, err
}

type goodsTrendSummary struct {
	saleCount        int64
	activeGoodsCount int64
}

// queryGoodsTrendSummary 查询商品趋势汇总数据。
func (c *GoodsAnalyticsCase) queryGoodsTrendSummary(ctx context.Context, timeType commonApi.AnalyticsTimeType, startAt, endAt time.Time) (map[int64]goodsTrendSummary, []string, error) {
	type row struct {
		Key              int64 `gorm:"column:key"`
		SaleCount        int64 `gorm:"column:sale_count"`
		ActiveGoodsCount int64 `gorm:"column:active_goods_count"`
	}
	rows := make([]*row, 0)
	selectExpr, axis := utils.GetAnalyticsGroupExpr(timeType, startAt, endAt)
	err := c.orderGoodsCase.Query(ctx).OrderGoods.WithContext(ctx).UnderlyingDB().
		Model(&models.OrderGoods{}).
		Select(selectExpr+" AS `key`, COALESCE(SUM(order_goods.num),0) AS sale_count, COUNT(DISTINCT order_goods.goods_id) AS active_goods_count").
		Joins("JOIN `"+models.TableNameOrderInfo+"` ON `"+models.TableNameOrderInfo+"`.id = order_goods.order_id").
		Where("`"+models.TableNameOrderInfo+"`.created_at >= ? AND `"+models.TableNameOrderInfo+"`.created_at < ?", startAt, endAt).
		Group("`key`").
		Scan(&rows).Error
	if err != nil {
		return nil, nil, err
	}

	res := make(map[int64]goodsTrendSummary, len(axis))
	for _, item := range rows {
		res[item.Key] = goodsTrendSummary{
			saleCount:        item.SaleCount,
			activeGoodsCount: item.ActiveGoodsCount,
		}
	}
	// 补齐空档位，保证前端图表序列长度与横轴一致。
	for i := range axis {
		key := int64(i + 1)
		if _, ok := res[key]; !ok {
			res[key] = goodsTrendSummary{}
		}
	}
	return res, axis, nil
}

// queryGoodsCategorySummary 查询商品分类销量分布。
func (c *GoodsAnalyticsCase) queryGoodsCategorySummary(ctx context.Context, startAt, endAt time.Time) ([]*struct {
	GoodsCount int64
	CategoryId int64
}, error) {
	categoryList, err := c.goodsCategoryCase.List(ctx)
	if err != nil {
		return nil, err
	}

	parentMap := make(map[int64]int64, len(categoryList))
	for _, category := range categoryList {
		parentMap[category.ID] = category.ParentID
	}

	getRootCategoryID := func(categoryID int64) int64 {
		for {
			parentID, ok := parentMap[categoryID]
			if !ok || parentID == 0 {
				return categoryID
			}
			categoryID = parentID
		}
	}

	rows := make([]*struct {
		GoodsCount int64 `gorm:"column:goods_count"`
		CategoryId int64 `gorm:"column:category_id"`
	}, 0)
	err = c.orderGoodsCase.Query(ctx).OrderGoods.WithContext(ctx).UnderlyingDB().
		Model(&models.OrderGoods{}).
		Select(models.TableNameGoodsInfo+".category_id, COALESCE(SUM(order_goods.num),0) AS goods_count").
		Joins("JOIN "+models.TableNameGoodsInfo+" ON "+models.TableNameGoodsInfo+".id = order_goods.goods_id").
		Joins("JOIN `"+models.TableNameOrderInfo+"` ON `"+models.TableNameOrderInfo+"`.id = order_goods.order_id").
		Where("`"+models.TableNameOrderInfo+"`.created_at >= ? AND `"+models.TableNameOrderInfo+"`.created_at < ?", startAt, endAt).
		Group(models.TableNameGoodsInfo + ".category_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	rootCategoryCount := make(map[int64]int64)
	for _, row := range rows {
		// 将子分类销量汇总到一级分类，便于页面展示大类分布。
		rootID := getRootCategoryID(row.CategoryId)
		rootCategoryCount[rootID] += row.GoodsCount
	}

	res := make([]*struct {
		GoodsCount int64
		CategoryId int64
	}, 0, len(rootCategoryCount))
	for categoryID, count := range rootCategoryCount {
		res = append(res, &struct {
			GoodsCount int64
			CategoryId int64
		}{GoodsCount: count, CategoryId: categoryID})
	}
	sort.Slice(res, func(i, j int) bool { return res[i].GoodsCount > res[j].GoodsCount })
	return res, nil
}
