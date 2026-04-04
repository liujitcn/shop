package admin

import (
	"shop/service/admin/biz"
	"shop/service/admin/task"
	"shop/service/admin/wx"

	"github.com/google/wire"
)

// ProviderSet is server providers.
var ProviderSet = wire.NewSet(

	biz.NewAuthCase,
	biz.NewBaseApiCase,
	biz.NewBaseConfigCase,
	biz.NewBaseDeptCase,
	biz.NewBaseDictCase,
	biz.NewBaseDictItemCase,
	biz.NewBaseJobCase,
	biz.NewBaseJobLogCase,
	biz.NewBaseLogCase,
	biz.NewBaseMenuCase,
	biz.NewBaseRoleCase,
	biz.NewBaseUserCase,
	biz.NewCasbinRuleCase,
	biz.NewGoodsInfoCase,
	biz.NewGoodsAnalyticsCase,
	biz.NewGoodsCategoryCase,
	biz.NewGoodsPropCase,
	biz.NewGoodsSkuCase,
	biz.NewGoodsSpecCase,
	biz.NewOrderInfoCase,
	biz.NewOrderAnalyticsCase,
	biz.NewOrderAddressCase,
	biz.NewOrderCancelCase,
	biz.NewOrderGoodsCase,
	biz.NewOrderLogisticsCase,
	biz.NewOrderPaymentCase,
	biz.NewOrderReportCase,
	biz.NewOrderRefundCase,
	biz.NewPayBillCase,
	biz.NewShopBannerCase,
	biz.NewShopHotCase,
	biz.NewShopHotItemCase,
	biz.NewShopServiceCase,
	biz.NewUserAnalyticsCase,
	biz.NewUserStoreCase,
	biz.NewWorkspaceCase,

	task.NewOrderStatDay,
	task.NewTradeBill,
	task.NewTaskList,

	wx.NewWxPayCase,

	NewAuthService,
	NewBaseApiService,
	NewBaseConfigService,
	NewBaseDeptService,
	NewBaseDictService,
	NewBaseJobService,
	NewBaseLogService,
	NewBaseMenuService,
	NewBaseRoleService,
	NewBaseUserService,
	NewGoodsAnalyticsService,
	NewGoodsCategoryService,
	NewGoodsPropService,
	NewGoodsInfoService,
	NewGoodsSkuService,
	NewGoodsSpecService,
	NewOrderAnalyticsService,
	NewOrderReportService,
	NewOrderInfoService,
	NewPayBillService,
	NewShopBannerService,
	NewShopHotService,
	NewShopServiceService,
	NewUserAnalyticsService,
	NewUserStoreService,
	NewWorkspaceService,
)
