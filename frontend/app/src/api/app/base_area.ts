import { http } from '@/utils/http'
import type { BaseAreaService } from '@/rpc/app/base_area'
import type { Empty } from '@/rpc/google/protobuf/empty'
import type { AppTreeOptionResponse } from '@/rpc/common/common'

const BASE_AREA_URL = '/app/base/area'

/** 行政区域服务 */
export class BaseAreaServiceImpl implements BaseAreaService {
  /** 查询行政区域树形列表 */
  TreeBaseArea(request: Empty): Promise<AppTreeOptionResponse> {
    return http<AppTreeOptionResponse>({
      url: `${BASE_AREA_URL}/tree`,
      method: 'GET',
      data: request,
    })
  }
}

export const defBaseAreaService = new BaseAreaServiceImpl()
