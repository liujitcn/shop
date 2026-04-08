package biz

import (
	"context"
	"strconv"
	"strings"

	"github.com/go-kratos/kratos/v2/transport"
	auth "github.com/liujitcn/kratos-kit/auth"
)

const recommendAnonymousIdHeader = "X-Recommend-Anonymous-Id"

// RecommendActor 推荐主体。
type RecommendActor struct {
	ActorType int32
	ActorId   int64
	UserId    int64
}

// resolveRecommendActor 解析推荐主体，登录态优先覆盖匿名 ID。
func (c *RecommendCase) resolveRecommendActor(ctx context.Context) *RecommendActor {
	return resolveRecommendActor(ctx)
}

// resolveRecommendActor 解析推荐主体，登录态优先覆盖匿名 ID。
func resolveRecommendActor(ctx context.Context) *RecommendActor {
	authInfo, err := auth.FromContext(ctx)
	// 已登录场景统一使用用户主体，避免匿名主体影响实名画像。
	if err == nil && authInfo != nil && authInfo.UserId > 0 {
		return &RecommendActor{
			ActorType: recommendActorTypeUser,
			ActorId:   authInfo.UserId,
			UserId:    authInfo.UserId,
		}
	}

	anonymousId := extractRecommendAnonymousId(ctx)
	if anonymousId <= 0 {
		return &RecommendActor{ActorType: recommendActorTypeAnonymous}
	}
	return &RecommendActor{
		ActorType: recommendActorTypeAnonymous,
		ActorId:   anonymousId,
		UserId:    0,
	}
}

// extractRecommendAnonymousId 从请求上下文提取匿名 ID。
func extractRecommendAnonymousId(ctx context.Context) int64 {
	serverTransport, ok := transport.FromServerContext(ctx)
	if !ok {
		return 0
	}
	rawValue := strings.TrimSpace(serverTransport.RequestHeader().Get(recommendAnonymousIdHeader))
	if rawValue == "" {
		return 0
	}
	anonymousId, err := strconv.ParseInt(rawValue, 10, 64)
	if err != nil || anonymousId <= 0 {
		return 0
	}
	return anonymousId
}
