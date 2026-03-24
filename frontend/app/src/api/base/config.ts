import { http } from '@/utils/http'
import type { ConfigRequest, ConfigResponse, ConfigService } from '@/rpc/base/config'

const CONFIG_URL = '/config'

/** 系统配置公共服务 */
export class ConfigServiceImpl implements ConfigService {
  /** 获取系统配置 */
  GetConfig(request: ConfigRequest): Promise<ConfigResponse> {
    return http<ConfigResponse>({
      url: `${CONFIG_URL}`,
      method: 'GET',
      data: request,
    })
  }
}

export const defConfigService = new ConfigServiceImpl()
