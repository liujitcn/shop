import service from "@/utils/request";
import type {
  OptionCategoryRequest,
  OptionCategoryResponse,
  ConfigResponse,
  DeleteItemRequest,
  DeleteUserRequest,
  ListDashboardItemRequest,
  ListDashboardItemResponse,
  ExportDataRequest,
  ExportDataResponse,
  FeedbackResponse,
  GetConfigRequest,
  GetItemRequest,
  GetUserRequest,
  ImportDataRequest,
  ImportDataResponse,
  ItemListResponse,
  ListTaskResponse,
  GetTimeSeriesRequest,
  Item,
  PageItemRequest,
  PageItemResponse,
  ListTaskRequest,
  PageUserRequest,
  PageUserResponse,
  PreviewExternalRequest,
  PreviewExternalResponse,
  PreviewRankerPromptRequest,
  PreviewRankerPromptResponse,
  RecommendGorseService,
  ResetConfigRequest,
  SaveConfigRequest,
  GetUserRecommendRequest,
  GetItemSimilarRequest,
  GetUserSimilarRequest,
  TimeSeriesResponse,
  GetUserFeedbackRequest,
  UserResponse,
  UserSimilarResponse
} from "@/rpc/shop/admin/v1/recommend_gorse";
import type { Empty } from "@/rpc/google/protobuf/empty";

const RECOMMEND_GORSE_URL = "/v1/admin/recommend/gorse";

/** Gorse 推荐管理服务。 */
export class RecommendGorseServiceImpl implements RecommendGorseService {
  /** 查询 Gorse 推荐时间序列。 */
  GetTimeSeries(request: GetTimeSeriesRequest): Promise<TimeSeriesResponse> {
    return service<GetTimeSeriesRequest, TimeSeriesResponse>({
      url: `${RECOMMEND_GORSE_URL}/timeseries/${encodeURIComponent(request.name)}`,
      method: "get",
      params: {
        begin: request.begin,
        end: request.end
      }
    });
  }

  /** 查询 Gorse 推荐分类。 */
  OptionCategory(request: OptionCategoryRequest): Promise<OptionCategoryResponse> {
    return service<OptionCategoryRequest, OptionCategoryResponse>({
      url: `${RECOMMEND_GORSE_URL}/category/option`,
      method: "get",
      params: request
    });
  }

  /** 查询 Gorse 推荐仪表盘推荐商品。 */
  ListDashboardItem(request: ListDashboardItemRequest): Promise<ListDashboardItemResponse> {
    return service<ListDashboardItemRequest, ListDashboardItemResponse>({
      url: `${RECOMMEND_GORSE_URL}/dashboard`,
      method: "get",
      params: request
    });
  }

  /** 查询 Gorse 推荐配置。 */
  GetConfig(request: GetConfigRequest): Promise<ConfigResponse> {
    return service<GetConfigRequest, ConfigResponse>({
      url: `${RECOMMEND_GORSE_URL}/config`,
      method: "get",
      params: request
    });
  }

  /** 保存 Gorse 推荐配置。 */
  SaveConfig(request: SaveConfigRequest): Promise<ConfigResponse> {
    return service<ConfigResponse | undefined, ConfigResponse>({
      url: `${RECOMMEND_GORSE_URL}/config`,
      method: "post",
      data: request.config
    });
  }

  /** 重置 Gorse 推荐配置。 */
  ResetConfig(request: ResetConfigRequest): Promise<Empty> {
    return service<ResetConfigRequest, Empty>({
      url: `${RECOMMEND_GORSE_URL}/config`,
      method: "delete",
      params: request
    });
  }

  /** 预览 Gorse 推荐外部推荐脚本。 */
  PreviewExternal(request: PreviewExternalRequest): Promise<PreviewExternalResponse> {
    return service<PreviewExternalRequest, PreviewExternalResponse>({
      url: `${RECOMMEND_GORSE_URL}/external/preview`,
      method: "post",
      data: request
    });
  }

  /** 预览 Gorse 推荐排序提示词。 */
  PreviewRankerPrompt(request: PreviewRankerPromptRequest): Promise<PreviewRankerPromptResponse> {
    return service<PreviewRankerPromptRequest, PreviewRankerPromptResponse>({
      url: `${RECOMMEND_GORSE_URL}/ranker/prompt`,
      method: "post",
      data: request
    });
  }

  /** 查询 Gorse 推荐任务状态。 */
  ListTask(request: ListTaskRequest): Promise<ListTaskResponse> {
    return service<ListTaskRequest, ListTaskResponse>({
      url: `${RECOMMEND_GORSE_URL}/task`,
      method: "get",
      params: request
    });
  }

