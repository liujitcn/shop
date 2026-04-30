import { http } from '@/utils/http'
import type { UserAddress, UserAddressForm, UserAddressService } from '@/rpc/app/v1/user_address'
import type { Empty } from '@/rpc/google/protobuf/empty'

const USER_ADDRESS_URL = '/v1/app/user/address'

type IDRequestCompat = {
  id?: number
  value?: number
}

type UserAddressFormRequestCompat = Partial<UserAddressForm> & {
  id?: number
  user_address?: UserAddressForm
}

type ListUserAddressesResponseCompat = {
  user_addresses: UserAddress[]
  list: UserAddress[]
}

type ListUserAddressesHTTPResponse = Partial<ListUserAddressesResponseCompat>

/** 用户地址服务 */
export class UserAddressServiceImpl implements UserAddressService {
  /** 查询用户地址列表 */
  async ListUserAddresses(request: Empty): Promise<ListUserAddressesResponseCompat> {
    const response = await http<ListUserAddressesHTTPResponse>({
      url: `${USER_ADDRESS_URL}`,
      method: 'GET',
      data: request,
    })
    const userAddresses = response.user_addresses ?? response.list ?? []
    return {
      ...response,
      list: userAddresses,
      user_addresses: userAddresses,
    }
  }

  /** 查询用户地址列表（旧生成接口兼容） */
  ListUserAddress(request: Empty): Promise<ListUserAddressesResponseCompat> {
    return this.ListUserAddresses(request)
  }

  /** 查询用户地址 */
  GetUserAddress(request: IDRequestCompat): Promise<UserAddressForm> {
    const id = request.id ?? request.value ?? 0
    return http<UserAddressForm>({
      url: `${USER_ADDRESS_URL}/${id}`,
      method: 'GET',
    })
  }

  /** 创建用户地址 */
  CreateUserAddress(request: UserAddressFormRequestCompat): Promise<Empty> {
    const userAddress = request.user_address ?? (request as UserAddressForm)
    return http<Empty>({
      url: `${USER_ADDRESS_URL}`,
      method: 'POST',
      data: userAddress,
    })
  }

  /** 更新用户地址 */
  UpdateUserAddress(request: UserAddressFormRequestCompat): Promise<Empty> {
    const userAddress = request.user_address ?? (request as UserAddressForm)
    const id = request.id ?? userAddress.id
    return http<Empty>({
      url: `${USER_ADDRESS_URL}/${id}`,
      method: 'PUT',
      data: userAddress,
    })
  }

  /** 删除用户地址 */
  DeleteUserAddress(request: IDRequestCompat): Promise<Empty> {
    const id = request.id ?? request.value ?? 0
    return http<Empty>({
      url: `${USER_ADDRESS_URL}/${id}`,
      method: 'DELETE',
    })
  }
}

export const defUserAddressService = new UserAddressServiceImpl()
