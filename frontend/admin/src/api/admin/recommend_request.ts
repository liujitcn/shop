import service from "@/utils/request";
import type {
  GetRecommendRequestEventRequest,
  GetRecommendRequestEventResponse,
  PageRecommendRequestRequest,
  PageRecommendRequestResponse,
  RecommendRequestDetailResponse,
  RecommendRequestService
} from "@/rpc/admin/recommend_request";
import type { Int64Value } from "@/rpc/google/protobuf/wrappers";

const RECOMMEND_REQUEST_URL = "/admin/recommend/request";

/** Admin推荐请求服务 */
export class RecommendRequestServiceImpl implements RecommendRequestService {
  /** 查询推荐请求分页列表 */
  PageRecommendRequest(request: PageRecommendRequestRequest): Promise<PageRecommendRequestResponse> {
    return service<PageRecommendRequestRequest, PageRecommendRequestResponse>({
      url: `${RECOMMEND_REQUEST_URL}`,
      method: "get",
      params: request
    });
  }

  /** 查询推荐请求详情 */
  GetRecommendRequest(request: Int64Value): Promise<RecommendRequestDetailResponse> {
    return service<Int64Value, RecommendRequestDetailResponse>({
      url: `${RECOMMEND_REQUEST_URL}/${request.value}`,
      method: "get"
    });
  }

  /** 查询推荐请求商品关联事件 */
  GetRecommendRequestEvent(request: GetRecommendRequestEventRequest): Promise<GetRecommendRequestEventResponse> {
    return service<GetRecommendRequestEventRequest, GetRecommendRequestEventResponse>({
      url: `${RECOMMEND_REQUEST_URL}/${request.requestRecordId}/event`,
      method: "get",
      params: request
    });
  }
}

export const defRecommendRequestService = new RecommendRequestServiceImpl();
