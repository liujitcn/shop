import { http } from '@/utils/http'
import type {
  RecommendExposureRequest,
  RecommendGoodsRequest,
  RecommendGoodsResponse,
  RecommendService,
} from '@/rpc/app/recommend'
import type { Empty } from '@/rpc/google/protobuf/empty'

const RECOMMEND_URL = '/app/recommend'

/** 推荐服务 */
export class RecommendServiceImpl implements RecommendService {
  /** 查询推荐商品列表 */
  RecommendGoods(request: RecommendGoodsRequest): Promise<RecommendGoodsResponse> {
    const data = {
      ...request,
    }
    if (!data.cartGoodsIds || data.cartGoodsIds.length === 0) {
      delete (data as Partial<RecommendGoodsRequest>).cartGoodsIds
    }
    return http<RecommendGoodsResponse>({
      url: `${RECOMMEND_URL}/goods`,
      method: 'GET',
      data,
    })
  }

  /** 记录推荐曝光 */
  RecommendExposure(request: RecommendExposureRequest): Promise<Empty> {
    return http<Empty>({
      url: `${RECOMMEND_URL}/exposure`,
      method: 'POST',
      data: request,
    })
  }
}

export const defRecommendService = new RecommendServiceImpl()
