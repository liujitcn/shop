package biz

import (
	"context"
	"sort"
	"strconv"
	"time"

	adminApi "shop/api/gen/go/admin"
	commonApi "shop/api/gen/go/common"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	"shop/service/admin/dto"
	"shop/service/admin/utils"
)

// GoodsAnalyticsCase 商品分析业务。
type GoodsAnalyticsCase struct {
	goodsInfoCase     *GoodsInfoCase
	goodsCategoryCase *GoodsCategoryCase
	goodsStatDayRepo  *data.GoodsStatDayRepo
}

// NewGoodsAnalyticsCase 创建商品分析业务。
func NewGoodsAnalyticsCase(
	goodsInfoCase *GoodsInfoCase,
	goodsCategoryCase *GoodsCategoryCase,
	goodsStatDayRepo *data.GoodsStatDayRepo,
) *GoodsAnalyticsCase {
	return &GoodsAnalyticsCase{
		goodsInfoCase:     goodsInfoCase,
		goodsCategoryCase: goodsCategoryCase,
		goodsStatDayRepo:  goodsStatDayRepo,
	}
}

// GetGoodsAnalyticsSummary 查询商品摘要指标。
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
	behaviorSummary, err := c.queryGoodsBehaviorSummary(ctx, startAt, endAt)
	if err != nil {
		return nil, err
	}

	return &adminApi.GoodsAnalyticsSummaryResponse{
		NewGoodsCount:       newGoodsCount,
		PutOnGoodsRate:      utils.CalcRatio(putOnGoodsCount, totalGoodsCount),
		ActiveGoodsCount:    activeGoodsCount,
		ActiveGoodsRate:     utils.CalcRatio(activeGoodsCount, totalGoodsCount),
		SaleCount:           saleCount,
		SaleGrowthRate:      utils.CalcGrowthRate(prevSaleCount, saleCount),
		ViewCount:           behaviorSummary.ViewCount,
		CollectCount:        behaviorSummary.CollectCount,
		CartCount:           behaviorSummary.CartCount,
		OrderCount:          behaviorSummary.OrderCount,
		PayCount:            behaviorSummary.PayCount,
		PayAmount:           behaviorSummary.PayAmount,
		CartConversionRate:  utils.CalcRatio(behaviorSummary.CartCount, behaviorSummary.ViewCount),
		OrderConversionRate: utils.CalcRatio(behaviorSummary.OrderCount, behaviorSummary.CartCount),
		PayConversionRate:   utils.CalcRatio(behaviorSummary.PayCount, behaviorSummary.ViewCount),
		PayUnitPrice:        utils.CalcPerUnit(behaviorSummary.PayAmount, behaviorSummary.PayGoodsNum),
	}, nil
}

// GetGoodsAnalyticsTrend 查询商品趋势。
func (c *GoodsAnalyticsCase) GetGoodsAnalyticsTrend(ctx context.Context, req *commonApi.AnalyticsTimeRequest) (*commonApi.AnalyticsTrendResponse, error) {
	startAt, endAt := utils.GetAnalyticsTimeRange(req.GetTimeType())
	summary, axis, err := c.queryGoodsTrendSummary(ctx, req.GetTimeType(), startAt, endAt)
	if err != nil {
		return nil, err
	}

	viewRow := make([]int64, 0, len(axis))
	cartRow := make([]int64, 0, len(axis))
	payGoodsRow := make([]int64, 0, len(axis))
	payAmountRow := make([]int64, 0, len(axis))
	for i := range axis {
		key := int64(i + 1)
		viewRow = append(viewRow, summary[key].ViewCount)
		cartRow = append(cartRow, summary[key].CartCount)
		payGoodsRow = append(payGoodsRow, summary[key].PayGoodsNum)
		payAmountRow = append(payAmountRow, summary[key].PayAmount/100)
	}

	return &commonApi.AnalyticsTrendResponse{
		Axis: axis,
		Series: []*commonApi.AnalyticsTrendSeries{
			{Name: "浏览次数", Type: commonApi.AnalyticsSeriesType_LINE, Data: viewRow, YAxisIndex: 0},
			{Name: "加购件数", Type: commonApi.AnalyticsSeriesType_LINE, Data: cartRow, YAxisIndex: 0},
			{Name: "支付件数", Type: commonApi.AnalyticsSeriesType_BAR, Data: payGoodsRow, YAxisIndex: 0},
			{Name: "支付金额（元）", Type: commonApi.AnalyticsSeriesType_BAR, Data: payAmountRow, YAxisIndex: 1},
		},
		YAxisNames: []string{"次数 / 件数", "金额（元）"},
	}, nil
}

