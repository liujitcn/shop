package biz

import (
	"context"
	"encoding/json"
	"time"

	_const "shop/pkg/const"

	adminv1 "shop/api/gen/go/admin/v1"
	commonv1 "shop/api/gen/go/common/v1"
	"shop/service/admin/utils"

	"github.com/liujitcn/gorm-kit/repository"
	"gorm.io/gen/field"
)

// LOW_INVENTORY_THRESHOLD 表示工作台低库存提醒阈值。
const LOW_INVENTORY_THRESHOLD = 10

// DEFAULT_PENDING_COMMENT_LIMIT 表示工作台默认待审核评论数量。
const DEFAULT_PENDING_COMMENT_LIMIT = 5

// MAX_PENDING_COMMENT_LIMIT 表示工作台最大待审核评论数量。
const MAX_PENDING_COMMENT_LIMIT = 10

// LOW_SCORE_COMMENT_DAYS 表示工作台低分评论统计天数。
const LOW_SCORE_COMMENT_DAYS = 7

// WorkspaceCase 工作台业务实例。
type WorkspaceCase struct {
	orderInfoCase         *OrderInfoCase
	baseUserCase          *BaseUserCase
	orderGoodsCase        *OrderGoodsCase
	goodsInfoCase         *GoodsInfoCase
	goodsSKUCase          *GoodsSKUCase
	payBillCase           *PayBillCase
	commentInfoCase       *CommentInfoCase
	commentDiscussionCase *CommentDiscussionCase
	commentTagCase        *CommentTagCase
	commentSummaryCase    *CommentSummaryCase
}

// NewWorkspaceCase 创建工作台业务实例。
func NewWorkspaceCase(
	orderInfoCase *OrderInfoCase,
	baseUserCase *BaseUserCase,
	orderGoodsCase *OrderGoodsCase,
	goodsInfoCase *GoodsInfoCase,
	goodsSKUCase *GoodsSKUCase,
	payBillCase *PayBillCase,
	commentInfoCase *CommentInfoCase,
	commentDiscussionCase *CommentDiscussionCase,
	commentTagCase *CommentTagCase,
	commentSummaryCase *CommentSummaryCase,
) *WorkspaceCase {
	return &WorkspaceCase{
		orderInfoCase:         orderInfoCase,
		baseUserCase:          baseUserCase,
		orderGoodsCase:        orderGoodsCase,
		goodsInfoCase:         goodsInfoCase,
		goodsSKUCase:          goodsSKUCase,
		payBillCase:           payBillCase,
		commentInfoCase:       commentInfoCase,
		commentDiscussionCase: commentDiscussionCase,
		commentTagCase:        commentTagCase,
		commentSummaryCase:    commentSummaryCase,
	}
}

