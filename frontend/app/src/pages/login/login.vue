<script setup lang="ts">
import { useSettingStore, useUserStore } from '@/stores'
import type { WechatLoginRequest } from '@/rpc/app/auth'
import type { LoginRequest } from '@/rpc/base/login'
import { onLoad } from '@dcloudio/uni-app'
import { ref } from 'vue'
import { defLoginService } from '@/api/base/login'
import defaultLogo from '@/static/images/logo_icon.png'
import { homeTabPage } from '@/utils/navigation'

const userStore = useUserStore()
const settingStore = useSettingStore()

// 微信登录表单
const wechatLoginForm = ref<WechatLoginRequest>({
  code: '',
})

// 是否同意协议
const isAgreePrivacy = ref(false)
const isAgreePrivacyShakeY = ref(false)

const toggleAgreePrivacy = () => {
  isAgreePrivacy.value = !isAgreePrivacy.value
}

const triggerAgreePrivacyShake = () => {
  isAgreePrivacyShakeY.value = true
  setTimeout(() => {
    isAgreePrivacyShakeY.value = false
  }, 500)
}

// 打开服务条款
const onOpenServiceProtocol = () => {
  uni.navigateTo({ url: '/pages/login/protocal?type=service' })
}

// 打开隐私协议
const onOpenPrivacyContract = () => {
  uni.navigateTo({ url: '/pages/login/protocal?type=privacy' })
}

// #ifdef MP-WEIXIN
const wxLogin = async () => {
  const isAgreed = await checkedAgreePrivacy()
  if (!isAgreed) {
    return
  }
  const res = await wx.login()
  wechatLoginForm.value.code = res.code
  // 显示确认弹窗
  uni.showModal({
    title: '提示',
    content: '确定要使用微信登录吗？',
    success: (res) => {
      if (res.confirm) {
        userStore.wechatLogin(wechatLoginForm.value).then(() => {
          void loginSuccess()
        })
      }
    },
  })
}
// #endif

// #ifdef H5
const captchaBase64 = ref() // 验证码图片Base64字符串
// 获取验证码
const getCaptcha = () => {
  defLoginService.Captcha({}).then((data) => {
    form.value.captchaId = data.captchaId
    captchaBase64.value = data.captchaBase64
  })
}
// 传统表单登录。
const form = ref<LoginRequest>({
  userName: '',
  password: '',
  captchaId: '',
  captchaCode: '',
})
// 表单提交
const onSubmit = async () => {
  if (!form.value.userName) {
    await uni.showToast({
      icon: 'none',
      title: '请输入用户名或手机号',
    })
    return
  }
  if (!form.value.password) {
    await uni.showToast({
      icon: 'none',
      title: '请输入密码',
    })
    return
  }
  if (!form.value.captchaCode) {
    await uni.showToast({
      icon: 'none',
      title: '请输入验证码',
    })
    return
  }
  const isAgreed = await checkedAgreePrivacy()
  if (!isAgreed) {
    return
  }
  userStore
    .login(form.value)
    .then(() => {
      void loginSuccess()
    })
    .catch(() => {
      form.value.captchaCode = ''
      getCaptcha()
    })
}
// #endif
const loginSuccess = async () => {
  await userStore.getUserProfile()
  // 成功提示
  await uni.showToast({ icon: 'success', title: '登录成功' })
  setTimeout(() => {
    const lastRoute = uni.getStorageSync('lastRoute') || homeTabPage
    if (lastRoute.startsWith(homeTabPage)) {
      uni.setStorageSync('SwitchTabIndex', true)
    }
    uni.removeStorageSync('lastRoute')

    const pagePath = String(lastRoute).split('?')[0] || homeTabPage
    const tabBarPages = new Set([
      homeTabPage,
      '/pages/category/category',
      '/pages/cart/cart',
      '/pages/my/my',
    ])
    if (tabBarPages.has(pagePath)) {
      uni.switchTab({ url: pagePath })
      return
    }
    uni.reLaunch({ url: lastRoute })
  }, 500)
}

// 请先阅读并勾选协议
const checkedAgreePrivacy = async () => {
  if (isAgreePrivacy.value) {
    return true
  }

  triggerAgreePrivacyShake()

  return new Promise<boolean>((resolve) => {
    uni.showModal({
      title: '提示',
      content: '请先阅读并勾选协议内容，点击确定后将自动勾选并继续登录',
      confirmText: '确定',
      cancelText: '取消',
      success: ({ confirm }) => {
        if (confirm) {
          isAgreePrivacy.value = true
          resolve(true)
          return
        }
        resolve(false)
      },
      fail: () => resolve(false),
    })
  })
}

// 获取 code 登录凭证
onLoad(async () => {
  // #ifdef H5
  getCaptcha()
  // #endif
})
</script>

