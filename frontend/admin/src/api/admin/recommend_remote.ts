import service from "@/utils/request";
import type {
  RecommendRemoteCursorRequest,
  RecommendRemoteDashboardItemsRequest,
  RecommendRemoteDataRequest,
  RecommendRemoteIdRequest,
  RecommendRemoteImportRequest,
  RecommendRemoteJsonRequest,
  RecommendRemoteJsonResponse,
  RecommendRemoteNameRequest,
  RecommendRemoteService
} from "@/rpc/admin/recommend_remote";
import type { Empty } from "@/rpc/google/protobuf/empty";

const RECOMMEND_REMOTE_URL = "/admin/recommend/remote";

/** 远程推荐管理服务。 */
export class RecommendRemoteServiceImpl implements RecommendRemoteService {
  /** 查询远程推荐概览。 */
  GetRecommendRemoteOverview(request: Empty): Promise<RecommendRemoteJsonResponse> {
    return service<Empty, RecommendRemoteJsonResponse>({
      url: `${RECOMMEND_REMOTE_URL}/overview`,
      method: "get",
      params: request
    });
  }

  /** 查询远程推荐任务状态。 */
  GetRecommendRemoteTasks(request: Empty): Promise<RecommendRemoteJsonResponse> {
    return service<Empty, RecommendRemoteJsonResponse>({
      url: `${RECOMMEND_REMOTE_URL}/tasks`,
      method: "get",
      params: request
    });
  }

  /** 查询远程推荐分类。 */
  GetRecommendRemoteCategories(request: Empty): Promise<RecommendRemoteJsonResponse> {
    return service<Empty, RecommendRemoteJsonResponse>({
      url: `${RECOMMEND_REMOTE_URL}/categories`,
      method: "get",
      params: request
    });
  }

  /** 查询远程推荐时间序列。 */
  GetRecommendRemoteTimeseries(request: RecommendRemoteNameRequest): Promise<RecommendRemoteJsonResponse> {
    return service<RecommendRemoteNameRequest, RecommendRemoteJsonResponse>({
      url: `${RECOMMEND_REMOTE_URL}/timeseries/${encodeURIComponent(request.name)}`,
      method: "get",
      params: {
        begin: request.begin,
        end: request.end
      }
    });
  }

  /** 查询远程推荐仪表盘推荐商品。 */
  GetRecommendRemoteDashboardItems(request: RecommendRemoteDashboardItemsRequest): Promise<RecommendRemoteJsonResponse> {
    return service<RecommendRemoteDashboardItemsRequest, RecommendRemoteJsonResponse>({
      url: `${RECOMMEND_REMOTE_URL}/dashboard`,
      method: "get",
      params: {
        recommender: request.recommender,
        category: request.category,
        end: request.end
      }
    });
  }

  /** 查询远程推荐用户列表。 */
  PageRecommendRemoteUsers(request: RecommendRemoteCursorRequest): Promise<RecommendRemoteJsonResponse> {
    return service<RecommendRemoteCursorRequest, RecommendRemoteJsonResponse>({
      url: `${RECOMMEND_REMOTE_URL}/users`,
      method: "get",
      params: request
    });
  }

  /** 查询远程推荐用户。 */
  GetRecommendRemoteUser(request: RecommendRemoteIdRequest): Promise<RecommendRemoteJsonResponse> {
    return service<RecommendRemoteIdRequest, RecommendRemoteJsonResponse>({
      url: `${RECOMMEND_REMOTE_URL}/users/${encodeURIComponent(request.id)}`,
      method: "get"
    });
  }

  /** 删除远程推荐用户。 */
  DeleteRecommendRemoteUser(request: RecommendRemoteIdRequest): Promise<Empty> {
    return service<RecommendRemoteIdRequest, Empty>({
      url: `${RECOMMEND_REMOTE_URL}/users/${encodeURIComponent(request.id)}`,
      method: "delete"
    });
  }

  /** 查询远程推荐商品列表。 */
  PageRecommendRemoteItems(request: RecommendRemoteCursorRequest): Promise<RecommendRemoteJsonResponse> {
    return service<RecommendRemoteCursorRequest, RecommendRemoteJsonResponse>({
      url: `${RECOMMEND_REMOTE_URL}/items`,
      method: "get",
      params: request
    });
  }

  /** 查询远程推荐商品。 */
  GetRecommendRemoteItem(request: RecommendRemoteIdRequest): Promise<RecommendRemoteJsonResponse> {
    return service<RecommendRemoteIdRequest, RecommendRemoteJsonResponse>({
      url: `${RECOMMEND_REMOTE_URL}/items/${encodeURIComponent(request.id)}`,
      method: "get"
    });
  }

  /** 删除远程推荐商品。 */
  DeleteRecommendRemoteItem(request: RecommendRemoteIdRequest): Promise<Empty> {
    return service<RecommendRemoteIdRequest, Empty>({
      url: `${RECOMMEND_REMOTE_URL}/items/${encodeURIComponent(request.id)}`,
      method: "delete"
    });
  }

  /** 导出远程推荐数据。 */
  ExportRecommendRemoteData(request: RecommendRemoteDataRequest): Promise<RecommendRemoteJsonResponse> {
    return service<RecommendRemoteDataRequest, RecommendRemoteJsonResponse>({
      url: `${RECOMMEND_REMOTE_URL}/advance/export`,
      method: "get",
      params: request
    });
  }

  /** 导入远程推荐数据。 */
  ImportRecommendRemoteData(request: RecommendRemoteImportRequest): Promise<Empty> {
    return service<RecommendRemoteImportRequest, Empty>({
      url: `${RECOMMEND_REMOTE_URL}/advance/import`,
      method: "post",
      data: request
    });
  }

  /** 查询推荐编排配置。 */
  GetRecommendRemoteFlowConfig(request: Empty): Promise<RecommendRemoteJsonResponse> {
    return service<Empty, RecommendRemoteJsonResponse>({
      url: `${RECOMMEND_REMOTE_URL}/flow/config`,
      method: "get",
      params: request
    });
  }

  /** 保存推荐编排配置。 */
  SaveRecommendRemoteFlowConfig(request: RecommendRemoteJsonRequest): Promise<Empty> {
    return service<RecommendRemoteJsonRequest, Empty>({
      url: `${RECOMMEND_REMOTE_URL}/flow/config`,
      method: "post",
      data: request
    });
  }

  /** 重置推荐编排配置。 */
  ResetRecommendRemoteFlowConfig(request: Empty): Promise<Empty> {
    return service<Empty, Empty>({
      url: `${RECOMMEND_REMOTE_URL}/flow/config`,
      method: "delete",
      data: request
    });
  }

  /** 查询推荐编排配置结构。 */
  GetRecommendRemoteFlowSchema(request: Empty): Promise<RecommendRemoteJsonResponse> {
    return service<Empty, RecommendRemoteJsonResponse>({
      url: `${RECOMMEND_REMOTE_URL}/flow/schema`,
      method: "get",
      params: request
    });
  }

  /** 查询远程推荐配置。 */
  GetRecommendRemoteConfig(request: Empty): Promise<RecommendRemoteJsonResponse> {
    return service<Empty, RecommendRemoteJsonResponse>({
      url: `${RECOMMEND_REMOTE_URL}/config`,
      method: "get",
      params: request
    });
  }
}

export const defRecommendRemoteService = new RecommendRemoteServiceImpl();
