package biz

import (
	"context"
	"encoding/json"
	"strings"

	_const "shop/service/shop/consts"

	shopadminv1 "shop/api/gen/go/shop/admin/v1"
	shopcommonv1 "shop/api/gen/go/shop/common/v1"
	"shop/pkg/errorsx"
	"shop/service/shop/recommend/gorse"

	client "github.com/gorse-io/gorse-go"
	"google.golang.org/protobuf/encoding/protojson"
)

// RECOMMEND_GORSE_ADVANCE_PAGE_SIZE 表示 Gorse 推荐分页预取大小。
const RECOMMEND_GORSE_ADVANCE_PAGE_SIZE = 100

const gorseConfigSecretMask = "******"

var gorseConfigSecretPaths = [][]string{
	{"database", "data_store"},
	{"database", "cache_store"},
	{"master", "dashboard_password"},
	{"recommend", "ranker", "reranker_api", "auth_token"},
	{"openai", "auth_token"},
}

// RecommendGorseCase Gorse 推荐管理业务实例。
type RecommendGorseCase struct {
	dashboard *gorse.Dashboard
}

// NewRecommendGorseCase 创建 Gorse 推荐管理业务实例。
func NewRecommendGorseCase(recommend *gorse.Dashboard) *RecommendGorseCase {
	return &RecommendGorseCase{
		dashboard: recommend,
	}
}