// SummaryWorkspaceMetrics 查询工作台顶部指标。
func (c *WorkspaceCase) SummaryWorkspaceMetrics(ctx context.Context, req *adminv1.SummaryWorkspaceMetricsRequest) (*adminv1.SummaryWorkspaceMetricsResponse, error) {
	now := time.Now()
	startAt := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endAt := startAt.AddDate(0, 0, 1)
	prevStartAt := startAt.AddDate(0, 0, -1)
	prevEndAt := startAt

	useGlobalTradeScope, err := c.orderInfoCase.useGlobalOrderTradeScope(ctx, req.GetTenantId(), req.GetTenantStoreId())
	if err != nil {
		return nil, err
	}
	var todayOrderCount int64
	todayOrderCount, err = c.countOrderCount(ctx, req.GetTenantId(), req.GetTenantStoreId(), startAt, endAt, useGlobalTradeScope)
	if err != nil {
		return nil, err
	}

	var prevOrderCount int64
	prevOrderCount, err = c.countOrderCount(ctx, req.GetTenantId(), req.GetTenantStoreId(), prevStartAt, prevEndAt, useGlobalTradeScope)
	if err != nil {
		return nil, err
	}

	var paidOrderCount int64
	var todaySaleAmount int64
	paidOrderCount, todaySaleAmount, err = c.countPaidOrderSummary(ctx, req.GetTenantId(), req.GetTenantStoreId(), startAt, endAt, useGlobalTradeScope)
	if err != nil {
		return nil, err
	}

	var prevSaleAmount int64
	_, prevSaleAmount, err = c.countPaidOrderSummary(ctx, req.GetTenantId(), req.GetTenantStoreId(), prevStartAt, prevEndAt, useGlobalTradeScope)
	if err != nil {
		return nil, err
	}

	var todayOrderUserCount int64
	todayOrderUserCount, err = c.countDistinctOrderUsers(ctx, req.GetTenantId(), req.GetTenantStoreId(), startAt, endAt)
	if err != nil {
		return nil, err
	}

	var repurchaseUserCount int64
	repurchaseUserCount, err = c.countRepurchaseUsers(ctx, req.GetTenantId(), req.GetTenantStoreId(), startAt, endAt, useGlobalTradeScope)
	if err != nil {
		return nil, err
	}

	var todayNewUserCount int64
	todayNewUserCount, err = c.countNewUsers(ctx, startAt, endAt)
	if err != nil {
		return nil, err
	}

	var todaySaleCount int64
	todaySaleCount, err = c.countGoodsSaleNum(ctx, req.GetTenantId(), req.GetTenantStoreId(), startAt, endAt)
	if err != nil {
		return nil, err
	}

	var activeGoodsCount int64
	activeGoodsCount, err = c.countDistinctActiveGoods(ctx, req.GetTenantId(), req.GetTenantStoreId(), startAt, endAt)
	if err != nil {
		return nil, err
	}

	var todayNewGoodsCount int64
	todayNewGoodsCount, err = c.countNewGoods(ctx, req.GetTenantId(), req.GetTenantStoreId(), startAt, endAt)
	if err != nil {
		return nil, err
	}

	var todayCommentCount int64
	todayCommentCount, err = c.countCommentCount(ctx, req.GetTenantId(), req.GetTenantStoreId(), startAt, endAt)
	if err != nil {
		return nil, err
	}

	var averageCommentScore int64
	averageCommentScore, err = c.calcAverageCommentScore(ctx, req.GetTenantId(), req.GetTenantStoreId(), endAt.AddDate(0, 0, -LOW_SCORE_COMMENT_DAYS), endAt)
	if err != nil {
		return nil, err
	}

	return &adminv1.SummaryWorkspaceMetricsResponse{
		TodayOrderCount:      todayOrderCount,
		TodayOrderGrowthRate: utils.CalcGrowthRate(prevOrderCount, todayOrderCount),
		TodaySaleAmount:      todaySaleAmount,
		AverageOrderAmount:   utils.CalcPerUnit(todaySaleAmount, paidOrderCount),
		PayConversionRate:    utils.CalcRatio(paidOrderCount, todayOrderCount),
		TodayOrderUserCount:  todayOrderUserCount,
		RepurchaseRate:       utils.CalcRatio(repurchaseUserCount, todayOrderUserCount),
		TodayNewUserCount:    todayNewUserCount,
		TodaySaleCount:       todaySaleCount,
		ActiveGoodsCount:     activeGoodsCount,
		TodayNewGoodsCount:   todayNewGoodsCount,
		TodaySaleGrowthRate:  utils.CalcGrowthRate(prevSaleAmount, todaySaleAmount),
		TodayCommentCount:    todayCommentCount,
		AverageCommentScore:  averageCommentScore,
	}, nil
}

