import { http } from '@/utils/http'
import type { ListShopBannersResponse, ShopBannerService } from '@/rpc/app/v1/shop_banner'

const SHOP_BANNER_URL = '/v1/app/shop/banner'

/** 轮播图列表请求兼容站点筛选参数。 */
type ListShopBannersRequestCompat = {
  site?: number
}

/** 轮播图服务 */
export class ShopBannerServiceImpl implements ShopBannerService {
  /** 查询轮播图列表 */
  async ListShopBanners(request: ListShopBannersRequestCompat): Promise<ListShopBannersResponse> {
    const response = await http<Partial<ListShopBannersResponse>>({
      url: `${SHOP_BANNER_URL}`,
      method: 'GET',
      authMode: 'none',
      data: request,
    })
    return {
      ...response,
      shop_banners: response.shop_banners ?? [],
    }
  }

  /** 查询轮播图列表（旧生成接口兼容） */
  ListShopBanner(request: ListShopBannersRequestCompat): Promise<ListShopBannersResponse> {
    return this.ListShopBanners(request)
  }
}

export const defShopBannerService = new ShopBannerServiceImpl()
