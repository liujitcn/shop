package biz

import (
	"context"
	"time"

	adminApi "shop/api/gen/go/admin"
	commonApi "shop/api/gen/go/common"
	"shop/pkg/gen/models"
	"shop/service/admin/utils"
)

// UserAnalyticsCase 用户分析业务
type UserAnalyticsCase struct {
	baseUserCase *BaseUserCase
	orderCase    *OrderCase
}

// NewUserAnalyticsCase 创建用户分析业务
func NewUserAnalyticsCase(baseUserCase *BaseUserCase, orderCase *OrderCase) *UserAnalyticsCase {
	return &UserAnalyticsCase{
		baseUserCase: baseUserCase,
		orderCase:    orderCase,
	}
}

// GetUserAnalyticsSummary 查询用户摘要指标
func (c *UserAnalyticsCase) GetUserAnalyticsSummary(ctx context.Context, req *commonApi.AnalyticsTimeRequest) (*adminApi.UserAnalyticsSummaryResponse, error) {
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

	return &adminApi.UserAnalyticsSummaryResponse{
		NewUserCount:            newUserCount,
		NewUserGrowthRate:       utils.CalcGrowthRate(prevNewUserCount, newUserCount),
		OrderUserCount:          orderUserCount,
		OrderUserConversionRate: utils.CalcRatio(orderUserCount, newUserCount),
		ActiveUserCount:         activeUserCount,
		ActiveUserCoverageRate:  utils.CalcRatio(activeUserCount, newUserCount),
	}, nil
}

// GetUserAnalyticsTrend 查询用户趋势
func (c *UserAnalyticsCase) GetUserAnalyticsTrend(ctx context.Context, req *commonApi.AnalyticsTimeRequest) (*commonApi.AnalyticsTrendResponse, error) {
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

	return &commonApi.AnalyticsTrendResponse{
		Axis: axis,
		Series: []*commonApi.AnalyticsTrendSeries{
			{Name: "注册用户", Type: commonApi.AnalyticsSeriesType_BAR, Data: registerRow},
			{Name: "下单用户", Type: commonApi.AnalyticsSeriesType_LINE, Data: orderUserRow},
		},
		YAxisNames: []string{"人数"},
	}, nil
}

// GetUserAnalyticsRank 查询用户行为覆盖排行
func (c *UserAnalyticsCase) GetUserAnalyticsRank(ctx context.Context, req *commonApi.AnalyticsTimeRequest) (*commonApi.AnalyticsRankResponse, error) {
	_, endAt := utils.GetAnalyticsTimeRange(req.GetTimeType())

	addressUserCount, err := c.countDistinctBehaviorUsers(ctx, models.TableNameUserAddress, endAt)
	if err != nil {
		return nil, err
	}
	var collectUserCount int64
	collectUserCount, err = c.countDistinctBehaviorUsers(ctx, models.TableNameUserCollect, endAt)
	if err != nil {
		return nil, err
	}
	var cartUserCount int64
	cartUserCount, err = c.countDistinctBehaviorUsers(ctx, models.TableNameUserCart, endAt)
	if err != nil {
		return nil, err
	}
	var orderUserCount int64
	orderUserCount, err = c.countDistinctOrderUsers(ctx, time.Time{}, endAt)
	if err != nil {
		return nil, err
	}
	var storeUserCount int64
	storeUserCount, err = c.countDistinctBehaviorUsers(ctx, models.TableNameUserStore, endAt)
	if err != nil {
		return nil, err
	}

	return &commonApi.AnalyticsRankResponse{
		Items: []*commonApi.AnalyticsRankItem{
			{Name: "已填写地址", Value: addressUserCount},
			{Name: "有收藏行为", Value: collectUserCount},
			{Name: "有加购行为", Value: cartUserCount},
			{Name: "有下单行为", Value: orderUserCount},
			{Name: "有门店申请", Value: storeUserCount},
		},
	}, nil
}

func (c *UserAnalyticsCase) countNewUsers(ctx context.Context, startAt, endAt time.Time) (int64, error) {
	var count int64
	err := c.baseUserCase.Query(ctx).BaseUser.WithContext(ctx).UnderlyingDB().
		Model(&models.BaseUser{}).
		Where("created_at >= ? AND created_at < ?", startAt, endAt).
		Count(&count).Error
	return count, err
}

