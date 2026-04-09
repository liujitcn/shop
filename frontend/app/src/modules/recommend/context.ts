import type { RecommendContext, RecommendGoodsActionItem } from '@/rpc/app/recommend'
import { RecommendScene } from '@/rpc/common/enum'

/** 页面侧使用的推荐行为上下文，补充了 sku 和数量等本地字段。 */
export type RecommendGoodsActionContext = Omit<RecommendGoodsActionItem, 'recommendContext'> &
  RecommendContext & {
    skuCode?: string
    goodsNum?: number
  }

/** 为缺失字段补默认值，避免后续拼埋点对象时到处判空。 */
export const buildRecommendContext = (
  context: Partial<RecommendContext> = {},
): RecommendContext => {
  return {
    scene: context.scene ?? RecommendScene.RECOMMEND_SCENE_UNKNOWN,
    requestId: context.requestId || '',
    position: context.position || 0,
  }
}

/** 兼容路由 query 里字符串或枚举值形式的场景参数。 */
export const parseRecommendScene = (scene?: string): RecommendScene => {
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
export const stringifyRecommendScene = (scene?: RecommendScene): string => {
  const sceneValue = scene ?? RecommendScene.RECOMMEND_SCENE_UNKNOWN
  return sceneValue === RecommendScene.RECOMMEND_SCENE_UNKNOWN ? '' : RecommendScene[sceneValue]
}

/** 从完整上下文构建可直接上报的商品行为项。 */
export const buildRecommendGoodsActionItem = (
  context: RecommendGoodsActionContext,
): RecommendGoodsActionItem => {
  return {
    goodsId: context.goodsId,
    goodsNum: context.goodsNum || 1,
    recommendContext: buildRecommendContext(context),
  }
}

/** 已有 recommendContext 时复用，没有则补一份空上下文。 */
export const buildRecommendGoodsActionItemByContext = (
  goodsId: number,
  goodsNum: number,
  recommendContext?: RecommendContext,
): RecommendGoodsActionItem => {
  return {
    goodsId,
    goodsNum: goodsNum || 1,
    recommendContext: recommendContext || buildRecommendContext({}),
  }
}
