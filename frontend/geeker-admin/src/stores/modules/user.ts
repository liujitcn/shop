import { defineStore } from "pinia";
import { defAuthService } from "@/api/admin/auth";
import { defLoginService } from "@/api/base/login";
import type { LoginRequest } from "@/rpc/base/login";
import type { UserInfo } from "@/rpc/admin/auth";
import { UserState } from "@/stores/interface";
import piniaPersistConfig from "@/stores/helper/persist";

const defaultUserInfo: UserInfo = {
  userName: "",
  nickName: "",
  phone: "",
  avatar: "",
  roleCode: "",
  roleName: "",
  deptName: ""
};

export const useUserStore = defineStore({
  id: "geeker-user",
  state: (): UserState => ({
    token: "",
    refreshToken: "",
    tokenType: "",
    userInfo: defaultUserInfo
  }),
  getters: {},
  actions: {
    /** 设置访问令牌 */
    setToken(token: string) {
      this.token = token;
    },
    /** 设置刷新令牌 */
    setRefreshToken(refreshToken: string) {
      this.refreshToken = refreshToken;
    },
    /** 设置令牌类型 */
    setTokenType(tokenType: string) {
      this.tokenType = tokenType;
    },
    /** 设置用户信息 */
    setUserInfo(userInfo: UserState["userInfo"]) {
      this.userInfo = userInfo;
    },
    /** 登录 */
    async login(loginRequest: LoginRequest) {
      const data = await defLoginService.Login(loginRequest);
      const tokenPrefix = data.tokenType ? `${data.tokenType} ` : "";
      this.setToken(`${tokenPrefix}${data.accessToken}`.trim());
      this.setRefreshToken(data.refreshToken ?? "");
      this.setTokenType(data.tokenType ?? "");
    },
    /** 获取用户信息 */
    async getUserInfo() {
      const data = await defAuthService.GetUserInfo({});
      this.setUserInfo(data);
      return data;
    },
    /** 清理认证数据 */
    clearAuthData() {
      this.setToken("");
      this.setRefreshToken("");
      this.setTokenType("");
      this.setUserInfo({ ...defaultUserInfo });
    },
    /** 退出登录 */
    async logout() {
      await defLoginService.Logout({});
      this.clearAuthData();
    }
  },
  persist: piniaPersistConfig("geeker-user")
});
