import { http } from '@/utils/http'
import type {
  CommentService,
  CommentDiscussionItem,
  CommentFilterItem,
  CommentItem,
  CommentTagItem,
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
  PendingCommentGoodsItem,
  SaveCommentReactionRequest,
  SaveCommentReactionResponse,
} from '@/rpc/app/v1/comment'
import type { Empty } from '@/rpc/google/protobuf/empty'

const COMMENT_URL = '/v1/app/comment'
const COMMENT_GOODS_URL = `${COMMENT_URL}/goods`

/** 商品评价摘要响应兼容结构，保留旧版 previewList 字段。 */
type GoodsCommentOverviewResponseCompat = GoodsCommentOverviewResponse & {
  previewList: CommentItem[]
}

/** 商品评价摘要 HTTP 原始响应，兼容旧版预览列表字段。 */
type GoodsCommentOverviewHTTPResponse = Partial<GoodsCommentOverviewResponse> & {
  previewList?: CommentItem[]
}

/** 商品评价标签响应兼容结构，保留旧版 tagList 字段。 */
type GoodsCommentTagsResponseCompat = GoodsCommentTagsResponse & {
  tagList: CommentTagItem[]
}

/** 商品评价标签 HTTP 原始响应，兼容旧版标签列表字段。 */
type GoodsCommentTagsHTTPResponse = Partial<GoodsCommentTagsResponse> & {
  tagList?: CommentTagItem[]
}

/** 商品评价分页响应兼容结构，保留旧版 filterList 和 list 字段。 */
type PageGoodsCommentResponseCompat = PageGoodsCommentResponse & {
  filterList: CommentFilterItem[]
  list: CommentItem[]
}

/** 商品评价分页 HTTP 原始响应，兼容旧版筛选和列表字段。 */
type PageGoodsCommentHTTPResponse = Partial<PageGoodsCommentResponse> & {
  filterList?: CommentFilterItem[]
  list?: CommentItem[]
}

/** 评价讨论分页响应兼容结构，保留旧版 list 字段。 */
type PageCommentDiscussionResponseCompat = PageCommentDiscussionResponse & {
  list: CommentDiscussionItem[]
}

/** 评价讨论分页 HTTP 原始响应，兼容旧版 list 字段。 */
type PageCommentDiscussionHTTPResponse = Partial<PageCommentDiscussionResponse> & {
  list?: CommentDiscussionItem[]
}

/** 待评价商品分页响应兼容结构，保留旧版 list 字段。 */
type PagePendingCommentGoodsResponseCompat = PagePendingCommentGoodsResponse & {
  list: PendingCommentGoodsItem[]
}

/** 待评价商品分页 HTTP 原始响应，兼容旧版 list 字段。 */
type PagePendingCommentGoodsHTTPResponse = Partial<PagePendingCommentGoodsResponse> & {
  list?: PendingCommentGoodsItem[]
}

/** 我的评价分页响应兼容结构，保留旧版 list 字段。 */
type PageMyCommentResponseCompat = PageMyCommentResponse & {
  list: CommentItem[]
}

/** 我的评价分页 HTTP 原始响应，兼容旧版 list 字段。 */
type PageMyCommentHTTPResponse = Partial<PageMyCommentResponse> & {
  list?: CommentItem[]
}

/** 评价服务 */
export class CommentServiceImpl implements CommentService {
  /** 查询商品评价摘要 */
  async GoodsCommentOverview(
    request: GoodsCommentOverviewRequest,
  ): Promise<GoodsCommentOverviewResponseCompat> {
    const response = await http<GoodsCommentOverviewHTTPResponse>({
      url: `${COMMENT_GOODS_URL}/${request.goods_id}/overview`,
      method: 'GET',
      authMode: 'optional',
      data: request,
    })
    // 兼容未生成前的旧字段，同时向新协议的 previewComments 收敛。
    const previewComments = response.preview_comments ?? response.previewList ?? []
    return {
      ...response,
      comment_summary: response.comment_summary,
      preview_comments: previewComments,
      previewList: previewComments,
      total_count: response.total_count ?? 0,
      recent_days: response.recent_days ?? 0,
      recent_good_rate: response.recent_good_rate ?? 0,
    }
  }

  /** 查询商品评价标签列表 */
  async GoodsCommentTags(
    request: GoodsCommentTagsRequest,
  ): Promise<GoodsCommentTagsResponseCompat> {
    const response = await http<GoodsCommentTagsHTTPResponse>({
      url: `${COMMENT_GOODS_URL}/${request.goods_id}/tags`,
      method: 'GET',
      authMode: 'optional',
      data: request,
    })
    const commentTags = response.comment_tags ?? response.tagList ?? []
    return {
      ...response,
      comment_tags: commentTags,
      tagList: commentTags,
    }
  }

  /** 查询商品评价分页列表 */
  async PageGoodsComment(
    request: PageGoodsCommentRequest,
  ): Promise<PageGoodsCommentResponseCompat> {
    const response = await http<PageGoodsCommentHTTPResponse>({
      url: `${COMMENT_GOODS_URL}/${request.goods_id}`,
      method: 'GET',
      authMode: 'optional',
      data: request,
    })
    // 兼容未生成前的旧字段，同时向新协议的 commentFilters/comments 收敛。
    const commentFilters = response.comment_filters ?? response.filterList ?? []
    const comments = response.comments ?? response.list ?? []
    return {
      ...response,
      comment_filters: commentFilters,
      comments,
      comment_summary: response.comment_summary,
      filterList: commentFilters,
      list: comments,
      total: response.total ?? 0,
      page_num: response.page_num ?? 0,
      page_size: response.page_size ?? 0,
      has_more: response.has_more ?? false,
    }
  }

  /** 查询评价讨论分页列表 */
  async PageCommentDiscussion(
    request: PageCommentDiscussionRequest,
  ): Promise<PageCommentDiscussionResponseCompat> {
    const response = await http<PageCommentDiscussionHTTPResponse>({
      url: `${COMMENT_URL}/${request.comment_id}/discussion`,
      method: 'GET',
      authMode: 'optional',
      data: request,
    })
    // 兼容未生成前的旧响应 list，同时向新协议的 commentDiscussions 字段收敛。
    const commentDiscussions = response.comment_discussions ?? response.list ?? []
    return {
      ...response,
      comment_discussions: commentDiscussions,
      list: commentDiscussions,
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
  ): Promise<PagePendingCommentGoodsResponseCompat> {
    const response = await http<PagePendingCommentGoodsHTTPResponse>({
      url: `${COMMENT_URL}/pending`,
      method: 'GET',
      authMode: 'required',
      data: request,
    })
    // 兼容未生成前的旧响应 list，同时向新协议的 pendingCommentGoods 字段收敛。
    const pendingCommentGoods = response.pending_comment_goods ?? response.list ?? []
    return {
      ...response,
      pending_comment_goods: pendingCommentGoods,
      list: pendingCommentGoods,
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
  async PageMyComment(request: PageMyCommentRequest): Promise<PageMyCommentResponseCompat> {
    const response = await http<PageMyCommentHTTPResponse>({
      url: `${COMMENT_URL}/my`,
      method: 'GET',
      authMode: 'required',
      data: request,
    })
    // 兼容未生成前的旧响应 list，同时向新协议的 comments 字段收敛。
    const comments = response.comments ?? response.list ?? []
    return {
      ...response,
      comments,
      list: comments,
      total: response.total ?? 0,
      page_num: response.page_num ?? 0,
      page_size: response.page_size ?? 0,
      has_more: response.has_more ?? false,
    }
  }
}

export const defCommentService = new CommentServiceImpl()
