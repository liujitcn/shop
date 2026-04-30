import { http } from '@/utils/http'
import type {
  BaseAreaService,
  TreeBaseAreasRequest,
  TreeBaseAreasResponse,
} from '@/rpc/app/v1/base_area'
import type { AppTreeOptionResponse_Option } from '@/rpc/common/v1/common'

const BASE_AREA_URL = '/v1/app/base/area'

type TreeBaseAreasResponseCompat = TreeBaseAreasResponse & {
  list: AppTreeOptionResponse_Option[]
}

type TreeBaseAreasHTTPResponse = Partial<TreeBaseAreasResponse> & {
  list?: AppTreeOptionResponse_Option[]
}

/** 行政区域服务 */
export class BaseAreaServiceImpl implements BaseAreaService {
  /** 查询行政区域树形列表 */
  async TreeBaseAreas(request: TreeBaseAreasRequest): Promise<TreeBaseAreasResponseCompat> {
    const response = await http<TreeBaseAreasHTTPResponse>({
      url: `${BASE_AREA_URL}/tree`,
      method: 'GET',
      data: request,
    })
    // 兼容未生成前的旧响应 list，同时向新协议的 areas 字段收敛。
    const areas = response.areas ?? response.list ?? []
    return {
      ...response,
      list: areas,
      areas,
    }
  }
}

export const defBaseAreaService = new BaseAreaServiceImpl()
