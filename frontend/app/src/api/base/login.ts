import { http } from '@/utils/http'
import type {
  CaptchaResponse,
  LoginRequest,
  LoginResponse,
  RefreshTokenRequest,
  RefreshTokenResponse,
  LoginService,
} from '@/rpc/base/login'
import type { Empty } from '@/rpc/google/protobuf/empty'

const LOGIN_URL = '/login'

/** 登录公共服务 */
export class LoginServiceImpl implements LoginService {
  /** 验证码 */
  Captcha(request: Empty): Promise<CaptchaResponse> {
    return http<CaptchaResponse>({
      url: `${LOGIN_URL}/captcha`,
      method: 'GET',
      data: request,
      header: { Authorization: 'no-auth' },
    })
  }
  /** 登出 */
  Logout(request: Empty): Promise<Empty> {
    return http<Empty>({
      url: `${LOGIN_URL}/logout`,
      method: 'DELETE',
      data: request,
    })
  }
  /** 刷新认证令牌 */
  RefreshToken(request: RefreshTokenRequest): Promise<RefreshTokenResponse> {
    return http<RefreshTokenResponse>({
      url: `${LOGIN_URL}/refreshToken`,
      method: 'POST',
      data: request,
      header: { Authorization: 'no-auth' },
    })
  }
  /** 登录 */
  Login(request: LoginRequest): Promise<LoginResponse> {
    return http<LoginResponse>({
      url: `${LOGIN_URL}`,
      method: 'POST',
      data: request,
      header: { Authorization: 'no-auth' },
    })
  }
}

export const defLoginService = new LoginServiceImpl()
