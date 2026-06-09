package biz

import (
	"context"
	"fmt"
	"time"

	"shop/pkg/biz"
	"shop/pkg/errorsx"

	basev1 "shop/api/gen/go/base/v1"
	commonv1 "shop/api/gen/go/common/v1"
	"shop/pkg/gen/models"
	"shop/pkg/utils"

	"github.com/liujitcn/go-utils/crypto"
	"github.com/liujitcn/go-utils/id"
	"github.com/liujitcn/kratos-kit/auth/authn/engine"
	authData "github.com/liujitcn/kratos-kit/auth/data"
	"github.com/liujitcn/kratos-kit/captcha"
)

const loginCaptchaKeyPrefix = "login_captcha"
const loginCaptchaTokenKeyPrefix = "login_captcha_token"
const loginCaptchaTypeKeyPrefix = "login_captcha_type"
const loginCaptchaTokenExpire = 2 * time.Minute

// LoginCase 处理基础登录认证业务。
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
	driverType, ok := captchaDriverType(req.GetType())
	// 请求的验证码类型不在系统字典支持范围内时，直接拒绝生成。
	if !ok {
		return nil, errorsx.InvalidArgument("验证码类型不支持")
	}

	challenge, err := captcha.NewCaptcha(c.Cache,
		captcha.WithDriverType(driverType),
		captcha.WithKeyPrefix(loginCaptchaKeyPrefix),
	).Generate(ctx)
	if err != nil {
		return nil, errorsx.Internal("生成验证码失败").WithCause(err)
	}
	err = c.Cache.Set(loginCaptchaTypeKey(challenge.ID), string(driverType), captcha.DefaultConfig().Expire)
	if err != nil {
		return nil, errorsx.Internal("验证码类型保存失败").WithCause(err)
	}
	return &basev1.CaptchaResponse{
		CaptchaId:     challenge.ID,
		CaptchaBase64: challenge.Payload,
	}, nil
}

// VerifyCaptcha 预校验验证码并签发一次性登录令牌。
func (c *LoginCase) VerifyCaptcha(ctx context.Context, req *basev1.VerifyCaptchaRequest) (*basev1.VerifyCaptchaResponse, error) {
	// 验证码标识或答案缺失时，不允许继续签发登录令牌。
	if req.GetCaptchaId() == "" || req.GetCaptchaCode() == "" {
		return nil, errorsx.InvalidArgument("验证码不能为空")
	}

	driverType, ok := c.captchaDriverTypeByID(req.GetCaptchaId())
	// 验证码类型缺失时，说明验证码已过期或不是当前系统签发。
	if !ok {
		return nil, errorsx.InvalidArgument("验证码错误")
	}
	matched, err := captcha.NewCaptcha(c.Cache,
		captcha.WithDriverType(driverType),
		captcha.WithKeyPrefix(loginCaptchaKeyPrefix),
	).Verify(ctx, req.GetCaptchaId(), req.GetCaptchaCode())
	if err != nil {
		return nil, errorsx.Internal("验证码校验失败").WithCause(err)
	}
	// 验证码校验失败时，不签发可用于登录的一次性令牌。
	if !matched {
		return nil, errorsx.InvalidArgument("验证码错误")
	}

	token := id.NewGUIDv4NoHyphen()
	err = c.Cache.Set(loginCaptchaTokenKey(req.GetCaptchaId(), token), req.GetCaptchaId(), loginCaptchaTokenExpire)
	if err != nil {
		return nil, errorsx.Internal("验证码令牌保存失败").WithCause(err)
	}
	return &basev1.VerifyCaptchaResponse{
		CaptchaToken: token,
		ExpiresIn:    int64(loginCaptchaTokenExpire / time.Second),
	}, nil
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

	err := c.verifyLoginCaptcha(ctx, req.GetCaptchaId(), req.GetCaptchaCode())
	if err != nil {
		return nil, err
	}

	var user *models.BaseUser
	user, err = c.baseUserCase.FindByUserName(ctx, req.GetUserName())
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

// captchaDriverType 根据配置值解析验证码驱动类型。
func captchaDriverType(captchaType string) (captcha.DriverType, bool) {
	// 兼容未配置验证码类型的历史场景，默认继续使用数字验证码。
	switch captchaType {
	case "", string(captcha.DriverDigit):
		return captcha.DriverDigit, true
	case string(captcha.DriverString):
		return captcha.DriverString, true
	case string(captcha.DriverMath):
		return captcha.DriverMath, true
	case string(captcha.DriverChinese):
		return captcha.DriverChinese, true
	case string(captcha.DriverSlide):
		return captcha.DriverSlide, true
	case string(captcha.DriverClick):
		return captcha.DriverClick, true
	case string(captcha.DriverRotate):
		return captcha.DriverRotate, true
	default:
		return captcha.DriverDigit, false
	}
}

// verifyLoginCaptcha 校验登录请求携带的验证码或行为验证码令牌。
func (c *LoginCase) verifyLoginCaptcha(ctx context.Context, captchaID, captchaCode string) error {
	driverType, ok := c.captchaDriverTypeByID(captchaID)
	// 登录阶段先通过 captcha_id 取回类型，避免依赖验证码 ID 命名格式。
	if !ok {
		return errorsx.InvalidArgument("验证码错误")
	}
	matched, err := captcha.NewCaptcha(c.Cache,
		captcha.WithDriverType(driverType),
		captcha.WithKeyPrefix(loginCaptchaKeyPrefix),
	).Verify(ctx, captchaID, captchaCode)
	if err != nil {
		return errorsx.Internal("验证码校验失败").WithCause(err)
	}
	if matched {
		return nil
	}
	// 预校验会删除原始答案并签发 token，原始 code 未命中时再兼容 token 登录。
	consumed, err := c.consumeLoginCaptchaToken(captchaID, captchaCode)
	if err != nil {
		return err
	}
	if consumed {
		return nil
	}
	return errorsx.InvalidArgument("验证码错误")
}

// consumeLoginCaptchaToken 校验并消费验证码预校验签发的一次性令牌。
func (c *LoginCase) consumeLoginCaptchaToken(captchaID, token string) (bool, error) {
	key := loginCaptchaTokenKey(captchaID, token)
	value, err := c.Cache.Get(key)
	if err != nil || value != captchaID {
		return false, nil
	}
	err = c.Cache.Del(key)
	if err != nil {
		return false, errorsx.Internal("验证码令牌消费失败").WithCause(err)
	}
	return true, nil
}

// loginCaptchaTokenKey 生成验证码预校验令牌缓存键。
func loginCaptchaTokenKey(captchaID, token string) string {
	return fmt.Sprintf("%s:%s:%s", loginCaptchaTokenKeyPrefix, captchaID, token)
}

// loginCaptchaTypeKey 生成验证码类型缓存键。
func loginCaptchaTypeKey(captchaID string) string {
	return fmt.Sprintf("%s:%s", loginCaptchaTypeKeyPrefix, captchaID)
}

// captchaDriverTypeByID 根据验证码 ID 查询生成时保存的校验驱动类型。
func (c *LoginCase) captchaDriverTypeByID(captchaID string) (captcha.DriverType, bool) {
	captchaType, err := c.Cache.Get(loginCaptchaTypeKey(captchaID))
	if err != nil {
		return captcha.DriverDigit, false
	}
	driverType, ok := captchaDriverType(captchaType)
	// 验证码 ID 不承载类型语义，需要回读生成时保存的类型。
	return driverType, ok
}
