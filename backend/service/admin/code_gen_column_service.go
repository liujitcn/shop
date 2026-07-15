package admin

import (
	"context"
	"fmt"

	adminv1 "shop/api/gen/go/admin/v1"
	"shop/pkg/errorsx"
	"shop/service/admin/biz"

	"github.com/go-kratos/kratos/v3/log"
	"google.golang.org/grpc"
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
