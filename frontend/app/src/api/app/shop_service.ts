import { http } from '@/utils/http'
import type { ListShopServicesResponse, ShopServiceService } from '@/rpc/app/v1/shop_service'
import type { Empty } from '@/rpc/google/protobuf/empty'

const SHOP_SERVICE_URL = '/v1/app/shop/service'

/** 服务列表服务 */
export class ShopServiceServiceImpl implements ShopServiceService {
  /** 查询服务列表 */
  async ListShopServices(request: Empty): Promise<ListShopServicesResponse> {
    const response = await http<Partial<ListShopServicesResponse>>({
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

  /** 查询服务列表（旧生成接口兼容） */
  ListShopService(request: Empty): Promise<ListShopServicesResponse> {
    return this.ListShopServices(request)
  }
}

export const defShopServiceService = new ShopServiceServiceImpl()
