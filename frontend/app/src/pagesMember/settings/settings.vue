<script setup lang="ts">
import { onLoad } from '@dcloudio/uni-app'
import { ref } from 'vue'
import { useUserStore } from '@/stores'
import { navigateToLogin } from '@/utils/navigation'

const userStore = useUserStore()
const logoutLoading = ref(false)

onLoad(() => {
  if (!userStore.ensureAuthenticated()) {
    navigateToLogin()
  }
})

const onLogout = async () => {
  if (logoutLoading.value) {
    return
  }
  const result = await uni.showModal({
    content: '是否退出登录？',
    confirmColor: '#2f9f87',
  })
  if (!result.confirm) {
    return
  }

  logoutLoading.value = true
  try {
    await userStore.logout()
    uni.navigateBack()
  } catch {
    await uni.showToast({ icon: 'none', title: '退出登录失败' })
  } finally {
    logoutLoading.value = false
  }
}
</script>

<template>
  <view class="page">
    <view class="list">
      <navigator url="/pagesMember/profile/profile" hover-class="none" class="item">
        <text>个人资料</text>
        <text class="arrow">›</text>
      </navigator>
      <navigator url="/pages/login/protocal?type=service" hover-class="none" class="item">
        <text>服务条款</text>
        <text class="arrow">›</text>
      </navigator>
      <navigator url="/pages/login/protocal?type=privacy" hover-class="none" class="item">
        <text>隐私政策</text>
        <text class="arrow">›</text>
      </navigator>
      <!-- #ifdef MP-WEIXIN -->
      <button hover-class="none" class="item button-item" open-type="openSetting">
        <text>授权管理</text>
        <text class="arrow">›</text>
      </button>
      <button hover-class="none" class="item button-item" open-type="feedback">
        <text>问题反馈</text>
        <text class="arrow">›</text>
      </button>
      <!-- #endif -->
    </view>

    <button class="logout" :disabled="logoutLoading" @tap="onLogout">退出登录</button>
  </view>
</template>

<style lang="scss">
page {
  background: #f5f7f8;
}

.page {
  min-height: 100vh;
  padding: 32rpx;
  box-sizing: border-box;
}

.list {
  overflow: hidden;
  border-radius: 16rpx;
  background: #fff;
}

.item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  padding: 30rpx 28rpx;
  border: 0;
  border-bottom: 1rpx solid #edf1ef;
  box-sizing: border-box;
  background: #fff;
  color: #26332f;
  font-size: 30rpx;
  text-align: left;
}

.item:last-child {
  border-bottom: 0;
}

.button-item::after {
  border: 0;
}

.arrow {
  color: #9eaba6;
  font-size: 44rpx;
  line-height: 1;
}

.logout {
  margin-top: 32rpx;
  border-radius: 16rpx;
  background: #fff;
  color: #d14f4f;
  font-size: 30rpx;
}
</style>
