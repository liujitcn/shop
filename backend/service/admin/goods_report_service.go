package admin

import (
	"context"

	adminApi "shop/api/gen/go/admin"
	"shop/pkg/errorsx"
	"shop/service/admin/biz"

	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/grpc"
)

const _ = grpc.SupportPackageIsVersion7

// GoodsReportService Admin 商品报表服务。
type GoodsReportService struct {
	adminApi.UnimplementedGoodsReportServiceServer
	goodsReportCase *biz.GoodsReportCase
}

// NewGoodsReportService 创建 Admin 商品报表服务。
func NewGoodsReportService(goodsReportCase *biz.GoodsReportCase) *GoodsReportService {
	return &GoodsReportService{
		goodsReportCase: goodsReportCase,
	}
}

// GoodsMonthReportSummary 查询商品月报汇总。
func (s *GoodsReportService) GoodsMonthReportSummary(ctx context.Context, req *adminApi.GoodsMonthReportSummaryRequest) (*adminApi.GoodsMonthReportSummaryResponse, error) {
	res, err := s.goodsReportCase.GoodsMonthReportSummary(ctx, req)
	if err != nil {
		log.Errorf("GoodsMonthReportSummary %v", err)
		return nil, errorsx.WrapInternal(err, "查询商品月报汇总失败")
	}
	return res, nil
}

// GoodsMonthReportList 查询商品月报名细。
func (s *GoodsReportService) GoodsMonthReportList(ctx context.Context, req *adminApi.GoodsMonthReportListRequest) (*adminApi.GoodsMonthReportListResponse, error) {
	res, err := s.goodsReportCase.GoodsMonthReportList(ctx, req)
	if err != nil {
		log.Errorf("GoodsMonthReportList %v", err)
		return nil, errorsx.WrapInternal(err, "查询商品月报名细失败")
	}
	return res, nil
}

// GoodsDayReportSummary 查询商品日报汇总。
func (s *GoodsReportService) GoodsDayReportSummary(ctx context.Context, req *adminApi.GoodsDayReportSummaryRequest) (*adminApi.GoodsDayReportSummaryResponse, error) {
	res, err := s.goodsReportCase.GoodsDayReportSummary(ctx, req)
	if err != nil {
		log.Errorf("GoodsDayReportSummary %v", err)
		return nil, errorsx.WrapInternal(err, "查询商品日报汇总失败")
	}
	return res, nil
}

// GoodsDayReportList 查询商品日报明细。
func (s *GoodsReportService) GoodsDayReportList(ctx context.Context, req *adminApi.GoodsDayReportListRequest) (*adminApi.GoodsDayReportListResponse, error) {
	res, err := s.goodsReportCase.GoodsDayReportList(ctx, req)
	if err != nil {
		log.Errorf("GoodsDayReportList %v", err)
		return nil, errorsx.WrapInternal(err, "查询商品日报明细失败")
	}
	return res, nil
}
