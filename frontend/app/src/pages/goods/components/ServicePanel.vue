<script setup lang="ts">
import type { ShopService } from '@/rpc/app/v1/shop_service'
// 定义 props 接收
defineProps<{
  list: ShopService[]
}>()
// 子调父
const emit = defineEmits<{
  (event: 'close'): void
}>()
</script>

<template>
  <view class="service-panel">
    <!-- 关闭按钮 -->
    <text class="close icon-close" @tap="emit('close')"></text>
    <!-- 标题 -->
    <view class="title">服务说明</view>
    <!-- 内容 -->
    <view class="content">
      <view class="item" v-for="item in list" :key="item.value">
        <view class="dt">{{ item.label }}</view>
        <view class="dd">{{ item.value }}</view>
      </view>
    </view>
  </view>
</template>

<style lang="scss">
.service-panel {
  /* 新增样式 */
  height: 80vh; /* 设置固定高度 */
  display: flex;
  flex-direction: column;
  padding: 0 30rpx;
  border-radius: 10rpx 10rpx 0 0;
  position: relative;
  background-color: #fff;
}

.title {
  line-height: 1;
  padding: 40rpx 0;
  text-align: center;
  font-size: 32rpx;
  font-weight: normal;
  border-bottom: 1rpx solid #ddd;
  color: #444;
}

.close {
  position: absolute;
  right: 24rpx;
  top: 24rpx;
}

.content {
  /* 修改后样式 */
  flex: 1;
  overflow-y: auto; /* 允许垂直滚动 */
  padding: 20rpx 20rpx 100rpx 20rpx;
  -webkit-overflow-scrolling: touch; /* 优化移动端滚动 */

  .item {
    margin-top: 20rpx;
  }

  .dt {
    margin-bottom: 10rpx;
    font-size: 28rpx;
    color: #333;
    font-weight: 500;
    position: relative;

    &::before {
      content: '';
      width: 10rpx;
      height: 10rpx;
      border-radius: 50%;
      background-color: #eaeaea;
      transform: translateY(-50%);
      position: absolute;
      top: 50%;
      left: -20rpx;
    }
  }

  .dd {
    line-height: 1.6;
    font-size: 26rpx;
    color: #999;
  }
}
</style>