// SummaryWorkspaceTodo 查询工作台待处理事项。
func (c *WorkspaceCase) SummaryWorkspaceTodo(ctx context.Context, req *adminv1.SummaryWorkspaceTodoRequest) (*adminv1.SummaryWorkspaceTodoResponse, error) {
	useGlobalTradeScope, err := c.orderInfoCase.useGlobalOrderTradeScope(ctx, req.GetTenantId(), req.GetTenantStoreId())
	if err != nil {
		return nil, err
	}
	var pendingPayOrderCount int64
	pendingPayOrderCount, err = c.countPendingPayOrders(ctx, req.GetTenantId(), req.GetTenantStoreId(), useGlobalTradeScope)
	if err != nil {
		return nil, err
	}

	var pendingShippedOrderCount int64
	pendingShippedOrderCount, err = c.countOrderInfoStatus(ctx, req.GetTenantId(), req.GetTenantStoreId(), _const.ORDER_INFO_STATUS_WAIT_SHIPMENT)
	if err != nil {
		return nil, err
	}

	var lowInventorySKUCount int64
	lowInventorySKUCount, err = c.countLowInventorySKU(ctx, req.GetTenantId(), req.GetTenantStoreId())
	if err != nil {
		return nil, err
	}

	var pendingPutOnGoodsCount int64
	query := c.goodsInfoCase.Query(ctx).GoodsInfo
	dao := query.WithContext(ctx).Where(query.Status.Eq(_const.GOODS_STATUS_PULL_OFF))
	if req.GetTenantId() > 0 {
		dao = dao.Where(query.TenantID.Eq(req.GetTenantId()))
	}
	if req.GetTenantStoreId() > 0 {
		dao = dao.Where(query.TenantStoreID.Eq(req.GetTenantStoreId()))
	}
	pendingPutOnGoodsCount, err = dao.Count()
	if err != nil {
		return nil, err
	}

	var pendingCommentCount int64
	pendingCommentCount, err = c.countPendingComments(ctx, req.GetTenantId(), req.GetTenantStoreId())
	if err != nil {
		return nil, err
	}

	var pendingCommentDiscussionCount int64
	pendingCommentDiscussionCount, err = c.countPendingCommentDiscussions(ctx, req.GetTenantId(), req.GetTenantStoreId())
	if err != nil {
		return nil, err
	}

	return &adminv1.SummaryWorkspaceTodoResponse{
		PendingPayOrderCount:          pendingPayOrderCount,
		PendingShippedOrderCount:      pendingShippedOrderCount,
		LowInventorySkuCount:          lowInventorySKUCount,
		PendingPutOnGoodsCount:        pendingPutOnGoodsCount,
		PendingCommentCount:           pendingCommentCount,
		PendingCommentDiscussionCount: pendingCommentDiscussionCount,
	}, nil
}

// SummaryWorkspaceRisk 查询工作台风险提醒。
func (c *WorkspaceCase) SummaryWorkspaceRisk(ctx context.Context, req *adminv1.SummaryWorkspaceRiskRequest) (*adminv1.SummaryWorkspaceRiskResponse, error) {
	abnormalPayBillCount, err := c.countWorkspaceAbnormalPayBills(ctx)
	if err != nil {
		return nil, err
	}

	var zeroInventoryPutOnSKUCount int64
	zeroInventoryPutOnSKUCount, err = c.countZeroInventoryPutOnSKU(ctx, req.GetTenantId(), req.GetTenantStoreId())
	if err != nil {
		return nil, err
	}

	var abnormalPriceSKUCount int64
	abnormalPriceSKUCount, err = c.countAbnormalPriceSKU(ctx, req.GetTenantId(), req.GetTenantStoreId())
	if err != nil {
		return nil, err
	}

	now := time.Now()
	var lowScoreCommentCount int64
	lowScoreCommentCount, err = c.countLowScoreComments(ctx, req.GetTenantId(), req.GetTenantStoreId(), now.AddDate(0, 0, -LOW_SCORE_COMMENT_DAYS), now)
	if err != nil {
		return nil, err
	}

	return &adminv1.SummaryWorkspaceRiskResponse{
		AbnormalPayBillCount:       abnormalPayBillCount,
		ZeroInventoryPutOnSkuCount: zeroInventoryPutOnSKUCount,
		AbnormalPriceSkuCount:      abnormalPriceSKUCount,
		LowScoreCommentCount:       lowScoreCommentCount,
	}, nil
}

// countWorkspaceAbnormalPayBills 统计平台账单异常，租户管理员不展示平台账单风险。
func (c *WorkspaceCase) countWorkspaceAbnormalPayBills(ctx context.Context) (int64, error) {
	authInfo, err := c.payBillCase.GetAuthInfo(ctx)
	if err != nil {
		return 0, err
	}
	// 租户管理员不维护平台支付账单，工作台不展示账单异常入口。
	if authInfo.RoleCode == _const.BASE_ROLE_CODE_TENANT {
		return 0, nil
	}

	query := c.payBillCase.Query(ctx).PayBill
	return query.WithContext(ctx).
		Where(query.Status.Eq(_const.PAY_BILL_STATUS_HAS_ERROR)).
		Count()
}

