package gorse

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	stdhttp "net/http"
	"net/url"
	"strconv"
	"strings"

	"shop/pkg/recommend/dto"

	client "github.com/gorse-io/gorse-go"
)

// Dashboard 表示 Gorse 推荐仪表盘 API 客户端。
type Dashboard struct {
	recommend *Recommend
}

// NewDashboard 创建 Gorse 推荐仪表盘 API 客户端。
func NewDashboard(recommend *Recommend) *Dashboard {
	return &Dashboard{recommend: recommend}
}

// Config 查询 Gorse 推荐仪表盘配置。
func (d *Dashboard) Config(ctx context.Context) ([]byte, error) {
	return d.recommend.RequestJSON(ctx, stdhttp.MethodGet, "/api/dashboard/config", nil, "")
}

// SaveConfig 保存 Gorse 推荐仪表盘配置。
func (d *Dashboard) SaveConfig(ctx context.Context, body string) ([]byte, error) {
	return d.recommend.RequestJSON(ctx, stdhttp.MethodPost, "/api/dashboard/config", nil, body)
}

// ResetConfig 重置 Gorse 推荐仪表盘配置。
func (d *Dashboard) ResetConfig(ctx context.Context) ([]byte, error) {
	return d.recommend.RequestJSON(ctx, stdhttp.MethodDelete, "/api/dashboard/config", nil, "")
}

// ExternalPreview 预览 Gorse 推荐外部推荐脚本。
func (d *Dashboard) ExternalPreview(ctx context.Context, userID, script string) ([]byte, error) {
	queries := make(map[string]string, 2)
	queries["user-id"] = strings.TrimSpace(userID)
	// Gorse 原生接口要求脚本通过 base64 查询参数传入，Admin 侧请求体在这里统一转换。
	queries["script"] = base64.StdEncoding.EncodeToString([]byte(script))
	return d.recommend.RequestJSON(ctx, stdhttp.MethodGet, "/api/dashboard/external", queries, "")
}

// RankerPrompt 预览 Gorse 推荐排序提示词。
func (d *Dashboard) RankerPrompt(ctx context.Context, userID, queryTemplate, documentTemplate string) ([]byte, error) {
	queries := make(map[string]string, 3)
	queries["user-id"] = strings.TrimSpace(userID)
	// Gorse 原生接口要求模板通过 base64 查询参数传入，Admin 侧请求体在这里统一转换。
	queries["query-template"] = base64.StdEncoding.EncodeToString([]byte(queryTemplate))
	queries["document-template"] = base64.StdEncoding.EncodeToString([]byte(documentTemplate))
	return d.recommend.RequestJSON(ctx, stdhttp.MethodGet, "/api/dashboard/ranker/prompt", queries, "")
}

// Tasks 查询 Gorse 推荐仪表盘任务状态。
func (d *Dashboard) Tasks(ctx context.Context) ([]byte, error) {
	return d.recommend.RequestJSON(ctx, stdhttp.MethodGet, "/api/dashboard/tasks", nil, "")
}

// TimeSeries 查询 Gorse 推荐仪表盘时间序列。
func (d *Dashboard) TimeSeries(ctx context.Context, name, begin, end string) ([]byte, error) {
	queries := make(map[string]string, 2)
	// 调用方传入开始时间时，透传给 Gorse 仪表盘 API 限定统计窗口。
	if strings.TrimSpace(begin) != "" {
		queries["begin"] = begin
	}
	// 调用方传入结束时间时，透传给 Gorse 仪表盘 API 限定统计窗口。
	if strings.TrimSpace(end) != "" {
		queries["end"] = end
	}
	path := "/api/dashboard/timeseries/" + url.PathEscape(name)
	return d.recommend.RequestJSON(ctx, stdhttp.MethodGet, path, queries, "")
}

// Categories 查询 Gorse 推荐仪表盘分类列表。
func (d *Dashboard) Categories(ctx context.Context) ([]byte, error) {
	return d.recommend.RequestJSON(ctx, stdhttp.MethodGet, "/api/dashboard/categories", nil, "")
}

// DashboardItems 查询 Gorse 推荐仪表盘推荐商品列表。
func (d *Dashboard) DashboardItems(ctx context.Context, recommender, category string, end int32) ([]byte, error) {
	recommender = strings.Trim(strings.TrimSpace(recommender), "/")
	// 推荐器为空时，默认回退到Gorse 仪表盘 latest 推荐器。
	if recommender == "" {
		recommender = "latest"
	}

	queries := make(map[string]string, 2)
	// 分类不为空时，透传分类筛选条件。
	if strings.TrimSpace(category) != "" {
		queries["category"] = strings.TrimSpace(category)
	}
	// 结束位置大于零时，限制返回数量。
	if end > 0 {
		queries["end"] = fmt.Sprintf("%d", end)
	}
	path, err := buildDashboardRecommenderPath("/api/dashboard", recommender, false)
	if err != nil {
		return nil, err
	}
	return d.recommend.RequestJSON(ctx, stdhttp.MethodGet, path, queries, "")
}

