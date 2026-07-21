// Package admin 注册 shop.admin.v1 传输层服务。
package admin

import (
	shopadminv1 "shop/api/gen/go/shop/admin/v1"
	einoTool "shop/pkg/agent/eino/tool"
	"shop/pkg/job"
	host "shop/server"
	shopadmin "shop/service/shop/admin"
	shopadminbiz "shop/service/shop/admin/biz"
	"shop/service/shop/recommend"

	"github.com/go-kratos/kratos/v3/transport/grpc"
	kratosHTTP "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/google/wire"
	mcpserver "github.com/liujitcn/kratos-kit/transport/mcp"
)

// Services 汇总 shop.admin.v1 的服务实现。
type Services struct {
	CommentInfo      *shopadmin.CommentInfoService
	TenantStore      *shopadmin.TenantStoreService
	GoodsAnalytics   *shopadmin.GoodsAnalyticsService
	GoodsReport      *shopadmin.GoodsReportService
	GoodsCategory    *shopadmin.GoodsCategoryService
	GoodsProp        *shopadmin.GoodsPropService
	Goods            *shopadmin.GoodsInfoService
	GoodsSKU         *shopadmin.GoodsSkuService
	GoodsSpec        *shopadmin.GoodsSpecService
	OrderAnalytics   *shopadmin.OrderAnalyticsService
	OrderReport      *shopadmin.OrderReportService
	Order            *shopadmin.OrderInfoService
	PayBill          *shopadmin.PayBillService
	RecommendRequest *shopadmin.RecommendRequestService
	RecommendGorse   *shopadmin.RecommendGorseService
	ShopBanner       *shopadmin.ShopBannerService
	ShopHot          *shopadmin.ShopHotService
	ShopService      *shopadmin.ShopServiceService
	UserAnalytics    *shopadmin.UserAnalyticsService
	UserStore        *shopadmin.UserStoreService
	Workspace        *shopadmin.WorkspaceService
}

var _ host.Module = Services{}

// TaskSet 汇总商城管理端向调度运行时贡献的任务。
type TaskSet struct {
	tradeBill     *shopadminbiz.TradeBill
	orderStatDay  *shopadminbiz.OrderStatDay
	goodsStatDay  *shopadminbiz.GoodsStatDay
	recommendSync *recommend.RecommendSync
}

var _ host.TaskContributor = TaskSet{}

// NewTaskSet 创建商城管理端定时任务贡献。
func NewTaskSet(
	tradeBill *shopadminbiz.TradeBill,
	orderStatDay *shopadminbiz.OrderStatDay,
	goodsStatDay *shopadminbiz.GoodsStatDay,
	recommendSync *recommend.RecommendSync,
) TaskSet {
	return TaskSet{
		tradeBill:     tradeBill,
		orderStatDay:  orderStatDay,
		goodsStatDay:  goodsStatDay,
		recommendSync: recommendSync,
	}
}

// ProviderSet 汇总 shop.admin.v1 传输模块依赖注入提供者。
var ProviderSet = wire.NewSet(wire.Struct(new(Services), "*"))

// RegisterGRPC 注册 shop.admin.v1 的 gRPC 服务。
func (s Services) RegisterGRPC(srv *grpc.Server) {
	shopadminv1.RegisterCommentInfoServiceServer(srv, s.CommentInfo)
	shopadminv1.RegisterTenantStoreServiceServer(srv, s.TenantStore)
	shopadminv1.RegisterGoodsAnalyticsServiceServer(srv, s.GoodsAnalytics)
	shopadminv1.RegisterGoodsReportServiceServer(srv, s.GoodsReport)
	shopadminv1.RegisterGoodsCategoryServiceServer(srv, s.GoodsCategory)
	shopadminv1.RegisterGoodsPropServiceServer(srv, s.GoodsProp)
	shopadminv1.RegisterGoodsInfoServiceServer(srv, s.Goods)
	shopadminv1.RegisterGoodsSkuServiceServer(srv, s.GoodsSKU)
	shopadminv1.RegisterGoodsSpecServiceServer(srv, s.GoodsSpec)
	shopadminv1.RegisterOrderAnalyticsServiceServer(srv, s.OrderAnalytics)
	shopadminv1.RegisterOrderReportServiceServer(srv, s.OrderReport)
	shopadminv1.RegisterOrderInfoServiceServer(srv, s.Order)
	shopadminv1.RegisterPayBillServiceServer(srv, s.PayBill)
	shopadminv1.RegisterRecommendRequestServiceServer(srv, s.RecommendRequest)
	shopadminv1.RegisterRecommendGorseServiceServer(srv, s.RecommendGorse)
	shopadminv1.RegisterShopBannerServiceServer(srv, s.ShopBanner)
	shopadminv1.RegisterShopHotServiceServer(srv, s.ShopHot)
	shopadminv1.RegisterShopServiceServiceServer(srv, s.ShopService)
	shopadminv1.RegisterUserAnalyticsServiceServer(srv, s.UserAnalytics)
	shopadminv1.RegisterUserStoreServiceServer(srv, s.UserStore)
	shopadminv1.RegisterWorkspaceServiceServer(srv, s.Workspace)
}

