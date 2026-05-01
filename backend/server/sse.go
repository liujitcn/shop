package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	shopConfig "shop/pkg/config"
	_const "shop/pkg/const"
	shopMiddleware "shop/pkg/middleware"
	"shop/pkg/workspaceevent"

	kratosHTTP "github.com/go-kratos/kratos/v2/transport/http"

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

// registerSseHTTPServer 将 SSE 工作台刷新流挂载到当前 HTTP 服务。
func registerSseHTTPServer(srv *kratosHTTP.Server, ctx *bootstrap.Context) error {
	cfg := ctx.GetConfig()
	if srv == nil || cfg == nil || cfg.Server == nil || cfg.Server.Http == nil {
		return nil
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

	sseSrv, err := rpc.CreateSseHandler(nil, options...)
	if err != nil {
		return err
	}
	streamID := sseServer.StreamID(workspaceevent.StreamID(workspaceevent.StreamAdmin))
	sseSrv.CreateStream(streamID)

	authenticator := shopMiddleware.NewAuthenticator(shopConfig.ParseAuthnJWT(ctx))
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isAuthorizedSseRequest(r, authenticator) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		sseSrv.ServeHTTP(w, r)
	})
	srv.Route("/").GET(defaultSsePath, func(ctx kratosHTTP.Context) error {
		handler.ServeHTTP(ctx.Response(), ctx.Request())
		return nil
	})
	registerWorkspacePublisher(sseSrv)
	return nil
}

// isAuthorizedSseRequest 判断当前 SSE 请求是否来自后台登录用户。
func isAuthorizedSseRequest(r *http.Request, authenticator authnEngine.Authenticator) bool {
	if r == nil || authenticator == nil {
		return false
	}

	token := extractSseAccessToken(r)
	if token == "" {
		return false
	}

	authClaims, err := authenticator.AuthenticateToken(token)
	if err != nil || authClaims == nil {
		return false
	}
	userToken, err := authData.NewUserTokenPayloadWithClaims(authClaims)
	if err != nil || userToken == nil || userToken.UserId == 0 {
		return false
	}
	// 工作台只面向管理后台，商城端用户和游客令牌不能订阅后台刷新流。
	if userToken.RoleCode == _const.BASE_ROLE_CODE_USER || userToken.RoleCode == _const.BASE_ROLE_CODE_GUEST {
		return false
	}
	return true
}

// extractSseAccessToken 从查询参数或请求头中提取访问令牌。
func extractSseAccessToken(r *http.Request) string {
	token := r.URL.Query().Get(sseAccessTokenQuery)
	if token == "" {
		token = r.URL.Query().Get(sseTokenQuery)
	}
	if token == "" {
		token = r.Header.Get(authnEngine.HeaderAuthorize)
	}
	return trimBearerToken(token)
}

// trimBearerToken 去除 Bearer 前缀并返回原始 JWT。
func trimBearerToken(token string) string {
	token = strings.TrimSpace(token)
	if len(token) >= len(authnEngine.BearerWord)+1 && strings.EqualFold(token[:len(authnEngine.BearerWord)+1], authnEngine.BearerWord+" ") {
		return strings.TrimSpace(token[len(authnEngine.BearerWord)+1:])
	}
	return token
}

// registerWorkspacePublisher 注册工作台 SSE 刷新消息发布器。
func registerWorkspacePublisher(sseSrv *sseServer.Server) {
	workspaceevent.SetPublisher(func(ctx context.Context, payload workspaceevent.RefreshPayload) error {
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
}
