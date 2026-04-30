package biz

import (
	"context"
	"sort"
	"strconv"
	"time"

	_const "shop/pkg/const"

	adminv1 "shop/api/gen/go/admin/v1"
	commonv1 "shop/api/gen/go/common/v1"
	"shop/pkg/gen/data"
	"shop/service/admin/dto"
	"shop/service/admin/utils"
)

// GoodsAnalyticsCase 商品分析业务。
type GoodsAnalyticsCase struct {
	goodsInfoCase     *GoodsInfoCase
	goodsCategoryCase *GoodsCategoryCase
	goodsStatDayRepo  *data.GoodsStatDayRepository
}

// NewGoodsAnalyticsCase 创建商品分析业务。
func NewGoodsAnalyticsCase(
	goodsInfoCase *GoodsInfoCase,
	goodsCategoryCase *GoodsCategoryCase,
	goodsStatDayRepo *data.GoodsStatDayRepository,
) *GoodsAnalyticsCase {
	return &GoodsAnalyticsCase{
		goodsInfoCase:     goodsInfoCase,
		goodsCategoryCase: goodsCategoryCase,
		goodsStatDayRepo:  goodsStatDayRepo,
	}
}

// SummaryGoodsAnalytics 查询商品摘要指标。
func (c *GoodsAnalyticsCase) SummaryGoodsAnalytics(ctx context.Context, req *adminv1.SummaryGoodsAnalyticsRequest) (*adminv1.SummaryGoodsAnalyticsResponse, error) {
	startAt, endAt := utils.GetAnalyticsTimeRange(req.GetTimeType())
	prevStartAt, prevEndAt := utils.GetPreviousAnalyticsTimeRange(req.GetTimeType(), startAt)

	newGoodsCount, err := c.countNewGoods(ctx, startAt, endAt)
	if err != nil {
		return nil, err
	}
	query := c.goodsInfoCase.Query(ctx).GoodsInfo
	var totalGoodsCount int64
	totalGoodsCount, err = query.WithContext(ctx).Count()
	if err != nil {
		return nil, err
	}
	var putOnGoodsCount int64
	putOnGoodsCount, err = query.WithContext(ctx).
		Where(query.Status.Eq(_const.GOODS_STATUS_PUT_ON)).
		Count()
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
	behaviorSummary := &dto.GoodsAnalyticsSummaryRow{}
	behaviorSummary, err = c.queryGoodsBehaviorSummary(ctx, startAt, endAt)
	if err != nil {
		return nil, err
	}

	return &adminv1.SummaryGoodsAnalyticsResponse{
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

// TrendGoodsAnalytics 查询商品趋势。
func (c *GoodsAnalyticsCase) TrendGoodsAnalytics(ctx context.Context, req *adminv1.TrendGoodsAnalyticsRequest) (*commonv1.AnalyticsTrendResponse, error) {
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

	return &commonv1.AnalyticsTrendResponse{
		Axis: axis,
		Series: []*commonv1.AnalyticsTrendSeries{
			{Name: "浏览次数", Type: commonv1.AnalyticsSeriesType(_const.ANALYTICS_SERIES_TYPE_LINE), Data: viewRow, YAxisIndex: 0},
			{Name: "加购件数", Type: commonv1.AnalyticsSeriesType(_const.ANALYTICS_SERIES_TYPE_LINE), Data: cartRow, YAxisIndex: 0},
			{Name: "支付件数", Type: commonv1.AnalyticsSeriesType(_const.ANALYTICS_SERIES_TYPE_BAR), Data: payGoodsRow, YAxisIndex: 0},
			{Name: "支付金额（元）", Type: commonv1.AnalyticsSeriesType(_const.ANALYTICS_SERIES_TYPE_BAR), Data: payAmountRow, YAxisIndex: 1},
		},
		YAxisNames: []string{"次数 / 件数", "金额（元）"},
	}, nil
}

// PieGoodsAnalytics 查询商品分类分布。
func (c *GoodsAnalyticsCase) PieGoodsAnalytics(ctx context.Context, req *adminv1.PieGoodsAnalyticsRequest) (*commonv1.AnalyticsPieResponse, error) {
	startAt, endAt := utils.GetAnalyticsTimeRange(req.GetTimeType())
	summary, err := c.queryGoodsCategorySummary(ctx, startAt, endAt)
	if err != nil {
		return nil, err
	}

	parentID := int64(0)
	categoryNameMap := c.goodsCategoryCase.NameMap(ctx, &parentID)
	items := make([]*commonv1.AnalyticsPieItem, 0, len(summary))
	for _, item := range summary {
		items = append(items, &commonv1.AnalyticsPieItem{
			Name:  categoryNameMap[item.CategoryID],
			Value: item.GoodsCount,
		})
	}
	return &commonv1.AnalyticsPieResponse{Items: items}, nil
}

// RankGoodsAnalytics 查询商品支付排行。
func (c *GoodsAnalyticsCase) RankGoodsAnalytics(ctx context.Context, req *adminv1.RankGoodsAnalyticsRequest) (*commonv1.AnalyticsRankResponse, error) {
	startAt, endAt := utils.GetAnalyticsTimeRange(req.GetTimeType())
	rows, err := c.queryGoodsRankRows(ctx, startAt, endAt, 10)
	if err != nil {
		return nil, err
	}

	goodsIDs := make([]int64, 0, len(rows))
	for _, item := range rows {
		goodsIDs = append(goodsIDs, item.GoodsID)
	}
	nameMap := make(map[int64]string)
	nameMap, err = c.loadGoodsNameMap(ctx, goodsIDs)
	if err != nil {
		return nil, err
	}

	items := make([]*commonv1.AnalyticsRankItem, 0, len(rows))
	for _, item := range rows {
		name := nameMap[item.GoodsID]
		// 名称缺失时，回退成商品编号，避免排行出现空标签。
		if name == "" {
			name = "商品#" + strconv.FormatInt(item.GoodsID, 10)
		}
		items = append(items, &commonv1.AnalyticsRankItem{
			Name:  name,
			Value: item.PayAmount / 100,
		})
	}
	return &commonv1.AnalyticsRankResponse{Items: items}, nil
}

// countNewGoods 统计时间范围内新增商品数。
func (c *GoodsAnalyticsCase) countNewGoods(ctx context.Context, startAt, endAt time.Time) (int64, error) {
	query := c.goodsInfoCase.Query(ctx).GoodsInfo
	count, err := query.WithContext(ctx).
		Where(
			query.CreatedAt.Gte(startAt),
			query.CreatedAt.Lt(endAt),
		).
		Count()
	return count, err
}

// countDistinctActiveGoods 统计时间范围内动销商品数。
func (c *GoodsAnalyticsCase) countDistinctActiveGoods(ctx context.Context, startAt, endAt time.Time) (int64, error) {
	query := c.goodsStatDayRepo.Query(ctx).GoodsStatDay
	count, err := query.WithContext(ctx).
		Where(
			query.StatDate.Gte(startAt),
			query.StatDate.Lt(endAt),
			query.PayGoodsNum.Gt(0),
		).
		Distinct(query.GoodsID).
		Count()
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
	query := c.goodsStatDayRepo.Query(ctx).GoodsStatDay
	err := query.WithContext(ctx).
		Select(
			query.ViewCount.Sum().FloorDiv(1).IfNull(0).As("view_count"),
			query.CollectCount.Sum().FloorDiv(1).IfNull(0).As("collect_count"),
			query.CartCount.Sum().FloorDiv(1).IfNull(0).As("cart_count"),
			query.OrderCount.Sum().FloorDiv(1).IfNull(0).As("order_count"),
			query.PayCount.Sum().FloorDiv(1).IfNull(0).As("pay_count"),
			query.PayGoodsNum.Sum().FloorDiv(1).IfNull(0).As("pay_goods_num"),
			query.PayAmount.Sum().FloorDiv(1).IfNull(0).As("pay_amount"),
		).
		Where(
			query.StatDate.Gte(startAt),
			query.StatDate.Lt(endAt),
		).
		Scan(row)
	return row, err
}

// queryGoodsTrendSummary 查询商品趋势汇总数据。
func (c *GoodsAnalyticsCase) queryGoodsTrendSummary(
	ctx context.Context,
	timeType commonv1.AnalyticsTimeType,
	startAt, endAt time.Time,
) (map[int64]dto.GoodsAnalyticsTrendBucket, []string, error) {
	rows := make([]*dto.GoodsAnalyticsTrendRow, 0)
	query := c.goodsStatDayRepo.Query(ctx).GoodsStatDay
	groupField, axis := utils.GetAnalyticsGroupFieldByColumn(timeType, startAt, endAt, query.StatDate)
	err := query.WithContext(ctx).
		Select(
			groupField.As("key"),
			query.ViewCount.Sum().FloorDiv(1).IfNull(0).As("view_count"),
			query.CartCount.Sum().FloorDiv(1).IfNull(0).As("cart_count"),
			query.PayGoodsNum.Sum().FloorDiv(1).IfNull(0).As("pay_goods_num"),
			query.PayAmount.Sum().FloorDiv(1).IfNull(0).As("pay_amount"),
		).
		Where(
			query.StatDate.Gte(startAt),
			query.StatDate.Lt(endAt),
		).
		Group(utils.AnalyticsGroupAliasField()).
		Scan(&rows)
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

	getRootCategoryID := func(categoryID int64) int64 {
		for {
			parentID, ok := parentMap[categoryID]
			// 当前分类已到顶层或不存在映射时，直接返回当前分类编号。
			if !ok || parentID == 0 {
				return categoryID
			}
			categoryID = parentID
		}
	}

	rows := make([]*dto.GoodsAnalyticsCategorySummaryRow, 0)
	query := c.goodsStatDayRepo.Query(ctx).GoodsStatDay
	err = query.WithContext(ctx).
		Select(
			query.GoodsID,
			query.PayGoodsNum.Sum().FloorDiv(1).IfNull(0).As("goods_count"),
		).
		Where(
			query.StatDate.Gte(startAt),
			query.StatDate.Lt(endAt),
		).
		Group(query.GoodsID).
		Scan(&rows)
	if err != nil {
		return nil, err
	}
	// 统计周期内没有商品行为时，直接返回空分布。
	if len(rows) == 0 {
		return []*dto.GoodsAnalyticsCategorySummaryRow{}, nil
	}

	goodsIDs := make([]int64, 0, len(rows))
	goodsCountMap := make(map[int64]int64, len(rows))
	for _, row := range rows {
		goodsIDs = append(goodsIDs, row.GoodsID)
		goodsCountMap[row.GoodsID] = row.GoodsCount
	}

	goodsRows := make([]*dto.GoodsCategoryIDsRow, 0, len(goodsIDs))
	goodsQuery := c.goodsInfoCase.Query(ctx).GoodsInfo
	err = goodsQuery.WithContext(ctx).
		Select(goodsQuery.ID, goodsQuery.CategoryID).
		Where(
			goodsQuery.DeletedAt.IsNull(),
			goodsQuery.ID.In(goodsIDs...),
		).
		Scan(&goodsRows)
	if err != nil {
		return nil, err
	}

	rootCategoryCount := make(map[int64]int64)
	goodsRootMap := make(map[int64]map[int64]struct{})
	for _, row := range goodsRows {
		for _, categoryID := range c.goodsInfoCase.parseCategoryIDs(row.CategoryID) {
			// 将子分类成交件数汇总到一级分类，便于页面展示类目结构。
			rootID := getRootCategoryID(categoryID)
			// 同一商品命中同一一级分类的多个子分类时，只累计一次成交件数，避免根分类重复放大。
			if _, ok := goodsRootMap[row.GoodsID]; !ok {
				goodsRootMap[row.GoodsID] = make(map[int64]struct{})
			}
			// 当前商品已经累计过该一级分类时，直接跳过重复类目。
			if _, ok := goodsRootMap[row.GoodsID][rootID]; ok {
				continue
			}
			goodsRootMap[row.GoodsID][rootID] = struct{}{}
			rootCategoryCount[rootID] += goodsCountMap[row.GoodsID]
		}
	}

	res := make([]*dto.GoodsAnalyticsCategorySummaryRow, 0, len(rootCategoryCount))
	for categoryID, count := range rootCategoryCount {
		res = append(res, &dto.GoodsAnalyticsCategorySummaryRow{
			CategoryID: categoryID,
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
	query := c.goodsStatDayRepo.Query(ctx).GoodsStatDay
	payAmountField := query.PayAmount.Sum().FloorDiv(1).IfNull(0)
	err := query.WithContext(ctx).
		Select(
			query.GoodsID,
			payAmountField.As("pay_amount"),
		).
		Where(
			query.StatDate.Gte(startAt),
			query.StatDate.Lt(endAt),
		).
		Group(query.GoodsID).
		Order(query.PayAmount.Sum().Desc()).
		Limit(limit).
		Scan(&rows)
	return rows, err
}

// loadGoodsNameMap 加载商品名称映射。
func (c *GoodsAnalyticsCase) loadGoodsNameMap(ctx context.Context, goodsIDs []int64) (map[int64]string, error) {
	// 排行结果为空时，不需要回查商品名称。
	if len(goodsIDs) == 0 {
		return map[int64]string{}, nil
	}

	rows := make([]*dto.GoodsNameRow, 0, len(goodsIDs))
	query := c.goodsInfoCase.Query(ctx).GoodsInfo
	err := query.WithContext(ctx).
		Select(query.ID, query.Name).
		Where(
			query.DeletedAt.IsNull(),
			query.ID.In(goodsIDs...),
		).
		Scan(&rows)
	if err != nil {
		return nil, err
	}

	res := make(map[int64]string, len(rows))
	for _, item := range rows {
		res[item.GoodsID] = item.Name
	}
	return res, nil
}
