import service from "@/utils/request";
import type {
  CaptchaResponse,
  LoginRequest,
  LoginResponse,
  RefreshTokenRequest,
  RefreshTokenResponse,
  LoginService
} from "@/rpc/base/login";
import type { Empty } from "@/rpc/google/protobuf/empty";

const CAPTCHA_URL = "/login";
const AUTH_URL = "/auth";

/** 登录公共服务 */
export class LoginServiceImpl implements LoginService {
  /** 验证码 */
  Captcha(request: Empty): Promise<CaptchaResponse> {
    return service<Empty, CaptchaResponse>({
      url: `${CAPTCHA_URL}/captcha`,
      method: "get",
      params: request
    });
  }
  /** 登录 */
  Login(request: LoginRequest): Promise<LoginResponse> {
    return service<LoginRequest, LoginResponse>({
      url: `${AUTH_URL}`,
      method: "post",
      data: request
    });
  }
  /** 登出 */
  Logout(request: Empty): Promise<Empty> {
    return service<Empty, Empty>({
      url: `${AUTH_URL}`,
      method: "delete",
      data: request
    });
  }
  /** 刷新认证令牌 */
  RefreshToken(request: RefreshTokenRequest): Promise<RefreshTokenResponse> {
    return service<RefreshTokenRequest, RefreshTokenResponse>({
      url: `${AUTH_URL}/token`,
      method: "post",
      data: request
    });
  }
}

export const defLoginService = new LoginServiceImpl();
