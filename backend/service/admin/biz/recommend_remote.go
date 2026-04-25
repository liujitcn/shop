package biz

import (
	"context"
	"encoding/json"
	stdhttp "net/http"
	"strconv"
	"strings"

	adminApi "shop/api/gen/go/admin"
	"shop/pkg/errorsx"
	"shop/pkg/recommend/remote"
)

const (
	recommendRemoteDefaultListSize   = int64(20)
	recommendRemoteDefaultExportSize = int64(100)
)

// RecommendRemoteCase 远程推荐管理业务实例。
type RecommendRemoteCase struct {
	recommend *remote.Recommend
}

// NewRecommendRemoteCase 创建远程推荐管理业务实例。
func NewRecommendRemoteCase(recommend *remote.Recommend) *RecommendRemoteCase {
	return &RecommendRemoteCase{
		recommend: recommend,
	}
}

// GetRecommendRemoteOverview 查询远程推荐概览。
func (c *RecommendRemoteCase) GetRecommendRemoteOverview(ctx context.Context) (*adminApi.RecommendRemoteJsonResponse, error) {
	return c.requestJSON(ctx, stdhttp.MethodGet, "/api/dashboard/stats", nil, "")
}

// GetRecommendRemoteTasks 查询远程推荐任务状态。
func (c *RecommendRemoteCase) GetRecommendRemoteTasks(ctx context.Context) (*adminApi.RecommendRemoteJsonResponse, error) {
	return c.requestJSON(ctx, stdhttp.MethodGet, "/api/dashboard/tasks", nil, "")
}

// GetRecommendRemoteCategories 查询远程推荐分类。
func (c *RecommendRemoteCase) GetRecommendRemoteCategories(ctx context.Context) (*adminApi.RecommendRemoteJsonResponse, error) {
	return c.requestJSON(ctx, stdhttp.MethodGet, "/api/dashboard/categories", nil, "")
}

// GetRecommendRemoteTimeseries 查询远程推荐时间序列。
func (c *RecommendRemoteCase) GetRecommendRemoteTimeseries(ctx context.Context, req *adminApi.RecommendRemoteNameRequest) (*adminApi.RecommendRemoteJsonResponse, error) {
	name, err := c.requireName(req)
	if err != nil {
		return nil, err
	}
	path := "/api/dashboard/timeseries/" + remote.EscapePathSegment(name)
	return c.requestJSON(ctx, stdhttp.MethodGet, path, c.buildTimeseriesQueries(req), "")
}

// GetRecommendRemoteDashboardItems 查询远程推荐仪表盘推荐商品。
func (c *RecommendRemoteCase) GetRecommendRemoteDashboardItems(ctx context.Context, req *adminApi.RecommendRemoteDashboardItemsRequest) (*adminApi.RecommendRemoteJsonResponse, error) {
	recommender, err := c.requireRecommender(req)
	if err != nil {
		return nil, err
	}
	path := "/api/dashboard/" + c.escapeDashboardRecommender(recommender)
	return c.requestJSONWithLastModified(ctx, stdhttp.MethodGet, path, c.buildDashboardItemsQueries(req), "")
}

// GetRecommendRemoteRecommendations 查询远程推荐结果。
func (c *RecommendRemoteCase) GetRecommendRemoteRecommendations(ctx context.Context, req *adminApi.RecommendRemoteRecommendRequest) (*adminApi.RecommendRemoteJsonResponse, error) {
	path, err := c.resolveRecommendPath(req)
	if err != nil {
		return nil, err
	}
	return c.requestJSON(ctx, stdhttp.MethodGet, path, c.buildRecommendQueries(req), "")
}

// GetRecommendRemoteNeighbors 查询远程相似内容。
func (c *RecommendRemoteCase) GetRecommendRemoteNeighbors(ctx context.Context, req *adminApi.RecommendRemoteNeighborRequest) (*adminApi.RecommendRemoteJsonResponse, error) {
	path, err := c.resolveNeighborPath(req)
	if err != nil {
		return nil, err
	}
	return c.requestJSON(ctx, stdhttp.MethodGet, path, c.buildNeighborQueries(req), "")
}

