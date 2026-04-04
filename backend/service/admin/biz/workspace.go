package biz

import (
	"context"
	"time"

	adminApi "shop/api/gen/go/admin"
	commonApi "shop/api/gen/go/common"
	"shop/pkg/gen/models"
	"shop/service/admin/utils"
)

const lowInventoryThreshold = 10

// WorkspaceCase 工作台业务实例。
type WorkspaceCase struct {
	orderCase       *OrderCase
	baseUserCase    *BaseUserCase
	orderGoodsCase  *OrderGoodsCase
	goodsCase       *GoodsCase
	goodsSkuCase    *GoodsSkuCase
	payBillCase     *PayBillCase
	shopBannerCase  *ShopBannerCase
	shopHotCase     *ShopHotCase
	shopServiceCase *ShopServiceCase
}

// NewWorkspaceCase 创建工作台业务实例。
func NewWorkspaceCase(
	orderCase *OrderCase,
	baseUserCase *BaseUserCase,
	orderGoodsCase *OrderGoodsCase,
	goodsCase *GoodsCase,
	goodsSkuCase *GoodsSkuCase,
	payBillCase *PayBillCase,
	shopBannerCase *ShopBannerCase,
	shopHotCase *ShopHotCase,
	shopServiceCase *ShopServiceCase,
) *WorkspaceCase {
	return &WorkspaceCase{
		orderCase:       orderCase,
		baseUserCase:    baseUserCase,
		orderGoodsCase:  orderGoodsCase,
		goodsCase:       goodsCase,
		goodsSkuCase:    goodsSkuCase,
		payBillCase:     payBillCase,
		shopBannerCase:  shopBannerCase,
		shopHotCase:     shopHotCase,
		shopServiceCase: shopServiceCase,
	}
}

// GetWorkspaceMetrics 查询工作台顶部指标。
func (c *WorkspaceCase) GetWorkspaceMetrics(ctx context.Context, _ *adminApi.WorkspaceMetricsRequest) (*adminApi.WorkspaceMetricsResponse, error) {
	startAt, endAt := getTodayRange()
	prevStartAt, prevEndAt := getYesterdayRange(startAt)

	todayOrderCount, err := c.countOrderCount(ctx, startAt, endAt)
	if err != nil {
		return nil, err
	}

	prevOrderCount, err := c.countOrderCount(ctx, prevStartAt, prevEndAt)
	if err != nil {
		return nil, err
	}

	paidOrderCount, todaySaleAmount, err := c.countPaidOrderSummary(ctx, startAt, endAt)
	if err != nil {
		return nil, err
	}

	_, prevSaleAmount, err := c.countPaidOrderSummary(ctx, prevStartAt, prevEndAt)
	if err != nil {
		return nil, err
	}

	todayOrderUserCount, err := c.countDistinctOrderUsers(ctx, startAt, endAt)
	if err != nil {
		return nil, err
	}

	repurchaseUserCount, err := c.countRepurchaseUsers(ctx, startAt, endAt)
	if err != nil {
		return nil, err
	}

	todayNewUserCount, err := c.countNewUsers(ctx, startAt, endAt)
	if err != nil {
		return nil, err
	}

	todaySaleCount, err := c.countGoodsSaleNum(ctx, startAt, endAt)
	if err != nil {
		return nil, err
	}

	activeGoodsCount, err := c.countDistinctActiveGoods(ctx, startAt, endAt)
	if err != nil {
		return nil, err
	}

	todayNewGoodsCount, err := c.countNewGoods(ctx, startAt, endAt)
	if err != nil {
		return nil, err
	}

	return &adminApi.WorkspaceMetricsResponse{
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
	}, nil
}

