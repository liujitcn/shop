import { http } from '@/utils/http'
import type {
  UserCart,
  CreateUserCartRequest,
  SetUserCartSelectionRequest,
  UserCartForm,
  UserCartService,
  SetUserCartStatusRequest,
} from '@/rpc/app/v1/user_cart'
import type { Int32Value } from '@/rpc/google/protobuf/wrappers'
import type { Empty } from '@/rpc/google/protobuf/empty'

const USER_CART_URL = '/v1/app/user/cart'

type IDRequestCompat = {
  id?: number
  value?: number
}

type UpdateUserCartRequestCompat = Partial<UserCartForm> & {
  id: number
  user_cart?: UserCartForm
}

type CountUserCartResponseCompat = Int32Value & {
  count: number
  value: number
}

type CountUserCartHTTPResponse = Partial<CountUserCartResponseCompat>

type ListUserCartsResponseCompat = {
  user_carts: UserCart[]
  list: UserCart[]
}

type ListUserCartsHTTPResponse = Partial<ListUserCartsResponseCompat>

/** 购物车服务 */
export class UserCartServiceImpl implements UserCartService {
  /** 查询用户购物车数量 */
  async CountUserCart(request: Empty): Promise<CountUserCartResponseCompat> {
    const response = await http<CountUserCartHTTPResponse>({
      url: `${USER_CART_URL}/count`,
      method: 'GET',
      data: request,
    })
    const count = response.count ?? response.value ?? 0
    return {
      ...response,
      count,
      value: count,
    }
  }

  /** 查询购物车列表 */
  async ListUserCarts(request: Empty): Promise<ListUserCartsResponseCompat> {
    const response = await http<ListUserCartsHTTPResponse>({
      url: `${USER_CART_URL}`,
      method: 'GET',
      data: request,
    })
    const userCarts = response.user_carts ?? response.list ?? []
    return {
      ...response,
      list: userCarts,
      user_carts: userCarts,
    }
  }

  /** 查询购物车列表（旧生成接口兼容） */
  ListUserCart(request: Empty): Promise<ListUserCartsResponseCompat> {
    return this.ListUserCarts(request)
  }

  /** 创建购物车 */
  CreateUserCart(request: CreateUserCartRequest): Promise<Empty> {
    return http<Empty>({
      url: `${USER_CART_URL}`,
      method: 'POST',
      data: request,
    })
  }

  /** 更新购物车 */
  UpdateUserCart(request: UpdateUserCartRequestCompat): Promise<Empty> {
    const userCart = request.user_cart ?? (request as UserCartForm)
    return http<Empty>({
      url: `${USER_CART_URL}/${request.id}`,
      method: 'PUT',
      data: userCart,
    })
  }

  /** 删除购物车 */
  DeleteUserCart(request: IDRequestCompat): Promise<Empty> {
    const id = request.id ?? request.value ?? 0
    return http<Empty>({
      url: `${USER_CART_URL}/${id}`,
      method: 'DELETE',
    })
  }

  /** 设置状态 */
  SetUserCartStatus(request: SetUserCartStatusRequest): Promise<Empty> {
    return http<Empty>({
      url: `${USER_CART_URL}/${request.id}/status`,
      method: 'PUT',
      data: request,
    })
  }

  /** 设置全选 */
  SetUserCartSelection(request: SetUserCartSelectionRequest): Promise<Empty> {
    return http<Empty>({
      url: `${USER_CART_URL}/selection`,
      method: 'PUT',
      data: request,
    })
  }
}

export const defUserCartService = new UserCartServiceImpl()
