import type { RecommendContext } from '@/rpc/app/recommend'
import { RecommendScene } from '@/rpc/common/enum'
import { buildRecommendContext } from './context'

/** 页面路由里允许 scene/index 以字符串形式透传。 */
export type RecommendRouteQuery = Omit<Partial<RecommendContext>, 'scene' | 'position'> & {
  scene?: RecommendScene | string
  index?: string | number
}

/** 路由里的 index 统一收敛为非负整数位置。 */
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

/** 兼容路由 query 里字符串或枚举值形式的场景参数。 */
const parseRecommendScene = (scene?: string): RecommendScene => {
  if (scene === undefined || scene === null || scene === '') {
    return RecommendScene.RECOMMEND_SCENE_UNKNOWN
  }
  const value = String(scene).trim()
  if (!value) {
    return RecommendScene.RECOMMEND_SCENE_UNKNOWN
  }
  if (/^\d+$/.test(value)) {
    const sceneValue = Number(value)
    return RecommendScene[sceneValue] ? sceneValue : RecommendScene.RECOMMEND_SCENE_UNKNOWN
  }
  return (
    (RecommendScene as unknown as Record<string, RecommendScene | undefined>)[value] ||
    RecommendScene.RECOMMEND_SCENE_UNKNOWN
  )
}

/** 仅在场景有效时输出 query 参数值。 */
const stringifyRecommendScene = (scene?: RecommendScene): string => {
  const sceneValue = scene ?? RecommendScene.RECOMMEND_SCENE_UNKNOWN
  return sceneValue === RecommendScene.RECOMMEND_SCENE_UNKNOWN ? '' : RecommendScene[sceneValue]
}

/** 从路由 query 解析出规范的推荐上下文。 */
const resolveRecommendRouteContext = (
  query: RecommendRouteQuery,
): RecommendContext => {
  return {
    scene:
      typeof query.scene === 'string'
        ? parseRecommendScene(query.scene)
        : (query.scene ?? parseRecommendScene('')),
    requestId: query.requestId || '',
    position: resolveRecommendIndex(query.index),
  }
}

/** 给页面入口统一补齐默认推荐上下文字段。 */
export const buildRecommendContextByRoute = (query: RecommendRouteQuery): RecommendContext => {
  return buildRecommendContext(resolveRecommendRouteContext(query))
}

/** 构造落在商品详情页上的推荐 query 字符串。 */
const buildRecommendRouteQuery = (query: RecommendRouteQuery): string => {
  const routeContext = resolveRecommendRouteContext(query)
  const params: string[] = []
  const scene = stringifyRecommendScene(routeContext.scene)
  if (scene) {
    params.push(`scene=${encodeURIComponent(scene)}`)
  }
  if (routeContext.requestId) {
    params.push(`requestId=${encodeURIComponent(routeContext.requestId)}`)
  }
  params.push(`index=${routeContext.position || 0}`)
  return params.join('&')
}

/** 构建带推荐上下文的商品详情页地址。 */
export const buildRecommendGoodsUrl = (goodsId: number, query: RecommendRouteQuery): string => {
  const routeQuery = buildRecommendRouteQuery(query)
  if (!routeQuery) {
    return `/pages/goods/goods?id=${goodsId}`
  }
  return `/pages/goods/goods?id=${goodsId}&${routeQuery}`
}
