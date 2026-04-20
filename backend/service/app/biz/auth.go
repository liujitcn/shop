package biz

import (
	"context"
	"errors"
	"fmt"
	"shop/pkg/gorse"
	"time"

	"shop/pkg/biz"
	"shop/pkg/errorsx"
	"shop/pkg/gen/models"

	"shop/api/gen/go/app"
	"shop/api/gen/go/common"
	"shop/api/gen/go/conf"
	"shop/service/app/utils"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/liujitcn/go-utils/id"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/kratos-kit/auth/authn/engine"
	authData "github.com/liujitcn/kratos-kit/auth/data"
	"github.com/liujitcn/kratos-kit/sdk"
	"gorm.io/gorm"
)

const cacheKeyWxAccessToken = "wx_access_token"

// AuthCase 登录认证业务处理对象
type AuthCase struct {
	*biz.BaseCase
	userToken    *authData.UserToken
	baseUserCase *BaseUserCase
	baseRoleCase *BaseRoleCase
	baseDeptCase *BaseDeptCase
	wxMiniApp    *conf.WxMiniApp
	gorse        *gorse.Gorse
}

// NewAuthCase 创建登录认证业务处理对象
func NewAuthCase(
	baseCase *biz.BaseCase,
	userToken *authData.UserToken,
	baseUserCase *BaseUserCase,
	baseRoleCase *BaseRoleCase,
	baseDeptCase *BaseDeptCase,
	wxMiniApp *conf.WxMiniApp,
	gorse *gorse.Gorse,
) *AuthCase {
	return &AuthCase{
		BaseCase:     baseCase,
		userToken:    userToken,
		baseUserCase: baseUserCase,
		baseRoleCase: baseRoleCase,
		baseDeptCase: baseDeptCase,
		wxMiniApp:    wxMiniApp,
		gorse:        gorse,
	}
}

// WechatLogin 微信登录
func (c *AuthCase) WechatLogin(ctx context.Context, req *app.WechatLoginRequest) (*app.WechatLoginResponse, error) {
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
	// 未返回 Openid 时，当前登录请求无效。
	if sessionKey.Openid == "" {
		return nil, errorsx.Internal("登录失败")
	}

	var user *models.BaseUser
	user, err = c.baseUserCase.findByOpenid(ctx, sessionKey.Openid)
	// 按 Openid 查询用户失败时，仅对“未注册”场景继续自动注册。
	if err != nil {
		// 非“未注册”错误说明查询本身异常，直接返回登录失败。
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorsx.Internal("登录失败").WithCause(err)
		}

		// 用户不存在时自动注册一个小程序账号
		user = &models.BaseUser{
			Openid:   sessionKey.Openid,
			UserName: id.NewXID(),
			RoleID:   4,
			DeptID:   5,
			Phone:    "",
			Password: "",
			Gender:   3,
			Avatar:   "",
			Status:   int32(common.Status_ENABLE),
			Remark:   "自动注册用户",
		}
		// 自动注册用户失败时，直接返回登录失败。
		if err = c.baseUserCase.Create(ctx, user); err != nil {
			return nil, errorsx.Internal("登录失败").WithCause(err)
		}
	}
	// 用户被停用时，不允许继续登录。
	if user.Status != int32(common.Status_ENABLE) {
		return nil, errorsx.PermissionDenied("账号已被禁用")
	}

	// 登录凭证需要补齐角色和部门信息
	var role *models.BaseRole
	role, err = c.baseRoleCase.FindById(ctx, user.RoleID)
	if err != nil {
		return nil, errorsx.Internal("登录失败").WithCause(err)
	}
	var dept *models.BaseDept
	dept, err = c.baseDeptCase.FindById(ctx, user.DeptID)
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

	return &app.WechatLoginResponse{
		TokenType:    engine.BearerWord,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    c.userToken.GetAccessTokenExpires(),
	}, nil
}

