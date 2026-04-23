package admin

import (
	"context"

	adminApi "shop/api/gen/go/admin"
	"shop/pkg/errorsx"
	"shop/service/admin/biz"

	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const _ = grpc.SupportPackageIsVersion7

// RecommendRequestService Admin推荐请求服务。
type RecommendRequestService struct {
	adminApi.UnimplementedRecommendRequestServiceServer
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
	req *adminApi.PageRecommendRequestRequest,
) (*adminApi.PageRecommendRequestResponse, error) {
	page, err := s.recommendRequestCase.PageRecommendRequest(ctx, req)
	if err != nil {
		log.Errorf("PageRecommendRequest %v", err)
		return nil, errorsx.WrapInternal(err, "查询推荐请求分页列表失败")
	}
	return page, nil
}

// GetRecommendRequest 查询推荐请求详情。
func (s *RecommendRequestService) GetRecommendRequest(
	ctx context.Context,
	req *wrapperspb.Int64Value,
) (*adminApi.RecommendRequestDetailResponse, error) {
	res, err := s.recommendRequestCase.GetRecommendRequest(ctx, req.GetValue())
	if err != nil {
		log.Errorf("GetRecommendRequest %v", err)
		return nil, errorsx.WrapInternal(err, "查询推荐请求详情失败")
	}
	return res, nil
}

// GetRecommendRequestEvent 查询推荐请求商品关联事件。
func (s *RecommendRequestService) GetRecommendRequestEvent(
	ctx context.Context,
	req *adminApi.GetRecommendRequestEventRequest,
) (*adminApi.GetRecommendRequestEventResponse, error) {
	res, err := s.recommendRequestCase.GetRecommendRequestEvent(ctx, req.GetRequestRecordId(), req.GetGoodsId(), req.GetPosition())
	if err != nil {
		log.Errorf("GetRecommendRequestEvent %v", err)
		return nil, errorsx.WrapInternal(err, "查询推荐请求事件失败")
	}
	return res, nil
}
