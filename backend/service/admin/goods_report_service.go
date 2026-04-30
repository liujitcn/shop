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

// GoodsReportService Admin 商品报表服务。
type GoodsReportService struct {
	adminv1.UnimplementedGoodsReportServiceServer
	goodsReportCase *biz.GoodsReportCase
}

// NewGoodsReportService 创建 Admin 商品报表服务。
func NewGoodsReportService(goodsReportCase *biz.GoodsReportCase) *GoodsReportService {
	return &GoodsReportService{
		goodsReportCase: goodsReportCase,
	}
}

// SummaryGoodsMonthReport 查询商品月报汇总。
func (s *GoodsReportService) SummaryGoodsMonthReport(ctx context.Context, req *adminv1.SummaryGoodsMonthReportRequest) (*adminv1.SummaryGoodsMonthReportResponse, error) {
	res, err := s.goodsReportCase.SummaryGoodsMonthReport(ctx, req)
	if err != nil {
		log.Errorf("SummaryGoodsMonthReport %v", err)
		return nil, errorsx.WrapInternal(err, "查询商品月报汇总失败")
	}
	return res, nil
}

// ListGoodsMonthReports 查询商品月报名细。
func (s *GoodsReportService) ListGoodsMonthReports(ctx context.Context, req *adminv1.ListGoodsMonthReportsRequest) (*adminv1.ListGoodsMonthReportsResponse, error) {
	res, err := s.goodsReportCase.ListGoodsMonthReports(ctx, req)
	if err != nil {
		log.Errorf("ListGoodsMonthReports %v", err)
		return nil, errorsx.WrapInternal(err, "查询商品月报名细失败")
	}
	return res, nil
}

// SummaryGoodsDayReport 查询商品日报汇总。
func (s *GoodsReportService) SummaryGoodsDayReport(ctx context.Context, req *adminv1.SummaryGoodsDayReportRequest) (*adminv1.SummaryGoodsDayReportResponse, error) {
	res, err := s.goodsReportCase.SummaryGoodsDayReport(ctx, req)
	if err != nil {
		log.Errorf("SummaryGoodsDayReport %v", err)
		return nil, errorsx.WrapInternal(err, "查询商品日报汇总失败")
	}
	return res, nil
}

// ListGoodsDayReports 查询商品日报明细。
func (s *GoodsReportService) ListGoodsDayReports(ctx context.Context, req *adminv1.ListGoodsDayReportsRequest) (*adminv1.ListGoodsDayReportsResponse, error) {
	res, err := s.goodsReportCase.ListGoodsDayReports(ctx, req)
	if err != nil {
		log.Errorf("ListGoodsDayReports %v", err)
		return nil, errorsx.WrapInternal(err, "查询商品日报明细失败")
	}
	return res, nil
}
