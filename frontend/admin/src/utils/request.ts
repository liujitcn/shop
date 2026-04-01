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
const REFRESH_TOKEN_URL = "/login/refreshToken";
const NO_AUTH_URL_SET = new Set(["/login", "/login/captcha", REFRESH_TOKEN_URL]);

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

/** 读取访问令牌过期时间 */
function getTokenExpiresAt() {
  return getUserStore().tokenExpiresAt;
}

/** 统一处理认证失效 */
function handleAuthExpired() {
  ElMessageBox.confirm("当前页面已失效，请重新登录", "提示", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning"
  }).then(() => {
    const userStore = getUserStore();
    const currentRoute = router.currentRoute.value;
    const redirect = currentRoute.path === LOGIN_URL ? undefined : currentRoute.fullPath;
    userStore.clearAuthData();
    router.replace({
      path: LOGIN_URL,
      query: redirect ? { redirect } : undefined
    });
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
    if (error.response?.data) {
      const { code, message } = error.response.data;
      // token 过期,重新登录
      if (code === 401 || code === 403) {
        handleAuthExpired();
      } else {
        ElMessage.error(message || "系统出错");
      }
    } else {
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
    REFRESH_TOKEN_URL,
    { refreshToken: userStore.refreshToken },
    { headers: { Authorization: "no-auth" } }
  );
  const data = response.data;
  userStore.updateTokenAuth(data.accessToken, data.refreshToken ?? userStore.refreshToken, data.tokenType ?? "", data.expiresIn);
}
