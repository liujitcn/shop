import { http } from '@/utils/http'
import type {
  ShopBannerService,
  ListShopBannerRequest,
  ListShopBannerResponse,
} from '@/rpc/app/shop_banner'
const SHOP_BANNER_URL = '/app/shop/banner'

/** 轮播图服务 */
export class ShopBannerServiceImpl implements ShopBannerService {
  /** 查询轮播图分页列表 */
  ListShopBanner(request: ListShopBannerRequest): Promise<ListShopBannerResponse> {
    return http<ListShopBannerResponse>({
      url: `${SHOP_BANNER_URL}`,
      method: 'GET',
      data: request,
    })
  }
}

export const defShopBannerService = new ShopBannerServiceImpl()
