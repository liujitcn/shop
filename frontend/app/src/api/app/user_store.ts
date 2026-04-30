import { http } from '@/utils/http'
import type { UserStore, UserStoreForm, UserStoreService } from '@/rpc/app/v1/user_store'
import type { Empty } from '@/rpc/google/protobuf/empty'

const USER_STORE_URL = '/v1/app/user/store'

type UserStoreFormRequestCompat = Partial<UserStoreForm> & {
  id?: number
  user_store?: UserStoreForm
}

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
  CreateUserStore(request: UserStoreFormRequestCompat): Promise<Empty> {
    const userStore = request.user_store ?? (request as UserStoreForm)
    return http<Empty>({
      url: `${USER_STORE_URL}`,
      method: 'POST',
      data: userStore,
    })
  }

  /** 更新用户门店 */
  UpdateUserStore(request: UserStoreFormRequestCompat): Promise<Empty> {
    const userStore = request.user_store ?? (request as UserStoreForm)
    const id = request.id ?? userStore.id
    return http<Empty>({
      url: `${USER_STORE_URL}/${id}`,
      method: 'PUT',
      data: userStore,
    })
  }
}

export const defUserStoreService = new UserStoreServiceImpl()
