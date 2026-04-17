package admin

import (
	"context"

	adminApi "shop/api/gen/go/admin"
	"shop/pkg/errorsx"
	"shop/service/admin/biz"

	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/grpc"
)

const _ = grpc.SupportPackageIsVersion7

// RecommendModelVersionService Admin 推荐版本服务。
type RecommendModelVersionService struct {
	adminApi.UnimplementedRecommendModelVersionServiceServer
	recommendModelVersionCase *biz.RecommendModelVersionCase
}

// NewRecommendModelVersionService 创建 Admin 推荐版本服务。
func NewRecommendModelVersionService(recommendModelVersionCase *biz.RecommendModelVersionCase) *RecommendModelVersionService {
	return &RecommendModelVersionService{
		recommendModelVersionCase: recommendModelVersionCase,
	}
}

// PageRecommendModelVersion 查询推荐版本分页列表。
func (s *RecommendModelVersionService) PageRecommendModelVersion(ctx context.Context, req *adminApi.PageRecommendModelVersionRequest) (*adminApi.PageRecommendModelVersionResponse, error) {
	res, err := s.recommendModelVersionCase.PageRecommendModelVersion(ctx, req)
	if err != nil {
		log.Errorf("PageRecommendModelVersion %v", err)
		return nil, errorsx.WrapInternal(err, "查询推荐版本分页列表失败")
	}
	return res, nil
}

// PublishRecommendModelVersion 发布推荐版本。
func (s *RecommendModelVersionService) PublishRecommendModelVersion(ctx context.Context, req *adminApi.UpdateRecommendModelVersionPublishRequest) (*adminApi.UpdateRecommendModelVersionPublishResponse, error) {
	res, err := s.recommendModelVersionCase.PublishRecommendModelVersion(ctx, req)
	if err != nil {
		log.Errorf("PublishRecommendModelVersion %v", err)
		return nil, errorsx.WrapIfNeeded(err, errorsx.Internal("发布推荐版本失败"))
	}
	return res, nil
}
