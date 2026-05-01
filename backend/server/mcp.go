package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/liujitcn/gorm-kit/repository"
	"github.com/liujitcn/kratos-kit/bootstrap"
	"github.com/liujitcn/kratos-kit/rpc"
	mcpServer "github.com/liujitcn/kratos-kit/transport/mcp"
	"github.com/mark3labs/mcp-go/mcp"
	mcpGoServer "github.com/mark3labs/mcp-go/server"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"

	"shop/internal/cmd/server/assets"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
)

const (
	mcpDefaultInputSchema   = `{"type":"object","properties":{}}`
	mcpDefaultHTTPTimeout   = 10 * time.Second
	mcpDefaultEndpointPath  = "/mcp"
	mcpMaxHTTPResponseBytes = 8 << 20
	mcpMaxSchemaRefDepth    = 16

	mcpArgPositionPath   = "path"
	mcpArgPositionQuery  = "query"
	mcpArgPositionHeader = "header"
	mcpArgPositionBody   = "body"
)

type mcpArgMappingItem struct {
	Name        string `json:"name"`
	Position    string `json:"position"`
	Required    bool   `json:"required"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
}

// NewMcpServer 创建 MCP HTTP 处理器，并从 base_api 表加载接口工具。
func NewMcpServer(ctx *bootstrap.Context, baseAPIRepo *data.BaseAPIRepository) (*mcpServer.Server, error) {
	cfg := ctx.GetConfig()
	// 未启用 HTTP 服务时，不创建 MCP HTTP 处理器。
	if cfg == nil || cfg.Server == nil || cfg.Server.Http == nil {
		return nil, nil
	}

	appInfo := ctx.GetAppInfo()
	serverOptions := []mcpServer.ServerOption{
		mcpServer.WithStreamableHTTPOptions(mcpGoServer.WithEndpointPath(mcpDefaultEndpointPath)),
	}
	mcpOptions := []mcpGoServer.ServerOption{
		mcpGoServer.WithToolCapabilities(false),
		mcpGoServer.WithRecovery(),
	}
	// 应用信息存在时，同步作为 MCP 服务元信息。
	if appInfo != nil {
		if appInfo.GetName() != "" {
			serverOptions = append(serverOptions, mcpServer.WithServerName(appInfo.GetName()))
		}
		if appInfo.GetVersion() != "" {
			serverOptions = append(serverOptions, mcpServer.WithServerVersion(appInfo.GetVersion()))
		}
	}

	httpTimeout := mcpDefaultHTTPTimeout
	if cfg.Server.Http != nil && cfg.Server.Http.GetTimeout() != nil && cfg.Server.Http.GetTimeout().AsDuration() > 0 {
		httpTimeout = cfg.Server.Http.GetTimeout().AsDuration()
	}
	httpClient := newMcpHTTPClient(httpTimeout)
	openAPISchemas := loadMcpOpenAPISchemas()
	mcpToolOperationMap := make(map[string]string)
	mcpOptions = append(mcpOptions, mcpGoServer.WithToolFilter(func(ctx context.Context, tools []mcp.Tool) []mcp.Tool {
		return filterMcpTools(ctx, baseAPIRepo, mcpToolOperationMap, tools)
	}))
	serverOptions = append(serverOptions, rpc.WithMcpServerOptions(mcpOptions...))

	srv, err := rpc.CreateMcpHandler(cfg, serverOptions...)
	if err != nil {
		return nil, err
	}
	// 仓储未注入时只返回空 MCP 服务，避免影响应用启动。
	if baseAPIRepo == nil {
		return srv, nil
	}

	var apiList []*models.BaseAPI
	apiList, err = listMcpBaseAPIs(context.Background(), baseAPIRepo, false)
	if err != nil {
		return nil, err
	}

	httpEndpoint := ""
	// HTTP 地址存在时，MCP 工具调用会转发到本服务 HTTP 接口。
	if cfg.Server.Http != nil {
		httpEndpoint = normalizeMcpHTTPEndpoint(cfg.Server.Http.GetAddr())
	}
	toolNames := make(map[string]int, len(apiList))
	for _, baseAPI := range apiList {
		if baseAPI == nil {
			continue
		}
		var mappings []mcpArgMappingItem
		mappings, err = parseMcpArgMappings(baseAPI.ArgMapping)
		if err != nil {
			log.Errorf("解析 MCP 参数映射失败 operation=%s err=%v", baseAPI.Operation, err)
			continue
		}
		toolName, description, inputSchema := buildMcpToolDefinition(baseAPI, toolNames, openAPISchemas)
		mcpToolOperationMap[toolName] = baseAPI.Operation
		currentAPI := baseAPI
		currentMappings := mappings
		err = srv.RegisterHandler(
			mcp.NewToolWithRawSchema(toolName, description, json.RawMessage(inputSchema)),
			func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				return buildMcpToolResult(ctx, currentAPI, baseAPIRepo, httpEndpoint, httpClient, currentMappings, request)
			},
		)
		if err != nil {
			log.Errorf("注册 MCP 工具失败 operation=%s err=%v", baseAPI.Operation, err)
			continue
		}
		log.Infof("注册 MCP 工具成功 name=%s operation=%s", toolName, baseAPI.Operation)
	}

	return srv, nil
}

// listMcpBaseAPIs 查询 base_api 接口元数据。
func listMcpBaseAPIs(ctx context.Context, baseAPIRepo *data.BaseAPIRepository, onlyMcpEnabled bool) ([]*models.BaseAPI, error) {
	query := baseAPIRepo.Query(ctx).BaseAPI
	opts := make([]repository.QueryOption, 0, 2)
	if onlyMcpEnabled {
		opts = append(opts, repository.Where(query.McpEnabled.Is(true)))
	}
	opts = append(opts, repository.Order(query.ServiceName.Asc(), query.Operation.Asc()))
	return baseAPIRepo.List(ctx, opts...)
}

// filterMcpTools 根据当前 base_api 的 MCP 开关动态过滤工具列表。
func filterMcpTools(ctx context.Context, baseAPIRepo *data.BaseAPIRepository, toolOperationMap map[string]string, tools []mcp.Tool) []mcp.Tool {
	if baseAPIRepo == nil || len(tools) == 0 {
		return nil
	}
	apiList, err := listMcpBaseAPIs(ctx, baseAPIRepo, true)
	if err != nil {
		log.Errorf("刷新 MCP 工具列表失败 err=%v", err)
		return nil
	}
	enabledOperations := make(map[string]bool, len(apiList))
	for _, baseAPI := range apiList {
		if baseAPI == nil {
			continue
		}
		enabledOperations[baseAPI.Operation] = true
	}
	filteredTools := make([]mcp.Tool, 0, len(tools))
	for _, tool := range tools {
		operation := toolOperationMap[tool.Name]
		if enabledOperations[operation] {
			filteredTools = append(filteredTools, tool)
		}
	}
	return filteredTools
}

// buildMcpToolResult 构建 MCP 工具调用结果。
func buildMcpToolResult(ctx context.Context, baseAPI *models.BaseAPI, baseAPIRepo *data.BaseAPIRepository, httpEndpoint string, httpClient *http.Client, mappings []mcpArgMappingItem, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	apiPayload := mcpAPIPayload(baseAPI)
	if err := ensureMcpBaseAPIEnabled(ctx, baseAPIRepo, baseAPI); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	// 未配置 HTTP 服务地址时，仅返回工具元数据，避免 MCP 调用误判为空响应。
	if httpEndpoint == "" {
		payload := map[string]any{
			"api":       apiPayload,
			"arguments": request.GetRawArguments(),
			"executed":  false,
			"message":   "HTTP 服务地址未配置，已返回工具元数据",
		}
		return mcp.NewToolResultStructuredOnly(payload), nil
	}

	args, err := mcpArguments(request.GetRawArguments())
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	var req *http.Request
	req, err = buildMcpHTTPRequest(ctx, baseAPI, httpEndpoint, request.Header, mappings, args)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if httpClient == nil {
		httpClient = newMcpHTTPClient(mcpDefaultHTTPTimeout)
	}
	var resp *http.Response
	resp, err = httpClient.Do(req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.Errorf("关闭 MCP HTTP 响应失败 err=%v", closeErr)
		}
	}()

	var responseBytes []byte
	responseBytes, err = io.ReadAll(io.LimitReader(resp.Body, mcpMaxHTTPResponseBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(responseBytes)) > mcpMaxHTTPResponseBytes {
		return mcp.NewToolResultError(fmt.Sprintf("MCP HTTP 响应超过大小限制 %d 字节", mcpMaxHTTPResponseBytes)), nil
	}
	payload := map[string]any{
		"api":         apiPayload,
		"arguments":   request.GetRawArguments(),
		"executed":    true,
		"status_code": resp.StatusCode,
		"headers":     resp.Header.Clone(),
		"body":        parseJSONValue(string(responseBytes)),
	}
	result := mcp.NewToolResultStructuredOnly(payload)
	// HTTP 非 2xx 响应按工具调用错误返回，方便 MCP 客户端识别失败。
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		result.IsError = true
	}
	return result, nil
}

// buildMcpToolDefinition 根据接口元数据构建 MCP 工具名、描述和入参 Schema。
func buildMcpToolDefinition(baseAPI *models.BaseAPI, toolNames map[string]int, openAPISchemas map[string]any) (string, string, string) {
	name := strings.TrimSpace(baseAPI.Operation)
	// operation 为空时，使用请求方法和路径兜底生成稳定工具名。
	if name == "" {
		name = strings.TrimSpace(fmt.Sprintf("%s_%s", baseAPI.Method, baseAPI.Path))
	}

	var builder strings.Builder
	var previousUnderscore bool
	for _, r := range strings.Trim(name, "/") {
		// 字母与数字原样保留，其余字符统一折叠为下划线。
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			previousUnderscore = false
			builder.WriteRune(unicode.ToLower(r))
			continue
		}
		if previousUnderscore {
			continue
		}
		previousUnderscore = true
		builder.WriteRune('_')
	}
	name = strings.Trim(builder.String(), "_")
	if name == "" {
		name = "base_api_tool"
	}

	count := toolNames[name]
	toolNames[name] = count + 1
	// 同名工具再次出现时追加序号，避免注册冲突。
	if count > 0 {
		name = fmt.Sprintf("%s_%d", name, count+1)
	}

	description := strings.TrimSpace(baseAPI.Desc)
	if description == "" {
		description = strings.TrimSpace(baseAPI.ServiceDesc)
	}
	if description == "" {
		description = strings.TrimSpace(baseAPI.Operation)
	}

	inputSchema := normalizeMcpInputSchema(baseAPI.InputSchema, openAPISchemas)
	return name, description, inputSchema
}

// mcpAPIPayload 构建接口元数据返回值。
func mcpAPIPayload(baseAPI *models.BaseAPI) map[string]any {
	return map[string]any{
		"id":            baseAPI.ID,
		"service_name":  baseAPI.ServiceName,
		"service_desc":  baseAPI.ServiceDesc,
		"desc":          baseAPI.Desc,
		"operation":     baseAPI.Operation,
		"method":        baseAPI.Method,
		"path":          baseAPI.Path,
		"arg_mapping":   parseJSONValue(baseAPI.ArgMapping),
		"output_schema": parseJSONValue(baseAPI.OutputSchema),
	}
}

// normalizeMcpHTTPEndpoint 将 HTTP 监听地址转换为 MCP 内部调用地址。
func normalizeMcpHTTPEndpoint(addr string) string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return ""
	}
	if strings.HasPrefix(addr, "http://") || strings.HasPrefix(addr, "https://") {
		return strings.TrimRight(addr, "/")
	}
	if strings.HasPrefix(addr, ":") {
		return "http://127.0.0.1" + addr
	}
	if strings.HasPrefix(addr, "0.0.0.0:") {
		return "http://127.0.0.1:" + strings.TrimPrefix(addr, "0.0.0.0:")
	}
	return "http://" + addr
}

// newMcpHTTPClient 创建 MCP 内部 HTTP 转发客户端。
func newMcpHTTPClient(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = mcpDefaultHTTPTimeout
	}
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   3 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   20,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   5 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
}

// ensureMcpBaseAPIEnabled 确认工具调用时接口仍处于 MCP 启用状态。
func ensureMcpBaseAPIEnabled(ctx context.Context, baseAPIRepo *data.BaseAPIRepository, baseAPI *models.BaseAPI) error {
	if baseAPIRepo == nil || baseAPI == nil || baseAPI.ID == 0 {
		return nil
	}
	query := baseAPIRepo.Query(ctx).BaseAPI
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.ID.Eq(baseAPI.ID)))
	currentAPI, err := baseAPIRepo.Find(ctx, opts...)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("MCP 工具 %s 已被禁用或不存在", baseAPI.Operation)
		}
		return fmt.Errorf("查询 MCP 工具状态失败: %w", err)
	}
	if currentAPI == nil || !currentAPI.McpEnabled {
		return fmt.Errorf("MCP 工具 %s 已被禁用或不存在", baseAPI.Operation)
	}
	return nil
}

// mcpArguments 将 MCP 原始参数转换为 map。
func mcpArguments(raw any) (map[string]any, error) {
	if raw == nil {
		return map[string]any{}, nil
	}
	if args, ok := raw.(map[string]any); ok {
		return args, nil
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	args := make(map[string]any)
	if err = json.Unmarshal(data, &args); err != nil {
		return nil, err
	}
	return args, nil
}

// parseMcpArgMappings 解析 base_api 中的参数位置映射。
func parseMcpArgMappings(value string) ([]mcpArgMappingItem, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	var mappings []mcpArgMappingItem
	if err := json.Unmarshal([]byte(value), &mappings); err != nil {
		return nil, fmt.Errorf("解析参数映射: %w", err)
	}
	return mappings, nil
}

// buildMcpHTTPRequest 根据参数映射构建 MCP 工具转发的 HTTP 请求。
func buildMcpHTTPRequest(ctx context.Context, baseAPI *models.BaseAPI, endpoint string, sourceHeader http.Header, mappings []mcpArgMappingItem, args map[string]any) (*http.Request, error) {
	requestPath := baseAPI.Path
	queryValues := url.Values{}
	var bodyValue any
	for _, mapping := range mappings {
		value, ok := args[mapping.Name]
		// 必填参数缺失时直接返回工具错误，避免调用后端得到不明确的失败。
		if !ok {
			if mapping.Required {
				return nil, fmt.Errorf("缺少必填参数 %s", mapping.Name)
			}
			continue
		}
		switch strings.ToLower(mapping.Position) {
		// path 参数需要替换到资源路径中。
		case mcpArgPositionPath:
			placeholder := "{" + mapping.Name + "}"
			if !strings.Contains(requestPath, placeholder) {
				return nil, fmt.Errorf("路径参数 %s 未出现在请求路径 %s 中", mapping.Name, baseAPI.Path)
			}
			requestPath = strings.ReplaceAll(requestPath, placeholder, url.PathEscape(valueToString(value)))
		// query 参数追加到 URL 查询串中。
		case mcpArgPositionQuery:
			switch typedValue := value.(type) {
			case []any:
				for _, item := range typedValue {
					queryValues.Add(mapping.Name, valueToString(item))
				}
			case []string:
				for _, item := range typedValue {
					queryValues.Add(mapping.Name, item)
				}
			default:
				queryValues.Add(mapping.Name, valueToString(value))
			}
		// body 参数只取第一项作为请求体，兼容 OpenAPI 的单 body 语义。
		case mcpArgPositionBody:
			if bodyValue == nil {
				bodyValue = value
			}
		case mcpArgPositionHeader:
			continue
		default:
			return nil, fmt.Errorf("不支持的参数位置 %s", mapping.Position)
		}
	}
	if hasUnresolvedMcpPathParam(requestPath) {
		return nil, fmt.Errorf("请求路径仍存在未替换参数 %s", requestPath)
	}

	baseURL, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	var relativeURL *url.URL
	relativeURL, err = url.Parse(requestPath)
	if err != nil {
		return nil, err
	}
	requestURL := baseURL.ResolveReference(relativeURL)
	requestURL.RawQuery = queryValues.Encode()

	method := strings.ToUpper(baseAPI.Method)
	if method == "" {
		return nil, fmt.Errorf("接口 %s 未配置 HTTP 方法", baseAPI.Operation)
	}
	var bodyReader io.Reader
	// GET 与 HEAD 请求不携带 body，其余方法默认以 JSON 转发工具参数。
	if method != http.MethodGet && method != http.MethodHead {
		if bodyValue == nil && len(args) > 0 {
			bodyValue = args
		}
		if bodyValue != nil {
			var bodyBytes []byte
			bodyBytes, err = json.Marshal(bodyValue)
			if err != nil {
				return nil, err
			}
			bodyReader = bytes.NewReader(bodyBytes)
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, requestURL.String(), bodyReader)
	if err != nil {
		return nil, err
	}
	// 存在请求体时显式标记 JSON，避免后端绑定参数失败。
	if bodyReader != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if authorization := sourceHeader.Get("Authorization"); authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	if requestID := sourceHeader.Get("X-Request-ID"); requestID != "" {
		req.Header.Set("X-Request-ID", requestID)
	}
	for _, mapping := range mappings {
		// header 映射允许通过工具参数显式传递请求头。
		if !strings.EqualFold(mapping.Position, mcpArgPositionHeader) {
			continue
		}
		value, ok := args[mapping.Name]
		if !ok {
			continue
		}
		req.Header.Set(mapping.Name, valueToString(value))
	}
	return req, nil
}

// hasUnresolvedMcpPathParam 判断请求路径是否仍包含未替换的 path 参数。
func hasUnresolvedMcpPathParam(path string) bool {
	return strings.Contains(path, "{") || strings.Contains(path, "}")
}

// valueToString 将工具参数转换为 HTTP 参数字符串。
func valueToString(value any) string {
	switch typedValue := value.(type) {
	case string:
		return typedValue
	case fmt.Stringer:
		return typedValue.String()
	default:
		data, err := json.Marshal(typedValue)
		if err != nil {
			return fmt.Sprint(typedValue)
		}
		return strings.Trim(string(data), `"`)
	}
}

