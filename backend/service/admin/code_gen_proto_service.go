package admin

import (
	"context"
	"fmt"

	adminv1 "shop/api/gen/go/admin/v1"
	"shop/pkg/errorsx"
	"shop/service/admin/biz"

	"github.com/go-kratos/kratos/v3/log"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

const _ = grpc.SupportPackageIsVersion7

// CodeGenProtoService Admin代码生成Proto接口配置服务。
type CodeGenProtoService struct {
	adminv1.UnimplementedCodeGenProtoServiceServer
	codeGenProtoCase *biz.CodeGenProtoCase
}

// NewCodeGenProtoService 创建Admin代码生成Proto接口配置服务。
func NewCodeGenProtoService(codeGenProtoCase *biz.CodeGenProtoCase) *CodeGenProtoService {
	return &CodeGenProtoService{codeGenProtoCase: codeGenProtoCase}
}

// ListCodeGenProtos 查询代码生成Proto接口配置。
func (s *CodeGenProtoService) ListCodeGenProtos(ctx context.Context, req *adminv1.ListCodeGenProtosRequest) (*adminv1.ListCodeGenProtosResponse, error) {
	res, err := s.codeGenProtoCase.ListCodeGenProtos(ctx, req.GetTableId())
	if err != nil {
		log.Error(fmt.Sprintf("ListCodeGenProtos %v", err))
		return nil, errorsx.WrapInternal(err, "查询代码生成Proto接口配置失败")
	}
	return res, nil
}

// SaveCodeGenProtos 保存代码生成Proto接口配置。
func (s *CodeGenProtoService) SaveCodeGenProtos(ctx context.Context, req *adminv1.SaveCodeGenProtosRequest) (*emptypb.Empty, error) {
	err := s.codeGenProtoCase.SaveCodeGenProtos(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("SaveCodeGenProtos %v", err))
		return nil, errorsx.WrapInternal(err, "保存代码生成Proto接口配置失败")
	}
	return new(emptypb.Empty), nil
}