// GetUserProfile 获取当前登录用户信息
func (c *AuthCase) GetUserProfile(ctx context.Context) (*app.UserProfileForm, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}

	var user *models.BaseUser
	user, err = c.baseUserCase.FindById(ctx, authInfo.UserId)
	if err != nil {
		return nil, errorsx.ResourceNotFound("用户不存在").WithCause(err)
	}
	// 用户被停用时，不允许继续获取个人信息。
	if user.Status != int32(common.Status_ENABLE) {
		return nil, errorsx.PermissionDenied("账号已被禁用")
	}

	return &app.UserProfileForm{
		UserName: user.UserName,
		NickName: user.NickName,
		Gender:   user.Gender,
		Phone:    _string.DesensitizePhone(user.Phone),
		Avatar:   user.Avatar,
	}, nil
}

// UpdateUserProfile 修改个人中心用户信息
func (c *AuthCase) UpdateUserProfile(ctx context.Context, req *app.UserProfileForm) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}

	var oldBaseUser *models.BaseUser
	oldBaseUser, err = c.baseUserCase.FindById(ctx, authInfo.UserId)
	if err != nil {
		return errorsx.ResourceNotFound("用户不存在").WithCause(err)
	}

	baseUser := &models.BaseUser{
		ID:       authInfo.UserId,
		NickName: req.GetNickName(),
		Gender:   req.GetGender(),
		Avatar:   req.GetAvatar(),
	}
	// 用户资料更新失败时，直接返回错误交由上层处理。
	if err = c.baseUserCase.UpdateById(ctx, baseUser); err != nil {
		return errorsx.Internal("修改个人中心用户信息失败").WithCause(err)
	}

	// 删除被替换的旧头像文件
	oss := sdk.Runtime.GetOSS()
	// OSS 可用时，尝试清理被替换掉的历史头像文件。
	if oss != nil {
		// 新头像为空或发生变更时，旧头像文件需要尝试删除。
		if baseUser.Avatar == "" || oldBaseUser.Avatar != baseUser.Avatar {
			// 头像文件删除失败时，只记录日志不影响主流程。
			if err = oss.DeleteFile(oldBaseUser.Avatar); err != nil {
				log.Error("deleteFile err:", err.Error())
			}
		}
	}
	return nil
}

// BindUserPhone 手机号授权
func (c *AuthCase) BindUserPhone(ctx context.Context, req *app.BindUserPhoneRequest) (*app.BindUserPhoneResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}

	var accessToken string
	accessToken, err = sdk.Runtime.GetCache().Get(cacheKeyWxAccessToken)
	// 本地缓存未命中 access token 时，回源微信重新获取。
	if err != nil {
		token, tokenErr := utils.GetAccessToken(c.wxMiniApp.GetAppid(), c.wxMiniApp.GetSecret())
		// 微信 access token 获取失败时，直接返回授权失败。
		if tokenErr != nil {
			return nil, errorsx.Internal("手机号授权失败").WithCause(tokenErr)
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
		if cacheErr := sdk.Runtime.GetCache().Set(cacheKeyWxAccessToken, accessToken, time.Duration(token.ExpiresIn-300)); cacheErr != nil {
			log.Error("cache set accessToken err:", cacheErr.Error())
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
			errorsx.MetadataKeyConflictType: errorsx.ConflictTypeUniqueViolation,
			errorsx.MetadataKeyResource:     "base_user",
			errorsx.MetadataKeyField:        "phone",
		})
	}

	user := &models.BaseUser{
		ID:    authInfo.UserId,
		Phone: phone.PhoneInfo.PhoneNumber,
	}
	// 绑定手机号写库失败时，直接返回业务错误。
	if err = c.baseUserCase.UpdateById(ctx, user); err != nil {
		return nil, errorsx.Internal("手机号授权失败").WithCause(err)
	}

	return &app.BindUserPhoneResponse{
		Phone: _string.DesensitizePhone(user.Phone),
	}, nil
}
