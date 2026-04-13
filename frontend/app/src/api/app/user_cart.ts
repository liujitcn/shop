import { http } from '@/utils/http'
import type {
  ListUserCartResponse,
  CreateUserCartRequest,
  SetUserCartSelectionRequest,
  UserCartForm,
  UserCartService,
  SetUserCartStatusRequest,
} from '@/rpc/app/user_cart'
import type { Int64Value, Int32Value } from '@/rpc/google/protobuf/wrappers'
import type { Empty } from '@/rpc/google/protobuf/empty'
import type { SetStatusRequest } from '@/rpc/common/common'

const USER_CART_URL = '/app/user/cart'

/** 购物车服务 */
export class UserCartServiceImpl implements UserCartService {
  /** 查询用户购物车数量 */
  CountUserCart(request: Empty): Promise<Int32Value> {
    return http<Int32Value>({
      url: `${USER_CART_URL}/count`,
      method: 'GET',
      data: request,
    })
  }
  /** 查询购物车列表 */
  ListUserCart(request: Empty): Promise<ListUserCartResponse> {
    return http<ListUserCartResponse>({
      url: `${USER_CART_URL}`,
      method: 'GET',
      data: request,
    })
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
  UpdateUserCart(request: UserCartForm): Promise<Empty> {
    return http<Empty>({
      url: `${USER_CART_URL}/${request.id}`,
      method: 'PUT',
      data: request,
    })
  }
  /** 删除购物车 */
  DeleteUserCart(request: Int64Value): Promise<Empty> {
    return http<Empty>({
      url: `${USER_CART_URL}/${request.value}`,
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
