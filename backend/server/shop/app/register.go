// Package app 注册 shop.app.v1 传输层服务。
package app

import (
	shopappv1 "shop/api/gen/go/shop/app/v1"
	einoTool "shop/pkg/agent/eino/tool"
	host "shop/server"
	shopapp "shop/service/shop/app"

	"github.com/go-kratos/kratos/v3/transport/grpc"
	kratosHTTP "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/google/wire"
	mcpserver "github.com/liujitcn/kratos-kit/transport/mcp"
)

// Services 汇总 shop.app.v1 的服务实现。
type Services struct {
	CommentInfo   *shopapp.CommentInfoService
	GoodsCategory *shopapp.GoodsCategoryService
	Goods         *shopapp.GoodsInfoService
	TenantStore   *shopapp.TenantStoreService
	Order         *shopapp.OrderInfoService
	Pay           *shopapp.PayService
	Recommend     *shopapp.RecommendService
	ShopBanner    *shopapp.ShopBannerService
	ShopHot       *shopapp.ShopHotService
	ShopService   *shopapp.ShopServiceService
	UserAddress   *shopapp.UserAddressService
	UserCart      *shopapp.UserCartService
	UserCollect   *shopapp.UserCollectService
	UserStore     *shopapp.UserStoreService
}

var _ host.Module = Services{}

// ProviderSet 汇总 shop.app.v1 传输模块依赖注入提供者。
var ProviderSet = wire.NewSet(wire.Struct(new(Services), "*"))

// RegisterGRPC 注册 shop.app.v1 的 gRPC 服务。
func (s Services) RegisterGRPC(srv *grpc.Server) {
	shopappv1.RegisterCommentInfoServiceServer(srv, s.CommentInfo)
	shopappv1.RegisterGoodsCategoryServiceServer(srv, s.GoodsCategory)
	shopappv1.RegisterGoodsInfoServiceServer(srv, s.Goods)
	shopappv1.RegisterTenantStoreServiceServer(srv, s.TenantStore)
	shopappv1.RegisterOrderInfoServiceServer(srv, s.Order)
	shopappv1.RegisterPayServiceServer(srv, s.Pay)
	shopappv1.RegisterRecommendServiceServer(srv, s.Recommend)
	shopappv1.RegisterShopBannerServiceServer(srv, s.ShopBanner)
	shopappv1.RegisterShopHotServiceServer(srv, s.ShopHot)
	shopappv1.RegisterShopServiceServiceServer(srv, s.ShopService)
	shopappv1.RegisterUserAddressServiceServer(srv, s.UserAddress)
	shopappv1.RegisterUserCartServiceServer(srv, s.UserCart)
	shopappv1.RegisterUserCollectServiceServer(srv, s.UserCollect)
	shopappv1.RegisterUserStoreServiceServer(srv, s.UserStore)
}

// RegisterHTTP 注册 shop.app.v1 的 HTTP 服务。
func (s Services) RegisterHTTP(srv *kratosHTTP.Server) {
	shopappv1.RegisterCommentInfoServiceHTTPServer(srv, s.CommentInfo)
	shopappv1.RegisterGoodsCategoryServiceHTTPServer(srv, s.GoodsCategory)
	shopappv1.RegisterGoodsInfoServiceHTTPServer(srv, s.Goods)
	shopappv1.RegisterTenantStoreServiceHTTPServer(srv, s.TenantStore)
	shopappv1.RegisterOrderInfoServiceHTTPServer(srv, s.Order)
	shopappv1.RegisterPayServiceHTTPServer(srv, s.Pay)
	shopappv1.RegisterRecommendServiceHTTPServer(srv, s.Recommend)
	shopappv1.RegisterShopBannerServiceHTTPServer(srv, s.ShopBanner)
	shopappv1.RegisterShopHotServiceHTTPServer(srv, s.ShopHot)
	shopappv1.RegisterShopServiceServiceHTTPServer(srv, s.ShopService)
	shopappv1.RegisterUserAddressServiceHTTPServer(srv, s.UserAddress)
	shopappv1.RegisterUserCartServiceHTTPServer(srv, s.UserCart)
	shopappv1.RegisterUserCollectServiceHTTPServer(srv, s.UserCollect)
	shopappv1.RegisterUserStoreServiceHTTPServer(srv, s.UserStore)
}