// GetWorkspaceTodoList 查询工作台待处理事项。
func (c *WorkspaceCase) GetWorkspaceTodoList(ctx context.Context, _ *adminApi.WorkspaceTodoListRequest) (*adminApi.WorkspaceTodoListResponse, error) {
	pendingPayOrderCount, err := c.countOrderStatus(ctx, int32(commonApi.OrderStatus_CREATED))
	if err != nil {
		return nil, err
	}

	pendingShippedOrderCount, err := c.countOrderStatus(ctx, int32(commonApi.OrderStatus_PAID))
	if err != nil {
		return nil, err
	}

	lowInventorySkuCount, err := c.countLowInventorySku(ctx)
	if err != nil {
		return nil, err
	}

	pendingPutOnGoodsCount, err := c.countGoodsStatus(ctx, int32(commonApi.GoodsStatus_PULL_OFF))
	if err != nil {
		return nil, err
	}

	return &adminApi.WorkspaceTodoListResponse{
		PendingPayOrderCount:     pendingPayOrderCount,
		PendingShippedOrderCount: pendingShippedOrderCount,
		LowInventorySkuCount:     lowInventorySkuCount,
		PendingPutOnGoodsCount:   pendingPutOnGoodsCount,
	}, nil
}

// GetWorkspaceRiskList 查询工作台风险提醒。
func (c *WorkspaceCase) GetWorkspaceRiskList(ctx context.Context, _ *adminApi.WorkspaceRiskListRequest) (*adminApi.WorkspaceRiskListResponse, error) {
	abnormalPayBillCount, err := c.countAbnormalPayBill(ctx)
	if err != nil {
		return nil, err
	}

	zeroInventoryPutOnSkuCount, err := c.countZeroInventoryPutOnSku(ctx)
	if err != nil {
		return nil, err
	}

	abnormalPriceSkuCount, err := c.countAbnormalPriceSku(ctx)
	if err != nil {
		return nil, err
	}

	pendingOperationCount, err := c.countPendingOperation(ctx)
	if err != nil {
		return nil, err
	}

	return &adminApi.WorkspaceRiskListResponse{
		AbnormalPayBillCount:       abnormalPayBillCount,
		ZeroInventoryPutOnSkuCount: zeroInventoryPutOnSkuCount,
		AbnormalPriceSkuCount:      abnormalPriceSkuCount,
		PendingOperationCount:      pendingOperationCount,
	}, nil
}

// getTodayRange 返回今日起止时间。
func getTodayRange() (time.Time, time.Time) {
	now := time.Now()
	startAt := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	return startAt, startAt.AddDate(0, 0, 1)
}

// getYesterdayRange 返回昨日起止时间。
func getYesterdayRange(todayStartAt time.Time) (time.Time, time.Time) {
	startAt := todayStartAt.AddDate(0, 0, -1)
	return startAt, todayStartAt
}

// paidOrderStatuses 返回统计成交与销量时认可的订单状态。
func paidOrderStatuses() []int32 {
	return []int32{
		int32(commonApi.OrderStatus_PAID),
		int32(commonApi.OrderStatus_SHIPPED),
		int32(commonApi.OrderStatus_RECEIVED),
		int32(commonApi.OrderStatus_REFUNDING),
	}
}

// countOrderCount 统计时间范围内订单数。
func (c *WorkspaceCase) countOrderCount(ctx context.Context, startAt, endAt time.Time) (int64, error) {
	var count int64
	err := c.orderCase.Query(ctx).Order.WithContext(ctx).UnderlyingDB().
		Model(&models.Order{}).
		Where("created_at >= ? AND created_at < ?", startAt, endAt).
		Count(&count).Error
	return count, err
}

// countPaidOrderSummary 统计时间范围内已支付订单数与成交额。
func (c *WorkspaceCase) countPaidOrderSummary(ctx context.Context, startAt, endAt time.Time) (int64, int64, error) {
	type row struct {
		OrderCount int64 `gorm:"column:order_count"`
		SaleAmount int64 `gorm:"column:sale_amount"`
	}

	var result row
	statuses := paidOrderStatuses()
	err := c.orderCase.Query(ctx).Order.WithContext(ctx).UnderlyingDB().
		Model(&models.Order{}).
		Select("COUNT(*) AS order_count, COALESCE(SUM(pay_money),0) AS sale_amount").
		Where("created_at >= ? AND created_at < ?", startAt, endAt).
		Where("status IN ?", statuses).
		Scan(&result).Error
	return result.OrderCount, result.SaleAmount, err
}

// countDistinctOrderUsers 统计时间范围内下单用户数。
func (c *WorkspaceCase) countDistinctOrderUsers(ctx context.Context, startAt, endAt time.Time) (int64, error) {
	var count int64
	err := c.orderCase.Query(ctx).Order.WithContext(ctx).UnderlyingDB().
		Model(&models.Order{}).
		Where("created_at >= ? AND created_at < ?", startAt, endAt).
		Distinct("user_id").
		Count(&count).Error
	return count, err
}

