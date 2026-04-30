import service from "@/utils/request";
import {
  type CommentAi,
  type CommentDiscussion,
  type CommentInfo,
  type CommentInfoDetail,
  type CommentInfoService,
  type CommentReview,
  type CommentTag,
  type GetCommentInfoRequest,
  type GetGoodsCommentInfoRequest,
  type GoodsCommentInfoResponse,
  type ListCommentReviewsRequest,
  type ListCommentReviewsResponse,
  type PageCommentDiscussionsRequest,
  type PageCommentDiscussionsResponse,
  type PageCommentInfosRequest,
  type PageCommentInfosResponse,
  type SetCommentDiscussionStatusRequest,
  type SetCommentInfoStatusRequest
} from "@/rpc/admin/v1/comment_info";
import type { Empty } from "@/rpc/google/protobuf/empty";

const COMMENT_INFO_URL = "/v1/admin/comment/info";

type PageCommentInfosHTTPResponse = Partial<PageCommentInfosResponse> & {
  /** 旧协议评论分页数据字段 */
  list?: CommentInfo[];
};

type CommentInfoDetailHTTPResponse = Partial<CommentInfoDetail> & {
  /** 旧协议商品评论标签字段 */
  tagList?: CommentTag[];
  /** 旧协议评论讨论列表字段 */
  discussionList?: CommentDiscussion[];
  /** 旧协议商品评论 AI 摘要字段 */
  aiList?: CommentAi[];
  /** 旧协议评论审核记录字段 */
  reviewList?: CommentReview[];
};

type GoodsCommentInfoHTTPResponse = Partial<GoodsCommentInfoResponse> & {
  /** 旧协议评论列表字段 */
  commentList?: CommentInfo[];
  /** 旧协议商品评论标签字段 */
  tagList?: CommentTag[];
  /** 旧协议评论讨论列表字段 */
  discussionList?: CommentDiscussion[];
  /** 旧协议商品评论 AI 摘要字段 */
  aiList?: CommentAi[];
};

type PageCommentDiscussionsHTTPResponse = Partial<PageCommentDiscussionsResponse> & {
  /** 旧协议讨论分页数据字段 */
  list?: CommentDiscussion[];
};

type ListCommentReviewsHTTPResponse = Partial<ListCommentReviewsResponse> & {
  /** 旧协议审核记录字段 */
  list?: CommentReview[];
};

/** Admin评论管理服务。 */
export class CommentInfoServiceImpl implements CommentInfoService {
  /** 查询评论分页列表。 */
  async PageCommentInfos(request: PageCommentInfosRequest): Promise<PageCommentInfosResponse> {
    const response = await service<PageCommentInfosRequest, PageCommentInfosHTTPResponse>({
      url: `${COMMENT_INFO_URL}`,
      method: "get",
      params: request
    });
    // 兼容未统一生成前的旧响应 list，同时向新协议 comment_infos 收敛。
    const comment_infos = response.comment_infos ?? response.list ?? [];
    return { ...response, comment_infos, total: response.total ?? 0 } as PageCommentInfosResponse;
  }

  /** 按商品查询评论聚合信息。 */
  async GetGoodsCommentInfo(request: GetGoodsCommentInfoRequest): Promise<GoodsCommentInfoResponse> {
    const response = await service<GetGoodsCommentInfoRequest, GoodsCommentInfoHTTPResponse>({
      url: `${COMMENT_INFO_URL}/goods/${request.goods_id}`,
      method: "get"
    });
    // 兼容旧聚合字段，同时向新协议复数资源字段收敛。
    const comment_infos = response.comment_infos ?? response.commentList ?? [];
    const comment_tags = response.comment_tags ?? response.tagList ?? [];
    const comment_discussions = response.comment_discussions ?? response.discussionList ?? [];
    const comment_ais = response.comment_ais ?? response.aiList ?? [];
    return { ...response, comment_infos, comment_tags, comment_discussions, comment_ais } as GoodsCommentInfoResponse;
  }

  /** 查询评论详情。 */
  async GetCommentInfo(request: GetCommentInfoRequest): Promise<CommentInfoDetail> {
    const response = await service<GetCommentInfoRequest, CommentInfoDetailHTTPResponse>({
      url: `${COMMENT_INFO_URL}/${request.id}`,
      method: "get"
    });
    // 兼容旧详情字段，同时向新协议复数资源字段收敛。
    const comment_tags = response.comment_tags ?? response.tagList ?? [];
    const comment_discussions = response.comment_discussions ?? response.discussionList ?? [];
    const comment_ais = response.comment_ais ?? response.aiList ?? [];
    const comment_reviews = response.comment_reviews ?? response.reviewList ?? [];
    return {
      ...response,
      comment: response.comment,
      comment_tags,
      comment_discussions,
      comment_ais,
      comment_reviews
    } as CommentInfoDetail;
  }

  /** 查询评论审核记录列表。 */
  async ListCommentReviews(request: ListCommentReviewsRequest): Promise<ListCommentReviewsResponse> {
    const response = await service<ListCommentReviewsRequest, ListCommentReviewsHTTPResponse>({
      url: `${COMMENT_INFO_URL}/review`,
      method: "get",
      params: request
    });
    const comment_reviews = response.comment_reviews ?? response.list ?? [];
    return { ...response, comment_reviews } as ListCommentReviewsResponse;
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
  async PageCommentDiscussions(request: PageCommentDiscussionsRequest): Promise<PageCommentDiscussionsResponse> {
    const response = await service<PageCommentDiscussionsRequest, PageCommentDiscussionsHTTPResponse>({
      url: `${COMMENT_INFO_URL}/${request.comment_id}/discussion`,
      method: "get",
      params: request
    });
    // 兼容未统一生成前的旧响应 list，同时向新协议 comment_discussions 收敛。
    const comment_discussions = response.comment_discussions ?? response.list ?? [];
    return { ...response, comment_discussions, total: response.total ?? 0 } as PageCommentDiscussionsResponse;
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
