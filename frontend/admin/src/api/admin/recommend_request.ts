import service from "@/utils/request";
import type {
  GetRecommendRequestRequest,
  ListRecommendRequestEventsRequest,
  ListRecommendRequestEventsResponse,
  PageRecommendRequestsRequest,
  PageRecommendRequestsResponse,
  RecommendRequestDetailResponse,
  RecommendRequestService
} from "@/rpc/admin/v1/recommend_request";

const RECOMMEND_REQUEST_URL = "/v1/admin/recommend/request";

/** Admin推荐请求服务 */
export class RecommendRequestServiceImpl implements RecommendRequestService {
  /** 查询推荐请求分页列表 */
  PageRecommendRequests(request: PageRecommendRequestsRequest): Promise<PageRecommendRequestsResponse> {
    return service<PageRecommendRequestsRequest, PageRecommendRequestsResponse>({
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
  ListRecommendRequestEvents(request: ListRecommendRequestEventsRequest): Promise<ListRecommendRequestEventsResponse> {
    return service<ListRecommendRequestEventsRequest, ListRecommendRequestEventsResponse>({
      url: `${RECOMMEND_REQUEST_URL}/${request.request_record_id}/event`,
      method: "get",
      params: request
    });
  }
}

export const defRecommendRequestService = new RecommendRequestServiceImpl();
