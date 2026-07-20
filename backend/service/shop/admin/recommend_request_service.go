package admin

import (
	"context"
	"fmt"

	shopadminv1 "shop/api/gen/go/shop/admin/v1"
	"shop/pkg/errorsx"
	"shop/service/shop/admin/biz"

	"github.com/go-kratos/kratos/v3/log"
	"google.golang.org/grpc"
)

const _ = grpc.SupportPackageIsVersion7

// RecommendRequestService Admin推荐请求服务。
type RecommendRequestService struct {
	shopadminv1.UnimplementedRecommendRequestServiceServer
	recommendRequestCase *biz.RecommendRequestCase
}

// NewRecommendRequestService 创建Admin推荐请求服务。
func NewRecommendRequestService(recommendRequestCase *biz.RecommendRequestCase) *RecommendRequestService {
	return &RecommendRequestService{
		recommendRequestCase: recommendRequestCase,
	}
}

// PageRecommendRequest 查询推荐请求分页列表。
func (s *RecommendRequestService) PageRecommendRequest(
	ctx context.Context,
	req *shopadminv1.PageRecommendRequestRequest,
) (*shopadminv1.PageRecommendRequestResponse, error) {
	page, err := s.recommendRequestCase.PageRecommendRequest(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageRecommendRequest %v", err))
		return nil, errorsx.WrapInternal(err, "查询推荐请求分页列表失败")
	}
	return page, nil
}

// GetRecommendRequest 查询推荐请求详情。
func (s *RecommendRequestService) GetRecommendRequest(
	ctx context.Context,
	req *shopadminv1.GetRecommendRequestRequest,
) (*shopadminv1.RecommendRequestDetailResponse, error) {
	res, err := s.recommendRequestCase.GetRecommendRequest(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("GetRecommendRequest %v", err))
		return nil, errorsx.WrapInternal(err, "查询推荐请求详情失败")
	}
	return res, nil
}

// ListRecommendRequestEvent 查询推荐请求商品关联事件列表。
func (s *RecommendRequestService) ListRecommendRequestEvent(
	ctx context.Context,
	req *shopadminv1.ListRecommendRequestEventRequest,
) (*shopadminv1.ListRecommendRequestEventResponse, error) {
	res, err := s.recommendRequestCase.ListRecommendRequestEvent(ctx, req.GetRequestRecordId(), req.GetGoodsId(), req.GetPosition())
	if err != nil {
		log.Error(fmt.Sprintf("ListRecommendRequestEvent %v", err))
		return nil, errorsx.WrapInternal(err, "查询推荐请求事件失败")
	}
	return res, nil
}
