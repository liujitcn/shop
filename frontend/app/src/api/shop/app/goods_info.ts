import { http } from '@/utils/http'
import type {
  GetGoodsInfoRequest,
  GoodsInfoResponse,
  GoodsInfoService,
  PageGoodsInfoRequest,
  PageGoodsInfoResponse,
} from '@/rpc/shop/app/v1/goods_info'

const GOODS_INFO_URL = '/v1/app/goods/info'

/** 商品服务 */
export class GoodsInfoServiceImpl implements GoodsInfoService {
  /** 查询商品分页列表 */
  async PageGoodsInfo(request: PageGoodsInfoRequest): Promise<PageGoodsInfoResponse> {
    const response = await http<Partial<PageGoodsInfoResponse>>({
      url: `${GOODS_INFO_URL}`,
      method: 'GET',
      authMode: 'optional',
      data: request,
    })
    return {
      ...response,
      goods_infos: response.goods_infos ?? [],
      total: response.total ?? 0,
    }
  }

  /** 查询商品详情 */
  GetGoodsInfo(request: GetGoodsInfoRequest): Promise<GoodsInfoResponse> {
    return http<GoodsInfoResponse>({
      url: `${GOODS_INFO_URL}/${request.id}`,
      method: 'GET',
      authMode: 'optional',
    })
  }
}

export const defGoodsInfoService = new GoodsInfoServiceImpl()
