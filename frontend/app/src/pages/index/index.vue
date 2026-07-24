<script setup lang="ts">
import { onShow } from '@dcloudio/uni-app'
import { computed } from 'vue'
import { useUserStore } from '@/stores'
import { navigateToLogin } from '@/utils/navigation'

const userStore = useUserStore()
const displayName = computed(() => userStore.userInfo?.nick_name || userStore.userInfo?.user_name)

onShow(() => {
  if (userStore.isAuthenticated() && !userStore.userInfo) {
    void userStore.getUserProfile().catch(() => undefined)
  }
})

const openAssistant = () => {
  if (!userStore.ensureAuthenticated()) {
    navigateToLogin('/pagesMember/ai/index')
    return
  }
  uni.navigateTo({ url: '/pagesMember/ai/index' })
}

const openAccount = () => {
  if (!userStore.isAuthenticated()) {
    navigateToLogin()
    return
  }
  uni.navigateTo({ url: '/pages/my/my' })
}
</script>

<template>
  <view class="page">
    <view class="hero">
      <text class="eyebrow">APPLICATION SHELL</text>
      <text class="title">基础应用</text>
      <text class="subtitle">登录、账户与通用能力已就绪。</text>
    </view>

    <view class="section">
      <view class="section-title">快捷入口</view>
      <view class="entry-list">
        <view class="entry" @tap="openAssistant">
          <view>
            <text class="entry-title">AI 助手</text>
            <text class="entry-desc">使用基础 AI 会话能力</text>
          </view>
          <text class="entry-arrow">›</text>
        </view>
        <view class="entry" @tap="openAccount">
          <view>
            <text class="entry-title">{{ displayName || '账户中心' }}</text>
            <text class="entry-desc">{{
              displayName ? '查看个人资料和应用设置' : '登录后管理账户'
            }}</text>
          </view>
          <text class="entry-arrow">›</text>
        </view>
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
  padding: 48rpx 32rpx;
  box-sizing: border-box;
}

.hero {
  padding: 64rpx 12rpx 72rpx;
}

.eyebrow {
  display: block;
  color: #2f9f87;
  font-size: 22rpx;
  letter-spacing: 2rpx;
}

.title {
  display: block;
  margin-top: 20rpx;
  color: #17221f;
  font-size: 52rpx;
  font-weight: 700;
}

.subtitle {
  display: block;
  margin-top: 18rpx;
  color: #71807b;
  font-size: 28rpx;
}

.section-title {
  margin: 0 12rpx 20rpx;
  color: #26332f;
  font-size: 28rpx;
  font-weight: 600;
}

.entry-list {
  overflow: hidden;
  border-radius: 16rpx;
  background: #fff;
}

.entry {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 30rpx 28rpx;
  border-bottom: 1rpx solid #edf1ef;
}

.entry:last-child {
  border-bottom: 0;
}

.entry-title,
.entry-desc {
  display: block;
}

.entry-title {
  color: #1f2c28;
  font-size: 30rpx;
}

.entry-desc {
  margin-top: 8rpx;
  color: #8a9691;
  font-size: 24rpx;
}

.entry-arrow {
  color: #9eaba6;
  font-size: 44rpx;
  line-height: 1;
}
</style>
