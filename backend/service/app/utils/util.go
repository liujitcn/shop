package utils

import (
	"context"

	_const "shop/pkg/const"

	"github.com/liujitcn/kratos-kit/auth"
	"github.com/liujitcn/kratos-kit/auth/data"
)

// IsMember 根据上下文中的登录信息判断当前用户是否为会员
func IsMember(ctx context.Context) bool {
	authInfo, err := auth.FromContext(ctx)
	if err != nil {
		return false
	}
	return IsMemberByAuthInfo(authInfo)
}

// IsMemberByAuthInfo 根据登录载荷判断当前用户是否为会员
func IsMemberByAuthInfo(authInfo *data.UserTokenPayload) bool {
	// 游客或空角色码一律按非会员处理
	if len(authInfo.RoleCode) == 0 || authInfo.RoleCode == _const.BaseRoleCode_Guest {
		return false
	}
	// 当前只有普通用户角色享受会员价
	if authInfo.RoleCode == _const.BaseRoleCode_User {
		return true
	}
	return false
}
