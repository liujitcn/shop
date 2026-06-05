import { http } from '@/utils/http'
import type { GoodsInfo } from '@/rpc/app/v1/goods_info'
import type { ShopHot, ShopHotItem, ShopHotService } from '@/rpc/app/v1/shop_hot'
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

/** 热门推荐列表响应兼容结构，同时保留协议字段和旧版 list。 */
type ListShopHotsResponseCompat = {
  shop_hots: ShopHot[]
  list: ShopHot[]
}

/** 热门推荐列表 HTTP 原始响应，允许后端只返回部分字段。 */
type ListShopHotsHTTPResponse = Partial<ListShopHotsResponseCompat>

/** 热门推荐子项响应兼容结构，同时保留协议字段和旧版 list。 */
type ListShopHotItemsResponseCompat = {
  id: number
  title: string
  banner: string
  shop_hot_items: ShopHotItem[]
  list: ShopHotItem[]
}

/** 热门推荐子项 HTTP 原始响应，允许后端只返回部分字段。 */
type ListShopHotItemsHTTPResponse = Partial<ListShopHotItemsResponseCompat>

/** 热门推荐商品分页响应兼容结构，同时保留协议字段和旧版 list。 */
type PageShopHotGoodsResponseCompat = {
  goods_infos: GoodsInfo[]
  list: GoodsInfo[]
  total: number
}

/** 热门推荐商品分页 HTTP 原始响应，允许后端只返回部分字段。 */
type PageShopHotGoodsHTTPResponse = Partial<PageShopHotGoodsResponseCompat>

/** 热门推荐服务 */
export class ShopHotServiceImpl implements ShopHotService {
  /** 查询热门推荐列表 */
  async ListShopHots(request: Empty): Promise<ListShopHotsResponseCompat> {
    const response = await http<ListShopHotsHTTPResponse>({
      url: `${SHOP_HOT_URL}`,
      method: 'GET',
      data: request,
    })
    const shopHots = response.shop_hots ?? response.list ?? []
    return {
      ...response,
      list: shopHots,
      shop_hots: shopHots,
    }
  }

  /** 查询热门推荐列表（旧生成接口兼容） */
  ListShopHot(request: Empty): Promise<ListShopHotsResponseCompat> {
    return this.ListShopHots(request)
  }

  /** 查询热门推荐选项 */
  async ListShopHotItems(request: IDRequestCompat): Promise<ListShopHotItemsResponseCompat> {
    const id = request.id ?? request.value ?? 0
    const response = await http<ListShopHotItemsHTTPResponse>({
      url: `${SHOP_HOT_URL}/${id}/item`,
      method: 'GET',
    })
    const shopHotItems = response.shop_hot_items ?? response.list ?? []
    return {
      id: response.id ?? id,
      title: response.title ?? '',
      banner: response.banner ?? '',
      list: shopHotItems,
      shop_hot_items: shopHotItems,
    }
  }

  /** 查询热门推荐选项（旧生成接口兼容） */
  ListShopHotItem(request: IDRequestCompat): Promise<ListShopHotItemsResponseCompat> {
    return this.ListShopHotItems(request)
  }

  /** 查询热门推荐商品 */
  async PageShopHotGoods(
    request: PageShopHotGoodsRequestCompat,
  ): Promise<PageShopHotGoodsResponseCompat> {
    const response = await http<PageShopHotGoodsHTTPResponse>({
      url: `${SHOP_HOT_URL}/item/${request.hot_item_id}/goods`,
      method: 'GET',
      data: request,
    })
    const goodsInfos = response.goods_infos ?? response.list ?? []
    return {
      ...response,
      list: goodsInfos,
      goods_infos: goodsInfos,
      total: response.total ?? 0,
    }
  }
}

export const defShopHotService = new ShopHotServiceImpl()