// PageRecommendRemoteFeedback 查询远程推荐反馈列表。
func (c *RecommendRemoteCase) PageRecommendRemoteFeedback(ctx context.Context, req *adminApi.RecommendRemoteFeedbackRequest) (*adminApi.RecommendRemoteJsonResponse, error) {
	path, err := c.resolveFeedbackPath(req)
	if err != nil {
		return nil, err
	}
	return c.requestJSON(ctx, stdhttp.MethodGet, path, c.buildFeedbackQueries(req), "")
}

// ImportRecommendRemoteFeedback 写入远程推荐反馈。
func (c *RecommendRemoteCase) ImportRecommendRemoteFeedback(ctx context.Context, req *adminApi.RecommendRemoteJsonRequest) error {
	body, err := c.normalizeJSONBody(req.GetJson())
	if err != nil {
		return err
	}
	return c.requestNoContent(ctx, stdhttp.MethodPost, "/api/feedback", nil, body)
}

// DeleteRecommendRemoteFeedback 删除远程推荐反馈。
func (c *RecommendRemoteCase) DeleteRecommendRemoteFeedback(ctx context.Context, req *adminApi.RecommendRemoteFeedbackDeleteRequest) error {
	path, err := c.resolveFeedbackDeletePath(req)
	if err != nil {
		return err
	}
	return c.requestNoContent(ctx, stdhttp.MethodDelete, path, nil, "")
}

// PageRecommendRemoteUsers 查询远程推荐用户列表。
func (c *RecommendRemoteCase) PageRecommendRemoteUsers(ctx context.Context, req *adminApi.RecommendRemoteCursorRequest) (*adminApi.RecommendRemoteJsonResponse, error) {
	return c.requestJSON(ctx, stdhttp.MethodGet, "/api/users", c.buildCursorQueries(req, recommendRemoteDefaultListSize), "")
}

// GetRecommendRemoteUser 查询远程推荐用户。
func (c *RecommendRemoteCase) GetRecommendRemoteUser(ctx context.Context, req *adminApi.RecommendRemoteIdRequest) (*adminApi.RecommendRemoteJsonResponse, error) {
	id, err := c.requireId(req)
	if err != nil {
		return nil, err
	}
	path := "/api/user/" + remote.EscapePathSegment(id)
	return c.requestJSON(ctx, stdhttp.MethodGet, path, nil, "")
}

// DeleteRecommendRemoteUser 删除远程推荐用户。
func (c *RecommendRemoteCase) DeleteRecommendRemoteUser(ctx context.Context, req *adminApi.RecommendRemoteIdRequest) error {
	id, err := c.requireId(req)
	if err != nil {
		return err
	}
	path := "/api/user/" + remote.EscapePathSegment(id)
	return c.requestNoContent(ctx, stdhttp.MethodDelete, path, nil, "")
}

// PageRecommendRemoteItems 查询远程推荐商品列表。
func (c *RecommendRemoteCase) PageRecommendRemoteItems(ctx context.Context, req *adminApi.RecommendRemoteCursorRequest) (*adminApi.RecommendRemoteJsonResponse, error) {
	return c.requestJSON(ctx, stdhttp.MethodGet, "/api/items", c.buildCursorQueries(req, recommendRemoteDefaultListSize), "")
}

// GetRecommendRemoteItem 查询远程推荐商品。
func (c *RecommendRemoteCase) GetRecommendRemoteItem(ctx context.Context, req *adminApi.RecommendRemoteIdRequest) (*adminApi.RecommendRemoteJsonResponse, error) {
	id, err := c.requireId(req)
	if err != nil {
		return nil, err
	}
	path := "/api/item/" + remote.EscapePathSegment(id)
	return c.requestJSON(ctx, stdhttp.MethodGet, path, nil, "")
}

