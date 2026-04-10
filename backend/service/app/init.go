package app

import (
	"shop/service/app/biz"
	"shop/service/app/wx"

	"github.com/google/wire"
)

// ProviderSet is server providers.
var ProviderSet = wire.NewSet(
	biz.NewAuthCase,
	biz.NewBaseAreaCase,
	biz.NewBaseDeptCase,
	biz.NewBaseDictCase,
	biz.NewBaseDictItemCase,
	biz.NewBaseRoleCase,
	biz.NewBaseUserCase,
	biz.NewGoodsCategoryCase,
	biz.NewGoodsPropCase,
	biz.NewGoodsSkuCase,
	biz.NewGoodsSpecCase,
	biz.NewRecommendGoodsRelationCase,
	biz.NewRecommendGoodsActionCase,
	biz.NewRecommendUserGoodsPreferenceCase,
	biz.NewRecommendUserPreferenceCase,
	biz.NewRecommendExposureCase,
	biz.NewGoodsStatDayCase,
	biz.NewRecommendGoodsStatDayCase,
	biz.NewGoodsInfoCase,
	biz.NewOrderInfoCase,
	biz.NewOrderAddressCase,
	biz.NewOrderCancelCase,
	biz.NewOrderGoodsCase,
	biz.NewOrderLogisticsCase,
	biz.NewOrderPaymentCase,
	biz.NewOrderRefundCase,
	biz.NewOrderSchedulerCase,
	biz.NewPayCase,
	biz.NewRecommendRequestCase,
	biz.NewRecommendCase,
	biz.NewShopBannerCase,
	biz.NewShopHotCase,
	biz.NewShopHotItemCase,
	biz.NewShopServiceCase,
	biz.NewUserAddressCase,
	biz.NewUserCartCase,
	biz.NewUserCollectCase,
	biz.NewUserStoreCase,

	wx.NewWxPayCase,

	NewAuthService,
	NewBaseAreaService,
	NewBaseDictService,
	NewGoodsCategoryService,
	NewGoodsInfoService,
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