// Users 查询 Gorse 推荐仪表盘用户列表。
func (d *Dashboard) Users(ctx context.Context, cursor string, n int32) ([]byte, error) {
	queries := make(map[string]string, 2)
	queries["cursor"] = strings.TrimSpace(cursor)
	// 请求数量非法时，沿用Gorse 仪表盘列表默认每页 10 条。
	if n <= 0 {
		n = 10
	}
	queries["n"] = strconv.FormatInt(int64(n), 10)
	return d.recommend.RequestJSON(ctx, stdhttp.MethodGet, "/api/dashboard/users", queries, "")
}

// User 查询 Gorse 推荐仪表盘用户详情。
func (d *Dashboard) User(ctx context.Context, id string) ([]byte, error) {
	id = strings.TrimSpace(id)
	// 用户编号为空时，无法拼装Gorse 仪表盘用户详情路径。
	if id == "" {
		return nil, fmt.Errorf("gorse dashboard user id is empty")
	}
	return d.recommend.RequestJSON(ctx, stdhttp.MethodGet, "/api/dashboard/user/"+url.PathEscape(id), nil, "")
}

// DeleteUser 删除 Gorse 推荐用户。
func (d *Dashboard) DeleteUser(ctx context.Context, id string) ([]byte, error) {
	id = strings.TrimSpace(id)
	// 用户编号为空时，无法拼装Gorse 推荐删除用户路径。
	if id == "" {
		return nil, fmt.Errorf("gorse dashboard user id is empty")
	}
	return d.recommend.RequestJSON(ctx, stdhttp.MethodDelete, "/api/user/"+url.PathEscape(id), nil, "")
}

// UserSimilar 查询 Gorse 推荐相似用户。
func (d *Dashboard) UserSimilar(ctx context.Context, id, recommender string) ([]byte, error) {
	id = strings.TrimSpace(id)
	// 用户编号为空时，无法拼装Gorse 仪表盘相似用户路径。
	if id == "" {
		return nil, fmt.Errorf("gorse dashboard user id is empty")
	}
	recommender = strings.Trim(strings.TrimSpace(recommender), "/")
	// 推荐器为空时，默认使用当前商城配置中的 similar_users。
	if recommender == "" {
		recommender = "similar_users"
	}
	// 相似用户推荐器只允许单段名称，避免拼装出非预期仪表盘路径。
	if strings.Contains(recommender, "/") || strings.Contains(recommender, "..") {
		return nil, fmt.Errorf("gorse dashboard user recommender is invalid")
	}
	path := "/api/dashboard/user-to-user/" + url.PathEscape(recommender) + "/" + url.PathEscape(id)
	return d.recommend.RequestJSON(ctx, stdhttp.MethodGet, path, nil, "")
}

// UserFeedback 查询 Gorse 推荐用户反馈。
func (d *Dashboard) UserFeedback(ctx context.Context, id, feedbackType string, offset, n int32) ([]byte, error) {
	id = strings.TrimSpace(id)
	// 用户编号为空时，无法拼装Gorse 仪表盘用户反馈路径。
	if id == "" {
		return nil, fmt.Errorf("gorse dashboard user id is empty")
	}
	queries := make(map[string]string, 2)
	// 偏移量小于零没有业务意义，统一修正为首页。
	if offset < 0 {
		offset = 0
	}
	queries["offset"] = strconv.FormatInt(int64(offset), 10)
	// 请求数量非法时，沿用Gorse 仪表盘反馈列表默认每页 10 条。
	if n <= 0 {
		n = 10
	}
	queries["n"] = strconv.FormatInt(int64(n), 10)

	feedbackType = strings.Trim(strings.TrimSpace(feedbackType), "/")
	// 反馈类型为空时，必须保留尾部斜杠，匹配Gorse 仪表盘原始接口语义。
	if feedbackType == "" {
		return d.recommend.RequestJSON(ctx, stdhttp.MethodGet, "/api/dashboard/user/"+url.PathEscape(id)+"/feedback/", queries, "")
	}
	// 反馈类型只允许单段名称，避免拼装出非预期仪表盘路径。
	if strings.Contains(feedbackType, "/") || strings.Contains(feedbackType, "..") {
		return nil, fmt.Errorf("gorse dashboard feedback type is invalid")
	}
	path := "/api/dashboard/user/" + url.PathEscape(id) + "/feedback/" + url.PathEscape(feedbackType)
	return d.recommend.RequestJSON(ctx, stdhttp.MethodGet, path, queries, "")
}