// DeleteRecommendRemoteItem 删除远程推荐商品。
func (c *RecommendRemoteCase) DeleteRecommendRemoteItem(ctx context.Context, req *adminApi.RecommendRemoteIdRequest) error {
	id, err := c.requireId(req)
	if err != nil {
		return err
	}
	path := "/api/item/" + remote.EscapePathSegment(id)
	return c.requestNoContent(ctx, stdhttp.MethodDelete, path, nil, "")
}

// ExportRecommendRemoteData 导出远程推荐数据。
func (c *RecommendRemoteCase) ExportRecommendRemoteData(ctx context.Context, req *adminApi.RecommendRemoteDataRequest) (*adminApi.RecommendRemoteJsonResponse, error) {
	path, err := c.resolveDataPath(req.GetType())
	if err != nil {
		return nil, err
	}
	return c.requestJSON(ctx, stdhttp.MethodGet, path, c.buildDataQueries(req), "")
}

// ImportRecommendRemoteData 导入远程推荐数据。
func (c *RecommendRemoteCase) ImportRecommendRemoteData(ctx context.Context, req *adminApi.RecommendRemoteImportRequest) error {
	path, err := c.resolveDataPath(req.GetType())
	if err != nil {
		return err
	}
	var body string
	body, err = c.normalizeJSONBody(req.GetJson())
	if err != nil {
		return err
	}
	return c.requestNoContent(ctx, stdhttp.MethodPost, path, nil, body)
}

// PurgeRecommendRemoteData 清空远程推荐数据。
func (c *RecommendRemoteCase) PurgeRecommendRemoteData(ctx context.Context, req *adminApi.RecommendRemotePurgeRequest) error {
	checkList := c.normalizePurgeCheckList(req.GetCheckList())
	// 未完成全部确认项时，拒绝代理危险清空请求。
	if len(checkList) != 4 {
		return errorsx.InvalidArgument("请先确认清空用户、商品、反馈和缓存数据")
	}
	queries := map[string]string{
		"check_list": strings.Join(checkList, ","),
	}
	return c.requestNoContent(ctx, stdhttp.MethodPost, "/api/purge", queries, "")
}

// GetRecommendRemoteFlowConfig 查询推荐编排配置。
func (c *RecommendRemoteCase) GetRecommendRemoteFlowConfig(ctx context.Context) (*adminApi.RecommendRemoteJsonResponse, error) {
	return c.requestJSON(ctx, stdhttp.MethodGet, "/api/dashboard/config", nil, "")
}

// SaveRecommendRemoteFlowConfig 保存推荐编排配置。
func (c *RecommendRemoteCase) SaveRecommendRemoteFlowConfig(ctx context.Context, req *adminApi.RecommendRemoteJsonRequest) error {
	body, err := c.normalizeJSONBody(req.GetJson())
	if err != nil {
		return err
	}
	return c.requestNoContent(ctx, stdhttp.MethodPost, "/api/dashboard/config", nil, body)
}

// ResetRecommendRemoteFlowConfig 重置推荐编排配置。
func (c *RecommendRemoteCase) ResetRecommendRemoteFlowConfig(ctx context.Context) error {
	return c.requestNoContent(ctx, stdhttp.MethodDelete, "/api/dashboard/config", nil, "")
}

// GetRecommendRemoteFlowSchema 查询推荐编排配置结构。
func (c *RecommendRemoteCase) GetRecommendRemoteFlowSchema(ctx context.Context) (*adminApi.RecommendRemoteJsonResponse, error) {
	return c.requestJSON(ctx, stdhttp.MethodGet, "/api/dashboard/config/schema", nil, "")
}

// GetRecommendRemoteConfig 查询远程推荐配置。
func (c *RecommendRemoteCase) GetRecommendRemoteConfig(ctx context.Context) (*adminApi.RecommendRemoteJsonResponse, error) {
	return c.requestJSON(ctx, stdhttp.MethodGet, "/api/dashboard/config", nil, "")
}