// GetGoodsAnalyticsPie 查询商品分类分布。
func (c *GoodsAnalyticsCase) GetGoodsAnalyticsPie(ctx context.Context, req *commonApi.AnalyticsTimeRequest) (*commonApi.AnalyticsPieResponse, error) {
	startAt, endAt := utils.GetAnalyticsTimeRange(req.GetTimeType())
	summary, err := c.queryGoodsCategorySummary(ctx, startAt, endAt)
	if err != nil {
		return nil, err
	}

	parentId := int64(0)
	categoryNameMap := c.goodsCategoryCase.NameMap(ctx, &parentId)
	items := make([]*commonApi.AnalyticsPieItem, 0, len(summary))
	for _, item := range summary {
		items = append(items, &commonApi.AnalyticsPieItem{
			Name:  categoryNameMap[item.CategoryId],
			Value: item.GoodsCount,
		})
	}
	return &commonApi.AnalyticsPieResponse{Items: items}, nil
}

// GetGoodsAnalyticsRank 查询商品支付排行。
func (c *GoodsAnalyticsCase) GetGoodsAnalyticsRank(ctx context.Context, req *commonApi.AnalyticsTimeRequest) (*commonApi.AnalyticsRankResponse, error) {
	startAt, endAt := utils.GetAnalyticsTimeRange(req.GetTimeType())
	rows, err := c.queryGoodsRankRows(ctx, startAt, endAt, 10)
	if err != nil {
		return nil, err
	}

	goodsIds := make([]int64, 0, len(rows))
	for _, item := range rows {
		goodsIds = append(goodsIds, item.GoodsId)
	}
	nameMap, err := c.loadGoodsNameMap(ctx, goodsIds)
	if err != nil {
		return nil, err
	}

	items := make([]*commonApi.AnalyticsRankItem, 0, len(rows))
	for _, item := range rows {
		name := nameMap[item.GoodsId]
		// 名称缺失时，回退成商品编号，避免排行出现空标签。
		if name == "" {
			name = "商品#" + strconv.FormatInt(item.GoodsId, 10)
		}
		items = append(items, &commonApi.AnalyticsRankItem{
			Name:  name,
			Value: item.PayAmount / 100,
		})
	}
	return &commonApi.AnalyticsRankResponse{Items: items}, nil
}

// countNewGoods 统计时间范围内新增商品数。
func (c *GoodsAnalyticsCase) countNewGoods(ctx context.Context, startAt, endAt time.Time) (int64, error) {
	var count int64
	err := c.goodsInfoCase.Query(ctx).GoodsInfo.WithContext(ctx).UnderlyingDB().
		Model(&models.GoodsInfo{}).
		Where("created_at >= ? AND created_at < ?", startAt, endAt).
		Count(&count).Error
	return count, err
}

// countTotalGoods 统计商品总数。
func (c *GoodsAnalyticsCase) countTotalGoods(ctx context.Context) (int64, error) {
	var count int64
	err := c.goodsInfoCase.Query(ctx).GoodsInfo.WithContext(ctx).UnderlyingDB().Model(&models.GoodsInfo{}).Count(&count).Error
	return count, err
}

// countPutOnGoods 统计已上架商品数。
func (c *GoodsAnalyticsCase) countPutOnGoods(ctx context.Context) (int64, error) {
	var count int64
	err := c.goodsInfoCase.Query(ctx).GoodsInfo.WithContext(ctx).UnderlyingDB().
		Model(&models.GoodsInfo{}).
		Where("status = ?", int32(commonApi.GoodsStatus_PUT_ON)).
		Count(&count).Error
	return count, err
}

