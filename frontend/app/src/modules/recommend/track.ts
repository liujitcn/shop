import { defRecommendService } from '@/api/app/recommend'
import type {
  RecommendExposureReportRequest,
  RecommendGoodsActionItem,
  RecommendGoodsRequest,
  RecommendGoodsResponse,
} from '@/rpc/app/recommend'
import { RecommendGoodsActionType, RecommendScene } from '@/rpc/common/enum'
import { useRecommendStore } from '@/stores'
import {
  RECOMMEND_CART_TRACK_PREFIX,
  RECOMMEND_PAY_TRACK_PREFIX,
} from './constants'
import type { RecommendGoodsActionContext } from './context'

/** 拉取推荐商品前先确保匿名主体已初始化。 */
export const fetchRecommendGoods = async (
  request: RecommendGoodsRequest,
): Promise<RecommendGoodsResponse> => {
  await useRecommendStore().getAnonymousId()
  return defRecommendService.RecommendGoods(request)
}

/** 只有携带 requestId 且有商品数据时才上报曝光。 */
export const reportRecommendExposure = async (
  request: RecommendExposureReportRequest,
): Promise<void> => {
  if (!request.requestId || request.goodsIds.length === 0) {
    return
  }
  await useRecommendStore().getAnonymousId()
  await defRecommendService.RecommendExposureReport(request)
}

/** 商品行为上报和推荐查询共用同一个匿名主体。 */
export const reportRecommendGoodsAction = async (
  eventType: RecommendGoodsActionType,
  goodsItems: RecommendGoodsActionItem[],
): Promise<void> => {
  if (goodsItems.length === 0) {
    return
  }
  await useRecommendStore().getAnonymousId()
  await defRecommendService.RecommendGoodsActionReport(
    {
      eventType,
      goodsItems,
    },
  )
}

/** 支付成功页需要延迟消费这批商品行为数据。 */
export const saveRecommendPayTrack = (
  orderId: number,
  goodsItems: RecommendGoodsActionItem[],
): void => {
  if (orderId <= 0 || goodsItems.length === 0) {
    return
  }
  uni.setStorageSync(`${RECOMMEND_PAY_TRACK_PREFIX}${orderId}`, goodsItems)
}

/** 购物车场景按商品和 sku 维度缓存推荐上下文。 */
export const saveRecommendCartTrack = (context: RecommendGoodsActionContext): void => {
  if (!context.goodsId || !context.skuCode || !context.requestId) {
    return
  }
  uni.setStorageSync(`${RECOMMEND_CART_TRACK_PREFIX}${context.goodsId}_${context.skuCode}`, {
    goodsId: context.goodsId,
    skuCode: context.skuCode,
    goodsNum: context.goodsNum || 1,
    scene: context.scene ?? RecommendScene.RECOMMEND_SCENE_UNKNOWN,
    requestId: context.requestId,
    position: context.position || 0,
  } as RecommendGoodsActionContext)
}

/** 读取购物车里暂存的推荐上下文。 */
export const getRecommendCartTrack = (
  goodsId: number,
  skuCode: string,
): RecommendGoodsActionContext | undefined => {
  if (!goodsId || !skuCode) {
    return undefined
  }
  return uni.getStorageSync(`${RECOMMEND_CART_TRACK_PREFIX}${goodsId}_${skuCode}`) as
    | RecommendGoodsActionContext
    | undefined
}

/** 支付完成后消费一次埋点缓存，避免重复上报。 */
export const takeRecommendPayTrack = (orderId: number): RecommendGoodsActionItem[] => {
  if (orderId <= 0) {
    return []
  }
  const key = `${RECOMMEND_PAY_TRACK_PREFIX}${orderId}`
  const goodsItems = uni.getStorageSync(key) as RecommendGoodsActionItem[] | undefined
  uni.removeStorageSync(key)
  return goodsItems || []
}
