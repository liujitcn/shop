/**
 * 添加拦截器:
 *   拦截 request 请求
 *   拦截 uploadFile 文件上传
 *
 */

import { useUserStore } from '@/stores'
import { getToken, getTokenExpiresIn } from '@/utils/auth'

const apiBasePath = import.meta.env.VITE_APP_BASE_API || '/api'
const apiTargetUrl = import.meta.env.VITE_APP_API_URL || ''
const baseURL = `${apiTargetUrl}${apiBasePath}`

// 添加拦截器
const httpInterceptor = {
  // 拦截前触发
  invoke(options: UniApp.RequestOptions) {
    // 1. 非 http 开头需拼接地址
    if (!options.url.startsWith('http')) {
      options.url = baseURL + options.url
    }
    // 2. 请求超时, 默认 60s
    options.timeout = 10000
    // 3. 添加小程序端请求头标识
    options.header = {
      ...options.header,
      'source-client': 'miniapp',
    }
    // 4. 添加 token 请求头标识
    const accessToken = getToken()
    if (options.header.Authorization !== 'no-auth' && accessToken) {
      options.header.Authorization = accessToken
    } else {
      delete options.header.Authorization
    }
  },
}
uni.addInterceptor('request', httpInterceptor)
uni.addInterceptor('uploadFile', httpInterceptor)

/**
 * 请求函数
 * @param  UniApp.RequestOptions
 * @returns Promise
 *  1. 返回 Promise 对象
 *  2. 获取数据成功
 *    2.1 提取核心数据 res.data
 *    2.2 添加类型，支持泛型
 *  3. 获取数据失败
 *    3.1 401/403错误 -> 清理用户信息，跳转到登录页
 *    3.2 其他错误 -> 根据后端错误信息轻提示
 *    3.3 网络错误 -> 提示用户换网络
 */
type Data = {
  code: string
  message: string
  reason: string
}

// 2.2 添加类型，支持泛型
export const http = <T>(options: UniApp.RequestOptions) => {
  // 1. 返回 Promise 对象
  return new Promise<T>((resolve, reject) => {
    const sendRequest = async () => {
      try {
        const requestOptions = { ...options, header: { ...options.header } }
        const isRefreshTokenRequest = requestOptions.url.endsWith('/login/refreshToken')
        if (requestOptions.header?.Authorization !== 'no-auth' && !isRefreshTokenRequest) {
          await ensureValidToken()
          const accessToken = getToken()
          if (accessToken) {
            requestOptions.header = {
              ...requestOptions.header,
              Authorization: accessToken,
            }
          }
        }
        uni.request({
          ...requestOptions,
          // 响应成功
          success(res) {
            // 状态码 2xx， axios 就是这样设计的
            if (res.statusCode >= 200 && res.statusCode < 300) {
              // 2.1 提取核心数据 res.data
              resolve(res.data as T)
            } else if (res.statusCode === 401 || res.statusCode === 403) {
              // 401/403 错误 -> 清理用户信息，跳转到登录页
              void promptRelogin()
              reject(res)
            } else {
              // 其他错误 -> 根据后端错误信息轻提示
              void uni.showToast({
                icon: 'none',
                title: (res.data as Data).message || '请求错误',
              })
              reject(res)
            }
          },
          // 响应失败
          fail(err) {
            void uni.showToast({
              icon: 'none',
              title: '网络错误，换个网络试试',
            })
            reject(err)
          },
        })
      } catch (error) {
        reject(error)
      }
    }

    void sendRequest()
  })
}

// 刷新 Token 的锁
let isRefreshing = false
let refreshTokenPromise: Promise<void> | null = null
let isPromptingRelogin = false

function shouldRefreshToken() {
  const now = new Date().getTime()
  const expiresIn = getTokenExpiresIn()
  const remain = expiresIn - now
  return Boolean(expiresIn && remain <= 5 * 60 * 1000)
}

async function ensureValidToken() {
  if (!shouldRefreshToken()) {
    return
  }
  await handleTokenRefresh()
}

// 刷新 Token 处理
function handleTokenRefresh() {
  if (refreshTokenPromise) {
    return refreshTokenPromise
  }
  const userStore = useUserStore()
  isRefreshing = true
  refreshTokenPromise = userStore
    .refreshToken()
    .catch(async (error) => {
      await promptRelogin()
      throw error
    })
    .finally(() => {
      isRefreshing = false
      refreshTokenPromise = null
    })
  return refreshTokenPromise
}

async function promptRelogin() {
  if (isPromptingRelogin) {
    return
  }
  isPromptingRelogin = true
  await new Promise<void>((resolve) => {
    uni.showModal({
      title: '提示',
      content: '当前页面已失效，请重新登录',
      success: (res) => {
        if (res.confirm) {
          void clearUserData()
        }
        resolve()
      },
      complete: () => {
        isPromptingRelogin = false
      },
    })
  })
}

function clearUserData() {
  const userStore = useUserStore()
  userStore.clearUserData().then(() => {
    // 获取当前页面信息（兼容多平台）
    const pages = getCurrentPages()
    const currentPage = pages[pages.length - 1]

    // 1. 获取页面参数（兼容方案）
    let params: Record<string, string> = {}
    const miniPage = currentPage as { options?: Record<string, string> }
    const routePage = currentPage as { $vm?: { $route?: { query?: Record<string, string> } } }
    // 微信小程序
    // #ifdef MP-WEIXIN
    params = miniPage.options || {}
    // #endif

    // H5和APP
    // #ifdef H5 || APP-PLUS
    if (routePage.$vm && routePage.$vm.$route) {
      params = routePage.$vm.$route.query || {}
    }
    // #endif
    const query = Object.keys(params)
      .map((key) => `${key}=${encodeURIComponent(params[key])}`)
      .join('&')
    const url = query ? `${currentPage.route}?${query}` : currentPage.route

    // 存储路由信息
    uni.setStorageSync('lastRoute', '/' + url)

    uni.reLaunch({ url: '/pages/login/login' })
  })
}
