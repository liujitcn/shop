import { http } from '@/utils/http'
import { RecommendGoodsActionType } from '@/rpc/common/enum'
import type {
  RecommendExposureReportRequest,
  RecommendGoodsActionItem,
  RecommendGoodsActionReportRequest,
  RecommendGoodsRequest,
  RecommendGoodsResponse,
  RecommendService,
} from '@/rpc/app/recommend'
import type { Empty } from '@/rpc/google/protobuf/empty'

const RECOMMEND_URL = '/app/recommend'
const RECOMMEND_PAY_TRACK_PREFIX = 'recommend_pay_track_'

/** 推荐服务 */
export class RecommendServiceImpl implements RecommendService {
  /** 查询推荐商品列表 */
  RecommendGoods(request: RecommendGoodsRequest): Promise<RecommendGoodsResponse> {
    return http<RecommendGoodsResponse>({
      url: `${RECOMMEND_URL}/goods`,
      method: 'GET',
      data: request,
    })
  }

  /** 上报推荐曝光事件 */
  RecommendExposureReport(request: RecommendExposureReportRequest): Promise<Empty> {
    return http<Empty>({
      url: `${RECOMMEND_URL}/event/exposure`,
      method: 'POST',
      data: request,
    })
  }

  /** 上报推荐商品行为事件 */
  RecommendGoodsActionReport(request: RecommendGoodsActionReportRequest): Promise<Empty> {
    return http<Empty>({
      url: `${RECOMMEND_URL}/event/goods`,
      method: 'POST',
      data: request,
    })
  }
}

/** 推荐商品行为上下文。 */
export interface RecommendGoodsActionContext {
  goodsId: number
  goodsNum?: number
  source?: string
  scene?: string
  requestId?: string
  index?: number
}

/** 构建推荐商品行为事件项。 */
export const buildRecommendGoodsActionItem = (
  context: RecommendGoodsActionContext,
): RecommendGoodsActionItem => {
  return {
    goodsId: context.goodsId,
    goodsNum: context.goodsNum || 1,
    source: context.source || 'direct',
    scene: context.scene || '',
    requestId: context.requestId || '',
    index: context.index || 0,
  }
}

/** 上报推荐曝光事件。 */
export const reportRecommendExposure = async (
  request: RecommendExposureReportRequest,
): Promise<void> => {
  if (!request.requestId || request.goodsIds.length === 0) {
    return
  }
  await defRecommendService.RecommendExposureReport(request)
}

/** 上报推荐商品行为事件。 */
export const reportRecommendGoodsAction = async (
  eventType: RecommendGoodsActionType,
  goodsItems: RecommendGoodsActionItem[],
): Promise<void> => {
  if (goodsItems.length === 0) {
    return
  }
  await defRecommendService.RecommendGoodsActionReport({
    eventType,
    goodsItems,
  })
}

/** 暂存支付成功页所需的推荐商品行为数据。 */
export const saveRecommendPayTrack = (orderId: number, goodsItems: RecommendGoodsActionItem[]): void => {
  if (orderId <= 0 || goodsItems.length === 0) {
    return
  }
  uni.setStorageSync(`${RECOMMEND_PAY_TRACK_PREFIX}${orderId}`, goodsItems)
}

/** 读取并清理支付成功页所需的推荐商品行为数据。 */
export const takeRecommendPayTrack = (orderId: number): RecommendGoodsActionItem[] => {
  if (orderId <= 0) {
    return []
  }
  const key = `${RECOMMEND_PAY_TRACK_PREFIX}${orderId}`
  const goodsItems = uni.getStorageSync(key) as RecommendGoodsActionItem[] | undefined
  uni.removeStorageSync(key)
  return goodsItems || []
}

export const defRecommendService = new RecommendServiceImpl()
