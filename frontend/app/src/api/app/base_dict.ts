import { http } from '@/utils/http'
import type { BaseDictService, ListBaseDictResponse } from '@/rpc/app/base_dict'
import type { StringValue } from '@/rpc/google/protobuf/wrappers'

const BASE_DICT_URL = '/app/base/dict'

/** 字典服务 */
export class BaseDictServiceImpl implements BaseDictService {
  /** 查询字典列表 */
  ListBaseDict(request: StringValue): Promise<ListBaseDictResponse> {
    return http<ListBaseDictResponse>({
      url: `${BASE_DICT_URL}/list`,
      method: 'GET',
      data: request,
    })
  }
}

export const defBaseDictService = new BaseDictServiceImpl()