// parseJSONValue 解析 JSON 字段，解析失败时返回原始字符串。
func parseJSONValue(value string) any {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	var result any
	if err := json.Unmarshal([]byte(value), &result); err != nil {
		return value
	}
	return result
}

// loadMcpOpenAPISchemas 从内嵌 OpenAPI 文档中读取组件 Schema。
func loadMcpOpenAPISchemas() map[string]any {
	var document map[string]any
	if err := yaml.Unmarshal(assets.OpenAPIData, &document); err != nil {
		log.Errorf("解析 OpenAPI 文档失败 err=%v", err)
		return nil
	}
	components, ok := stringMapValue(document["components"])
	if !ok {
		return nil
	}
	schemas, ok := stringMapValue(components["schemas"])
	if !ok {
		return nil
	}
	return normalizeMcpSchemaMap(schemas)
}

// normalizeMcpInputSchema 生成 MCP 可独立解析的入参 Schema。
func normalizeMcpInputSchema(inputSchema string, openAPISchemas map[string]any) string {
	inputSchema = strings.TrimSpace(inputSchema)
	// OpenAPI 未生成有效入参 Schema 时，使用空对象作为 MCP 默认入参。
	if inputSchema == "" || !json.Valid([]byte(inputSchema)) {
		return mcpDefaultInputSchema
	}

	var schema any
	if err := json.Unmarshal([]byte(inputSchema), &schema); err != nil {
		return mcpDefaultInputSchema
	}
	if len(openAPISchemas) > 0 {
		schema = expandMcpSchemaRefs(schema, openAPISchemas, make(map[string]bool), 0)
	}
	data, err := json.Marshal(schema)
	if err != nil || !json.Valid(data) {
		return mcpDefaultInputSchema
	}
	return string(data)
}

