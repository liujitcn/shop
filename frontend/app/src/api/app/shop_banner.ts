import { http } from '@/utils/http'
import type { ShopBanner, ShopBannerService } from '@/rpc/app/v1/shop_banner'

const SHOP_BANNER_URL = '/v1/app/shop/banner'

/** 轮播图列表请求兼容站点筛选参数。 */
type ListShopBannersRequestCompat = {
  site?: number
}

/** 轮播图列表响应兼容结构，同时保留协议字段和旧版 list 字段。 */
type ListShopBannersResponseCompat = {
  shop_banners: ShopBanner[]
  list: ShopBanner[]
}

/** 轮播图 HTTP 原始响应，允许后端只返回部分字段。 */
type ListShopBannersHTTPResponse = Partial<ListShopBannersResponseCompat>

/** 轮播图服务 */
export class ShopBannerServiceImpl implements ShopBannerService {
  /** 查询轮播图列表 */
  async ListShopBanners(
    request: ListShopBannersRequestCompat,
  ): Promise<ListShopBannersResponseCompat> {
    const response = await http<ListShopBannersHTTPResponse>({
      url: `${SHOP_BANNER_URL}`,
      method: 'GET',
      data: request,
    })
    const shopBanners = response.shop_banners ?? response.list ?? []
    return {
      ...response,
      list: shopBanners,
      shop_banners: shopBanners,
    }
  }

  /** 查询轮播图列表（旧生成接口兼容） */
  ListShopBanner(request: ListShopBannersRequestCompat): Promise<ListShopBannersResponseCompat> {
    return this.ListShopBanners(request)
  }
}

export const defShopBannerService = new ShopBannerServiceImpl()
