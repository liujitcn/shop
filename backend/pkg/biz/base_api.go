package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	"sort"
	"strings"

	kitutils "github.com/liujitcn/kratos-kit/utils"
	"gopkg.in/yaml.v3"
)

// BaseAPICase 接口业务实例。
type BaseAPICase struct {
	*data.BaseAPIRepository
}

// NewBaseAPICase 创建接口业务实例。
func NewBaseAPICase(baseAPIRepo *data.BaseAPIRepository) *BaseAPICase {
	return &BaseAPICase{BaseAPIRepository: baseAPIRepo}
}

// ParseOpenAPI 解析 OpenAPI 文档。
func ParseOpenAPI(openAPIData []byte) (*OpenAPI, error) {
	var api OpenAPI
	err := yaml.Unmarshal(openAPIData, &api)
	if err != nil {
		return nil, err
	}
	return &api, nil
}

// Operation 按 path 和 method 获取 OpenAPI 操作定义。
func (api *OpenAPI) Operation(path, method string) *Operation {
	if api == nil {
		return nil
	}
	item, ok := api.Paths[path]
	if !ok {
		return nil
	}
	switch method {
	case "GET":
		return item.Get
	case "POST":
		return item.Post
	case "PUT":
		return item.Put
	case "DELETE":
		return item.Delete
	default:
		return nil
	}
}

// openAPIDataToBaseAPI 将 OpenAPI 文档转换为接口模型。
func (c *BaseAPICase) openAPIDataToBaseAPI(openAPIData []byte) ([]*models.BaseAPI, error) {
	api, err := ParseOpenAPI(openAPIData)
	if err != nil {
		return nil, err
	}

	tagsMap := buildTagsMap(api.Tags)

	baseAPIList := make([]*models.BaseAPI, 0)
	for path, item := range api.Paths {
		operations := []PathOperation{
			{Method: "GET", Operation: item.Get},
			{Method: "POST", Operation: item.Post},
			{Method: "PUT", Operation: item.Put},
			{Method: "DELETE", Operation: item.Delete},
		}

		for _, operation := range operations {
			var baseAPI *models.BaseAPI
			baseAPI, err = parseOperation(path, operation.Method, operation.Operation, tagsMap)
			if err != nil {
				return nil, err
			}
			// 当前路径存在对应 HTTP 操作时，写入接口权限与 MCP 元数据。
			if baseAPI != nil {
				baseAPIList = append(baseAPIList, baseAPI)
			}
		}
	}

	sort.Slice(baseAPIList, func(i, j int) bool {
		return baseAPIList[i].Operation < baseAPIList[j].Operation
	})
	return baseAPIList, nil
}

// batchCreateBaseAPI 批量同步接口数据。
func (c *BaseAPICase) batchCreateBaseAPI(ctx context.Context, apis []*models.BaseAPI) error {
	oldAPIList, err := c.List(ctx)
	if err != nil {
		return err
	}

	oldAPIIDMap := make(map[string][]int64, len(oldAPIList))
	oldAPIByOperation := make(map[string]*models.BaseAPI, len(oldAPIList))
	for _, oldAPI := range oldAPIList {
		oldAPIIDMap[oldAPI.Operation] = append(oldAPIIDMap[oldAPI.Operation], oldAPI.ID)
		// 同一个 operation 存在重复历史数据时，首条记录作为保留记录。
		if _, ok := oldAPIByOperation[oldAPI.Operation]; !ok {
			oldAPIByOperation[oldAPI.Operation] = oldAPI
		}
	}

	apiList := make([]*models.BaseAPI, 0)
	deleteAPIIDs := make([]int64, 0)
	for _, item := range apis {
		// 已存在的接口按主键更新，保留历史权限关联。
		if ids, ok := oldAPIIDMap[item.Operation]; ok && len(ids) > 0 {
			item.ID = ids[0]
			if oldAPI := oldAPIByOperation[item.Operation]; oldAPI != nil {
				// 同步 OpenAPI 元数据时保留原来的工具开关，避免刷新接口覆盖人工配置。
				item.McpEnabled = oldAPI.McpEnabled
				item.AgentEnabled = oldAPI.AgentEnabled
				// 工具提示词允许后台人工维护，刷新接口时只保留真正自定义过的提示词。
				oldPrompts := decodeToolPrompts(oldAPI.ToolPrompts)
				if len(oldPrompts) > 0 && !sameToolPrompts(oldPrompts, defaultToolPrompts(oldAPI.ServiceDesc, oldAPI.Desc)) {
					item.ToolPrompts = oldAPI.ToolPrompts
				}
			}
			err = c.UpdateByID(ctx, item)
			if err != nil {
				return err
			}
			// 同一个 operation 只保留一条记录，其余历史重复数据同步清理。
			if len(ids) > 1 {
				deleteAPIIDs = append(deleteAPIIDs, ids[1:]...)
			}
			delete(oldAPIIDMap, item.Operation)
			continue
		}
		apiList = append(apiList, item)
	}

	// 历史接口存在但 OpenAPI 已删除时，同步清理失效接口。
	if len(oldAPIIDMap) > 0 {
		for _, ids := range oldAPIIDMap {
			deleteAPIIDs = append(deleteAPIIDs, ids...)
		}
	}

	// 存在失效或重复接口时，统一按主键软删除。
	if len(deleteAPIIDs) > 0 {
		err = c.DeleteByIDs(ctx, deleteAPIIDs)
		if err != nil {
			return err
		}
	}

	// 没有新增接口时，无需再执行批量创建。
	if len(apiList) == 0 {
		return nil
	}
	return c.BatchCreate(ctx, apiList)
}

