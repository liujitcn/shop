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

/** 购物车 ID 请求兼容结构，支持旧版 value 和新版 id。 */
type IDRequestCompat = {
  id?: number
  value?: number
}

/** 更新购物车请求兼容结构，支持包裹表单和扁平表单。 */
type UpdateUserCartRequestCompat = Partial<UserCartForm> & {
  id: number
  user_cart?: UserCartForm
}

/** 购物车数量响应兼容结构，同时保留 count 和 value。 */
type CountUserCartResponseCompat = Int32Value & {
  count: number
  value: number
}

/** 购物车数量 HTTP 原始响应，允许 count 或 value 为空。 */
type CountUserCartHTTPResponse = Partial<CountUserCartResponseCompat>

/** 购物车列表响应兼容结构，同时保留协议字段和旧版 list。 */
type ListUserCartsResponseCompat = {
  user_carts: UserCart[]
  list: UserCart[]
}

/** 购物车列表 HTTP 原始响应，允许后端只返回部分字段。 */
type ListUserCartsHTTPResponse = Partial<ListUserCartsResponseCompat>

/** 购物车服务 */
export class UserCartServiceImpl implements UserCartService {
  /** 查询用户购物车数量 */
  async CountUserCart(request: Empty): Promise<CountUserCartResponseCompat> {
    const response = await http<CountUserCartHTTPResponse>({
      url: `${USER_CART_URL}/count`,
      method: 'GET',
      authMode: 'required',
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
      authMode: 'required',
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
      authMode: 'required',
      data: request,
    })
  }

  /** 更新购物车 */
  UpdateUserCart(request: UpdateUserCartRequestCompat): Promise<Empty> {
    const userCart = request.user_cart ?? (request as UserCartForm)
    return http<Empty>({
      url: `${USER_CART_URL}/${request.id}`,
      method: 'PUT',
      authMode: 'required',
      data: userCart,
    })
  }

  /** 删除购物车 */
  DeleteUserCart(request: IDRequestCompat): Promise<Empty> {
    const id = request.id ?? request.value ?? 0
    return http<Empty>({
      url: `${USER_CART_URL}/${id}`,
      method: 'DELETE',
      authMode: 'required',
    })
  }

  /** 设置状态 */
  SetUserCartStatus(request: SetUserCartStatusRequest): Promise<Empty> {
    return http<Empty>({
      url: `${USER_CART_URL}/${request.id}/status`,
      method: 'PUT',
      authMode: 'required',
      data: request,
    })
  }

  /** 设置全选 */
  SetUserCartSelection(request: SetUserCartSelectionRequest): Promise<Empty> {
    return http<Empty>({
      url: `${USER_CART_URL}/selection`,
      method: 'PUT',
      authMode: 'required',
      data: request,
    })
  }
}

export const defUserCartService = new UserCartServiceImpl()
