import type { UserInfo, WxLoginRequest } from '@/rpc/app/auth'
import type { LoginRequest } from '@/rpc/base/login'
import { defAuthService } from '@/api/app/auth'
import { bindRecommendAnonymousActor } from '@/api/app/recommendActor'
import { defLoginService } from '@/api/base/login'
import { defineStore } from 'pinia'
import { ref } from 'vue'
import {
  setToken,
  setRefreshToken,
  getRefreshToken,
  clearToken,
  setTokenExpiresIn,
} from '@/utils/auth'

// 定义 Store
export const useUserStore = defineStore(
  'user',
  () => {
    // 会员信息
    const userInfo = ref<UserInfo>()

    /**
     * 登录
     *
     * @param request
     * @returns
     */
    function login(request: LoginRequest) {
      return new Promise<void>((resolve, reject) => {
        defLoginService
          .Login(request)
          .then((data) => {
            const { tokenType, accessToken, refreshToken, expiresIn } = data
            setToken(tokenType + ' ' + accessToken) // Bearer eyJhbGciOiJIUzI1NiJ9.xxx.xxx
            setRefreshToken(refreshToken)
            setTokenExpiresIn(expiresIn)
            bindRecommendAnonymousActor()
              .catch((error) => {
                console.warn('bindRecommendAnonymousActor failed', error)
              })
              .finally(() => {
                resolve()
              })
          })
          .catch((error) => {
            reject(error)
          })
      })
    }

    /**
     * 微信登录
     *
     * @param request
     * @returns
     */
    function wxLogin(request: WxLoginRequest) {
      return new Promise<void>((resolve, reject) => {
        defAuthService
          .WxLogin(request)
          .then((data) => {
            const { tokenType, accessToken, refreshToken, expiresIn } = data
            setToken(tokenType + ' ' + accessToken)
            setRefreshToken(refreshToken)
            setTokenExpiresIn(expiresIn)
            bindRecommendAnonymousActor()
              .catch((error) => {
                console.warn('bindRecommendAnonymousActor failed', error)
              })
              .finally(() => {
                resolve()
              })
          })
          .catch((error) => {
            reject(error)
          })
      })
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
              reject('Verification failed, please Login again.')
              return
            }
            userInfo.value = data
            resolve(data)
          })
          .catch((error) => {
            reject(error)
          })
      })
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
              resolve()
            })
          })
          .catch((error) => {
            reject(error)
          })
      })
    }

    /**
     * 刷新 token
     */
    function refreshToken() {
      const refreshToken = getRefreshToken()
      return new Promise<void>((resolve, reject) => {
        defLoginService
          .RefreshToken({
            refreshToken: refreshToken,
          })
          .then((data) => {
            const { tokenType, accessToken, refreshToken, expiresIn } = data
            setToken(tokenType + ' ' + accessToken)
            setRefreshToken(refreshToken)
            setTokenExpiresIn(expiresIn)
            resolve()
          })
          .catch((error) => {
            console.log(' refreshToken  刷新失败', error)
            reject(error)
          })
      })
    }

    /**
     * 清理用户数据
     *
     * @returns
     */
    function clearUserData() {
      return new Promise<void>((resolve) => {
        clearToken()
        userInfo.value = undefined
        resolve()
      })
    }
    return {
      userInfo,
      getUserInfo,
      login,
      wxLogin,
      logout,
      clearUserData,
      refreshToken,
    }
  },
  {
    // 网页端配置
    // persist: true,
    // 小程序端配置
    persist: {
      storage: {
        getItem(key) {
          return uni.getStorageSync(key)
        },
        setItem(key, value) {
          uni.setStorageSync(key, value)
        },
      },
    },
  },
)
