package biz

import (
	"context"
	"errors"
	"fmt"
	"time"

	_const "shop/pkg/const"

	"shop/pkg/biz"
	"shop/pkg/errorsx"
	"shop/pkg/gen/models"
	"shop/pkg/queue"

	appv1 "shop/api/gen/go/app/v1"
	configv1 "shop/api/gen/go/config/v1"
	"shop/service/app/utils"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/liujitcn/go-utils/id"
	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/kratos-kit/auth/authn/engine"
	authData "github.com/liujitcn/kratos-kit/auth/data"
	"github.com/liujitcn/kratos-kit/sdk"
	"gorm.io/gorm"
)

const CACHE_KEY_WX_ACCESS_TOKEN = "wx_access_token"

// AuthCase 登录认证业务处理对象
type AuthCase struct {
	*biz.BaseCase
	userToken     *authData.UserToken
	baseUserCase  *BaseUserCase
	baseRoleCase  *BaseRoleCase
	baseDeptCase  *BaseDeptCase
	wxMiniApp     *configv1.WxMiniApp
	profileMapper *mapper.CopierMapper[
		appv1.UserProfileForm,
		models.BaseUser,
	]
}

// NewAuthCase 创建登录认证业务处理对象
func NewAuthCase(
	baseCase *biz.BaseCase,
	userToken *authData.UserToken,
	baseUserCase *BaseUserCase,
	baseRoleCase *BaseRoleCase,
	baseDeptCase *BaseDeptCase,
	wxMiniApp *configv1.WxMiniApp,
) *AuthCase {
	return &AuthCase{
		BaseCase:     baseCase,
		userToken:    userToken,
		baseUserCase: baseUserCase,
		baseRoleCase: baseRoleCase,
		baseDeptCase: baseDeptCase,
		wxMiniApp:    wxMiniApp,
		profileMapper: mapper.NewCopierMapper[
			appv1.UserProfileForm,
			models.BaseUser,
		](),
	}
}

// WechatLogin 微信登录
func (c *AuthCase) WechatLogin(ctx context.Context, req *appv1.WechatLoginRequest) (*appv1.WechatLoginResponse, error) {
	sessionKey, err := utils.GetSessionKey(c.wxMiniApp.GetAppid(), c.wxMiniApp.GetSecret(), req.GetCode())
	if err != nil {
		return nil, errorsx.Internal("登录失败").WithCause(err)
	}
	// 微信侧返回业务错误时，直接透传错误信息。
	if sessionKey.ErrCode != 0 {
		return nil, errorsx.Internal("登录失败").WithMetadata(map[string]string{
			"provider":      "wechat",
			"provider_code": fmt.Sprintf("%d", sessionKey.ErrCode),
		})
	}
	// 未返回 OpenID 时，当前登录请求无效。
	if sessionKey.OpenID == "" {
		return nil, errorsx.Internal("登录失败")
	}

	var user *models.BaseUser
	user, err = c.baseUserCase.findByOpenID(ctx, sessionKey.OpenID)
	// 按 OpenID 查询用户失败时，仅对“未注册”场景继续自动注册。
	if err != nil {
		// 非“未注册”错误说明查询本身异常，直接返回登录失败。
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorsx.Internal("登录失败").WithCause(err)
		}

		// 用户不存在时自动注册一个小程序账号
		user = &models.BaseUser{
			Openid:   sessionKey.OpenID,
			UserName: id.NewXID(),
			RoleID:   4,
			DeptID:   5,
			Phone:    "",
			Password: "",
			Gender:   3,
			Avatar:   "",
			Status:   _const.STATUS_ENABLE,
			Remark:   "自动注册用户",
		}
		// 自动注册用户失败时，直接返回登录失败。
		if err = c.baseUserCase.Create(ctx, user); err != nil {
			return nil, errorsx.Internal("登录失败").WithCause(err)
		}
	}
	// 用户被停用时，不允许继续登录。
	if user.Status != _const.STATUS_ENABLE {
		return nil, errorsx.PermissionDenied("账号已被禁用")
	}
	// 登录成功前，异步同步用户画像到推荐系统，保证推荐主体随登录链路逐步补齐。
	queue.DispatchRecommendSyncBaseUser(user.ID)

	// 登录凭证需要补齐角色和部门信息
	var role *models.BaseRole
	role, err = c.baseRoleCase.FindByID(ctx, user.RoleID)
	if err != nil {
		return nil, errorsx.Internal("登录失败").WithCause(err)
	}
	var dept *models.BaseDept
	dept, err = c.baseDeptCase.FindByID(ctx, user.DeptID)
	if err != nil {
		return nil, errorsx.Internal("登录失败").WithCause(err)
	}

	var accessToken, refreshToken string
	accessToken, refreshToken, err = c.userToken.GenerateToken(&authData.UserTokenPayload{
		UserId:   user.ID,
		UserName: user.UserName,
		RoleId:   user.RoleID,
		RoleCode: role.Code,
		RoleName: role.Name,
		DeptId:   dept.ID,
		DeptName: dept.Name,
		OpenId:   user.Openid,
	})
	if err != nil {
		return nil, errorsx.Internal("登录失败").WithCause(err)
	}

	return &appv1.WechatLoginResponse{
		TokenType:    engine.BearerWord,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    c.userToken.GetAccessTokenExpires(),
	}, nil
}

