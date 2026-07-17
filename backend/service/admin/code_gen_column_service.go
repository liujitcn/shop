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

// CodeGenColumnService Admin代码生成字段服务。
type CodeGenColumnService struct {
	adminv1.UnimplementedCodeGenColumnServiceServer
	codeGenColumnCase *biz.CodeGenColumnCase
}

// NewCodeGenColumnService 创建Admin代码生成字段服务。
func NewCodeGenColumnService(codeGenColumnCase *biz.CodeGenColumnCase) *CodeGenColumnService {
	return &CodeGenColumnService{codeGenColumnCase: codeGenColumnCase}
}

// ListCodeGenDatabaseColumns 查询数据库表字段列表。
func (s *CodeGenColumnService) ListCodeGenDatabaseColumns(ctx context.Context, req *adminv1.ListCodeGenDatabaseColumnsRequest) (*adminv1.ListCodeGenDatabaseColumnsResponse, error) {
	res, err := s.codeGenColumnCase.ListCodeGenDatabaseColumns(ctx, req.GetTableName())
	if err != nil {
		log.Error(fmt.Sprintf("ListCodeGenDatabaseColumns %v", err))
		return nil, errorsx.WrapInternal(err, "查询数据库表字段列表失败")
	}
	return res, nil
}

// ListCodeGenColumns 查询代码生成字段配置。
func (s *CodeGenColumnService) ListCodeGenColumns(ctx context.Context, req *adminv1.ListCodeGenColumnsRequest) (*adminv1.ListCodeGenColumnsResponse, error) {
	res, err := s.codeGenColumnCase.ListCodeGenColumns(ctx, req.GetTableId())
	if err != nil {
		log.Error(fmt.Sprintf("ListCodeGenColumns %v", err))
		return nil, errorsx.WrapInternal(err, "查询代码生成字段配置失败")
	}
	return res, nil
}

// SaveCodeGenColumns 保存代码生成字段配置。
func (s *CodeGenColumnService) SaveCodeGenColumns(ctx context.Context, req *adminv1.SaveCodeGenColumnsRequest) (*emptypb.Empty, error) {
	err := s.codeGenColumnCase.SaveCodeGenColumns(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("SaveCodeGenColumns %v", err))
		return nil, errorsx.WrapInternal(err, "保存代码生成字段配置失败")
	}
	return new(emptypb.Empty), nil
}
