import { http } from '@/utils/http'
import type { ListShopBannerResponse, ShopBannerService } from '@/rpc/app/v1/shop_banner'

const SHOP_BANNER_URL = '/v1/app/shop/banner'

/** 轮播图列表请求兼容站点筛选参数。 */
type ListShopBannerRequestCompat = {
  site?: number
}

/** 轮播图服务 */
export class ShopBannerServiceImpl implements ShopBannerService {
  /** 查询轮播图列表 */
  async ListShopBanner(request: ListShopBannerRequestCompat): Promise<ListShopBannerResponse> {
    const response = await http<Partial<ListShopBannerResponse>>({
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
}

export const defShopBannerService = new ShopBannerServiceImpl()
