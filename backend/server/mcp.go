package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"unicode"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/liujitcn/gorm-kit/repository"
	"github.com/liujitcn/kratos-kit/bootstrap"
	mcpServer "github.com/liujitcn/kratos-kit/transport/mcp"
	"github.com/mark3labs/mcp-go/mcp"
	mcpGoServer "github.com/mark3labs/mcp-go/server"

	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
)

const (
	mcpDefaultInputSchema = `{"type":"object","properties":{}}`

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

// NewMcpServer 创建 MCP Server，并从 base_api 表加载已启用的工具。
func NewMcpServer(ctx *bootstrap.Context, baseAPIRepo *data.BaseAPIRepository) (*mcpServer.Server, error) {
	cfg := ctx.GetConfig()
	// 未配置 MCP 时不创建对应传输服务。
	if cfg == nil || cfg.Server == nil || cfg.Server.Mcp == nil {
		return nil, nil
	}

	appInfo := ctx.GetAppInfo()
	options := []mcpServer.ServerOption{
		mcpServer.WithMCPServerOptions(
			mcpGoServer.WithToolCapabilities(false),
			mcpGoServer.WithRecovery(),
		),
	}
	// 应用信息存在时，同步作为 MCP 服务元信息。
	if appInfo != nil {
		if appInfo.GetName() != "" {
			options = append(options, mcpServer.WithServerName(appInfo.GetName()))
		}
		if appInfo.GetVersion() != "" {
			options = append(options, mcpServer.WithServerVersion(appInfo.GetVersion()))
		}
	}

	srv := mcpServer.NewServer(options...)
	// 仓储未注入时只返回空 MCP 服务，避免影响应用启动。
	if baseAPIRepo == nil {
		return srv, nil
	}

	apiList, err := listMcpBaseAPIs(context.Background(), baseAPIRepo)
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
		toolName, description, inputSchema := buildMcpToolDefinition(baseAPI, toolNames)
		currentAPI := baseAPI
		err = srv.RegisterHandlerWithJsonSchema(
			toolName,
			description,
			inputSchema,
			func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				return buildMcpToolResult(ctx, currentAPI, httpEndpoint, request)
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

// listMcpBaseAPIs 查询 base_api 中启用 MCP 的接口。
func listMcpBaseAPIs(ctx context.Context, baseAPIRepo *data.BaseAPIRepository) ([]*models.BaseAPI, error) {
	query := baseAPIRepo.Query(ctx).BaseAPI
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.McpEnabled.Is(true)))
	opts = append(opts, repository.Order(query.ServiceName.Asc(), query.Operation.Asc()))
	return baseAPIRepo.List(ctx, opts...)
}

// buildMcpToolResult 构建 MCP 工具调用结果。
func buildMcpToolResult(ctx context.Context, baseAPI *models.BaseAPI, httpEndpoint string, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	apiPayload := mcpAPIPayload(baseAPI)
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
	mappings := parseMcpArgMappings(baseAPI.ArgMapping)
	var req *http.Request
	req, err = buildMcpHTTPRequest(ctx, baseAPI, httpEndpoint, request.Header, mappings, args)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var resp *http.Response
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.Errorf("关闭 MCP HTTP 响应失败 err=%v", closeErr)
		}
	}()

	var responseBytes []byte
	responseBytes, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
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
func buildMcpToolDefinition(baseAPI *models.BaseAPI, toolNames map[string]int) (string, string, string) {
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

	inputSchema := strings.TrimSpace(baseAPI.InputSchema)
	// OpenAPI 未生成有效入参 Schema 时，使用空对象作为 MCP 默认入参。
	if inputSchema == "" || !json.Valid([]byte(inputSchema)) {
		inputSchema = mcpDefaultInputSchema
	}
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
func parseMcpArgMappings(value string) []mcpArgMappingItem {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	var mappings []mcpArgMappingItem
	if err := json.Unmarshal([]byte(value), &mappings); err != nil {
		return nil
	}
	return mappings
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
			requestPath = strings.ReplaceAll(requestPath, "{"+mapping.Name+"}", url.PathEscape(valueToString(value)))
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
		}
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
