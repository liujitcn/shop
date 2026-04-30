package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	openAPIStatusOK       = "200"
	openAPIStatusCreated  = "201"
	openAPIStatusAccepted = "202"
	openAPIStatusNoBody   = "204"
	openAPIStatusDefault  = "default"
	openAPIJSONMediaType  = "application/json"
)

// BaseAPICase 接口业务实例。
type BaseAPICase struct {
	*data.BaseAPIRepository
}

// NewBaseAPICase 创建接口业务实例。
func NewBaseAPICase(baseAPIRepo *data.BaseAPIRepository) *BaseAPICase {
	return &BaseAPICase{BaseAPIRepository: baseAPIRepo}
}

// openAPIDataToBaseAPI 将 OpenAPI 文档转换为接口模型。
func (c *BaseAPICase) openAPIDataToBaseAPI(openAPIData []byte) ([]*models.BaseAPI, error) {
	var api OpenAPI
	err := yaml.Unmarshal(openAPIData, &api)
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
				// 同步 OpenAPI 元数据时保留原来的 MCP 开关，避免刷新接口覆盖人工配置。
				item.McpEnabled = oldAPI.McpEnabled
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

	inputSchema, argMapping, outputSchema, err := buildOpenAPIMetadata(op)
	if err != nil {
		return nil, err
	}

	serviceName := fmt.Sprintf("%s.%s", packageName, serviceTag)
	serviceDesc := tagsMap[serviceName]
	// 存在标签时，优先使用首个标签作为服务归属。
	if len(op.Tags) > 0 {
		serviceName = fmt.Sprintf("%s.%s", packageName, op.Tags[0])
		// 标签描述存在时，同步写入服务描述字段。
		if value, ok := tagsMap[serviceName]; ok {
			serviceDesc = value
		}
	}

	return &models.BaseAPI{
		ServiceName:  serviceName,
		ServiceDesc:  serviceDesc,
		Desc:         operationDescription(op),
		Operation:    fmt.Sprintf("/%s.%s/%s", packageName, serviceTag, methodName),
		Method:       method,
		Path:         path,
		InputSchema:  inputSchema,
		ArgMapping:   argMapping,
		OutputSchema: outputSchema,
	}, nil
}

// buildOpenAPIMetadata 根据 OpenAPI 操作项构建接口 Schema 与参数映射。
func buildOpenAPIMetadata(op *Operation) (string, string, string, error) {
	properties := make(map[string]any)
	required := make([]string, 0)
	argMappings := make([]ArgMappingItem, 0)

	for _, parameter := range op.Parameters {
		property := buildParameterSchema(parameter)
		properties[parameter.Name] = property
		// 必填参数需要同步写入 JSON Schema required 列表。
		if parameter.Required {
			required = append(required, parameter.Name)
		}
		argMappings = append(argMappings, ArgMappingItem{
			Name:        parameter.Name,
			Position:    parameter.In,
			Required:    parameter.Required,
			Type:        schemaType(property),
			Description: schemaDescription(parameter.Description, property),
		})
	}

	bodySchema := selectRequestBodySchema(op.RequestBody)
	// 存在请求体时，统一以 body 参数承载，便于 MCP 调用层按位置转发。
	if bodySchema != nil {
		properties["body"] = bodySchema
		// 请求体标记为必填时，同步写入 JSON Schema required 列表。
		if op.RequestBody.Required {
			required = append(required, "body")
		}
		argMappings = append(argMappings, ArgMappingItem{
			Name:        "body",
			Position:    "body",
			Required:    op.RequestBody.Required,
			Type:        schemaType(bodySchema),
			Description: schemaDescription(op.RequestBody.Description, bodySchema),
		})
	}

	inputSchema := map[string]any{
		"type":       "object",
		"properties": properties,
		"required":   required,
	}
	inputSchemaBytes, err := json.Marshal(inputSchema)
	if err != nil {
		return "", "", "", err
	}

	var argMappingBytes []byte
	argMappingBytes, err = json.Marshal(argMappings)
	if err != nil {
		return "", "", "", err
	}

	outputSchema := selectResponseSchema(op.Responses)
	// 响应没有结构化 body 时，使用空对象避免 JSON 字段写入空字符串。
	if outputSchema == nil {
		outputSchema = map[string]any{}
	}
	var outputSchemaBytes []byte
	outputSchemaBytes, err = json.Marshal(outputSchema)
	if err != nil {
		return "", "", "", err
	}

	return string(inputSchemaBytes), string(argMappingBytes), string(outputSchemaBytes), nil
}

