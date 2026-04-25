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

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	remoteDefaultListSize   = int64(20)
	remoteDefaultExportSize = int64(100)
)

// RemoteCase 远程推荐管理业务实例。
type RemoteCase struct {
	recommend *remote.Recommend
}

// NewRemoteCase 创建远程推荐管理业务实例。
func NewRemoteCase(recommend *remote.Recommend) *RemoteCase {
	return &RemoteCase{
		recommend: recommend,
	}
}

// GetOverview 查询远程推荐概览。
func (c *RemoteCase) GetOverview(ctx context.Context) (*adminApi.OverviewResponse, error) {
	config, err := c.GetConfig(ctx)
	if err != nil {
		return nil, err
	}
	return &adminApi.OverviewResponse{
		Config: config.GetConfig(),
	}, nil
}

// GetTask 查询远程推荐任务状态。
func (c *RemoteCase) GetTask(ctx context.Context) (*adminApi.TasksResponse, error) {
	data, err := c.requestRaw(ctx, stdhttp.MethodGet, "/api/dashboard/tasks", nil, "")
	if err != nil {
		return nil, err
	}
	return &adminApi.TasksResponse{List: c.buildTasks(data)}, nil
}

// GetCategory 查询远程推荐分类。
func (c *RemoteCase) GetCategory(ctx context.Context) (*adminApi.CategoriesResponse, error) {
	data, err := c.requestRaw(ctx, stdhttp.MethodGet, "/api/dashboard/categories", nil, "")
	if err != nil {
		return nil, err
	}
	return &adminApi.CategoriesResponse{List: c.buildRemoteCategories(data)}, nil
}

// GetTimeseries 查询远程推荐时间序列。
func (c *RemoteCase) GetTimeseries(ctx context.Context, req *adminApi.NameRequest) (*adminApi.TimeseriesResponse, error) {
	name, err := c.requireName(req)
	if err != nil {
		return nil, err
	}
	path := "/api/dashboard/timeseries/" + remote.EscapePathSegment(name)
	data, err := c.requestRaw(ctx, stdhttp.MethodGet, path, c.buildTimeseriesQueries(req), "")
	if err != nil {
		return nil, err
	}
	return &adminApi.TimeseriesResponse{Points: c.buildTimeseriesPoints(data)}, nil
}

// GetDashboardItems 查询远程推荐仪表盘推荐商品。
func (c *RemoteCase) GetDashboardItems(ctx context.Context, req *adminApi.DashboardItemsRequest) (*adminApi.RecordsResponse, error) {
	recommender, err := c.requireRecommender(req)
	if err != nil {
		return nil, err
	}
	path := "/api/dashboard/" + c.escapeDashboardRecommender(recommender)
	data, lastModified, err := c.requestRawWithLastModified(ctx, stdhttp.MethodGet, path, c.buildDashboardItemsQueries(req), "")
	if err != nil {
		return nil, err
	}
	return &adminApi.RecordsResponse{
		List:         c.buildRecords(data),
		LastModified: lastModified,
	}, nil
}

// GetRecommendation 查询远程推荐结果。
func (c *RemoteCase) GetRecommendation(ctx context.Context, req *adminApi.RecommendationRequest) (*adminApi.RecordsResponse, error) {
	path, err := c.resolveRecommendPath(req)
	if err != nil {
		return nil, err
	}
	data, err := c.requestRaw(ctx, stdhttp.MethodGet, path, c.buildRecommendQueries(req), "")
	if err != nil {
		return nil, err
	}
	return &adminApi.RecordsResponse{List: c.buildRecords(data)}, nil
}

// GetNeighbor 查询远程相似内容。
func (c *RemoteCase) GetNeighbor(ctx context.Context, req *adminApi.NeighborRequest) (*adminApi.RecordsResponse, error) {
	path, err := c.resolveNeighborPath(req)
	if err != nil {
		return nil, err
	}
	data, err := c.requestRaw(ctx, stdhttp.MethodGet, path, c.buildNeighborQueries(req), "")
	if err != nil {
		return nil, err
	}
	return &adminApi.RecordsResponse{List: c.buildRecords(data)}, nil
}

