import service from "@/utils/request";
import type {
  CursorRequest,
  DashboardItemsRequest,
  DataRequest,
  FeedbackDeleteRequest,
  FeedbackRequest,
  IdRequest,
  ImportRequest,
  JsonRequest,
  CategoriesResponse,
  ConfigResponse,
  DataPageResponse,
  FeedbackPageResponse,
  Item,
  ItemsPageResponse,
  OverviewResponse,
  RecordsResponse,
  TasksResponse,
  TimeseriesResponse,
  User,
  UsersPageResponse,
  NeighborRequest,
  NameRequest,
  PurgeRequest,
  RecommendationRequest,
  RecommendRemoteService
} from "@/rpc/admin/recommend_remote";
import type { Empty } from "@/rpc/google/protobuf/empty";

const RECOMMEND_REMOTE_URL = "/admin/recommend/remote";

/** 远程推荐管理服务。 */
export class RecommendRemoteServiceImpl implements RecommendRemoteService {
  /** 查询远程推荐概览。 */
  GetOverview(request: Empty): Promise<OverviewResponse> {
    return service<Empty, OverviewResponse>({
      url: `${RECOMMEND_REMOTE_URL}/overview`,
      method: "get",
      params: request
    });
  }

  /** 查询远程推荐任务状态。 */
  GetTask(request: Empty): Promise<TasksResponse> {
    return service<Empty, TasksResponse>({
      url: `${RECOMMEND_REMOTE_URL}/task`,
      method: "get",
      params: request
    });
  }

  /** 查询远程推荐分类。 */
  GetCategory(request: Empty): Promise<CategoriesResponse> {
    return service<Empty, CategoriesResponse>({
      url: `${RECOMMEND_REMOTE_URL}/category`,
      method: "get",
      params: request
    });
  }

  /** 查询远程推荐时间序列。 */
  GetTimeseries(request: NameRequest): Promise<TimeseriesResponse> {
    return service<NameRequest, TimeseriesResponse>({
      url: `${RECOMMEND_REMOTE_URL}/timeseries/${encodeURIComponent(request.name)}`,
      method: "get",
      params: {
        begin: request.begin,
        end: request.end
      }
    });
  }

  /** 查询远程推荐仪表盘推荐商品。 */
  GetDashboardItems(request: DashboardItemsRequest): Promise<RecordsResponse> {
    return service<DashboardItemsRequest, RecordsResponse>({
      url: `${RECOMMEND_REMOTE_URL}/dashboard`,
      method: "get",
      params: {
        recommender: request.recommender,
        category: request.category,
        end: request.end
      }
    });
  }

  /** 查询远程推荐结果。 */
  GetRecommendation(request: RecommendationRequest): Promise<RecordsResponse> {
    return service<RecommendationRequest, RecordsResponse>({
      url: `${RECOMMEND_REMOTE_URL}/recommendation`,
      method: "get",
      params: request
    });
  }

  /** 查询远程相似内容。 */
  GetNeighbor(request: NeighborRequest): Promise<RecordsResponse> {
    return service<NeighborRequest, RecordsResponse>({
      url: `${RECOMMEND_REMOTE_URL}/neighbor`,
      method: "get",
      params: request
    });
  }

  /** 查询远程推荐反馈列表。 */
  PageFeedback(request: FeedbackRequest): Promise<FeedbackPageResponse> {
    return service<FeedbackRequest, FeedbackPageResponse>({
      url: `${RECOMMEND_REMOTE_URL}/feedback`,
      method: "get",
      params: request
    });
  }

  /** 写入远程推荐反馈。 */
  ImportFeedback(request: JsonRequest): Promise<Empty> {
    return service<JsonRequest, Empty>({
      url: `${RECOMMEND_REMOTE_URL}/feedback`,
      method: "post",
      data: request
    });
  }

  /** 删除远程推荐反馈。 */
  DeleteFeedback(request: FeedbackDeleteRequest): Promise<Empty> {
    return service<FeedbackDeleteRequest, Empty>({
      url: `${RECOMMEND_REMOTE_URL}/feedback`,
      method: "delete",
      params: request
    });
  }

