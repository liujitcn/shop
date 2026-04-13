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

const CAPTCHA_URL = '/login'
const AUTH_URL = '/auth'

/** 登录公共服务 */
export class LoginServiceImpl implements LoginService {
  /** 验证码 */
  Captcha(request: Empty): Promise<CaptchaResponse> {
    return http<CaptchaResponse>({
      url: `${CAPTCHA_URL}/captcha`,
      method: 'GET',
      data: request,
      header: { Authorization: 'no-auth' },
    })
  }
  /** 登出 */
  Logout(request: Empty): Promise<Empty> {
    return http<Empty>({
      url: `${AUTH_URL}`,
      method: 'DELETE',
      data: request,
    })
  }
  /** 刷新认证令牌 */
  RefreshToken(request: RefreshTokenRequest): Promise<RefreshTokenResponse> {
    return http<RefreshTokenResponse>({
      url: `${AUTH_URL}/token`,
      method: 'POST',
      data: request,
      header: { Authorization: 'no-auth' },
    })
  }
  /** 登录 */
  Login(request: LoginRequest): Promise<LoginResponse> {
    return http<LoginResponse>({
      url: `${AUTH_URL}`,
      method: 'POST',
      data: request,
      header: { Authorization: 'no-auth' },
    })
  }
}

export const defLoginService = new LoginServiceImpl()
