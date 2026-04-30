package biz

import (
	"context"
	"time"

	_const "shop/pkg/const"

	adminv1 "shop/api/gen/go/admin/v1"
	commonv1 "shop/api/gen/go/common/v1"
	"shop/service/admin/utils"
)

const (
	USER_BEHAVIOR_ADDRESS = "address"
	USER_BEHAVIOR_COLLECT = "collect"
	USER_BEHAVIOR_CART    = "cart"
	USER_BEHAVIOR_STORE   = "store"
)

// UserAnalyticsCase 用户分析业务
type UserAnalyticsCase struct {
	baseUserCase  *BaseUserCase
	orderInfoCase *OrderInfoCase
}

// NewUserAnalyticsCase 创建用户分析业务
func NewUserAnalyticsCase(baseUserCase *BaseUserCase, orderInfoCase *OrderInfoCase) *UserAnalyticsCase {
	return &UserAnalyticsCase{
		baseUserCase:  baseUserCase,
		orderInfoCase: orderInfoCase,
	}
}

// SummaryUserAnalytics 查询用户摘要指标
func (c *UserAnalyticsCase) SummaryUserAnalytics(ctx context.Context, req *adminv1.SummaryUserAnalyticsRequest) (*adminv1.SummaryUserAnalyticsResponse, error) {
	startAt, endAt := utils.GetAnalyticsTimeRange(req.GetTimeType())
	prevStartAt, prevEndAt := utils.GetPreviousAnalyticsTimeRange(req.GetTimeType(), startAt)

	newUserCount, err := c.countNewUsers(ctx, startAt, endAt)
	if err != nil {
		return nil, err
	}
	var prevNewUserCount int64
	prevNewUserCount, err = c.countNewUsers(ctx, prevStartAt, prevEndAt)
	if err != nil {
		return nil, err
	}
	var orderUserCount int64
	orderUserCount, err = c.countDistinctOrderUsers(ctx, startAt, endAt)
	if err != nil {
		return nil, err
	}
	var activeUserCount int64
	activeUserCount, err = c.countDistinctActiveUsers(ctx, startAt, endAt)
	if err != nil {
		return nil, err
	}

	return &adminv1.SummaryUserAnalyticsResponse{
		NewUserCount:            newUserCount,
		NewUserGrowthRate:       utils.CalcGrowthRate(prevNewUserCount, newUserCount),
		OrderUserCount:          orderUserCount,
		OrderUserConversionRate: utils.CalcRatio(orderUserCount, newUserCount),
		ActiveUserCount:         activeUserCount,
		ActiveUserCoverageRate:  utils.CalcRatio(activeUserCount, newUserCount),
	}, nil
}

// TrendUserAnalytics 查询用户趋势
func (c *UserAnalyticsCase) TrendUserAnalytics(ctx context.Context, req *adminv1.TrendUserAnalyticsRequest) (*commonv1.AnalyticsTrendResponse, error) {
	startAt, endAt := utils.GetAnalyticsTimeRange(req.GetTimeType())
	registerSummary, axis, err := c.queryUserRegisterSummary(ctx, req.GetTimeType(), startAt, endAt)
	if err != nil {
		return nil, err
	}
	var orderUserSummary map[int64]int64
	orderUserSummary, err = c.queryOrderUserSummary(ctx, req.GetTimeType(), startAt, endAt)
	if err != nil {
		return nil, err
	}

	registerRow := make([]int64, 0, len(axis))
	orderUserRow := make([]int64, 0, len(axis))
	for i := range axis {
		key := int64(i + 1)
		registerRow = append(registerRow, registerSummary[key])
		orderUserRow = append(orderUserRow, orderUserSummary[key])
	}

	return &commonv1.AnalyticsTrendResponse{
		Axis: axis,
		Series: []*commonv1.AnalyticsTrendSeries{
			{Name: "注册用户", Type: commonv1.AnalyticsSeriesType(_const.ANALYTICS_SERIES_TYPE_BAR), Data: registerRow},
			{Name: "下单用户", Type: commonv1.AnalyticsSeriesType(_const.ANALYTICS_SERIES_TYPE_LINE), Data: orderUserRow},
		},
		YAxisNames: []string{"人数"},
	}, nil
}

