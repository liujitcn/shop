package admin

import (
	"context"
	"errors"

	adminApi "shop/api/gen/go/admin"
	"shop/service/admin/biz"

	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/grpc"
)

const _ = grpc.SupportPackageIsVersion7

// OrderReportService Admin 订单报表服务
type OrderReportService struct {
	adminApi.UnimplementedOrderReportServiceServer
	orderReportCase *biz.OrderReportCase
}

// NewOrderReportService 创建 Admin 订单报表服务
func NewOrderReportService(orderReportCase *biz.OrderReportCase) *OrderReportService {
	return &OrderReportService{
		orderReportCase: orderReportCase,
	}
}

// OrderMonthReportSummary 查询订单月报汇总
func (s *OrderReportService) OrderMonthReportSummary(ctx context.Context, req *adminApi.OrderMonthReportSummaryRequest) (*adminApi.OrderMonthReportSummaryResponse, error) {
	res, err := s.orderReportCase.OrderMonthReportSummary(ctx, req)
	if err != nil {
		log.Error("OrderMonthReportSummary err:", err.Error())
		return nil, errors.New("查询订单月报汇总失败")
	}
	return res, nil
}

// OrderMonthReportList 查询订单月报名细
func (s *OrderReportService) OrderMonthReportList(ctx context.Context, req *adminApi.OrderMonthReportListRequest) (*adminApi.OrderMonthReportListResponse, error) {
	res, err := s.orderReportCase.OrderMonthReportList(ctx, req)
	if err != nil {
		log.Error("OrderMonthReportList err:", err.Error())
		return nil, errors.New("查询订单月报名细失败")
	}
	return res, nil
}

// OrderDayReportSummary 查询订单日报汇总
func (s *OrderReportService) OrderDayReportSummary(ctx context.Context, req *adminApi.OrderDayReportSummaryRequest) (*adminApi.OrderDayReportSummaryResponse, error) {
	res, err := s.orderReportCase.OrderDayReportSummary(ctx, req)
	if err != nil {
		log.Error("OrderDayReportSummary err:", err.Error())
		return nil, errors.New("查询订单日报汇总失败")
	}
	return res, nil
}

// OrderDayReportList 查询订单日报明细
func (s *OrderReportService) OrderDayReportList(ctx context.Context, req *adminApi.OrderDayReportListRequest) (*adminApi.OrderDayReportListResponse, error) {
	res, err := s.orderReportCase.OrderDayReportList(ctx, req)
	if err != nil {
		log.Error("OrderDayReportList err:", err.Error())
		return nil, errors.New("查询订单日报明细失败")
	}
	return res, nil
}
