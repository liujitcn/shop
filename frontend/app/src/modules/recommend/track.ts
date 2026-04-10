import { defRecommendService } from '@/api/app/recommend'
import type { RecommendGoodsActionItem } from '@/rpc/app/recommend'
import { RecommendGoodsActionType } from '@/rpc/common/enum'
import { useRecommendStore } from '@/stores'

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