// RankUserAnalytics 查询用户行为覆盖排行
func (c *UserAnalyticsCase) RankUserAnalytics(ctx context.Context, req *adminv1.RankUserAnalyticsRequest) (*commonv1.AnalyticsRankResponse, error) {
	_, endAt := utils.GetAnalyticsTimeRange(req.GetTimeType())

	addressUserCount, err := c.countDistinctBehaviorUsers(ctx, USER_BEHAVIOR_ADDRESS, endAt)
	if err != nil {
		return nil, err
	}
	var collectUserCount int64
	collectUserCount, err = c.countDistinctBehaviorUsers(ctx, USER_BEHAVIOR_COLLECT, endAt)
	if err != nil {
		return nil, err
	}
	var cartUserCount int64
	cartUserCount, err = c.countDistinctBehaviorUsers(ctx, USER_BEHAVIOR_CART, endAt)
	if err != nil {
		return nil, err
	}
	var orderUserCount int64
	orderUserCount, err = c.countDistinctOrderUsers(ctx, time.Time{}, endAt)
	if err != nil {
		return nil, err
	}
	var storeUserCount int64
	storeUserCount, err = c.countDistinctBehaviorUsers(ctx, USER_BEHAVIOR_STORE, endAt)
	if err != nil {
		return nil, err
	}

	return &commonv1.AnalyticsRankResponse{
		Items: []*commonv1.AnalyticsRankItem{
			{Name: "已填写地址", Value: addressUserCount},
			{Name: "有收藏行为", Value: collectUserCount},
			{Name: "有加购行为", Value: cartUserCount},
			{Name: "有下单行为", Value: orderUserCount},
			{Name: "有门店申请", Value: storeUserCount},
		},
	}, nil
}

// countNewUsers 统计时间范围内新增用户数。
func (c *UserAnalyticsCase) countNewUsers(ctx context.Context, startAt, endAt time.Time) (int64, error) {
	query := c.baseUserCase.Query(ctx).BaseUser
	count, err := query.WithContext(ctx).
		Where(
			query.CreatedAt.Gte(startAt),
			query.CreatedAt.Lt(endAt),
		).
		Count()
	return count, err
}

// countDistinctOrderUsers 统计时间范围内下单用户数。
func (c *UserAnalyticsCase) countDistinctOrderUsers(ctx context.Context, startAt, endAt time.Time) (int64, error) {
	query := c.orderInfoCase.Query(ctx).OrderInfo
	dao := query.WithContext(ctx)
	// 指定开始时间时，按完整区间统计下单用户。
	if !startAt.IsZero() {
		dao = dao.Where(
			query.CreatedAt.Gte(startAt),
			query.CreatedAt.Lt(endAt),
		)
	} else {
		// 未指定开始时间时，统计截止时间之前的累计下单用户。
		dao = dao.Where(query.CreatedAt.Lt(endAt))
	}
	count, err := dao.Distinct(query.UserID).Count()
	return count, err
}

// countDistinctActiveUsers 查询周期内活跃用户数。
func (c *UserAnalyticsCase) countDistinctActiveUsers(ctx context.Context, startAt, endAt time.Time) (int64, error) {
	rootQuery := c.baseUserCase.Query(ctx)
	userIDSet := make(map[int64]struct{})

	addressQuery := rootQuery.UserAddress
	addressUserIDs := make([]int64, 0)
	err := addressQuery.WithContext(ctx).
		Where(
			addressQuery.CreatedAt.Gte(startAt),
			addressQuery.CreatedAt.Lt(endAt),
		).
		Distinct(addressQuery.UserID).
		Pluck(addressQuery.UserID, &addressUserIDs)
	if err != nil {
		return 0, err
	}
	mergeUserIDs(userIDSet, addressUserIDs)

	collectQuery := rootQuery.UserCollect
	collectUserIDs := make([]int64, 0)
	err = collectQuery.WithContext(ctx).
		Where(
			collectQuery.CreatedAt.Gte(startAt),
			collectQuery.CreatedAt.Lt(endAt),
		).
		Distinct(collectQuery.UserID).
		Pluck(collectQuery.UserID, &collectUserIDs)
	if err != nil {
		return 0, err
	}
	mergeUserIDs(userIDSet, collectUserIDs)

	cartQuery := rootQuery.UserCart
	cartUserIDs := make([]int64, 0)
	err = cartQuery.WithContext(ctx).
		Where(
			cartQuery.CreatedAt.Gte(startAt),
			cartQuery.CreatedAt.Lt(endAt),
		).
		Distinct(cartQuery.UserID).
		Pluck(cartQuery.UserID, &cartUserIDs)
	if err != nil {
		return 0, err
	}
	mergeUserIDs(userIDSet, cartUserIDs)

	storeQuery := rootQuery.UserStore
	storeUserIDs := make([]int64, 0)
	err = storeQuery.WithContext(ctx).
		Where(
			storeQuery.CreatedAt.Gte(startAt),
			storeQuery.CreatedAt.Lt(endAt),
		).
		Distinct(storeQuery.UserID).
		Pluck(storeQuery.UserID, &storeUserIDs)
	if err != nil {
		return 0, err
	}
	mergeUserIDs(userIDSet, storeUserIDs)

	orderQuery := rootQuery.OrderInfo
	orderUserIDs := make([]int64, 0)
	err = orderQuery.WithContext(ctx).
		Where(
			orderQuery.CreatedAt.Gte(startAt),
			orderQuery.CreatedAt.Lt(endAt),
		).
		Distinct(orderQuery.UserID).
		Pluck(orderQuery.UserID, &orderUserIDs)
	if err != nil {
		return 0, err
	}
	mergeUserIDs(userIDSet, orderUserIDs)
	return int64(len(userIDSet)), nil
}

