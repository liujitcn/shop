package admin

import (
	"context"
	"fmt"

	shopadminv1 "shop/api/gen/go/shop/admin/v1"
	"shop/pkg/errorsx"
	"shop/service/shop/admin/biz"

	"github.com/go-kratos/kratos/v3/log"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

const _ = grpc.SupportPackageIsVersion7

// RecommendGorseService Admin Gorse 推荐服务。
type RecommendGorseService struct {
	shopadminv1.UnimplementedRecommendGorseServiceServer
	recommendGorseCase *biz.RecommendGorseCase
}

// NewRecommendGorseService 创建 Admin Gorse 推荐服务。
func NewRecommendGorseService(recommendGorseCase *biz.RecommendGorseCase) *RecommendGorseService {
	return &RecommendGorseService{
		recommendGorseCase: recommendGorseCase,
	}
}

// GetTimeSeries 查询 Gorse 推荐时间序列。
func (s *RecommendGorseService) GetTimeSeries(ctx context.Context, req *shopadminv1.GetTimeSeriesRequest) (*shopadminv1.TimeSeriesResponse, error) {
	res, err := s.recommendGorseCase.GetTimeSeries(ctx, req.GetName(), req.GetBegin(), req.GetEnd())
	if err != nil {
		log.Error(fmt.Sprintf("GetTimeSeries %v", err))
		return nil, errorsx.WrapInternal(err, "查询 Gorse 推荐时间序列失败")
	}
	return res, nil
}

// OptionCategory 查询 Gorse 推荐分类列表。
func (s *RecommendGorseService) OptionCategory(ctx context.Context, req *shopadminv1.OptionCategoryRequest) (*shopadminv1.OptionCategoryResponse, error) {
	res, err := s.recommendGorseCase.OptionCategory(ctx)
	if err != nil {
		log.Error(fmt.Sprintf("OptionCategory %v", err))
		return nil, errorsx.WrapInternal(err, "查询 Gorse 推荐分类列表失败")
	}
	return res, nil
}

// ListDashboardItem 查询 Gorse 推荐仪表盘推荐商品。
func (s *RecommendGorseService) ListDashboardItem(
	ctx context.Context,
	req *shopadminv1.ListDashboardItemRequest,
) (*shopadminv1.ListDashboardItemResponse, error) {
	res, err := s.recommendGorseCase.ListDashboardItem(ctx, req.GetRecommender(), req.GetCategory(), req.GetEnd())
	if err != nil {
		log.Error(fmt.Sprintf("ListDashboardItem %v", err))
		return nil, errorsx.WrapInternal(err, "查询 Gorse 推荐仪表盘推荐商品失败")
	}
	return res, nil
}

// ListTask 查询 Gorse 推荐任务状态。
func (s *RecommendGorseService) ListTask(ctx context.Context, req *shopadminv1.ListTaskRequest) (*shopadminv1.ListTaskResponse, error) {
	res, err := s.recommendGorseCase.ListTask(ctx)
	if err != nil {
		log.Error(fmt.Sprintf("ListTask %v", err))
		return nil, errorsx.WrapInternal(err, "查询 Gorse 推荐任务状态失败")
	}
	return res, nil
}

// PageUser 查询 Gorse 推荐用户列表。
func (s *RecommendGorseService) PageUser(
	ctx context.Context,
	req *shopadminv1.PageUserRequest,
) (*shopadminv1.PageUserResponse, error) {
	res, err := s.recommendGorseCase.PageUser(ctx, req.GetCursor(), req.GetN())
	if err != nil {
		log.Error(fmt.Sprintf("PageUser %v", err))
		return nil, errorsx.WrapInternal(err, "查询 Gorse 推荐用户列表失败")
	}
	return res, nil
}

// GetUser 查询 Gorse 推荐用户。
func (s *RecommendGorseService) GetUser(ctx context.Context, req *shopadminv1.GetUserRequest) (*shopadminv1.UserResponse, error) {
	res, err := s.recommendGorseCase.GetUser(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("GetUser %v", err))
		return nil, errorsx.WrapInternal(err, "查询 Gorse 推荐用户失败")
	}
	return res, nil
}

