import { http } from '@/utils/http'
import {
  type AuthService,
  type UserInfo,
  type PhoneAuthRequest,
  type PhoneAuthResponse,
  type UpdateUserInfoRequest,
  type WxLoginRequest,
  type WxLoginResponse,
} from '@/rpc/app/auth'
import type { Empty } from '@/rpc/google/protobuf/empty'

const AUTH_URL = '/app/auth'

/** 用户登录认证服务 */
export class AuthServiceImpl implements AuthService {
  /** 微信登录 */
  WxLogin(request: WxLoginRequest): Promise<WxLoginResponse> {
    return http<WxLoginResponse>({
      url: `${AUTH_URL}/login/wx`,
      method: 'POST',
      data: request,
    })
  }
  /** 获取已经登录的用户的数据 */
  GetUserInfo(request: Empty): Promise<UserInfo> {
    return http<UserInfo>({
      url: `${AUTH_URL}/userInfo`,
      method: 'GET',
      data: request,
    })
  }
  /** 修改个人中心用户信息 */
  UpdateUserInfo(request: UpdateUserInfoRequest): Promise<Empty> {
    return http<Empty>({
      url: `${AUTH_URL}/userInfo`,
      method: 'PUT',
      data: request,
    })
  }
  /** 手机号授权 */
  PhoneAuth(request: PhoneAuthRequest): Promise<PhoneAuthResponse> {
    return http<PhoneAuthResponse>({
      url: `${AUTH_URL}/userInfo/phone`,
      method: 'PUT',
      data: request,
    })
  }
}

export const defAuthService = new AuthServiceImpl()
