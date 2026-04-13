package app

import (
	"context"

	"shop/api/gen/go/app"
	"shop/pkg/errorsx"
	"shop/service/app/biz"

	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const _ = grpc.SupportPackageIsVersion7

// RecommendService 推荐服务。
type RecommendService struct {
	app.UnimplementedRecommendServiceServer
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
func (s *RecommendService) RecommendAnonymousActor(ctx context.Context, req *emptypb.Empty) (*wrapperspb.Int64Value, error) {
	res, err := s.recommendCase.RecommendAnonymousActor(ctx, req)
	if err != nil {
		log.Errorf("RecommendAnonymousActor %v", err)
		return nil, errorsx.WrapInternal(err, "获取匿名推荐主体失败")
	}
	return res, nil
}

// BindRecommendAnonymousActor 绑定匿名推荐主体到当前登录用户。
func (s *RecommendService) BindRecommendAnonymousActor(ctx context.Context, req *emptypb.Empty) (*emptypb.Empty, error) {
	err := s.recommendCase.BindRecommendAnonymousActor(ctx, req)
	if err != nil {
		log.Errorf("BindRecommendAnonymousActor %v", err)
		return nil, errorsx.WrapInternal(err, "绑定匿名推荐主体失败")
	}
	return &emptypb.Empty{}, nil
}

// RecommendGoods 查询推荐商品列表。
func (s *RecommendService) RecommendGoods(ctx context.Context, req *app.RecommendGoodsRequest) (*app.RecommendGoodsResponse, error) {
	res, err := s.recommendCase.RecommendGoods(ctx, req)
	if err != nil {
		log.Errorf("RecommendGoods %v", err)
		return nil, errorsx.WrapInternal(err, "查询推荐商品失败")
	}
	return res, nil
}

// RecommendExposureReport 上报推荐曝光事件。
func (s *RecommendService) RecommendExposureReport(ctx context.Context, req *app.RecommendExposureReportRequest) (*emptypb.Empty, error) {
	err := s.recommendCase.RecommendExposureReport(ctx, req)
	if err != nil {
		log.Errorf("RecommendExposureReport %v", err)
		return nil, errorsx.WrapInternal(err, "上报推荐曝光失败")
	}
	return &emptypb.Empty{}, nil
}

// RecommendGoodsActionReport 上报推荐商品行为事件。
func (s *RecommendService) RecommendGoodsActionReport(ctx context.Context, req *app.RecommendGoodsActionReportRequest) (*emptypb.Empty, error) {
	err := s.recommendCase.RecommendGoodsActionReport(ctx, req)
	if err != nil {
		log.Errorf("RecommendGoodsActionReport %v", err)
		return nil, errorsx.WrapInternal(err, "上报推荐商品行为失败")
	}
	return &emptypb.Empty{}, nil
}