// SummaryWorkspaceReputation 查询工作台口碑洞察。
func (c *WorkspaceCase) SummaryWorkspaceReputation(ctx context.Context, req *adminv1.SummaryWorkspaceReputationRequest) (*adminv1.SummaryWorkspaceReputationResponse, error) {
	now := time.Now()
	endAt := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, 1)
	averageCommentScore, err := c.calcAverageCommentScore(ctx, req.GetTenantId(), req.GetTenantStoreId(), endAt.AddDate(0, 0, -LOW_SCORE_COMMENT_DAYS), endAt)
	if err != nil {
		return nil, err
	}

	var hotTags []*adminv1.WorkspaceReputationTag
	hotTags, err = c.listHotCommentTags(ctx, req.GetTenantId(), req.GetTenantStoreId())
	if err != nil {
		return nil, err
	}

	var commentSummary string
	commentSummary, err = c.latestCommentSummary(ctx, req.GetTenantId(), req.GetTenantStoreId())
	if err != nil {
		return nil, err
	}

	return &adminv1.SummaryWorkspaceReputationResponse{
		AverageCommentScore: averageCommentScore,
		HotTags:             hotTags,
		CommentSummary:      commentSummary,
	}, nil
}

// ListWorkspacePendingComments 查询工作台待审核评价。
func (c *WorkspaceCase) ListWorkspacePendingComments(ctx context.Context, req *adminv1.ListWorkspacePendingCommentsRequest) (*adminv1.ListWorkspacePendingCommentsResponse, error) {
	limit := int(req.GetLimit())
	if limit <= 0 {
		limit = DEFAULT_PENDING_COMMENT_LIMIT
	}
	if limit > MAX_PENDING_COMMENT_LIMIT {
		limit = MAX_PENDING_COMMENT_LIMIT
	}

	query := c.commentInfoCase.Query(ctx).CommentInfo
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Where(query.Status.Eq(_const.COMMENT_STATUS_PENDING_REVIEW)))
	if req.GetTenantId() > 0 {
		opts = append(opts, repository.Where(query.TenantID.Eq(req.GetTenantId())))
	}
	if req.GetTenantStoreId() > 0 {
		opts = append(opts, repository.Where(query.TenantStoreID.Eq(req.GetTenantStoreId())))
	}
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	opts = append(opts, repository.Limit(limit))
	list, err := c.commentInfoCase.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	pendingComments := make([]*adminv1.WorkspacePendingComment, 0, len(list))
	for _, item := range list {
		pendingComments = append(pendingComments, &adminv1.WorkspacePendingComment{
			Id:         item.ID,
			GoodsId:    item.GoodsID,
			GoodsName:  item.GoodsNameSnapshot,
			UserName:   item.UserNameSnapshot,
			GoodsScore: item.GoodsScore,
			Content:    truncateWorkspaceCommentContent(item.Content),
			CreatedAt:  item.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return &adminv1.ListWorkspacePendingCommentsResponse{PendingComments: pendingComments}, nil
}

// countDistinctOrderUsers 统计时间范围内下单用户数。
func (c *WorkspaceCase) countDistinctOrderUsers(ctx context.Context, tenantID, tenantStoreID int64, startAt, endAt time.Time) (int64, error) {
	query := c.orderInfoCase.Query(ctx).OrderInfo
	dao := query.WithContext(ctx).
		Where(
			query.CreatedAt.Gte(startAt),
			query.CreatedAt.Lt(endAt),
		)
	if tenantID > 0 {
		dao = dao.Where(query.TenantID.Eq(tenantID))
	}
	if tenantStoreID > 0 {
		dao = dao.Where(query.TenantStoreID.Eq(tenantStoreID))
	}
	count, err := dao.
		Distinct(query.UserID).
		Count()
	return count, err
}

// countRepurchaseUsers 统计时间范围内复购用户数。
func (c *WorkspaceCase) countRepurchaseUsers(ctx context.Context, tenantID, tenantStoreID int64, startAt, endAt time.Time, useGlobalTradeScope bool) (int64, error) {
	paidFacts, err := c.orderInfoCase.queryPaidOrderFacts(ctx, 0, 0, tenantID, tenantStoreID, startAt, endAt, useGlobalTradeScope)
	if err != nil {
		return 0, err
	}
	userTrades := make(map[int64]map[int64]struct{})
	for _, fact := range paidFacts {
		if userTrades[fact.UserID] == nil {
			userTrades[fact.UserID] = make(map[int64]struct{})
		}
		userTrades[fact.UserID][fact.TradeID] = struct{}{}
	}
	var count int64
	for _, tradeSet := range userTrades {
		if len(tradeSet) >= 2 {
			count++
		}
	}
	return count, nil
}

// countNewUsers 统计时间范围内新增用户数。
func (c *WorkspaceCase) countNewUsers(ctx context.Context, startAt, endAt time.Time) (int64, error) {
	query := c.baseUserCase.Query(ctx).BaseUser
	count, err := query.WithContext(ctx).
		Where(
			query.CreatedAt.Gte(startAt),
			query.CreatedAt.Lt(endAt),
		).
		Count()
	return count, err
}

// countGoodsSaleNum 统计时间范围内商品销量。
func (c *WorkspaceCase) countGoodsSaleNum(ctx context.Context, tenantID, tenantStoreID int64, startAt, endAt time.Time) (int64, error) {
	type row struct {
		SaleCount int64 `gorm:"column:sale_count"`
	}

	orderIDs, err := c.listPaidOrderIDs(ctx, tenantID, tenantStoreID, startAt, endAt)
	if err != nil {
		return 0, err
	}
	// 指定时间段内没有支付订单时，销量为 0。
	if len(orderIDs) == 0 {
		return 0, nil
	}

	var result row
	query := c.orderGoodsCase.Query(ctx).OrderGoods
	err = query.WithContext(ctx).
		Select(query.Num.Sum().FloorDiv(1).IfNull(0).As("sale_count")).
		Where(query.OrderID.In(orderIDs...)).
		Scan(&result)
	return result.SaleCount, err
}

// countDistinctActiveGoods 统计时间范围内动销商品数。
func (c *WorkspaceCase) countDistinctActiveGoods(ctx context.Context, tenantID, tenantStoreID int64, startAt, endAt time.Time) (int64, error) {
	orderIDs, err := c.listPaidOrderIDs(ctx, tenantID, tenantStoreID, startAt, endAt)
	if err != nil {
		return 0, err
	}
	// 指定时间段内没有支付订单时，动销商品数为 0。
	if len(orderIDs) == 0 {
		return 0, nil
	}

	query := c.orderGoodsCase.Query(ctx).OrderGoods
	var count int64
	count, err = query.WithContext(ctx).
		Where(query.OrderID.In(orderIDs...)).
		Distinct(query.GoodsID).
		Count()
	return count, err
}

// countNewGoods 统计时间范围内新增商品数。
func (c *WorkspaceCase) countNewGoods(ctx context.Context, tenantID, tenantStoreID int64, startAt, endAt time.Time) (int64, error) {
	query := c.goodsInfoCase.Query(ctx).GoodsInfo
	dao := query.WithContext(ctx).
		Where(
			query.CreatedAt.Gte(startAt),
			query.CreatedAt.Lt(endAt),
		)
	if tenantID > 0 {
		dao = dao.Where(query.TenantID.Eq(tenantID))
	}
	if tenantStoreID > 0 {
		dao = dao.Where(query.TenantStoreID.Eq(tenantStoreID))
	}
	count, err := dao.Count()
	return count, err
}

// countOrderCount 按当前订单口径统计时间范围内订单数。
func (c *WorkspaceCase) countOrderCount(ctx context.Context, tenantID, tenantStoreID int64, startAt, endAt time.Time, useGlobalTradeScope bool) (int64, error) {
	// 默认租户全局视角直接统计交易单，避免多门店子单重复。
	if useGlobalTradeScope {
		query := c.orderInfoCase.orderTradeRepo.Query(ctx).OrderTrade
		return query.WithContext(ctx).
			Where(
				query.CreatedAt.Gte(startAt),
				query.CreatedAt.Lt(endAt),
			).
			Count()
	}
	query := c.orderInfoCase.Query(ctx).OrderInfo
	dao := query.WithContext(ctx).
		Where(
			query.CreatedAt.Gte(startAt),
			query.CreatedAt.Lt(endAt),
		)
	if tenantID > 0 {
		dao = dao.Where(query.TenantID.Eq(tenantID))
	}
	if tenantStoreID > 0 {
		dao = dao.Where(query.TenantStoreID.Eq(tenantStoreID))
	}
	count, err := dao.Count()
	return count, err
}

// countPaidOrderSummary 按当前订单口径统计时间范围内已支付订单数与成交额。
func (c *WorkspaceCase) countPaidOrderSummary(ctx context.Context, tenantID, tenantStoreID int64, startAt, endAt time.Time, useGlobalTradeScope bool) (int64, int64, error) {
	paidFacts, err := c.orderInfoCase.queryPaidOrderFacts(ctx, 0, 0, tenantID, tenantStoreID, startAt, endAt, useGlobalTradeScope)
	if err != nil {
		return 0, 0, err
	}
	var saleAmount int64
	for _, fact := range paidFacts {
		saleAmount += fact.PayMoney
	}
	return int64(len(paidFacts)), saleAmount, nil
}

// listPaidOrderIDs 查询时间范围内支付成功口径订单编号。
func (c *WorkspaceCase) listPaidOrderIDs(ctx context.Context, tenantID, tenantStoreID int64, startAt, endAt time.Time) ([]int64, error) {
	paidFacts, err := c.orderInfoCase.queryPaidOrderFacts(ctx, 0, 0, tenantID, tenantStoreID, startAt, endAt, false)
	if err != nil {
		return nil, err
	}
	orderIDs := make([]int64, 0, len(paidFacts))
	for _, fact := range paidFacts {
		orderIDs = append(orderIDs, fact.OrderID)
	}
	return orderIDs, nil
}

// countCommentCount 统计时间范围内评价数。
func (c *WorkspaceCase) countCommentCount(ctx context.Context, tenantID, tenantStoreID int64, startAt, endAt time.Time) (int64, error) {
	query := c.commentInfoCase.Query(ctx).CommentInfo
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Where(
		query.CreatedAt.Gte(startAt),
		query.CreatedAt.Lt(endAt),
	))
	if tenantID > 0 {
		opts = append(opts, repository.Where(query.TenantID.Eq(tenantID)))
	}
	if tenantStoreID > 0 {
		opts = append(opts, repository.Where(query.TenantStoreID.Eq(tenantStoreID)))
	}
	return c.commentInfoCase.Count(ctx, opts...)
}

// calcAverageCommentScore 计算时间范围内评价平均分，返回十分位。
func (c *WorkspaceCase) calcAverageCommentScore(ctx context.Context, tenantID, tenantStoreID int64, startAt, endAt time.Time) (int64, error) {
	query := c.commentInfoCase.Query(ctx).CommentInfo
	scores := make([]int32, 0)
	dao := query.WithContext(ctx).
		Where(
			query.CreatedAt.Gte(startAt),
			query.CreatedAt.Lt(endAt),
			query.Status.Eq(_const.COMMENT_STATUS_APPROVED),
		)
	if tenantID > 0 {
		dao = dao.Where(query.TenantID.Eq(tenantID))
	}
	if tenantStoreID > 0 {
		dao = dao.Where(query.TenantStoreID.Eq(tenantStoreID))
	}
	err := dao.Pluck(query.GoodsScore, &scores)
	if err != nil {
		return 0, err
	}
	if len(scores) == 0 {
		return 0, nil
	}

	var total int64
	for _, score := range scores {
		total += int64(score)
	}
	return total * 10 / int64(len(scores)), nil
}

// countPendingPayOrders 按当前订单口径统计待支付订单数量。
func (c *WorkspaceCase) countPendingPayOrders(ctx context.Context, tenantID, tenantStoreID int64, useGlobalTradeScope bool) (int64, error) {
	// 默认租户全局视角按待支付与支付中交易单统计，避免多门店子单重复。
	if useGlobalTradeScope {
		query := c.orderInfoCase.orderTradeRepo.Query(ctx).OrderTrade
		return query.WithContext(ctx).
			Where(query.Status.In(
				_const.ORDER_TRADE_STATUS_PENDING_PAYMENT,
				_const.ORDER_TRADE_STATUS_PAYING,
			)).
			Count()
	}
	return c.countOrderInfoStatus(ctx, tenantID, tenantStoreID, _const.ORDER_INFO_STATUS_NOT_STARTED)
}

// countOrderInfoStatus 统计指定履约状态的门店订单数量。
func (c *WorkspaceCase) countOrderInfoStatus(ctx context.Context, tenantID, tenantStoreID int64, status int32) (int64, error) {
	query := c.orderInfoCase.Query(ctx).OrderInfo
	dao := query.WithContext(ctx).Where(query.Status.Eq(status))
	if tenantID > 0 {
		dao = dao.Where(query.TenantID.Eq(tenantID))
	}
	if tenantStoreID > 0 {
		dao = dao.Where(query.TenantStoreID.Eq(tenantStoreID))
	}
	count, err := dao.Count()
	return count, err
}

// countLowInventorySKU 统计低库存SKU数量。
func (c *WorkspaceCase) countLowInventorySKU(ctx context.Context, tenantID, tenantStoreID int64) (int64, error) {
	goodsIDs, err := c.listPutOnGoodsIDs(ctx, tenantID, tenantStoreID)
	if err != nil {
		return 0, err
	}
	// 没有上架商品时，不需要继续统计库存。
	if len(goodsIDs) == 0 {
		return 0, nil
	}

	query := c.goodsSKUCase.Query(ctx).GoodsSKU
	dao := query.WithContext(ctx).
		Where(
			query.DeletedAt.IsNull(),
			query.GoodsID.In(goodsIDs...),
			query.Inventory.Gt(0),
			query.Inventory.Lte(LOW_INVENTORY_THRESHOLD),
		)
	if tenantID > 0 {
		dao = dao.Where(query.TenantID.Eq(tenantID))
	}
	if tenantStoreID > 0 {
		dao = dao.Where(query.TenantStoreID.Eq(tenantStoreID))
	}
	var count int64
	count, err = dao.Count()
	return count, err
}

// listPutOnGoodsIDs 查询当前上架商品编号。
func (c *WorkspaceCase) listPutOnGoodsIDs(ctx context.Context, tenantID, tenantStoreID int64) ([]int64, error) {
	query := c.goodsInfoCase.Query(ctx).GoodsInfo
	goodsIDs := make([]int64, 0)
	dao := query.WithContext(ctx).
		Where(
			query.DeletedAt.IsNull(),
			query.Status.Eq(_const.GOODS_STATUS_PUT_ON),
		)
	if tenantID > 0 {
		dao = dao.Where(query.TenantID.Eq(tenantID))
	}
	if tenantStoreID > 0 {
		dao = dao.Where(query.TenantStoreID.Eq(tenantStoreID))
	}
	err := dao.Pluck(query.ID, &goodsIDs)
	return goodsIDs, err
}

// countPendingComments 统计待审核评价数。
func (c *WorkspaceCase) countPendingComments(ctx context.Context, tenantID, tenantStoreID int64) (int64, error) {
	query := c.commentInfoCase.Query(ctx).CommentInfo
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Where(query.Status.Eq(_const.COMMENT_STATUS_PENDING_REVIEW)))
	if tenantID > 0 {
		opts = append(opts, repository.Where(query.TenantID.Eq(tenantID)))
	}
	if tenantStoreID > 0 {
		opts = append(opts, repository.Where(query.TenantStoreID.Eq(tenantStoreID)))
	}
	return c.commentInfoCase.Count(ctx, opts...)
}

