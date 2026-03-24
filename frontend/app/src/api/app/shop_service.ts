import { http } from '@/utils/http'
import type { ShopServiceService, ListShopServiceResponse } from '@/rpc/app/shop_service'
import type { Empty } from '@/rpc/google/protobuf/empty.ts'
const SHOP_SERVICE_URL = '/app/shop/service'

/** 服务列表服务 */
export class ShopServiceServiceImpl implements ShopServiceService {
  /** 查询服务列表分页列表 */
  ListShopService(request: Empty): Promise<ListShopServiceResponse> {
    return http<ListShopServiceResponse>({
      url: `${SHOP_SERVICE_URL}`,
      method: 'GET',
      data: request,
    })
  }
}

export const defShopServiceService = new ShopServiceServiceImpl()
