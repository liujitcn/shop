package biz

import (
	"context"
	"encoding/json"
	"strings"

	_const "shop/pkg/const"

	adminv1 "shop/api/gen/go/admin/v1"
	commonv1 "shop/api/gen/go/common/v1"
	"shop/pkg/errorsx"
	"shop/pkg/recommend/gorse"

	client "github.com/gorse-io/gorse-go"
	"google.golang.org/protobuf/encoding/protojson"
)

const RECOMMEND_GORSE_ADVANCE_PAGE_SIZE = 100

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

// GetTimeSeries 查询 Gorse 推荐单项时间序列。
func (c *RecommendGorseCase) GetTimeSeries(ctx context.Context, name, begin, end string) (*adminv1.TimeSeriesResponse, error) {
	name = strings.TrimSpace(name)
	// 指标名称为空时，无法拼装Gorse 仪表盘 API 路径。
	if name == "" {
		return nil, errorsx.InvalidArgument("指标名称不能为空")
	}

	data, err := c.dashboard.TimeSeries(ctx, name, strings.TrimSpace(begin), strings.TrimSpace(end))
	if err != nil {
		return nil, err
	}

	response := new(adminv1.TimeSeriesResponse)
	// Gorse 时间序列接口原始返回数组，Proto 响应需要外层字段承载，内部字段通过 json_name 直接对齐原始结构。
	if strings.TrimSpace(string(data)) == "null" {
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

// OptionCategories 查询 Gorse 推荐分类列表。
func (c *RecommendGorseCase) OptionCategories(ctx context.Context) (*adminv1.OptionCategoriesResponse, error) {
	data, err := c.dashboard.Categories(ctx)
	if err != nil {
		return nil, err
	}

	response := new(adminv1.OptionCategoriesResponse)
	err = (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal([]byte(`{"categories":`+string(data)+`}`), response)
	if err != nil {
		return nil, err
	}
	return response, nil
}

// ListDashboardItems 查询 Gorse 推荐仪表盘推荐商品。
func (c *RecommendGorseCase) ListDashboardItems(
	ctx context.Context,
	recommender string,
	category string,
	end int32,
) (*adminv1.ListDashboardItemsResponse, error) {
	data, err := c.dashboard.DashboardItems(ctx, recommender, category, end)
	if err != nil {
		return nil, err
	}

	response := new(adminv1.ListDashboardItemsResponse)
	// Gorse 仪表盘推荐商品接口原始返回数组，字段结构通过 json_name 与 Proto 商品结构保持一致。
	if strings.TrimSpace(string(data)) == "null" {
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

// ListTasks 查询 Gorse 推荐任务状态。
func (c *RecommendGorseCase) ListTasks(ctx context.Context) (*adminv1.ListTasksResponse, error) {
	data, err := c.dashboard.Tasks(ctx)
	if err != nil {
		return nil, err
	}

	response := new(adminv1.ListTasksResponse)
	// Gorse 任务状态接口原始返回数组，任务字段通过 json_name 直接对齐 Tracer、Name 等原始字段。
	if strings.TrimSpace(string(data)) == "null" {
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

// PageUsers 查询 Gorse 推荐用户列表。
func (c *RecommendGorseCase) PageUsers(ctx context.Context, cursor string, n int32) (*adminv1.PageUsersResponse, error) {
	data, err := c.dashboard.Users(ctx, strings.TrimSpace(cursor), n)
	if err != nil {
		return nil, err
	}

	response := new(adminv1.PageUsersResponse)
	err = (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(data, response)
	if err != nil {
		return nil, err
	}
	return response, nil
}

// GetUser 查询 Gorse 推荐用户。
func (c *RecommendGorseCase) GetUser(ctx context.Context, id string) (*adminv1.UserResponse, error) {
	id = strings.TrimSpace(id)
	// 用户编号为空时，无法继续代理 Gorse用户详情接口。
	if id == "" {
		return nil, errorsx.InvalidArgument("用户编号不能为空")
	}

	data, err := c.dashboard.User(ctx, id)
	if err != nil {
		return nil, err
	}

	response := new(adminv1.UserResponse)
	err = (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(data, response)
	if err != nil {
		return nil, err
	}
	return response, nil
}

// DeleteUser 删除 Gorse 推荐用户。
func (c *RecommendGorseCase) DeleteUser(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	// 用户编号为空时，无法继续代理 Gorse用户删除接口。
	if id == "" {
		return errorsx.InvalidArgument("用户编号不能为空")
	}

	_, err := c.dashboard.DeleteUser(ctx, id)
	if err != nil {
		return err
	}
	return nil
}

// GetUserSimilar 查询 Gorse 推荐相似用户。
func (c *RecommendGorseCase) GetUserSimilar(ctx context.Context, id, recommender string) (*adminv1.UserSimilarResponse, error) {
	id = strings.TrimSpace(id)
	// 用户编号为空时，无法继续代理 Gorse相似用户接口。
	if id == "" {
		return nil, errorsx.InvalidArgument("用户编号不能为空")
	}

	data, err := c.dashboard.UserSimilar(ctx, id, strings.TrimSpace(recommender))
	if err != nil {
		return nil, err
	}

	response := new(adminv1.UserSimilarResponse)
	// Gorse 相似用户接口原始返回数组，用户字段通过 json_name 直接对齐原始结构。
	if strings.TrimSpace(string(data)) == "null" {
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

// GetUserFeedback 查询 Gorse 推荐用户反馈。
func (c *RecommendGorseCase) GetUserFeedback(
	ctx context.Context,
	id string,
	feedbackType string,
	offset int32,
	n int32,
) (*adminv1.FeedbackResponse, error) {
	id = strings.TrimSpace(id)
	// 用户编号为空时，无法继续代理 Gorse用户反馈接口。
	if id == "" {
		return nil, errorsx.InvalidArgument("用户编号不能为空")
	}

	data, err := c.dashboard.UserFeedback(ctx, id, strings.TrimSpace(feedbackType), offset, n)
	if err != nil {
		return nil, err
	}

	response := new(adminv1.FeedbackResponse)
	// Gorse 用户反馈接口原始返回数组，反馈字段通过 json_name 直接对齐 FeedbackType、Item 等原始字段。
	if strings.TrimSpace(string(data)) == "null" {
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
) (*adminv1.ItemListResponse, error) {
	id = strings.TrimSpace(id)
	// 用户编号为空时，无法继续代理 Gorse用户推荐接口。
	if id == "" {
		return nil, errorsx.InvalidArgument("用户编号不能为空")
	}

	data, err := c.dashboard.UserRecommend(ctx, id, strings.TrimSpace(recommender), strings.TrimSpace(category), n)
	if err != nil {
		return nil, err
	}

	response := new(adminv1.ItemListResponse)
	// Gorse 用户推荐接口原始返回数组，商品字段通过 json_name 直接对齐原始结构。
	if strings.TrimSpace(string(data)) == "null" {
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

// PageItems 查询 Gorse 推荐商品列表。
func (c *RecommendGorseCase) PageItems(ctx context.Context, cursor string, n int32) (*adminv1.PageItemsResponse, error) {
	data, err := c.dashboard.Items(ctx, strings.TrimSpace(cursor), n)
	if err != nil {
		return nil, err
	}

	response := new(adminv1.PageItemsResponse)
	err = (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(data, response)
	if err != nil {
		return nil, err
	}
	return response, nil
}

// GetItem 查询 Gorse 推荐商品。
func (c *RecommendGorseCase) GetItem(ctx context.Context, id string) (*adminv1.Item, error) {
	id = strings.TrimSpace(id)
	// 商品编号为空时，无法继续代理 Gorse商品详情接口。
	if id == "" {
		return nil, errorsx.InvalidArgument("商品编号不能为空")
	}

	data, err := c.dashboard.Item(ctx, id)
	if err != nil {
		return nil, err
	}

	response := new(adminv1.Item)
	err = (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(data, response)
	if err != nil {
		return nil, err
	}
	return response, nil
}

// DeleteItem 删除 Gorse 推荐商品。
func (c *RecommendGorseCase) DeleteItem(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	// 商品编号为空时，无法继续代理 Gorse商品删除接口。
	if id == "" {
		return errorsx.InvalidArgument("商品编号不能为空")
	}

	_, err := c.dashboard.DeleteItem(ctx, id)
	if err != nil {
		return err
	}
	return nil
}

// GetItemSimilar 查询 Gorse 推荐相似商品。
func (c *RecommendGorseCase) GetItemSimilar(ctx context.Context, id, recommender, category string) (*adminv1.ItemListResponse, error) {
	id = strings.TrimSpace(id)
	// 商品编号为空时，无法继续代理 Gorse相似商品接口。
	if id == "" {
		return nil, errorsx.InvalidArgument("商品编号不能为空")
	}

	data, err := c.dashboard.ItemSimilar(ctx, id, strings.TrimSpace(recommender), strings.TrimSpace(category))
	if err != nil {
		return nil, err
	}

	response := new(adminv1.ItemListResponse)
	// Gorse 相似商品接口原始返回数组，商品字段通过 json_name 直接对齐原始结构。
	if strings.TrimSpace(string(data)) == "null" {
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

// ExportData 导出 Gorse 推荐数据。
func (c *RecommendGorseCase) ExportData(
	ctx context.Context,
	req *adminv1.ExportDataRequest,
) (*adminv1.ExportDataResponse, error) {
	// 导出请求为空时，无法判断需要导出的数据类型。
	if req == nil {
		return nil, errorsx.InvalidArgument("导出请求不能为空")
	}
	// 数据类型非法时，无法确定需要导出的Gorse 推荐数据集。
	if req.GetDataType() == commonv1.AdvanceDataType(_const.ADVANCE_DATA_TYPE_UNKNOWN) {
		return nil, errorsx.InvalidArgument("导出数据类型不能为空")
	}

	// 不同高级调试数据类型分别走各自的 JSONL 导出逻辑。
	switch req.GetDataType() {
	case commonv1.AdvanceDataType(_const.ADVANCE_DATA_TYPE_USER):
		content, exportErr := c.exportRecommendGorseUsers(ctx)
		if exportErr != nil {
			return nil, exportErr
		}
		return &adminv1.ExportDataResponse{
			FileName: "users.jsonl",
			Content:  content,
		}, nil
	case commonv1.AdvanceDataType(_const.ADVANCE_DATA_TYPE_ITEM):
		content, exportErr := c.exportRecommendGorseItems(ctx)
		if exportErr != nil {
			return nil, exportErr
		}
		return &adminv1.ExportDataResponse{
			FileName: "items.jsonl",
			Content:  content,
		}, nil
	case commonv1.AdvanceDataType(_const.ADVANCE_DATA_TYPE_FEEDBACK):
		content, exportErr := c.exportRecommendGorseFeedback(ctx)
		if exportErr != nil {
			return nil, exportErr
		}
		return &adminv1.ExportDataResponse{
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
	req *adminv1.ImportDataRequest,
) (*adminv1.ImportDataResponse, error) {
	// 导入请求为空时，无法解析 JSONL 文件内容。
	if req == nil {
		return nil, errorsx.InvalidArgument("导入请求不能为空")
	}
	// 数据类型非法时，无法确定 JSONL 应该写入哪类Gorse 推荐数据。
	if req.GetDataType() == commonv1.AdvanceDataType(_const.ADVANCE_DATA_TYPE_UNKNOWN) {
		return nil, errorsx.InvalidArgument("导入数据类型不能为空")
	}
	// 文件内容为空时，没有可供导入的Gorse 推荐数据行。
	if strings.TrimSpace(req.GetContent()) == "" {
		return nil, errorsx.InvalidArgument("导入文件内容不能为空")
	}

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
	case commonv1.AdvanceDataType(_const.ADVANCE_DATA_TYPE_USER):
		successCount, err = c.importRecommendGorseUsers(ctx, recordList)
	case commonv1.AdvanceDataType(_const.ADVANCE_DATA_TYPE_ITEM):
		successCount, err = c.importRecommendGorseItems(ctx, recordList)
	case commonv1.AdvanceDataType(_const.ADVANCE_DATA_TYPE_FEEDBACK):
		successCount, err = c.importRecommendGorseFeedback(ctx, recordList)
	default:
		return nil, errorsx.InvalidArgument("暂不支持当前导入数据类型")
	}
	if err != nil {
		return nil, err
	}

	return &adminv1.ImportDataResponse{
		SuccessCount: int32(successCount),
	}, nil
}

// GetConfig 查询 Gorse 推荐配置。
func (c *RecommendGorseCase) GetConfig(ctx context.Context) (*adminv1.ConfigResponse, error) {
	data, err := c.dashboard.Config(ctx)
	if err != nil {
		return nil, err
	}

	config := new(adminv1.ConfigResponse)
	err = (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(data, config)
	if err != nil {
		return nil, err
	}
	return config, nil
}

// SaveConfig 保存 Gorse 推荐配置。
func (c *RecommendGorseCase) SaveConfig(ctx context.Context, req *adminv1.ConfigResponse) (*adminv1.ConfigResponse, error) {
	// 配置为空时，无法继续覆盖Gorse 推荐服务当前配置。
	if req == nil {
		return nil, errorsx.InvalidArgument("推荐配置不能为空")
	}

	body, err := marshalGorseConfig(req)
	if err != nil {
		return nil, err
	}

	data := make([]byte, 0)
	data, err = c.dashboard.SaveConfig(ctx, string(body))
	if err != nil {
		return nil, err
	}

	config := new(adminv1.ConfigResponse)
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
func (c *RecommendGorseCase) PreviewExternal(ctx context.Context, req *adminv1.PreviewExternalRequest) (*adminv1.PreviewExternalResponse, error) {
	if req == nil {
		req = &adminv1.PreviewExternalRequest{}
	}
	script := strings.TrimSpace(req.GetScript())
	// 外部推荐脚本为空时，Gorse 预览接口会执行失败，提前按参数错误拦截。
	if script == "" {
		return nil, errorsx.InvalidArgument("外部推荐脚本不能为空")
	}

	data, err := c.dashboard.ExternalPreview(ctx, req.GetUserId(), script)
	if err != nil {
		return nil, err
	}

	response := new(adminv1.PreviewExternalResponse)
	// Gorse 外部推荐预览接口原始返回数组，Proto 响应需要外层 items 字段承载。
	if strings.TrimSpace(string(data)) == "null" {
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
func (c *RecommendGorseCase) PreviewRankerPrompt(ctx context.Context, req *adminv1.PreviewRankerPromptRequest) (*adminv1.PreviewRankerPromptResponse, error) {
	// 请求为空时，按空参数代理预览接口，保持与 Gorse 仪表盘输入行为一致。
	if req == nil {
		req = &adminv1.PreviewRankerPromptRequest{}
	}
	queryTemplate := strings.TrimSpace(req.GetQueryTemplate())
	documentTemplate := strings.TrimSpace(req.GetDocumentTemplate())
	// 排序提示词预览只适用于大语言模型排序器，必须同时提供查询模板与文档模板。
	if queryTemplate == "" || documentTemplate == "" {
		return nil, errorsx.InvalidArgument("查询模板和文档模板不能为空")
	}

	data, err := c.dashboard.RankerPrompt(ctx, req.GetUserId(), queryTemplate, documentTemplate)
	if err != nil {
		return nil, err
	}

	response := new(adminv1.PreviewRankerPromptResponse)
	// Gorse 返回 null 时，Admin 侧使用空响应避免前端出现反序列化错误。
	if strings.TrimSpace(string(data)) == "null" {
		return response, nil
	}
	err = (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(data, response)
	if err != nil {
		return nil, err
	}
	return response, nil
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
		if strings.TrimSpace(result.Cursor) == "" {
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
		if strings.TrimSpace(result.Cursor) == "" {
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
		if strings.TrimSpace(result.Cursor) == "" {
			break
		}
		cursor = result.Cursor
	}
	return marshalRecommendGorseJSONL(feedbackList)
}

// importRecommendGorseUsers 导入 Gorse 推荐用户 JSONL。
func (c *RecommendGorseCase) importRecommendGorseUsers(ctx context.Context, recordList []json.RawMessage) (int, error) {
	userList := make([]client.User, 0, len(recordList))
	for _, record := range recordList {
		user := client.User{}
		decodeErr := json.Unmarshal(record, &user)
		if decodeErr != nil {
			return 0, errorsx.InvalidArgument("用户数据不是合法的JSON对象")
		}
		// 用户编号为空时，当前记录无法作为合法的Gorse 推荐用户导入。
		if strings.TrimSpace(user.UserId) == "" {
			return 0, errorsx.InvalidArgument("用户数据缺少 user_id")
		}
		userList = append(userList, user)
	}
	// 有效用户对象为空时，不应该继续发起Gorse导入请求。
	if len(userList) == 0 {
		return 0, errorsx.InvalidArgument("导入文件缺少有效用户数据")
	}

	err := c.dashboard.InsertUsers(ctx, userList)
	if err != nil {
		return 0, err
	}
	return len(userList), nil
}

// importRecommendGorseItems 导入 Gorse 推荐商品 JSONL。
func (c *RecommendGorseCase) importRecommendGorseItems(ctx context.Context, recordList []json.RawMessage) (int, error) {
	itemList := make([]client.Item, 0, len(recordList))
	for _, record := range recordList {
		item := client.Item{}
		decodeErr := json.Unmarshal(record, &item)
		if decodeErr != nil {
			return 0, errorsx.InvalidArgument("商品数据不是合法的JSON对象")
		}
		// 商品编号为空时，当前记录无法作为合法的Gorse 推荐商品导入。
		if strings.TrimSpace(item.ItemId) == "" {
			return 0, errorsx.InvalidArgument("商品数据缺少 item_id")
		}
		itemList = append(itemList, item)
	}
	// 有效商品对象为空时，不应该继续发起Gorse导入请求。
	if len(itemList) == 0 {
		return 0, errorsx.InvalidArgument("导入文件缺少有效商品数据")
	}

	err := c.dashboard.InsertItems(ctx, itemList)
	if err != nil {
		return 0, err
	}
	return len(itemList), nil
}

// importRecommendGorseFeedback 导入 Gorse 推荐反馈 JSONL。
func (c *RecommendGorseCase) importRecommendGorseFeedback(ctx context.Context, recordList []json.RawMessage) (int, error) {
	feedbackList := make([]client.Feedback, 0, len(recordList))
	for _, record := range recordList {
		feedback := client.Feedback{}
		decodeErr := json.Unmarshal(record, &feedback)
		if decodeErr != nil {
			return 0, errorsx.InvalidArgument("反馈数据不是合法的JSON对象")
		}
		// 反馈类型为空时，当前记录无法作为合法的Gorse 推荐反馈导入。
		if strings.TrimSpace(feedback.FeedbackType) == "" {
			return 0, errorsx.InvalidArgument("反馈数据缺少 feedback_type")
		}
		// 用户编号为空时，当前反馈无法定位反馈主体。
		if strings.TrimSpace(feedback.UserId) == "" {
			return 0, errorsx.InvalidArgument("反馈数据缺少 user_id")
		}
		// 商品编号为空时，当前反馈无法定位反馈商品。
		if strings.TrimSpace(feedback.ItemId) == "" {
			return 0, errorsx.InvalidArgument("反馈数据缺少 item_id")
		}
		feedbackList = append(feedbackList, feedback)
	}
	// 有效反馈对象为空时，不应该继续发起Gorse导入请求。
	if len(feedbackList) == 0 {
		return 0, errorsx.InvalidArgument("导入文件缺少有效反馈数据")
	}

	err := c.dashboard.InsertFeedbacks(ctx, feedbackList)
	if err != nil {
		return 0, err
	}
	return len(feedbackList), nil
}

// parseRecommendGorseJSONRecords 解析Gorse 推荐 JSON 或 JSONL 内容。
func parseRecommendGorseJSONRecords(content string) ([]json.RawMessage, error) {
	content = strings.TrimSpace(strings.TrimPrefix(content, "\ufeff"))
	// 处理完 BOM 与首尾空白后内容为空时，不存在任何可解析的数据对象。
	if content == "" {
		return nil, errorsx.InvalidArgument("导入文件内容不能为空")
	}

	// 文件整体是 JSON 数组时，优先按数组结构解析，兼容部分调试工具的批量导出格式。
	if strings.HasPrefix(content, "[") {
		recordList := make([]json.RawMessage, 0)
		decodeErr := json.Unmarshal([]byte(content), &recordList)
		if decodeErr != nil {
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
		line = strings.TrimSpace(line)
		// 空白行对 JSONL 导入没有业务意义，统一跳过。
		if line == "" {
			continue
		}
		// JSONL 每一行都必须是单个 JSON 对象，避免把半截数据误导入推荐服务。
		if !strings.HasPrefix(line, "{") {
			return nil, errorsx.InvalidArgument("导入文件不是合法的JSONL格式")
		}

		rawMessage := json.RawMessage{}
		decodeErr := json.Unmarshal([]byte(line), &rawMessage)
		if decodeErr != nil {
			return nil, errorsx.InvalidArgument("导入文件不是合法的JSONL格式")
		}
		recordList = append(recordList, rawMessage)
	}
	return recordList, nil
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

// marshalGorseConfig 将 Proto 配置转换为 Gorse 仪表盘接口需要的 JSON 结构。
func marshalGorseConfig(config *adminv1.ConfigResponse) ([]byte, error) {
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
