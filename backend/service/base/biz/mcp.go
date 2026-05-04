package biz

import (
	"context"

	basev1 "shop/api/gen/go/base/v1"
	commonv1 "shop/api/gen/go/common/v1"
	"shop/pkg/errorsx"

	"google.golang.org/protobuf/types/known/emptypb"
)

// McpCase 处理 MCP 公共业务。
type McpCase struct{}

// NewMcpCase 创建 MCP 业务实例。
func NewMcpCase() *McpCase {
	return &McpCase{}
}

// HandleMcp 校验 MCP Streamable HTTP 请求参数。
func (c *McpCase) HandleMcp(ctx context.Context, req *basev1.HandleMcpRequest) (*emptypb.Empty, error) {
	switch req.GetTerminal() {
	// 仅允许已定义的商城端和管理端 MCP 入口。
	case commonv1.McpTerminal_MCP_TERMINAL_APP, commonv1.McpTerminal_MCP_TERMINAL_ADMIN:
		return &emptypb.Empty{}, nil
	default:
		return nil, errorsx.InvalidArgument("终端类型不支持")
	}
}
