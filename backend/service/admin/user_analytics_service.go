package admin

import (
	"context"
	"errors"

	adminApi "shop/api/gen/go/admin"
	commonApi "shop/api/gen/go/common"
	"shop/service/admin/biz"

	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/grpc"
)

const _ = grpc.SupportPackageIsVersion7

// UserAnalyticsService Admin 用户分析服务
type UserAnalyticsService struct {
	adminApi.UnimplementedUserAnalyticsServiceServer
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

// GetUserAnalyticsSummary 查询用户摘要指标
func (s *UserAnalyticsService) GetUserAnalyticsSummary(ctx context.Context, req *commonApi.AnalyticsTimeRequest) (*adminApi.UserAnalyticsSummaryResponse, error) {
	res, err := s.userAnalyticsCase.GetUserAnalyticsSummary(ctx, req)
	if err != nil {
		log.Error("GetUserAnalyticsSummary err:", err.Error())
		return nil, errors.New("查询用户摘要指标失败")
	}
	return res, nil
}

// GetUserAnalyticsTrend 查询用户趋势
func (s *UserAnalyticsService) GetUserAnalyticsTrend(ctx context.Context, req *commonApi.AnalyticsTimeRequest) (*commonApi.AnalyticsTrendResponse, error) {
	res, err := s.userAnalyticsCase.GetUserAnalyticsTrend(ctx, req)
	if err != nil {
		log.Error("GetUserAnalyticsTrend err:", err.Error())
		return nil, errors.New("查询用户趋势失败")
	}
	return res, nil
}

// GetUserAnalyticsRank 查询用户行为覆盖排行
func (s *UserAnalyticsService) GetUserAnalyticsRank(ctx context.Context, req *commonApi.AnalyticsTimeRequest) (*commonApi.AnalyticsRankResponse, error) {
	res, err := s.userAnalyticsCase.GetUserAnalyticsRank(ctx, req)
	if err != nil {
		log.Error("GetUserAnalyticsRank err:", err.Error())
		return nil, errors.New("查询用户行为覆盖排行失败")
	}
	return res, nil
}
