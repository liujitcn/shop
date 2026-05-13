package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	basev1 "shop/api/gen/go/base/v1"
	commonv1 "shop/api/gen/go/common/v1"
	"shop/pkg/agent/stream"
	_const "shop/pkg/const"
	"shop/pkg/errorsx"
	"shop/pkg/workspaceevent"

	kratosHTTP "github.com/go-kratos/kratos/v2/transport/http"
	authnEngine "github.com/liujitcn/kratos-kit/auth/authn/engine"
	authData "github.com/liujitcn/kratos-kit/auth/data"
	"github.com/liujitcn/kratos-kit/bootstrap"
	sseServer "github.com/liujitcn/kratos-kit/transport/sse"
	"google.golang.org/protobuf/types/known/emptypb"
)

// SseCase 处理 SSE 公共业务。
type SseCase struct {
	authenticator authnEngine.Authenticator
	sse           *sseServer.Server
}

// NewSseCase 创建 SSE 业务实例。
func NewSseCase(ctx *bootstrap.Context, authenticator authnEngine.Authenticator, sse *sseServer.Server) (*SseCase, error) {
	handler := &SseCase{
		authenticator: authenticator,
		sse:           sse,
	}
	cfg := ctx.GetConfig()
	// 未启用 HTTP 服务时，不创建 SSE HTTP 处理器。
	if cfg == nil || cfg.Server == nil || cfg.Server.Http == nil {
		return handler, nil
	}
	workspaceevent.SetPublisher(func(ctx context.Context, payload workspaceevent.RefreshPayload) error {
		var data []byte
		data, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("marshal workspace refresh payload: %w", err)
		}
		sse.Publish(ctx, sseServer.StreamID(workspaceevent.StreamID(workspaceevent.StreamAdminWorkspace)), &sseServer.Event{
			Event: []byte(workspaceevent.EventID(workspaceevent.EventWorkspaceRefresh)),
			Data:  data,
		})
		return nil
	})
	return handler, nil
}

// SubscribeSse 订阅 SSE 事件流。
func (h *SseCase) SubscribeSse(ctx context.Context, req *basev1.SubscribeSseRequest) (*emptypb.Empty, error) {
	if h == nil || h.sse == nil {
		return nil, errorsx.Internal("SSE服务未初始化")
	}
	switch req.GetStream() {
	// 当前支持管理后台工作台刷新流和 AI 助手流。
	case commonv1.SseStream_SSE_STREAM_ADMIN_WORKSPACE, commonv1.SseStream_SSE_STREAM_ADMIN_AI_ASSISTANT:
	default:
		return nil, errorsx.InvalidArgument("SSE流不支持")
	}
	w, ok := kratosHTTP.ResponseWriterFromServerContext(ctx)
	if !ok || w == nil {
		return nil, errorsx.InvalidArgument("SSE订阅仅支持HTTP访问")
	}
	var r *http.Request
	r, ok = kratosHTTP.RequestFromServerContext(ctx)
	if !ok || r == nil {
		return nil, errorsx.InvalidArgument("SSE订阅仅支持HTTP访问")
	}
	userToken, authorized := h.authorizeRequest(r)
	if !authorized {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return nil, nil
	}

	streamID := stream.ResolveAdminStreamID(req.GetStream(), userToken.UserId)
	if strings.TrimSpace(streamID) == "" {
		return nil, errorsx.InvalidArgument("SSE流不支持")
	}
	h.sse.ServeStreamHTTP(w, r, sseServer.StreamID(streamID))
	return &emptypb.Empty{}, nil
}

// authorizeRequest 判断当前 SSE 请求是否来自后台登录用户，并返回用户信息。
func (h *SseCase) authorizeRequest(r *http.Request) (*authData.UserTokenPayload, bool) {
	userToken, err := h.authenticatorFromRequest(r)
	if err != nil || userToken == nil || userToken.UserId == 0 {
		return nil, false
	}
	// 工作台只面向管理后台，商城端用户和游客令牌不能订阅后台刷新流。
	if userToken.RoleCode == _const.BASE_ROLE_CODE_USER || userToken.RoleCode == _const.BASE_ROLE_CODE_GUEST {
		return nil, false
	}
	return userToken, true
}

// authenticatorFromRequest 从 SSE 请求中解析并校验后台登录用户。
func (h *SseCase) authenticatorFromRequest(r *http.Request) (*authData.UserTokenPayload, error) {
	token := strings.TrimSpace(r.Header.Get(authnEngine.HeaderAuthorize))
	if token == "" {
		return nil, errorsx.Unauthenticated("SSE访问令牌为空")
	}
	if len(token) >= len(authnEngine.BearerWord)+1 && strings.EqualFold(token[:len(authnEngine.BearerWord)+1], authnEngine.BearerWord+" ") {
		token = strings.TrimSpace(token[len(authnEngine.BearerWord)+1:])
	}

	authClaims, err := h.authenticator.AuthenticateToken(token)
	if err != nil || authClaims == nil {
		return nil, errorsx.Unauthenticated("SSE访问令牌无效").WithCause(err)
	}
	var userToken *authData.UserTokenPayload
	userToken, err = authData.NewUserTokenPayloadWithClaims(authClaims)
	if err != nil || userToken == nil || userToken.UserId == 0 {
		return nil, errorsx.Unauthenticated("SSE访问令牌无效").WithCause(err)
	}
	return userToken, nil
}
