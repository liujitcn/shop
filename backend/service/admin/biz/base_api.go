package biz

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	bootstrapConfigv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"

	adminv1 "shop/api/gen/go/admin/v1"
	"shop/internal/cmd/server/assets"
	"shop/pkg/biz"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repository"
	"gorm.io/gen"
)

const (
	baseAPIDocJSONMediaType = "application/json"
	baseAPIDocMaxDepth      = 8
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
	opts := make([]repository.QueryOption, 0, 10)
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
	if req.AgentEnabled != nil {
		opts = append(opts, repository.Where(query.AgentEnabled.Is(req.GetAgentEnabled())))
	}
	// 传入工具名时，按工具名模糊匹配。
	if req.GetToolName() != "" {
		opts = append(opts, repository.Where(query.ToolName.Like("%"+req.GetToolName()+"%")))
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

// GetBaseAPIDoc 查询接口 OpenAPI 文档
func (c *BaseAPICase) GetBaseAPIDoc(ctx context.Context, id int64) (*adminv1.BaseApiDoc, error) {
	query := c.Query(ctx).BaseAPI
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.ID.Eq(id)))

	baseAPI, err := c.Find(ctx, opts...)
	if err != nil {
		return nil, err
	}

	var api *biz.OpenAPI
	api, err = biz.ParseOpenAPI(assets.OpenAPIData)
	if err != nil {
		return nil, err
	}
	operation := api.Operation(baseAPI.Path, baseAPI.Method)
	if operation == nil {
		return nil, errorsx.Internal("查询API文档失败").WithCause(fmt.Errorf("openapi operation not found: %s %s", baseAPI.Method, baseAPI.Path))
	}

	return &adminv1.BaseApiDoc{
		Id:          baseAPI.ID,
		Summary:     operation.Summary,
		Description: operation.Description,
		Parameters:  buildBaseAPIDocParameters(api, operation.Parameters),
		RequestBody: buildBaseAPIDocRequestBody(api, operation.RequestBody),
		Responses:   buildBaseAPIDocResponses(api, operation.Responses),
	}, nil
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

// SetBaseAPIAgentEnabled 设置接口 Agent 启用状态
func (c *BaseAPICase) SetBaseAPIAgentEnabled(ctx context.Context, req *adminv1.SetBaseApiAgentEnabledRequest) error {
	query := c.Query(ctx).BaseAPI
	conditions := make([]gen.Condition, 0, 2)
	baseAPI, err := c.Find(ctx, repository.Where(query.ID.Eq(req.GetId())))
	if err != nil {
		return err
	}
	// 同名工具可能来自历史重复 API 记录，开关需要同步到同一个 Agent Tool 名称。
	if baseAPI.ToolName != "" {
		conditions = append(conditions, query.ToolName.Eq(baseAPI.ToolName))
	} else {
		conditions = append(conditions, query.ID.Eq(req.GetId()))
	}
	_, err = query.WithContext(ctx).
		Where(conditions...).
		UpdateSimple(query.AgentEnabled.Value(req.GetAgentEnabled()))
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

// buildBaseAPIDocParameters 构建请求参数文档。
func buildBaseAPIDocParameters(api *biz.OpenAPI, parameters []biz.Parameter) []*adminv1.BaseApiDocSchema {
	items := make([]*adminv1.BaseApiDocSchema, 0, len(parameters))
	for _, parameter := range parameters {
		item := buildBaseAPIDocSchema(api, parameter.Name, parameter.Name, parameter.In, parameter.Required, parameter.Schema, 0)
		if parameter.Description != "" {
			item.Description = parameter.Description
		}
		items = append(items, item)
	}
	return items
}

// buildBaseAPIDocSchema 展开 OpenAPI Schema 为前端可直接渲染的字段树。
func buildBaseAPIDocSchema(api *biz.OpenAPI, name, path, in string, required bool, schema biz.Schema, depth int) *adminv1.BaseApiDocSchema {
	schema, refName := dereferenceBaseAPIDocSchema(api, schema)
	item := &adminv1.BaseApiDocSchema{
		Name:        name,
		Path:        path,
		In:          in,
		Type:        schema.Type,
		Format:      schema.Format,
		Required:    required,
		Description: schema.Description,
		Ref:         refName,
		Enum:        schema.Enum,
	}
	if item.Type == "" {
		item.Type = inferBaseAPIDocSchemaType(schema)
	}
	if depth >= baseAPIDocMaxDepth {
		return item
	}
	if schema.Items != nil {
		child := buildBaseAPIDocSchema(api, name+"[]", path+"[]", in, false, *schema.Items, depth+1)
		item.Children = []*adminv1.BaseApiDocSchema{child}
	}
	if len(schema.Properties) > 0 {
		requiredFields := make(map[string]bool, len(schema.Required))
		for _, field := range schema.Required {
			requiredFields[field] = true
		}
		item.Children = make([]*adminv1.BaseApiDocSchema, 0, len(schema.Properties))
		fieldNames := make([]string, 0, len(schema.Properties))
		for fieldName := range schema.Properties {
			fieldNames = append(fieldNames, fieldName)
		}
		sort.Strings(fieldNames)
		for _, fieldName := range fieldNames {
			fieldSchema := schema.Properties[fieldName]
			fieldPath := fieldName
			if path != "" {
				fieldPath = path + "." + fieldName
			}
			item.Children = append(item.Children, buildBaseAPIDocSchema(api, fieldName, fieldPath, in, requiredFields[fieldName], fieldSchema, depth+1))
		}
	}
	return item
}

// dereferenceBaseAPIDocSchema 解析本地组件引用。
func dereferenceBaseAPIDocSchema(api *biz.OpenAPI, schema biz.Schema) (biz.Schema, string) {
	refName := strings.TrimPrefix(schema.Ref, "#/components/schemas/")
	if refName == "" || api == nil {
		return schema, refName
	}
	refSchema, ok := api.Components.Schemas[refName]
	if !ok {
		return schema, refName
	}
	if refSchema.Description == "" {
		refSchema.Description = schema.Description
	}
	return refSchema, refName
}

// inferBaseAPIDocSchemaType 推断缺省 Schema 类型。
func inferBaseAPIDocSchemaType(schema biz.Schema) string {
	if len(schema.Properties) > 0 {
		return "object"
	}
	if schema.Items != nil {
		return "array"
	}
	if schema.Ref != "" {
		return "object"
	}
	return "string"
}

// buildBaseAPIDocRequestBody 构建请求体文档。
func buildBaseAPIDocRequestBody(api *biz.OpenAPI, requestBody *biz.RequestBody) *adminv1.BaseApiDocSchema {
	if requestBody == nil {
		return nil
	}
	schema := selectBaseAPIDocContentSchema(requestBody.Content)
	if schema == nil {
		return nil
	}
	item := buildBaseAPIDocSchema(api, "body", "body", "body", requestBody.Required, *schema, 0)
	if requestBody.Description != "" {
		item.Description = requestBody.Description
	}
	return item
}

// selectBaseAPIDocContentSchema 选择可展示的 JSON Schema。
func selectBaseAPIDocContentSchema(content map[string]biz.MediaType) *biz.Schema {
	if len(content) == 0 {
		return nil
	}
	if media, ok := content[baseAPIDocJSONMediaType]; ok {
		return &media.Schema
	}
	for _, media := range content {
		return &media.Schema
	}
	return nil
}

// buildBaseAPIDocResponses 构建响应文档。
func buildBaseAPIDocResponses(api *biz.OpenAPI, responses map[string]biz.Response) []*adminv1.BaseApiDocResponse {
	items := make([]*adminv1.BaseApiDocResponse, 0, len(responses))
	statuses := make([]string, 0, len(responses))
	for status := range responses {
		statuses = append(statuses, status)
	}
	sort.Strings(statuses)
	for _, status := range statuses {
		response := responses[status]
		schema := selectBaseAPIDocContentSchema(response.Content)
		item := &adminv1.BaseApiDocResponse{
			Status:      status,
			Description: response.Description,
		}
		if schema != nil {
			item.Body = buildBaseAPIDocSchema(api, "body", "body", "body", false, *schema, 0)
		}
		items = append(items, item)
	}
	return items
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