// expandMcpSchemaRefs 展开 OpenAPI components/schemas 引用，避免 MCP 客户端无法解析外部引用。
func expandMcpSchemaRefs(value any, openAPISchemas map[string]any, refStack map[string]bool, depth int) any {
	if depth > mcpMaxSchemaRefDepth {
		return value
	}
	switch typedValue := value.(type) {
	case map[string]any:
		if ref, ok := typedValue["$ref"].(string); ok {
			schemaName, found := strings.CutPrefix(ref, "#/components/schemas/")
			if found {
				schema, exists := openAPISchemas[schemaName]
				if !exists || refStack[schemaName] {
					return normalizeMcpSchemaValue(typedValue)
				}
				refStack[schemaName] = true
				expanded := expandMcpSchemaRefs(normalizeMcpSchemaValue(schema), openAPISchemas, refStack, depth+1)
				delete(refStack, schemaName)
				expandedMap, ok := expanded.(map[string]any)
				if !ok {
					return expanded
				}
				for key, item := range typedValue {
					if key == "$ref" {
						continue
					}
					expandedMap[key] = expandMcpSchemaRefs(item, openAPISchemas, refStack, depth+1)
				}
				return expandedMap
			}
		}
		result := make(map[string]any, len(typedValue))
		for key, item := range typedValue {
			result[key] = expandMcpSchemaRefs(item, openAPISchemas, refStack, depth+1)
		}
		return result
	case []any:
		result := make([]any, 0, len(typedValue))
		for _, item := range typedValue {
			result = append(result, expandMcpSchemaRefs(item, openAPISchemas, refStack, depth+1))
		}
		return result
	default:
		return typedValue
	}
}