// GetUserProfile 获取当前登录用户信息
func (c *AuthCase) GetUserProfile(ctx context.Context) (*appv1.UserProfileForm, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}

	var user *models.BaseUser
	user, err = c.baseUserCase.FindByID(ctx, authInfo.UserId)
	if err != nil {
		return nil, errorsx.ResourceNotFound("用户不存在").WithCause(err)
	}
	// 用户被停用时，不允许继续获取个人信息。
	if user.Status != _const.STATUS_ENABLE {
		return nil, errorsx.PermissionDenied("账号已被禁用")
	}

	res := c.profileMapper.ToDTO(user)
	res.Phone = _string.DesensitizePhone(user.Phone)
	return res, nil
}

// UpdateUserProfile 修改个人中心用户信息
func (c *AuthCase) UpdateUserProfile(ctx context.Context, req *appv1.UserProfileForm) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}

	var oldBaseUser *models.BaseUser
	oldBaseUser, err = c.baseUserCase.FindByID(ctx, authInfo.UserId)
	if err != nil {
		return errorsx.ResourceNotFound("用户不存在").WithCause(err)
	}
	originalAvatar := oldBaseUser.Avatar

	baseUser := &models.BaseUser{
		ID:       authInfo.UserId,
		NickName: req.GetNickName(),
		Gender:   req.GetGender(),
		Avatar:   req.GetAvatar(),
	}
	// 用户资料更新失败时，直接返回错误交由上层处理。
	if err = c.baseUserCase.UpdateByID(ctx, baseUser); err != nil {
		return errorsx.Internal("修改个人中心用户信息失败").WithCause(err)
	}
	// 用户资料写库成功后，再异步同步最新画像到推荐系统。
	queue.DispatchRecommendSyncBaseUser(authInfo.UserId)

	// 删除被替换的旧头像文件
	oss := sdk.Runtime.GetOSS()
	// OSS 可用时，尝试清理被替换掉的历史头像文件。
	if oss != nil {
		// 新头像为空或发生变更时，旧头像文件需要尝试删除。
		if baseUser.Avatar == "" || originalAvatar != baseUser.Avatar {
			// 头像文件删除失败时，只记录日志不影响主流程。
			if err = oss.DeleteFile(originalAvatar); err != nil {
				log.Errorf("DeleteFile %v", err)
			}
		}
	}
	return nil
}

// BindUserPhone 手机号授权
func (c *AuthCase) BindUserPhone(ctx context.Context, req *appv1.BindUserPhoneRequest) (*appv1.BindUserPhoneResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}

	var accessToken string
	accessToken, err = sdk.Runtime.GetCache().Get(CACHE_KEY_WX_ACCESS_TOKEN)
	// 本地缓存未命中 access token 时，回源微信重新获取。
	if err != nil {
		var token *utils.WxAccessToken
		token, err = utils.GetAccessToken(c.wxMiniApp.GetAppid(), c.wxMiniApp.GetSecret())
		// 微信 access token 获取失败时，直接返回授权失败。
		if err != nil {
			return nil, errorsx.Internal("手机号授权失败").WithCause(err)
		}
		// 微信侧返回 access token 业务错误时，直接返回授权失败。
		if token.ErrCode != 0 {
			return nil, errorsx.Internal("手机号授权失败").WithMetadata(map[string]string{
				"provider":      "wechat",
				"provider_code": fmt.Sprintf("%d", token.ErrCode),
			})
		}
		accessToken = token.AccessToken
		// 新 access token 缓存失败时，只记录日志不影响主流程。
		err = sdk.Runtime.GetCache().Set(CACHE_KEY_WX_ACCESS_TOKEN, accessToken, time.Duration(token.ExpiresIn-300))
		if err != nil {
			log.Errorf("SetWxAccessTokenCache %v", err)
		}
	}

	var phone *utils.PhoneNumber
	phone, err = utils.GetPhoneNumber(accessToken, req.GetCode())
	if err != nil {
		return nil, errorsx.Internal("手机号授权失败").WithCause(err)
	}
	// 微信侧返回手机号授权错误时，直接返回授权失败。
	if phone.ErrCode != 0 {
		return nil, errorsx.Internal("手机号授权失败").WithMetadata(map[string]string{
			"provider":      "wechat",
			"provider_code": fmt.Sprintf("%d", phone.ErrCode),
		})
	}

	var find *models.BaseUser
	find, err = c.baseUserCase.findByPhone(ctx, phone.PhoneInfo.PhoneNumber)
	// 手机号占用查询异常时，直接返回授权失败。
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errorsx.Internal("手机号授权失败").WithCause(err)
	}
	// 手机号已绑定其他账号时，不允许继续授权。
	if find != nil && find.ID != authInfo.UserId {
		return nil, errorsx.Conflict("手机号已被占用").WithMetadata(map[string]string{
			errorsx.METADATA_KEY_CONFLICT_TYPE: errorsx.CONFLICT_TYPE_UNIQUE_VIOLATION,
			errorsx.METADATA_KEY_RESOURCE:      "base_user",
			errorsx.METADATA_KEY_FIELD:         "phone",
		})
	}

	user := &models.BaseUser{
		ID:    authInfo.UserId,
		Phone: phone.PhoneInfo.PhoneNumber,
	}
	// 绑定手机号写库失败时，直接返回业务错误。
	if err = c.baseUserCase.UpdateByID(ctx, user); err != nil {
		return nil, errorsx.Internal("手机号授权失败").WithCause(err)
	}
	// 手机号绑定成功后，再异步同步最新用户画像到推荐系统。
	queue.DispatchRecommendSyncBaseUser(authInfo.UserId)

	return &appv1.BindUserPhoneResponse{
		Phone: _string.DesensitizePhone(user.Phone),
	}, nil
}
