import { http } from '@/utils/http'
import type {
  CreateUserCollectRequest,
  DeleteUserCollectRequest,
  GetIsCollectRequest,
  GetIsCollectResponse,
  PageUserCollectsRequest,
  PageUserCollectsResponse,
  UserCollect,
  UserCollectForm,
  UserCollectService,
} from '@/rpc/app/v1/user_collect'
import type { Empty } from '@/rpc/google/protobuf/empty'

const USER_COLLECT_URL = '/v1/app/user/collect'

/** 创建收藏请求兼容结构，支持包裹请求和扁平表单。 */
type CreateUserCollectRequestCompat = CreateUserCollectRequest | UserCollectForm

/** 收藏状态响应兼容结构，同时保留 is_collected 和旧版 value。 */
type GetIsCollectResponseCompat = GetIsCollectResponse & {
  value: boolean
}

/** 收藏状态 HTTP 原始响应，允许任一状态字段为空。 */
type GetIsCollectHTTPResponse = Partial<GetIsCollectResponseCompat>

/** 收藏分页响应兼容结构，保留旧版 list 字段。 */
type PageUserCollectsResponseCompat = PageUserCollectsResponse & {
  list: UserCollect[]
}

/** 收藏分页 HTTP 原始响应，兼容旧版 list 字段。 */
type PageUserCollectsHTTPResponse = Partial<PageUserCollectsResponseCompat>

/** 收藏服务 */
export class UserCollectServiceImpl implements UserCollectService {
  /** 查询用户收藏列表 */
  async PageUserCollects(
    request: PageUserCollectsRequest,
  ): Promise<PageUserCollectsResponseCompat> {
    const response = await http<PageUserCollectsHTTPResponse>({
      url: `${USER_COLLECT_URL}`,
      method: 'GET',
      data: request,
    })
    const userCollects = response.user_collects ?? response.list ?? []
    return {
      ...response,
      list: userCollects,
      user_collects: userCollects,
      total: response.total ?? 0,
    }
  }

  /** 查询用户是否收藏 */
  async GetIsCollect(request: GetIsCollectRequest): Promise<GetIsCollectResponseCompat> {
    const response = await http<GetIsCollectHTTPResponse>({
      url: `${USER_COLLECT_URL}/status`,
      method: 'GET',
      data: request,
    })
    const isCollected = response.is_collected ?? response.value ?? false
    return {
      ...response,
      is_collected: isCollected,
      value: isCollected,
    }
  }

  /** 创建用户收藏 */
  CreateUserCollect(request: CreateUserCollectRequestCompat): Promise<Empty> {
    const userCollect = 'user_collect' in request ? request.user_collect : request
    return http<Empty>({
      url: `${USER_COLLECT_URL}`,
      method: 'POST',
      data: userCollect,
    })
  }

  /** 删除用户收藏 */
  DeleteUserCollect(request: DeleteUserCollectRequest): Promise<Empty> {
    return http<Empty>({
      url: `${USER_COLLECT_URL}/${request.ids}`,
      method: 'DELETE',
    })
  }
}

export const defUserCollectService = new UserCollectServiceImpl()