// normalizeMcpSchemaMap 递归规范化 OpenAPI YAML 解码后的 Schema。
func normalizeMcpSchemaMap(value map[string]any) map[string]any {
	result := make(map[string]any, len(value))
	for key, item := range value {
		result[key] = normalizeMcpSchemaValue(item)
	}
	return result
}

// normalizeMcpSchemaValue 将 YAML map 转换为 JSON Schema 更易处理的 map[string]any。
func normalizeMcpSchemaValue(value any) any {
	switch typedValue := value.(type) {
	case map[string]any:
		return normalizeMcpSchemaMap(typedValue)
	case map[any]any:
		result := make(map[string]any, len(typedValue))
		for key, item := range typedValue {
			result[fmt.Sprint(key)] = normalizeMcpSchemaValue(item)
		}
		return result
	case []any:
		result := make([]any, 0, len(typedValue))
		for _, item := range typedValue {
			result = append(result, normalizeMcpSchemaValue(item))
		}
		return result
	default:
		return typedValue
	}
}

// stringMapValue 将任意 map 值转换为字符串键 map。
func stringMapValue(value any) (map[string]any, bool) {
	switch typedValue := value.(type) {
	case map[string]any:
		return typedValue, true
	case map[any]any:
		result := make(map[string]any, len(typedValue))
		for key, item := range typedValue {
			result[fmt.Sprint(key)] = item
		}
		return result, true
	default:
		return nil, false
	}
}