func (c *UserAnalyticsCase) countDistinctOrderUsers(ctx context.Context, startAt, endAt time.Time) (int64, error) {
	var count int64
	db := c.orderCase.Query(ctx).Order.WithContext(ctx).UnderlyingDB().Model(&models.Order{})
	if !startAt.IsZero() {
		db = db.Where("created_at >= ? AND created_at < ?", startAt, endAt)
	} else {
		db = db.Where("created_at < ?", endAt)
	}
	err := db.Distinct("user_id").Count(&count).Error
	return count, err
}

// countDistinctActiveUsers 查询周期内活跃用户数。
func (c *UserAnalyticsCase) countDistinctActiveUsers(ctx context.Context, startAt, endAt time.Time) (int64, error) {
	type row struct {
		UserID int64 `gorm:"column:user_id"`
	}
	rows := make([]row, 0)
	sql := "" +
		"SELECT DISTINCT user_id FROM (" +
		" SELECT user_id FROM user_address WHERE created_at >= ? AND created_at < ?" +
		" UNION ALL" +
		" SELECT user_id FROM user_collect WHERE created_at >= ? AND created_at < ?" +
		" UNION ALL" +
		" SELECT user_id FROM user_cart WHERE created_at >= ? AND created_at < ?" +
		" UNION ALL" +
		" SELECT user_id FROM user_store WHERE created_at >= ? AND created_at < ?" +
		" UNION ALL" +
		" SELECT user_id FROM `order` WHERE created_at >= ? AND created_at < ?" +
		") t"
	// 将多种活跃行为表合并后去重，得到周期内真实活跃用户数。
	err := c.baseUserCase.Query(ctx).BaseUser.WithContext(ctx).UnderlyingDB().
		Raw(sql, startAt, endAt, startAt, endAt, startAt, endAt, startAt, endAt, startAt, endAt).
		Scan(&rows).Error
	return int64(len(rows)), err
}

func (c *UserAnalyticsCase) countDistinctBehaviorUsers(ctx context.Context, tableName string, endAt time.Time) (int64, error) {
	var count int64
	err := c.baseUserCase.Query(ctx).BaseUser.WithContext(ctx).UnderlyingDB().
		Table(tableName).
		Where("created_at < ?", endAt).
		Distinct("user_id").
		Count(&count).Error
	return count, err
}

// queryUserRegisterSummary 查询用户注册趋势汇总。
func (c *UserAnalyticsCase) queryUserRegisterSummary(ctx context.Context, timeType commonApi.AnalyticsTimeType, startAt, endAt time.Time) (map[int64]int64, []string, error) {
	type row struct {
		Key   int64 `gorm:"column:key"`
		Count int64 `gorm:"column:count"`
	}
	rows := make([]*row, 0)
	selectExpr, axis := utils.GetAnalyticsGroupExpr(timeType, startAt, endAt)
	err := c.baseUserCase.Query(ctx).BaseUser.WithContext(ctx).UnderlyingDB().
		Model(&models.BaseUser{}).
		Select(selectExpr+" AS `key`, COUNT(*) AS count").
		Where("created_at >= ? AND created_at < ?", startAt, endAt).
		Group("`key`").
		Scan(&rows).Error
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
func (c *UserAnalyticsCase) queryOrderUserSummary(ctx context.Context, timeType commonApi.AnalyticsTimeType, startAt, endAt time.Time) (map[int64]int64, error) {
	type row struct {
		Key   int64 `gorm:"column:key"`
		Count int64 `gorm:"column:count"`
	}
	rows := make([]*row, 0)
	selectExpr, _ := utils.GetAnalyticsGroupExpr(timeType, startAt, endAt)
	err := c.orderCase.Query(ctx).Order.WithContext(ctx).UnderlyingDB().
		Model(&models.Order{}).
		Select(selectExpr+" AS `key`, COUNT(DISTINCT user_id) AS count").
		Where("created_at >= ? AND created_at < ?", startAt, endAt).
		Group("`key`").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	res := make(map[int64]int64, len(rows))
	for _, item := range rows {
		res[item.Key] = item.Count
	}
	return res, nil
}
