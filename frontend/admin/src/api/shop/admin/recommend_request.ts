import service from "@/utils/request";
import type {
  GetRecommendRequestRequest,
  ListRecommendRequestEventRequest,
  ListRecommendRequestEventResponse,
  PageRecommendRequestRequest,
  PageRecommendRequestResponse,
  RecommendRequestDetailResponse,
  RecommendRequestService
} from "@/rpc/shop/admin/v1/recommend_request";

const RECOMMEND_REQUEST_URL = "/v1/admin/recommend/request";

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
  GetRecommendRequest(request: GetRecommendRequestRequest): Promise<RecommendRequestDetailResponse> {
    return service<GetRecommendRequestRequest, RecommendRequestDetailResponse>({
      url: `${RECOMMEND_REQUEST_URL}/${request.id}`,
      method: "get"
    });
  }

  /** 查询推荐请求商品关联事件 */
  ListRecommendRequestEvent(request: ListRecommendRequestEventRequest): Promise<ListRecommendRequestEventResponse> {
    return service<ListRecommendRequestEventRequest, ListRecommendRequestEventResponse>({
      url: `${RECOMMEND_REQUEST_URL}/${request.request_record_id}/event`,
      method: "get",
      params: request
    });
  }
}

export const defRecommendRequestService = new RecommendRequestServiceImpl();
