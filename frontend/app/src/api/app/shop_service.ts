import { http } from '@/utils/http'
import type { ShopService, ShopServiceService } from '@/rpc/app/v1/shop_service'
import type { Empty } from '@/rpc/google/protobuf/empty'

const SHOP_SERVICE_URL = '/v1/app/shop/service'

type ListShopServicesResponseCompat = {
  shop_services: ShopService[]
  list: ShopService[]
}

type ListShopServicesHTTPResponse = Partial<ListShopServicesResponseCompat>

/** 服务列表服务 */
export class ShopServiceServiceImpl implements ShopServiceService {
  /** 查询服务列表 */
  async ListShopServices(request: Empty): Promise<ListShopServicesResponseCompat> {
    const response = await http<ListShopServicesHTTPResponse>({
      url: `${SHOP_SERVICE_URL}`,
      method: 'GET',
      data: request,
    })
    const shopServices = response.shop_services ?? response.list ?? []
    return {
      ...response,
      list: shopServices,
      shop_services: shopServices,
    }
  }

  /** 查询服务列表（旧生成接口兼容） */
  ListShopService(request: Empty): Promise<ListShopServicesResponseCompat> {
    return this.ListShopServices(request)
  }
}

export const defShopServiceService = new ShopServiceServiceImpl()
