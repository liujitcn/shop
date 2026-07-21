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

// OrderReportService Admin 订单报表服务
type OrderReportService struct {
	shopadminv1.UnimplementedOrderReportServiceServer
	orderReportCase *biz.OrderReportCase
}

// NewOrderReportService 创建 Admin 订单报表服务
func NewOrderReportService(orderReportCase *biz.OrderReportCase) *OrderReportService {
	return &OrderReportService{
		orderReportCase: orderReportCase,
	}
}

// ListOrderDayReport 查询订单日报明细
func (s *OrderReportService) ListOrderDayReport(ctx context.Context, req *shopadminv1.ListOrderDayReportRequest) (*shopadminv1.ListOrderDayReportResponse, error) {
	res, err := s.orderReportCase.ListOrderDayReport(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("ListOrderDayReport %v", err))
		return nil, errorsx.WrapInternal(err, "查询订单日报明细失败")
	}
	return res, nil
}

// ListOrderMonthReport 查询订单月报名细
func (s *OrderReportService) ListOrderMonthReport(ctx context.Context, req *shopadminv1.ListOrderMonthReportRequest) (*shopadminv1.ListOrderMonthReportResponse, error) {
	res, err := s.orderReportCase.ListOrderMonthReport(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("ListOrderMonthReport %v", err))
		return nil, errorsx.WrapInternal(err, "查询订单月报名细失败")
	}
	return res, nil
}

// SummaryOrderMonthReport 查询订单月报汇总
func (s *OrderReportService) SummaryOrderMonthReport(ctx context.Context, req *shopadminv1.SummaryOrderMonthReportRequest) (*shopadminv1.SummaryOrderMonthReportResponse, error) {
	res, err := s.orderReportCase.SummaryOrderMonthReport(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("SummaryOrderMonthReport %v", err))
		return nil, errorsx.WrapInternal(err, "查询订单月报汇总失败")
	}
	return res, nil
}

// SummaryOrderDayReport 查询订单日报汇总
func (s *OrderReportService) SummaryOrderDayReport(ctx context.Context, req *shopadminv1.SummaryOrderDayReportRequest) (*shopadminv1.SummaryOrderDayReportResponse, error) {
	res, err := s.orderReportCase.SummaryOrderDayReport(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("SummaryOrderDayReport %v", err))
		return nil, errorsx.WrapInternal(err, "查询订单日报汇总失败")
	}
	return res, nil
}