// requestJSON 请求远程推荐引擎并返回 JSON 字符串。
func (c *RecommendRemoteCase) requestJSON(ctx context.Context, method, path string, queries map[string]string, body string) (*adminApi.RecommendRemoteJsonResponse, error) {
	data, err := c.requestRaw(ctx, method, path, queries, body)
	if err != nil {
		return nil, err
	}
	return &adminApi.RecommendRemoteJsonResponse{
		Json: string(data),
	}, nil
}

// requestJSONWithLastModified 请求远程推荐引擎并返回 JSON 字符串和最后更新时间。
func (c *RecommendRemoteCase) requestJSONWithLastModified(ctx context.Context, method, path string, queries map[string]string, body string) (*adminApi.RecommendRemoteJsonResponse, error) {
	// 远程推荐客户端未注入时，说明服务启动配置不完整。
	if c.recommend == nil {
		return nil, errorsx.Internal("远程推荐客户端未初始化")
	}
	data, lastModified, err := c.recommend.RequestJSONWithLastModified(ctx, method, path, queries, body)
	if err != nil {
		return nil, err
	}
	return &adminApi.RecommendRemoteJsonResponse{
		Json:         string(data),
		LastModified: lastModified,
	}, nil
}

// requestNoContent 请求远程推荐引擎并忽略响应内容。
func (c *RecommendRemoteCase) requestNoContent(ctx context.Context, method, path string, queries map[string]string, body string) error {
	_, err := c.requestRaw(ctx, method, path, queries, body)
	return err
}

// requestRaw 请求远程推荐引擎原始 JSON 内容。
func (c *RecommendRemoteCase) requestRaw(ctx context.Context, method, path string, queries map[string]string, body string) ([]byte, error) {
	// 远程推荐客户端未注入时，说明服务启动配置不完整。
	if c.recommend == nil {
		return nil, errorsx.Internal("远程推荐客户端未初始化")
	}
	return c.recommend.RequestJSON(ctx, method, path, queries, body)
}

// buildCursorQueries 构建游标查询参数。
func (c *RecommendRemoteCase) buildCursorQueries(req *adminApi.RecommendRemoteCursorRequest, defaultSize int64) map[string]string {
	size := c.resolveListSize(req.GetN(), defaultSize)
	queries := map[string]string{
		"n": strconv.FormatInt(size, 10),
	}
	// 传入游标时，继续查询下一页远程数据。
	if strings.TrimSpace(req.GetCursor()) != "" {
		queries["cursor"] = strings.TrimSpace(req.GetCursor())
	}
	// 传入编号关键字时，保留给远程接口做原生筛选。
	if strings.TrimSpace(req.GetId()) != "" {
		queries["id"] = strings.TrimSpace(req.GetId())
	}
	return queries
}

// buildTimeseriesQueries 构建远程时间序列查询参数。
func (c *RecommendRemoteCase) buildTimeseriesQueries(req *adminApi.RecommendRemoteNameRequest) map[string]string {
	queries := make(map[string]string, 2)
	// 传入开始时间时，按 Gorse Dashboard 查询指定时间窗口。
	if strings.TrimSpace(req.GetBegin()) != "" {
		queries["begin"] = strings.TrimSpace(req.GetBegin())
	}
	// 传入结束时间时，按 Gorse Dashboard 查询指定时间窗口。
	if strings.TrimSpace(req.GetEnd()) != "" {
		queries["end"] = strings.TrimSpace(req.GetEnd())
	}
	return queries
}

// buildDashboardItemsQueries 构建仪表盘推荐商品查询参数。
func (c *RecommendRemoteCase) buildDashboardItemsQueries(req *adminApi.RecommendRemoteDashboardItemsRequest) map[string]string {
	size := c.resolveListSize(req.GetEnd(), recommendRemoteDefaultExportSize)
	queries := map[string]string{
		"end": strconv.FormatInt(size, 10),
	}
	// 选择分类时，只查询当前分类下的推荐商品。
	if strings.TrimSpace(req.GetCategory()) != "" {
		queries["category"] = strings.TrimSpace(req.GetCategory())
	}
	return queries
}