// countRepurchaseUsers 统计时间范围内复购用户数。
func (c *WorkspaceCase) countRepurchaseUsers(ctx context.Context, startAt, endAt time.Time) (int64, error) {
	type row struct {
		Total int64 `gorm:"column:total"`
	}

	var result row
	sql := "" +
		"SELECT COUNT(*) AS total FROM (" +
		" SELECT user_id" +
		" FROM `order`" +
		" WHERE created_at >= ? AND created_at < ?" +
		" GROUP BY user_id" +
		" HAVING COUNT(*) >= 2" +
		") t"
	err := c.orderCase.Query(ctx).Order.WithContext(ctx).UnderlyingDB().Raw(sql, startAt, endAt).Scan(&result).Error
	return result.Total, err
}

// countNewUsers 统计时间范围内新增用户数。
func (c *WorkspaceCase) countNewUsers(ctx context.Context, startAt, endAt time.Time) (int64, error) {
	var count int64
	err := c.baseUserCase.Query(ctx).BaseUser.WithContext(ctx).UnderlyingDB().
		Model(&models.BaseUser{}).
		Where("created_at >= ? AND created_at < ?", startAt, endAt).
		Count(&count).Error
	return count, err
}

// countGoodsSaleNum 统计时间范围内商品销量。
func (c *WorkspaceCase) countGoodsSaleNum(ctx context.Context, startAt, endAt time.Time) (int64, error) {
	type row struct {
		SaleCount int64 `gorm:"column:sale_count"`
	}

	var result row
	statuses := paidOrderStatuses()
	err := c.orderGoodsCase.Query(ctx).OrderGoods.WithContext(ctx).UnderlyingDB().
		Model(&models.OrderGoods{}).
		Select("COALESCE(SUM(order_goods.num),0) AS sale_count").
		Joins("JOIN `order` ON `order`.id = order_goods.order_id").
		Where("`order`.created_at >= ? AND `order`.created_at < ?", startAt, endAt).
		Where("`order`.status IN ?", statuses).
		Scan(&result).Error
	return result.SaleCount, err
}

// countDistinctActiveGoods 统计时间范围内动销商品数。
func (c *WorkspaceCase) countDistinctActiveGoods(ctx context.Context, startAt, endAt time.Time) (int64, error) {
	var count int64
	statuses := paidOrderStatuses()
	err := c.orderGoodsCase.Query(ctx).OrderGoods.WithContext(ctx).UnderlyingDB().
		Model(&models.OrderGoods{}).
		Joins("JOIN `order` ON `order`.id = order_goods.order_id").
		Where("`order`.created_at >= ? AND `order`.created_at < ?", startAt, endAt).
		Where("`order`.status IN ?", statuses).
		Distinct("order_goods.goods_id").
		Count(&count).Error
	return count, err
}

// countNewGoods 统计时间范围内新增商品数。
func (c *WorkspaceCase) countNewGoods(ctx context.Context, startAt, endAt time.Time) (int64, error) {
	var count int64
	err := c.goodsCase.Query(ctx).Goods.WithContext(ctx).UnderlyingDB().
		Model(&models.Goods{}).
		Where("created_at >= ? AND created_at < ?", startAt, endAt).
		Count(&count).Error
	return count, err
}

// countOrderStatus 统计指定订单状态数量。
func (c *WorkspaceCase) countOrderStatus(ctx context.Context, status int32) (int64, error) {
	var count int64
	err := c.orderCase.Query(ctx).Order.WithContext(ctx).UnderlyingDB().
		Model(&models.Order{}).
		Where("status = ?", status).
		Count(&count).Error
	return count, err
}

// countLowInventorySku 统计低库存SKU数量。
func (c *WorkspaceCase) countLowInventorySku(ctx context.Context) (int64, error) {
	var count int64
	err := c.goodsSkuCase.Query(ctx).GoodsSku.WithContext(ctx).UnderlyingDB().
		Model(&models.GoodsSku{}).
		Joins("JOIN goods ON goods.id = goods_sku.goods_id").
		Where("goods.deleted_at IS NULL").
		Where("goods_sku.deleted_at IS NULL").
		Where("goods.status = ?", int32(commonApi.GoodsStatus_PUT_ON)).
		Where("goods_sku.inventory > 0 AND goods_sku.inventory <= ?", lowInventoryThreshold).
		Distinct("goods_sku.id").
		Count(&count).Error
	return count, err
}

