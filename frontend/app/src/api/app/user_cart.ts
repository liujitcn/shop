import { http } from '@/utils/http'
import type {
  CountUserCartResponse,
  ListUserCartsResponse,
  CreateUserCartRequest,
  SetUserCartSelectionRequest,
  UserCartForm,
  UserCartService,
  SetUserCartStatusRequest,
} from '@/rpc/app/v1/user_cart'
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

/** 购物车服务 */
export class UserCartServiceImpl implements UserCartService {
  /** 查询用户购物车数量 */
  async CountUserCart(request: Empty): Promise<CountUserCartResponse> {
    const response = await http<Partial<CountUserCartResponse>>({
      url: `${USER_CART_URL}/count`,
      method: 'GET',
      authMode: 'required',
      data: request,
    })
    return {
      ...response,
      count: response.count ?? 0,
    }
  }

  /** 查询购物车列表 */
  ListUserCarts(request: Empty): Promise<ListUserCartsResponse> {
    return http<ListUserCartsResponse>({
      url: `${USER_CART_URL}`,
      method: 'GET',
      authMode: 'required',
      data: request,
    })
  }

  /** 查询购物车列表（旧生成接口兼容） */
  ListUserCart(request: Empty): Promise<ListUserCartsResponse> {
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