// countPendingCommentDiscussions 统计待审核评价讨论数。
func (c *WorkspaceCase) countPendingCommentDiscussions(ctx context.Context, tenantID, tenantStoreID int64) (int64, error) {
	type row struct {
		PendingCount int64 `gorm:"column:pending_count"`
	}

	query := c.commentInfoCase.Query(ctx).CommentInfo
	dao := query.WithContext(ctx).
		Select(query.PendingDiscussionCount.Sum().FloorDiv(1).IfNull(0).As("pending_count")).
		Where(query.PendingDiscussionCount.Gt(0))
	if tenantID > 0 {
		dao = dao.Where(query.TenantID.Eq(tenantID))
	}
	if tenantStoreID > 0 {
		dao = dao.Where(query.TenantStoreID.Eq(tenantStoreID))
	}
	var result row
	err := dao.Scan(&result)
	return result.PendingCount, err
}

// countZeroInventoryPutOnSKU 统计零库存仍上架SKU数量。
func (c *WorkspaceCase) countZeroInventoryPutOnSKU(ctx context.Context, tenantID, tenantStoreID int64) (int64, error) {
	goodsIDs, err := c.listPutOnGoodsIDs(ctx, tenantID, tenantStoreID)
	if err != nil {
		return 0, err
	}
	// 没有上架商品时，不需要继续统计库存。
	if len(goodsIDs) == 0 {
		return 0, nil
	}

	query := c.goodsSKUCase.Query(ctx).GoodsSKU
	dao := query.WithContext(ctx).
		Where(
			query.DeletedAt.IsNull(),
			query.GoodsID.In(goodsIDs...),
			query.Inventory.Eq(0),
		)
	if tenantID > 0 {
		dao = dao.Where(query.TenantID.Eq(tenantID))
	}
	if tenantStoreID > 0 {
		dao = dao.Where(query.TenantStoreID.Eq(tenantStoreID))
	}
	var count int64
	count, err = dao.Count()
	return count, err
}

