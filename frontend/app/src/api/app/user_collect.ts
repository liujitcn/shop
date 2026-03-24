import { http } from '@/utils/http'
import type {
  UserCollectService,
  PageUserCollectRequest,
  PageUserCollectResponse,
  UserCollectForm,
  IsCollectRequest,
} from '@/rpc/app/user_collect'
import type { BoolValue, StringValue } from '@/rpc/google/protobuf/wrappers'
import type { Empty } from '@/rpc/google/protobuf/empty'

const USER_COLLECT_URL = '/app/user/collect'

/** 收藏服务 */
export class UserCollectServiceImpl implements UserCollectService {
  /** 查询用户收藏列表 */
  PageUserCollect(request: PageUserCollectRequest): Promise<PageUserCollectResponse> {
    return http<PageUserCollectResponse>({
      url: `${USER_COLLECT_URL}`,
      method: 'GET',
      data: request,
    })
  }
  /** 查询用户是否收藏 */
  GetIsCollect(request: IsCollectRequest): Promise<BoolValue> {
    return http<BoolValue>({
      url: `${USER_COLLECT_URL}/status`,
      method: 'GET',
      data: request,
    })
  }
  /** 创建用户收藏 */
  CreateUserCollect(request: UserCollectForm): Promise<Empty> {
    return http<Empty>({
      url: `${USER_COLLECT_URL}`,
      method: 'POST',
      data: request,
    })
  }
  /** 删除用户收藏 */
  DeleteUserCollect(request: StringValue): Promise<Empty> {
    return http<Empty>({
      url: `${USER_COLLECT_URL}/${request.value}`,
      method: 'DELETE',
    })
  }
}

export const defUserCollectService = new UserCollectServiceImpl()
