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
	remoteCase *biz.RemoteCase
}

// NewRecommendRemoteService 创建Admin远程推荐服务。
func NewRecommendRemoteService(remoteCase *biz.RemoteCase) *RecommendRemoteService {
	return &RecommendRemoteService{
		remoteCase: remoteCase,
	}
}

// GetOverview 查询远程推荐概览。
func (s *RecommendRemoteService) GetOverview(ctx context.Context, _ *emptypb.Empty) (*adminApi.OverviewResponse, error) {
	res, err := s.remoteCase.GetOverview(ctx)
	if err != nil {
		log.Errorf("GetOverview %v", err)
		return nil, errorsx.WrapInternal(err, "查询远程推荐概览失败")
	}
	return res, nil
}

// GetTask 查询远程推荐任务状态。
func (s *RecommendRemoteService) GetTask(ctx context.Context, _ *emptypb.Empty) (*adminApi.TasksResponse, error) {
	res, err := s.remoteCase.GetTask(ctx)
	if err != nil {
		log.Errorf("GetTask %v", err)
		return nil, errorsx.WrapInternal(err, "查询远程推荐任务状态失败")
	}
	return res, nil
}

// GetCategory 查询远程推荐分类。
func (s *RecommendRemoteService) GetCategory(ctx context.Context, _ *emptypb.Empty) (*adminApi.CategoriesResponse, error) {
	res, err := s.remoteCase.GetCategory(ctx)
	if err != nil {
		log.Errorf("GetCategory %v", err)
		return nil, errorsx.WrapInternal(err, "查询远程推荐分类失败")
	}
	return res, nil
}

// GetTimeseries 查询远程推荐时间序列。
func (s *RecommendRemoteService) GetTimeseries(ctx context.Context, req *adminApi.NameRequest) (*adminApi.TimeseriesResponse, error) {
	res, err := s.remoteCase.GetTimeseries(ctx, req)
	if err != nil {
		log.Errorf("GetTimeseries %v", err)
		return nil, errorsx.WrapInternal(err, "查询远程推荐时间序列失败")
	}
	return res, nil
}

// GetDashboardItems 查询远程推荐仪表盘推荐商品。
func (s *RecommendRemoteService) GetDashboardItems(ctx context.Context, req *adminApi.DashboardItemsRequest) (*adminApi.RecordsResponse, error) {
	res, err := s.remoteCase.GetDashboardItems(ctx, req)
	if err != nil {
		log.Errorf("GetDashboardItems %v", err)
		return nil, errorsx.WrapInternal(err, "查询远程推荐仪表盘推荐商品失败")
	}
	return res, nil
}

// GetRecommendation 查询远程推荐结果。
func (s *RecommendRemoteService) GetRecommendation(ctx context.Context, req *adminApi.RecommendationRequest) (*adminApi.RecordsResponse, error) {
	res, err := s.remoteCase.GetRecommendation(ctx, req)
	if err != nil {
		log.Errorf("GetRecommendation %v", err)
		return nil, errorsx.WrapInternal(err, "查询远程推荐结果失败")
	}
	return res, nil
}

// GetNeighbor 查询远程相似内容。
func (s *RecommendRemoteService) GetNeighbor(ctx context.Context, req *adminApi.NeighborRequest) (*adminApi.RecordsResponse, error) {
	res, err := s.remoteCase.GetNeighbor(ctx, req)
	if err != nil {
		log.Errorf("GetNeighbor %v", err)
		return nil, errorsx.WrapInternal(err, "查询远程相似内容失败")
	}
	return res, nil
}

// PageFeedback 查询远程推荐反馈列表。
func (s *RecommendRemoteService) PageFeedback(ctx context.Context, req *adminApi.FeedbackRequest) (*adminApi.FeedbackPageResponse, error) {
	res, err := s.remoteCase.PageFeedback(ctx, req)
	if err != nil {
		log.Errorf("PageFeedback %v", err)
		return nil, errorsx.WrapInternal(err, "查询远程推荐反馈列表失败")
	}
	return res, nil
}

// ImportFeedback 写入远程推荐反馈。
func (s *RecommendRemoteService) ImportFeedback(ctx context.Context, req *adminApi.JsonRequest) (*emptypb.Empty, error) {
	err := s.remoteCase.ImportFeedback(ctx, req)
	if err != nil {
		log.Errorf("ImportFeedback %v", err)
		return nil, errorsx.WrapInternal(err, "写入远程推荐反馈失败")
	}
	return new(emptypb.Empty), nil
}

// DeleteFeedback 删除远程推荐反馈。
func (s *RecommendRemoteService) DeleteFeedback(ctx context.Context, req *adminApi.FeedbackDeleteRequest) (*emptypb.Empty, error) {
	err := s.remoteCase.DeleteFeedback(ctx, req)
	if err != nil {
		log.Errorf("DeleteFeedback %v", err)
		return nil, errorsx.WrapInternal(err, "删除远程推荐反馈失败")
	}
	return new(emptypb.Empty), nil
}

// PageUser 查询远程推荐用户列表。
func (s *RecommendRemoteService) PageUser(ctx context.Context, req *adminApi.CursorRequest) (*adminApi.UsersPageResponse, error) {
	res, err := s.remoteCase.PageUser(ctx, req)
	if err != nil {
		log.Errorf("PageUser %v", err)
		return nil, errorsx.WrapInternal(err, "查询远程推荐用户列表失败")
	}
	return res, nil
}

