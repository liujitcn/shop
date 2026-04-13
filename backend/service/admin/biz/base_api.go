package biz

import (
	"context"
	"regexp"
	"strings"

	"shop/api/gen/go/admin"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	bootstrapConf "github.com/liujitcn/kratos-kit/api/gen/go/conf"
)

// BaseApiCase 接口业务实例
type BaseApiCase struct {
	*biz.BaseCase
	*data.BaseApiRepo
	mapper *mapper.CopierMapper[admin.BaseApi, models.BaseApi]
	jwtCfg *bootstrapConf.Authentication_Jwt
}

// NewBaseApiCase 创建接口业务实例
func NewBaseApiCase(baseCase *biz.BaseCase, baseApiRepo *data.BaseApiRepo, jwtCfg *bootstrapConf.Authentication_Jwt) *BaseApiCase {
	return &BaseApiCase{
		BaseCase:    baseCase,
		BaseApiRepo: baseApiRepo,
		mapper:      mapper.NewCopierMapper[admin.BaseApi, models.BaseApi](),
		jwtCfg:      jwtCfg,
	}
}

// ListBaseApi 查询接口列表
func (c *BaseApiCase) ListBaseApi(ctx context.Context) (*admin.ListBaseApiResponse, error) {
	list, err := c.List(ctx)
	if err != nil {
		return nil, err
	}

	resList := make([]*admin.BaseApi, 0, len(list))
	for _, item := range list {
		// 命中免 token 或可选鉴权规则的接口，不再返回给菜单管理页面。
		if c.isNoTokenOperation(item.Operation) {
			continue
		}
		baseApi := c.mapper.ToDTO(item)
		resList = append(resList, baseApi)
	}

	return &admin.ListBaseApiResponse{List: resList}, nil
}

// isNoTokenOperation 判断接口是否命中 auth.yaml 中的不强制 token 规则。
func (c *BaseApiCase) isNoTokenOperation(operation string) bool {
	// 未提供 JWT 配置时，保持原有接口列表返回行为。
	if c.jwtCfg == nil {
		return false
	}
	return matchAuthWhiteList(c.jwtCfg.GetWhiteList(), operation) || matchAuthWhiteList(c.jwtCfg.GetOptionalAuth(), operation)
}

// matchAuthWhiteList 按认证白名单规则匹配当前接口操作名。
func matchAuthWhiteList(whiteList *bootstrapConf.Authentication_Jwt_WhiteList, operation string) bool {
	// 白名单配置为空时，当前规则无需参与匹配。
	if whiteList == nil {
		return false
	}
	for _, prefix := range whiteList.GetPrefix() {
		// 前缀规则命中时，直接判定为免 token 接口。
		if strings.HasPrefix(operation, prefix) {
			return true
		}
	}
	for _, regexValue := range whiteList.GetRegex() {
		regex, err := regexp.Compile(regexValue)
		if err != nil {
			continue
		}
		// 正则完整命中当前操作名时，按白名单处理。
		if regex.FindString(operation) == operation {
			return true
		}
	}
	for _, path := range whiteList.GetPath() {
		// Path 精确匹配命中时，按白名单处理。
		if path == operation {
			return true
		}
	}
	for _, item := range whiteList.GetMatch() {
		// Match 精确匹配命中时，按白名单处理。
		if item == operation {
			return true
		}
	}
	return false
}
