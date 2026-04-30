package admin

import (
	"context"

	adminv1 "shop/api/gen/go/admin/v1"
	"shop/pkg/errorsx"
	"shop/service/admin/biz"

	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

const _ = grpc.SupportPackageIsVersion7

// RecommendGorseService Admin Gorse 推荐服务。
type RecommendGorseService struct {
	adminv1.UnimplementedRecommendGorseServiceServer
	recommendGorseCase *biz.RecommendGorseCase
}

// NewRecommendGorseService 创建 Admin Gorse 推荐服务。
func NewRecommendGorseService(recommendGorseCase *biz.RecommendGorseCase) *RecommendGorseService {
	return &RecommendGorseService{
		recommendGorseCase: recommendGorseCase,
	}
}

// GetTimeSeries 查询 Gorse 推荐时间序列。
func (s *RecommendGorseService) GetTimeSeries(ctx context.Context, req *adminv1.GetTimeSeriesRequest) (*adminv1.TimeSeriesResponse, error) {
	res, err := s.recommendGorseCase.GetTimeSeries(ctx, req.GetName(), req.GetBegin(), req.GetEnd())
	if err != nil {
		log.Errorf("GetTimeSeries %v", err)
		return nil, errorsx.WrapInternal(err, "查询 Gorse 推荐时间序列失败")
	}
	return res, nil
}

// OptionCategories 查询 Gorse 推荐分类列表。
func (s *RecommendGorseService) OptionCategories(ctx context.Context, req *adminv1.OptionCategoriesRequest) (*adminv1.OptionCategoriesResponse, error) {
	res, err := s.recommendGorseCase.OptionCategories(ctx)
	if err != nil {
		log.Errorf("OptionCategories %v", err)
		return nil, errorsx.WrapInternal(err, "查询 Gorse 推荐分类列表失败")
	}
	return res, nil
}

// ListDashboardItems 查询 Gorse 推荐仪表盘推荐商品。
func (s *RecommendGorseService) ListDashboardItems(
	ctx context.Context,
	req *adminv1.ListDashboardItemsRequest,
) (*adminv1.ListDashboardItemsResponse, error) {
	res, err := s.recommendGorseCase.ListDashboardItems(ctx, req.GetRecommender(), req.GetCategory(), req.GetEnd())
	if err != nil {
		log.Errorf("ListDashboardItems %v", err)
		return nil, errorsx.WrapInternal(err, "查询 Gorse 推荐仪表盘推荐商品失败")
	}
	return res, nil
}

// ListTasks 查询 Gorse 推荐任务状态。
func (s *RecommendGorseService) ListTasks(ctx context.Context, req *adminv1.ListTasksRequest) (*adminv1.ListTasksResponse, error) {
	res, err := s.recommendGorseCase.ListTasks(ctx)
	if err != nil {
		log.Errorf("ListTasks %v", err)
		return nil, errorsx.WrapInternal(err, "查询 Gorse 推荐任务状态失败")
	}
	return res, nil
}

// PageUsers 查询 Gorse 推荐用户列表。
func (s *RecommendGorseService) PageUsers(
	ctx context.Context,
	req *adminv1.PageUsersRequest,
) (*adminv1.PageUsersResponse, error) {
	res, err := s.recommendGorseCase.PageUsers(ctx, req.GetCursor(), req.GetN())
	if err != nil {
		log.Errorf("PageUsers %v", err)
		return nil, errorsx.WrapInternal(err, "查询 Gorse 推荐用户列表失败")
	}
	return res, nil
}

// GetUser 查询 Gorse 推荐用户。
func (s *RecommendGorseService) GetUser(ctx context.Context, req *adminv1.GetUserRequest) (*adminv1.UserResponse, error) {
	res, err := s.recommendGorseCase.GetUser(ctx, req.GetId())
	if err != nil {
		log.Errorf("GetUser %v", err)
		return nil, errorsx.WrapInternal(err, "查询 Gorse 推荐用户失败")
	}
	return res, nil
}

// DeleteUser 删除 Gorse 推荐用户。
func (s *RecommendGorseService) DeleteUser(ctx context.Context, req *adminv1.DeleteUserRequest) (*emptypb.Empty, error) {
	err := s.recommendGorseCase.DeleteUser(ctx, req.GetId())
	if err != nil {
		log.Errorf("DeleteUser %v", err)
		return nil, errorsx.WrapInternal(err, "删除 Gorse 推荐用户失败")
	}
	return &emptypb.Empty{}, nil
}

// GetUserSimilar 查询 Gorse 推荐相似用户。
func (s *RecommendGorseService) GetUserSimilar(
	ctx context.Context,
	req *adminv1.GetUserSimilarRequest,
) (*adminv1.UserSimilarResponse, error) {
	res, err := s.recommendGorseCase.GetUserSimilar(ctx, req.GetId(), req.GetRecommender())
	if err != nil {
		log.Errorf("GetUserSimilar %v", err)
		return nil, errorsx.WrapInternal(err, "查询 Gorse 推荐相似用户失败")
	}
	return res, nil
}

// GetUserFeedback 查询 Gorse 推荐用户反馈。
func (s *RecommendGorseService) GetUserFeedback(
	ctx context.Context,
	req *adminv1.GetUserFeedbackRequest,
) (*adminv1.FeedbackResponse, error) {
	res, err := s.recommendGorseCase.GetUserFeedback(ctx, req.GetId(), req.GetFeedbackType(), req.GetOffset(), req.GetN())
	if err != nil {
		log.Errorf("GetUserFeedback %v", err)
		return nil, errorsx.WrapInternal(err, "查询 Gorse 推荐用户反馈失败")
	}
	return res, nil
}

// GetUserRecommend 查询 Gorse 推荐用户推荐结果。
func (s *RecommendGorseService) GetUserRecommend(
	ctx context.Context,
	req *adminv1.GetUserRecommendRequest,
) (*adminv1.ItemListResponse, error) {
	res, err := s.recommendGorseCase.GetUserRecommend(ctx, req.GetId(), req.GetRecommender(), req.GetCategory(), req.GetN())
	if err != nil {
		log.Errorf("GetUserRecommend %v", err)
		return nil, errorsx.WrapInternal(err, "查询 Gorse 推荐用户推荐结果失败")
	}
	return res, nil
}

// PageItems 查询 Gorse 推荐商品列表。
func (s *RecommendGorseService) PageItems(
	ctx context.Context,
	req *adminv1.PageItemsRequest,
) (*adminv1.PageItemsResponse, error) {
	res, err := s.recommendGorseCase.PageItems(ctx, req.GetCursor(), req.GetN())
	if err != nil {
		log.Errorf("PageItems %v", err)
		return nil, errorsx.WrapInternal(err, "查询 Gorse 推荐商品列表失败")
	}
	return res, nil
}

// GetItem 查询 Gorse 推荐商品。
func (s *RecommendGorseService) GetItem(ctx context.Context, req *adminv1.GetItemRequest) (*adminv1.Item, error) {
	res, err := s.recommendGorseCase.GetItem(ctx, req.GetId())
	if err != nil {
		log.Errorf("GetItem %v", err)
		return nil, errorsx.WrapInternal(err, "查询 Gorse 推荐商品失败")
	}
	return res, nil
}

// DeleteItem 删除 Gorse 推荐商品。
func (s *RecommendGorseService) DeleteItem(ctx context.Context, req *adminv1.DeleteItemRequest) (*emptypb.Empty, error) {
	err := s.recommendGorseCase.DeleteItem(ctx, req.GetId())
	if err != nil {
		log.Errorf("DeleteItem %v", err)
		return nil, errorsx.WrapInternal(err, "删除 Gorse 推荐商品失败")
	}
	return &emptypb.Empty{}, nil
}

// GetItemSimilar 查询 Gorse 推荐相似商品。
func (s *RecommendGorseService) GetItemSimilar(
	ctx context.Context,
	req *adminv1.GetItemSimilarRequest,
) (*adminv1.ItemListResponse, error) {
	res, err := s.recommendGorseCase.GetItemSimilar(ctx, req.GetId(), req.GetRecommender(), req.GetCategory())
	if err != nil {
		log.Errorf("GetItemSimilar %v", err)
		return nil, errorsx.WrapInternal(err, "查询 Gorse 推荐相似商品失败")
	}
	return res, nil
}

// ExportData 导出 Gorse 推荐数据。
func (s *RecommendGorseService) ExportData(
	ctx context.Context,
	req *adminv1.ExportDataRequest,
) (*adminv1.ExportDataResponse, error) {
	res, err := s.recommendGorseCase.ExportData(ctx, req)
	if err != nil {
		log.Errorf("ExportData %v", err)
		return nil, errorsx.WrapInternal(err, "导出 Gorse 推荐数据失败")
	}
	return res, nil
}

// ImportData 导入 Gorse 推荐数据。
func (s *RecommendGorseService) ImportData(
	ctx context.Context,
	req *adminv1.ImportDataRequest,
) (*adminv1.ImportDataResponse, error) {
	res, err := s.recommendGorseCase.ImportData(ctx, req)
	if err != nil {
		log.Errorf("ImportData %v", err)
		return nil, errorsx.WrapInternal(err, "导入 Gorse 推荐数据失败")
	}
	return res, nil
}

// GetConfig 查询 Gorse 推荐配置。
func (s *RecommendGorseService) GetConfig(ctx context.Context, req *adminv1.GetConfigRequest) (*adminv1.ConfigResponse, error) {
	res, err := s.recommendGorseCase.GetConfig(ctx)
	if err != nil {
		log.Errorf("GetConfig %v", err)
		return nil, errorsx.WrapInternal(err, "查询 Gorse 推荐配置失败")
	}
	return res, nil
}

// SaveConfig 保存 Gorse 推荐配置。
func (s *RecommendGorseService) SaveConfig(ctx context.Context, req *adminv1.SaveConfigRequest) (*adminv1.ConfigResponse, error) {
	res, err := s.recommendGorseCase.SaveConfig(ctx, req.GetConfig())
	if err != nil {
		log.Errorf("SaveConfig %v", err)
		return nil, errorsx.WrapInternal(err, "保存 Gorse 推荐配置失败")
	}
	return res, nil
}

// ResetConfig 重置 Gorse 推荐配置。
func (s *RecommendGorseService) ResetConfig(ctx context.Context, req *adminv1.ResetConfigRequest) (*emptypb.Empty, error) {
	err := s.recommendGorseCase.ResetConfig(ctx)
	if err != nil {
		log.Errorf("ResetConfig %v", err)
		return nil, errorsx.WrapInternal(err, "重置 Gorse 推荐配置失败")
	}
	return &emptypb.Empty{}, nil
}

// PreviewExternal 预览 Gorse 推荐外部推荐脚本。
func (s *RecommendGorseService) PreviewExternal(
	ctx context.Context,
	req *adminv1.PreviewExternalRequest,
) (*adminv1.PreviewExternalResponse, error) {
	res, err := s.recommendGorseCase.PreviewExternal(ctx, req)
	if err != nil {
		log.Errorf("PreviewExternal %v", err)
		return nil, errorsx.WrapInternal(err, "预览 Gorse 推荐外部推荐脚本失败")
	}
	return res, nil
}

// PreviewRankerPrompt 预览 Gorse 推荐排序提示词。
func (s *RecommendGorseService) PreviewRankerPrompt(
	ctx context.Context,
	req *adminv1.PreviewRankerPromptRequest,
) (*adminv1.PreviewRankerPromptResponse, error) {
	res, err := s.recommendGorseCase.PreviewRankerPrompt(ctx, req)
	if err != nil {
		log.Errorf("PreviewRankerPrompt %v", err)
		return nil, errorsx.WrapInternal(err, "预览 Gorse 推荐排序提示词失败")
	}
	return res, nil
}
