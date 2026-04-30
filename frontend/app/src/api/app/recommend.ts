import { http } from '@/utils/http'
import type { GoodsInfo } from '@/rpc/app/v1/goods_info'
import type {
  BindRecommendAnonymousActorRequest,
  RecommendAnonymousActorRequest,
  RecommendAnonymousActorResponse,
  RecommendEventReportRequest,
  RecommendGoodsRequest,
  RecommendGoodsResponse,
  RecommendService,
} from '@/rpc/app/v1/recommend'
import type { Empty } from '@/rpc/google/protobuf/empty'
import { useRecommendStore } from '@/stores/modules/recommend'

const RECOMMEND_URL = '/v1/app/recommend'

type RecommendAnonymousActorResponseCompat = RecommendAnonymousActorResponse & {
  value: number
}

type RecommendAnonymousActorHTTPResponse = Partial<RecommendAnonymousActorResponse> & {
  value?: number
}

type RecommendGoodsResponseCompat = RecommendGoodsResponse & {
  list: GoodsInfo[]
}

type RecommendGoodsHTTPResponse = Partial<RecommendGoodsResponse> & {
  list?: GoodsInfo[]
}

/** 推荐服务 */
export class RecommendServiceImpl implements RecommendService {
  /** 获取匿名推荐主体 */
  async RecommendAnonymousActor(
    _: RecommendAnonymousActorRequest,
  ): Promise<RecommendAnonymousActorResponseCompat> {
    const response = await http<RecommendAnonymousActorHTTPResponse>({
      url: `${RECOMMEND_URL}/actor/anonymous`,
      method: 'GET',
    })
    // 兼容未生成前的旧包装响应 value，同时向新协议的 anonymousId 字段收敛。
    const anonymousId = response.anonymous_id ?? response.value ?? 0
    return {
      ...response,
      anonymous_id: anonymousId,
      value: anonymousId,
    }
  }

  /** 绑定匿名推荐主体到当前登录用户 */
  BindRecommendAnonymousActor(_: BindRecommendAnonymousActorRequest): Promise<Empty> {
    return http<Empty>({
      url: `${RECOMMEND_URL}/actor/binding`,
      method: 'POST',
      // 推荐匿名标识统一由 store 维护，API 层只负责透传到请求头。
      header: useRecommendStore().buildAnonymousHeader(),
      data: {},
    })
  }

  /** 查询推荐商品列表 */
  async RecommendGoods(request: RecommendGoodsRequest): Promise<RecommendGoodsResponseCompat> {
    const response = await http<RecommendGoodsHTTPResponse>({
      url: `${RECOMMEND_URL}/goods`,
      method: 'GET',
      // 商品推荐请求依赖匿名标识命中未登录用户画像。
      header: useRecommendStore().buildAnonymousHeader(),
      data: request,
    })
    // 兼容未生成前的旧响应 list，同时向新协议的 goodsInfos 字段收敛。
    const goodsInfos = response.goods_infos ?? response.list ?? []
    return {
      ...response,
      goods_infos: goodsInfos,
      list: goodsInfos,
      total: response.total ?? 0,
      request_id: response.request_id ?? 0,
    }
  }

  /** 上报推荐事件 */
  RecommendEventReport(request: RecommendEventReportRequest): Promise<Empty> {
    return http<Empty>({
      url: `${RECOMMEND_URL}/event`,
      method: 'POST',
      // 埋点接口和推荐查询共用同一份匿名标识。
      header: useRecommendStore().buildAnonymousHeader(),
      data: request,
    })
  }
}

export const defRecommendService = new RecommendServiceImpl()
