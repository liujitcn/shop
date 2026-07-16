import { http } from '@/utils/http'
import type {
  CommentService,
  CreateCommentDiscussionRequest,
  CreateCommentDiscussionResponse,
  CreateCommentRequest,
  CreateCommentResponse,
  DeleteCommentRequest,
  GoodsCommentOverviewRequest,
  GoodsCommentOverviewResponse,
  GoodsCommentTagsRequest,
  GoodsCommentTagsResponse,
  PageCommentDiscussionRequest,
  PageCommentDiscussionResponse,
  PageGoodsCommentRequest,
  PageGoodsCommentResponse,
  PageMyCommentRequest,
  PageMyCommentResponse,
  PagePendingCommentGoodsRequest,
  PagePendingCommentGoodsResponse,
  SaveCommentReactionRequest,
  SaveCommentReactionResponse,
} from '@/rpc/app/v1/comment'
import type { Empty } from '@/rpc/google/protobuf/empty'

const COMMENT_URL = '/v1/app/comment'
const COMMENT_GOODS_URL = `${COMMENT_URL}/goods`

/** 评价服务 */
export class CommentServiceImpl implements CommentService {
  /** 查询商品评价摘要 */
  async GoodsCommentOverview(
    request: GoodsCommentOverviewRequest,
  ): Promise<GoodsCommentOverviewResponse> {
    const response = await http<Partial<GoodsCommentOverviewResponse>>({
      url: `${COMMENT_GOODS_URL}/${request.goods_id}/overview`,
      method: 'GET',
      authMode: 'optional',
      data: request,
    })
    return {
      ...response,
      comment_summary: response.comment_summary,
      preview_comments: response.preview_comments ?? [],
      total_count: response.total_count ?? 0,
      recent_days: response.recent_days ?? 0,
      recent_good_rate: response.recent_good_rate ?? 0,
    }
  }

  /** 查询商品评价标签列表 */
  async GoodsCommentTags(request: GoodsCommentTagsRequest): Promise<GoodsCommentTagsResponse> {
    const response = await http<Partial<GoodsCommentTagsResponse>>({
      url: `${COMMENT_GOODS_URL}/${request.goods_id}/tags`,
      method: 'GET',
      authMode: 'optional',
      data: request,
    })
    return {
      ...response,
      comment_tags: response.comment_tags ?? [],
    }
  }

  /** 查询商品评价分页列表 */
  async PageGoodsComment(request: PageGoodsCommentRequest): Promise<PageGoodsCommentResponse> {
    const response = await http<Partial<PageGoodsCommentResponse>>({
      url: `${COMMENT_GOODS_URL}/${request.goods_id}`,
      method: 'GET',
      authMode: 'optional',
      data: request,
    })
    // 评论筛选项按当前接口的 snake_case 字段读取。
    const commentFilters = response.comment_filters ?? []
    return {
      ...response,
      comment_filters: commentFilters,
      comments: response.comments ?? [],
      comment_summary: response.comment_summary,
      total: response.total ?? 0,
      page_num: response.page_num ?? 0,
      page_size: response.page_size ?? 0,
      has_more: response.has_more ?? false,
    }
  }

  /** 查询评价讨论分页列表 */
  async PageCommentDiscussion(
    request: PageCommentDiscussionRequest,
  ): Promise<PageCommentDiscussionResponse> {
    const response = await http<Partial<PageCommentDiscussionResponse>>({
      url: `${COMMENT_URL}/${request.comment_id}/discussion`,
      method: 'GET',
      authMode: 'optional',
      data: request,
    })
    return {
      ...response,
      comment_discussions: response.comment_discussions ?? [],
      comment_id: response.comment_id ?? request.comment_id,
      total: response.total ?? 0,
      page_num: response.page_num ?? 0,
      page_size: response.page_size ?? 0,
      has_more: response.has_more ?? false,
    }
  }

  /** 发布评价讨论 */
  CreateCommentDiscussion(
    request: CreateCommentDiscussionRequest,
  ): Promise<CreateCommentDiscussionResponse> {
    return http<CreateCommentDiscussionResponse>({
      url: `${COMMENT_URL}/${request.comment_id}/discussion`,
      method: 'POST',
      authMode: 'required',
      data: request,
    })
  }

  /** 保存评价互动状态 */
  SaveCommentReaction(request: SaveCommentReactionRequest): Promise<SaveCommentReactionResponse> {
    return http<SaveCommentReactionResponse>({
      url: `${COMMENT_URL}/reaction`,
      method: 'PUT',
      authMode: 'required',
      data: request,
    })
  }

  /** 查询待评价商品分页列表 */
  async PagePendingCommentGoods(
    request: PagePendingCommentGoodsRequest,
  ): Promise<PagePendingCommentGoodsResponse> {
    const response = await http<Partial<PagePendingCommentGoodsResponse>>({
      url: `${COMMENT_URL}/pending`,
      method: 'GET',
      authMode: 'required',
      data: request,
    })
    return {
      ...response,
      pending_comment_goods: response.pending_comment_goods ?? [],
      total: response.total ?? 0,
      page_num: response.page_num ?? 0,
      page_size: response.page_size ?? 0,
      has_more: response.has_more ?? false,
    }
  }

  /** 发布商品评价 */
  CreateComment(request: CreateCommentRequest): Promise<CreateCommentResponse> {
    return http<CreateCommentResponse>({
      url: `${COMMENT_URL}`,
      method: 'POST',
      authMode: 'required',
      data: request,
    })
  }

  /** 删除商品评价 */
  DeleteComment(request: DeleteCommentRequest): Promise<Empty> {
    return http<Empty>({
      url: `${COMMENT_URL}/${request.id}`,
      method: 'DELETE',
      authMode: 'required',
    })
  }

  /** 查询我的评价分页列表 */
  async PageMyComment(request: PageMyCommentRequest): Promise<PageMyCommentResponse> {
    const response = await http<Partial<PageMyCommentResponse>>({
      url: `${COMMENT_URL}/my`,
      method: 'GET',
      authMode: 'required',
      data: request,
    })
    return {
      ...response,
      comments: response.comments ?? [],
      total: response.total ?? 0,
      page_num: response.page_num ?? 0,
      page_size: response.page_size ?? 0,
      has_more: response.has_more ?? false,
    }
  }
}

export const defCommentService = new CommentServiceImpl()
