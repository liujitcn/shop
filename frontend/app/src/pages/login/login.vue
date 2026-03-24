<script setup lang="ts">
import type { LoginRequest } from '@/rpc/base/login'
import { useUserStore } from '@/stores'
import type { WxLoginRequest } from '@/rpc/app/auth'
import { onLoad } from '@dcloudio/uni-app'
import { ref } from 'vue'
import { defLoginService } from '@/api/base/login'
import defaultLogo from '@/static/images/logo_icon.png'
const userStore = useUserStore()
import { useSettingStore } from '@/stores'

const settingStore = useSettingStore()

// 传统表单登录。
const wxLoginForm = ref<WxLoginRequest>({
  code: '',
})

// 是否同意协议
const isAgreePrivacy = ref(false)
const toggleAgreePrivacy = () => {
  isAgreePrivacy.value = !isAgreePrivacy.value
}

// 打开协议详情页
const openProtocolPage = (type: 'service' | 'privacy') => {
  uni.navigateTo({
    url: `/pages/login/protocal?type=${type}`,
  })
}

// 打开服务条款
const onOpenServiceProtocol = () => {
  openProtocolPage('service')
}

// 打开隐私协议
const onOpenPrivacyContract = () => {
  openProtocolPage('privacy')
}

// #ifdef MP-WEIXIN
const wxLogin = async () => {
  await checkedAgreePrivacy()
  const res = await wx.login()
  wxLoginForm.value.code = res.code
  // 显示确认弹窗
  uni.showModal({
    title: '提示',
    content: '确定要使用微信登录吗？',
    success: (res) => {
      if (res.confirm) {
        userStore.wxLogin(wxLoginForm.value).then(() => {
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
      title: '请输入手机号',
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
  await checkedAgreePrivacy()
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
  await userStore.getUserInfo()
  // 成功提示
  await uni.showToast({ icon: 'success', title: '登录成功' })
  setTimeout(() => {
    const lastRoute = uni.getStorageSync('lastRoute') || '/pages/index/index'
    if (lastRoute.startsWith('/pages/index/index')) {
      uni.setStorageSync('SwitchTabIndex', true)
    }
    uni.removeStorageSync('lastRoute')
    const tab = [
      '/pages/index/index',
      '/pages/category/category',
      '/pages/cart/cart',
      '/pages/my/my',
    ]
    if (tab.includes(lastRoute)) {
      uni.switchTab({ url: lastRoute })
    } else {
      uni.reLaunch({ url: lastRoute })
    }
  }, 500)
}

// 请先阅读并勾选协议
const isAgreePrivacyShakeY = ref(false)
const checkedAgreePrivacy = async () => {
  if (!isAgreePrivacy.value) {
    uni.showToast({
      icon: 'none',
      title: '请先阅读并勾选协议内容',
    })
    // 震动提示
    isAgreePrivacyShakeY.value = true
    setTimeout(() => {
      isAgreePrivacyShakeY.value = false
    }, 500)
    // 返回错误
    return Promise.reject(new Error('请先阅读并勾选协议内容'))
  }
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
    <view class="logo">
      <image :src="settingStore.getData('sysLogo') || defaultLogo" />
    </view>
    <view class="login">
      <!-- 网页端表单登录 -->
      <!-- #ifdef H5 -->
      <input
        v-model="form.userName"
        class="input"
        type="text"
        placeholder="请输入用户名/手机号码"
      />
      <input v-model="form.password" class="input" type="text" password placeholder="请输入密码" />
      <view class="captcha-row">
        <input v-model="form.captchaCode" class="input" type="text" placeholder="请输入验证码" />
        <image class="captcha-image" :src="captchaBase64" mode="widthFix" @tap="getCaptcha" />
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
        <text>登录或注册即表示你已阅读并同意</text>
      </view>
      <text class="link" @tap="onOpenServiceProtocol">《服务条款》</text>
      和
      <text class="link" @tap="onOpenPrivacyContract">《隐私协议》</text>
    </view>
  </view>
</template>

<style lang="scss">
page {
  height: 100%;
}

.viewport {
  display: flex;
  flex-direction: column;
  height: 100%;
  padding: 20rpx 40rpx;
}

.logo {
  flex: 1;
  text-align: center;
  image {
    width: 220rpx;
    height: 220rpx;
    margin-top: 15vh;
  }
}

.login {
  display: flex;
  flex-direction: column;
  height: 60vh;
  padding: 40rpx 20rpx 20rpx;

  .input {
    width: 100%;
    height: 80rpx;
    font-size: 28rpx;
    border-radius: 72rpx;
    border: 1px solid #ddd;
    padding-left: 30rpx;
    margin-bottom: 20rpx;
  }

  .button {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 100%;
    height: 80rpx;
    font-size: 28rpx;
    border-radius: 72rpx;
    color: #fff;
    .icon {
      font-size: 40rpx;
      margin-right: 6rpx;
    }
  }

  .phone {
    background-color: #28bb9c;
  }

  // 新增验证码相关样式
  .captcha-row {
    display: flex;
    gap: 20rpx;
    width: 100%;
    margin-bottom: 20rpx;

    .input {
      flex: 1;
    }

    .captcha-image {
      flex-shrink: 0;
      width: 240rpx;
      height: 80rpx;
      border-radius: 8rpx;
      border: 1rpx solid #ddd;
      cursor: pointer;
    }
  }
}
.tips {
  position: absolute;
  bottom: 80rpx;
  left: 20rpx;
  right: 20rpx;
  font-size: 22rpx;
  color: #999;
  text-align: center;

  .label {
    display: inline-flex;
    align-items: center;
    margin-right: 8rpx;
  }

  .agree-icon {
    width: 28rpx;
    height: 28rpx;
    margin-right: 10rpx;
    border: 2rpx solid #c9c9c9;
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
    display: inline;
    color: #28bb9c;
  }
}
</style>