// PageFeedback 查询远程推荐反馈列表。
func (c *RemoteCase) PageFeedback(ctx context.Context, req *adminApi.FeedbackRequest) (*adminApi.FeedbackPageResponse, error) {
	path, err := c.resolveFeedbackPath(req)
	if err != nil {
		return nil, err
	}
	data, err := c.requestRaw(ctx, stdhttp.MethodGet, path, c.buildFeedbackQueries(req), "")
	if err != nil {
		return nil, err
	}
	return c.buildFeedbackPage(data), nil
}

// ImportFeedback 写入远程推荐反馈。
func (c *RemoteCase) ImportFeedback(ctx context.Context, req *adminApi.JsonRequest) error {
	body, err := c.normalizeJSONBody(req.GetJson())
	if err != nil {
		return err
	}
	return c.requestNoContent(ctx, stdhttp.MethodPost, "/api/feedback", nil, body)
}

// DeleteFeedback 删除远程推荐反馈。
func (c *RemoteCase) DeleteFeedback(ctx context.Context, req *adminApi.FeedbackDeleteRequest) error {
	path, err := c.resolveFeedbackDeletePath(req)
	if err != nil {
		return err
	}
	return c.requestNoContent(ctx, stdhttp.MethodDelete, path, nil, "")
}

// PageUser 查询远程推荐用户列表。
func (c *RemoteCase) PageUser(ctx context.Context, req *adminApi.CursorRequest) (*adminApi.UsersPageResponse, error) {
	data, err := c.requestRaw(ctx, stdhttp.MethodGet, "/api/users", c.buildCursorQueries(req, remoteDefaultListSize), "")
	if err != nil {
		return nil, err
	}
	return c.buildUsersPage(data), nil
}

// GetUser 查询远程推荐用户。
func (c *RemoteCase) GetUser(ctx context.Context, req *adminApi.IdRequest) (*adminApi.User, error) {
	id, err := c.requireId(req)
	if err != nil {
		return nil, err
	}
	path := "/api/user/" + remote.EscapePathSegment(id)
	data, err := c.requestRaw(ctx, stdhttp.MethodGet, path, nil, "")
	if err != nil {
		return nil, err
	}
	return c.buildUser(c.parseRecord(data)), nil
}

// DeleteUser 删除远程推荐用户。
func (c *RemoteCase) DeleteUser(ctx context.Context, req *adminApi.IdRequest) error {
	id, err := c.requireId(req)
	if err != nil {
		return err
	}
	path := "/api/user/" + remote.EscapePathSegment(id)
	return c.requestNoContent(ctx, stdhttp.MethodDelete, path, nil, "")
}

// PageItem 查询远程推荐商品列表。
func (c *RemoteCase) PageItem(ctx context.Context, req *adminApi.CursorRequest) (*adminApi.ItemsPageResponse, error) {
	data, err := c.requestRaw(ctx, stdhttp.MethodGet, "/api/items", c.buildCursorQueries(req, remoteDefaultListSize), "")
	if err != nil {
		return nil, err
	}
	return c.buildItemsPage(data), nil
}

// GetItem 查询远程推荐商品。
func (c *RemoteCase) GetItem(ctx context.Context, req *adminApi.IdRequest) (*adminApi.Item, error) {
	id, err := c.requireId(req)
	if err != nil {
		return nil, err
	}
	path := "/api/item/" + remote.EscapePathSegment(id)
	data, err := c.requestRaw(ctx, stdhttp.MethodGet, path, nil, "")
	if err != nil {
		return nil, err
	}
	return c.buildItem(c.parseRecord(data)), nil
}

