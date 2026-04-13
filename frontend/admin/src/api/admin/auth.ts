import service from "@/utils/request";
import {
  type AuthService,
  type TreeRouteResponse,
  type UserInfoForm,
  type UserProfileForm,
  type SendPhoneCodeRequest,
  type UserPhoneForm,
  type UserPasswordForm
} from "@/rpc/admin/auth";
import type { StringValues } from "@/rpc/common/types";
import type { Empty } from "@/rpc/google/protobuf/empty";

const AUTH_URL = "/admin/auth";

/** Admin用户登录认证服务 */
export class AuthServiceImpl implements AuthService {
  /** 获取已经登录的用户的数据 */
  GetUserInfo(request: Empty): Promise<UserInfoForm> {
    return service<Empty, UserInfoForm>({
      url: `${AUTH_URL}/user`,
      method: "get",
      params: request
    });
  }
  /** 获取已经登录的用户菜单 */
  GetUserMenu(request: Empty): Promise<TreeRouteResponse> {
    return service<Empty, TreeRouteResponse>({
      url: `${AUTH_URL}/menu`,
      method: "get",
      params: request
    });
  }
  /** 获取已经登录的用户按钮权限 */
  GetUserButton(request: Empty): Promise<StringValues> {
    return service<Empty, StringValues>({
      url: `${AUTH_URL}/button`,
      method: "get",
      params: request
    });
  }
  /** 获取个人中心用户信息 */
  GetUserProfile(request: Empty): Promise<UserProfileForm> {
    return service<Empty, UserProfileForm>({
      url: `${AUTH_URL}/profile`,
      method: "get",
      params: request
    });
  }
  /** 修改个人中心用户信息 */
  UpdateUserProfile(request: UserProfileForm): Promise<Empty> {
    return service<UserProfileForm, Empty>({
      url: `${AUTH_URL}/profile`,
      method: "put",
      data: request
    });
  }
  /** 发送手机号验证码 */
  SendPhoneCode(request: SendPhoneCodeRequest): Promise<Empty> {
    return service<SendPhoneCodeRequest, Empty>({
      url: `${AUTH_URL}/phone/code`,
      method: "post",
      data: request
    });
  }
  /** 修改个人中心手机号 */
  UpdateUserPhone(request: UserPhoneForm): Promise<Empty> {
    return service<UserPhoneForm, Empty>({
      url: `${AUTH_URL}/phone`,
      method: "put",
      data: request
    });
  }
  /** 修改个人中心密码 */
  UpdateUserPassword(request: UserPasswordForm): Promise<Empty> {
    return service<UserPasswordForm, Empty>({
      url: `${AUTH_URL}/password`,
      method: "put",
      data: request
    });
  }
}

export const defAuthService = new AuthServiceImpl();
