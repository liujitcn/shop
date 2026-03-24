import { http } from '@/utils/http'
import type { UserStore, UserStoreForm, UserStoreService } from '@/rpc/app/user_store'
import type { Empty } from '@/rpc/google/protobuf/empty'

const USER_STORE_URL = '/app/user/store'

/** 用户门店服务 */
export class UserStoreServiceImpl implements UserStoreService {
  /** 查询用户门店 */
  GetUserStore(request: Empty): Promise<UserStore> {
    return http<UserStore>({
      url: `${USER_STORE_URL}`,
      method: 'GET',
      data: request,
    })
  }
  /** 创建用户门店 */
  CreateUserStore(request: UserStoreForm): Promise<Empty> {
    return http<Empty>({
      url: `${USER_STORE_URL}`,
      method: 'POST',
      data: request,
    })
  }
  /** 更新用户门店 */
  UpdateUserStore(request: UserStoreForm): Promise<Empty> {
    return http<Empty>({
      url: `${USER_STORE_URL}/${request.id}`,
      method: 'PUT',
      data: request,
    })
  }
}

export const defUserStoreService = new UserStoreServiceImpl()