// DeleteItem 删除远程推荐商品。
func (c *RemoteCase) DeleteItem(ctx context.Context, req *adminApi.IdRequest) error {
	id, err := c.requireId(req)
	if err != nil {
		return err
	}
	path := "/api/item/" + remote.EscapePathSegment(id)
	return c.requestNoContent(ctx, stdhttp.MethodDelete, path, nil, "")
}

// ExportData 导出远程推荐数据。
func (c *RemoteCase) ExportData(ctx context.Context, req *adminApi.DataRequest) (*adminApi.DataPageResponse, error) {
	path, err := c.resolveDataPath(req.GetType())
	if err != nil {
		return nil, err
	}
	data, err := c.requestRaw(ctx, stdhttp.MethodGet, path, c.buildDataQueries(req), "")
	if err != nil {
		return nil, err
	}
	page := c.buildRecordPage(data)
	return &adminApi.DataPageResponse{List: page.List, Cursor: page.Cursor}, nil
}

// ImportData 导入远程推荐数据。
func (c *RemoteCase) ImportData(ctx context.Context, req *adminApi.ImportRequest) error {
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

// PurgeData 清空远程推荐数据。
func (c *RemoteCase) PurgeData(ctx context.Context, req *adminApi.PurgeRequest) error {
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

// GetFlowConfig 查询推荐编排配置。
func (c *RemoteCase) GetFlowConfig(ctx context.Context) (*adminApi.ConfigResponse, error) {
	data, err := c.requestRaw(ctx, stdhttp.MethodGet, "/api/dashboard/config", nil, "")
	if err != nil {
		return nil, err
	}
	return &adminApi.ConfigResponse{Config: c.parseRemoteStruct(data)}, nil
}

// SaveFlowConfig 保存推荐编排配置。
func (c *RemoteCase) SaveFlowConfig(ctx context.Context, req *adminApi.JsonRequest) error {
	body, err := c.normalizeJSONBody(req.GetJson())
	if err != nil {
		return err
	}
	return c.requestNoContent(ctx, stdhttp.MethodPost, "/api/dashboard/config", nil, body)
}

// ResetFlowConfig 重置推荐编排配置。
func (c *RemoteCase) ResetFlowConfig(ctx context.Context) error {
	return c.requestNoContent(ctx, stdhttp.MethodDelete, "/api/dashboard/config", nil, "")
}

// GetFlowSchema 查询推荐编排配置结构。
func (c *RemoteCase) GetFlowSchema(ctx context.Context) (*adminApi.ConfigResponse, error) {
	data, err := c.requestRaw(ctx, stdhttp.MethodGet, "/api/dashboard/config/schema", nil, "")
	if err != nil {
		return nil, err
	}
	return &adminApi.ConfigResponse{Config: c.parseRemoteStruct(data)}, nil
}

// GetConfig 查询远程推荐配置。
func (c *RemoteCase) GetConfig(ctx context.Context) (*adminApi.ConfigResponse, error) {
	data, err := c.requestRaw(ctx, stdhttp.MethodGet, "/api/dashboard/config", nil, "")
	if err != nil {
		return nil, err
	}
	return &adminApi.ConfigResponse{Config: c.parseRemoteStruct(data)}, nil
}

// requestRawWithLastModified 请求远程推荐引擎并返回原始 JSON 内容和最后更新时间。
func (c *RemoteCase) requestRawWithLastModified(ctx context.Context, method, path string, queries map[string]string, body string) ([]byte, string, error) {
	// 远程推荐客户端未注入时，说明服务启动配置不完整。
	if c.recommend == nil {
		return nil, "", errorsx.Internal("远程推荐客户端未初始化")
	}
	return c.recommend.RequestJSONWithLastModified(ctx, method, path, queries, body)
}

// requestNoContent 请求远程推荐引擎并忽略响应内容。
func (c *RemoteCase) requestNoContent(ctx context.Context, method, path string, queries map[string]string, body string) error {
	_, err := c.requestRaw(ctx, method, path, queries, body)
	return err
}

// requestRaw 请求远程推荐引擎原始 JSON 内容。
func (c *RemoteCase) requestRaw(ctx context.Context, method, path string, queries map[string]string, body string) ([]byte, error) {
	// 远程推荐客户端未注入时，说明服务启动配置不完整。
	if c.recommend == nil {
		return nil, errorsx.Internal("远程推荐客户端未初始化")
	}
	return c.recommend.RequestJSON(ctx, method, path, queries, body)
}

// buildTasks 将远程任务响应转换为前端可直接消费的任务列表。
func (c *RemoteCase) buildTasks(data []byte) []*adminApi.Task {
	records := c.parseRecordList(data, []string{"Tasks", "tasks", "Nodes", "nodes"})
	tasks := make([]*adminApi.Task, 0, len(records))
	for _, record := range records {
		tasks = append(tasks, &adminApi.Task{
			Name:       c.resolveString(record, []string{"Name", "name", "Task", "task"}),
			Status:     c.resolveString(record, []string{"Status", "status"}),
			Total:      c.resolveInt64(record, []string{"Total", "total"}),
			Count:      c.resolveInt64(record, []string{"Count", "count"}),
			Error:      c.resolveString(record, []string{"Error", "error"}),
			StartTime:  c.resolveString(record, []string{"StartTime", "startTime", "start_time"}),
			FinishTime: c.resolveString(record, []string{"FinishTime", "finishTime", "finish_time"}),
			Raw:        c.mapToStruct(record),
		})
	}
	return tasks
}

// buildRemoteCategories 将远程分类响应转换为分类列表。
func (c *RemoteCase) buildRemoteCategories(data []byte) []*adminApi.Category {
	values := c.parseRemoteValueList(data, []string{"Categories", "categories", "Items", "items", "List", "list"})
	categories := make([]*adminApi.Category, 0, len(values))
	for index, value := range values {
		record, ok := value.(map[string]interface{})
		// 分类可能直接以字符串数组返回，此时用字符串作为分类名称。
		if !ok {
			name := strings.TrimSpace(c.valueToString(value))
			if name == "" {
				continue
			}
			categories = append(categories, &adminApi.Category{Name: name, Raw: c.valueToProto(value)})
			continue
		}
		name := c.resolveString(record, []string{"Name", "name", "Category", "category", "Label", "label", "Key", "key"})
		if name == "" && index >= 0 {
			name = c.valueToString(value)
		}
		if name == "" {
			continue
		}
		categories = append(categories, &adminApi.Category{
			Name:  name,
			Count: c.resolveString(record, []string{"Count", "count", "Value", "value", "Total", "total"}),
			Raw:   c.valueToProto(value),
		})
	}
	return categories
}

// buildTimeseriesPoints 将远程时间序列响应转换为指标点列表。
func (c *RemoteCase) buildTimeseriesPoints(data []byte) []*adminApi.TimeseriesPoint {
	values := c.parseRemoteValueList(data, []string{"Timeseries", "timeseries", "Items", "items", "Values", "values"})
	points := make([]*adminApi.TimeseriesPoint, 0, len(values))
	for _, value := range values {
		if record, ok := value.(map[string]interface{}); ok {
			points = append(points, &adminApi.TimeseriesPoint{
				Timestamp: c.resolveString(record, []string{"Timestamp", "timestamp", "Time", "time", "Date", "date"}),
				Value:     c.resolveFloat64(record, []string{"Value", "value", "Count", "count"}),
			})
			continue
		}
		points = append(points, &adminApi.TimeseriesPoint{Value: c.numberFromValue(value)})
	}
	return points
}

// buildRecords 将远程推荐结果转换为通用记录列表。
func (c *RemoteCase) buildRecords(data []byte) []*adminApi.ResultRecord {
	records := c.parseRecordList(data, []string{"Items", "items", "Results", "results", "Recommendations", "recommendations", "Neighbors", "neighbors"})
	list := make([]*adminApi.ResultRecord, 0, len(records))
	for _, record := range records {
		list = append(list, c.buildRecord(record))
	}
	return list
}

// buildRecordPage 将远程游标数据转换为通用记录分页。
func (c *RemoteCase) buildRecordPage(data []byte) *adminApi.DataPageResponse {
	records, cursor := c.parseRemoteCursorRecords(data, []string{"Users", "users", "Items", "items", "List", "list"})
	list := make([]*adminApi.ResultRecord, 0, len(records))
	for _, record := range records {
		list = append(list, c.buildRecord(record))
	}
	return &adminApi.DataPageResponse{List: list, Cursor: cursor}
}

// buildRecord 将远程记录转换为前端通用记录。
func (c *RemoteCase) buildRecord(record map[string]interface{}) *adminApi.ResultRecord {
	return &adminApi.ResultRecord{
		Id:         c.resolveString(record, []string{"ItemId", "itemId", "item_id", "UserId", "userId", "user_id", "Id", "id"}),
		Categories: c.resolveStringList(record, []string{"Categories", "categories"}),
		Labels:     c.valueToProto(c.resolveValue(record, []string{"Labels", "labels"})),
		Comment:    c.resolveString(record, []string{"Comment", "comment", "Description", "description"}),
		Timestamp:  c.resolveString(record, []string{"Timestamp", "timestamp", "LastUpdateTime", "lastUpdateTime", "last_update_time"}),
		Score:      c.resolveFloat64(record, []string{"Score", "score"}),
		IsHidden:   c.resolveBool(record, []string{"IsHidden", "isHidden", "is_hidden"}),
		Raw:        c.mapToStruct(record),
	}
}

// buildFeedbackPage 将远程反馈响应转换为反馈分页。
func (c *RemoteCase) buildFeedbackPage(data []byte) *adminApi.FeedbackPageResponse {
	records, cursor := c.parseRemoteCursorRecords(data, []string{"Feedback", "feedback", "Items", "items", "List", "list"})
	list := make([]*adminApi.Feedback, 0, len(records))
	for _, record := range records {
		list = append(list, &adminApi.Feedback{
			FeedbackType: c.resolveString(record, []string{"FeedbackType", "feedbackType", "feedback_type", "Type", "type"}),
			UserId:       c.resolveString(record, []string{"UserId", "userId", "user_id"}),
			ItemId:       c.resolveString(record, []string{"ItemId", "itemId", "item_id"}),
			Timestamp:    c.resolveString(record, []string{"Timestamp", "timestamp", "Time", "time"}),
			Detail:       c.valueToProto(c.resolveValue(record, []string{"Detail", "detail", "Comment", "comment"})),
			Raw:          c.mapToStruct(record),
		})
	}
	return &adminApi.FeedbackPageResponse{List: list, Cursor: cursor}
}

// buildUsersPage 将远程用户响应转换为用户分页。
func (c *RemoteCase) buildUsersPage(data []byte) *adminApi.UsersPageResponse {
	records, cursor := c.parseRemoteCursorRecords(data, []string{"Users", "users"})
	list := make([]*adminApi.User, 0, len(records))
	for _, record := range records {
		list = append(list, c.buildUser(record))
	}
	return &adminApi.UsersPageResponse{List: list, Cursor: cursor}
}

// buildUser 将远程用户记录转换为用户对象。
func (c *RemoteCase) buildUser(record map[string]interface{}) *adminApi.User {
	return &adminApi.User{
		Id:             c.resolveString(record, []string{"UserId", "userId", "user_id", "Id", "id"}),
		Labels:         c.resolveStringList(record, []string{"Labels", "labels"}),
		Subscribe:      c.resolveString(record, []string{"Subscribe", "subscribe", "SubscribeCategories", "subscribeCategories"}),
		Comment:        c.resolveString(record, []string{"Comment", "comment", "Description", "description"}),
		LastUpdateTime: c.resolveString(record, []string{"LastUpdateTime", "lastUpdateTime", "last_update_time", "Timestamp", "timestamp"}),
		Raw:            c.mapToStruct(record),
	}
}

// buildItemsPage 将远程商品响应转换为商品分页。
func (c *RemoteCase) buildItemsPage(data []byte) *adminApi.ItemsPageResponse {
	records, cursor := c.parseRemoteCursorRecords(data, []string{"Items", "items"})
	list := make([]*adminApi.Item, 0, len(records))
	for _, record := range records {
		list = append(list, c.buildItem(record))
	}
	return &adminApi.ItemsPageResponse{List: list, Cursor: cursor}
}

// buildItem 将远程商品记录转换为商品对象。
func (c *RemoteCase) buildItem(record map[string]interface{}) *adminApi.Item {
	return &adminApi.Item{
		Id:         c.resolveString(record, []string{"ItemId", "itemId", "item_id", "Id", "id"}),
		Categories: c.resolveStringList(record, []string{"Categories", "categories"}),
		Labels:     c.resolveStringList(record, []string{"Labels", "labels"}),
		Comment:    c.resolveString(record, []string{"Comment", "comment", "Description", "description"}),
		IsHidden:   c.resolveBool(record, []string{"IsHidden", "isHidden", "is_hidden"}),
		Timestamp:  c.resolveString(record, []string{"Timestamp", "timestamp", "LastUpdateTime", "lastUpdateTime", "last_update_time"}),
		Raw:        c.mapToStruct(record),
	}
}

// parseRemoteStruct 将远程 JSON 对象转换为 protobuf Struct。
func (c *RemoteCase) parseRemoteStruct(data []byte) *structpb.Struct {
	record := c.parseRecord(data)
	return c.mapToStruct(record)
}

// parseRecord 将远程 JSON 转换为单条记录。
func (c *RemoteCase) parseRecord(data []byte) map[string]interface{} {
	var raw interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return map[string]interface{}{}
	}
	if record, ok := raw.(map[string]interface{}); ok {
		return record
	}
	return map[string]interface{}{}
}

// parseRecordList 将远程 JSON 转换为记录列表。
func (c *RemoteCase) parseRecordList(data []byte, listKeys []string) []map[string]interface{} {
	values := c.parseRemoteValueList(data, listKeys)
	records := make([]map[string]interface{}, 0, len(values))
	for _, value := range values {
		if record, ok := value.(map[string]interface{}); ok {
			records = append(records, record)
		}
	}
	return records
}

// parseRemoteCursorRecords 将远程游标 JSON 转换为记录列表和下一页游标。
func (c *RemoteCase) parseRemoteCursorRecords(data []byte, listKeys []string) ([]map[string]interface{}, string) {
	record := c.parseRecord(data)
	return c.parseRecordList(data, listKeys), c.resolveString(record, []string{"Cursor", "cursor"})
}

// parseRemoteValueList 将远程 JSON 转换为值列表。
func (c *RemoteCase) parseRemoteValueList(data []byte, listKeys []string) []interface{} {
	var raw interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}
	if list, ok := raw.([]interface{}); ok {
		return list
	}
	record, ok := raw.(map[string]interface{})
	if !ok {
		return nil
	}
	for _, key := range listKeys {
		if list, ok := record[key].([]interface{}); ok {
			return list
		}
	}
	for _, value := range record {
		if list, ok := value.([]interface{}); ok {
			return list
		}
	}
	return nil
}