// countAbnormalPriceSKU 统计价格配置异常SKU数量。
func (c *WorkspaceCase) countAbnormalPriceSKU(ctx context.Context, tenantID, tenantStoreID int64) (int64, error) {
	query := c.goodsSKUCase.Query(ctx).GoodsSKU
	dao := query.WithContext(ctx).
		Where(
			field.Or(
				query.Price.Lte(0),
				query.DiscountPrice.Lt(0),
				field.And(
					query.DiscountPrice.Gt(0),
					query.DiscountPrice.GtCol(query.Price),
				),
			),
		)
	if tenantID > 0 {
		dao = dao.Where(query.TenantID.Eq(tenantID))
	}
	if tenantStoreID > 0 {
		dao = dao.Where(query.TenantStoreID.Eq(tenantStoreID))
	}
	count, err := dao.Count()
	return count, err
}

// countLowScoreComments 统计近期低分评价数。
func (c *WorkspaceCase) countLowScoreComments(ctx context.Context, tenantID, tenantStoreID int64, startAt, endAt time.Time) (int64, error) {
	query := c.commentInfoCase.Query(ctx).CommentInfo
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Where(
		query.CreatedAt.Gte(startAt),
		query.CreatedAt.Lt(endAt),
		query.GoodsScore.Lte(2),
		query.Status.In(_const.COMMENT_STATUS_PENDING_REVIEW, _const.COMMENT_STATUS_APPROVED),
	))
	if tenantID > 0 {
		opts = append(opts, repository.Where(query.TenantID.Eq(tenantID)))
	}
	if tenantStoreID > 0 {
		opts = append(opts, repository.Where(query.TenantStoreID.Eq(tenantStoreID)))
	}
	return c.commentInfoCase.Count(ctx, opts...)
}

