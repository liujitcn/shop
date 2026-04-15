package admin

import (
	"context"

	adminApi "shop/api/gen/go/admin"
	commonApi "shop/api/gen/go/common"
	"shop/pkg/errorsx"
	"shop/service/admin/biz"

	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/grpc"
)

const _ = grpc.SupportPackageIsVersion7

// GoodsAnalyticsService Admin 商品分析服务
type GoodsAnalyticsService struct {
	adminApi.UnimplementedGoodsAnalyticsServiceServer
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

// GetGoodsAnalyticsSummary 查询商品摘要指标
func (s *GoodsAnalyticsService) GetGoodsAnalyticsSummary(ctx context.Context, req *commonApi.AnalyticsTimeRequest) (*adminApi.GoodsAnalyticsSummaryResponse, error) {
	res, err := s.goodsAnalyticsCase.GetGoodsAnalyticsSummary(ctx, req)
	if err != nil {
		log.Errorf("GetGoodsAnalyticsSummary %v", err)
		return nil, errorsx.WrapInternal(err, "查询商品摘要指标失败")
	}
	return res, nil
}

// GetGoodsAnalyticsTrend 查询商品趋势
func (s *GoodsAnalyticsService) GetGoodsAnalyticsTrend(ctx context.Context, req *commonApi.AnalyticsTimeRequest) (*commonApi.AnalyticsTrendResponse, error) {
	res, err := s.goodsAnalyticsCase.GetGoodsAnalyticsTrend(ctx, req)
	if err != nil {
		log.Errorf("GetGoodsAnalyticsTrend %v", err)
		return nil, errorsx.WrapInternal(err, "查询商品趋势失败")
	}
	return res, nil
}

// GetGoodsAnalyticsPie 查询商品分类分布
func (s *GoodsAnalyticsService) GetGoodsAnalyticsPie(ctx context.Context, req *commonApi.AnalyticsTimeRequest) (*commonApi.AnalyticsPieResponse, error) {
	res, err := s.goodsAnalyticsCase.GetGoodsAnalyticsPie(ctx, req)
	if err != nil {
		log.Errorf("GetGoodsAnalyticsPie %v", err)
		return nil, errorsx.WrapInternal(err, "查询商品分类分布失败")
	}
	return res, nil
}

// GetGoodsAnalyticsRank 查询商品支付排行
func (s *GoodsAnalyticsService) GetGoodsAnalyticsRank(ctx context.Context, req *commonApi.AnalyticsTimeRequest) (*commonApi.AnalyticsRankResponse, error) {
	res, err := s.goodsAnalyticsCase.GetGoodsAnalyticsRank(ctx, req)
	if err != nil {
		log.Errorf("GetGoodsAnalyticsRank %v", err)
		return nil, errorsx.WrapInternal(err, "查询商品支付排行失败")
	}
	return res, nil
}
