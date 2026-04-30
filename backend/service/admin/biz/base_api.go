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
	"github.com/liujitcn/gorm-kit/repository"
	"gorm.io/gen"
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

// PageBaseAPIs 分页查询接口列表
func (c *BaseAPICase) PageBaseAPIs(ctx context.Context, req *adminv1.PageBaseApisRequest) (*adminv1.PageBaseApisResponse, error) {
	query := c.Query(ctx).BaseAPI
	opts := make([]repository.QueryOption, 0, 9)
	opts = append(opts, repository.Order(query.ID.Desc()))
	// 传入服务名关键字时，按服务名模糊匹配。
	if req.GetServiceName() != "" {
		opts = append(opts, repository.Where(query.ServiceName.Like("%"+req.GetServiceName()+"%")))
	}
	// 传入服务描述关键字时，按服务描述模糊匹配。
	if req.GetServiceDesc() != "" {
		opts = append(opts, repository.Where(query.ServiceDesc.Like("%"+req.GetServiceDesc()+"%")))
	}
	// 传入描述关键字时，按接口描述模糊匹配。
	if req.GetDesc() != "" {
		opts = append(opts, repository.Where(query.Desc.Like("%"+req.GetDesc()+"%")))
	}
	// 传入操作方法关键字时，按操作方法模糊匹配。
	if req.GetOperation() != "" {
		opts = append(opts, repository.Where(query.Operation.Like("%"+req.GetOperation()+"%")))
	}
	// 传入请求方式时，按请求方式精确匹配。
	if req.GetMethod() != "" {
		opts = append(opts, repository.Where(query.Method.Eq(req.GetMethod())))
	}
	// 传入请求地址关键字时，按请求地址模糊匹配。
	if req.GetPath() != "" {
		opts = append(opts, repository.Where(query.Path.Like("%"+req.GetPath()+"%")))
	}
	if req.McpEnabled != nil {
		opts = append(opts, repository.Where(query.McpEnabled.Is(req.GetMcpEnabled())))
	}

	list, total, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}

	baseAPIs := make([]*adminv1.BaseApi, 0, len(list))
	for _, item := range list {
		baseAPI := c.mapper.ToDTO(item)
		baseAPIs = append(baseAPIs, baseAPI)
	}

	return &adminv1.PageBaseApisResponse{
		BaseApis: baseAPIs,
		Total:    int32(total),
	}, nil
}

// GetBaseAPI 根据主键查询接口详情
func (c *BaseAPICase) GetBaseAPI(ctx context.Context, id int64) (*adminv1.BaseApi, error) {
	query := c.Query(ctx).BaseAPI
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.ID.Eq(id)))

	baseAPI, err := c.Find(ctx, opts...)
	if err != nil {
		return nil, err
	}

	return c.mapper.ToDTO(baseAPI), nil
}

// SetBaseAPIMcpEnabled 设置接口 MCP 启用状态
func (c *BaseAPICase) SetBaseAPIMcpEnabled(ctx context.Context, req *adminv1.SetBaseApiMcpEnabledRequest) error {
	query := c.Query(ctx).BaseAPI
	conditions := make([]gen.Condition, 0, 1)
	conditions = append(conditions, query.ID.Eq(req.GetId()))
	_, err := query.WithContext(ctx).
		Where(conditions...).
		UpdateSimple(query.McpEnabled.Value(req.GetMcpEnabled()))
	return err
}

// ListBaseAPIs 查询菜单分配接口选项列表
func (c *BaseAPICase) ListBaseAPIs(ctx context.Context, _ *adminv1.ListBaseApisRequest) (*adminv1.ListBaseApisResponse, error) {
	query := c.Query(ctx).BaseAPI
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Order(query.ServiceName.Asc(), query.Operation.Asc()))
	list, err := c.List(ctx, opts...)
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
