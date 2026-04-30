package app

import (
	"context"

	appv1 "shop/api/gen/go/app/v1"
	"shop/pkg/errorsx"
	"shop/service/app/biz"

	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

const _ = grpc.SupportPackageIsVersion7

// RecommendService 推荐服务。
type RecommendService struct {
	appv1.UnimplementedRecommendServiceServer
	recommendCase *biz.RecommendCase
}

// NewRecommendService 创建推荐服务。
func NewRecommendService(recommendCase *biz.RecommendCase) *RecommendService {
	var ss = RecommendService{
		recommendCase: recommendCase,
	}
	return &ss
}

// RecommendAnonymousActor 获取匿名推荐主体。
func (s *RecommendService) RecommendAnonymousActor(ctx context.Context, req *appv1.RecommendAnonymousActorRequest) (*appv1.RecommendAnonymousActorResponse, error) {
	res, err := s.recommendCase.RecommendAnonymousActor(ctx, req)
	if err != nil {
		log.Errorf("RecommendAnonymousActor %v", err)
		return nil, errorsx.WrapInternal(err, "获取匿名推荐主体失败")
	}
	return res, nil
}

// BindRecommendAnonymousActor 绑定匿名推荐主体到当前登录用户。
func (s *RecommendService) BindRecommendAnonymousActor(ctx context.Context, req *appv1.BindRecommendAnonymousActorRequest) (*emptypb.Empty, error) {
	err := s.recommendCase.BindRecommendAnonymousActor(ctx, req)
	if err != nil {
		log.Errorf("BindRecommendAnonymousActor %v", err)
		return nil, errorsx.WrapInternal(err, "绑定匿名推荐主体失败")
	}
	return &emptypb.Empty{}, nil
}

// RecommendGoods 查询推荐商品列表。
func (s *RecommendService) RecommendGoods(ctx context.Context, req *appv1.RecommendGoodsRequest) (*appv1.RecommendGoodsResponse, error) {
	res, err := s.recommendCase.RecommendGoods(ctx, req)
	if err != nil {
		log.Errorf("RecommendGoods %v", err)
		return nil, errorsx.WrapInternal(err, "查询推荐商品失败")
	}
	return res, nil
}

// RecommendEventReport 上报推荐事件。
func (s *RecommendService) RecommendEventReport(ctx context.Context, req *appv1.RecommendEventReportRequest) (*emptypb.Empty, error) {
	err := s.recommendCase.RecommendEventReport(ctx, req)
	if err != nil {
		log.Errorf("RecommendEventReport %v", err)
		return nil, errorsx.WrapInternal(err, "上报推荐事件失败")
	}
	return &emptypb.Empty{}, nil
}
