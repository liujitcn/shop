import { http } from '@/utils/http'
import type {
  ListUserAddressResponse,
  UserAddressForm,
  UserAddressService,
} from '@/rpc/app/v1/user_address'
import type { Empty } from '@/rpc/google/protobuf/empty'

const USER_ADDRESS_URL = '/v1/app/user/address'

/** 用户地址 ID 请求兼容结构，支持旧版 value 和新版 id。 */
type IDRequestCompat = {
  id?: number
  value?: number
}

/** 用户地址表单请求兼容结构，支持包裹表单和扁平表单。 */
type UserAddressFormRequestCompat = Partial<UserAddressForm> & {
  id?: number
  user_address?: UserAddressForm
}

/** 用户地址服务 */
export class UserAddressServiceImpl implements UserAddressService {
  /** 查询用户地址列表 */
  async ListUserAddress(request: Empty): Promise<ListUserAddressResponse> {
    const response = await http<Partial<ListUserAddressResponse>>({
      url: `${USER_ADDRESS_URL}`,
      method: 'GET',
      authMode: 'required',
      data: request,
    })
    return {
      ...response,
      user_addresses: response.user_addresses ?? [],
    }
  }

  /** 查询用户地址 */
  GetUserAddress(request: IDRequestCompat): Promise<UserAddressForm> {
    const id = request.id ?? request.value ?? 0
    return http<UserAddressForm>({
      url: `${USER_ADDRESS_URL}/${id}`,
      method: 'GET',
      authMode: 'required',
    })
  }

  /** 创建用户地址 */
  CreateUserAddress(request: UserAddressFormRequestCompat): Promise<Empty> {
    const userAddress = request.user_address ?? (request as UserAddressForm)
    return http<Empty>({
      url: `${USER_ADDRESS_URL}`,
      method: 'POST',
      authMode: 'required',
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
      authMode: 'required',
      data: userAddress,
    })
  }

  /** 删除用户地址 */
  DeleteUserAddress(request: IDRequestCompat): Promise<Empty> {
    const id = request.id ?? request.value ?? 0
    return http<Empty>({
      url: `${USER_ADDRESS_URL}/${id}`,
      method: 'DELETE',
      authMode: 'required',
    })
  }
}

export const defUserAddressService = new UserAddressServiceImpl()