// OptionCategory 查询 Gorse 推荐分类列表。
func (c *RecommendGorseCase) OptionCategory(ctx context.Context) (*shopadminv1.OptionCategoryResponse, error) {
	data, err := c.dashboard.Categories(ctx)
	if err != nil {
		return nil, err
	}

	response := new(shopadminv1.OptionCategoryResponse)
	err = (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal([]byte(`{"categories":`+string(data)+`}`), response)
	if err != nil {
		return nil, err
	}
	return response, nil
}

// PageItem 查询 Gorse 推荐商品列表。
func (c *RecommendGorseCase) PageItem(ctx context.Context, cursor string, n int32) (*shopadminv1.PageItemResponse, error) {
	data, err := c.dashboard.Items(ctx, cursor, n)
	if err != nil {
		return nil, err
	}

	response := new(shopadminv1.PageItemResponse)
	err = (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(data, response)
	if err != nil {
		return nil, err
	}
	return response, nil
}

// PageUser 查询 Gorse 推荐用户列表。
func (c *RecommendGorseCase) PageUser(ctx context.Context, cursor string, n int32) (*shopadminv1.PageUserResponse, error) {
	data, err := c.dashboard.Users(ctx, cursor, n)
	if err != nil {
		return nil, err
	}

	response := new(shopadminv1.PageUserResponse)
	err = (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(data, response)
	if err != nil {
		return nil, err
	}
	return response, nil
}

// ListDashboardItem 查询 Gorse 推荐仪表盘推荐商品。
func (c *RecommendGorseCase) ListDashboardItem(
	ctx context.Context,
	recommender string,
	category string,
	end int32,
) (*shopadminv1.ListDashboardItemResponse, error) {
	data, err := c.dashboard.DashboardItems(ctx, recommender, category, end)
	if err != nil {
		return nil, err
	}

	response := new(shopadminv1.ListDashboardItemResponse)
	// Gorse 仪表盘推荐商品接口原始返回数组，字段结构通过 json_name 与 Proto 商品结构保持一致。
	if string(data) == "null" {
		return response, nil
	}
	wrappedData := make([]byte, 0, len(data)+11)
	wrappedData = append(wrappedData, `{"Items":`...)
	wrappedData = append(wrappedData, data...)
	wrappedData = append(wrappedData, '}')
	err = (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(wrappedData, response)
	if err != nil {
		return nil, err
	}
	return response, nil
}

// ListTask 查询 Gorse 推荐任务状态。
func (c *RecommendGorseCase) ListTask(ctx context.Context) (*shopadminv1.ListTaskResponse, error) {
	data, err := c.dashboard.Tasks(ctx)
	if err != nil {
		return nil, err
	}

	response := new(shopadminv1.ListTaskResponse)
	// Gorse 任务状态接口原始返回数组，任务字段通过 json_name 直接对齐 Tracer、Name 等原始字段。
	if string(data) == "null" {
		return response, nil
	}
	wrappedData := make([]byte, 0, len(data)+11)
	wrappedData = append(wrappedData, `{"Tasks":`...)
	wrappedData = append(wrappedData, data...)
	wrappedData = append(wrappedData, '}')
	err = (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(wrappedData, response)
	if err != nil {
		return nil, err
	}
	return response, nil
}

// GetConfig 查询 Gorse 推荐配置。
func (c *RecommendGorseCase) GetConfig(ctx context.Context) (*shopadminv1.ConfigResponse, error) {
	data, err := c.dashboard.Config(ctx)
	if err != nil {
		return nil, err
	}

	data, err = redactGorseConfig(data)
	if err != nil {
		return nil, err
	}

	config := new(shopadminv1.ConfigResponse)
	err = (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(data, config)
	if err != nil {
		return nil, err
	}
	return config, nil
}

// GetItem 查询 Gorse 推荐商品。
func (c *RecommendGorseCase) GetItem(ctx context.Context, id string) (*shopadminv1.Item, error) {
	data, err := c.dashboard.Item(ctx, id)
	if err != nil {
		return nil, err
	}

	response := new(shopadminv1.Item)
	err = (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(data, response)
	if err != nil {
		return nil, err
	}
	return response, nil
}

// GetItemSimilar 查询 Gorse 推荐相似商品。
func (c *RecommendGorseCase) GetItemSimilar(ctx context.Context, id, recommender, category string) (*shopadminv1.ItemListResponse, error) {
	data, err := c.dashboard.ItemSimilar(ctx, id, recommender, category)
	if err != nil {
		return nil, err
	}

	response := new(shopadminv1.ItemListResponse)
	// Gorse 相似商品接口原始返回数组，商品字段通过 json_name 直接对齐原始结构。
	if string(data) == "null" {
		return response, nil
	}
	wrappedData := make([]byte, 0, len(data)+11)
	wrappedData = append(wrappedData, `{"Items":`...)
	wrappedData = append(wrappedData, data...)
	wrappedData = append(wrappedData, '}')
	err = (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(wrappedData, response)
	if err != nil {
		return nil, err
	}
	return response, nil
}

// GetTimeSeries 查询 Gorse 推荐单项时间序列。
func (c *RecommendGorseCase) GetTimeSeries(ctx context.Context, name, begin, end string) (*shopadminv1.TimeSeriesResponse, error) {
	data, err := c.dashboard.TimeSeries(ctx, name, begin, end)
	if err != nil {
		return nil, err
	}

	response := new(shopadminv1.TimeSeriesResponse)
	// Gorse 时间序列接口原始返回数组，Proto 响应需要外层字段承载，内部字段通过 json_name 直接对齐原始结构。
	if string(data) == "null" {
		return response, nil
	}
	wrappedData := make([]byte, 0, len(data)+12)
	wrappedData = append(wrappedData, `{"Points":`...)
	wrappedData = append(wrappedData, data...)
	wrappedData = append(wrappedData, '}')
	err = (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(wrappedData, response)
	if err != nil {
		return nil, err
	}
	return response, nil
}

// GetUser 查询 Gorse 推荐用户。
func (c *RecommendGorseCase) GetUser(ctx context.Context, id string) (*shopadminv1.UserResponse, error) {
	data, err := c.dashboard.User(ctx, id)
	if err != nil {
		return nil, err
	}

	response := new(shopadminv1.UserResponse)
	err = (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(data, response)
	if err != nil {
		return nil, err
	}
	return response, nil
}

// GetUserFeedback 查询 Gorse 推荐用户反馈。
func (c *RecommendGorseCase) GetUserFeedback(
	ctx context.Context,
	id string,
	feedbackType string,
	offset int32,
	n int32,
) (*shopadminv1.FeedbackResponse, error) {
	data, err := c.dashboard.UserFeedback(ctx, id, feedbackType, offset, n)
	if err != nil {
		return nil, err
	}

	response := new(shopadminv1.FeedbackResponse)
	// Gorse 用户反馈接口原始返回数组，反馈字段通过 json_name 直接对齐 FeedbackType、Item 等原始字段。
	if string(data) == "null" {
		return response, nil
	}
	wrappedData := make([]byte, 0, len(data)+14)
	wrappedData = append(wrappedData, `{"Feedback":`...)
	wrappedData = append(wrappedData, data...)
	wrappedData = append(wrappedData, '}')
	err = (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(wrappedData, response)
	if err != nil {
		return nil, err
	}
	return response, nil
}

// GetUserRecommend 查询 Gorse 推荐用户推荐结果。
func (c *RecommendGorseCase) GetUserRecommend(
	ctx context.Context,
	id string,
	recommender string,
	category string,
	n int32,
) (*shopadminv1.ItemListResponse, error) {
	data, err := c.dashboard.UserRecommend(ctx, id, recommender, category, n)
	if err != nil {
		return nil, err
	}

	response := new(shopadminv1.ItemListResponse)
	// Gorse 用户推荐接口原始返回数组，商品字段通过 json_name 直接对齐原始结构。
	if string(data) == "null" {
		return response, nil
	}
	wrappedData := make([]byte, 0, len(data)+11)
	wrappedData = append(wrappedData, `{"Items":`...)
	wrappedData = append(wrappedData, data...)
	wrappedData = append(wrappedData, '}')
	err = (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(wrappedData, response)
	if err != nil {
		return nil, err
	}
	return response, nil
}

// GetUserSimilar 查询 Gorse 推荐相似用户。
func (c *RecommendGorseCase) GetUserSimilar(ctx context.Context, id, recommender string) (*shopadminv1.UserSimilarResponse, error) {
	data, err := c.dashboard.UserSimilar(ctx, id, recommender)
	if err != nil {
		return nil, err
	}

	response := new(shopadminv1.UserSimilarResponse)
	// Gorse 相似用户接口原始返回数组，用户字段通过 json_name 直接对齐原始结构。
	if string(data) == "null" {
		return response, nil
	}
	wrappedData := make([]byte, 0, len(data)+11)
	wrappedData = append(wrappedData, `{"Users":`...)
	wrappedData = append(wrappedData, data...)
	wrappedData = append(wrappedData, '}')
	err = (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(wrappedData, response)
	if err != nil {
		return nil, err
	}
	return response, nil
}

// DeleteItem 删除 Gorse 推荐商品。
func (c *RecommendGorseCase) DeleteItem(ctx context.Context, id string) error {
	_, err := c.dashboard.DeleteItem(ctx, id)
	if err != nil {
		return err
	}
	return nil
}

// DeleteUser 删除 Gorse 推荐用户。
func (c *RecommendGorseCase) DeleteUser(ctx context.Context, id string) error {
	_, err := c.dashboard.DeleteUser(ctx, id)
	if err != nil {
		return err
	}
	return nil
}

// ExportData 导出 Gorse 推荐数据。
func (c *RecommendGorseCase) ExportData(
	ctx context.Context,
	req *shopadminv1.ExportDataRequest,
) (*shopadminv1.ExportDataResponse, error) {
	var err error
	// 不同高级调试数据类型分别走各自的 JSONL 导出逻辑。
	switch req.GetDataType() {
	case shopcommonv1.AdvanceDataType(_const.ADVANCE_DATA_TYPE_USER):
		var content string
		content, err = c.exportRecommendGorseUsers(ctx)
		if err != nil {
			return nil, err
		}
		return &shopadminv1.ExportDataResponse{
			FileName: "users.jsonl",
			Content:  content,
		}, nil
	case shopcommonv1.AdvanceDataType(_const.ADVANCE_DATA_TYPE_ITEM):
		var content string
		content, err = c.exportRecommendGorseItems(ctx)
		if err != nil {
			return nil, err
		}
		return &shopadminv1.ExportDataResponse{
			FileName: "items.jsonl",
			Content:  content,
		}, nil
	case shopcommonv1.AdvanceDataType(_const.ADVANCE_DATA_TYPE_FEEDBACK):
		var content string
		content, err = c.exportRecommendGorseFeedback(ctx)
		if err != nil {
			return nil, err
		}
		return &shopadminv1.ExportDataResponse{
			FileName: "feedback.jsonl",
			Content:  content,
		}, nil
	default:
		return nil, errorsx.InvalidArgument("暂不支持当前导出数据类型")
	}
}

// ImportData 导入 Gorse 推荐数据。
func (c *RecommendGorseCase) ImportData(
	ctx context.Context,
	req *shopadminv1.ImportDataRequest,
) (*shopadminv1.ImportDataResponse, error) {
	recordList, err := parseRecommendGorseJSONRecords(req.GetContent())
	if err != nil {
		return nil, err
	}
	// 过滤空白行后仍没有记录时，说明文件不包含有效 JSON 数据对象。
	if len(recordList) == 0 {
		return nil, errorsx.InvalidArgument("导入文件缺少有效数据")
	}

	successCount := 0
	// 不同高级调试数据类型分别按各自模型解析并导入 Gorse 推荐服务。
	switch req.GetDataType() {
	case shopcommonv1.AdvanceDataType(_const.ADVANCE_DATA_TYPE_USER):
		successCount, err = c.importRecommendGorseUsers(ctx, recordList)
	case shopcommonv1.AdvanceDataType(_const.ADVANCE_DATA_TYPE_ITEM):
		successCount, err = c.importRecommendGorseItems(ctx, recordList)
	case shopcommonv1.AdvanceDataType(_const.ADVANCE_DATA_TYPE_FEEDBACK):
		successCount, err = c.importRecommendGorseFeedback(ctx, recordList)
	default:
		return nil, errorsx.InvalidArgument("暂不支持当前导入数据类型")
	}
	if err != nil {
		return nil, err
	}

	return &shopadminv1.ImportDataResponse{
		SuccessCount: int32(successCount),
	}, nil
}

// SaveConfig 保存 Gorse 推荐配置。
func (c *RecommendGorseCase) SaveConfig(ctx context.Context, req *shopadminv1.ConfigResponse) (*shopadminv1.ConfigResponse, error) {
	body, err := marshalGorseConfig(req)
	if err != nil {
		return nil, err
	}
	body, err = c.restoreMaskedGorseConfig(ctx, body)
	if err != nil {
		return nil, err
	}

	var data []byte
	data, err = c.dashboard.SaveConfig(ctx, string(body))
	if err != nil {
		return nil, err
	}

	data, err = redactGorseConfig(data)
	if err != nil {
		return nil, err
	}

	config := new(shopadminv1.ConfigResponse)
	err = (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(data, config)
	if err != nil {
		return nil, err
	}
	return config, nil
}

// ResetConfig 重置 Gorse 推荐配置。
func (c *RecommendGorseCase) ResetConfig(ctx context.Context) error {
	_, err := c.dashboard.ResetConfig(ctx)
	if err != nil {
		return err
	}
	return nil
}

// PreviewExternal 预览 Gorse 推荐外部推荐脚本。
func (c *RecommendGorseCase) PreviewExternal(ctx context.Context, req *shopadminv1.PreviewExternalRequest) (*shopadminv1.PreviewExternalResponse, error) {
	script := req.GetScript()
	data, err := c.dashboard.ExternalPreview(ctx, req.GetUserId(), script)
	if err != nil {
		return nil, err
	}

	response := new(shopadminv1.PreviewExternalResponse)
	// Gorse 外部推荐预览接口原始返回数组，Proto 响应需要外层 items 字段承载。
	if string(data) == "null" {
		return response, nil
	}
	wrappedData := make([]byte, 0, len(data)+10)
	wrappedData = append(wrappedData, `{"items":`...)
	wrappedData = append(wrappedData, data...)
	wrappedData = append(wrappedData, '}')
	err = (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(wrappedData, response)
	if err != nil {
		return nil, err
	}
	return response, nil
}

// PreviewRankerPrompt 预览 Gorse 推荐排序提示词。
func (c *RecommendGorseCase) PreviewRankerPrompt(ctx context.Context, req *shopadminv1.PreviewRankerPromptRequest) (*shopadminv1.PreviewRankerPromptResponse, error) {
	queryTemplate := req.GetQueryTemplate()
	documentTemplate := req.GetDocumentTemplate()
	data, err := c.dashboard.RankerPrompt(ctx, req.GetUserId(), queryTemplate, documentTemplate)
	if err != nil {
		return nil, err
	}

	response := new(shopadminv1.PreviewRankerPromptResponse)
	// Gorse 返回 null 时，Admin 侧使用空响应避免前端出现反序列化错误。
	if string(data) == "null" {
		return response, nil
	}
	err = (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(data, response)
	if err != nil {
		return nil, err
	}
	return response, nil
}

// restoreMaskedGorseConfig 将前端回传的脱敏占位符恢复为当前真实配置，避免保存编排时覆盖凭据。
func (c *RecommendGorseCase) restoreMaskedGorseConfig(ctx context.Context, body []byte) ([]byte, error) {
	currentBody, err := c.dashboard.Config(ctx)
	if err != nil {
		return nil, err
	}
	var currentConfig map[string]interface{}
	currentConfig, err = unmarshalGorseConfigMap(currentBody)
	if err != nil {
		return nil, err
	}
	var nextConfig map[string]interface{}
	nextConfig, err = unmarshalGorseConfigMap(body)
	if err != nil {
		return nil, err
	}
	for _, path := range gorseConfigSecretPaths {
		restoreMaskedGorseConfigValue(nextConfig, currentConfig, path)
	}
	return json.Marshal(nextConfig)
}

// exportRecommendGorseUsers 导出 Gorse 推荐用户 JSONL。
func (c *RecommendGorseCase) exportRecommendGorseUsers(ctx context.Context) (string, error) {
	userList := make([]client.User, 0)
	cursor := ""
	for {
		result, err := c.dashboard.UsersRaw(ctx, cursor, RECOMMEND_GORSE_ADVANCE_PAGE_SIZE)
		if err != nil {
			return "", err
		}
		userList = append(userList, result.Users...)
		// 下一页游标为空时，说明当前用户数据集已经全部导出完成。
		if result.Cursor == "" {
			break
		}
		cursor = result.Cursor
	}
	return marshalRecommendGorseJSONL(userList)
}

// exportRecommendGorseItems 导出 Gorse 推荐商品 JSONL。
func (c *RecommendGorseCase) exportRecommendGorseItems(ctx context.Context) (string, error) {
	itemList := make([]client.Item, 0)
	cursor := ""
	for {
		result, err := c.dashboard.ItemsRaw(ctx, cursor, RECOMMEND_GORSE_ADVANCE_PAGE_SIZE)
		if err != nil {
			return "", err
		}
		itemList = append(itemList, result.Items...)
		// 下一页游标为空时，说明当前商品数据集已经全部导出完成。
		if result.Cursor == "" {
			break
		}
		cursor = result.Cursor
	}
	return marshalRecommendGorseJSONL(itemList)
}

// exportRecommendGorseFeedback 导出 Gorse 推荐反馈 JSONL。
func (c *RecommendGorseCase) exportRecommendGorseFeedback(ctx context.Context) (string, error) {
	feedbackList := make([]client.Feedback, 0)
	cursor := ""
	for {
		result, err := c.dashboard.Feedbacks(ctx, cursor, RECOMMEND_GORSE_ADVANCE_PAGE_SIZE)
		if err != nil {
			return "", err
		}
		feedbackList = append(feedbackList, result.Feedback...)
		// 下一页游标为空时，说明当前反馈数据集已经全部导出完成。
		if result.Cursor == "" {
			break
		}
		cursor = result.Cursor
	}
	return marshalRecommendGorseJSONL(feedbackList)
}

// importRecommendGorseUsers 导入 Gorse 推荐用户 JSONL。
func (c *RecommendGorseCase) importRecommendGorseUsers(ctx context.Context, recordList []json.RawMessage) (int, error) {
	userList := make([]client.User, 0, len(recordList))
	var err error
	for _, record := range recordList {
		user := client.User{}
		err = json.Unmarshal(record, &user)
		if err != nil {
			return 0, errorsx.InvalidArgument("用户数据不是合法的JSON对象")
		}
		// 用户编号为空时，当前记录无法作为合法的Gorse 推荐用户导入。
		if user.UserId == "" {
			return 0, errorsx.InvalidArgument("用户数据缺少 user_id")
		}
		userList = append(userList, user)
	}
	// 有效用户对象为空时，不应该继续发起Gorse导入请求。
	if len(userList) == 0 {
		return 0, errorsx.InvalidArgument("导入文件缺少有效用户数据")
	}

	err = c.dashboard.InsertUsers(ctx, userList)
	if err != nil {
		return 0, err
	}
	return len(userList), nil
}

// importRecommendGorseItems 导入 Gorse 推荐商品 JSONL。
func (c *RecommendGorseCase) importRecommendGorseItems(ctx context.Context, recordList []json.RawMessage) (int, error) {
	itemList := make([]client.Item, 0, len(recordList))
	var err error
	for _, record := range recordList {
		item := client.Item{}
		err = json.Unmarshal(record, &item)
		if err != nil {
			return 0, errorsx.InvalidArgument("商品数据不是合法的JSON对象")
		}
		// 商品编号为空时，当前记录无法作为合法的Gorse 推荐商品导入。
		if item.ItemId == "" {
			return 0, errorsx.InvalidArgument("商品数据缺少 item_id")
		}
		itemList = append(itemList, item)
	}
	// 有效商品对象为空时，不应该继续发起Gorse导入请求。
	if len(itemList) == 0 {
		return 0, errorsx.InvalidArgument("导入文件缺少有效商品数据")
	}

	err = c.dashboard.InsertItems(ctx, itemList)
	if err != nil {
		return 0, err
	}
	return len(itemList), nil
}

// importRecommendGorseFeedback 导入 Gorse 推荐反馈 JSONL。
func (c *RecommendGorseCase) importRecommendGorseFeedback(ctx context.Context, recordList []json.RawMessage) (int, error) {
	feedbackList := make([]client.Feedback, 0, len(recordList))
	var err error
	for _, record := range recordList {
		feedback := client.Feedback{}
		err = json.Unmarshal(record, &feedback)
		if err != nil {
			return 0, errorsx.InvalidArgument("反馈数据不是合法的JSON对象")
		}
		// 反馈类型为空时，当前记录无法作为合法的Gorse 推荐反馈导入。
		if feedback.FeedbackType == "" {
			return 0, errorsx.InvalidArgument("反馈数据缺少 feedback_type")
		}
		// 用户编号为空时，当前反馈无法定位反馈主体。
		if feedback.UserId == "" {
			return 0, errorsx.InvalidArgument("反馈数据缺少 user_id")
		}
		// 商品编号为空时，当前反馈无法定位反馈商品。
		if feedback.ItemId == "" {
			return 0, errorsx.InvalidArgument("反馈数据缺少 item_id")
		}
		feedbackList = append(feedbackList, feedback)
	}
	// 有效反馈对象为空时，不应该继续发起Gorse导入请求。
	if len(feedbackList) == 0 {
		return 0, errorsx.InvalidArgument("导入文件缺少有效反馈数据")
	}

	err = c.dashboard.InsertFeedbacks(ctx, feedbackList)
	if err != nil {
		return 0, err
	}
	return len(feedbackList), nil
}

// marshalRecommendGorseJSONL 将对象列表序列化为 JSONL 文本。
func marshalRecommendGorseJSONL[T any](recordList []T) (string, error) {
	lineList := make([]string, 0, len(recordList))
	for _, record := range recordList {
		data, err := json.Marshal(record)
		if err != nil {
			return "", err
		}
		lineList = append(lineList, string(data))
	}
	return strings.Join(lineList, "\n"), nil
}

// parseRecommendGorseJSONRecords 解析Gorse 推荐 JSON 或 JSONL 内容。
func parseRecommendGorseJSONRecords(content string) ([]json.RawMessage, error) {
	content = strings.TrimPrefix(content, "\ufeff")
	// 处理完 BOM 与首尾空白后内容为空时，不存在任何可解析的数据对象。
	if content == "" {
		return nil, errorsx.InvalidArgument("导入文件内容不能为空")
	}

	var err error
	// 文件整体是 JSON 数组时，优先按数组结构解析，兼容部分调试工具的批量导出格式。
	if strings.HasPrefix(content, "[") {
		recordList := make([]json.RawMessage, 0)
		err = json.Unmarshal([]byte(content), &recordList)
		if err != nil {
			return nil, errorsx.InvalidArgument("导入文件不是合法的JSON数组")
		}
		return recordList, nil
	}

	// 文件整体是单个 JSON 对象时，统一包装成一条记录，兼容单对象手工调试导入。
	if strings.HasPrefix(content, "{") && !strings.Contains(content, "\n") {
		return []json.RawMessage{json.RawMessage(content)}, nil
	}

	lineList := strings.Split(content, "\n")
	recordList := make([]json.RawMessage, 0, len(lineList))
	for _, line := range lineList {
		// 空白行对 JSONL 导入没有业务意义，统一跳过。
		if line == "" {
			continue
		}
		// JSONL 每一行都必须是单个 JSON 对象，避免把半截数据误导入推荐服务。
		if !strings.HasPrefix(line, "{") {
			return nil, errorsx.InvalidArgument("导入文件不是合法的JSONL格式")
		}

		rawMessage := json.RawMessage{}
		err = json.Unmarshal([]byte(line), &rawMessage)
		if err != nil {
			return nil, errorsx.InvalidArgument("导入文件不是合法的JSONL格式")
		}
		recordList = append(recordList, rawMessage)
	}
	return recordList, nil
}

// marshalGorseConfig 将 Proto 配置转换为 Gorse 仪表盘接口需要的 JSON 结构。
func marshalGorseConfig(config *shopadminv1.ConfigResponse) ([]byte, error) {
	data, err := (protojson.MarshalOptions{UseProtoNames: true, EmitUnpopulated: true}).Marshal(config)
	if err != nil {
		return nil, err
	}

	record := make(map[string]interface{})
	err = json.Unmarshal(data, &record)
	if err != nil {
		return nil, err
	}

	recommend, matched := record["recommend"].(map[string]interface{})
	// 推荐器数组字段在 Gorse 配置中使用中划线命名，这里从 Proto 字段名转换回原始配置命名。
	if matched {
		renameJSONKey(recommend, "non_personalized", "non-personalized")
		renameJSONKey(recommend, "item_to_item", "item-to-item")
		renameJSONKey(recommend, "user_to_user", "user-to-user")
	}

	data, err = json.Marshal(record)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// redactGorseConfig 将 Gorse 配置中的基础设施凭据替换为固定占位符。
func redactGorseConfig(data []byte) ([]byte, error) {
	record, err := unmarshalGorseConfigMap(data)
	if err != nil {
		return nil, err
	}
	for _, path := range gorseConfigSecretPaths {
		redactGorseConfigValue(record, path)
	}
	return json.Marshal(record)
}

// unmarshalGorseConfigMap 将 Gorse 配置 JSON 转成可按路径处理的对象。
func unmarshalGorseConfigMap(data []byte) (map[string]interface{}, error) {
	record := make(map[string]interface{})
	err := json.Unmarshal(data, &record)
	if err != nil {
		return nil, err
	}
	return record, nil
}

// redactGorseConfigValue 对存在的敏感字段设置脱敏占位符。
func redactGorseConfigValue(record map[string]interface{}, path []string) {
	value, ok := readGorseConfigPath(record, path)
	if !ok || value == nil || value == "" {
		return
	}
	setGorseConfigPath(record, path, gorseConfigSecretMask)
}

// restoreMaskedGorseConfigValue 仅在新配置仍为脱敏占位符时恢复真实值。
func restoreMaskedGorseConfigValue(nextConfig map[string]interface{}, currentConfig map[string]interface{}, path []string) {
	nextValue, ok := readGorseConfigPath(nextConfig, path)
	if !ok || nextValue != gorseConfigSecretMask {
		return
	}
	currentValue, ok := readGorseConfigPath(currentConfig, path)
	if !ok {
		return
	}
	setGorseConfigPath(nextConfig, path, currentValue)
}

// readGorseConfigPath 从配置对象中按路径读取字段值。
func readGorseConfigPath(record map[string]interface{}, path []string) (interface{}, bool) {
	var value interface{} = record
	for _, key := range path {
		currentRecord, ok := value.(map[string]interface{})
		if !ok {
			return nil, false
		}
		value, ok = currentRecord[key]
		if !ok {
			return nil, false
		}
	}
	return value, true
}

// setGorseConfigPath 从配置对象中按路径写入字段值。
func setGorseConfigPath(record map[string]interface{}, path []string, value interface{}) {
	if len(path) == 0 {
		return
	}
	currentRecord := record
	for _, key := range path[:len(path)-1] {
		nextRecord, ok := currentRecord[key].(map[string]interface{})
		if !ok {
			nextRecord = make(map[string]interface{})
			currentRecord[key] = nextRecord
		}
		currentRecord = nextRecord
	}
	currentRecord[path[len(path)-1]] = value
}

// renameJSONKey 重命名 JSON 对象字段。
func renameJSONKey(record map[string]interface{}, oldKey string, newKey string) {
	value, exists := record[oldKey]
	// 原字段不存在时无需处理，避免额外写入空字段改变Gorse 配置语义。
	if !exists {
		return
	}
	record[newKey] = value
	delete(record, oldKey)
}