// countGoodsStatus 统计指定商品状态数量。
func (c *WorkspaceCase) countGoodsStatus(ctx context.Context, status int32) (int64, error) {
	var count int64
	err := c.goodsCase.Query(ctx).Goods.WithContext(ctx).UnderlyingDB().
		Model(&models.Goods{}).
		Where("status = ?", status).
		Count(&count).Error
	return count, err
}

// countAbnormalPayBill 统计对账异常数量。
func (c *WorkspaceCase) countAbnormalPayBill(ctx context.Context) (int64, error) {
	var count int64
	err := c.payBillCase.Query(ctx).PayBill.WithContext(ctx).UnderlyingDB().
		Model(&models.PayBill{}).
		Where("status = ?", int32(commonApi.PayBillStatus_HAS_ERROR)).
		Count(&count).Error
	return count, err
}

// countZeroInventoryPutOnSku 统计零库存仍上架SKU数量。
func (c *WorkspaceCase) countZeroInventoryPutOnSku(ctx context.Context) (int64, error) {
	var count int64
	err := c.goodsSkuCase.Query(ctx).GoodsSku.WithContext(ctx).UnderlyingDB().
		Model(&models.GoodsSku{}).
		Joins("JOIN goods ON goods.id = goods_sku.goods_id").
		Where("goods.deleted_at IS NULL").
		Where("goods_sku.deleted_at IS NULL").
		Where("goods.status = ?", int32(commonApi.GoodsStatus_PUT_ON)).
		Where("goods_sku.inventory = 0").
		Distinct("goods_sku.id").
		Count(&count).Error
	return count, err
}

// countAbnormalPriceSku 统计价格配置异常SKU数量。
func (c *WorkspaceCase) countAbnormalPriceSku(ctx context.Context) (int64, error) {
	var count int64
	err := c.goodsSkuCase.Query(ctx).GoodsSku.WithContext(ctx).UnderlyingDB().
		Model(&models.GoodsSku{}).
		Where("price <= 0 OR discount_price < 0 OR (discount_price > 0 AND discount_price > price)").
		Count(&count).Error
	return count, err
}

// countPendingOperation 统计运营位待检查项数量。
func (c *WorkspaceCase) countPendingOperation(ctx context.Context) (int64, error) {
	disableStatus := int32(commonApi.Status_DISABLE)

	shopBannerCount, err := c.countShopBannerStatus(ctx, disableStatus)
	if err != nil {
		return 0, err
	}

	shopHotCount, err := c.countShopHotStatus(ctx, disableStatus)
	if err != nil {
		return 0, err
	}

	shopServiceCount, err := c.countShopServiceStatus(ctx, disableStatus)
	if err != nil {
		return 0, err
	}

	return shopBannerCount + shopHotCount + shopServiceCount, nil
}

// countShopBannerStatus 统计指定轮播图状态数量。
func (c *WorkspaceCase) countShopBannerStatus(ctx context.Context, status int32) (int64, error) {
	var count int64
	err := c.shopBannerCase.Query(ctx).ShopBanner.WithContext(ctx).UnderlyingDB().
		Model(&models.ShopBanner{}).
		Where("status = ?", status).
		Count(&count).Error
	return count, err
}

// countShopHotStatus 统计指定热门推荐状态数量。
func (c *WorkspaceCase) countShopHotStatus(ctx context.Context, status int32) (int64, error) {
	var count int64
	err := c.shopHotCase.Query(ctx).ShopHot.WithContext(ctx).UnderlyingDB().
		Model(&models.ShopHot{}).
		Where("status = ?", status).
		Count(&count).Error
	return count, err
}

// countShopServiceStatus 统计指定服务配置状态数量。
func (c *WorkspaceCase) countShopServiceStatus(ctx context.Context, status int32) (int64, error) {
	var count int64
	err := c.shopServiceCase.Query(ctx).ShopService.WithContext(ctx).UnderlyingDB().
		Model(&models.ShopService{}).
		Where("status = ?", status).
		Count(&count).Error
	return count, err
}
