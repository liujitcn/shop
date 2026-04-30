import { http } from '@/utils/http'
import type { ShopBanner, ShopBannerService } from '@/rpc/app/v1/shop_banner'

const SHOP_BANNER_URL = '/v1/app/shop/banner'

type ListShopBannersRequestCompat = {
  site?: number
}

type ListShopBannersResponseCompat = {
  shop_banners: ShopBanner[]
  list: ShopBanner[]
}

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