  /** 查询 Gorse 推荐用户列表。 */
  PageUser(request: PageUserRequest): Promise<PageUserResponse> {
    return service<PageUserRequest, PageUserResponse>({
      url: `${RECOMMEND_GORSE_URL}/user`,
      method: "get",
      params: request
    });
  }

  /** 查询 Gorse 推荐用户。 */
  GetUser(request: GetUserRequest): Promise<UserResponse> {
    return service<GetUserRequest, UserResponse>({
      url: `${RECOMMEND_GORSE_URL}/user/${encodeURIComponent(request.id)}`,
      method: "get"
    });
  }

  /** 删除 Gorse 推荐用户。 */
  DeleteUser(request: DeleteUserRequest): Promise<Empty> {
    return service<DeleteUserRequest, Empty>({
      url: `${RECOMMEND_GORSE_URL}/user/${encodeURIComponent(request.id)}`,
      method: "delete"
    });
  }

  /** 查询 Gorse 推荐相似用户。 */
  GetUserSimilar(request: GetUserSimilarRequest): Promise<UserSimilarResponse> {
    return service<GetUserSimilarRequest, UserSimilarResponse>({
      url: `${RECOMMEND_GORSE_URL}/user/${encodeURIComponent(request.id)}/similar`,
      method: "get",
      params: {
        recommender: request.recommender,
        category: request.category
      }
    });
  }

  /** 查询 Gorse 推荐用户反馈。 */
  GetUserFeedback(request: GetUserFeedbackRequest): Promise<FeedbackResponse> {
    return service<GetUserFeedbackRequest, FeedbackResponse>({
      url: `${RECOMMEND_GORSE_URL}/user/${encodeURIComponent(request.id)}/feedback`,
      method: "get",
      params: {
        feedback_type: request.feedback_type,
        offset: request.offset,
        n: request.n
      }
    });
  }

  /** 查询 Gorse 推荐用户推荐结果。 */
  GetUserRecommend(request: GetUserRecommendRequest): Promise<ItemListResponse> {
    return service<GetUserRecommendRequest, ItemListResponse>({
      url: `${RECOMMEND_GORSE_URL}/user/${encodeURIComponent(request.id)}/recommend`,
      method: "get",
      params: {
        recommender: request.recommender,
        category: request.category,
        n: request.n
      }
    });
  }

  /** 查询 Gorse 推荐商品列表。 */
  PageItem(request: PageItemRequest): Promise<PageItemResponse> {
    return service<PageItemRequest, PageItemResponse>({
      url: `${RECOMMEND_GORSE_URL}/item`,
      method: "get",
      params: request
    });
  }

  /** 查询 Gorse 推荐商品。 */
  GetItem(request: GetItemRequest): Promise<Item> {
    return service<GetItemRequest, Item>({
      url: `${RECOMMEND_GORSE_URL}/item/${encodeURIComponent(request.id)}`,
      method: "get"
    });
  }

  /** 删除 Gorse 推荐商品。 */
  DeleteItem(request: DeleteItemRequest): Promise<Empty> {
    return service<DeleteItemRequest, Empty>({
      url: `${RECOMMEND_GORSE_URL}/item/${encodeURIComponent(request.id)}`,
      method: "delete"
    });
  }

  /** 查询 Gorse 推荐相似商品。 */
  GetItemSimilar(request: GetItemSimilarRequest): Promise<ItemListResponse> {
    return service<GetItemSimilarRequest, ItemListResponse>({
      url: `${RECOMMEND_GORSE_URL}/item/${encodeURIComponent(request.id)}/similar`,
      method: "get",
      params: {
        recommender: request.recommender,
        category: request.category
      }
    });
  }

  /** 导出 Gorse 推荐数据。 */
  ExportData(request: ExportDataRequest): Promise<ExportDataResponse> {
    return service<ExportDataRequest, ExportDataResponse>({
      url: `${RECOMMEND_GORSE_URL}/export`,
      method: "get",
      params: request
    });
  }

  /** 导入 Gorse 推荐数据。 */
  ImportData(request: ImportDataRequest): Promise<ImportDataResponse> {
    return service<ImportDataRequest, ImportDataResponse>({
      url: `${RECOMMEND_GORSE_URL}/import`,
      method: "post",
      data: request
    });
  }
}

export const defRecommendGorseService = new RecommendGorseServiceImpl();
