<script setup lang="ts">
import { onShow } from '@dcloudio/uni-app'
import { computed } from 'vue'
import { useUserStore } from '@/stores'
import { navigateToLogin } from '@/utils/navigation'

const userStore = useUserStore()
const displayName = computed(
  () => userStore.userInfo?.nick_name || userStore.userInfo?.user_name || '未登录',
)

onShow(() => {
  if (userStore.isAuthenticated()) {
    void userStore.getUserProfile().catch(() => undefined)
  }
})

const openPage = (url: string) => {
  if (!userStore.ensureAuthenticated()) {
    navigateToLogin(url)
    return
  }
  uni.navigateTo({ url })
}
</script>

<template>
  <view class="page">
    <view class="profile">
      <image class="avatar" src="@/static/images/avatar.png" mode="aspectFill" />
      <view class="profile-copy">
        <text class="name">{{ displayName }}</text>
        <text class="status">{{
          userStore.isAuthenticated() ? '已登录' : '登录后使用更多功能'
        }}</text>
      </view>
      <button v-if="!userStore.isAuthenticated()" class="login-button" @tap="navigateToLogin">
        登录
      </button>
    </view>

    <view class="list">
      <view class="item" @tap="openPage('/pagesMember/profile/profile')">
        <text>个人资料</text>
        <text class="arrow">›</text>
      </view>
      <view class="item" @tap="openPage('/pagesMember/settings/settings')">
        <text>应用设置</text>
        <text class="arrow">›</text>
      </view>
      <view class="item" @tap="openPage('/pagesMember/ai/index')">
        <text>AI 助手</text>
        <text class="arrow">›</text>
      </view>
    </view>
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

.profile {
  display: flex;
  align-items: center;
  padding: 36rpx 28rpx;
  border-radius: 16rpx;
  background: #fff;
}

.avatar {
  width: 112rpx;
  height: 112rpx;
  border-radius: 50%;
  background: #eaf1ef;
}

.profile-copy {
  flex: 1;
  margin-left: 24rpx;
}

.name,
.status {
  display: block;
}

.name {
  color: #1f2c28;
  font-size: 34rpx;
  font-weight: 600;
}

.status {
  margin-top: 10rpx;
  color: #8a9691;
  font-size: 24rpx;
}

.login-button {
  margin: 0;
  padding: 0 24rpx;
  border: 0;
  border-radius: 32rpx;
  background: #2f9f87;
  color: #fff;
  font-size: 24rpx;
  line-height: 64rpx;
}

.login-button::after {
  border: 0;
}

.list {
  margin-top: 28rpx;
  overflow: hidden;
  border-radius: 16rpx;
  background: #fff;
}

.item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 30rpx 28rpx;
  border-bottom: 1rpx solid #edf1ef;
  color: #26332f;
  font-size: 30rpx;
}

.item:last-child {
  border-bottom: 0;
}

.arrow {
  color: #9eaba6;
  font-size: 44rpx;
  line-height: 1;
}
</style>