// buildRecommendQueries 构建推荐结果查询参数。
func (c *RecommendRemoteCase) buildRecommendQueries(req *adminApi.RecommendRemoteRecommendRequest) map[string]string {
	queries := c.buildSizedQueries(req.GetN(), req.GetOffset(), recommendRemoteDefaultListSize)
	// 非个性化或协同过滤查询可通过 user-id 过滤用户已读商品。
	if strings.TrimSpace(req.GetId()) != "" {
		queries["user-id"] = strings.TrimSpace(req.GetId())
	}
	// 指定写回类型时，Gorse 会把推荐结果写回为对应反馈类型。
	if strings.TrimSpace(req.GetWriteBackType()) != "" {
		queries["write-back-type"] = strings.TrimSpace(req.GetWriteBackType())
	}
	// 指定候选集时，透传给远程推荐引擎做候选商品约束。
	if strings.TrimSpace(req.GetCandidates()) != "" {
		queries["candidates"] = strings.TrimSpace(req.GetCandidates())
	}
	return queries
}

// buildNeighborQueries 构建相似内容查询参数。
func (c *RecommendRemoteCase) buildNeighborQueries(req *adminApi.RecommendRemoteNeighborRequest) map[string]string {
	return c.buildSizedQueries(req.GetN(), req.GetOffset(), recommendRemoteDefaultListSize)
}

// buildFeedbackQueries 构建反馈列表查询参数。
func (c *RecommendRemoteCase) buildFeedbackQueries(req *adminApi.RecommendRemoteFeedbackRequest) map[string]string {
	queries := c.buildCursorQueries(&adminApi.RecommendRemoteCursorRequest{
		Cursor: req.GetCursor(),
		N:      req.GetN(),
	}, recommendRemoteDefaultListSize)
	return queries
}

// buildSizedQueries 构建支持数量和偏移量的查询参数。
func (c *RecommendRemoteCase) buildSizedQueries(size int64, offset int64, defaultSize int64) map[string]string {
	queries := map[string]string{
		"n": strconv.FormatInt(c.resolveListSize(size, defaultSize), 10),
	}
	// 指定偏移量时，从远程推荐缓存的对应位置开始读取。
	if offset > 0 {
		queries["offset"] = strconv.FormatInt(offset, 10)
	}
	return queries
}

// buildDataQueries 构建数据导出查询参数。
func (c *RecommendRemoteCase) buildDataQueries(req *adminApi.RecommendRemoteDataRequest) map[string]string {
	size := c.resolveListSize(req.GetN(), recommendRemoteDefaultExportSize)
	queries := map[string]string{
		"n": strconv.FormatInt(size, 10),
	}
	// 传入游标时，继续导出下一页远程数据。
	if strings.TrimSpace(req.GetCursor()) != "" {
		queries["cursor"] = strings.TrimSpace(req.GetCursor())
	}
	return queries
}

// resolveListSize 解析远程列表返回数量。
func (c *RecommendRemoteCase) resolveListSize(size int64, defaultSize int64) int64 {
	// 数量未指定或非法时，使用当前功能默认数量。
	if size <= 0 {
		return defaultSize
	}
	return size
}

// requireId 校验远程推荐资源编号。
func (c *RecommendRemoteCase) requireId(req *adminApi.RecommendRemoteIdRequest) (string, error) {
	// 请求为空时，无法定位远程资源。
	if req == nil {
		return "", errorsx.InvalidArgument("远程推荐编号不能为空")
	}
	id := strings.TrimSpace(req.GetId())
	// 编号为空时，无法定位远程资源。
	if id == "" {
		return "", errorsx.InvalidArgument("远程推荐编号不能为空")
	}
	return id, nil
}

// requireName 校验远程推荐指标名称。
func (c *RecommendRemoteCase) requireName(req *adminApi.RecommendRemoteNameRequest) (string, error) {
	// 请求为空时，无法定位远程指标。
	if req == nil {
		return "", errorsx.InvalidArgument("远程推荐指标名称不能为空")
	}
	name := strings.TrimSpace(req.GetName())
	// 名称为空时，无法定位远程指标。
	if name == "" {
		return "", errorsx.InvalidArgument("远程推荐指标名称不能为空")
	}
	return name, nil
}