// listHotCommentTags 查询工作台高频评价标签。
func (c *WorkspaceCase) listHotCommentTags(ctx context.Context, tenantID, tenantStoreID int64) ([]*adminv1.WorkspaceReputationTag, error) {
	query := c.commentTagCase.Query(ctx).CommentTag
	opts := make([]repository.QueryOption, 0, 5)
	opts = append(opts, repository.Where(query.MentionCount.Gt(0)))
	if tenantID > 0 {
		opts = append(opts, repository.Where(query.TenantID.Eq(tenantID)))
	}
	if tenantStoreID > 0 {
		opts = append(opts, repository.Where(query.TenantStoreID.Eq(tenantStoreID)))
	}
	opts = append(opts, repository.Order(query.MentionCount.Desc()))
	opts = append(opts, repository.Limit(3))
	list, err := c.commentTagCase.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	tags := make([]*adminv1.WorkspaceReputationTag, 0, len(list))
	for _, item := range list {
		tags = append(tags, &adminv1.WorkspaceReputationTag{
			Name:         item.Name,
			MentionCount: int64(item.MentionCount),
		})
	}
	return tags, nil
}

// latestCommentSummary 查询最近的评价摘要内容。
func (c *WorkspaceCase) latestCommentSummary(ctx context.Context, tenantID, tenantStoreID int64) (string, error) {
	query := c.commentSummaryCase.Query(ctx).CommentSummary
	opts := make([]repository.QueryOption, 0, 5)
	opts = append(opts, repository.Where(query.Scene.Eq(_const.COMMENT_SUMMARY_SCENE_OVERVIEW)))
	if tenantID > 0 {
		opts = append(opts, repository.Where(query.TenantID.Eq(tenantID)))
	}
	if tenantStoreID > 0 {
		opts = append(opts, repository.Where(query.TenantStoreID.Eq(tenantStoreID)))
	}
	opts = append(opts, repository.Order(query.UpdatedAt.Desc()))
	opts = append(opts, repository.Limit(1))
	list, err := c.commentSummaryCase.List(ctx, opts...)
	if err != nil {
		return "", err
	}
	if len(list) == 0 {
		return "暂无评价摘要", nil
	}

	var items []*commonv1.CommentSummaryContentItem
	err = json.Unmarshal([]byte(list[0].Content), &items)
	if err != nil || len(items) == 0 {
		return "暂无评价摘要", nil
	}
	return items[0].Content, nil
}

// truncateWorkspaceCommentContent 截断工作台评价摘要。
func truncateWorkspaceCommentContent(content string) string {
	runes := []rune(content)
	if len(runes) <= 28 {
		return content
	}
	return string(runes[:28]) + "..."
}