// mergeUserIDs 将查询到的用户编号合并到去重集合。
func mergeUserIDs(userIDSet map[int64]struct{}, userIDs []int64) {
	for _, userID := range userIDs {
		userIDSet[userID] = struct{}{}
	}
}

// countDistinctBehaviorUsers 统计截止指定时间的行为用户数。
func (c *UserAnalyticsCase) countDistinctBehaviorUsers(ctx context.Context, behaviorType string, endAt time.Time) (int64, error) {
	rootQuery := c.baseUserCase.Query(ctx)
	// 不同行为来自不同业务表，分别使用对应 gorm/gen 查询对象统计去重用户。
	switch behaviorType {
	case USER_BEHAVIOR_ADDRESS:
		query := rootQuery.UserAddress
		return query.WithContext(ctx).
			Where(query.CreatedAt.Lt(endAt)).
			Distinct(query.UserID).
			Count()
	case USER_BEHAVIOR_COLLECT:
		query := rootQuery.UserCollect
		return query.WithContext(ctx).
			Where(query.CreatedAt.Lt(endAt)).
			Distinct(query.UserID).
			Count()
	case USER_BEHAVIOR_CART:
		query := rootQuery.UserCart
		return query.WithContext(ctx).
			Where(query.CreatedAt.Lt(endAt)).
			Distinct(query.UserID).
			Count()
	case USER_BEHAVIOR_STORE:
		query := rootQuery.UserStore
		return query.WithContext(ctx).
			Where(query.CreatedAt.Lt(endAt)).
			Distinct(query.UserID).
			Count()
	default:
		return 0, nil
	}
}

// queryUserRegisterSummary 查询用户注册趋势汇总。
func (c *UserAnalyticsCase) queryUserRegisterSummary(ctx context.Context, timeType commonv1.AnalyticsTimeType, startAt, endAt time.Time) (map[int64]int64, []string, error) {
	type row struct {
		Key   int64 `gorm:"column:key"`
		Count int64 `gorm:"column:count"`
	}
	rows := make([]*row, 0)
	query := c.baseUserCase.Query(ctx).BaseUser
	groupField, axis := utils.GetAnalyticsGroupFieldByColumn(timeType, startAt, endAt, query.CreatedAt)
	err := query.WithContext(ctx).
		Select(
			groupField.As("key"),
			query.ID.Count().As("count"),
		).
		Where(
			query.CreatedAt.Gte(startAt),
			query.CreatedAt.Lt(endAt),
		).
		Group(utils.AnalyticsGroupAliasField()).
		Scan(&rows)
	if err != nil {
		return nil, nil, err
	}
	res := make(map[int64]int64, len(rows))
	for _, item := range rows {
		res[item.Key] = item.Count
	}
	return res, axis, nil
}

// queryOrderUserSummary 查询下单用户趋势汇总。
func (c *UserAnalyticsCase) queryOrderUserSummary(ctx context.Context, timeType commonv1.AnalyticsTimeType, startAt, endAt time.Time) (map[int64]int64, error) {
	type row struct {
		Key   int64 `gorm:"column:key"`
		Count int64 `gorm:"column:count"`
	}
	rows := make([]*row, 0)
	query := c.orderInfoCase.Query(ctx).OrderInfo
	groupField, _ := utils.GetAnalyticsGroupFieldByColumn(timeType, startAt, endAt, query.CreatedAt)
	err := query.WithContext(ctx).
		Select(
			groupField.As("key"),
			query.UserID.Distinct().Count().As("count"),
		).
		Where(
			query.CreatedAt.Gte(startAt),
			query.CreatedAt.Lt(endAt),
		).
		Group(utils.AnalyticsGroupAliasField()).
		Scan(&rows)
	if err != nil {
		return nil, err
	}
	res := make(map[int64]int64, len(rows))
	for _, item := range rows {
		res[item.Key] = item.Count
	}
	return res, nil
}
