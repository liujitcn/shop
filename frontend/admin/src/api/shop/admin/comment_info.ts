import service from "@/utils/request";
import {
  type CommentInfoDetail,
  type CommentInfoService,
  type GetCommentInfoRequest,
  type GetGoodsCommentInfoRequest,
  type GoodsCommentInfoResponse,
  type ListCommentReviewRequest,
  type ListCommentReviewResponse,
  type PageCommentDiscussionRequest,
  type PageCommentDiscussionResponse,
  type PageCommentInfoRequest,
  type PageCommentInfoResponse,
  type SetCommentDiscussionStatusRequest,
  type SetCommentInfoStatusRequest
} from "@/rpc/shop/admin/v1/comment_info";
import type { Empty } from "@/rpc/google/protobuf/empty";

const COMMENT_INFO_URL = "/v1/admin/comment/info";

/** Admin评论管理服务。 */
export class CommentInfoServiceImpl implements CommentInfoService {
  /** 查询评论分页列表。 */
  PageCommentInfo(request: PageCommentInfoRequest): Promise<PageCommentInfoResponse> {
    return service<PageCommentInfoRequest, PageCommentInfoResponse>({
      url: `${COMMENT_INFO_URL}`,
      method: "get",
      params: request
    });
  }

  /** 按商品查询评论聚合信息。 */
  GetGoodsCommentInfo(request: GetGoodsCommentInfoRequest): Promise<GoodsCommentInfoResponse> {
    return service<GetGoodsCommentInfoRequest, GoodsCommentInfoResponse>({
      url: `${COMMENT_INFO_URL}/goods/${request.goods_id}`,
      method: "get"
    });
  }

  /** 查询评论详情。 */
  GetCommentInfo(request: GetCommentInfoRequest): Promise<CommentInfoDetail> {
    return service<GetCommentInfoRequest, CommentInfoDetail>({
      url: `${COMMENT_INFO_URL}/${request.id}`,
      method: "get"
    });
  }

  /** 查询评论审核记录列表。 */
  ListCommentReview(request: ListCommentReviewRequest): Promise<ListCommentReviewResponse> {
    return service<ListCommentReviewRequest, ListCommentReviewResponse>({
      url: `${COMMENT_INFO_URL}/review`,
      method: "get",
      params: request
    });
  }

  /** 设置评论审核状态。 */
  SetCommentInfoStatus(request: SetCommentInfoStatusRequest): Promise<Empty> {
    return service<SetCommentInfoStatusRequest, Empty>({
      url: `${COMMENT_INFO_URL}/${request.id}/status`,
      method: "put",
      data: { ...request, reason: request.reason ?? "" }
    });
  }

  /** 查询评论讨论分页列表。 */
  PageCommentDiscussion(request: PageCommentDiscussionRequest): Promise<PageCommentDiscussionResponse> {
    return service<PageCommentDiscussionRequest, PageCommentDiscussionResponse>({
      url: `${COMMENT_INFO_URL}/${request.comment_id}/discussion`,
      method: "get",
      params: request
    });
  }

  /** 设置评论讨论审核状态。 */
  SetCommentDiscussionStatus(request: SetCommentDiscussionStatusRequest): Promise<Empty> {
    return service<SetCommentDiscussionStatusRequest, Empty>({
      url: `${COMMENT_INFO_URL}/discussion/${request.id}/status`,
      method: "put",
      data: { ...request, reason: request.reason ?? "" }
    });
  }
}

export const defCommentInfoService = new CommentInfoServiceImpl();
