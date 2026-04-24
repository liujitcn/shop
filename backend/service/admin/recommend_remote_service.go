package admin

import (
	"context"

	adminApi "shop/api/gen/go/admin"
	"shop/pkg/errorsx"
	"shop/service/admin/biz"

	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

const _ = grpc.SupportPackageIsVersion7

// RecommendRemoteService Admin远程推荐服务。
type RecommendRemoteService struct {
	adminApi.UnimplementedRecommendRemoteServiceServer
	recommendRemoteCase *biz.RecommendRemoteCase
}

// NewRecommendRemoteService 创建Admin远程推荐服务。
func NewRecommendRemoteService(recommendRemoteCase *biz.RecommendRemoteCase) *RecommendRemoteService {
	return &RecommendRemoteService{
		recommendRemoteCase: recommendRemoteCase,
	}
}

// GetRecommendRemoteOverview 查询远程推荐概览。
func (s *RecommendRemoteService) GetRecommendRemoteOverview(ctx context.Context, _ *emptypb.Empty) (*adminApi.RecommendRemoteJsonResponse, error) {
	res, err := s.recommendRemoteCase.GetRecommendRemoteOverview(ctx)
	if err != nil {
		log.Errorf("GetRecommendRemoteOverview %v", err)
		return nil, errorsx.WrapInternal(err, "查询远程推荐概览失败")
	}
	return res, nil
}

// GetRecommendRemoteTasks 查询远程推荐任务状态。
func (s *RecommendRemoteService) GetRecommendRemoteTasks(ctx context.Context, _ *emptypb.Empty) (*adminApi.RecommendRemoteJsonResponse, error) {
	res, err := s.recommendRemoteCase.GetRecommendRemoteTasks(ctx)
	if err != nil {
		log.Errorf("GetRecommendRemoteTasks %v", err)
		return nil, errorsx.WrapInternal(err, "查询远程推荐任务状态失败")
	}
	return res, nil
}

// GetRecommendRemoteCategories 查询远程推荐分类。
func (s *RecommendRemoteService) GetRecommendRemoteCategories(ctx context.Context, _ *emptypb.Empty) (*adminApi.RecommendRemoteJsonResponse, error) {
	res, err := s.recommendRemoteCase.GetRecommendRemoteCategories(ctx)
	if err != nil {
		log.Errorf("GetRecommendRemoteCategories %v", err)
		return nil, errorsx.WrapInternal(err, "查询远程推荐分类失败")
	}
	return res, nil
}

// GetRecommendRemoteTimeseries 查询远程推荐时间序列。
func (s *RecommendRemoteService) GetRecommendRemoteTimeseries(ctx context.Context, req *adminApi.RecommendRemoteNameRequest) (*adminApi.RecommendRemoteJsonResponse, error) {
	res, err := s.recommendRemoteCase.GetRecommendRemoteTimeseries(ctx, req)
	if err != nil {
		log.Errorf("GetRecommendRemoteTimeseries %v", err)
		return nil, errorsx.WrapInternal(err, "查询远程推荐时间序列失败")
	}
	return res, nil
}

// GetRecommendRemoteDashboardItems 查询远程推荐仪表盘推荐商品。
func (s *RecommendRemoteService) GetRecommendRemoteDashboardItems(ctx context.Context, req *adminApi.RecommendRemoteDashboardItemsRequest) (*adminApi.RecommendRemoteJsonResponse, error) {
	res, err := s.recommendRemoteCase.GetRecommendRemoteDashboardItems(ctx, req)
	if err != nil {
		log.Errorf("GetRecommendRemoteDashboardItems %v", err)
		return nil, errorsx.WrapInternal(err, "查询远程推荐仪表盘推荐商品失败")
	}
	return res, nil
}

// GetRecommendRemoteRecommendations 查询远程推荐结果。
func (s *RecommendRemoteService) GetRecommendRemoteRecommendations(ctx context.Context, req *adminApi.RecommendRemoteRecommendRequest) (*adminApi.RecommendRemoteJsonResponse, error) {
	res, err := s.recommendRemoteCase.GetRecommendRemoteRecommendations(ctx, req)
	if err != nil {
		log.Errorf("GetRecommendRemoteRecommendations %v", err)
		return nil, errorsx.WrapInternal(err, "查询远程推荐结果失败")
	}
	return res, nil
}

// GetRecommendRemoteNeighbors 查询远程相似内容。
func (s *RecommendRemoteService) GetRecommendRemoteNeighbors(ctx context.Context, req *adminApi.RecommendRemoteNeighborRequest) (*adminApi.RecommendRemoteJsonResponse, error) {
	res, err := s.recommendRemoteCase.GetRecommendRemoteNeighbors(ctx, req)
	if err != nil {
		log.Errorf("GetRecommendRemoteNeighbors %v", err)
		return nil, errorsx.WrapInternal(err, "查询远程相似内容失败")
	}
	return res, nil
}

// PageRecommendRemoteFeedback 查询远程推荐反馈列表。
func (s *RecommendRemoteService) PageRecommendRemoteFeedback(ctx context.Context, req *adminApi.RecommendRemoteFeedbackRequest) (*adminApi.RecommendRemoteJsonResponse, error) {
	res, err := s.recommendRemoteCase.PageRecommendRemoteFeedback(ctx, req)
	if err != nil {
		log.Errorf("PageRecommendRemoteFeedback %v", err)
		return nil, errorsx.WrapInternal(err, "查询远程推荐反馈列表失败")
	}
	return res, nil
}

// ImportRecommendRemoteFeedback 写入远程推荐反馈。
func (s *RecommendRemoteService) ImportRecommendRemoteFeedback(ctx context.Context, req *adminApi.RecommendRemoteJsonRequest) (*emptypb.Empty, error) {
	err := s.recommendRemoteCase.ImportRecommendRemoteFeedback(ctx, req)
	if err != nil {
		log.Errorf("ImportRecommendRemoteFeedback %v", err)
		return nil, errorsx.WrapInternal(err, "写入远程推荐反馈失败")
	}
	return new(emptypb.Empty), nil
}

// DeleteRecommendRemoteFeedback 删除远程推荐反馈。
func (s *RecommendRemoteService) DeleteRecommendRemoteFeedback(ctx context.Context, req *adminApi.RecommendRemoteFeedbackDeleteRequest) (*emptypb.Empty, error) {
	err := s.recommendRemoteCase.DeleteRecommendRemoteFeedback(ctx, req)
	if err != nil {
		log.Errorf("DeleteRecommendRemoteFeedback %v", err)
		return nil, errorsx.WrapInternal(err, "删除远程推荐反馈失败")
	}
	return new(emptypb.Empty), nil
}

// PageRecommendRemoteUsers 查询远程推荐用户列表。
func (s *RecommendRemoteService) PageRecommendRemoteUsers(ctx context.Context, req *adminApi.RecommendRemoteCursorRequest) (*adminApi.RecommendRemoteJsonResponse, error) {
	res, err := s.recommendRemoteCase.PageRecommendRemoteUsers(ctx, req)
	if err != nil {
		log.Errorf("PageRecommendRemoteUsers %v", err)
		return nil, errorsx.WrapInternal(err, "查询远程推荐用户列表失败")
	}
	return res, nil
}

// GetRecommendRemoteUser 查询远程推荐用户。
func (s *RecommendRemoteService) GetRecommendRemoteUser(ctx context.Context, req *adminApi.RecommendRemoteIdRequest) (*adminApi.RecommendRemoteJsonResponse, error) {
	res, err := s.recommendRemoteCase.GetRecommendRemoteUser(ctx, req)
	if err != nil {
		log.Errorf("GetRecommendRemoteUser %v", err)
		return nil, errorsx.WrapInternal(err, "查询远程推荐用户失败")
	}
	return res, nil
}

// DeleteRecommendRemoteUser 删除远程推荐用户。
func (s *RecommendRemoteService) DeleteRecommendRemoteUser(ctx context.Context, req *adminApi.RecommendRemoteIdRequest) (*emptypb.Empty, error) {
	err := s.recommendRemoteCase.DeleteRecommendRemoteUser(ctx, req)
	if err != nil {
		log.Errorf("DeleteRecommendRemoteUser %v", err)
		return nil, errorsx.WrapInternal(err, "删除远程推荐用户失败")
	}
	return new(emptypb.Empty), nil
}

// PageRecommendRemoteItems 查询远程推荐商品列表。
func (s *RecommendRemoteService) PageRecommendRemoteItems(ctx context.Context, req *adminApi.RecommendRemoteCursorRequest) (*adminApi.RecommendRemoteJsonResponse, error) {
	res, err := s.recommendRemoteCase.PageRecommendRemoteItems(ctx, req)
	if err != nil {
		log.Errorf("PageRecommendRemoteItems %v", err)
		return nil, errorsx.WrapInternal(err, "查询远程推荐商品列表失败")
	}
	return res, nil
}

// GetRecommendRemoteItem 查询远程推荐商品。
func (s *RecommendRemoteService) GetRecommendRemoteItem(ctx context.Context, req *adminApi.RecommendRemoteIdRequest) (*adminApi.RecommendRemoteJsonResponse, error) {
	res, err := s.recommendRemoteCase.GetRecommendRemoteItem(ctx, req)
	if err != nil {
		log.Errorf("GetRecommendRemoteItem %v", err)
		return nil, errorsx.WrapInternal(err, "查询远程推荐商品失败")
	}
	return res, nil
}

// DeleteRecommendRemoteItem 删除远程推荐商品。
func (s *RecommendRemoteService) DeleteRecommendRemoteItem(ctx context.Context, req *adminApi.RecommendRemoteIdRequest) (*emptypb.Empty, error) {
	err := s.recommendRemoteCase.DeleteRecommendRemoteItem(ctx, req)
	if err != nil {
		log.Errorf("DeleteRecommendRemoteItem %v", err)
		return nil, errorsx.WrapInternal(err, "删除远程推荐商品失败")
	}
	return new(emptypb.Empty), nil
}

// ExportRecommendRemoteData 导出远程推荐数据。
func (s *RecommendRemoteService) ExportRecommendRemoteData(ctx context.Context, req *adminApi.RecommendRemoteDataRequest) (*adminApi.RecommendRemoteJsonResponse, error) {
	res, err := s.recommendRemoteCase.ExportRecommendRemoteData(ctx, req)
	if err != nil {
		log.Errorf("ExportRecommendRemoteData %v", err)
		return nil, errorsx.WrapInternal(err, "导出远程推荐数据失败")
	}
	return res, nil
}

// ImportRecommendRemoteData 导入远程推荐数据。
func (s *RecommendRemoteService) ImportRecommendRemoteData(ctx context.Context, req *adminApi.RecommendRemoteImportRequest) (*emptypb.Empty, error) {
	err := s.recommendRemoteCase.ImportRecommendRemoteData(ctx, req)
	if err != nil {
		log.Errorf("ImportRecommendRemoteData %v", err)
		return nil, errorsx.WrapInternal(err, "导入远程推荐数据失败")
	}
	return new(emptypb.Empty), nil
}

// GetRecommendRemoteFlowConfig 查询推荐编排配置。
func (s *RecommendRemoteService) GetRecommendRemoteFlowConfig(ctx context.Context, _ *emptypb.Empty) (*adminApi.RecommendRemoteJsonResponse, error) {
	res, err := s.recommendRemoteCase.GetRecommendRemoteFlowConfig(ctx)
	if err != nil {
		log.Errorf("GetRecommendRemoteFlowConfig %v", err)
		return nil, errorsx.WrapInternal(err, "查询推荐编排配置失败")
	}
	return res, nil
}

// SaveRecommendRemoteFlowConfig 保存推荐编排配置。
func (s *RecommendRemoteService) SaveRecommendRemoteFlowConfig(ctx context.Context, req *adminApi.RecommendRemoteJsonRequest) (*emptypb.Empty, error) {
	err := s.recommendRemoteCase.SaveRecommendRemoteFlowConfig(ctx, req)
	if err != nil {
		log.Errorf("SaveRecommendRemoteFlowConfig %v", err)
		return nil, errorsx.WrapInternal(err, "保存推荐编排配置失败")
	}
	return new(emptypb.Empty), nil
}

// ResetRecommendRemoteFlowConfig 重置推荐编排配置。
func (s *RecommendRemoteService) ResetRecommendRemoteFlowConfig(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	err := s.recommendRemoteCase.ResetRecommendRemoteFlowConfig(ctx)
	if err != nil {
		log.Errorf("ResetRecommendRemoteFlowConfig %v", err)
		return nil, errorsx.WrapInternal(err, "重置推荐编排配置失败")
	}
	return new(emptypb.Empty), nil
}

// GetRecommendRemoteFlowSchema 查询推荐编排配置结构。
func (s *RecommendRemoteService) GetRecommendRemoteFlowSchema(ctx context.Context, _ *emptypb.Empty) (*adminApi.RecommendRemoteJsonResponse, error) {
	res, err := s.recommendRemoteCase.GetRecommendRemoteFlowSchema(ctx)
	if err != nil {
		log.Errorf("GetRecommendRemoteFlowSchema %v", err)
		return nil, errorsx.WrapInternal(err, "查询推荐编排配置结构失败")
	}
	return res, nil
}

// GetRecommendRemoteConfig 查询远程推荐配置。
func (s *RecommendRemoteService) GetRecommendRemoteConfig(ctx context.Context, _ *emptypb.Empty) (*adminApi.RecommendRemoteJsonResponse, error) {
	res, err := s.recommendRemoteCase.GetRecommendRemoteConfig(ctx)
	if err != nil {
		log.Errorf("GetRecommendRemoteConfig %v", err)
		return nil, errorsx.WrapInternal(err, "查询远程推荐配置失败")
	}
	return res, nil
}
