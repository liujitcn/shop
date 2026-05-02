package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	_const "shop/pkg/const"
	"shop/pkg/workspaceevent"

	authnEngine "github.com/liujitcn/kratos-kit/auth/authn/engine"
	authData "github.com/liujitcn/kratos-kit/auth/data"
	"github.com/liujitcn/kratos-kit/bootstrap"
	"github.com/liujitcn/kratos-kit/rpc"
	sseServer "github.com/liujitcn/kratos-kit/transport/sse"
)

const (
	sseAccessTokenQuery = "access_token"
	sseTokenQuery       = "token"
	defaultSsePath      = "/events"
	defaultSseCodec     = "json"
	defaultSseEventTTL  = 300 * time.Second
)

// SseHTTPHandler 包装 SSE 标准 HTTP 处理器，便于注入到统一 HTTP 服务。
type SseHTTPHandler struct {
	http.Handler

	authenticator authnEngine.Authenticator
}

// NewSseHTTPHandler 创建 SSE 工作台刷新流 HTTP 处理器。
func NewSseHTTPHandler(ctx *bootstrap.Context, authenticator authnEngine.Authenticator) (*SseHTTPHandler, error) {
	cfg := ctx.GetConfig()
	// 未启用 HTTP 服务时，不创建 SSE HTTP 处理器。
	if cfg == nil || cfg.Server == nil || cfg.Server.Http == nil {
		return nil, nil
	}
	options := []sseServer.ServerOption{
		sseServer.WithPath(defaultSsePath),
		sseServer.WithCodec(defaultSseCodec),
		sseServer.WithEventTTL(defaultSseEventTTL),
		sseServer.WithAutoStream(true),
		sseServer.WithAutoReply(true),
	}
	if cfg.Server.Http.Timeout != nil {
		options = append(options, sseServer.WithTimeout(cfg.Server.Http.Timeout.AsDuration()))
	}

	sseSrv, err := rpc.CreateSseHandler(cfg, options...)
	if err != nil {
		return nil, err
	}
	streamID := sseServer.StreamID(workspaceevent.StreamID(workspaceevent.StreamAdmin))
	sseSrv.CreateStream(streamID)

	handler := &SseHTTPHandler{
		authenticator: authenticator,
	}
	handler.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !handler.isAuthorizedRequest(r) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		sseSrv.ServeHTTP(w, r)
	})
	workspaceevent.SetPublisher(func(ctx context.Context, payload workspaceevent.RefreshPayload) error {
		var data []byte
		data, err = json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("marshal workspace refresh payload: %w", err)
		}
		sseSrv.Publish(ctx, sseServer.StreamID(workspaceevent.StreamID(workspaceevent.StreamAdmin)), &sseServer.Event{
			Event: []byte(workspaceevent.EventID(workspaceevent.EventWorkspaceRefresh)),
			Data:  data,
		})
		return nil
	})
	return handler, nil
}

// isAuthorizedRequest 判断当前 SSE 请求是否来自后台登录用户。
func (h *SseHTTPHandler) isAuthorizedRequest(r *http.Request) bool {
	token := strings.TrimSpace(r.URL.Query().Get(sseAccessTokenQuery))
	if token == "" {
		token = strings.TrimSpace(r.URL.Query().Get(sseTokenQuery))
	}
	if token == "" {
		token = strings.TrimSpace(r.Header.Get(authnEngine.HeaderAuthorize))
	}
	if token == "" {
		return false
	}
	if len(token) >= len(authnEngine.BearerWord)+1 && strings.EqualFold(token[:len(authnEngine.BearerWord)+1], authnEngine.BearerWord+" ") {
		token = strings.TrimSpace(token[len(authnEngine.BearerWord)+1:])
	}

	authClaims, err := h.authenticator.AuthenticateToken(token)
	if err != nil || authClaims == nil {
		return false
	}
	var userToken *authData.UserTokenPayload
	userToken, err = authData.NewUserTokenPayloadWithClaims(authClaims)
	if err != nil || userToken == nil || userToken.UserId == 0 {
		return false
	}
	// 工作台只面向管理后台，商城端用户和游客令牌不能订阅后台刷新流。
	if userToken.RoleCode == _const.BASE_ROLE_CODE_USER || userToken.RoleCode == _const.BASE_ROLE_CODE_GUEST {
		return false
	}
	return true
}
