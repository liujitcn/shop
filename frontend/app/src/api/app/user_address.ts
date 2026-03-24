import { http } from '@/utils/http'
import type {
  ListUserAddressResponse,
  UserAddressForm,
  UserAddressService,
} from '@/rpc/app/user_address'
import type { Int64Value } from '@/rpc/google/protobuf/wrappers'
import type { Empty } from '@/rpc/google/protobuf/empty'

const USER_ADDRESS_URL = '/app/user/address'

/** 用户地址服务 */
export class UserAddressServiceImpl implements UserAddressService {
  /** 查询用户地址列表 */
  ListUserAddress(request: Empty): Promise<ListUserAddressResponse> {
    return http<ListUserAddressResponse>({
      url: `${USER_ADDRESS_URL}`,
      method: 'GET',
      data: request,
    })
  }
  /** 查询用户地址 */
  GetUserAddress(request: Int64Value): Promise<UserAddressForm> {
    return http<UserAddressForm>({
      url: `${USER_ADDRESS_URL}/${request.value}`,
      method: 'GET',
    })
  }
  /** 创建用户地址 */
  CreateUserAddress(request: UserAddressForm): Promise<Empty> {
    return http<Empty>({
      url: `${USER_ADDRESS_URL}`,
      method: 'POST',
      data: request,
    })
  }
  /** 更新用户地址 */
  UpdateUserAddress(request: UserAddressForm): Promise<Empty> {
    return http<Empty>({
      url: `${USER_ADDRESS_URL}/${request.id}`,
      method: 'PUT',
      data: request,
    })
  }
  /** 删除用户地址 */
  DeleteUserAddress(request: Int64Value): Promise<Empty> {
    return http<Empty>({
      url: `${USER_ADDRESS_URL}/${request.value}`,
      method: 'DELETE',
    })
  }
}

export const defUserAddressService = new UserAddressServiceImpl()