// requireRecommender 校验远程推荐仪表盘推荐器名称。
func (c *RecommendRemoteCase) requireRecommender(req *adminApi.RecommendRemoteDashboardItemsRequest) (string, error) {
	// 请求为空时，无法定位远程推荐器。
	if req == nil {
		return "", errorsx.InvalidArgument("远程推荐器名称不能为空")
	}
	recommender := strings.TrimSpace(req.GetRecommender())
	// 推荐器名称为空时，默认使用 Gorse Dashboard 的 latest 推荐器。
	if recommender == "" {
		return "latest", nil
	}
	return recommender, nil
}

// escapeDashboardRecommender 转义仪表盘推荐器路径。
func (c *RecommendRemoteCase) escapeDashboardRecommender(recommender string) string {
	segments := strings.Split(recommender, "/")
	for i, segment := range segments {
		segments[i] = remote.EscapePathSegment(segment)
	}
	return strings.Join(segments, "/")
}

// resolveRecommendPath 解析推荐查询对应的 Gorse 原生接口路径。
func (c *RecommendRemoteCase) resolveRecommendPath(req *adminApi.RecommendRemoteRecommendRequest) (string, error) {
	// 请求为空时，无法判断推荐查询类型。
	if req == nil {
		return "", errorsx.InvalidArgument("远程推荐查询条件不能为空")
	}
	recommendType := strings.ToLower(strings.TrimSpace(req.GetType()))
	category := strings.TrimSpace(req.GetCategory())
	// 根据管理端选择的推荐类型映射到远程推荐原生接口。
	switch {
	case recommendType == "", recommendType == "recommend":
		return c.buildRequiredIdPath("/api/recommend", req.GetId(), category, "用户编号不能为空")
	case recommendType == "latest", recommendType == "popular":
		return c.buildOptionalCategoryPath("/api/"+recommendType, category), nil
	case recommendType == "collaborative", recommendType == "collaborative-filtering":
		return c.buildRequiredIdPath("/api/collaborative-filtering", req.GetId(), category, "用户编号不能为空")
	case strings.HasPrefix(recommendType, "item-to-item/"):
		name := strings.TrimPrefix(recommendType, "item-to-item/")
		return c.buildNamedRequiredIdPath("/api/item-to-item", name, req.GetId(), category, "商品编号不能为空")
	case strings.HasPrefix(recommendType, "user-to-user/"):
		name := strings.TrimPrefix(recommendType, "user-to-user/")
		return c.buildNamedRequiredIdPath("/api/user-to-user", name, req.GetId(), "", "用户编号不能为空")
	default:
		return "", errorsx.InvalidArgument("不支持的远程推荐类型")
	}
}

// resolveNeighborPath 解析相似内容查询对应的 Gorse 原生接口路径。
func (c *RecommendRemoteCase) resolveNeighborPath(req *adminApi.RecommendRemoteNeighborRequest) (string, error) {
	// 请求为空时，无法判断相似内容查询类型。
	if req == nil {
		return "", errorsx.InvalidArgument("远程相似内容查询条件不能为空")
	}
	neighborType := strings.ToLower(strings.TrimSpace(req.GetType()))
	category := strings.TrimSpace(req.GetCategory())
	// 根据相似内容类型映射到远程推荐原生接口。
	switch {
	case neighborType == "", neighborType == "item":
		return c.buildRequiredMiddlePath("/api/item", req.GetId(), "neighbors", category, "商品编号不能为空")
	case neighborType == "user":
		return c.buildRequiredMiddlePath("/api/user", req.GetId(), "neighbors", "", "用户编号不能为空")
	case strings.HasPrefix(neighborType, "item-to-item/"):
		name := strings.TrimPrefix(neighborType, "item-to-item/")
		return c.buildNamedRequiredIdPath("/api/item-to-item", name, req.GetId(), category, "商品编号不能为空")
	case strings.HasPrefix(neighborType, "user-to-user/"):
		name := strings.TrimPrefix(neighborType, "user-to-user/")
		return c.buildNamedRequiredIdPath("/api/user-to-user", name, req.GetId(), "", "用户编号不能为空")
	default:
		return "", errorsx.InvalidArgument("不支持的远程相似内容类型")
	}
}

