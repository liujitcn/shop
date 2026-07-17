import { http } from '@/utils/http'
import type {
  CreateUserCollectRequest,
  DeleteUserCollectRequest,
  GetIsCollectRequest,
  GetIsCollectResponse,
  PageUserCollectRequest,
  PageUserCollectResponse,
  UserCollectForm,
  UserCollectService,
} from '@/rpc/app/v1/user_collect'
import type { Empty } from '@/rpc/google/protobuf/empty'

const USER_COLLECT_URL = '/v1/app/user/collect'

/** 创建收藏请求兼容结构，支持包裹请求和扁平表单。 */
type CreateUserCollectRequestCompat = CreateUserCollectRequest | UserCollectForm

/** 收藏服务 */
export class UserCollectServiceImpl implements UserCollectService {
  /** 查询用户收藏列表 */
  async PageUserCollect(request: PageUserCollectRequest): Promise<PageUserCollectResponse> {
    const response = await http<Partial<PageUserCollectResponse>>({
      url: `${USER_COLLECT_URL}`,
      method: 'GET',
      authMode: 'required',
      data: request,
    })
    return {
      ...response,
      user_collects: response.user_collects ?? [],
      total: response.total ?? 0,
    }
  }

  /** 查询用户是否收藏 */
  async GetIsCollect(request: GetIsCollectRequest): Promise<GetIsCollectResponse> {
    const response = await http<Partial<GetIsCollectResponse>>({
      url: `${USER_COLLECT_URL}/status`,
      method: 'GET',
      authMode: 'optional',
      data: request,
    })
    return {
      ...response,
      is_collected: response.is_collected ?? false,
    }
  }

  /** 创建用户收藏 */
  CreateUserCollect(request: CreateUserCollectRequestCompat): Promise<Empty> {
    const userCollect = 'user_collect' in request ? request.user_collect : request
    return http<Empty>({
      url: `${USER_COLLECT_URL}`,
      method: 'POST',
      authMode: 'required',
      data: userCollect,
    })
  }

  /** 删除用户收藏 */
  DeleteUserCollect(request: DeleteUserCollectRequest): Promise<Empty> {
    return http<Empty>({
      url: `${USER_COLLECT_URL}/${request.ids}`,
      method: 'DELETE',
      authMode: 'required',
    })
  }
}

export const defUserCollectService = new UserCollectServiceImpl()
