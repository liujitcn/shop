package app

import (
	"shop/service/shop/app/biz"

	"github.com/google/wire"
)

// ProviderSet 汇总商城端服务依赖注入提供者。
var ProviderSet = wire.NewSet(
	biz.NewCommentInfoCase,
	biz.NewCommentSummaryCase,
	biz.NewCommentTagCase,
	biz.NewCommentReviewCase,
	biz.NewCommentDiscussionCase,
	biz.NewCommentReactionCase,
	biz.NewCommentCase,
	biz.NewGoodsCategoryCase,
	biz.NewGoodsPropCase,
	biz.NewGoodsSKUCase,
	biz.NewGoodsSpecCase,
	biz.NewGoodsInfoCase,
	biz.NewTenantStoreCase,
	biz.NewOrderTradeCase,
	biz.NewOrderInfoCase,
	biz.NewOrderAddressCase,
	biz.NewOrderCancelCase,
	biz.NewOrderGoodsCase,
	biz.NewOrderLogisticsCase,
	biz.NewOrderPaymentCase,
	biz.NewOrderRefundCase,
	biz.NewOrderInventoryCase,
	biz.NewOrderRefundResultCase,
	biz.NewOrderSchedulerCase,
	biz.NewPayCase,
	biz.NewRecommendRequestCase,
	biz.NewRecommendEventCase,
	biz.NewRecommendCase,
	biz.NewShopBannerCase,
	biz.NewShopHotCase,
	biz.NewShopHotItemCase,
	biz.NewShopServiceCase,
	biz.NewUserAddressCase,
	biz.NewUserCartCase,
	biz.NewUserCollectCase,
	biz.NewUserStoreCase,
	biz.NewCommentAuditRetry,
	biz.NewOrderRefundRetry,

	NewCommentInfoService,
	NewGoodsCategoryService,
	NewGoodsInfoService,
	NewTenantStoreService,
	NewOrderInfoService,
	NewPayService,
	NewRecommendService,
	NewShopBannerService,
	NewShopHotService,
	NewShopServiceService,
	NewUserAddressService,
	NewUserCartService,
	NewUserCollectService,
	NewUserStoreService,
)
