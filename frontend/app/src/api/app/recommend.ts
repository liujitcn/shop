import { http } from '@/utils/http'
import { RecommendGoodsActionType, RecommendScene, RecommendSource } from '@/rpc/common/enum'
import type {
  RecommendContext,
  RecommendExposureReportRequest,
  RecommendGoodsActionItem,
  RecommendGoodsActionReportRequest,
  RecommendGoodsRequest,
  RecommendGoodsResponse,
  RecommendService,
} from '@/rpc/app/recommend'
import type { Empty } from '@/rpc/google/protobuf/empty'
import type { Int64Value } from '@/rpc/google/protobuf/wrappers'
import { useUserStore } from '@/stores'

const RECOMMEND_URL = '/app/recommend'
const RECOMMEND_PAY_TRACK_PREFIX = 'recommend_pay_track_'
const RECOMMEND_CART_TRACK_PREFIX = 'recommend_cart_track_'
const RECOMMEND_ANONYMOUS_ACTOR_KEY = 'recommend_anonymous_actor'
const RECOMMEND_ANONYMOUS_ID_HEADER = 'X-Recommend-Anonymous-Id'

/** 推荐服务 */
export class RecommendServiceImpl implements RecommendService {
  /** 获取匿名推荐主体 */
  async RecommendAnonymousActor(_: Empty): Promise<Int64Value> {
    return http<Int64Value>({
      url: `${RECOMMEND_URL}/actor/anonymous`,
      method: 'GET',
      header: {
        Authorization: 'no-auth',
      },
    })
  }

  /** 查询推荐商品列表 */
  async RecommendGoods(request: RecommendGoodsRequest): Promise<RecommendGoodsResponse> {
    const anonymousId = await resolveRecommendAnonymousId()
    return http<RecommendGoodsResponse>({
      url: `${RECOMMEND_URL}/goods`,
      method: 'GET',
      header: buildRecommendHeader(anonymousId),
      data: request,
    })
  }

  /** 上报推荐曝光事件 */
  async RecommendExposureReport(request: RecommendExposureReportRequest): Promise<Empty> {
    const anonymousId = await resolveRecommendAnonymousId()
    return http<Empty>({
      url: `${RECOMMEND_URL}/event/exposure`,
      method: 'POST',
      header: buildRecommendHeader(anonymousId),
      data: request,
    })
  }

  /** 上报推荐商品行为事件 */
  async RecommendGoodsActionReport(request: RecommendGoodsActionReportRequest): Promise<Empty> {
    const anonymousId = await resolveRecommendAnonymousId()
    return http<Empty>({
      url: `${RECOMMEND_URL}/event/goods`,
      method: 'POST',
      header: buildRecommendHeader(anonymousId),
      data: request,
    })
  }
}

/** 构建匿名 ID 请求头。 */
const buildRecommendHeader = (anonymousId: number): Record<string, string> => {
  if (!anonymousId) {
    return {}
  }
  return {
    [RECOMMEND_ANONYMOUS_ID_HEADER]: String(anonymousId),
  }
}

/** 解析当前匿名推荐 ID。 */
const resolveRecommendAnonymousId = async (): Promise<number> => {
  const userStore = useUserStore()
  if (userStore.userInfo) {
    return 0
  }

  const cachedActor = uni.getStorageSync(RECOMMEND_ANONYMOUS_ACTOR_KEY) as
    | Int64Value
    | undefined
  if (cachedActor?.value) {
    return cachedActor.value
  }

  const actor = await defRecommendService.RecommendAnonymousActor({})
  uni.setStorageSync(RECOMMEND_ANONYMOUS_ACTOR_KEY, actor)
  return actor.value || 0
}

/** 推荐商品行为上下文。 */
export interface RecommendGoodsActionContext {
  goodsId: number
  skuCode?: string
  goodsNum?: number
  source?: string | number
  scene?: string | number
  requestId?: string
  index?: number
}

/** 规范化推荐来源值。 */
export const normalizeRecommendSource = (source?: string | number): RecommendSource => {
  if (source === undefined || source === null || source === '') {
    return RecommendSource.DIRECT
  }
  if (typeof source === 'number') {
    return source === RecommendSource.RECOMMEND ? RecommendSource.RECOMMEND : RecommendSource.DIRECT
  }
  const value = String(source).trim()
  if (!value) {
    return RecommendSource.DIRECT
  }
  if (/^\d+$/.test(value)) {
    return Number(value) === RecommendSource.RECOMMEND ? RecommendSource.RECOMMEND : RecommendSource.DIRECT
  }
  return value.toLowerCase() === 'recommend' ? RecommendSource.RECOMMEND : RecommendSource.DIRECT
}

/** 将推荐来源格式化为路由字符串。 */
export const formatRecommendSource = (source?: string | number): string => {
  return normalizeRecommendSource(source) === RecommendSource.RECOMMEND ? 'recommend' : 'direct'
}

/** 构建推荐上下文。 */
export const buildRecommendContext = (
  context: Omit<RecommendGoodsActionContext, 'goodsId' | 'skuCode' | 'goodsNum'>,
): RecommendContext => {
  return {
    source: normalizeRecommendSource(context.source),
    scene: normalizeRecommendScene(context.scene),
    requestId: context.requestId || '',
    position: context.index || 0,
  }
}

/** 规范化推荐场景值。 */
export const normalizeRecommendScene = (scene?: string | number): RecommendScene => {
  if (scene === undefined || scene === null || scene === '') {
    return RecommendScene.RECOMMEND_SCENE_UNKNOWN
  }
  if (typeof scene === 'number') {
    return RecommendScene[scene] ? scene : RecommendScene.RECOMMEND_SCENE_UNKNOWN
  }
  const value = String(scene).trim()
  if (!value) {
    return RecommendScene.RECOMMEND_SCENE_UNKNOWN
  }
  if (/^\d+$/.test(value)) {
    const sceneValue = Number(value)
    return RecommendScene[sceneValue] ? sceneValue : RecommendScene.RECOMMEND_SCENE_UNKNOWN
  }
  return (RecommendScene as unknown as Record<string, RecommendScene | undefined>)[value] || RecommendScene.RECOMMEND_SCENE_UNKNOWN
}

export const formatRecommendScene = (scene?: string | number): string => {
  const sceneValue = normalizeRecommendScene(scene)
  return sceneValue === RecommendScene.RECOMMEND_SCENE_UNKNOWN ? '' : RecommendScene[sceneValue]
}

/** 构建推荐商品行为事件项。 */
export const buildRecommendGoodsActionItem = (
  context: RecommendGoodsActionContext,
): RecommendGoodsActionItem => {
  return {
    goodsId: context.goodsId,
    goodsNum: context.goodsNum || 1,
    recommendContext: buildRecommendContext(context),
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

/** 暂存购物车商品的推荐上下文。 */
export const saveRecommendCartTrack = (context: RecommendGoodsActionContext): void => {
  if (
    !context.goodsId ||
    !context.skuCode ||
    normalizeRecommendSource(context.source) !== RecommendSource.RECOMMEND ||
    !context.requestId
  ) {
    return
  }
  uni.setStorageSync(`${RECOMMEND_CART_TRACK_PREFIX}${context.goodsId}_${context.skuCode}`, {
    goodsId: context.goodsId,
    skuCode: context.skuCode,
    goodsNum: context.goodsNum || 1,
    source: normalizeRecommendSource(context.source),
    scene: normalizeRecommendScene(context.scene),
    requestId: context.requestId,
    index: context.index || 0,
  } as RecommendGoodsActionContext)
}

/** 读取购物车商品的推荐上下文。 */
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