// RegisterMCP 注册 shop.app.v1 的 MCP 工具。
func (s Services) RegisterMCP(server *mcpserver.Server) {
	mcpSrv := server.MCPServer()
	shopappv1.RegisterCommentInfoServiceMCPTools(mcpSrv, s.CommentInfo)
	shopappv1.RegisterGoodsCategoryServiceMCPTools(mcpSrv, s.GoodsCategory)
	shopappv1.RegisterGoodsInfoServiceMCPTools(mcpSrv, s.Goods)
	shopappv1.RegisterTenantStoreServiceMCPTools(mcpSrv, s.TenantStore)
	shopappv1.RegisterOrderInfoServiceMCPTools(mcpSrv, s.Order)
	shopappv1.RegisterPayServiceMCPTools(mcpSrv, s.Pay)
	shopappv1.RegisterRecommendServiceMCPTools(mcpSrv, s.Recommend)
	shopappv1.RegisterShopBannerServiceMCPTools(mcpSrv, s.ShopBanner)
	shopappv1.RegisterShopHotServiceMCPTools(mcpSrv, s.ShopHot)
	shopappv1.RegisterShopServiceServiceMCPTools(mcpSrv, s.ShopService)
	shopappv1.RegisterUserAddressServiceMCPTools(mcpSrv, s.UserAddress)
	shopappv1.RegisterUserCartServiceMCPTools(mcpSrv, s.UserCart)
	shopappv1.RegisterUserCollectServiceMCPTools(mcpSrv, s.UserCollect)
	shopappv1.RegisterUserStoreServiceMCPTools(mcpSrv, s.UserStore)
}

// AppAgentTools 创建 shop.app.v1 的商城端 AI 助手工具。
func (s Services) AppAgentTools() ([]einoTool.Invokable, error) {
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
	if err := appendTools(shopappv1.NewCommentInfoServiceAgentTools(s.CommentInfo)); err != nil {
		return nil, err
	}
	if err := appendTools(shopappv1.NewGoodsCategoryServiceAgentTools(s.GoodsCategory)); err != nil {
		return nil, err
	}
	if err := appendTools(shopappv1.NewGoodsInfoServiceAgentTools(s.Goods)); err != nil {
		return nil, err
	}
	if err := appendTools(shopappv1.NewTenantStoreServiceAgentTools(s.TenantStore)); err != nil {
		return nil, err
	}
	if err := appendTools(shopappv1.NewOrderInfoServiceAgentTools(s.Order)); err != nil {
		return nil, err
	}
	if err := appendTool(shopappv1.NewPayServiceJsapiPayAgentTool(s.Pay)); err != nil {
		return nil, err
	}
	if err := appendTool(shopappv1.NewPayServiceH5PayAgentTool(s.Pay)); err != nil {
		return nil, err
	}
	if err := appendTool(shopappv1.NewRecommendServiceRecommendAnonymousActorAgentTool(s.Recommend)); err != nil {
		return nil, err
	}
	if err := appendTool(shopappv1.NewRecommendServiceRecommendGoodsAgentTool(s.Recommend)); err != nil {
		return nil, err
	}
	if err := appendTool(shopappv1.NewRecommendServiceRecommendEventReportAgentTool(s.Recommend)); err != nil {
		return nil, err
	}
	if err := appendTools(shopappv1.NewShopBannerServiceAgentTools(s.ShopBanner)); err != nil {
		return nil, err
	}
	if err := appendTools(shopappv1.NewShopHotServiceAgentTools(s.ShopHot)); err != nil {
		return nil, err
	}
	if err := appendTools(shopappv1.NewShopServiceServiceAgentTools(s.ShopService)); err != nil {
		return nil, err
	}
	if err := appendTools(shopappv1.NewUserAddressServiceAgentTools(s.UserAddress)); err != nil {
		return nil, err
	}
	if err := appendTools(shopappv1.NewUserCartServiceAgentTools(s.UserCart)); err != nil {
		return nil, err
	}
	if err := appendTools(shopappv1.NewUserCollectServiceAgentTools(s.UserCollect)); err != nil {
		return nil, err
	}
	if err := appendTools(shopappv1.NewUserStoreServiceAgentTools(s.UserStore)); err != nil {
		return nil, err
	}
	return tools, nil
}