// DeleteUser 删除 Gorse 推荐用户。
func (s *RecommendGorseService) DeleteUser(ctx context.Context, req *shopadminv1.DeleteUserRequest) (*emptypb.Empty, error) {
	err := s.recommendGorseCase.DeleteUser(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("DeleteUser %v", err))
		return nil, errorsx.WrapInternal(err, "删除 Gorse 推荐用户失败")
	}
	return &emptypb.Empty{}, nil
}

// GetUserSimilar 查询 Gorse 推荐相似用户。
func (s *RecommendGorseService) GetUserSimilar(
	ctx context.Context,
	req *shopadminv1.GetUserSimilarRequest,
) (*shopadminv1.UserSimilarResponse, error) {
	res, err := s.recommendGorseCase.GetUserSimilar(ctx, req.GetId(), req.GetRecommender())
	if err != nil {
		log.Error(fmt.Sprintf("GetUserSimilar %v", err))
		return nil, errorsx.WrapInternal(err, "查询 Gorse 推荐相似用户失败")
	}
	return res, nil
}

// GetUserFeedback 查询 Gorse 推荐用户反馈。
func (s *RecommendGorseService) GetUserFeedback(
	ctx context.Context,
	req *shopadminv1.GetUserFeedbackRequest,
) (*shopadminv1.FeedbackResponse, error) {
	res, err := s.recommendGorseCase.GetUserFeedback(ctx, req.GetId(), req.GetFeedbackType(), req.GetOffset(), req.GetN())
	if err != nil {
		log.Error(fmt.Sprintf("GetUserFeedback %v", err))
		return nil, errorsx.WrapInternal(err, "查询 Gorse 推荐用户反馈失败")
	}
	return res, nil
}

// GetUserRecommend 查询 Gorse 推荐用户推荐结果。
func (s *RecommendGorseService) GetUserRecommend(
	ctx context.Context,
	req *shopadminv1.GetUserRecommendRequest,
) (*shopadminv1.ItemListResponse, error) {
	res, err := s.recommendGorseCase.GetUserRecommend(ctx, req.GetId(), req.GetRecommender(), req.GetCategory(), req.GetN())
	if err != nil {
		log.Error(fmt.Sprintf("GetUserRecommend %v", err))
		return nil, errorsx.WrapInternal(err, "查询 Gorse 推荐用户推荐结果失败")
	}
	return res, nil
}

// PageItem 查询 Gorse 推荐商品列表。
func (s *RecommendGorseService) PageItem(
	ctx context.Context,
	req *shopadminv1.PageItemRequest,
) (*shopadminv1.PageItemResponse, error) {
	res, err := s.recommendGorseCase.PageItem(ctx, req.GetCursor(), req.GetN())
	if err != nil {
		log.Error(fmt.Sprintf("PageItem %v", err))
		return nil, errorsx.WrapInternal(err, "查询 Gorse 推荐商品列表失败")
	}
	return res, nil
}

// GetItem 查询 Gorse 推荐商品。
func (s *RecommendGorseService) GetItem(ctx context.Context, req *shopadminv1.GetItemRequest) (*shopadminv1.Item, error) {
	res, err := s.recommendGorseCase.GetItem(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("GetItem %v", err))
		return nil, errorsx.WrapInternal(err, "查询 Gorse 推荐商品失败")
	}
	return res, nil
}

// DeleteItem 删除 Gorse 推荐商品。
func (s *RecommendGorseService) DeleteItem(ctx context.Context, req *shopadminv1.DeleteItemRequest) (*emptypb.Empty, error) {
	err := s.recommendGorseCase.DeleteItem(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("DeleteItem %v", err))
		return nil, errorsx.WrapInternal(err, "删除 Gorse 推荐商品失败")
	}
	return &emptypb.Empty{}, nil
}

