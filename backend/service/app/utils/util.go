package utils

import (
	"context"
	"strconv"
	"strings"

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
	if len(authInfo.RoleCode) == 0 || authInfo.RoleCode == _const.BASE_ROLE_CODE_GUEST {
		return false
	}
	// 当前只有普通用户角色享受会员价
	if authInfo.RoleCode == _const.BASE_ROLE_CODE_USER {
		return true
	}
	return false
}

// BuildOrderGoodsCommentKey 构建订单商品评价关联键。
func BuildOrderGoodsCommentKey(orderID int64, goodsID int64, skuCode string) string {
	return strconv.FormatInt(orderID, 10) + "_" + strconv.FormatInt(goodsID, 10) + "_" + strings.TrimSpace(skuCode)
}
