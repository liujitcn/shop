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

// OrderAnalyticsService Admin 订单分析服务
type OrderAnalyticsService struct {
	adminv1.UnimplementedOrderAnalyticsServiceServer
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

// SummaryOrderAnalytics 查询订单摘要指标
func (s *OrderAnalyticsService) SummaryOrderAnalytics(ctx context.Context, req *adminv1.SummaryOrderAnalyticsRequest) (*adminv1.SummaryOrderAnalyticsResponse, error) {
	res, err := s.orderAnalyticsCase.SummaryOrderAnalytics(ctx, req)
	if err != nil {
		log.Errorf("SummaryOrderAnalytics %v", err)
		return nil, errorsx.WrapInternal(err, "查询订单摘要指标失败")
	}
	return res, nil
}

// TrendOrderAnalytics 查询订单趋势
func (s *OrderAnalyticsService) TrendOrderAnalytics(ctx context.Context, req *adminv1.TrendOrderAnalyticsRequest) (*commonv1.AnalyticsTrendResponse, error) {
	res, err := s.orderAnalyticsCase.TrendOrderAnalytics(ctx, req)
	if err != nil {
		log.Errorf("TrendOrderAnalytics %v", err)
		return nil, errorsx.WrapInternal(err, "查询订单趋势失败")
	}
	return res, nil
}

// PieOrderAnalytics 查询订单状态分布
func (s *OrderAnalyticsService) PieOrderAnalytics(ctx context.Context, req *adminv1.PieOrderAnalyticsRequest) (*commonv1.AnalyticsPieResponse, error) {
	res, err := s.orderAnalyticsCase.PieOrderAnalytics(ctx, req)
	if err != nil {
		log.Errorf("PieOrderAnalytics %v", err)
		return nil, errorsx.WrapInternal(err, "查询订单状态分布失败")
	}
	return res, nil
}
