import { http } from '@/utils/http'
import type { BaseDictForm, BaseDictService } from '@/rpc/app/base_dict'
import type { StringValue } from '@/rpc/google/protobuf/wrappers'

const BASE_DICT_URL = '/app/base/dict'

/** 字典服务 */
export class BaseDictServiceImpl implements BaseDictService {
  /** 查询单个字典 */
  GetBaseDict(request: StringValue): Promise<BaseDictForm> {
    return http<BaseDictForm>({
      url: `${BASE_DICT_URL}/${request.value}`,
      method: 'GET',
    })
  }
}

export const defBaseDictService = new BaseDictServiceImpl()
