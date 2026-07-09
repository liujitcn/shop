import { http } from '@/utils/http'
import type {
  GetTenantStoreRequest,
  TenantStore,
  TenantStoreService,
} from '@/rpc/app/v1/tenant_store'

const TENANT_STORE_URL = '/v1/app/tenant/store'

/** 租户门店服务 */
export class TenantStoreServiceImpl implements TenantStoreService {
  /** 查询租户门店首页 */
  GetTenantStore(request: GetTenantStoreRequest): Promise<TenantStore> {
    return http<TenantStore>({
      url: `${TENANT_STORE_URL}/${request.id}`,
      method: 'GET',
      authMode: 'optional',
    })
  }
}

export const defTenantStoreService = new TenantStoreServiceImpl()
