import axios, { type InternalAxiosRequestConfig, type AxiosResponse } from "axios";
import qs from "qs";
import { useUserStoreHook } from "@/store/modules/user";
import { getToken, getTokenExpiresIn } from "@/utils/auth";

const apiBasePath = import.meta.env.VITE_APP_BASE_API;
const apiTargetUrl = import.meta.env.VITE_APP_API_URL;
const baseURL = `${apiTargetUrl}${apiBasePath}`;

// 创建 axios 实例
const service = axios.create({
  baseURL: baseURL,
  timeout: 50000,
  headers: { "Content-Type": "application/json;charset=utf-8" },
  paramsSerializer: (params) => qs.stringify(params),
});

// 请求拦截器
service.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    const now = new Date().getTime();
    const expiresIn = getTokenExpiresIn();
    const t = expiresIn - now;
    if (expiresIn && t <= 5 * 50 * 1000) {
      handleTokenRefresh();
    }

    const accessToken = getToken();
    // 如果 Authorization 设置为 no-auth，则不携带 Token，用于登录、刷新 Token 等接口
    if (config.headers.Authorization !== "no-auth" && accessToken) {
      config.headers.Authorization = accessToken;
    } else {
      delete config.headers.Authorization;
    }
    return config;
  },
  (error) => Promise.reject(error)
);

// 响应拦截器
service.interceptors.response.use(
  (response: AxiosResponse) => {
    // 如果响应是二进制流，则直接返回，用于下载文件、Excel 导出等
    if (response.config.responseType === "blob") {
      return response;
    }

    const { code, message, reason, metadata } = response.data;
    if (
      code === undefined ||
      message === undefined ||
      reason === undefined ||
      metadata === undefined
    ) {
      return response.data;
    }

    ElMessage.error(message || "系统出错");
    return Promise.reject(new Error(message || "Error"));
  },
  (error: any) => {
    if (error.response.data) {
      const { code, message } = error.response.data;
      // token 过期,重新登录
      if (code === 401 || code === 403) {
        ElMessageBox.confirm("当前页面已失效，请重新登录", "提示", {
          confirmButtonText: "确定",
          cancelButtonText: "取消",
          type: "warning",
        }).then(() => {
          const userStore = useUserStoreHook();
          userStore.clearUserData().then(() => {
            location.reload();
          });
        });
      } else {
        ElMessage.error(message || "系统出错");
      }
    }
    return Promise.reject(error.message);
  }
);

export default service;

// 刷新 Token 的锁
let isRefreshing = false;

// 刷新 Token 处理
function handleTokenRefresh() {
  if (!isRefreshing) {
    isRefreshing = true;
    // 刷新 Token
    useUserStoreHook()
      .refreshToken()
      .then(() => {
        console.log("token 刷新成功");
      })
      .catch((error) => {
        console.log("token 刷新失败" + error);
        ElMessageBox.confirm("当前页面已失效，请重新登录", "提示", {
          confirmButtonText: "确定",
          cancelButtonText: "取消",
          type: "warning",
        }).then(() => {
          const userStore = useUserStoreHook();
          userStore.clearUserData().then(() => {
            location.reload();
          });
        });
      })
      .finally(() => {
        isRefreshing = false;
      });
  }
}