  /** 查询远程推荐用户列表。 */
  PageUser(request: CursorRequest): Promise<UsersPageResponse> {
    return service<CursorRequest, UsersPageResponse>({
      url: `${RECOMMEND_REMOTE_URL}/user`,
      method: "get",
      params: request
    });
  }

  /** 查询远程推荐用户。 */
  GetUser(request: IdRequest): Promise<User> {
    return service<IdRequest, User>({
      url: `${RECOMMEND_REMOTE_URL}/user/${encodeURIComponent(request.id)}`,
      method: "get"
    });
  }

  /** 删除远程推荐用户。 */
  DeleteUser(request: IdRequest): Promise<Empty> {
    return service<IdRequest, Empty>({
      url: `${RECOMMEND_REMOTE_URL}/user/${encodeURIComponent(request.id)}`,
      method: "delete"
    });
  }

  /** 查询远程推荐商品列表。 */
  PageItem(request: CursorRequest): Promise<ItemsPageResponse> {
    return service<CursorRequest, ItemsPageResponse>({
      url: `${RECOMMEND_REMOTE_URL}/item`,
      method: "get",
      params: request
    });
  }

  /** 查询远程推荐商品。 */
  GetItem(request: IdRequest): Promise<Item> {
    return service<IdRequest, Item>({
      url: `${RECOMMEND_REMOTE_URL}/item/${encodeURIComponent(request.id)}`,
      method: "get"
    });
  }

  /** 删除远程推荐商品。 */
  DeleteItem(request: IdRequest): Promise<Empty> {
    return service<IdRequest, Empty>({
      url: `${RECOMMEND_REMOTE_URL}/item/${encodeURIComponent(request.id)}`,
      method: "delete"
    });
  }

  /** 导出远程推荐数据。 */
  ExportData(request: DataRequest): Promise<DataPageResponse> {
    return service<DataRequest, DataPageResponse>({
      url: `${RECOMMEND_REMOTE_URL}/advance/export`,
      method: "get",
      params: request
    });
  }

  /** 导入远程推荐数据。 */
  ImportData(request: ImportRequest): Promise<Empty> {
    return service<ImportRequest, Empty>({
      url: `${RECOMMEND_REMOTE_URL}/advance/import`,
      method: "post",
      data: request
    });
  }

  /** 清空远程推荐数据。 */
  PurgeData(request: PurgeRequest): Promise<Empty> {
    return service<PurgeRequest, Empty>({
      url: `${RECOMMEND_REMOTE_URL}/advance/purge`,
      method: "post",
      data: request
    });
  }

  /** 查询推荐编排配置。 */
  GetFlowConfig(request: Empty): Promise<ConfigResponse> {
    return service<Empty, ConfigResponse>({
      url: `${RECOMMEND_REMOTE_URL}/flow/config`,
      method: "get",
      params: request
    });
  }

  /** 保存推荐编排配置。 */
  SaveFlowConfig(request: JsonRequest): Promise<Empty> {
    return service<JsonRequest, Empty>({
      url: `${RECOMMEND_REMOTE_URL}/flow/config`,
      method: "post",
      data: request
    });
  }

  /** 重置推荐编排配置。 */
  ResetFlowConfig(request: Empty): Promise<Empty> {
    return service<Empty, Empty>({
      url: `${RECOMMEND_REMOTE_URL}/flow/config`,
      method: "delete",
      data: request
    });
  }

  /** 查询推荐编排配置结构。 */
  GetFlowSchema(request: Empty): Promise<ConfigResponse> {
    return service<Empty, ConfigResponse>({
      url: `${RECOMMEND_REMOTE_URL}/flow/schema`,
      method: "get",
      params: request
    });
  }

  /** 查询远程推荐配置。 */
  GetConfig(request: Empty): Promise<ConfigResponse> {
    return service<Empty, ConfigResponse>({
      url: `${RECOMMEND_REMOTE_URL}/config`,
      method: "get",
      params: request
    });
  }
}

export const defRecommendRemoteService = new RecommendRemoteServiceImpl();
