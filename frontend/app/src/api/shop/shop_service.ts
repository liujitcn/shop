import { http } from '@/utils/http'
import type { ListShopServiceResponse, ShopServiceService } from '@/rpc/shop/app/v1/shop_service'
import type { Empty } from '@/rpc/google/protobuf/empty'

const SHOP_SERVICE_URL = '/v1/app/shop/service'

/** 服务列表服务 */
export class ShopServiceServiceImpl implements ShopServiceService {
  /** 查询服务列表 */
  async ListShopService(request: Empty): Promise<ListShopServiceResponse> {
    const response = await http<Partial<ListShopServiceResponse>>({
      url: `${SHOP_SERVICE_URL}`,
      method: 'GET',
      authMode: 'none',
      data: request,
    })
    return {
      ...response,
      shop_services: response.shop_services ?? [],
    }
  }
}

export const defShopServiceService = new ShopServiceServiceImpl()