// RegisterHTTP 注册 shop.admin.v1 的 HTTP 服务。
func (s Services) RegisterHTTP(srv *kratosHTTP.Server) {
	shopadminv1.RegisterCommentInfoServiceHTTPServer(srv, s.CommentInfo)
	shopadminv1.RegisterTenantStoreServiceHTTPServer(srv, s.TenantStore)
	shopadminv1.RegisterGoodsAnalyticsServiceHTTPServer(srv, s.GoodsAnalytics)
	shopadminv1.RegisterGoodsReportServiceHTTPServer(srv, s.GoodsReport)
	shopadminv1.RegisterGoodsCategoryServiceHTTPServer(srv, s.GoodsCategory)
	shopadminv1.RegisterGoodsPropServiceHTTPServer(srv, s.GoodsProp)
	shopadminv1.RegisterGoodsInfoServiceHTTPServer(srv, s.Goods)
	shopadminv1.RegisterGoodsSkuServiceHTTPServer(srv, s.GoodsSKU)
	shopadminv1.RegisterGoodsSpecServiceHTTPServer(srv, s.GoodsSpec)
	shopadminv1.RegisterOrderAnalyticsServiceHTTPServer(srv, s.OrderAnalytics)
	shopadminv1.RegisterOrderReportServiceHTTPServer(srv, s.OrderReport)
	shopadminv1.RegisterOrderInfoServiceHTTPServer(srv, s.Order)
	shopadminv1.RegisterPayBillServiceHTTPServer(srv, s.PayBill)
	shopadminv1.RegisterRecommendRequestServiceHTTPServer(srv, s.RecommendRequest)
	shopadminv1.RegisterRecommendGorseServiceHTTPServer(srv, s.RecommendGorse)
	shopadminv1.RegisterShopBannerServiceHTTPServer(srv, s.ShopBanner)
	shopadminv1.RegisterShopHotServiceHTTPServer(srv, s.ShopHot)
	shopadminv1.RegisterShopServiceServiceHTTPServer(srv, s.ShopService)
	shopadminv1.RegisterUserAnalyticsServiceHTTPServer(srv, s.UserAnalytics)
	shopadminv1.RegisterUserStoreServiceHTTPServer(srv, s.UserStore)
	shopadminv1.RegisterWorkspaceServiceHTTPServer(srv, s.Workspace)
}

// RegisterMCP 注册 shop.admin.v1 的 MCP 工具。
func (s Services) RegisterMCP(server *mcpserver.Server) {
	mcpSrv := server.MCPServer()
	shopadminv1.RegisterCommentInfoServiceMCPTools(mcpSrv, s.CommentInfo)
	shopadminv1.RegisterTenantStoreServiceMCPTools(mcpSrv, s.TenantStore)
	shopadminv1.RegisterGoodsAnalyticsServiceMCPTools(mcpSrv, s.GoodsAnalytics)
	shopadminv1.RegisterGoodsReportServiceMCPTools(mcpSrv, s.GoodsReport)
	shopadminv1.RegisterGoodsCategoryServiceMCPTools(mcpSrv, s.GoodsCategory)
	shopadminv1.RegisterGoodsPropServiceMCPTools(mcpSrv, s.GoodsProp)
	shopadminv1.RegisterGoodsInfoServiceMCPTools(mcpSrv, s.Goods)
	shopadminv1.RegisterGoodsSkuServiceMCPTools(mcpSrv, s.GoodsSKU)
	shopadminv1.RegisterGoodsSpecServiceMCPTools(mcpSrv, s.GoodsSpec)
	shopadminv1.RegisterOrderAnalyticsServiceMCPTools(mcpSrv, s.OrderAnalytics)
	shopadminv1.RegisterOrderReportServiceMCPTools(mcpSrv, s.OrderReport)
	shopadminv1.RegisterOrderInfoServiceMCPTools(mcpSrv, s.Order)
	shopadminv1.RegisterPayBillServiceMCPTools(mcpSrv, s.PayBill)
	shopadminv1.RegisterRecommendRequestServiceMCPTools(mcpSrv, s.RecommendRequest)
	shopadminv1.RegisterRecommendGorseServiceMCPTools(mcpSrv, s.RecommendGorse)
	shopadminv1.RegisterShopBannerServiceMCPTools(mcpSrv, s.ShopBanner)
	shopadminv1.RegisterShopHotServiceMCPTools(mcpSrv, s.ShopHot)
	shopadminv1.RegisterShopServiceServiceMCPTools(mcpSrv, s.ShopService)
	shopadminv1.RegisterUserAnalyticsServiceMCPTools(mcpSrv, s.UserAnalytics)
	shopadminv1.RegisterUserStoreServiceMCPTools(mcpSrv, s.UserStore)
	shopadminv1.RegisterWorkspaceServiceMCPTools(mcpSrv, s.Workspace)
}

