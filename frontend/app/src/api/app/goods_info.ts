import { http } from '@/utils/http'
import type {
  GetGoodsInfoRequest,
  GoodsInfo,
  GoodsInfoResponse,
  GoodsInfoService,
  PageGoodsInfoRequest,
  PageGoodsInfoResponse,
} from '@/rpc/app/v1/goods_info'

const GOODS_INFO_URL = '/v1/app/goods/info'

type PageGoodsInfoResponseCompat = PageGoodsInfoResponse & {
  list: GoodsInfo[]
}

type PageGoodsInfoHTTPResponse = Partial<PageGoodsInfoResponse> & {
  list?: GoodsInfo[]
}

/** 商品服务 */
export class GoodsInfoServiceImpl implements GoodsInfoService {
  /** 查询商品分页列表 */
  async PageGoodsInfo(request: PageGoodsInfoRequest): Promise<PageGoodsInfoResponseCompat> {
    const response = await http<PageGoodsInfoHTTPResponse>({
      url: `${GOODS_INFO_URL}`,
      method: 'GET',
      data: request,
    })
    // 兼容未生成前的旧响应 list，同时向新协议的 goodsInfos 字段收敛。
    const goodsInfos = response.goods_infos ?? response.list ?? []
    return {
      ...response,
      goods_infos: goodsInfos,
      list: goodsInfos,
      total: response.total ?? 0,
    }
  }

  GetGoodsInfo(request: GetGoodsInfoRequest): Promise<GoodsInfoResponse> {
    return http<GoodsInfoResponse>({
      url: `${GOODS_INFO_URL}/${request.id}`,
      method: 'GET',
    })
  }
}

export const defGoodsInfoService = new GoodsInfoServiceImpl()