// resolveValue 按候选字段读取远程记录值。
func (c *RemoteCase) resolveValue(record map[string]interface{}, keys []string) interface{} {
	for _, key := range keys {
		value, ok := record[key]
		if !ok || value == nil || strings.TrimSpace(c.valueToString(value)) == "" {
			continue
		}
		return value
	}
	return nil
}

// resolveString 按候选字段读取字符串。
func (c *RemoteCase) resolveString(record map[string]interface{}, keys []string) string {
	return strings.TrimSpace(c.valueToString(c.resolveValue(record, keys)))
}

// resolveStringList 按候选字段读取字符串数组。
func (c *RemoteCase) resolveStringList(record map[string]interface{}, keys []string) []string {
	value := c.resolveValue(record, keys)
	if value == nil {
		return nil
	}
	if list, ok := value.([]interface{}); ok {
		items := make([]string, 0, len(list))
		for _, item := range list {
			text := strings.TrimSpace(c.valueToString(item))
			if text != "" {
				items = append(items, text)
			}
		}
		return items
	}
	text := strings.TrimSpace(c.valueToString(value))
	if text == "" {
		return nil
	}
	return []string{text}
}

// resolveInt64 按候选字段读取整数。
func (c *RemoteCase) resolveInt64(record map[string]interface{}, keys []string) int64 {
	return int64(c.resolveFloat64(record, keys))
}

