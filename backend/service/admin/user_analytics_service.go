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

// UserAnalyticsService Admin 用户分析服务
type UserAnalyticsService struct {
	adminv1.UnimplementedUserAnalyticsServiceServer
	userAnalyticsCase *biz.UserAnalyticsCase
}

// NewUserAnalyticsService 创建 Admin 用户分析服务
func NewUserAnalyticsService(
	userAnalyticsCase *biz.UserAnalyticsCase,
) *UserAnalyticsService {
	return &UserAnalyticsService{
		userAnalyticsCase: userAnalyticsCase,
	}
}

// SummaryUserAnalytics 查询用户摘要指标
func (s *UserAnalyticsService) SummaryUserAnalytics(ctx context.Context, req *adminv1.SummaryUserAnalyticsRequest) (*adminv1.SummaryUserAnalyticsResponse, error) {
	res, err := s.userAnalyticsCase.SummaryUserAnalytics(ctx, req)
	if err != nil {
		log.Errorf("SummaryUserAnalytics %v", err)
		return nil, errorsx.WrapInternal(err, "查询用户摘要指标失败")
	}
	return res, nil
}

// TrendUserAnalytics 查询用户趋势
func (s *UserAnalyticsService) TrendUserAnalytics(ctx context.Context, req *adminv1.TrendUserAnalyticsRequest) (*commonv1.AnalyticsTrendResponse, error) {
	res, err := s.userAnalyticsCase.TrendUserAnalytics(ctx, req)
	if err != nil {
		log.Errorf("TrendUserAnalytics %v", err)
		return nil, errorsx.WrapInternal(err, "查询用户趋势失败")
	}
	return res, nil
}

// RankUserAnalytics 查询用户行为覆盖排行
func (s *UserAnalyticsService) RankUserAnalytics(ctx context.Context, req *adminv1.RankUserAnalyticsRequest) (*commonv1.AnalyticsRankResponse, error) {
	res, err := s.userAnalyticsCase.RankUserAnalytics(ctx, req)
	if err != nil {
		log.Errorf("RankUserAnalytics %v", err)
		return nil, errorsx.WrapInternal(err, "查询用户行为覆盖排行失败")
	}
	return res, nil
}
