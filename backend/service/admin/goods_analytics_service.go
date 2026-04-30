package admin

import (
	"context"

	adminv1 "shop/api/gen/go/admin/v1"
	commonv1 "shop/api/gen/go/common/v1"
	"shop/pkg/errorsx"
	"shop/service/admin/biz"

	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/grpc"
)

const _ = grpc.SupportPackageIsVersion7

// GoodsAnalyticsService Admin 商品分析服务
type GoodsAnalyticsService struct {
	adminv1.UnimplementedGoodsAnalyticsServiceServer
	goodsAnalyticsCase *biz.GoodsAnalyticsCase
}

// NewGoodsAnalyticsService 创建 Admin 商品分析服务
func NewGoodsAnalyticsService(
	goodsAnalyticsCase *biz.GoodsAnalyticsCase,
) *GoodsAnalyticsService {
	return &GoodsAnalyticsService{
		goodsAnalyticsCase: goodsAnalyticsCase,
	}
}

// SummaryGoodsAnalytics 查询商品摘要指标
func (s *GoodsAnalyticsService) SummaryGoodsAnalytics(ctx context.Context, req *adminv1.SummaryGoodsAnalyticsRequest) (*adminv1.SummaryGoodsAnalyticsResponse, error) {
	res, err := s.goodsAnalyticsCase.SummaryGoodsAnalytics(ctx, req)
	if err != nil {
		log.Errorf("SummaryGoodsAnalytics %v", err)
		return nil, errorsx.WrapInternal(err, "查询商品摘要指标失败")
	}
	return res, nil
}

// TrendGoodsAnalytics 查询商品趋势
func (s *GoodsAnalyticsService) TrendGoodsAnalytics(ctx context.Context, req *adminv1.TrendGoodsAnalyticsRequest) (*commonv1.AnalyticsTrendResponse, error) {
	res, err := s.goodsAnalyticsCase.TrendGoodsAnalytics(ctx, req)
	if err != nil {
		log.Errorf("TrendGoodsAnalytics %v", err)
		return nil, errorsx.WrapInternal(err, "查询商品趋势失败")
	}
	return res, nil
}

// PieGoodsAnalytics 查询商品分类分布
func (s *GoodsAnalyticsService) PieGoodsAnalytics(ctx context.Context, req *adminv1.PieGoodsAnalyticsRequest) (*commonv1.AnalyticsPieResponse, error) {
	res, err := s.goodsAnalyticsCase.PieGoodsAnalytics(ctx, req)
	if err != nil {
		log.Errorf("PieGoodsAnalytics %v", err)
		return nil, errorsx.WrapInternal(err, "查询商品分类分布失败")
	}
	return res, nil
}

// RankGoodsAnalytics 查询商品支付排行
func (s *GoodsAnalyticsService) RankGoodsAnalytics(ctx context.Context, req *adminv1.RankGoodsAnalyticsRequest) (*commonv1.AnalyticsRankResponse, error) {
	res, err := s.goodsAnalyticsCase.RankGoodsAnalytics(ctx, req)
	if err != nil {
		log.Errorf("RankGoodsAnalytics %v", err)
		return nil, errorsx.WrapInternal(err, "查询商品支付排行失败")
	}
	return res, nil
}