// resolveFeedbackPath 解析反馈查询对应的 Gorse 原生接口路径。
func (c *RecommendRemoteCase) resolveFeedbackPath(req *adminApi.RecommendRemoteFeedbackRequest) (string, error) {
	// 请求为空时，默认查询反馈列表。
	if req == nil {
		return "/api/feedback", nil
	}
	feedbackType := strings.TrimSpace(req.GetFeedbackType())
	userId := strings.TrimSpace(req.GetUserId())
	itemId := strings.TrimSpace(req.GetItemId())
	// 查询指定用户和商品的指定反馈类型时，使用最精确的反馈详情接口。
	if feedbackType != "" && userId != "" && itemId != "" {
		path := "/api/feedback/" + remote.EscapePathSegment(feedbackType) + "/" + remote.EscapePathSegment(userId) + "/" + remote.EscapePathSegment(itemId)
		return path, nil
	}
	// 查询指定用户和商品的全部反馈时，使用用户商品二元反馈接口。
	if userId != "" && itemId != "" {
		path := "/api/feedback/" + remote.EscapePathSegment(userId) + "/" + remote.EscapePathSegment(itemId)
		return path, nil
	}
	// 查询指定用户下的某类反馈时，使用用户反馈分类接口。
	if userId != "" && feedbackType != "" {
		path := "/api/user/" + remote.EscapePathSegment(userId) + "/feedback/" + remote.EscapePathSegment(feedbackType)
		return path, nil
	}
	// 查询指定用户下的全部反馈时，使用用户反馈接口。
	if userId != "" {
		path := "/api/user/" + remote.EscapePathSegment(userId) + "/feedback"
		return path, nil
	}
	return "/api/feedback", nil
}

// resolveFeedbackDeletePath 解析反馈删除对应的 Gorse 原生接口路径。
func (c *RecommendRemoteCase) resolveFeedbackDeletePath(req *adminApi.RecommendRemoteFeedbackDeleteRequest) (string, error) {
	// 请求为空时，无法定位要删除的反馈。
	if req == nil {
		return "", errorsx.InvalidArgument("反馈删除条件不能为空")
	}
	userId := strings.TrimSpace(req.GetUserId())
	itemId := strings.TrimSpace(req.GetItemId())
	feedbackType := strings.TrimSpace(req.GetFeedbackType())
	// 用户或商品为空时，无法定位要删除的反馈关系。
	if userId == "" || itemId == "" {
		return "", errorsx.InvalidArgument("用户编号和商品编号不能为空")
	}
	// 指定反馈类型时，只删除该类型反馈。
	if feedbackType != "" {
		path := "/api/feedback/" + remote.EscapePathSegment(feedbackType) + "/" + remote.EscapePathSegment(userId) + "/" + remote.EscapePathSegment(itemId)
		return path, nil
	}
	path := "/api/feedback/" + remote.EscapePathSegment(userId) + "/" + remote.EscapePathSegment(itemId)
	return path, nil
}

// buildRequiredIdPath 构建带必填主体编号和可选分类的路径。
func (c *RecommendRemoteCase) buildRequiredIdPath(basePath string, rawId string, category string, emptyMessage string) (string, error) {
	id := strings.TrimSpace(rawId)
	// 主体编号为空时，远程推荐无法定位查询对象。
	if id == "" {
		return "", errorsx.InvalidArgument(emptyMessage)
	}
	path := basePath + "/" + remote.EscapePathSegment(id)
	// 分类为空时，查询全局推荐结果。
	if strings.TrimSpace(category) == "" {
		return path, nil
	}
	return path + "/" + remote.EscapePathSegment(strings.TrimSpace(category)), nil
}

