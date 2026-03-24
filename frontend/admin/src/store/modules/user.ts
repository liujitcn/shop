import { store } from "@/store";
import { usePermissionStoreHook } from "@/store/modules/permission";
import { useDictStoreHook } from "@/store/modules/dict";

import { defAuthService } from "@/api/admin/auth";
import { defLoginService } from "@/api/base/login";
import { type UserInfo } from "@/rpc/admin/auth";
import {  LoginRequest } from "@/rpc/base/login";

import {
  setToken,
  setRefreshToken,
  getRefreshToken,
  clearToken,
  setTokenExpiresIn,
} from "@/utils/auth";

export const useUserStore = defineStore("user", () => {
  const userInfo = useStorage<UserInfo>("userInfo", {} as UserInfo);

  /**
   * 登录
   *
   * @param request
   * @returns
   */
  function login(request: LoginRequest) {
    return new Promise<void>((resolve, reject) => {
      // 登录统一走公共登录服务，避免和认证资料接口混用。
      defLoginService
        .Login(request)
        .then((data) => {
          const { tokenType, accessToken, refreshToken, expiresIn } = data;
          setToken(tokenType + " " + accessToken); // Bearer eyJhbGciOiJIUzI1NiJ9.xxx.xxx
          setRefreshToken(refreshToken);
          setTokenExpiresIn(expiresIn);
          resolve();
        })
        .catch((error) => {
          reject(error);
        });
    });
  }

  /**
   * 获取用户信息
   */
  function getUserInfo() {
    return new Promise<UserInfo>((resolve, reject) => {
      defAuthService
        .GetUserInfo({})
        .then((data) => {
          if (!data) {
            reject("Verification failed, please Login again.");
            return;
          }
          Object.assign(userInfo.value, { ...data });
          resolve(data);
        })
        .catch((error) => {
          reject(error);
        });
    });
  }

  /**
   * 登出
   */
  function logout() {
    return new Promise<void>((resolve, reject) => {
      defLoginService
        .Logout({})
        .then(() => {
          clearUserData().then(() => {
            resolve();
          });
        })
        .catch((error) => {
          reject(error);
        });
    });
  }

  /**
   * 刷新 token
   */
  function refreshToken() {
    const refreshToken = getRefreshToken();
    return new Promise<void>((resolve, reject) => {
      defLoginService
        .RefreshToken({
          refreshToken: refreshToken,
        })
        .then((data) => {
          const { tokenType, accessToken, refreshToken, expiresIn } = data;
          setToken(tokenType + " " + accessToken);
          setRefreshToken(refreshToken);
          setTokenExpiresIn(expiresIn);
          resolve();
        })
        .catch((error) => {
          console.log(" refreshToken  刷新失败", error);
          reject(error);
        });
    });
  }

  /**
   * 清理用户数据
   *
   * @returns
   */
  function clearUserData() {
    return new Promise<void>((resolve) => {
      clearToken();
      usePermissionStoreHook().resetRouter();
      useDictStoreHook().clearDictionaryCache();
      resolve();
    });
  }

  /**
   * 判断是否有权限
   *
   * @returns
   */
  function hasPerm(requiredPerms: any) {
    const { permission } = useUserStore().userInfo;
    // 检查权限
    return Array.isArray(requiredPerms)
      ? requiredPerms.some((perm) => permission.includes(perm))
      : permission.includes(requiredPerms);
  }

  return {
    userInfo,
    getUserInfo,
    login,
    logout,
    clearUserData,
    refreshToken,
    hasPerm,
  };
});

/**
 * 用于在组件外部（如在Pinia Store 中）使用 Pinia 提供的 store 实例。
 * 官方文档解释了如何在组件外部使用 Pinia Store：
 * https://pinia.vuejs.org/core-concepts/outside-component-usage.html#using-a-store-outside-of-a-component
 */
export function useUserStoreHook() {
  return useUserStore(store);
}
