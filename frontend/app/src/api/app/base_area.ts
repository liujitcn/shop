import { http } from '@/utils/http'
import type {
  BaseAreaService,
  TreeBaseAreasRequest,
  TreeBaseAreasResponse,
} from '@/rpc/app/v1/base_area'

const BASE_AREA_URL = '/v1/app/base/area'

/** 行政区域服务 */
export class BaseAreaServiceImpl implements BaseAreaService {
  /** 查询行政区域树形列表 */
  async TreeBaseAreas(request: TreeBaseAreasRequest): Promise<TreeBaseAreasResponse> {
    const response = await http<Partial<TreeBaseAreasResponse>>({
      url: `${BASE_AREA_URL}/tree`,
      method: 'GET',
      data: request,
    })
    return {
      ...response,
      areas: response.areas ?? [],
    }
  }
}

export const defBaseAreaService = new BaseAreaServiceImpl()
