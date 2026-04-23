package remote

import (
	"context"
	"fmt"
	stdhttp "net/http"
	"net/url"
	"strings"

	"shop/api/gen/go/conf"
	pkgQueue "shop/pkg/queue"

	client "github.com/gorse-io/gorse-go"
	_http "github.com/liujitcn/go-utils/http"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// Recommend 表示推荐系统基础客户端，统一承载公共配置和通用请求能力。
type Recommend struct {
	gorseClient *client.GorseClient
	entryPoint  string
	apiKey      string
	httpClient  *_http.Client
}

// NewRecommend 创建推荐系统基础客户端。
func NewRecommend(cfg *conf.Recommend) *Recommend {
	// 推荐配置缺失时，直接关闭推荐系统链路并走本地兜底。
	if cfg == nil {
		pkgQueue.SetRecommendEnabled(false)
		return &Recommend{}
	}

	entryPoint := strings.TrimSpace(cfg.GetEntryPoint())
	// 未配置入口地址时，直接关闭推荐系统链路并走本地兜底。
	if entryPoint == "" {
		pkgQueue.SetRecommendEnabled(false)
		return &Recommend{}
	}

	httpClientOptions := make([]_http.ClientOption, 0, 3)
	httpClientOptions = append(httpClientOptions, _http.WithBaseURL(entryPoint))
	httpClientOptions = append(httpClientOptions, _http.WithHTTPClient(&stdhttp.Client{
		Transport: otelhttp.NewTransport(stdhttp.DefaultTransport),
	}))
	// 当前配置了 API Key 时，通过默认请求头统一透传给 Gorse。
	if strings.TrimSpace(cfg.GetApiKey()) != "" {
		httpClientOptions = append(httpClientOptions, _http.WithDefaultHeader("X-API-Key", cfg.GetApiKey()))
	}

	pkgQueue.SetRecommendEnabled(true)
	return &Recommend{
		gorseClient: client.NewGorseClient(entryPoint, cfg.GetApiKey()),
		entryPoint:  entryPoint,
		apiKey:      cfg.GetApiKey(),
		httpClient:  _http.NewClient(httpClientOptions...),
	}
}

// Enabled 判断当前推荐系统基础客户端是否可用。
func (r *Recommend) Enabled() bool {
	return r.gorseClient != nil
}

// RequestJSON 通过远程推荐引擎原生 HTTP API 请求 JSON 内容。
func (r *Recommend) RequestJSON(ctx context.Context, method, path string, queries map[string]string, body string) ([]byte, error) {
	// 客户端未启用时，管理端无法继续代理远程推荐引擎请求。
	if !r.Enabled() || r.httpClient == nil || strings.TrimSpace(r.entryPoint) == "" {
		return nil, fmt.Errorf("remote recommend client is not enabled")
	}
	path = strings.TrimSpace(path)
	// 请求路径为空时，说明调用方未明确指定远程接口。
	if path == "" {
		return nil, fmt.Errorf("remote recommend request path is empty")
	}

	options := make([]_http.RequestOption, 0, len(queries)+3)
	options = append(options, _http.WithContext(ctx))
	for key, value := range queries {
		// 空查询名没有业务意义，直接跳过避免生成异常 query。
		if strings.TrimSpace(key) == "" {
			continue
		}
		options = append(options, _http.WithQuery(key, value))
	}
	// 携带请求体时，按 JSON 透传给远程推荐引擎。
	if strings.TrimSpace(body) != "" {
		options = append(options, _http.WithBodyString(body), _http.WithContentType("application/json"))
	}

	resp, err := r.httpClient.Do(method, path, options...)
	if err != nil {
		return nil, err
	}
	// 远端返回非 2xx 状态码时，带上响应体方便排查接口与配置问题。
	if resp.StatusCode < stdhttp.StatusOK || resp.StatusCode >= stdhttp.StatusMultipleChoices {
		return nil, fmt.Errorf("remote recommend request failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(resp.String()))
	}
	// 成功响应体为空时，统一返回空对象，方便管理端 JSON 面板展示。
	if len(resp.Body) == 0 {
		return []byte("{}"), nil
	}
	return resp.Body, nil
}

// EscapePathSegment 转义远程推荐引擎路径片段。
func EscapePathSegment(value string) string {
	return url.PathEscape(strings.TrimSpace(value))
}

// requestScores 通过原始 HTTP API 请求评分列表结果。
func (r *Recommend) requestScores(ctx context.Context, path string) ([]client.Score, error) {
	// 客户端未启用、入口地址缺失或请求路径为空时，直接返回空结果。
	if !r.Enabled() || r.httpClient == nil || strings.TrimSpace(r.entryPoint) == "" || strings.TrimSpace(path) == "" {
		return []client.Score{}, nil
	}

	resp, err := r.httpClient.Do(stdhttp.MethodGet, path, _http.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	// 远端返回非成功状态码时，直接抛出响应体内容，方便业务侧定位配置或路径问题。
	if resp.StatusCode != stdhttp.StatusOK {
		return nil, fmt.Errorf("gorse request failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(resp.String()))
	}

	scores := make([]client.Score, 0)
	// 远端请求成功但响应为空时，直接返回空评分列表。
	if len(resp.Body) == 0 {
		return scores, nil
	}
	err = resp.DecodeJSON(&scores)
	if err != nil {
		return nil, err
	}
	return scores, nil
}
