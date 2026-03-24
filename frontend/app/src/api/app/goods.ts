import { http } from '@/utils/http'
import type {
  GoodsResponse,
  GoodsService,
  PageGoodsRequest,
  PageGoodsResponse,
} from '@/rpc/app/goods'
import type { Int64Value } from '@/rpc/google/protobuf/wrappers'

const GOODS_URL = '/app/goods/info'

/** 商品服务 */
export class GoodsServiceImpl implements GoodsService {
  /** 查询商品分页列表 */
  PageGoods(request: PageGoodsRequest): Promise<PageGoodsResponse> {
    return http<PageGoodsResponse>({
      url: `${GOODS_URL}`,
      method: 'GET',
      data: request,
    })
  }

  GetGoods(request: Int64Value): Promise<GoodsResponse> {
    return http<GoodsResponse>({
      url: `${GOODS_URL}/${request.value}`,
      method: 'GET',
    })
  }
}

export const defGoodsService = new GoodsServiceImpl()
