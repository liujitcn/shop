package biz

import (
	"context"

	"shop/pkg/biz"
	"shop/pkg/errorsx"

	basev1 "shop/api/gen/go/base/v1"
	commonv1 "shop/api/gen/go/common/v1"
	"shop/pkg/gen/models"
	"shop/pkg/utils"

	"github.com/liujitcn/go-utils/crypto"
	"github.com/liujitcn/kratos-kit/auth/authn/engine"
	authData "github.com/liujitcn/kratos-kit/auth/data"
	"github.com/liujitcn/kratos-kit/captcha"
)

type LoginCase struct {
	*biz.BaseCase
	userToken    *authData.UserToken
	baseDeptCase *BaseDeptCase
	baseRoleCase *BaseRoleCase
	baseUserCase *BaseUserCase
}

// NewLoginCase 创建登录业务实例。
func NewLoginCase(
	baseCase *biz.BaseCase,
	userToken *authData.UserToken,
	baseDeptRepo *BaseDeptCase,
	baseRoleRepo *BaseRoleCase,
	baseUserRepo *BaseUserCase,
) *LoginCase {
	return &LoginCase{
		BaseCase:     baseCase,
		userToken:    userToken,
		baseDeptCase: baseDeptRepo,
		baseRoleCase: baseRoleRepo,
		baseUserCase: baseUserRepo,
	}
}

// Captcha 生成验证码。
func (c *LoginCase) Captcha(ctx context.Context, req *basev1.CaptchaRequest) (*basev1.CaptchaResponse, error) {
	id, b64s, _, err := captcha.DriverDigitFunc()
	if err != nil {
		return nil, err
	}
	return &basev1.CaptchaResponse{
		CaptchaId:     id,
		CaptchaBase64: b64s,
	}, err
}

// PasswordPublicKey 生成密码加密临时公钥。
func (c *LoginCase) PasswordPublicKey(ctx context.Context, req *basev1.PasswordPublicKeyRequest) (*basev1.PasswordPublicKeyResponse, error) {
	return utils.GeneratePasswordPublicKey(req.GetScene())
}

// Logout 退出登录。
func (c *LoginCase) Logout(ctx context.Context, req *basev1.LogoutRequest) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	err = c.userToken.RemoveToken(authInfo.UserId)
	if err != nil {
		return errorsx.Internal("退出登录失败").WithCause(err)
	}
	return nil
}

// RefreshToken 刷新认证令牌。
func (c *LoginCase) RefreshToken(ctx context.Context, req *basev1.RefreshTokenRequest) (*basev1.RefreshTokenResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}

	// 校验刷新令牌
	refreshToken := c.userToken.GetRefreshToken(authInfo.UserId)
	// 客户端刷新令牌与缓存不一致时，拒绝刷新访问令牌。
	if refreshToken != req.GetRefreshToken() {
		return nil, errorsx.Unauthenticated("刷新认证令牌失败")
	}

	// 生成新的访问令牌
	var accessToken string
	accessToken, err = c.userToken.GenerateAccessToken(authInfo)
	if err != nil {
		return nil, errorsx.Internal("刷新认证令牌失败").WithCause(err)
	}
	// Token 有效期
	expiresIn := c.userToken.GetAccessTokenExpires()

	return &basev1.RefreshTokenResponse{
		TokenType:    engine.BearerWord,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
	}, nil
}

// Login 执行登录。
func (c *LoginCase) Login(ctx context.Context, req *basev1.LoginRequest) (*basev1.LoginResponse, error) {
	// 验证码标识或验证码缺失时，不允许继续登录。
	if req.GetCaptchaId() == "" || req.GetCaptchaCode() == "" {
		return nil, errorsx.InvalidArgument("验证码不能为空")
	}
	// 验证码校验失败时，直接拒绝登录请求。
	if !captcha.Verify(req.GetCaptchaId(), req.GetCaptchaCode(), true) {
		return nil, errorsx.InvalidArgument("验证码错误")
	}

	user, err := c.baseUserCase.FindByUserName(ctx, req.GetUserName())
	if err != nil {
		return nil, errorsx.Unauthenticated("用户名或密码错误")
	}
	// 用户被停用时，不允许签发新的登录令牌。
	if user.Status != 1 {
		return nil, errorsx.PermissionDenied("账号已被禁用")
	}
	var password string
	password, err = utils.DecryptPassword(req.GetPassword(), commonv1.PasswordCryptoScene_LOGIN)
	if err != nil {
		return nil, errorsx.Unauthenticated("用户名或密码错误").WithCause(err)
	}
	err = crypto.Verify(password, user.Password)
	if err != nil {
		return nil, errorsx.Unauthenticated("用户名或密码错误")
	}

	// 查询角色信息
	var role *models.BaseRole
	role, err = c.baseRoleCase.FindByID(ctx, user.RoleID)
	if err != nil {
		return nil, errorsx.Internal("登录失败").WithCause(err)
	}

	// 查询部门信息
	var dept *models.BaseDept
	dept, err = c.baseDeptCase.FindByID(ctx, user.DeptID)
	if err != nil {
		return nil, errorsx.Internal("登录失败").WithCause(err)
	}

	// 生成访问令牌
	var accessToken, refreshToken string
	accessToken, refreshToken, err = c.userToken.GenerateToken(&authData.UserTokenPayload{
		UserId:   user.ID,
		UserName: user.UserName,
		RoleId:   user.RoleID,
		RoleCode: role.Code,
		RoleName: role.Name,
		DeptId:   user.DeptID,
		DeptName: dept.Name,
	})
	if err != nil {
		return nil, errorsx.Internal("登录失败").WithCause(err)
	}

	// Token 有效期
	expiresIn := c.userToken.GetAccessTokenExpires()

	return &basev1.LoginResponse{
		TokenType:    engine.BearerWord,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
	}, nil
}