// GetUser 查询远程推荐用户。
func (s *RecommendRemoteService) GetUser(ctx context.Context, req *adminApi.IdRequest) (*adminApi.User, error) {
	res, err := s.remoteCase.GetUser(ctx, req)
	if err != nil {
		log.Errorf("GetUser %v", err)
		return nil, errorsx.WrapInternal(err, "查询远程推荐用户失败")
	}
	return res, nil
}

// DeleteUser 删除远程推荐用户。
func (s *RecommendRemoteService) DeleteUser(ctx context.Context, req *adminApi.IdRequest) (*emptypb.Empty, error) {
	err := s.remoteCase.DeleteUser(ctx, req)
	if err != nil {
		log.Errorf("DeleteUser %v", err)
		return nil, errorsx.WrapInternal(err, "删除远程推荐用户失败")
	}
	return new(emptypb.Empty), nil
}

// PageItem 查询远程推荐商品列表。
func (s *RecommendRemoteService) PageItem(ctx context.Context, req *adminApi.CursorRequest) (*adminApi.ItemsPageResponse, error) {
	res, err := s.remoteCase.PageItem(ctx, req)
	if err != nil {
		log.Errorf("PageItem %v", err)
		return nil, errorsx.WrapInternal(err, "查询远程推荐商品列表失败")
	}
	return res, nil
}

// GetItem 查询远程推荐商品。
func (s *RecommendRemoteService) GetItem(ctx context.Context, req *adminApi.IdRequest) (*adminApi.Item, error) {
	res, err := s.remoteCase.GetItem(ctx, req)
	if err != nil {
		log.Errorf("GetItem %v", err)
		return nil, errorsx.WrapInternal(err, "查询远程推荐商品失败")
	}
	return res, nil
}

// DeleteItem 删除远程推荐商品。
func (s *RecommendRemoteService) DeleteItem(ctx context.Context, req *adminApi.IdRequest) (*emptypb.Empty, error) {
	err := s.remoteCase.DeleteItem(ctx, req)
	if err != nil {
		log.Errorf("DeleteItem %v", err)
		return nil, errorsx.WrapInternal(err, "删除远程推荐商品失败")
	}
	return new(emptypb.Empty), nil
}

// ExportData 导出远程推荐数据。
func (s *RecommendRemoteService) ExportData(ctx context.Context, req *adminApi.DataRequest) (*adminApi.DataPageResponse, error) {
	res, err := s.remoteCase.ExportData(ctx, req)
	if err != nil {
		log.Errorf("ExportData %v", err)
		return nil, errorsx.WrapInternal(err, "导出远程推荐数据失败")
	}
	return res, nil
}

// ImportData 导入远程推荐数据。
func (s *RecommendRemoteService) ImportData(ctx context.Context, req *adminApi.ImportRequest) (*emptypb.Empty, error) {
	err := s.remoteCase.ImportData(ctx, req)
	if err != nil {
		log.Errorf("ImportData %v", err)
		return nil, errorsx.WrapInternal(err, "导入远程推荐数据失败")
	}
	return new(emptypb.Empty), nil
}

// PurgeData 清空远程推荐数据。
func (s *RecommendRemoteService) PurgeData(ctx context.Context, req *adminApi.PurgeRequest) (*emptypb.Empty, error) {
	err := s.remoteCase.PurgeData(ctx, req)
	if err != nil {
		log.Errorf("PurgeData %v", err)
		return nil, errorsx.WrapInternal(err, "清空远程推荐数据失败")
	}
	return new(emptypb.Empty), nil
}

// GetFlowConfig 查询推荐编排配置。
func (s *RecommendRemoteService) GetFlowConfig(ctx context.Context, _ *emptypb.Empty) (*adminApi.ConfigResponse, error) {
	res, err := s.remoteCase.GetFlowConfig(ctx)
	if err != nil {
		log.Errorf("GetFlowConfig %v", err)
		return nil, errorsx.WrapInternal(err, "查询推荐编排配置失败")
	}
	return res, nil
}

// SaveFlowConfig 保存推荐编排配置。
func (s *RecommendRemoteService) SaveFlowConfig(ctx context.Context, req *adminApi.JsonRequest) (*emptypb.Empty, error) {
	err := s.remoteCase.SaveFlowConfig(ctx, req)
	if err != nil {
		log.Errorf("SaveFlowConfig %v", err)
		return nil, errorsx.WrapInternal(err, "保存推荐编排配置失败")
	}
	return new(emptypb.Empty), nil
}

// ResetFlowConfig 重置推荐编排配置。
func (s *RecommendRemoteService) ResetFlowConfig(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	err := s.remoteCase.ResetFlowConfig(ctx)
	if err != nil {
		log.Errorf("ResetFlowConfig %v", err)
		return nil, errorsx.WrapInternal(err, "重置推荐编排配置失败")
	}
	return new(emptypb.Empty), nil
}

// GetFlowSchema 查询推荐编排配置结构。
func (s *RecommendRemoteService) GetFlowSchema(ctx context.Context, _ *emptypb.Empty) (*adminApi.ConfigResponse, error) {
	res, err := s.remoteCase.GetFlowSchema(ctx)
	if err != nil {
		log.Errorf("GetFlowSchema %v", err)
		return nil, errorsx.WrapInternal(err, "查询推荐编排配置结构失败")
	}
	return res, nil
}

// GetConfig 查询远程推荐配置。
func (s *RecommendRemoteService) GetConfig(ctx context.Context, _ *emptypb.Empty) (*adminApi.ConfigResponse, error) {
	res, err := s.remoteCase.GetConfig(ctx)
	if err != nil {
		log.Errorf("GetConfig %v", err)
		return nil, errorsx.WrapInternal(err, "查询远程推荐配置失败")
	}
	return res, nil
}
