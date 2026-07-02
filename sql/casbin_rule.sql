TRUNCATE TABLE `casbin_rule`;
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.AuthService/GetUserInfo', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.AuthService/GetUserProfile', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.AuthService/ListUserButtons', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.AuthService/SendPhoneCode', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.AuthService/TreeUserMenus', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.AuthService/UpdateUserPassword', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.AuthService/UpdateUserPhone', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.AuthService/UpdateUserProfile', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.BaseConfigService/GetBaseConfig', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.BaseConfigService/PageBaseConfigs', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.BaseConfigService/RefreshBaseConfigCache', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.BaseConfigService/UpdateBaseConfig', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.BaseDeptService/CreateBaseDept', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.BaseDeptService/DeleteBaseDept', 'DELETE', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.BaseDeptService/GetBaseDept', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.BaseDeptService/OptionBaseDepts', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.BaseDeptService/SetBaseDeptStatus', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.BaseDeptService/TreeBaseDepts', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.BaseDeptService/UpdateBaseDept', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.BaseDictService/CreateBaseDictItem', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.BaseDictService/DeleteBaseDictItem', 'DELETE', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.BaseDictService/GetBaseDictItem', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.BaseDictService/OptionBaseDicts', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.BaseDictService/PageBaseDictItems', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.BaseDictService/PageBaseDicts', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.BaseDictService/SetBaseDictItemStatus', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.BaseDictService/UpdateBaseDictItem', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.BaseMenuService/OptionBaseMenus', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.BaseRoleService/CreateBaseRole', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.BaseRoleService/DeleteBaseRole', 'DELETE', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.BaseRoleService/GetBaseRole', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.BaseRoleService/OptionBaseRoles', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.BaseRoleService/PageBaseRoles', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.BaseRoleService/SetBaseRoleMenu', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.BaseRoleService/SetBaseRoleStatus', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.BaseRoleService/UpdateBaseRole', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.BaseTenantService/CreateBaseTenant', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.BaseTenantService/DeleteBaseTenant', 'DELETE', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.BaseTenantService/GetBaseTenant', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.BaseTenantService/OptionBaseTenants', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.BaseTenantService/PageBaseTenants', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.BaseTenantService/SetBaseTenantStatus', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.BaseTenantService/UpdateBaseTenant', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.BaseUserService/CreateBaseUser', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.BaseUserService/DeleteBaseUser', 'DELETE', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.BaseUserService/GetBaseUser', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.BaseUserService/OptionBaseUsers', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.BaseUserService/PageBaseUsers', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.BaseUserService/ResetBaseUserPassword', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.BaseUserService/SetBaseUserStatus', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.BaseUserService/UpdateBaseUser', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.CommentInfoService/GetCommentInfo', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.CommentInfoService/GetGoodsCommentInfo', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.CommentInfoService/ListCommentReviews', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.CommentInfoService/PageCommentDiscussions', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.CommentInfoService/PageCommentInfos', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.CommentInfoService/SetCommentDiscussionStatus', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.CommentInfoService/SetCommentInfoStatus', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.GoodsAnalyticsService/PieGoodsAnalytics', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.GoodsAnalyticsService/RankGoodsAnalytics', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.GoodsAnalyticsService/SummaryGoodsAnalytics', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.GoodsAnalyticsService/TrendGoodsAnalytics', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.GoodsCategoryService/CreateGoodsCategory', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.GoodsCategoryService/DeleteGoodsCategory', 'DELETE', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.GoodsCategoryService/GetGoodsCategory', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.GoodsCategoryService/OptionGoodsCategories', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.GoodsCategoryService/SetGoodsCategoryStatus', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.GoodsCategoryService/TreeGoodsCategories', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.GoodsCategoryService/UpdateGoodsCategory', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.GoodsInfoService/CreateGoodsInfo', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.GoodsInfoService/DeleteGoodsInfo', 'DELETE', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.GoodsInfoService/GetGoodsInfo', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.GoodsInfoService/OptionGoodsInfos', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.GoodsInfoService/PageGoodsInfos', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.GoodsInfoService/SetGoodsInfoStatus', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.GoodsInfoService/UpdateGoodsInfo', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.GoodsPropService/CreateGoodsProp', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.GoodsPropService/DeleteGoodsProp', 'DELETE', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.GoodsPropService/GetGoodsProp', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.GoodsPropService/PageGoodsProps', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.GoodsPropService/UpdateGoodsProp', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.GoodsReportService/ListGoodsDayReports', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.GoodsReportService/ListGoodsMonthReports', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.GoodsReportService/SummaryGoodsDayReport', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.GoodsReportService/SummaryGoodsMonthReport', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.GoodsSkuService/GetGoodsSku', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.GoodsSkuService/PageGoodsSkus', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.GoodsSkuService/UpdateGoodsSku', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.GoodsSpecService/ListGoodsSpecs', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.OrderAnalyticsService/PieOrderAnalytics', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.OrderAnalyticsService/SummaryOrderAnalytics', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.OrderAnalyticsService/TrendOrderAnalytics', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.OrderInfoService/GetOrderInfo', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.OrderInfoService/GetOrderInfoRefund', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.OrderInfoService/GetOrderInfoShipment', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.OrderInfoService/PageOrderInfos', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.OrderInfoService/RefundOrderInfo', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.OrderInfoService/ShipOrderInfo', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.OrderReportService/ListOrderDayReports', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.OrderReportService/ListOrderMonthReports', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.OrderReportService/SummaryOrderDayReport', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.OrderReportService/SummaryOrderMonthReport', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.PayBillService/PagePayBills', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.ShopBannerService/CreateShopBanner', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.ShopBannerService/DeleteShopBanner', 'DELETE', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.ShopBannerService/GetShopBanner', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.ShopBannerService/PageShopBanners', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.ShopBannerService/SetShopBannerStatus', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.ShopBannerService/UpdateShopBanner', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.ShopHotService/CreateShopHot', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.ShopHotService/CreateShopHotItem', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.ShopHotService/DeleteShopHot', 'DELETE', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.ShopHotService/DeleteShopHotItem', 'DELETE', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.ShopHotService/GetShopHot', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.ShopHotService/GetShopHotItem', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.ShopHotService/PageShopHotItems', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.ShopHotService/PageShopHots', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.ShopHotService/SetShopHotItemStatus', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.ShopHotService/SetShopHotStatus', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.ShopHotService/UpdateShopHot', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.ShopHotService/UpdateShopHotItem', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.ShopServiceService/CreateShopService', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.ShopServiceService/DeleteShopService', 'DELETE', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.ShopServiceService/GetShopService', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.ShopServiceService/PageShopServices', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.ShopServiceService/SetShopServiceStatus', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.ShopServiceService/UpdateShopService', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.UserAnalyticsService/RankUserAnalytics', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.UserAnalyticsService/SummaryUserAnalytics', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.UserAnalyticsService/TrendUserAnalytics', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.UserStoreService/AuditUserStore', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.UserStoreService/GetUserStore', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.UserStoreService/PageUserStores', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.WorkspaceService/ListWorkspacePendingComments', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.WorkspaceService/SummaryWorkspaceMetrics', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.WorkspaceService/SummaryWorkspaceReputation', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.WorkspaceService/SummaryWorkspaceRisk', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/admin.v1.WorkspaceService/SummaryWorkspaceTodo', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/app.v1.AuthService/BindUserPhone', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/app.v1.AuthService/GetUserProfile', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/app.v1.AuthService/UpdateUserProfile', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/app.v1.BaseAreaService/TreeBaseAreas', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/app.v1.BaseDictService/GetBaseDict', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/app.v1.CommentService/CreateComment', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/app.v1.CommentService/CreateCommentDiscussion', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/app.v1.CommentService/DeleteComment', 'DELETE', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/app.v1.CommentService/GoodsCommentOverview', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/app.v1.CommentService/PageCommentDiscussion', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/app.v1.CommentService/PageGoodsComment', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/app.v1.CommentService/PageMyComment', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/app.v1.CommentService/PagePendingCommentGoods', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/app.v1.CommentService/SaveCommentReaction', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/app.v1.OrderInfoService/BuyNowOrderInfo', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/app.v1.OrderInfoService/CancelOrderInfo', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/app.v1.OrderInfoService/ConfirmOrderInfo', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/app.v1.OrderInfoService/CountOrderInfo', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/app.v1.OrderInfoService/CreateOrderInfo', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/app.v1.OrderInfoService/DeleteOrderInfo', 'DELETE', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/app.v1.OrderInfoService/GetOrderInfoById', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/app.v1.OrderInfoService/GetOrderInfoIdByOrderNo', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/app.v1.OrderInfoService/PageOrderInfo', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/app.v1.OrderInfoService/ReceiveOrderInfo', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/app.v1.OrderInfoService/RefundOrderInfo', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/app.v1.OrderInfoService/RepurchaseOrderInfo', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/app.v1.PayService/H5Pay', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/app.v1.PayService/JsapiPay', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/app.v1.RecommendService/BindRecommendAnonymousActor', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/app.v1.RecommendService/RecommendAnonymousActor', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/app.v1.RecommendService/RecommendEventReport', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/app.v1.RecommendService/RecommendGoods', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/app.v1.UserAddressService/CreateUserAddress', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/app.v1.UserAddressService/DeleteUserAddress', 'DELETE', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/app.v1.UserAddressService/GetUserAddress', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/app.v1.UserAddressService/ListUserAddresses', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/app.v1.UserAddressService/UpdateUserAddress', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/app.v1.UserCartService/CountUserCart', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/app.v1.UserCartService/CreateUserCart', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/app.v1.UserCartService/DeleteUserCart', 'DELETE', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/app.v1.UserCartService/ListUserCarts', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/app.v1.UserCartService/SetUserCartSelection', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/app.v1.UserCartService/SetUserCartStatus', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/app.v1.UserCartService/UpdateUserCart', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/app.v1.UserCollectService/CreateUserCollect', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/app.v1.UserCollectService/DeleteUserCollect', 'DELETE', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/app.v1.UserCollectService/GetIsCollect', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/app.v1.UserCollectService/PageUserCollects', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/app.v1.UserStoreService/CreateUserStore', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/app.v1.UserStoreService/GetUserStore', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/app.v1.UserStoreService/UpdateUserStore', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/base.v1.AiAssistantMessageService/DeleteAiAssistantMessage', 'DELETE', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/base.v1.AiAssistantMessageService/RegenerateAiAssistantMessage', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/base.v1.AiAssistantMessageService/RetryAiAssistantUserMessage', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/base.v1.AiAssistantMessageService/SendAiAssistantMessage', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/base.v1.AiAssistantMessageService/UpdateAiAssistantMessage', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/base.v1.AiAssistantService/CreateAiAssistantSession', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/base.v1.AiAssistantService/CreateAiAssistantSessionBranch', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/base.v1.AiAssistantService/DeleteAiAssistantSession', 'DELETE', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/base.v1.AiAssistantService/ListAiAssistantMessages', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/base.v1.AiAssistantService/ListAiAssistantShortcuts', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/base.v1.AiAssistantService/ListAiAssistantSessions', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/base.v1.AiAssistantService/UpdateAiAssistantSession', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/base.v1.FileService/DownloadFile', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/base.v1.FileService/MultiUploadFile', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/base.v1.FileService/UploadFile', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/base.v1.LoginService/Logout', 'DELETE', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'admin', '/base.v1.LoginService/RefreshToken', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`)
SELECT `ptype`, `v0`, 'tenant', `v2`, `v3`, `v4`, `v5`
FROM `casbin_rule`
WHERE `v0` = 'default'
  AND `v1` = 'admin'
  AND `v2` IN (
    '/admin.v1.AuthService/GetUserInfo',
    '/admin.v1.AuthService/GetUserProfile',
    '/admin.v1.AuthService/ListUserButtons',
    '/admin.v1.AuthService/SendPhoneCode',
    '/admin.v1.AuthService/TreeUserMenus',
    '/admin.v1.AuthService/UpdateUserPassword',
    '/admin.v1.AuthService/UpdateUserPhone',
    '/admin.v1.AuthService/UpdateUserProfile',
    '/admin.v1.BaseDeptService/CreateBaseDept',
    '/admin.v1.BaseDeptService/DeleteBaseDept',
    '/admin.v1.BaseDeptService/GetBaseDept',
    '/admin.v1.BaseDeptService/OptionBaseDepts',
    '/admin.v1.BaseDeptService/SetBaseDeptStatus',
    '/admin.v1.BaseDeptService/TreeBaseDepts',
    '/admin.v1.BaseDeptService/UpdateBaseDept',
    '/admin.v1.BaseDictService/OptionBaseDicts',
    '/admin.v1.BaseMenuService/OptionBaseMenus',
    '/admin.v1.BaseRoleService/CreateBaseRole',
    '/admin.v1.BaseRoleService/DeleteBaseRole',
    '/admin.v1.BaseRoleService/GetBaseRole',
    '/admin.v1.BaseRoleService/OptionBaseRoles',
    '/admin.v1.BaseRoleService/PageBaseRoles',
    '/admin.v1.BaseRoleService/SetBaseRoleMenu',
    '/admin.v1.BaseRoleService/SetBaseRoleStatus',
    '/admin.v1.BaseRoleService/UpdateBaseRole',
    '/admin.v1.BaseUserService/CreateBaseUser',
    '/admin.v1.BaseUserService/DeleteBaseUser',
    '/admin.v1.BaseUserService/GetBaseUser',
    '/admin.v1.BaseUserService/OptionBaseUsers',
    '/admin.v1.BaseUserService/PageBaseUsers',
    '/admin.v1.BaseUserService/ResetBaseUserPassword',
    '/admin.v1.BaseUserService/SetBaseUserStatus',
    '/admin.v1.BaseUserService/UpdateBaseUser',
    '/admin.v1.CommentInfoService/GetCommentInfo',
    '/admin.v1.CommentInfoService/GetGoodsCommentInfo',
    '/admin.v1.CommentInfoService/ListCommentReviews',
    '/admin.v1.CommentInfoService/PageCommentDiscussions',
    '/admin.v1.CommentInfoService/PageCommentInfos',
    '/admin.v1.CommentInfoService/SetCommentDiscussionStatus',
    '/admin.v1.CommentInfoService/SetCommentInfoStatus',
    '/admin.v1.GoodsCategoryService/CreateGoodsCategory',
    '/admin.v1.GoodsCategoryService/DeleteGoodsCategory',
    '/admin.v1.GoodsCategoryService/GetGoodsCategory',
    '/admin.v1.GoodsCategoryService/OptionGoodsCategories',
    '/admin.v1.GoodsCategoryService/SetGoodsCategoryStatus',
    '/admin.v1.GoodsCategoryService/TreeGoodsCategories',
    '/admin.v1.GoodsCategoryService/UpdateGoodsCategory',
    '/admin.v1.GoodsInfoService/CreateGoodsInfo',
    '/admin.v1.GoodsInfoService/DeleteGoodsInfo',
    '/admin.v1.GoodsInfoService/GetGoodsInfo',
    '/admin.v1.GoodsInfoService/PageGoodsInfos',
    '/admin.v1.GoodsInfoService/SetGoodsInfoStatus',
    '/admin.v1.GoodsInfoService/UpdateGoodsInfo',
    '/admin.v1.GoodsPropService/CreateGoodsProp',
    '/admin.v1.GoodsPropService/DeleteGoodsProp',
    '/admin.v1.GoodsPropService/GetGoodsProp',
    '/admin.v1.GoodsPropService/PageGoodsProps',
    '/admin.v1.GoodsPropService/UpdateGoodsProp',
    '/admin.v1.GoodsSkuService/GetGoodsSku',
    '/admin.v1.GoodsSkuService/PageGoodsSkus',
    '/admin.v1.GoodsSkuService/UpdateGoodsSku',
    '/admin.v1.GoodsSpecService/ListGoodsSpecs',
    '/admin.v1.OrderAnalyticsService/PieOrderAnalytics',
    '/admin.v1.OrderAnalyticsService/SummaryOrderAnalytics',
    '/admin.v1.OrderAnalyticsService/TrendOrderAnalytics',
    '/admin.v1.OrderInfoService/GetOrderInfo',
    '/admin.v1.OrderInfoService/GetOrderInfoRefund',
    '/admin.v1.OrderInfoService/GetOrderInfoShipment',
    '/admin.v1.OrderInfoService/PageOrderInfos',
    '/admin.v1.OrderInfoService/RefundOrderInfo',
    '/admin.v1.OrderInfoService/ShipOrderInfo',
    '/admin.v1.WorkspaceService/ListWorkspacePendingComments',
    '/admin.v1.WorkspaceService/SummaryWorkspaceMetrics',
    '/admin.v1.WorkspaceService/SummaryWorkspaceReputation',
    '/admin.v1.WorkspaceService/SummaryWorkspaceRisk',
    '/admin.v1.WorkspaceService/SummaryWorkspaceTodo',
    '/base.v1.FileService/DownloadFile',
    '/base.v1.FileService/MultiUploadFile',
    '/base.v1.FileService/UploadFile',
    '/base.v1.LoginService/Logout',
    '/base.v1.LoginService/RefreshToken'
  );
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/app.v1.AuthService/BindUserPhone', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/app.v1.AuthService/GetUserProfile', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/app.v1.AuthService/UpdateUserProfile', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/app.v1.BaseAreaService/TreeBaseAreas', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/app.v1.BaseDictService/GetBaseDict', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/app.v1.CommentService/CreateComment', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/app.v1.CommentService/CreateCommentDiscussion', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/app.v1.CommentService/DeleteComment', 'DELETE', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/app.v1.CommentService/GoodsCommentOverview', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/app.v1.CommentService/PageCommentDiscussion', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/app.v1.CommentService/PageGoodsComment', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/app.v1.CommentService/PageMyComment', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/app.v1.CommentService/PagePendingCommentGoods', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/app.v1.CommentService/SaveCommentReaction', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/app.v1.OrderInfoService/BuyNowOrderInfo', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/app.v1.OrderInfoService/CancelOrderInfo', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/app.v1.OrderInfoService/ConfirmOrderInfo', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/app.v1.OrderInfoService/CountOrderInfo', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/app.v1.OrderInfoService/CreateOrderInfo', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/app.v1.OrderInfoService/DeleteOrderInfo', 'DELETE', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/app.v1.OrderInfoService/GetOrderInfoById', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/app.v1.OrderInfoService/GetOrderInfoIdByOrderNo', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/app.v1.OrderInfoService/PageOrderInfo', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/app.v1.OrderInfoService/ReceiveOrderInfo', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/app.v1.OrderInfoService/RefundOrderInfo', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/app.v1.OrderInfoService/RepurchaseOrderInfo', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/app.v1.PayService/H5Pay', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/app.v1.PayService/JsapiPay', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/app.v1.RecommendService/BindRecommendAnonymousActor', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/app.v1.RecommendService/RecommendAnonymousActor', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/app.v1.RecommendService/RecommendEventReport', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/app.v1.RecommendService/RecommendGoods', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/app.v1.UserAddressService/CreateUserAddress', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/app.v1.UserAddressService/DeleteUserAddress', 'DELETE', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/app.v1.UserAddressService/GetUserAddress', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/app.v1.UserAddressService/ListUserAddresses', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/app.v1.UserAddressService/UpdateUserAddress', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/app.v1.UserCartService/CountUserCart', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/app.v1.UserCartService/CreateUserCart', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/app.v1.UserCartService/DeleteUserCart', 'DELETE', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/app.v1.UserCartService/ListUserCarts', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/app.v1.UserCartService/SetUserCartSelection', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/app.v1.UserCartService/SetUserCartStatus', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/app.v1.UserCartService/UpdateUserCart', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/app.v1.UserCollectService/CreateUserCollect', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/app.v1.UserCollectService/DeleteUserCollect', 'DELETE', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/app.v1.UserCollectService/GetIsCollect', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/app.v1.UserCollectService/PageUserCollects', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/app.v1.UserStoreService/CreateUserStore', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/app.v1.UserStoreService/GetUserStore', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/app.v1.UserStoreService/UpdateUserStore', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/base.v1.AiAssistantMessageService/DeleteAiAssistantMessage', 'DELETE', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/base.v1.AiAssistantMessageService/RegenerateAiAssistantMessage', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/base.v1.AiAssistantMessageService/RetryAiAssistantUserMessage', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/base.v1.AiAssistantMessageService/SendAiAssistantMessage', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/base.v1.AiAssistantMessageService/UpdateAiAssistantMessage', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/base.v1.AiAssistantService/CreateAiAssistantSession', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/base.v1.AiAssistantService/CreateAiAssistantSessionBranch', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/base.v1.AiAssistantService/DeleteAiAssistantSession', 'DELETE', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/base.v1.AiAssistantService/ListAiAssistantMessages', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/base.v1.AiAssistantService/ListAiAssistantShortcuts', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/base.v1.AiAssistantService/ListAiAssistantSessions', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/base.v1.AiAssistantService/UpdateAiAssistantSession', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/base.v1.FileService/DownloadFile', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/base.v1.FileService/MultiUploadFile', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/base.v1.FileService/UploadFile', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/base.v1.LoginService/Logout', 'DELETE', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'user', '/base.v1.LoginService/RefreshToken', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/app.v1.AuthService/BindUserPhone', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/app.v1.AuthService/GetUserProfile', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/app.v1.AuthService/UpdateUserProfile', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/app.v1.BaseAreaService/TreeBaseAreas', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/app.v1.BaseDictService/GetBaseDict', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/app.v1.CommentService/CreateComment', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/app.v1.CommentService/CreateCommentDiscussion', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/app.v1.CommentService/DeleteComment', 'DELETE', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/app.v1.CommentService/GoodsCommentOverview', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/app.v1.CommentService/PageCommentDiscussion', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/app.v1.CommentService/PageGoodsComment', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/app.v1.CommentService/PageMyComment', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/app.v1.CommentService/PagePendingCommentGoods', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/app.v1.CommentService/SaveCommentReaction', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/app.v1.OrderInfoService/BuyNowOrderInfo', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/app.v1.OrderInfoService/CancelOrderInfo', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/app.v1.OrderInfoService/ConfirmOrderInfo', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/app.v1.OrderInfoService/CountOrderInfo', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/app.v1.OrderInfoService/CreateOrderInfo', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/app.v1.OrderInfoService/DeleteOrderInfo', 'DELETE', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/app.v1.OrderInfoService/GetOrderInfoById', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/app.v1.OrderInfoService/GetOrderInfoIdByOrderNo', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/app.v1.OrderInfoService/PageOrderInfo', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/app.v1.OrderInfoService/ReceiveOrderInfo', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/app.v1.OrderInfoService/RefundOrderInfo', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/app.v1.OrderInfoService/RepurchaseOrderInfo', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/app.v1.PayService/H5Pay', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/app.v1.PayService/JsapiPay', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/app.v1.RecommendService/BindRecommendAnonymousActor', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/app.v1.RecommendService/RecommendAnonymousActor', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/app.v1.RecommendService/RecommendEventReport', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/app.v1.RecommendService/RecommendGoods', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/app.v1.UserAddressService/CreateUserAddress', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/app.v1.UserAddressService/DeleteUserAddress', 'DELETE', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/app.v1.UserAddressService/GetUserAddress', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/app.v1.UserAddressService/ListUserAddresses', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/app.v1.UserAddressService/UpdateUserAddress', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/app.v1.UserCartService/CountUserCart', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/app.v1.UserCartService/CreateUserCart', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/app.v1.UserCartService/DeleteUserCart', 'DELETE', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/app.v1.UserCartService/ListUserCarts', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/app.v1.UserCartService/SetUserCartSelection', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/app.v1.UserCartService/SetUserCartStatus', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/app.v1.UserCartService/UpdateUserCart', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/app.v1.UserCollectService/CreateUserCollect', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/app.v1.UserCollectService/DeleteUserCollect', 'DELETE', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/app.v1.UserCollectService/GetIsCollect', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/app.v1.UserCollectService/PageUserCollects', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/app.v1.UserStoreService/CreateUserStore', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/app.v1.UserStoreService/GetUserStore', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/app.v1.UserStoreService/UpdateUserStore', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/base.v1.AiAssistantMessageService/DeleteAiAssistantMessage', 'DELETE', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/base.v1.AiAssistantMessageService/RegenerateAiAssistantMessage', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/base.v1.AiAssistantMessageService/RetryAiAssistantUserMessage', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/base.v1.AiAssistantMessageService/SendAiAssistantMessage', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/base.v1.AiAssistantMessageService/UpdateAiAssistantMessage', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/base.v1.AiAssistantService/CreateAiAssistantSession', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/base.v1.AiAssistantService/CreateAiAssistantSessionBranch', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/base.v1.AiAssistantService/DeleteAiAssistantSession', 'DELETE', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/base.v1.AiAssistantService/ListAiAssistantMessages', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/base.v1.AiAssistantService/ListAiAssistantShortcuts', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/base.v1.AiAssistantService/ListAiAssistantSessions', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/base.v1.AiAssistantService/UpdateAiAssistantSession', 'PUT', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/base.v1.FileService/DownloadFile', 'GET', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/base.v1.FileService/MultiUploadFile', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/base.v1.FileService/UploadFile', 'POST', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/base.v1.LoginService/Logout', 'DELETE', '*', '');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES ('p', 'default', 'guest', '/base.v1.LoginService/RefreshToken', 'POST', '*', '');
