import { http } from '@/utils/http'
import type {
  GetGoodsInfoRequest,
  GoodsInfoResponse,
  GoodsInfoService,
  PageGoodsInfoRequest,
  PageGoodsInfoResponse,
} from '@/rpc/app/goods_info'

const GOODS_URL = '/app/goods/info'

/** 商品服务 */
export class GoodsServiceImpl implements GoodsInfoService {
  /** 查询商品分页列表 */
  PageGoodsInfo(request: PageGoodsInfoRequest): Promise<PageGoodsInfoResponse> {
    return http<PageGoodsInfoResponse>({
      url: `${GOODS_URL}`,
      method: 'GET',
      data: request,
    })
  }

  GetGoodsInfo(request: GetGoodsInfoRequest): Promise<GoodsInfoResponse> {
    return http<GoodsInfoResponse>({
      url: `${GOODS_URL}/${request.id}`,
      method: 'GET',
      data: request,
    })
  }
}

export const defGoodsInfoService = new GoodsServiceImpl()
