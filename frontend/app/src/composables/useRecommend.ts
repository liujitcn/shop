import {
  buildRecommendContext,
  buildRecommendGoodsActionItem,
  parseRecommendScene,
  parseRecommendSource,
  reportRecommendGoodsAction,
  saveRecommendCartTrack,
  stringifyRecommendScene,
  stringifyRecommendSource,
} from '@/api/app/recommend'
import type { RecommendContext, RecommendGoodsActionItem } from '@/rpc/app/recommend'
import type { RecommendGoodsActionType, RecommendScene, RecommendSource } from '@/rpc/common/enum'

/** 推荐路由上下文参数。 */
export interface RecommendRouteQuery {
  /** 推荐来源。 */
  source?: RecommendSource | string
  /** 推荐场景。 */
  scene?: RecommendScene | string
  /** 推荐请求 ID。 */
  requestId?: string
  /** 推荐位序号。 */
  index?: string | number
}

/** 解析推荐位序号。 */
const resolveRecommendIndex = (index?: string | number): number => {
  if (index === undefined || index === null || index === '') {
    return 0
  }
  const position = Number(index)
  if (Number.isNaN(position) || position < 0) {
    return 0
  }
  return position
}

/** 解析路由中的推荐上下文。 */
export const resolveRecommendRouteContext = (
  query: RecommendRouteQuery,
): RecommendContext => {
  return {
    source:
      typeof query.source === 'string'
        ? parseRecommendSource(query.source)
        : (query.source ?? parseRecommendSource('')),
    scene:
      typeof query.scene === 'string'
        ? parseRecommendScene(query.scene)
        : (query.scene ?? parseRecommendScene('')),
    requestId: query.requestId || '',
    position: resolveRecommendIndex(query.index),
  }
}

/** 根据路由参数构建推荐上下文。 */
export const buildRecommendContextByRoute = (query: RecommendRouteQuery): RecommendContext => {
  return buildRecommendContext(resolveRecommendRouteContext(query))
}

/** 根据路由参数构建推荐行为事件项。 */
export const buildRecommendGoodsActionItemByRoute = (
  query: RecommendRouteQuery,
  goodsId: number,
  goodsNum = 1,
): RecommendGoodsActionItem => {
  return buildRecommendGoodsActionItem({
    ...resolveRecommendRouteContext(query),
    goodsId,
    goodsNum,
  })
}

/** 按当前路由上下文上报商品行为事件。 */
export const reportRecommendGoodsActionByRoute = async (
  eventType: RecommendGoodsActionType,
  query: RecommendRouteQuery,
  goodsId: number,
  goodsNum = 1,
): Promise<void> => {
  await reportRecommendGoodsAction(eventType, [
    buildRecommendGoodsActionItemByRoute(query, goodsId, goodsNum),
  ])
}

/** 按当前路由上下文暂存购物车推荐归因。 */
export const saveRecommendCartTrackByRoute = (
  query: RecommendRouteQuery,
  goodsId: number,
  skuCode: string,
  goodsNum = 1,
): void => {
  saveRecommendCartTrack({
    ...resolveRecommendRouteContext(query),
    goodsId,
    skuCode,
    goodsNum,
  })
}

/** 构建推荐路由查询串。 */
export const buildRecommendRouteQuery = (query: RecommendRouteQuery): string => {
  const routeContext = resolveRecommendRouteContext(query)
  const params: string[] = []
  params.push(`source=${encodeURIComponent(stringifyRecommendSource(routeContext.source))}`)
  const scene = stringifyRecommendScene(routeContext.scene)
  // 推荐场景为空时不再拼接，避免产生无意义参数。
  if (scene) {
    params.push(`scene=${encodeURIComponent(scene)}`)
  }
  // requestId 为空时不透传，避免 URL 冗余。
  if (routeContext.requestId) {
    params.push(`requestId=${encodeURIComponent(routeContext.requestId)}`)
  }
  // 推荐位序号为 0 时也保留，确保首位商品能被稳定回溯。
  params.push(`index=${routeContext.position || 0}`)
  return params.join('&')
}

/** 构建带推荐上下文的商品详情页地址。 */
export const buildRecommendGoodsUrl = (goodsId: number, query: RecommendRouteQuery): string => {
  const routeQuery = buildRecommendRouteQuery(query)
  // 没有推荐参数时只保留商品 ID，避免生成多余的空查询串。
  if (!routeQuery) {
    return `/pages/goods/goods?id=${goodsId}`
  }
  return `/pages/goods/goods?id=${goodsId}&${routeQuery}`
}