<template>
  <view class="viewport">
    <view class="hero">
      <view class="logo-shell">
        <image :src="settingStore.getData('sysLogo') || defaultLogo" />
      </view>
      <view class="hero-copy">
        <text class="title">欢迎登录</text>
      </view>
    </view>
    <view class="login-panel">
      <view class="login">
        <!-- 网页端表单登录 -->
        <!-- #ifdef H5 -->
        <input
          v-model="form.userName"
          class="input"
          type="text"
          placeholder="请输入用户名/手机号码"
        />
        <input
          v-model="form.password"
          class="input"
          type="text"
          password
          placeholder="请输入密码"
        />
        <view class="captcha-row">
          <input
            v-model="form.captchaCode"
            class="input captcha-input"
            type="text"
            placeholder="请输入验证码"
          />
          <view class="captcha-divider"></view>
          <view class="captcha-trigger" @tap="getCaptcha">
            <image class="captcha-image" :src="captchaBase64" mode="aspectFit" />
          </view>
        </view>
        <button @tap="onSubmit" class="button phone">登录</button>
        <!-- #endif -->

        <!-- 小程序端授权登录 -->
        <!-- #ifdef MP-WEIXIN -->
        <button class="button phone" @tap="wxLogin">
          <text class="icon icon-phone"></text>
          微信一键登录
        </button>
        <!-- #endif -->
      </view>
      <view class="tips" :class="{ animate__shakeY: isAgreePrivacyShakeY }">
        <view class="label" @tap="toggleAgreePrivacy">
          <view class="agree-icon" :class="{ checked: isAgreePrivacy }"></view>
          <text class="desc">我已阅读并同意</text>
          <text class="link" @tap="onOpenServiceProtocol">《服务条款》</text>
          <text class="separator">和</text>
          <text class="link" @tap="onOpenPrivacyContract">《隐私协议》</text>
        </view>
      </view>
    </view>
  </view>
</template>

<style lang="scss">
page {
  height: 100%;
  background: linear-gradient(180deg, #f4fbf8 0%, #ffffff 45%, #ffffff 100%);
}

.viewport {
  display: flex;
  flex-direction: column;
  justify-content: center;
  height: 100%;
  padding: 48rpx 40rpx 56rpx;
}

.hero {
  margin-bottom: 40rpx;
  text-align: center;

  .logo-shell {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 180rpx;
    height: 180rpx;
    margin-bottom: 24rpx;
    border-radius: 44rpx;
    background: rgba(255, 255, 255, 0.92);
    box-shadow: 0 18rpx 44rpx rgba(40, 187, 156, 0.12);
  }

  image {
    width: 120rpx;
    height: 120rpx;
  }

  .hero-copy {
    display: flex;
    flex-direction: column;
  }

  .title {
    font-size: 44rpx;
    font-weight: 600;
    color: #1f2937;
  }
}

.login-panel {
  padding: 36rpx 28rpx 30rpx;
  border-radius: 36rpx;
  background: rgba(255, 255, 255, 0.96);
  box-shadow: 0 24rpx 60rpx rgba(15, 23, 42, 0.08);
}

.login {
  display: flex;
  flex-direction: column;

  .input {
    width: 100%;
    height: 88rpx;
    font-size: 28rpx;
    color: #1f2937;
    border-radius: 24rpx;
    border: 1px solid #e5e7eb;
    background: #f9fbfa;
    padding: 0 30rpx;
    margin-bottom: 20rpx;
    box-sizing: border-box;
  }

  .button {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 100%;
    height: 88rpx;
    margin-top: 12rpx;
    font-size: 30rpx;
    font-weight: 500;
    border-radius: 24rpx;
    color: #fff;
    box-shadow: 0 18rpx 36rpx rgba(40, 187, 156, 0.24);

    &::after {
      border: 0;
    }

    .icon {
      font-size: 40rpx;
      margin-right: 10rpx;
    }
  }

  .phone {
    background: linear-gradient(135deg, #34c8aa 0%, #28bb9c 100%);
  }

  .captcha-row {
    display: flex;
    align-items: center;
    width: 100%;
    height: 88rpx;
    margin-bottom: 20rpx;
    padding: 0 12rpx 0 30rpx;
    border: 1px solid #e5e7eb;
    border-radius: 24rpx;
    background: #f9fbfa;
    box-sizing: border-box;

    .captcha-input {
      flex: 1;
      height: 100%;
      margin-bottom: 0;
      padding: 0;
      border: 0;
      background: transparent;
    }

    .captcha-divider {
      width: 1rpx;
      height: 44rpx;
      background: #e7ebf0;
    }

    .captcha-trigger {
      display: flex;
      align-items: center;
      justify-content: center;
      width: 260rpx;
      height: 72rpx;
      margin-left: 14rpx;
      border-radius: 18rpx;
      background: #fff;
      box-sizing: border-box;
    }

    .captcha-image {
      flex-shrink: 0;
      width: 210rpx;
      height: 60rpx;
    }
  }
}

.tips {
  margin-top: 26rpx;
  font-size: 22rpx;
  color: #999;
  line-height: 1.6;

  .label {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 6rpx;
    white-space: nowrap;
  }

  .separator {
    white-space: nowrap;
  }

  .agree-icon {
    width: 26rpx;
    height: 26rpx;
    margin-right: 2rpx;
    border: 2rpx solid #d1d5db;
    border-radius: 50%;
    background-color: #fff;
    box-sizing: border-box;
  }

  .agree-icon.checked {
    border-color: #28bb9c;
    background-color: #28bb9c;
    box-shadow: inset 0 0 0 6rpx #fff;
  }

  .link {
    white-space: nowrap;
    color: #28bb9c;
  }

  .desc {
    color: #9ca3af;
    white-space: nowrap;
  }
}
</style>
