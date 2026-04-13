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

// OrderAnalyticsService Admin 订单分析服务
type OrderAnalyticsService struct {
	adminApi.UnimplementedOrderAnalyticsServiceServer
	orderAnalyticsCase *biz.OrderAnalyticsCase
}

// NewOrderAnalyticsService 创建 Admin 订单分析服务
func NewOrderAnalyticsService(
	orderAnalyticsCase *biz.OrderAnalyticsCase,
) *OrderAnalyticsService {
	return &OrderAnalyticsService{
		orderAnalyticsCase: orderAnalyticsCase,
	}
}

// GetOrderAnalyticsSummary 查询订单摘要指标
func (s *OrderAnalyticsService) GetOrderAnalyticsSummary(ctx context.Context, req *commonApi.AnalyticsTimeRequest) (*adminApi.OrderAnalyticsSummaryResponse, error) {
	res, err := s.orderAnalyticsCase.GetOrderAnalyticsSummary(ctx, req)
	if err != nil {
		log.Errorf("GetOrderAnalyticsSummary %v", err)
		return nil, errorsx.WrapInternal(err, "查询订单摘要指标失败")
	}
	return res, nil
}

// GetOrderAnalyticsTrend 查询订单趋势
func (s *OrderAnalyticsService) GetOrderAnalyticsTrend(ctx context.Context, req *commonApi.AnalyticsTimeRequest) (*commonApi.AnalyticsTrendResponse, error) {
	res, err := s.orderAnalyticsCase.GetOrderAnalyticsTrend(ctx, req)
	if err != nil {
		log.Errorf("GetOrderAnalyticsTrend %v", err)
		return nil, errorsx.WrapInternal(err, "查询订单趋势失败")
	}
	return res, nil
}

// GetOrderAnalyticsPie 查询订单状态分布
func (s *OrderAnalyticsService) GetOrderAnalyticsPie(ctx context.Context, req *commonApi.AnalyticsTimeRequest) (*commonApi.AnalyticsPieResponse, error) {
	res, err := s.orderAnalyticsCase.GetOrderAnalyticsPie(ctx, req)
	if err != nil {
		log.Errorf("GetOrderAnalyticsPie %v", err)
		return nil, errorsx.WrapInternal(err, "查询订单状态分布失败")
	}
	return res, nil
}
