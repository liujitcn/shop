package server

import (
	"github.com/go-kratos/kratos/v3/transport/grpc"
	kratosHTTP "github.com/go-kratos/kratos/v3/transport/http"
	mcpserver "github.com/liujitcn/kratos-kit/transport/mcp"
)

// Module 表示可挂载到服务端宿主的业务模块。
type Module interface {
	RegisterGRPC(*grpc.Server)
	RegisterHTTP(*kratosHTTP.Server)
	RegisterMCP(*mcpserver.Server)
}

// Modules 表示当前进程启用的业务模块集合。
type Modules []Module

// RegisterGRPC 将全部业务模块注册到 GRPC 服务。
func (modules Modules) RegisterGRPC(srv *grpc.Server) {
	for _, module := range modules {
		module.RegisterGRPC(srv)
	}
}

// RegisterHTTP 将全部业务模块注册到 HTTP 服务。
func (modules Modules) RegisterHTTP(srv *kratosHTTP.Server) {
	for _, module := range modules {
		module.RegisterHTTP(srv)
	}
}

// RegisterMCP 将全部业务模块注册到 MCP 服务。
func (modules Modules) RegisterMCP(srv *mcpserver.Server) {
	for _, module := range modules {
		module.RegisterMCP(srv)
	}
}
