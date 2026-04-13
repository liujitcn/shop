package actor

import (
	"context"
	"strconv"
	"strings"

	recommendevent "shop/pkg/recommend/event"
	appDto "shop/service/app/dto"

	"github.com/go-kratos/kratos/v2/transport"
	"github.com/liujitcn/kratos-kit/auth"
)

const AnonymousIdHeader = "X-Recommend-Anonymous-Id"

// Resolve 解析推荐主体，登录态优先覆盖匿名主体。
func Resolve(ctx context.Context) *appDto.RecommendActor {
	authInfo, err := auth.FromContext(ctx)
	// 已登录用户优先使用登录态用户标识作为推荐主体。
	if err == nil && authInfo != nil && authInfo.UserId > 0 {
		return &appDto.RecommendActor{
			ActorType: recommendevent.ActorTypeUser,
			ActorId:   authInfo.UserId,
		}
	}

	anonymousId := ExtractAnonymousId(ctx)
	// 未携带有效匿名主体时，仅返回匿名类型等待后续补充标识。
	if anonymousId <= 0 {
		return &appDto.RecommendActor{ActorType: recommendevent.ActorTypeAnonymous}
	}
	return &appDto.RecommendActor{
		ActorType: recommendevent.ActorTypeAnonymous,
		ActorId:   anonymousId,
	}
}

// ExtractAnonymousId 从请求头提取匿名推荐主体 Id。
func ExtractAnonymousId(ctx context.Context) int64 {
	serverTransport, ok := transport.FromServerContext(ctx)
	// 当前上下文不是服务端请求上下文时，无法提取请求头。
	if !ok {
		return 0
	}
	rawValue := strings.TrimSpace(serverTransport.RequestHeader().Get(AnonymousIdHeader))
	// 请求头为空时，视为未携带匿名主体标识。
	if rawValue == "" {
		return 0
	}
	anonymousId, err := strconv.ParseInt(rawValue, 10, 64)
	// 请求头不是正整数时，按无效匿名主体处理。
	if err != nil || anonymousId <= 0 {
		return 0
	}
	return anonymousId
}
