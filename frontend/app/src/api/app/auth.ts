import { http } from '@/utils/http'
import {
  type AuthService,
  type BindUserPhoneRequest,
  type BindUserPhoneResponse,
  type UserProfileForm,
  type WechatLoginRequest,
  type WechatLoginResponse,
} from '@/rpc/app/v1/auth'
import type { Empty } from '@/rpc/google/protobuf/empty'

const AUTH_URL = '/v1/app/auth'

type GetUserProfileRequestCompat = Empty

type UpdateUserProfileRequestCompat = Partial<UserProfileForm> & {
  user_profile?: UserProfileForm
}

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
  GetUserProfile(request: GetUserProfileRequestCompat): Promise<UserProfileForm> {
    return http<UserProfileForm>({
      url: `${AUTH_URL}/profile`,
      method: 'GET',
      data: request,
    })
  }

  /** 修改个人中心用户信息 */
  UpdateUserProfile(request: UpdateUserProfileRequestCompat): Promise<Empty> {
    const userProfile = request.user_profile ?? (request as UserProfileForm)
    return http<Empty>({
      url: `${AUTH_URL}/profile`,
      method: 'PUT',
      data: userProfile,
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
