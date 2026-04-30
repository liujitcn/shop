import { http } from '@/utils/http'
import type { BaseDictForm, BaseDictService } from '@/rpc/app/v1/base_dict'

const BASE_DICT_URL = '/v1/app/base/dict'

type GetBaseDictRequestCompat = {
  code?: string
  value?: string
}

/** 字典服务 */
export class BaseDictServiceImpl implements BaseDictService {
  /** 查询单个字典 */
  GetBaseDict(request: GetBaseDictRequestCompat): Promise<BaseDictForm> {
    const code = request.code ?? request.value ?? ''
    return http<BaseDictForm>({
      url: `${BASE_DICT_URL}/${code}`,
      method: 'GET',
    })
  }
}

export const defBaseDictService = new BaseDictServiceImpl()
