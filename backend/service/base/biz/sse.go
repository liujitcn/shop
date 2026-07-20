package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	basev1 "shop/api/gen/go/base/v1"
	commonv1 "shop/api/gen/go/common/v1"
	systemadminv1 "shop/api/gen/go/system/admin/v1"
	_const "shop/pkg/const"
	"shop/pkg/errorsx"
	"shop/service/shop/workspaceevent"
	"shop/service/system/admin/codegen"

	kratosHTTP "github.com/go-kratos/kratos/v3/transport/http"
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
	progress      *codegen.Manager
}

// NewSseCase 创建 SSE 业务实例。
func NewSseCase(ctx *bootstrap.Context, authenticator authnEngine.Authenticator, sse *sseServer.Server, progress *codegen.Manager) (*SseCase, error) {
	handler := &SseCase{
		authenticator: authenticator,
		sse:           sse,
		progress:      progress,
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
	progress.SetPublisher(func(ctx context.Context, taskID string, task *systemadminv1.CodeGenTask) {
		data, err := json.Marshal(task)
		if err != nil {
			return
		}
		sse.TryPublish(ctx, sseServer.StreamID(codegen.StreamID(taskID)), &sseServer.Event{
			Event: []byte(workspaceevent.EventID(commonv1.SseEvent_SSE_EVENT_CODE_GEN_PROGRESS)),
			Data:  data,
		})
	})
	return handler, nil
}

// SubscribeSse 订阅 SSE 事件流。
func (h *SseCase) SubscribeSse(ctx context.Context, req *basev1.SubscribeSseRequest) (*emptypb.Empty, error) {
	if h == nil || h.sse == nil {
		return nil, errorsx.Internal("SSE服务未初始化")
	}
	if req.GetStream() != commonv1.SseStream_SSE_STREAM_ADMIN_WORKSPACE && req.GetStream() != commonv1.SseStream_SSE_STREAM_ADMIN_CODE_GEN {
		return nil, errorsx.InvalidArgument("SSE流不支持")
	}
	if req.GetStream() == commonv1.SseStream_SSE_STREAM_ADMIN_CODE_GEN && req.GetChannelId() == "" {
		return nil, errorsx.InvalidArgument("代码生成任务ID不能为空")
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

	streamID := workspaceevent.StreamID(req.GetStream())
	// 代码生成进度只允许任务创建者订阅，并按任务ID隔离事件流。
	if req.GetStream() == commonv1.SseStream_SSE_STREAM_ADMIN_CODE_GEN {
		if !h.progress.IsOwner(req.GetChannelId(), userToken.UserId) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			http.Error(w, "forbidden", http.StatusForbidden)
			return nil, nil
		}
		streamID = codegen.StreamID(req.GetChannelId())
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
	token := r.Header.Get(authnEngine.HeaderAuthorize)
	if token == "" {
		return nil, errorsx.Unauthenticated("SSE访问令牌为空")
	}
	if len(token) >= len(authnEngine.BearerWord)+1 && strings.EqualFold(token[:len(authnEngine.BearerWord)+1], authnEngine.BearerWord+" ") {
		token = token[len(authnEngine.BearerWord)+1:]
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
