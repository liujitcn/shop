package biz

import (
	"context"
	"regexp"
	"strings"

	bootstrapConfigv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"

	adminv1 "shop/api/gen/go/admin/v1"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/go-utils/mapper"
)

// BaseAPICase 接口业务实例
type BaseAPICase struct {
	*biz.BaseCase
	*data.BaseAPIRepository
	mapper *mapper.CopierMapper[adminv1.BaseApi, models.BaseAPI]
	jwtCfg *bootstrapConfigv1.Authentication_Jwt
}

// NewBaseAPICase 创建接口业务实例
func NewBaseAPICase(baseCase *biz.BaseCase, baseAPIRepo *data.BaseAPIRepository, jwtCfg *bootstrapConfigv1.Authentication_Jwt) *BaseAPICase {
	return &BaseAPICase{
		BaseCase:          baseCase,
		BaseAPIRepository: baseAPIRepo,
		mapper:            mapper.NewCopierMapper[adminv1.BaseApi, models.BaseAPI](),
		jwtCfg:            jwtCfg,
	}
}

// ListBaseAPIs 查询接口列表
func (c *BaseAPICase) ListBaseAPIs(ctx context.Context, _ *adminv1.ListBaseApisRequest) (*adminv1.ListBaseApisResponse, error) {
	list, err := c.List(ctx)
	if err != nil {
		return nil, err
	}

	baseAPIs := make([]*adminv1.BaseApi, 0, len(list))
	for _, item := range list {
		// 命中免 token 或可选鉴权规则的接口，不再返回给菜单管理页面。
		if c.jwtCfg != nil {
			isNoTokenOperation := matchAuthWhiteList(c.jwtCfg.GetWhiteList(), item.Operation) ||
				matchAuthWhiteList(c.jwtCfg.GetOptionalAuth(), item.Operation)
			if isNoTokenOperation {
				continue
			}
		}
		baseAPI := c.mapper.ToDTO(item)
		baseAPIs = append(baseAPIs, baseAPI)
	}

	return &adminv1.ListBaseApisResponse{BaseApis: baseAPIs}, nil
}

// matchAuthWhiteList 按认证白名单规则匹配当前接口操作名。
func matchAuthWhiteList(whiteList *bootstrapConfigv1.Authentication_Jwt_WhiteList, operation string) bool {
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
