package admin

import (
	"context"

	adminv1 "shop/api/gen/go/admin/v1"
	"shop/pkg/errorsx"
	"shop/service/admin/biz"

	"github.com/go-kratos/kratos/v3/log"
	"google.golang.org/grpc"
)

const _ = grpc.SupportPackageIsVersion7

// CodeGenService Admin代码生成执行服务。
type CodeGenService struct {
	adminv1.UnimplementedCodeGenServiceServer
	codeGenCase *biz.CodeGenCase
}

// NewCodeGenService 创建Admin代码生成执行服务。
func NewCodeGenService(codeGenCase *biz.CodeGenCase) *CodeGenService {
	return &CodeGenService{codeGenCase: codeGenCase}
}

// PreviewCodeGen 预览代码生成文件。
func (s *CodeGenService) PreviewCodeGen(ctx context.Context, req *adminv1.PreviewCodeGenRequest) (*adminv1.PreviewCodeGenResponse, error) {
	res, err := s.codeGenCase.PreviewCodeGen(ctx, req.GetTableId(), req.GetOutputPaths())
	if err != nil {
		log.Error("PreviewCodeGen", "error", err)
		return nil, errorsx.WrapInternal(err, "预览代码生成文件失败")
	}
	return res, nil
}

// StartCodeGenTask 启动代码生成任务。
func (s *CodeGenService) StartCodeGenTask(ctx context.Context, req *adminv1.StartCodeGenTaskRequest) (*adminv1.StartCodeGenTaskResponse, error) {
	res, err := s.codeGenCase.StartCodeGenTask(ctx, req)
	if err != nil {
		log.Error("StartCodeGenTask", "error", err)
		return nil, errorsx.WrapInternal(err, "启动代码生成任务失败")
	}
	return res, nil
}

// GetCodeGenTask 查询代码生成任务进度。
func (s *CodeGenService) GetCodeGenTask(ctx context.Context, req *adminv1.GetCodeGenTaskRequest) (*adminv1.CodeGenTask, error) {
	res, err := s.codeGenCase.GetCodeGenTask(ctx, req.GetTaskId())
	if err != nil {
		log.Error("GetCodeGenTask", "error", err)
		return nil, errorsx.WrapInternal(err, "查询代码生成任务进度失败")
	}
	return res, nil
}
