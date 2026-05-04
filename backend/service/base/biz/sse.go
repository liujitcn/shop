package biz

import (
	"context"

	basev1 "shop/api/gen/go/base/v1"
	commonv1 "shop/api/gen/go/common/v1"
	"shop/pkg/errorsx"

	"google.golang.org/protobuf/types/known/emptypb"
)

// SseCase 处理 SSE 公共业务。
type SseCase struct{}

// NewSseCase 创建 SSE 业务实例。
func NewSseCase() *SseCase {
	return &SseCase{}
}

// SubscribeSse 校验 SSE 订阅请求参数。
func (c *SseCase) SubscribeSse(ctx context.Context, req *basev1.SubscribeSseRequest) (*emptypb.Empty, error) {
	switch req.GetStream() {
	// 当前仅支持管理后台工作台刷新流。
	case commonv1.SseStream_SSE_STREAM_ADMIN:
		return &emptypb.Empty{}, nil
	default:
		return nil, errorsx.InvalidArgument("SSE流不支持")
	}
}