// countDistinctActiveGoods 统计时间范围内动销商品数。
func (c *GoodsAnalyticsCase) countDistinctActiveGoods(ctx context.Context, startAt, endAt time.Time) (int64, error) {
	var count int64
	err := c.goodsStatDayRepo.Query(ctx).GoodsStatDay.WithContext(ctx).UnderlyingDB().
		Model(&models.GoodsStatDay{}).
		Where("stat_date >= ? AND stat_date < ? AND pay_goods_num > 0", startAt, endAt).
		Distinct("goods_id").
		Count(&count).Error
	return count, err
}

// countGoodsSaleNum 统计时间范围内商品销量。
func (c *GoodsAnalyticsCase) countGoodsSaleNum(ctx context.Context, startAt, endAt time.Time) (int64, error) {
	summary, err := c.queryGoodsBehaviorSummary(ctx, startAt, endAt)
	if err != nil {
		return 0, err
	}
	return summary.PayGoodsNum, nil
}

// queryGoodsBehaviorSummary 查询商品行为汇总。
func (c *GoodsAnalyticsCase) queryGoodsBehaviorSummary(ctx context.Context, startAt, endAt time.Time) (*dto.GoodsAnalyticsSummaryRow, error) {
	row := &dto.GoodsAnalyticsSummaryRow{}
	err := c.goodsStatDayRepo.Query(ctx).GoodsStatDay.WithContext(ctx).UnderlyingDB().
		Model(&models.GoodsStatDay{}).
		Select(""+
			"COALESCE(SUM(view_count), 0) AS view_count,"+
			" COALESCE(SUM(collect_count), 0) AS collect_count,"+
			" COALESCE(SUM(cart_count), 0) AS cart_count,"+
			" COALESCE(SUM(order_count), 0) AS order_count,"+
			" COALESCE(SUM(pay_count), 0) AS pay_count,"+
			" COALESCE(SUM(pay_goods_num), 0) AS pay_goods_num,"+
			" COALESCE(SUM(pay_amount), 0) AS pay_amount").
		Where("stat_date >= ? AND stat_date < ?", startAt, endAt).
		Scan(row).Error
	return row, err
}

// queryGoodsTrendSummary 查询商品趋势汇总数据。
func (c *GoodsAnalyticsCase) queryGoodsTrendSummary(
	ctx context.Context,
	timeType commonApi.AnalyticsTimeType,
	startAt, endAt time.Time,
) (map[int64]dto.GoodsAnalyticsTrendBucket, []string, error) {
	rows := make([]*dto.GoodsAnalyticsTrendRow, 0)
	selectExpr, axis := utils.GetAnalyticsGroupExprByColumn(timeType, startAt, endAt, "stat_date")
	err := c.goodsStatDayRepo.Query(ctx).GoodsStatDay.WithContext(ctx).UnderlyingDB().
		Model(&models.GoodsStatDay{}).
		Select(selectExpr+" AS `key`,"+
			" COALESCE(SUM(view_count), 0) AS view_count,"+
			" COALESCE(SUM(cart_count), 0) AS cart_count,"+
			" COALESCE(SUM(pay_goods_num), 0) AS pay_goods_num,"+
			" COALESCE(SUM(pay_amount), 0) AS pay_amount").
		Where("stat_date >= ? AND stat_date < ?", startAt, endAt).
		Group("`key`").
		Scan(&rows).Error
	if err != nil {
		return nil, nil, err
	}

	res := make(map[int64]dto.GoodsAnalyticsTrendBucket, len(axis))
	for _, item := range rows {
		res[item.Key] = dto.GoodsAnalyticsTrendBucket{
			ViewCount:   item.ViewCount,
			CartCount:   item.CartCount,
			PayGoodsNum: item.PayGoodsNum,
			PayAmount:   item.PayAmount,
		}
	}
	for i := range axis {
		key := int64(i + 1)
		// 当前桶位缺少聚合结果时，补齐空对象保证前端序列完整。
		if _, ok := res[key]; !ok {
			res[key] = dto.GoodsAnalyticsTrendBucket{}
		}
	}
	return res, axis, nil
}

