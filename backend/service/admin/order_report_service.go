package admin

import (
	"context"

	adminv1 "shop/api/gen/go/admin/v1"
	"shop/pkg/errorsx"
	"shop/service/admin/biz"

	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/grpc"
)

const _ = grpc.SupportPackageIsVersion7

// OrderReportService Admin 订单报表服务
type OrderReportService struct {
	adminv1.UnimplementedOrderReportServiceServer
	orderReportCase *biz.OrderReportCase
}

// NewOrderReportService 创建 Admin 订单报表服务
func NewOrderReportService(orderReportCase *biz.OrderReportCase) *OrderReportService {
	return &OrderReportService{
		orderReportCase: orderReportCase,
	}
}

// SummaryOrderMonthReport 查询订单月报汇总
func (s *OrderReportService) SummaryOrderMonthReport(ctx context.Context, req *adminv1.SummaryOrderMonthReportRequest) (*adminv1.SummaryOrderMonthReportResponse, error) {
	res, err := s.orderReportCase.SummaryOrderMonthReport(ctx, req)
	if err != nil {
		log.Errorf("SummaryOrderMonthReport %v", err)
		return nil, errorsx.WrapInternal(err, "查询订单月报汇总失败")
	}
	return res, nil
}

// ListOrderMonthReports 查询订单月报名细
func (s *OrderReportService) ListOrderMonthReports(ctx context.Context, req *adminv1.ListOrderMonthReportsRequest) (*adminv1.ListOrderMonthReportsResponse, error) {
	res, err := s.orderReportCase.ListOrderMonthReports(ctx, req)
	if err != nil {
		log.Errorf("ListOrderMonthReports %v", err)
		return nil, errorsx.WrapInternal(err, "查询订单月报名细失败")
	}
	return res, nil
}

// SummaryOrderDayReport 查询订单日报汇总
func (s *OrderReportService) SummaryOrderDayReport(ctx context.Context, req *adminv1.SummaryOrderDayReportRequest) (*adminv1.SummaryOrderDayReportResponse, error) {
	res, err := s.orderReportCase.SummaryOrderDayReport(ctx, req)
	if err != nil {
		log.Errorf("SummaryOrderDayReport %v", err)
		return nil, errorsx.WrapInternal(err, "查询订单日报汇总失败")
	}
	return res, nil
}

// ListOrderDayReports 查询订单日报明细
func (s *OrderReportService) ListOrderDayReports(ctx context.Context, req *adminv1.ListOrderDayReportsRequest) (*adminv1.ListOrderDayReportsResponse, error) {
	res, err := s.orderReportCase.ListOrderDayReports(ctx, req)
	if err != nil {
		log.Errorf("ListOrderDayReports %v", err)
		return nil, errorsx.WrapInternal(err, "查询订单日报明细失败")
	}
	return res, nil
}