// UserRecommend 查询 Gorse 推荐用户推荐结果。
func (d *Dashboard) UserRecommend(ctx context.Context, id, recommender, category string, n int32) ([]byte, error) {
	id = strings.TrimSpace(id)
	// 用户编号为空时，无法拼装Gorse 仪表盘用户推荐路径。
	if id == "" {
		return nil, fmt.Errorf("gorse dashboard user id is empty")
	}
	queries := make(map[string]string, 2)
	// 请求数量非法时，沿用Gorse 仪表盘推荐列表默认 100 条。
	if n <= 0 {
		n = 100
	}
	queries["n"] = strconv.FormatInt(int64(n), 10)
	// 分类不为空时，透传分类筛选条件。
	if strings.TrimSpace(category) != "" {
		queries["category"] = strings.TrimSpace(category)
	}

	recommender = strings.Trim(strings.TrimSpace(recommender), "/")
	// 推荐器为空时，必须保留尾部斜杠，匹配Gorse 仪表盘原始接口语义。
	if recommender == "" {
		return d.recommend.RequestJSON(ctx, stdhttp.MethodGet, "/api/dashboard/recommend/"+url.PathEscape(id)+"/", queries, "")
	}
	path, err := buildDashboardRecommenderPath("/api/dashboard/recommend/"+url.PathEscape(id), recommender, false)
	if err != nil {
		return nil, err
	}
	return d.recommend.RequestJSON(ctx, stdhttp.MethodGet, path, queries, "")
}

// Items 查询 Gorse 推荐商品列表。
func (d *Dashboard) Items(ctx context.Context, cursor string, n int32) ([]byte, error) {
	queries := make(map[string]string, 2)
	queries["cursor"] = strings.TrimSpace(cursor)
	// 请求数量非法时，沿用Gorse 仪表盘列表默认每页 10 条。
	if n <= 0 {
		n = 10
	}
	queries["n"] = strconv.FormatInt(int64(n), 10)
	return d.recommend.RequestJSON(ctx, stdhttp.MethodGet, "/api/items", queries, "")
}

// Item 查询 Gorse 推荐商品详情。
func (d *Dashboard) Item(ctx context.Context, id string) ([]byte, error) {
	id = strings.TrimSpace(id)
	// 商品编号为空时，无法拼装Gorse 推荐商品详情路径。
	if id == "" {
		return nil, fmt.Errorf("gorse dashboard item id is empty")
	}
	return d.recommend.RequestJSON(ctx, stdhttp.MethodGet, "/api/item/"+url.PathEscape(id), nil, "")
}

// DeleteItem 删除 Gorse 推荐商品。
func (d *Dashboard) DeleteItem(ctx context.Context, id string) ([]byte, error) {
	id = strings.TrimSpace(id)
	// 商品编号为空时，无法拼装Gorse 推荐删除商品路径。
	if id == "" {
		return nil, fmt.Errorf("gorse dashboard item id is empty")
	}
	return d.recommend.RequestJSON(ctx, stdhttp.MethodDelete, "/api/item/"+url.PathEscape(id), nil, "")
}

// ItemSimilar 查询 Gorse 推荐相似商品。
func (d *Dashboard) ItemSimilar(ctx context.Context, id, recommender, category string) ([]byte, error) {
	id = strings.TrimSpace(id)
	// 商品编号为空时，无法拼装Gorse 仪表盘相似商品路径。
	if id == "" {
		return nil, fmt.Errorf("gorse dashboard item id is empty")
	}
	recommender = strings.Trim(strings.TrimSpace(recommender), "/")
	// 推荐器为空时，默认使用当前商城配置中的 goods_relation。
	if recommender == "" {
		recommender = "goods_relation"
	}
	// 相似商品推荐器只允许单段名称，避免拼装出非预期仪表盘路径。
	if strings.Contains(recommender, "/") || strings.Contains(recommender, "..") {
		return nil, fmt.Errorf("gorse dashboard item recommender is invalid")
	}
	queries := make(map[string]string, 1)
	// 分类不为空时，透传分类筛选条件。
	if strings.TrimSpace(category) != "" {
		queries["category"] = strings.TrimSpace(category)
	}
	path := "/api/dashboard/item-to-item/" + url.PathEscape(recommender) + "/" + url.PathEscape(id)
	return d.recommend.RequestJSON(ctx, stdhttp.MethodGet, path, queries, "")
}

