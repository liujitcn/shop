import axios, { type InternalAxiosRequestConfig, type AxiosResponse } from "axios";
import qs from "qs";
import { ElMessage, ElMessageBox } from "element-plus";
import router from "@/routers";
import { LOGIN_URL } from "@/config";
import pinia from "@/stores";
import { useUserStore } from "@/stores/modules/user";

const apiBasePath = import.meta.env.VITE_APP_BASE_API || "";
const apiTargetUrl = import.meta.env.VITE_API_URL || import.meta.env.VITE_APP_API_URL || "";
const baseURL = `${apiTargetUrl}${apiBasePath}`;
const SESSION_URL = "/v1/base/session";
const TOKEN_URL = "/v1/base/token";
const CAPTCHA_URL = "/v1/base/captcha";
const LEGACY_AUTH_URL = "/auth";
const LEGACY_REFRESH_TOKEN_URL = `${LEGACY_AUTH_URL}/token`;
const LEGACY_CAPTCHA_URL = "/login/captcha";
// 认证公共接口不携带旧 token，同时保留旧路径兼容迁移期间的灰度访问。
const NO_AUTH_URL_SET = new Set([
  SESSION_URL,
  TOKEN_URL,
  CAPTCHA_URL,
  LEGACY_AUTH_URL,
  LEGACY_CAPTCHA_URL,
  LEGACY_REFRESH_TOKEN_URL
]);
const AUTH_EXPIRED_EXCLUDED_URL_SET = new Set([SESSION_URL, CAPTCHA_URL, LEGACY_AUTH_URL, LEGACY_CAPTCHA_URL]);

// 创建 axios 实例
const service = axios.create({
  baseURL: baseURL,
  timeout: 50000,
  headers: { "Content-Type": "application/json;charset=utf-8" },
  paramsSerializer: params => qs.stringify(params)
});

// 刷新令牌请求使用独立实例，避免与主请求拦截器互相递归。
const refreshService = axios.create({
  baseURL: baseURL,
  timeout: 50000,
  headers: { "Content-Type": "application/json;charset=utf-8" }
});

/** 获取用户状态仓库 */
function getUserStore() {
  return useUserStore(pinia);
}

/** 判断当前请求是否需要跳过认证头 */
function shouldSkipAuth(config: InternalAxiosRequestConfig) {
  if (config.headers.Authorization === "no-auth") return true;

  const requestUrl = config.url ?? "";
  return NO_AUTH_URL_SET.has(requestUrl);
}

/** 判断当前请求是否不应触发登录失效弹窗 */
function shouldSkipAuthExpiredPrompt(config?: InternalAxiosRequestConfig) {
  if (!config) return false;

  const requestUrl = config.url ?? "";
  return AUTH_EXPIRED_EXCLUDED_URL_SET.has(requestUrl);
}

/** 判断当前请求是否需要静默掉错误提示。 */
function shouldSkipErrorMessage(config?: InternalAxiosRequestConfig) {
  if (!config) return false;

  const requestUrl = config.url ?? "";
  const requestMethod = String(config.method ?? "").toLowerCase();
  return (requestUrl === SESSION_URL || requestUrl === LEGACY_AUTH_URL) && requestMethod === "delete";
}

/** 读取访问令牌过期时间 */
function getTokenExpiresAt() {
  return getUserStore().tokenExpiresAt;
}

// 防止并发 401/403 重复弹出认证失效确认框。
let isHandlingAuthExpired = false;

/** 统一处理认证失效 */
function handleAuthExpired() {
  if (isHandlingAuthExpired) {
    return;
  }

  isHandlingAuthExpired = true;
  ElMessageBox.confirm("登录状态已失效，请重新登录", "提示", {
    confirmButtonText: "重新登录",
    cancelButtonText: "取消",
    type: "warning",
    closeOnClickModal: false,
    closeOnPressEscape: false
  })
    .then(() => {
      const userStore = getUserStore();
      const currentRoute = router.currentRoute.value;
      const redirect = currentRoute.path === LOGIN_URL ? undefined : currentRoute.fullPath;
      userStore.clearAuthData();
      return router.replace({
        path: LOGIN_URL,
        query: redirect ? { redirect } : undefined
      });
    })
    .finally(() => {
      isHandlingAuthExpired = false;
    });
}

// 请求拦截器
service.interceptors.request.use(
  async (config: InternalAxiosRequestConfig) => {
    const now = new Date().getTime();
    const expiresAt = getTokenExpiresAt();
    const remainingTime = expiresAt - now;
    if (!shouldSkipAuth(config) && expiresAt && remainingTime <= 5 * 60 * 1000) {
      await handleTokenRefresh();
    }

    const accessToken = getUserStore().token;
    // 登录、验证码、刷新令牌接口不携带旧 token，避免请求头污染。
    if (!shouldSkipAuth(config) && accessToken) {
      config.headers.Authorization = accessToken;
    } else {
      delete config.headers.Authorization;
    }
    return config;
  },
  error => Promise.reject(error)
);

// 响应拦截器
service.interceptors.response.use(
  (response: AxiosResponse) => {
    // 如果响应是二进制流，则直接返回，用于下载文件、Excel 导出等
    if (response.config.responseType === "blob") {
      return response;
    }

    const { code, message, reason, metadata } = response.data;
    if (code === undefined || message === undefined || reason === undefined || metadata === undefined) {
      return response.data;
    }

    ElMessage.error(message || "系统出错");
    return Promise.reject(new Error(message || "Error"));
  },
  (error: any) => {
    const status = error.response?.status;
    const code = error.response?.data?.code;
    const message = error.response?.data?.message;
    const requestConfig = error.config as InternalAxiosRequestConfig | undefined;

    // 登录与验证码接口上的 401/403 属于当前请求失败，不应触发“登录失效”流程。
    if ((status === 401 || status === 403 || code === 401 || code === 403) && !shouldSkipAuthExpiredPrompt(requestConfig)) {
      handleAuthExpired();
    } else if (error.response?.data) {
      if (!shouldSkipErrorMessage(requestConfig)) {
        if (message) {
          ElMessage.error(message);
        } else {
          ElMessage.error("系统出错");
        }
      }
    } else if (!shouldSkipErrorMessage(requestConfig)) {
      ElMessage.error(error.message || "系统出错");
    }
    return Promise.reject(error.message);
  }
);

export default service;

// 刷新 Token 的锁
let isRefreshing = false;
let refreshPromise: Promise<void> | null = null;

/** 刷新 Token 处理 */
async function handleTokenRefresh() {
  if (!isRefreshing) {
    isRefreshing = true;
    refreshPromise = refreshAccessToken()
      .catch(error => {
        console.log("token 刷新失败", error);
        handleAuthExpired();
        throw error;
      })
      .finally(() => {
        isRefreshing = false;
        refreshPromise = null;
      });
  }

  if (refreshPromise) {
    await refreshPromise;
  }
}

/** 调用刷新令牌接口并回写最新认证信息 */
async function refreshAccessToken() {
  const userStore = getUserStore();
  if (!userStore.refreshToken) {
    return Promise.reject(new Error("refresh token 不存在"));
  }

  const response = await refreshService.post(
    TOKEN_URL,
    { refresh_token: userStore.refreshToken },
    { headers: { Authorization: "no-auth" } }
  );
  const data = response.data;
  userStore.updateTokenAuth(
    data.access_token,
    data.refresh_token ?? userStore.refreshToken,
    data.token_type ?? "",
    data.expires_in
  );
}
