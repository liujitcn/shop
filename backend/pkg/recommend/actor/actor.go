package actor

import (
	"context"
	"strconv"
	"strings"

	recommendevent "shop/pkg/recommend/event"
	appdto "shop/service/app/dto"

	"github.com/go-kratos/kratos/v2/transport"
	auth "github.com/liujitcn/kratos-kit/auth"
)

const AnonymousIDHeader = "X-Recommend-Anonymous-Id"

// Resolve 解析推荐主体，登录态优先覆盖匿名主体。
func Resolve(ctx context.Context) *appdto.RecommendActor {
	authInfo, err := auth.FromContext(ctx)
	if err == nil && authInfo != nil && authInfo.UserId > 0 {
		return &appdto.RecommendActor{
			ActorType: recommendevent.ActorTypeUser,
			ActorId:   authInfo.UserId,
			UserId:    authInfo.UserId,
		}
	}

	anonymousID := ExtractAnonymousID(ctx)
	if anonymousID <= 0 {
		return &appdto.RecommendActor{ActorType: recommendevent.ActorTypeAnonymous}
	}
	return &appdto.RecommendActor{
		ActorType: recommendevent.ActorTypeAnonymous,
		ActorId:   anonymousID,
	}
}

// ExtractAnonymousID 从请求头提取匿名推荐主体 ID。
func ExtractAnonymousID(ctx context.Context) int64 {
	serverTransport, ok := transport.FromServerContext(ctx)
	if !ok {
		return 0
	}
	rawValue := strings.TrimSpace(serverTransport.RequestHeader().Get(AnonymousIDHeader))
	if rawValue == "" {
		return 0
	}
	anonymousID, err := strconv.ParseInt(rawValue, 10, 64)
	if err != nil || anonymousID <= 0 {
		return 0
	}
	return anonymousID
}
