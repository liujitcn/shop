package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	basev1 "shop/api/gen/go/base/v1"
	commonv1 "shop/api/gen/go/common/v1"
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

const (
	sseAccessTokenQuery = "access_token"
	sseTokenQuery       = "token"
)

// SseCase 处理 SSE 公共业务。
type SseCase struct {
	http.Handler

	authenticator authnEngine.Authenticator
	path          string
}

// NewSseCase 创建 SSE 业务实例。
func NewSseCase(ctx *bootstrap.Context, authenticator authnEngine.Authenticator, sseSrv *sseServer.Server) (*SseCase, error) {
	handler := &SseCase{
		authenticator: authenticator,
		path:          "/events",
	}
	cfg := ctx.GetConfig()
	// 未启用 HTTP 服务时，不创建 SSE HTTP 处理器。
	if cfg == nil || cfg.Server == nil || cfg.Server.Http == nil {
		return handler, nil
	}
	if cfg.Server.Sse != nil && cfg.Server.Sse.GetPath() != "" {
		handler.path = cfg.Server.Sse.GetPath()
	}
	streamID := sseServer.StreamID(workspaceevent.StreamID(workspaceevent.StreamAdmin))
	sseSrv.CreateStream(streamID)

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
		data, err := json.Marshal(payload)
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

// SubscribeSse 订阅 SSE 事件流。
func (h *SseCase) SubscribeSse(ctx context.Context, req *basev1.SubscribeSseRequest) (*emptypb.Empty, error) {
	if h == nil || h.Handler == nil {
		return nil, errorsx.Internal("SSE服务未初始化")
	}
	switch req.GetStream() {
	// 当前仅支持管理后台工作台刷新流。
	case commonv1.SseStream_SSE_STREAM_ADMIN:
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
	h.Handler.ServeHTTP(w, h.normalizeSubscribeRequest(r, req))
	return &emptypb.Empty{}, nil
}

// normalizeSubscribeRequest 将路径参数中的流标识转换为 SSE 处理器使用的查询参数。
func (h *SseCase) normalizeSubscribeRequest(r *http.Request, req *basev1.SubscribeSseRequest) *http.Request {
	if r.URL.Query().Get("stream") != "" {
		return r
	}
	clonedRequest := r.Clone(r.Context())
	urlCopy := *r.URL
	urlCopy.Path = h.path
	urlCopy.RawPath = ""
	query := urlCopy.Query()
	if query.Get("stream") == "" {
		query.Set("stream", strconv.FormatInt(int64(req.GetStream()), 10))
	}
	urlCopy.RawQuery = query.Encode()
	clonedRequest.URL = &urlCopy
	return clonedRequest
}

// isAuthorizedRequest 判断当前 SSE 请求是否来自后台登录用户。
func (h *SseCase) isAuthorizedRequest(r *http.Request) bool {
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