// parseOperation 解析单个 OpenAPI 操作项。
func parseOperation(path, method string, op *Operation, tagsMap map[string]string) (*models.BaseAPI, error) {
	// 操作项为空时，当前请求方法无需生成接口权限数据。
	if op == nil {
		return nil, nil
	}

	packageName := inferProtoPackage(path)
	// additional_bindings 生成的旧路径不落库，避免同一个 RPC 生成多条接口权限。
	if packageName == "" {
		return nil, nil
	}

	serviceTag, methodName := parseOperationID(op.OperationID)
	// operationId 缺少服务或方法时无法还原运行时 operation，直接跳过异常数据。
	if serviceTag == "" || methodName == "" {
		return nil, nil
	}

	serviceName := fmt.Sprintf("%s.%s", packageName, serviceTag)
	serviceDesc := tagsMap[serviceName]
	operation := fmt.Sprintf("/%s.%s/%s", packageName, serviceTag, methodName)
	// 存在标签时，优先使用首个标签作为服务归属。
	if len(op.Tags) > 0 {
		serviceName = fmt.Sprintf("%s.%s", packageName, op.Tags[0])
		// 标签描述存在时，同步写入服务描述字段。
		if value, ok := tagsMap[serviceName]; ok {
			serviceDesc = value
		}
	}

	operationDesc := operationDescription(op)

	return &models.BaseAPI{
		McpEnabled:   true,
		AgentEnabled: true,
		ToolName:     kitutils.ToolNameFromRPCPath(operation),
		ToolPrompts:  encodeToolPrompts(defaultToolPrompts(serviceDesc, operationDesc)),
		ServiceName:  serviceName,
		ServiceDesc:  serviceDesc,
		Desc:         operationDesc,
		Operation:    operation,
		Method:       method,
		Path:         path,
	}, nil
}

// defaultToolPrompts 根据 OpenAPI 原始服务描述和接口描述生成默认工具提示词。
func defaultToolPrompts(serviceDesc, desc string) []string {
	values := make([]string, 0, 2)
	// 服务描述与接口描述同时存在时，保留组合提示，增强完整语义命中。
	if serviceDesc != "" && desc != "" {
		values = append(values, serviceDesc+"："+desc)
	}
	if desc != "" {
		values = append(values, desc)
	}
	if serviceDesc != "" && desc == "" {
		values = append(values, serviceDesc)
	}
	return values
}

// encodeToolPrompts 将工具提示词编码为数据库 JSON 字段。
func encodeToolPrompts(prompts []string) string {
	raw, err := json.Marshal(prompts)
	if err != nil {
		return "[]"
	}
	return string(raw)
}

// decodeToolPrompts 将数据库 JSON 字段解析为工具提示词。
func decodeToolPrompts(value string) []string {
	if value == "" {
		return nil
	}
	var prompts []string
	err := json.Unmarshal([]byte(value), &prompts)
	if err != nil {
		return nil
	}
	return prompts
}

// sameToolPrompts 判断两组工具提示词是否完全一致。
func sameToolPrompts(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index, item := range left {
		if item != right[index] {
			return false
		}
	}
	return true
}

// operationDescription 获取接口描述。
func operationDescription(op *Operation) string {
	// description 更适合作为工具说明，存在时优先使用。
	if op.Description != "" {
		return op.Description
	}
	return op.Summary
}

