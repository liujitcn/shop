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

/** 获取用户资料请求兼容空请求结构。 */
type GetUserProfileRequestCompat = Empty

/** 更新用户资料请求兼容新旧两种表单包裹方式。 */
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
      authMode: 'none',
      data: request,
    })
  }

  /** 获取已登录用户资料 */
  GetUserProfile(request: GetUserProfileRequestCompat): Promise<UserProfileForm> {
    return http<UserProfileForm>({
      url: `${AUTH_URL}/profile`,
      method: 'GET',
      authMode: 'required',
      data: request,
    })
  }

  /** 修改个人中心用户信息 */
  UpdateUserProfile(request: UpdateUserProfileRequestCompat): Promise<Empty> {
    const userProfile = request.user_profile ?? (request as UserProfileForm)
    return http<Empty>({
      url: `${AUTH_URL}/profile`,
      method: 'PUT',
      authMode: 'required',
      data: userProfile,
    })
  }

  /** 手机号授权 */
  BindUserPhone(request: BindUserPhoneRequest): Promise<BindUserPhoneResponse> {
    return http<BindUserPhoneResponse>({
      url: `${AUTH_URL}/phone`,
      method: 'PUT',
      authMode: 'required',
      data: request,
    })
  }
}

export const defAuthService = new AuthServiceImpl()