// buildNamedRequiredIdPath 构建命名推荐器路径。
func (c *RecommendRemoteCase) buildNamedRequiredIdPath(basePath string, rawName string, rawId string, category string, emptyMessage string) (string, error) {
	name := strings.TrimSpace(rawName)
	// 推荐器名称为空时，远程命名推荐器无法定位。
	if name == "" {
		return "", errorsx.InvalidArgument("推荐器名称不能为空")
	}
	path, err := c.buildRequiredIdPath(basePath+"/"+remote.EscapePathSegment(name), rawId, category, emptyMessage)
	if err != nil {
		return "", err
	}
	return path, nil
}

// buildRequiredMiddlePath 构建中间带资源动作的路径。
func (c *RecommendRemoteCase) buildRequiredMiddlePath(basePath string, rawId string, action string, category string, emptyMessage string) (string, error) {
	id := strings.TrimSpace(rawId)
	// 主体编号为空时，远程相似内容无法定位查询对象。
	if id == "" {
		return "", errorsx.InvalidArgument(emptyMessage)
	}
	path := basePath + "/" + remote.EscapePathSegment(id) + "/" + remote.EscapePathSegment(action)
	// 分类为空时，查询全局相似内容。
	if strings.TrimSpace(category) == "" {
		return path, nil
	}
	return path + "/" + remote.EscapePathSegment(strings.TrimSpace(category)), nil
}

// buildOptionalCategoryPath 构建可选分类路径。
func (c *RecommendRemoteCase) buildOptionalCategoryPath(basePath string, category string) string {
	// 分类为空时，查询全局结果。
	if strings.TrimSpace(category) == "" {
		return basePath
	}
	return basePath + "/" + remote.EscapePathSegment(strings.TrimSpace(category))
}

// normalizeJSONBody 校验并清洗 JSON 请求体。
func (c *RecommendRemoteCase) normalizeJSONBody(raw string) (string, error) {
	body := strings.TrimSpace(raw)
	// 请求体为空时，远程接口无法完成导入或保存。
	if body == "" {
		return "", errorsx.InvalidArgument("JSON内容不能为空")
	}
	// 请求体不是合法 JSON 时，提前拦截避免污染远程配置。
	if !json.Valid([]byte(body)) {
		return "", errorsx.InvalidArgument("JSON内容格式不正确")
	}
	return body, nil
}

// resolveDataPath 解析远程推荐数据类型对应的原生接口路径。
func (c *RecommendRemoteCase) resolveDataPath(dataType string) (string, error) {
	normalizedType := strings.ToLower(strings.TrimSpace(dataType))
	// 根据管理端支持的数据类型映射到远程推荐原生数据接口。
	switch normalizedType {
	// 未传类型或选择用户时，代理到 Gorse 用户数据接口。
	case "", "user", "users":
		return "/api/users", nil
	// 选择商品时，代理到 Gorse 商品数据接口。
	case "item", "items":
		return "/api/items", nil
	default:
		return "", errorsx.InvalidArgument("数据类型仅支持用户或商品")
	}
}

// normalizePurgeCheckList 校验并标准化远程推荐清空确认项。
func (c *RecommendRemoteCase) normalizePurgeCheckList(values []string) []string {
	requiredSet := map[string]struct{}{
		"delete_users":    {},
		"delete_items":    {},
		"delete_feedback": {},
		"delete_cache":    {},
	}
	checkList := make([]string, 0, len(requiredSet))
	seenSet := make(map[string]struct{}, len(requiredSet))
	for _, value := range values {
		normalizedValue := strings.ToLower(strings.TrimSpace(value))
		if _, ok := requiredSet[normalizedValue]; !ok {
			continue
		}
		// 同一确认项重复传入时，仅保留一次，避免远程参数冗余。
		if _, ok := seenSet[normalizedValue]; ok {
			continue
		}
		seenSet[normalizedValue] = struct{}{}
		checkList = append(checkList, normalizedValue)
	}
	return checkList
}
