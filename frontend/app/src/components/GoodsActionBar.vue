<script setup lang="ts">
import { computed } from 'vue'

const emit = defineEmits<{
  collect: []
  addCart: []
  buyNow: []
}>()

const props = withDefaults(
  defineProps<{
    collected?: boolean
    cartNum?: number
    fixed?: boolean
    safeBottom?: number
    showContact?: boolean
    buyLoading?: boolean
    cartUrl?: string
  }>(),
  {
    collected: false,
    cartNum: 0,
    fixed: false,
    safeBottom: 0,
    showContact: false,
    buyLoading: false,
    cartUrl: '/pages/cart/cart2',
  },
)

const rootStyle = computed(() => {
  return {
    paddingBottom: `${props.safeBottom}px`,
  }
})

const cartBadgeText = computed(() => {
  return props.cartNum > 99 ? '99+' : props.cartNum
})
</script>

<template>
  <view
    class="goods-action-bar"
    :class="{ 'goods-action-bar--fixed': props.fixed }"
    :style="rootStyle"
  >
    <view class="goods-action-bar__icons">
      <button class="goods-action-bar__icons-button" @tap="emit('collect')">
        <text class="icon-heart" :class="{ 'goods-action-bar__heart--active': props.collected }" />
        {{ props.collected ? '已收藏' : '收藏' }}
      </button>
      <!-- #ifdef MP-WEIXIN -->
      <button v-if="props.showContact" class="goods-action-bar__icons-button" open-type="contact">
        <text class="icon-handset" />客服
      </button>
      <!-- #endif -->
      <navigator class="goods-action-bar__icons-button" :url="props.cartUrl" open-type="navigate">
        <text class="icon-cart" />购物车
        <view v-if="props.cartNum > 0" class="goods-action-bar__cart-badge">
          {{ cartBadgeText }}
        </view>
      </navigator>
    </view>
    <view class="goods-action-bar__buttons">
      <view class="goods-action-bar__addcart" @tap="emit('addCart')">加入购物车</view>
      <view
        class="goods-action-bar__payment"
        :class="{ 'goods-action-bar__payment--loading': props.buyLoading }"
        @tap="emit('buyNow')"
      >
        {{ props.buyLoading ? '加载中' : '立即购买' }}
      </view>
    </view>
  </view>
</template>

<style lang="scss">
.goods-action-bar {
  height: 100rpx;
  padding: 0 20rpx;
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-top: 1rpx solid #eaeaea;
  box-sizing: content-box;
  background-color: #fff;
}

.goods-action-bar--fixed {
  position: fixed;
  left: 0;
  right: 0;
  bottom: calc((var(--window-bottom)));
  z-index: 1;
}

.goods-action-bar__icons {
  position: relative;
  flex: 1;
  display: flex;
  align-items: center;
  padding-right: 20rpx;
}

.goods-action-bar__icons-button {
  position: relative;
  flex: 1;
  margin: 0;
  padding: 0;
  border-radius: 0;
  color: #333;
  font-size: 20rpx;
  line-height: 1.4;
  text-align: center;
  background-color: #fff;

  &::after {
    border: none;
  }
}

.goods-action-bar__icons-button text {
  display: block;
  font-size: 34rpx;
  transition: color 0.3s ease;
}

.goods-action-bar__heart--active::before {
  color: #ff0000 !important;
}

.goods-action-bar__buttons {
  display: flex;
}

.goods-action-bar__buttons > view {
  width: 220rpx;
  border-radius: 72rpx;
  color: #fff;
  font-size: 26rpx;
  line-height: 72rpx;
  text-align: center;
}

.goods-action-bar__addcart {
  background-color: #ffa868;
}

.goods-action-bar__payment {
  margin-left: 20rpx;
  background-color: #27ba9b;
}

.goods-action-bar__payment--loading {
  opacity: 0.72;
}

.goods-action-bar__cart-badge {
  position: absolute;
  top: -5rpx;
  right: -5rpx;
  min-width: 36rpx;
  height: 36rpx;
  padding: 0 8rpx;
  border-radius: 100rpx;
  color: #fff;
  font-size: 20rpx;
  line-height: 36rpx;
  text-align: center;
  background-color: #ff4444;
}
</style>