// UsersRaw 查询 Gorse 推荐原始用户列表。
func (d *Dashboard) UsersRaw(ctx context.Context, cursor string, n int32) (*client.UserIterator, error) {
	// 客户端未启用时，无法继续查询 Gorse 推荐原始用户列表。
	if !d.recommend.Enabled() {
		return nil, fmt.Errorf("gorse recommend client is not enabled")
	}
	// 请求数量非法时，沿用Gorse 推荐用户列表默认每页 100 条。
	if n <= 0 {
		n = 100
	}

	result, err := d.recommend.gorseClient.GetUsers(ctx, int(n), strings.TrimSpace(cursor))
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ItemsRaw 查询 Gorse 推荐原始商品列表。
func (d *Dashboard) ItemsRaw(ctx context.Context, cursor string, n int32) (*client.ItemIterator, error) {
	// 客户端未启用时，无法继续查询 Gorse 推荐原始商品列表。
	if !d.recommend.Enabled() {
		return nil, fmt.Errorf("gorse recommend client is not enabled")
	}
	// 请求数量非法时，沿用Gorse 推荐商品列表默认每页 100 条。
	if n <= 0 {
		n = 100
	}

	result, err := d.recommend.gorseClient.GetItems(ctx, int(n), strings.TrimSpace(cursor))
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Feedbacks 查询 Gorse 推荐反馈列表。
func (d *Dashboard) Feedbacks(ctx context.Context, cursor string, n int32) (*dto.FeedbackIterator, error) {
	queries := make(map[string]string, 2)
	queries["cursor"] = strings.TrimSpace(cursor)
	// 请求数量非法时，沿用Gorse 推荐反馈列表默认每页 100 条。
	if n <= 0 {
		n = 100
	}
	queries["n"] = strconv.FormatInt(int64(n), 10)

	data, err := d.recommend.RequestJSON(ctx, stdhttp.MethodGet, "/api/feedback", queries, "")
	if err != nil {
		return nil, err
	}

	result := &dto.FeedbackIterator{}
	err = json.Unmarshal(data, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// InsertUsers 批量导入 Gorse 推荐用户。
func (d *Dashboard) InsertUsers(ctx context.Context, users []client.User) error {
	// 客户端未启用时，无法继续导入 Gorse 推荐用户。
	if !d.recommend.Enabled() {
		return fmt.Errorf("gorse recommend client is not enabled")
	}
	// 待导入用户为空时，无需继续请求 Gorse 推荐服务。
	if len(users) == 0 {
		return nil
	}
	_, err := d.recommend.gorseClient.InsertUsers(ctx, users)
	if err != nil {
		return err
	}
	return nil
}

// InsertItems 批量导入 Gorse 推荐商品。
func (d *Dashboard) InsertItems(ctx context.Context, items []client.Item) error {
	// 客户端未启用时，无法继续导入 Gorse 推荐商品。
	if !d.recommend.Enabled() {
		return fmt.Errorf("gorse recommend client is not enabled")
	}
	// 待导入商品为空时，无需继续请求 Gorse 推荐服务。
	if len(items) == 0 {
		return nil
	}
	_, err := d.recommend.gorseClient.InsertItems(ctx, items)
	if err != nil {
		return err
	}
	return nil
}

// InsertFeedbacks 批量导入 Gorse 推荐反馈。
func (d *Dashboard) InsertFeedbacks(ctx context.Context, feedbackList []client.Feedback) error {
	// 客户端未启用时，无法继续导入 Gorse 推荐反馈。
	if !d.recommend.Enabled() {
		return fmt.Errorf("gorse recommend client is not enabled")
	}
	// 待导入反馈为空时，无需继续请求 Gorse 推荐服务。
	if len(feedbackList) == 0 {
		return nil
	}
	_, err := d.recommend.gorseClient.InsertFeedback(ctx, feedbackList)
	if err != nil {
		return err
	}
	return nil
}

// buildDashboardRecommenderPath 构建Gorse 推荐 dashboard 推荐器路径。
func buildDashboardRecommenderPath(basePath, recommender string, keepTrailingSlash bool) (string, error) {
	recommender = strings.Trim(strings.TrimSpace(recommender), "/")
	// 推荐器为空时，根据调用方语义决定是否保留尾部斜杠。
	if recommender == "" {
		// 需要兼容Gorse 仪表盘原始接口尾斜杠语义时，保留 basePath 末尾斜杠。
		if keepTrailingSlash {
			return basePath + "/", nil
		}
		return basePath, nil
	}

	segments := strings.Split(recommender, "/")
	encodedSegments := make([]string, 0, len(segments))
	for _, segment := range segments {
		segment = strings.TrimSpace(segment)
		// 路径段为空或带有目录穿越语义时，直接拒绝代理请求。
		if segment == "" || segment == "." || segment == ".." {
			return "", fmt.Errorf("gorse dashboard recommender is invalid")
		}
		encodedSegments = append(encodedSegments, url.PathEscape(segment))
	}
	return basePath + "/" + strings.Join(encodedSegments, "/"), nil
}
