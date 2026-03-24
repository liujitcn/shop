<script setup lang="ts">
import { ref } from 'vue'
// 获取屏幕边界到安全区域距离
const { safeAreaInsets } = uni.getSystemInfoSync()
const searchValue = ref('')

// 执行搜索
const handleSearch = () => {
  if (!searchValue.value.trim()) return
  uni.navigateTo({
    url: `/pages/search/index?name=${encodeURIComponent(searchValue.value)}`,
  })
  searchValue.value = ''
}
import { useSettingStore } from '@/stores'

const settingStore = useSettingStore()
</script>

<template>
  <view class="navbar" :style="{ paddingTop: safeAreaInsets!.top + 10 + 'px' }">
    <!-- 标题区域 -->
    <view class="title-area">
      <text class="main-title">{{
        settingStore.getData('mainTitle') || '海沃丰农资批发商城'
      }}</text>
      <text class="sub-title">{{
        settingStore.getData('subTitle') || '一站式作物整体解决方案集成商'
      }}</text>
    </view>
    <!-- 搜索条 -->
    <view class="search">
      <input
        v-model="searchValue"
        class="input"
        placeholder="搜索商品"
        placeholder-class="placeholder"
        confirm-type="search"
        @blur="handleSearch"
        @confirm="handleSearch"
      />
    </view>
  </view>
</template>

<style lang="scss">
/* 自定义导航条 */
.navbar {
  background-image: url(@/static/images/navigator_bg.png);
  background-size: cover;
  position: relative;
  display: flex;
  flex-direction: column;
  padding-top: 20px;
  .title-area {
    display: flex;
    flex-direction: column;
    padding-left: 30rpx;
    margin-bottom: 10rpx;
    .main-title {
      font-size: 48rpx;
      font-weight: bold;
      color: #fff;
      line-height: 1.2;
    }
    .sub-title {
      font-size: 24rpx;
      color: rgba(255, 255, 255, 0.9);
      line-height: 1.3;
      margin-top: 8rpx;
    }
  }
  .search {
    height: 64rpx;
    margin: 16rpx 20rpx;
    background-color: rgba(255, 255, 255, 0.5);
    border-radius: 32rpx;
    .input {
      height: 100%;
      padding: 0 30rpx;
      color: #fff;
      font-size: 28rpx;
    }
    .placeholder {
      color: rgba(255, 255, 255, 0.8);
    }
  }
}
</style>