// AdminAgentTools 创建 shop.admin.v1 的管理端 AI 助手工具。
func (s Services) AdminAgentTools() ([]einoTool.Invokable, error) {
	var tools []einoTool.Invokable
	appendTools := func(values []einoTool.Invokable, err error) error {
		if err != nil {
			return err
		}
		tools = append(tools, values...)
		return nil
	}
	appendTool := func(value einoTool.Invokable, err error) error {
		if err != nil {
			return err
		}
		tools = append(tools, value)
		return nil
	}
	if err := appendTools(shopadminv1.NewWorkspaceServiceAgentTools(s.Workspace)); err != nil {
		return nil, err
	}
	if err := appendTool(shopadminv1.NewOrderInfoServicePageOrderInfoAgentTool(s.Order)); err != nil {
		return nil, err
	}
	if err := appendTool(shopadminv1.NewOrderInfoServiceGetOrderInfoAgentTool(s.Order)); err != nil {
		return nil, err
	}
	if err := appendTool(shopadminv1.NewOrderInfoServiceGetOrderInfoRefundAgentTool(s.Order)); err != nil {
		return nil, err
	}
	if err := appendTool(shopadminv1.NewOrderInfoServiceGetOrderInfoShipmentAgentTool(s.Order)); err != nil {
		return nil, err
	}
	if err := appendTool(shopadminv1.NewOrderInfoServiceShipOrderInfoAgentTool(s.Order)); err != nil {
		return nil, err
	}
	if err := appendTools(shopadminv1.NewCommentInfoServiceAgentTools(s.CommentInfo)); err != nil {
		return nil, err
	}
	if err := appendTools(shopadminv1.NewGoodsInfoServiceAgentTools(s.Goods)); err != nil {
		return nil, err
	}
	if err := appendTools(shopadminv1.NewTenantStoreServiceAgentTools(s.TenantStore)); err != nil {
		return nil, err
	}
	if err := appendTools(shopadminv1.NewGoodsAnalyticsServiceAgentTools(s.GoodsAnalytics)); err != nil {
		return nil, err
	}
	if err := appendTools(shopadminv1.NewOrderAnalyticsServiceAgentTools(s.OrderAnalytics)); err != nil {
		return nil, err
	}
	if err := appendTools(shopadminv1.NewUserStoreServiceAgentTools(s.UserStore)); err != nil {
		return nil, err
	}
	if err := appendTools(shopadminv1.NewRecommendGorseServiceAgentTools(s.RecommendGorse)); err != nil {
		return nil, err
	}
	if err := appendTools(shopadminv1.NewPayBillServiceAgentTools(s.PayBill)); err != nil {
		return nil, err
	}
	if err := appendTools(shopadminv1.NewOrderReportServiceAgentTools(s.OrderReport)); err != nil {
		return nil, err
	}
	if err := appendTools(shopadminv1.NewGoodsReportServiceAgentTools(s.GoodsReport)); err != nil {
		return nil, err
	}
	if err := appendTools(shopadminv1.NewUserAnalyticsServiceAgentTools(s.UserAnalytics)); err != nil {
		return nil, err
	}
	if err := appendTools(shopadminv1.NewRecommendRequestServiceAgentTools(s.RecommendRequest)); err != nil {
		return nil, err
	}
	return tools, nil
}

// Tasks 返回商城管理端需要调度运行时执行的具名任务。
func (s TaskSet) Tasks() []job.Task {
	return []job.Task{
		{Name: "TradeBill", Exec: s.tradeBill},
		{Name: "OrderStatDay", Exec: s.orderStatDay},
		{Name: "GoodsStatDay", Exec: s.goodsStatDay},
		{Name: "RecommendSync", Exec: s.recommendSync},
	}
}
