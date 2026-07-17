import { http } from '@/utils/http'
import type {
  ListShopHotItemResponse,
  ListShopHotResponse,
  PageShopHotGoodsResponse,
  ShopHotService,
} from '@/rpc/app/v1/shop_hot'
import type { Empty } from '@/rpc/google/protobuf/empty'

const SHOP_HOT_URL = '/v1/app/shop/hot'

/** 热门推荐 ID 请求兼容结构，支持旧版 value 和新版 id。 */
type IDRequestCompat = {
  id?: number
  value?: number
}

/** 热门推荐商品分页请求结构。 */
type PageShopHotGoodsRequestCompat = {
  hot_item_id: number
  page_num: number
  page_size: number
}

/** 热门推荐服务 */
export class ShopHotServiceImpl implements ShopHotService {
  /** 查询热门推荐列表 */
  async ListShopHot(request: Empty): Promise<ListShopHotResponse> {
    const response = await http<Partial<ListShopHotResponse>>({
      url: `${SHOP_HOT_URL}`,
      method: 'GET',
      authMode: 'none',
      data: request,
    })
    return {
      ...response,
      shop_hots: response.shop_hots ?? [],
    }
  }

  /** 查询热门推荐选项 */
  async ListShopHotItem(request: IDRequestCompat): Promise<ListShopHotItemResponse> {
    const id = request.id ?? request.value ?? 0
    const response = await http<Partial<ListShopHotItemResponse>>({
      url: `${SHOP_HOT_URL}/${id}/item`,
      method: 'GET',
      authMode: 'none',
    })
    return {
      id: response.id ?? id,
      title: response.title ?? '',
      banner: response.banner ?? '',
      shop_hot_items: response.shop_hot_items ?? [],
    }
  }

  /** 查询热门推荐商品 */
  async PageShopHotGoods(
    request: PageShopHotGoodsRequestCompat,
  ): Promise<PageShopHotGoodsResponse> {
    const response = await http<Partial<PageShopHotGoodsResponse>>({
      url: `${SHOP_HOT_URL}/item/${request.hot_item_id}/goods`,
      method: 'GET',
      authMode: 'none',
      data: request,
    })
    return {
      ...response,
      goods_infos: response.goods_infos ?? [],
      total: response.total ?? 0,
    }
  }
}

export const defShopHotService = new ShopHotServiceImpl()