// queryGoodsCategorySummary 查询商品分类分布。
func (c *GoodsAnalyticsCase) queryGoodsCategorySummary(ctx context.Context, startAt, endAt time.Time) ([]*dto.GoodsAnalyticsCategorySummaryRow, error) {
	categoryList, err := c.goodsCategoryCase.List(ctx)
	if err != nil {
		return nil, err
	}

	parentMap := make(map[int64]int64, len(categoryList))
	for _, category := range categoryList {
		parentMap[category.ID] = category.ParentID
	}

	getRootCategoryId := func(categoryId int64) int64 {
		for {
			parentId, ok := parentMap[categoryId]
			// 当前分类已到顶层或不存在映射时，直接返回当前分类编号。
			if !ok || parentId == 0 {
				return categoryId
			}
			categoryId = parentId
		}
	}

	rows := make([]*dto.GoodsAnalyticsCategorySummaryRow, 0)
	err = c.goodsStatDayRepo.Query(ctx).GoodsStatDay.WithContext(ctx).UnderlyingDB().
		Model(&models.GoodsStatDay{}).
		Select(models.TableNameGoodsInfo+".category_id, COALESCE(SUM("+models.TableNameGoodsStatDay+".pay_goods_num),0) AS goods_count").
		Joins("JOIN "+models.TableNameGoodsInfo+" ON "+models.TableNameGoodsInfo+".id = "+models.TableNameGoodsStatDay+".goods_id").
		Where(models.TableNameGoodsInfo+".deleted_at IS NULL").
		Where(models.TableNameGoodsStatDay+".stat_date >= ? AND "+models.TableNameGoodsStatDay+".stat_date < ?", startAt, endAt).
		Group(models.TableNameGoodsInfo + ".category_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	rootCategoryCount := make(map[int64]int64)
	for _, row := range rows {
		// 将子分类成交件数汇总到一级分类，便于页面展示类目结构。
		rootId := getRootCategoryId(row.CategoryId)
		rootCategoryCount[rootId] += row.GoodsCount
	}

	res := make([]*dto.GoodsAnalyticsCategorySummaryRow, 0, len(rootCategoryCount))
	for categoryId, count := range rootCategoryCount {
		res = append(res, &dto.GoodsAnalyticsCategorySummaryRow{
			CategoryId: categoryId,
			GoodsCount: count,
		})
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].GoodsCount > res[j].GoodsCount
	})
	return res, nil
}

// queryGoodsRankRows 查询商品支付排行。
func (c *GoodsAnalyticsCase) queryGoodsRankRows(ctx context.Context, startAt, endAt time.Time, limit int) ([]*dto.GoodsAnalyticsRankRow, error) {
	rows := make([]*dto.GoodsAnalyticsRankRow, 0)
	err := c.goodsStatDayRepo.Query(ctx).GoodsStatDay.WithContext(ctx).UnderlyingDB().
		Model(&models.GoodsStatDay{}).
		Select("goods_id, COALESCE(SUM(pay_amount), 0) AS pay_amount").
		Where("stat_date >= ? AND stat_date < ?", startAt, endAt).
		Group("goods_id").
		Order("pay_amount DESC").
		Limit(limit).
		Scan(&rows).Error
	return rows, err
}

// loadGoodsNameMap 加载商品名称映射。
func (c *GoodsAnalyticsCase) loadGoodsNameMap(ctx context.Context, goodsIds []int64) (map[int64]string, error) {
	// 排行结果为空时，不需要回查商品名称。
	if len(goodsIds) == 0 {
		return map[int64]string{}, nil
	}

	rows := make([]*dto.GoodsNameRow, 0, len(goodsIds))
	err := c.goodsInfoCase.Query(ctx).GoodsInfo.WithContext(ctx).UnderlyingDB().
		Model(&models.GoodsInfo{}).
		Select("id, name").
		Where("deleted_at IS NULL").
		Where("id IN ?", goodsIds).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	res := make(map[int64]string, len(rows))
	for _, item := range rows {
		res[item.GoodsId] = item.Name
	}
	return res, nil
}