// resolveFloat64 按候选字段读取浮点数。
func (c *RemoteCase) resolveFloat64(record map[string]interface{}, keys []string) float64 {
	return c.numberFromValue(c.resolveValue(record, keys))
}

// resolveBool 按候选字段读取布尔值。
func (c *RemoteCase) resolveBool(record map[string]interface{}, keys []string) bool {
	value := c.resolveValue(record, keys)
	if boolValue, ok := value.(bool); ok {
		return boolValue
	}
	if numberValue, ok := value.(float64); ok {
		return numberValue == 1
	}
	return strings.EqualFold(c.valueToString(value), "true")
}

// numberFromValue 将远程值转换为数值。
func (c *RemoteCase) numberFromValue(value interface{}) float64 {
	switch typedValue := value.(type) {
	case float64:
		return typedValue
	case float32:
		return float64(typedValue)
	case int:
		return float64(typedValue)
	case int64:
		return float64(typedValue)
	case json.Number:
		numberValue, _ := typedValue.Float64()
		return numberValue
	case string:
		numberValue, _ := strconv.ParseFloat(strings.TrimSpace(typedValue), 64)
		return numberValue
	default:
		return 0
	}
}

// valueToString 将远程值转换为字符串。
func (c *RemoteCase) valueToString(value interface{}) string {
	switch typedValue := value.(type) {
	case nil:
		return ""
	case string:
		return typedValue
	case json.Number:
		return typedValue.String()
	case bool:
		return strconv.FormatBool(typedValue)
	case float64:
		return strconv.FormatFloat(typedValue, 'f', -1, 64)
	default:
		data, err := json.Marshal(typedValue)
		if err != nil {
			return ""
		}
		return string(data)
	}
}