// buildTagsMap 构建带 proto package 的服务标签描述索引。
func buildTagsMap(tags []TagsItem) map[string]string {
	tagsMap := make(map[string]string, len(tags))
	for _, item := range tags {
		packageName := inferProtoPackageByTagDescription(item.Description)
		// 标签描述包含终端前缀时，使用完整服务名索引，避免同名服务互相覆盖。
		if packageName != "" {
			tagsMap[fmt.Sprintf("%s.%s", packageName, item.Name)] = item.Description
			continue
		}
		tagsMap[item.Name] = item.Description
	}
	return tagsMap
}

// inferProtoPackage 根据主版本 HTTP 路径推断 proto package。
func inferProtoPackage(path string) string {
	paths := strings.Split(strings.Trim(path, "/"), "/")
	// 只同步 /api/v1/{terminal} 主路径，忽略 additional_bindings 生成的旧路径。
	if len(paths) < 3 || paths[0] != "api" || paths[1] != "v1" {
		return ""
	}

	// 根据 v1 后的终端段映射到对应 proto package。
	switch paths[2] {
	// 管理后台接口对应 admin.v1。
	case "admin":
		return "admin.v1"
	// 商城端接口对应 app.v1。
	case "app":
		return "app.v1"
	// 公共基础接口对应 base.v1。
	case "base":
		return "base.v1"
	default:
		return ""
	}
}

// inferProtoPackageByTagDescription 根据标签描述推断 proto package。
func inferProtoPackageByTagDescription(description string) string {
	// Admin 标签描述对应管理后台服务。
	if strings.HasPrefix(description, "Admin") {
		return "admin.v1"
	}
	// App 标签描述对应商城端服务。
	if strings.HasPrefix(description, "App") {
		return "app.v1"
	}
	// Base 标签描述对应公共基础服务。
	if strings.HasPrefix(description, "Base") {
		return "base.v1"
	}
	return ""
}

// parseOperationID 从 OpenAPI operationId 中拆出服务名和方法名。
func parseOperationID(operationID string) (string, string) {
	parts := strings.SplitN(operationID, "_", 2)
	// operationId 必须满足 Service_Method 格式才能还原 Kratos 运行时 operation。
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", ""
	}
	return parts[0], parts[1]
}

// OpenAPI 描述 OpenAPI 文档结构。
type OpenAPI struct {
	Paths      map[string]PathItem `yaml:"paths"`
	Tags       []TagsItem          `yaml:"tags"`
	Components Components          `yaml:"components"`
}

// PathItem 描述单个路径的请求方法。
type PathItem struct {
	Get    *Operation `yaml:"get,omitempty"`
	Post   *Operation `yaml:"post,omitempty"`
	Put    *Operation `yaml:"put,omitempty"`
	Delete *Operation `yaml:"delete,omitempty"`
}

// PathOperation 描述单个路径下的 HTTP 操作。
type PathOperation struct {
	Method    string
	Operation *Operation
}

// TagsItem 描述 OpenAPI 标签信息。
type TagsItem struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// Operation 描述单个接口操作项。
type Operation struct {
	Tags        []string            `yaml:"tags"`
	Summary     string              `yaml:"summary"`
	Description string              `yaml:"description"`
	OperationID string              `yaml:"operationId"`
	Parameters  []Parameter         `yaml:"parameters"`
	RequestBody *RequestBody        `yaml:"requestBody"`
	Responses   map[string]Response `yaml:"responses"`
}

// Components 描述 OpenAPI 组件定义。
type Components struct {
	Schemas map[string]Schema `yaml:"schemas"`
}

// Parameter 描述 OpenAPI 请求参数。
type Parameter struct {
	Name        string `yaml:"name"`
	In          string `yaml:"in"`
	Description string `yaml:"description"`
	Required    bool   `yaml:"required"`
	Schema      Schema `yaml:"schema"`
}

// RequestBody 描述 OpenAPI 请求体。
type RequestBody struct {
	Description string               `yaml:"description"`
	Required    bool                 `yaml:"required"`
	Content     map[string]MediaType `yaml:"content"`
}

// Response 描述 OpenAPI 响应。
type Response struct {
	Description string               `yaml:"description"`
	Content     map[string]MediaType `yaml:"content"`
}

// MediaType 描述 OpenAPI 媒体类型。
type MediaType struct {
	Schema Schema `yaml:"schema"`
}

// Schema 描述 OpenAPI Schema。
type Schema struct {
	Ref         string            `yaml:"$ref"`
	Type        string            `yaml:"type"`
	Format      string            `yaml:"format"`
	Description string            `yaml:"description"`
	Enum        []string          `yaml:"enum"`
	Required    []string          `yaml:"required"`
	Properties  map[string]Schema `yaml:"properties"`
	Items       *Schema           `yaml:"items"`
}
