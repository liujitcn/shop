import { defineStore } from "pinia";
import { defAuthService } from "@/api/admin/auth";
import { defLoginService } from "@/api/base/login";
import type { LoginRequest } from "@/rpc/base/login";
import type { UserInfo } from "@/rpc/admin/auth";
import { UserState } from "@/stores/interface";
import piniaPersistConfig from "@/stores/helper/persist";
import { useDictStoreHook } from "@/stores/modules/dict";

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
    tokenExpiresAt: 0,
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
    /** 设置令牌过期时间戳 */
    setTokenExpiresAt(tokenExpiresAt: number) {
      this.tokenExpiresAt = tokenExpiresAt;
    },
    /** 设置用户信息 */
    setUserInfo(userInfo: UserState["userInfo"]) {
      this.userInfo = userInfo;
    },
    /** 根据接口返回统一更新令牌信息 */
    updateTokenAuth(accessToken: string, refreshToken: string, tokenType: string, expiresIn?: number) {
      const tokenPrefix = tokenType ? `${tokenType} ` : "";
      const expiresAt = expiresIn ? Date.now() + expiresIn * 1000 : 0;

      this.setToken(`${tokenPrefix}${accessToken}`.trim());
      this.setRefreshToken(refreshToken ?? "");
      this.setTokenType(tokenType ?? "");
      this.setTokenExpiresAt(expiresAt);
    },
    /** 登录 */
    async login(loginRequest: LoginRequest) {
      const data = await defLoginService.Login(loginRequest);
      this.updateTokenAuth(data.accessToken, data.refreshToken ?? "", data.tokenType ?? "", data.expiresIn);
    },
    /** 刷新认证令牌 */
    async refreshAccessToken() {
      if (!this.refreshToken) {
        return Promise.reject(new Error("refresh token 不存在"));
      }

      const data = await defLoginService.RefreshToken({ refreshToken: this.refreshToken });
      this.updateTokenAuth(data.accessToken, data.refreshToken ?? this.refreshToken, data.tokenType ?? "", data.expiresIn);
      return data;
    },
    /** 获取用户信息 */
    async getUserInfo() {
      const data = await defAuthService.GetUserInfo({});
      this.setUserInfo(data);
      return data;
    },
    /** 清理认证数据 */
    clearAuthData() {
      // 清理登录态时同步清空字典缓存，避免切换账号后读到旧字典。
      useDictStoreHook().clearDictionaryCache();
      this.setToken("");
      this.setRefreshToken("");
      this.setTokenType("");
      this.setTokenExpiresAt(0);
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
