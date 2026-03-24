import { http } from '@/utils/http'
import type {
  ShopHotService,
  ListShopHotResponse,
  ListShopHotItemResponse,
  PageShopHotGoodsRequest,
  PageShopHotGoodsResponse,
} from '@/rpc/app/shop_hot'
import { type Empty } from '@/rpc/google/protobuf/empty'
import { type Int64Value } from '@/rpc/google/protobuf/wrappers'
const SHOP_HOT_URL = '/app/shop/hot'

/** 热们推荐服务 */
export class ShopHotServiceImpl implements ShopHotService {
  /** 查询热门推荐列表 */
  ListShopHot(request: Empty): Promise<ListShopHotResponse> {
    return http<ListShopHotResponse>({
      url: `${SHOP_HOT_URL}`,
      method: 'GET',
      data: request,
    })
  }
  /** 查询热门推荐选项 */
  ListShopHotItem(request: Int64Value): Promise<ListShopHotItemResponse> {
    return http<ListShopHotItemResponse>({
      url: `${SHOP_HOT_URL}/item`,
      method: 'GET',
      data: request,
    })
  }
  /** 查询热门推荐商品 */
  PageShopHotGoods(request: PageShopHotGoodsRequest): Promise<PageShopHotGoodsResponse> {
    return http<PageShopHotGoodsResponse>({
      url: `${SHOP_HOT_URL}/goods`,
      method: 'GET',
      data: request,
    })
  }
}

export const defShopHotService = new ShopHotServiceImpl()
