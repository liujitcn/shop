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
