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

// CodeGenTableService Admin代码生成表配置服务。
type CodeGenTableService struct {
	adminv1.UnimplementedCodeGenTableServiceServer
	codeGenTableCase *biz.CodeGenTableCase
}

// NewCodeGenTableService 创建Admin代码生成表配置服务。
func NewCodeGenTableService(codeGenTableCase *biz.CodeGenTableCase) *CodeGenTableService {
	return &CodeGenTableService{codeGenTableCase: codeGenTableCase}
}

// ListCodeGenDatabaseTables 查询数据库表列表。
func (s *CodeGenTableService) ListCodeGenDatabaseTables(ctx context.Context, _ *adminv1.ListCodeGenDatabaseTablesRequest) (*adminv1.ListCodeGenDatabaseTablesResponse, error) {
	res, err := s.codeGenTableCase.ListCodeGenDatabaseTables(ctx)
	if err != nil {
		log.Error(fmt.Sprintf("ListCodeGenDatabaseTables %v", err))
		return nil, errorsx.WrapInternal(err, "查询数据库表列表失败")
	}
	return res, nil
}

// PageCodeGenTables 查询代码生成表配置列表。
func (s *CodeGenTableService) PageCodeGenTables(ctx context.Context, req *adminv1.PageCodeGenTablesRequest) (*adminv1.PageCodeGenTablesResponse, error) {
	page, err := s.codeGenTableCase.PageCodeGenTables(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageCodeGenTables %v", err))
		return nil, errorsx.WrapInternal(err, "查询代码生成表配置列表失败")
	}
	return page, nil
}

// GetCodeGenTable 查询代码生成表配置。
func (s *CodeGenTableService) GetCodeGenTable(ctx context.Context, req *adminv1.GetCodeGenTableRequest) (*adminv1.CodeGenTableForm, error) {
	item, err := s.codeGenTableCase.GetCodeGenTable(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("GetCodeGenTable %v", err))
		return nil, errorsx.WrapInternal(err, "查询代码生成表配置失败")
	}
	return item, nil
}

// CreateCodeGenTable 创建代码生成表配置。
func (s *CodeGenTableService) CreateCodeGenTable(ctx context.Context, req *adminv1.CreateCodeGenTableRequest) (*emptypb.Empty, error) {
	err := s.codeGenTableCase.CreateCodeGenTable(ctx, req.GetCodeGenTable())
	if err != nil {
		log.Error(fmt.Sprintf("CreateCodeGenTable %v", err))
		return nil, errorsx.WrapInternal(err, "创建代码生成表配置失败")
	}
	return new(emptypb.Empty), nil
}

// UpdateCodeGenTable 更新代码生成表配置。
func (s *CodeGenTableService) UpdateCodeGenTable(ctx context.Context, req *adminv1.UpdateCodeGenTableRequest) (*emptypb.Empty, error) {
	err := s.codeGenTableCase.UpdateCodeGenTable(ctx, req.GetId(), req.GetCodeGenTable())
	if err != nil {
		log.Error(fmt.Sprintf("UpdateCodeGenTable %v", err))
		return nil, errorsx.WrapInternal(err, "更新代码生成表配置失败")
	}
	return new(emptypb.Empty), nil
}

// DeleteCodeGenTable 删除代码生成表配置。
func (s *CodeGenTableService) DeleteCodeGenTable(ctx context.Context, req *adminv1.DeleteCodeGenTableRequest) (*emptypb.Empty, error) {
	err := s.codeGenTableCase.DeleteCodeGenTable(ctx, req.GetIds())
	if err != nil {
		log.Error(fmt.Sprintf("DeleteCodeGenTable %v", err))
		return nil, errorsx.WrapInternal(err, "删除代码生成表配置失败")
	}
	return new(emptypb.Empty), nil
}
