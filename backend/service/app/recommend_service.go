package app

import (
	"context"
	"errors"

	"shop/api/gen/go/app"
	"shop/service/app/biz"

	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

const _ = grpc.SupportPackageIsVersion7

// RecommendService 推荐服务。
type RecommendService struct {
	app.UnimplementedRecommendServiceServer
	recommendCase *biz.RecommendCase
}

// NewRecommendService 创建推荐服务。
func NewRecommendService(recommendCase *biz.RecommendCase, _ *biz.RecommendEventCase) *RecommendService {
	var ss = RecommendService{
		recommendCase: recommendCase,
	}
	return &ss
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

// RecommendExposure 记录推荐曝光。
func (s *RecommendService) RecommendExposure(ctx context.Context, req *app.RecommendExposureRequest) (*emptypb.Empty, error) {
	err := s.recommendCase.RecommendExposure(ctx, req)
	if err != nil {
		log.Error("RecommendExposure err:", err.Error())
		return nil, errors.New("记录推荐曝光失败")
	}
	return &emptypb.Empty{}, nil
}
