package middleware

import (
	"context"

	bootstrapConf "github.com/liujitcn/kratos-kit/api/gen/go/conf"
	authnEngine "github.com/liujitcn/kratos-kit/auth/authn/engine"
	"github.com/liujitcn/kratos-kit/auth/authn/engine/jwt"
	authzEngine "github.com/liujitcn/kratos-kit/auth/authz/engine"
	authzEngineCasbin "github.com/liujitcn/kratos-kit/auth/authz/engine/casbin"
	authData "github.com/liujitcn/kratos-kit/auth/data"
	"github.com/liujitcn/kratos-kit/cache"
)

// NewAuthenticator 创建认证器
func NewAuthenticator(cfg *bootstrapConf.Authentication_Jwt) authnEngine.Authenticator {
	authenticator, _ := jwt.NewAuthenticator(
		jwt.WithKey([]byte(cfg.GetSecret())),
		jwt.WithSigningMethod(cfg.GetMethod()),
	)
	return authenticator
}

// NewAuthzEngine 创建鉴权引擎
func NewAuthzEngine() (authzEngine.Engine, error) {
	return authzEngineCasbin.NewEngine(context.Background())
}

func NewUserToken(cfg *bootstrapConf.Authentication_Jwt, cache cache.Cache, authenticator authnEngine.Authenticator) *authData.UserToken {
	const (
		userAccessTokenKeyPrefix  = "uat_"
		userRefreshTokenKeyPrefix = "urt_"
	)
	return authData.NewUserToken(
		cache,
		authenticator,
		userAccessTokenKeyPrefix,
		userRefreshTokenKeyPrefix,
		cfg.GetAccessTokenExpires().AsDuration(),
		cfg.GetRefreshTokenExpires().AsDuration(),
	)
}
