import { http } from '@/utils/http'
import {
  type AuthService,
  type BindUserPhoneRequest,
  type BindUserPhoneResponse,
  type UserProfileForm,
  type WechatLoginRequest,
  type WechatLoginResponse,
} from '@/rpc/app/auth'
import type { Empty } from '@/rpc/google/protobuf/empty'

const AUTH_URL = '/app/auth'

/** 用户登录认证服务 */
export class AuthServiceImpl implements AuthService {
  /** 微信登录 */
  WechatLogin(request: WechatLoginRequest): Promise<WechatLoginResponse> {
    return http<WechatLoginResponse>({
      url: `${AUTH_URL}/wechat`,
      method: 'POST',
      data: request,
    })
  }
  /** 获取已登录用户资料 */
  GetUserProfile(request: Empty): Promise<UserProfileForm> {
    return http<UserProfileForm>({
      url: `${AUTH_URL}/profile`,
      method: 'GET',
      data: request,
    })
  }
  /** 修改个人中心用户信息 */
  UpdateUserProfile(request: UserProfileForm): Promise<Empty> {
    return http<Empty>({
      url: `${AUTH_URL}/profile`,
      method: 'PUT',
      data: request,
    })
  }
  /** 手机号授权 */
  BindUserPhone(request: BindUserPhoneRequest): Promise<BindUserPhoneResponse> {
    return http<BindUserPhoneResponse>({
      url: `${AUTH_URL}/phone`,
      method: 'PUT',
      data: request,
    })
  }
}

export const defAuthService = new AuthServiceImpl()
