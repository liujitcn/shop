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

// GoodsReportService Admin 商品报表服务。
type GoodsReportService struct {
	shopadminv1.UnimplementedGoodsReportServiceServer
	goodsReportCase *biz.GoodsReportCase
}

// NewGoodsReportService 创建 Admin 商品报表服务。
func NewGoodsReportService(goodsReportCase *biz.GoodsReportCase) *GoodsReportService {
	return &GoodsReportService{
		goodsReportCase: goodsReportCase,
	}
}

// SummaryGoodsMonthReport 查询商品月报汇总。
func (s *GoodsReportService) SummaryGoodsMonthReport(ctx context.Context, req *shopadminv1.SummaryGoodsMonthReportRequest) (*shopadminv1.SummaryGoodsMonthReportResponse, error) {
	res, err := s.goodsReportCase.SummaryGoodsMonthReport(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("SummaryGoodsMonthReport %v", err))
		return nil, errorsx.WrapInternal(err, "查询商品月报汇总失败")
	}
	return res, nil
}

// ListGoodsMonthReport 查询商品月报名细。
func (s *GoodsReportService) ListGoodsMonthReport(ctx context.Context, req *shopadminv1.ListGoodsMonthReportRequest) (*shopadminv1.ListGoodsMonthReportResponse, error) {
	res, err := s.goodsReportCase.ListGoodsMonthReport(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("ListGoodsMonthReport %v", err))
		return nil, errorsx.WrapInternal(err, "查询商品月报名细失败")
	}
	return res, nil
}

// SummaryGoodsDayReport 查询商品日报汇总。
func (s *GoodsReportService) SummaryGoodsDayReport(ctx context.Context, req *shopadminv1.SummaryGoodsDayReportRequest) (*shopadminv1.SummaryGoodsDayReportResponse, error) {
	res, err := s.goodsReportCase.SummaryGoodsDayReport(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("SummaryGoodsDayReport %v", err))
		return nil, errorsx.WrapInternal(err, "查询商品日报汇总失败")
	}
	return res, nil
}

// ListGoodsDayReport 查询商品日报明细。
func (s *GoodsReportService) ListGoodsDayReport(ctx context.Context, req *shopadminv1.ListGoodsDayReportRequest) (*shopadminv1.ListGoodsDayReportResponse, error) {
	res, err := s.goodsReportCase.ListGoodsDayReport(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("ListGoodsDayReport %v", err))
		return nil, errorsx.WrapInternal(err, "查询商品日报明细失败")
	}
	return res, nil
}
