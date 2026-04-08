package app

import (
	"context"
	"errors"

	"shop/api/gen/go/app"
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
	recommendCase      *biz.RecommendCase
	recommendEventCase *biz.RecommendEventCase
}

// NewRecommendService 创建推荐服务。
func NewRecommendService(recommendCase *biz.RecommendCase, recommendEventCase *biz.RecommendEventCase) *RecommendService {
	var ss = RecommendService{
		recommendCase:      recommendCase,
		recommendEventCase: recommendEventCase,
	}
	return &ss
}

// RecommendAnonymousActor 获取匿名推荐主体。
func (s *RecommendService) RecommendAnonymousActor(ctx context.Context, req *emptypb.Empty) (*wrapperspb.Int64Value, error) {
	res, err := s.recommendCase.RecommendAnonymousActor(ctx, req)
	if err != nil {
		log.Error("RecommendAnonymousActor err:", err.Error())
		return nil, errors.New("获取匿名推荐主体失败")
	}
	return res, nil
}

// RecommendGoods 查询推荐商品列表。
func (s *RecommendService) RecommendGoods(ctx context.Context, req *app.RecommendGoodsRequest) (*app.RecommendGoodsResponse, error) {
	res, err := s.recommendCase.RecommendGoods(ctx, req)
	if err != nil {
		log.Error("RecommendGoods err:", err.Error())
		return nil, errors.New("查询推荐商品失败")
	}
	return res, nil
}

// RecommendExposureReport 上报推荐曝光事件。
func (s *RecommendService) RecommendExposureReport(ctx context.Context, req *app.RecommendExposureReportRequest) (*emptypb.Empty, error) {
	err := s.recommendEventCase.RecommendExposureReport(ctx, req)
	if err != nil {
		log.Error("RecommendExposureReport err:", err.Error())
		return nil, errors.New("上报推荐曝光失败")
	}
	return &emptypb.Empty{}, nil
}

// RecommendGoodsActionReport 上报推荐商品行为事件。
func (s *RecommendService) RecommendGoodsActionReport(ctx context.Context, req *app.RecommendGoodsActionReportRequest) (*emptypb.Empty, error) {
	err := s.recommendEventCase.RecommendGoodsActionReport(ctx, req)
	if err != nil {
		log.Error("RecommendGoodsActionReport err:", err.Error())
		return nil, errors.New("上报推荐商品行为失败")
	}
	return &emptypb.Empty{}, nil
}
