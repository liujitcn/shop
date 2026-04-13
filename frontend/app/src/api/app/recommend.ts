import { http } from '@/utils/http'
import type {
  RecommendExposureReportRequest,
  RecommendGoodsActionReportRequest,
  RecommendGoodsRequest,
  RecommendGoodsResponse,
  RecommendService,
} from '@/rpc/app/recommend'
import type { Empty } from '@/rpc/google/protobuf/empty'
import type { Int64Value } from '@/rpc/google/protobuf/wrappers'
import { useRecommendStore } from '@/stores/modules/recommend'

const RECOMMEND_URL = '/app/recommend'

/** 推荐服务 */
export class RecommendServiceImpl implements RecommendService {
  /** 获取匿名推荐主体 */
  RecommendAnonymousActor(_: Empty): Promise<Int64Value> {
    return http<Int64Value>({
      url: `${RECOMMEND_URL}/actor/anonymous`,
      method: 'GET',
    })
  }

  /** 绑定匿名推荐主体到当前登录用户 */
  BindRecommendAnonymousActor(_: Empty): Promise<Empty> {
    return http<Empty>({
      url: `${RECOMMEND_URL}/actor/binding`,
      method: 'POST',
      // 推荐匿名标识统一由 store 维护，API 层只负责透传到请求头。
      header: useRecommendStore().buildAnonymousHeader(),
      data: {},
    })
  }

  /** 查询推荐商品列表 */
  RecommendGoods(request: RecommendGoodsRequest): Promise<RecommendGoodsResponse> {
    return http<RecommendGoodsResponse>({
      url: `${RECOMMEND_URL}/goods`,
      method: 'GET',
      // 商品推荐请求依赖匿名标识命中未登录用户画像。
      header: useRecommendStore().buildAnonymousHeader(),
      data: request,
    })
  }

  /** 上报推荐曝光事件 */
  RecommendExposureReport(request: RecommendExposureReportRequest): Promise<Empty> {
    return http<Empty>({
      url: `${RECOMMEND_URL}/event/exposure`,
      method: 'POST',
      // 埋点接口和推荐查询共用同一份匿名标识。
      header: useRecommendStore().buildAnonymousHeader(),
      data: request,
    })
  }

  /** 上报推荐商品行为事件 */
  RecommendGoodsActionReport(request: RecommendGoodsActionReportRequest): Promise<Empty> {
    return http<Empty>({
      url: `${RECOMMEND_URL}/event/goods`,
      method: 'POST',
      // 行为上报需要和曝光、查询落在同一个匿名主体上。
      header: useRecommendStore().buildAnonymousHeader(),
      data: request,
    })
  }
}

export const defRecommendService = new RecommendServiceImpl()