// mapToStruct 将远程记录转换为 protobuf Struct。
func (c *RemoteCase) mapToStruct(record map[string]interface{}) *structpb.Struct {
	value, err := structpb.NewStruct(record)
	if err == nil {
		return value
	}
	data, marshalErr := json.Marshal(record)
	if marshalErr != nil {
		return &structpb.Struct{}
	}
	structValue := &structpb.Struct{}
	if unmarshalErr := protojson.Unmarshal(data, structValue); unmarshalErr != nil {
		return &structpb.Struct{}
	}
	return structValue
}

// valueToProto 将远程动态值转换为 protobuf Value。
func (c *RemoteCase) valueToProto(value interface{}) *structpb.Value {
	protoValue, err := structpb.NewValue(value)
	if err == nil {
		return protoValue
	}
	data, marshalErr := json.Marshal(value)
	if marshalErr != nil {
		return structpb.NewNullValue()
	}
	parsedValue := &structpb.Value{}
	if unmarshalErr := protojson.Unmarshal(data, parsedValue); unmarshalErr != nil {
		return structpb.NewNullValue()
	}
	return parsedValue
}

// buildCursorQueries 构建游标查询参数。
func (c *RemoteCase) buildCursorQueries(req *adminApi.CursorRequest, defaultSize int64) map[string]string {
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
func (c *RemoteCase) buildTimeseriesQueries(req *adminApi.NameRequest) map[string]string {
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
func (c *RemoteCase) buildDashboardItemsQueries(req *adminApi.DashboardItemsRequest) map[string]string {
	size := c.resolveListSize(req.GetEnd(), remoteDefaultExportSize)
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
func (c *RemoteCase) buildRecommendQueries(req *adminApi.RecommendationRequest) map[string]string {
	queries := c.buildSizedQueries(req.GetN(), req.GetOffset(), remoteDefaultListSize)
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
func (c *RemoteCase) buildNeighborQueries(req *adminApi.NeighborRequest) map[string]string {
	return c.buildSizedQueries(req.GetN(), req.GetOffset(), remoteDefaultListSize)
}

// buildFeedbackQueries 构建反馈列表查询参数。
func (c *RemoteCase) buildFeedbackQueries(req *adminApi.FeedbackRequest) map[string]string {
	queries := c.buildCursorQueries(&adminApi.CursorRequest{
		Cursor: req.GetCursor(),
		N:      req.GetN(),
	}, remoteDefaultListSize)
	return queries
}

// buildSizedQueries 构建支持数量和偏移量的查询参数。
func (c *RemoteCase) buildSizedQueries(size int64, offset int64, defaultSize int64) map[string]string {
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
func (c *RemoteCase) buildDataQueries(req *adminApi.DataRequest) map[string]string {
	size := c.resolveListSize(req.GetN(), remoteDefaultExportSize)
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
func (c *RemoteCase) resolveListSize(size int64, defaultSize int64) int64 {
	// 数量未指定或非法时，使用当前功能默认数量。
	if size <= 0 {
		return defaultSize
	}
	return size
}

// requireId 校验远程推荐资源编号。
func (c *RemoteCase) requireId(req *adminApi.IdRequest) (string, error) {
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
func (c *RemoteCase) requireName(req *adminApi.NameRequest) (string, error) {
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
func (c *RemoteCase) requireRecommender(req *adminApi.DashboardItemsRequest) (string, error) {
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
func (c *RemoteCase) escapeDashboardRecommender(recommender string) string {
	segments := strings.Split(recommender, "/")
	for i, segment := range segments {
		segments[i] = remote.EscapePathSegment(segment)
	}
	return strings.Join(segments, "/")
}

// resolveRecommendPath 解析推荐查询对应的 Gorse 原生接口路径。
func (c *RemoteCase) resolveRecommendPath(req *adminApi.RecommendationRequest) (string, error) {
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
func (c *RemoteCase) resolveNeighborPath(req *adminApi.NeighborRequest) (string, error) {
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
func (c *RemoteCase) resolveFeedbackPath(req *adminApi.FeedbackRequest) (string, error) {
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
func (c *RemoteCase) resolveFeedbackDeletePath(req *adminApi.FeedbackDeleteRequest) (string, error) {
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
func (c *RemoteCase) buildRequiredIdPath(basePath string, rawId string, category string, emptyMessage string) (string, error) {
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
func (c *RemoteCase) buildNamedRequiredIdPath(basePath string, rawName string, rawId string, category string, emptyMessage string) (string, error) {
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
func (c *RemoteCase) buildRequiredMiddlePath(basePath string, rawId string, action string, category string, emptyMessage string) (string, error) {
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
func (c *RemoteCase) buildOptionalCategoryPath(basePath string, category string) string {
	// 分类为空时，查询全局结果。
	if strings.TrimSpace(category) == "" {
		return basePath
	}
	return basePath + "/" + remote.EscapePathSegment(strings.TrimSpace(category))
}

// normalizeJSONBody 校验并清洗 JSON 请求体。
func (c *RemoteCase) normalizeJSONBody(raw string) (string, error) {
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
func (c *RemoteCase) resolveDataPath(dataType string) (string, error) {
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
func (c *RemoteCase) normalizePurgeCheckList(values []string) []string {
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
