package biz

import (
	"context"
	"errors"
	"fmt"
	"time"

	"shop/pkg/biz"
	"shop/pkg/gen/models"

	"shop/api/gen/go/app"
	"shop/api/gen/go/common"
	"shop/api/gen/go/conf"
	"shop/service/app/util"

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
}

// NewAuthCase 创建登录认证业务处理对象
func NewAuthCase(
	baseCase *biz.BaseCase,
	userToken *authData.UserToken,
	baseUserCase *BaseUserCase,
	baseRoleCase *BaseRoleCase,
	baseDeptCase *BaseDeptCase,
	wxMiniApp *conf.WxMiniApp,
) *AuthCase {
	return &AuthCase{
		BaseCase:     baseCase,
		userToken:    userToken,
		baseUserCase: baseUserCase,
		baseRoleCase: baseRoleCase,
		baseDeptCase: baseDeptCase,
		wxMiniApp:    wxMiniApp,
	}
}

// WxLogin 微信登录
func (c *AuthCase) WxLogin(ctx context.Context, req *app.WxLoginRequest) (*app.WxLoginResponse, error) {
	sessionKey, err := util.GetSessionKey(c.wxMiniApp.GetAppid(), c.wxMiniApp.GetSecret(), req.GetCode())
	if err != nil {
		return nil, errors.New("登录失败，code错误")
	}
	if sessionKey.ErrCode != 0 {
		return nil, fmt.Errorf("【%d】%s", sessionKey.ErrCode, sessionKey.ErrMsg)
	}
	if sessionKey.Openid == "" {
		return nil, errors.New("登录失败，Openid错误")
	}

	var user *models.BaseUser
	user, err = c.baseUserCase.findByOpenid(ctx, sessionKey.Openid)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("用户不存在")
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
		if err = c.baseUserCase.Create(ctx, user); err != nil {
			return nil, errors.New("用户不存在")
		}
	}
	if user.Status != int32(common.Status_ENABLE) {
		return nil, errors.New("用户状态错误")
	}

	// 登录凭证需要补齐角色和部门信息
	var role *models.BaseRole
	role, err = c.baseRoleCase.FindById(ctx, user.RoleID)
	if err != nil {
		return nil, errors.New("角色不存在")
	}
	var dept *models.BaseDept
	dept, err = c.baseDeptCase.FindById(ctx, user.DeptID)
	if err != nil {
		return nil, errors.New("角色不存在")
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
		return nil, errors.New("登录失败")
	}

	return &app.WxLoginResponse{
		TokenType:    engine.BearerWord,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    c.userToken.GetAccessTokenExpires(),
	}, nil
}

// GetUserInfo 获取当前登录用户信息
func (c *AuthCase) GetUserInfo(ctx context.Context) (*app.UserInfo, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}

	var user *models.BaseUser
	user, err = c.baseUserCase.FindById(ctx, authInfo.UserId)
	if err != nil {
		return nil, errors.New("用户不存在")
	}
	if user.Status != int32(common.Status_ENABLE) {
		return nil, errors.New("用户状态错误")
	}

	return &app.UserInfo{
		UserName: user.UserName,
		NickName: user.NickName,
		Gender:   user.Gender,
		Phone:    _string.DesensitizePhone(user.Phone),
		Avatar:   user.Avatar,
	}, nil
}

// UpdateUserInfo 修改个人中心用户信息
func (c *AuthCase) UpdateUserInfo(ctx context.Context, req *app.UpdateUserInfoRequest) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}

	var oldBaseUser *models.BaseUser
	oldBaseUser, err = c.baseUserCase.FindById(ctx, authInfo.UserId)
	if err != nil {
		log.Error("UpdateUserInfo find user err:", err.Error())
		return errors.New("修改个人中心用户信息失败")
	}

	baseUser := &models.BaseUser{
		ID:       authInfo.UserId,
		NickName: req.GetNickName(),
		Gender:   req.GetGender(),
		Avatar:   req.GetAvatar(),
	}
	if err = c.baseUserCase.UpdateById(ctx, baseUser); err != nil {
		log.Error("UpdateUserInfo update user err:", err.Error())
		return errors.New("修改个人中心用户信息失败")
	}

	// 删除被替换的旧头像文件
	oss := sdk.Runtime.GetOSS()
	if oss != nil {
		if baseUser.Avatar == "" || oldBaseUser.Avatar != baseUser.Avatar {
			if err = oss.DeleteFile(oldBaseUser.Avatar); err != nil {
				log.Error("deleteFile err:", err.Error())
			}
		}
	}
	return nil
}

// PhoneAuth 手机号授权
func (c *AuthCase) PhoneAuth(ctx context.Context, req *app.PhoneAuthRequest) (*app.PhoneAuthResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}

	var accessToken string
	accessToken, err = sdk.Runtime.GetCache().Get(cacheKeyWxAccessToken)
	if err != nil {
		token, tokenErr := util.GetAccessToken(c.wxMiniApp.GetAppid(), c.wxMiniApp.GetSecret())
		if tokenErr != nil {
			return nil, fmt.Errorf("授权失败:%s", tokenErr.Error())
		}
		if token.ErrCode != 0 {
			return nil, fmt.Errorf("授权失败:%s", token.ErrMsg)
		}
		accessToken = token.AccessToken
		if cacheErr := sdk.Runtime.GetCache().Set(cacheKeyWxAccessToken, accessToken, time.Duration(token.ExpiresIn-300)); cacheErr != nil {
			log.Error("cache set accessToken err:", cacheErr.Error())
		}
	}

	var phone *util.PhoneNumber
	phone, err = util.GetPhoneNumber(accessToken, req.GetCode())
	if err != nil {
		return nil, fmt.Errorf("授权失败:%s", err.Error())
	}
	if phone.ErrCode != 0 {
		return nil, fmt.Errorf("授权失败:%s", phone.ErrMsg)
	}

	var find *models.BaseUser
	find, err = c.baseUserCase.findByPhone(ctx, phone.PhoneInfo.PhoneNumber)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("授权失败")
	}
	if find != nil && find.ID != authInfo.UserId {
		return nil, fmt.Errorf("授权失败，手机号被占用")
	}

	user := &models.BaseUser{
		ID:    authInfo.UserId,
		Phone: phone.PhoneInfo.PhoneNumber,
	}
	if err = c.baseUserCase.UpdateById(ctx, user); err != nil {
		log.Error("PhoneAuth update user err:", err.Error())
		return nil, errors.New("手机号授权失败")
	}

	return &app.PhoneAuthResponse{
		Phone: _string.DesensitizePhone(user.Phone),
	}, nil
}