// buildParameterSchema 构建单个参数的 JSON Schema。
func buildParameterSchema(parameter Parameter) map[string]any {
	schema := cloneSchema(parameter.Schema)
	// 参数 schema 缺少类型时，默认按字符串处理，避免生成无类型参数。
	if len(schema) == 0 {
		schema["type"] = "string"
	}
	// OpenAPI 参数描述优先补入 JSON Schema，方便 MCP 客户端展示。
	if parameter.Description != "" {
		if _, ok := schema["description"]; !ok {
			schema["description"] = parameter.Description
		}
	}
	return schema
}

// cloneSchema 复制 OpenAPI Schema，避免补充 description 时修改原始结构。
func cloneSchema(schema map[string]any) map[string]any {
	result := make(map[string]any, len(schema)+1)
	for key, value := range schema {
		result[key] = value
	}
	return result
}

// selectRequestBodySchema 选择请求体中的首个可用 JSON Schema。
func selectRequestBodySchema(requestBody *RequestBody) map[string]any {
	// 请求体为空时，当前接口没有 body 参数。
	if requestBody == nil {
		return nil
	}
	return selectContentSchema(requestBody.Content)
}

// selectResponseSchema 选择 2xx 或默认响应中的 JSON Schema。
func selectResponseSchema(responses map[string]Response) map[string]any {
	// 响应定义为空时，当前接口没有可记录的输出结构。
	if len(responses) == 0 {
		return nil
	}

	preferredStatuses := []string{openAPIStatusOK, openAPIStatusCreated, openAPIStatusAccepted, openAPIStatusNoBody, openAPIStatusDefault}
	for _, status := range preferredStatuses {
		response, ok := responses[status]
		// 优先响应状态存在时，尝试读取对应 content schema。
		if ok {
			schema := selectContentSchema(response.Content)
			if schema != nil {
				return schema
			}
		}
	}

	statuses := make([]string, 0, len(responses))
	for status := range responses {
		// 兜底只从 2xx 响应中选择输出结构，避免错误响应污染正常返回 Schema。
		if strings.HasPrefix(status, "2") {
			statuses = append(statuses, status)
		}
	}
	sort.Strings(statuses)
	for _, status := range statuses {
		response := responses[status]
		schema := selectContentSchema(response.Content)
		if schema != nil {
			return schema
		}
	}
	return nil
}

// selectContentSchema 从 content 中选择 JSON Schema。
func selectContentSchema(content map[string]MediaType) map[string]any {
	// content 为空时，当前请求或响应没有结构化内容。
	if len(content) == 0 {
		return nil
	}

	jsonMedia, ok := content[openAPIJSONMediaType]
	// 优先选择 application/json，保持与当前接口默认编解码一致。
	if ok && len(jsonMedia.Schema) > 0 {
		return cloneSchema(jsonMedia.Schema)
	}

	mediaTypes := make([]string, 0, len(content))
	for mediaType := range content {
		mediaTypes = append(mediaTypes, mediaType)
	}
	sort.Strings(mediaTypes)
	for _, mediaType := range mediaTypes {
		media := content[mediaType]
		// JSON 媒体类型不存在时，按稳定顺序选择首个带 Schema 的内容定义。
		if len(media.Schema) > 0 {
			return cloneSchema(media.Schema)
		}
	}
	return nil
}

// schemaType 获取 Schema 类型描述。
func schemaType(schema map[string]any) string {
	value, ok := schema["type"].(string)
	// Schema 显式声明类型时，直接复用 OpenAPI 类型。
	if ok && value != "" {
		return value
	}
	_, ok = schema["$ref"]
	// 引用类型默认按 object 处理。
	if ok {
		return "object"
	}
	return "string"
}

// schemaDescription 获取参数描述。
func schemaDescription(defaultDescription string, schema map[string]any) string {
	value, ok := schema["description"].(string)
	// Schema 已存在描述时，优先使用 Schema 描述。
	if ok && value != "" {
		return value
	}
	return defaultDescription
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
	Paths map[string]PathItem `yaml:"paths"`
	Tags  []TagsItem          `yaml:"tags"`
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

// Parameter 描述 OpenAPI 参数项。
type Parameter struct {
	Name        string         `yaml:"name"`
	In          string         `yaml:"in"`
	Description string         `yaml:"description"`
	Required    bool           `yaml:"required"`
	Schema      map[string]any `yaml:"schema"`
}

// RequestBody 描述 OpenAPI 请求体。
type RequestBody struct {
	Description string               `yaml:"description"`
	Required    bool                 `yaml:"required"`
	Content     map[string]MediaType `yaml:"content"`
}

// Response 描述 OpenAPI 响应项。
type Response struct {
	Description string               `yaml:"description"`
	Content     map[string]MediaType `yaml:"content"`
}

// MediaType 描述 OpenAPI 媒体类型内容。
type MediaType struct {
	Schema map[string]any `yaml:"schema"`
}

// ArgMappingItem 描述接口参数位置映射。
type ArgMappingItem struct {
	Name        string `json:"name"`
	Position    string `json:"position"`
	Required    bool   `json:"required"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
}
