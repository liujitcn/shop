import { http } from '@/utils/http'
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

const RECOMMEND_URL = '/v1/app/recommend'
type RecommendRequestHeader = UniApp.RequestOptions['header']

/** 推荐服务 */
export class RecommendServiceImpl implements RecommendService {
  /** 获取匿名推荐主体 */
  RecommendAnonymousActor(_: RecommendAnonymousActorRequest) {
    return http<RecommendAnonymousActorResponse>({
      url: `${RECOMMEND_URL}/actor/anonymous`,
      method: 'GET',
    })
  }

  /** 绑定匿名推荐主体到当前登录用户 */
  BindRecommendAnonymousActor(
    _: BindRecommendAnonymousActorRequest,
    header?: RecommendRequestHeader,
  ): Promise<Empty> {
    return http<Empty>({
      url: `${RECOMMEND_URL}/actor/binding`,
      method: 'POST',
      header,
      data: {},
    })
  }

  /** 查询推荐商品列表 */
  RecommendGoods(request: RecommendGoodsRequest, header?: RecommendRequestHeader) {
    return http<RecommendGoodsResponse>({
      url: `${RECOMMEND_URL}/goods`,
      method: 'GET',
      header,
      data: request,
    })
  }

  /** 上报推荐事件 */
  RecommendEventReport(
    request: RecommendEventReportRequest,
    header?: RecommendRequestHeader,
  ): Promise<Empty> {
    return http<Empty>({
      url: `${RECOMMEND_URL}/event`,
      method: 'POST',
      header,
      data: request,
    })
  }
}

export const defRecommendService = new RecommendServiceImpl()