// GetItemSimilar 查询 Gorse 推荐相似商品。
func (s *RecommendGorseService) GetItemSimilar(
	ctx context.Context,
	req *shopadminv1.GetItemSimilarRequest,
) (*shopadminv1.ItemListResponse, error) {
	res, err := s.recommendGorseCase.GetItemSimilar(ctx, req.GetId(), req.GetRecommender(), req.GetCategory())
	if err != nil {
		log.Error(fmt.Sprintf("GetItemSimilar %v", err))
		return nil, errorsx.WrapInternal(err, "查询 Gorse 推荐相似商品失败")
	}
	return res, nil
}

// ExportData 导出 Gorse 推荐数据。
func (s *RecommendGorseService) ExportData(
	ctx context.Context,
	req *shopadminv1.ExportDataRequest,
) (*shopadminv1.ExportDataResponse, error) {
	res, err := s.recommendGorseCase.ExportData(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("ExportData %v", err))
		return nil, errorsx.WrapInternal(err, "导出 Gorse 推荐数据失败")
	}
	return res, nil
}

// ImportData 导入 Gorse 推荐数据。
func (s *RecommendGorseService) ImportData(
	ctx context.Context,
	req *shopadminv1.ImportDataRequest,
) (*shopadminv1.ImportDataResponse, error) {
	res, err := s.recommendGorseCase.ImportData(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("ImportData %v", err))
		return nil, errorsx.WrapInternal(err, "导入 Gorse 推荐数据失败")
	}
	return res, nil
}

// GetConfig 查询 Gorse 推荐配置。
func (s *RecommendGorseService) GetConfig(ctx context.Context, req *shopadminv1.GetConfigRequest) (*shopadminv1.ConfigResponse, error) {
	res, err := s.recommendGorseCase.GetConfig(ctx)
	if err != nil {
		log.Error(fmt.Sprintf("GetConfig %v", err))
		return nil, errorsx.WrapInternal(err, "查询 Gorse 推荐配置失败")
	}
	return res, nil
}

// SaveConfig 保存 Gorse 推荐配置。
func (s *RecommendGorseService) SaveConfig(ctx context.Context, req *shopadminv1.SaveConfigRequest) (*shopadminv1.ConfigResponse, error) {
	res, err := s.recommendGorseCase.SaveConfig(ctx, req.GetConfig())
	if err != nil {
		log.Error(fmt.Sprintf("SaveConfig %v", err))
		return nil, errorsx.WrapInternal(err, "保存 Gorse 推荐配置失败")
	}
	return res, nil
}

// ResetConfig 重置 Gorse 推荐配置。
func (s *RecommendGorseService) ResetConfig(ctx context.Context, req *shopadminv1.ResetConfigRequest) (*emptypb.Empty, error) {
	err := s.recommendGorseCase.ResetConfig(ctx)
	if err != nil {
		log.Error(fmt.Sprintf("ResetConfig %v", err))
		return nil, errorsx.WrapInternal(err, "重置 Gorse 推荐配置失败")
	}
	return &emptypb.Empty{}, nil
}

// PreviewExternal 预览 Gorse 推荐外部推荐脚本。
func (s *RecommendGorseService) PreviewExternal(
	ctx context.Context,
	req *shopadminv1.PreviewExternalRequest,
) (*shopadminv1.PreviewExternalResponse, error) {
	res, err := s.recommendGorseCase.PreviewExternal(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PreviewExternal %v", err))
		return nil, errorsx.WrapInternal(err, "预览 Gorse 推荐外部推荐脚本失败")
	}
	return res, nil
}

// PreviewRankerPrompt 预览 Gorse 推荐排序提示词。
func (s *RecommendGorseService) PreviewRankerPrompt(
	ctx context.Context,
	req *shopadminv1.PreviewRankerPromptRequest,
) (*shopadminv1.PreviewRankerPromptResponse, error) {
	res, err := s.recommendGorseCase.PreviewRankerPrompt(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PreviewRankerPrompt %v", err))
		return nil, errorsx.WrapInternal(err, "预览 Gorse 推荐排序提示词失败")
	}
	return res, nil
}
