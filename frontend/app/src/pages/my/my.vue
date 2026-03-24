<script setup lang="ts">
import { useGuessList } from '@/composables'
import { useUserStore } from '@/stores'
import { onShow } from '@dcloudio/uni-app'
import { defOrderService } from '@/api/app/order'
import { defBaseDictService } from '@/api/app/base_dict'
import { ref } from 'vue'
import type { CountOrderResponse_Count } from '@/rpc/app/order'
import { formatSrc } from '@/utils'
import { OrderStatus } from '@/rpc/common/enum.ts'
// 获取屏幕边界到安全区域距离
const { safeAreaInsets } = uni.getSystemInfoSync()

const orderCount = ref<(CountOrderResponse_Count & { icon?: string; text?: string })[]>([
  { status: OrderStatus.CREATED, icon: 'icon-currency', num: 0 },
  { status: OrderStatus.PAID, icon: 'icon-gift', num: 0 },
  { status: OrderStatus.SHIPPED, icon: 'icon-check', num: 0 },
  { status: OrderStatus.RECEIVED, icon: 'icon-comment', num: 0 },
])
// 获取会员信息
const userStore = useUserStore()
const getOrderData = async () => {
  const numMap = new Map<number, number>()
  const res = await defOrderService.CountOrder({})
  if (res.count) {
    res.count.map((item) => {
      numMap.set(item.status, item.num)
    })
  }

  const code = 'order_status'
  const orderStatus = await defBaseDictService.ListBaseDict({
    value: code,
  })
  const textMap = new Map<number, string>()
  if (orderStatus && orderStatus.list) {
    orderStatus.list.map((dict) => {
      if (dict.code === code)
        if (dict.items) {
          dict.items.map((dictItem) => {
            textMap.set(Number(dictItem.value), dictItem.label)
          })
        }
    })
  }

  orderCount.value.map((item) => {
    item.num = numMap.get(item.status) || 0
    item.text = textMap.get(item.status)
  })
}

const { guessRef, onScrollToLower } = useGuessList()
// 初始化调用: 页面显示触发
onShow(() => {
  if (userStore.userInfo) {
    getOrderData()
  }
})
</script>

<template>
  <scroll-view enable-back-to-top @scrolltolower="onScrollToLower" class="viewport" scroll-y>
    <!-- 个人资料 -->
    <view class="profile" :style="{ paddingTop: safeAreaInsets!.top + 'px' }">
      <!-- 情况1：已登录 -->
      <view class="overview" v-if="userStore.userInfo">
        <navigator url="/pagesMember/profile/profile" hover-class="none">
          <image
            v-if="userStore.userInfo.avatar"
            class="avatar"
            :src="formatSrc(userStore.userInfo.avatar)"
            mode="aspectFill"
          ></image>
          <image v-else class="avatar" src="@/static/images/avatar.png" mode="aspectFill"></image>
        </navigator>
        <view class="meta">
          <view class="nickname">
            {{ userStore.userInfo.nickName }}
          </view>
          <navigator class="extra" url="/pagesMember/profile/profile" hover-class="none">
            <text class="update">更新头像昵称</text>
          </navigator>
        </view>
      </view>
      <!-- 情况2：未登录 -->
      <view class="overview" v-else>
        <navigator url="/pages/login/login" hover-class="none">
          <image class="avatar gray" mode="aspectFill" src="@/static/images/avatar.png"></image>
        </navigator>
        <view class="meta">
          <navigator url="/pages/login/login" hover-class="none" class="nickname">
            未登录
          </navigator>
          <view class="extra">
            <text class="tips">点击登录账号</text>
          </view>
        </view>
      </view>
      <navigator class="settings" url="/pagesMember/settings/settings" hover-class="none">
        设置
      </navigator>
    </view>
    <!-- 我的订单 -->
    <view class="orders">
      <view class="title">
        我的订单
        <navigator
          class="navigator"
          :url="userStore.userInfo ? '/pagesOrder/list/list?status=0' : '/pages/login/login'"
          hover-class="none"
        >
          查看全部订单<text class="icon-right"></text>
        </navigator>
      </view>
      <view class="section">
        <!-- 订单 -->
        <navigator
          v-for="item in orderCount"
          :key="item.status"
          :class="item.icon"
          :url="
            userStore.userInfo
              ? `/pagesOrder/list/list?status=${item.status}`
              : '/pages/login/login'
          "
          class="navigator"
          hover-class="none"
        >
          <span class="badge" v-if="item.num">{{ item.num > 99 ? '99+' : item.num }}</span>
          {{ item.text }}
        </navigator>
      </view>
    </view>
    <!-- 猜你喜欢 -->
    <view class="guess">
      <XtxGuess ref="guessRef" />
    </view>
  </scroll-view>
</template>

<style lang="scss">
page {
  height: 100%;
  overflow: hidden;
  background-color: #f7f7f8;
}

.viewport {
  height: 100%;
  background-repeat: no-repeat;
  background-image: url(@/static/images/center_bg.png);
  background-size: 100% auto;
}

/* 用户信息 */
.profile {
  margin-top: 30rpx;
  position: relative;

  .overview {
    display: flex;
    height: 120rpx;
    padding: 0 36rpx;
    color: #fff;
  }

  .avatar {
    width: 120rpx;
    height: 120rpx;
    border-radius: 50%;
    background-color: #eee;
  }

  .gray {
    filter: grayscale(100%);
  }

  .meta {
    display: flex;
    flex-direction: column;
    justify-content: center;
    align-items: flex-start;
    line-height: 30rpx;
    padding: 16rpx 0;
    margin-left: 20rpx;
  }

  .nickname {
    max-width: 180rpx;
    margin-bottom: 16rpx;
    font-size: 30rpx;

    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .extra {
    display: flex;
    font-size: 20rpx;
  }

  .tips {
    font-size: 22rpx;
  }

  .update {
    padding: 3rpx 10rpx 1rpx;
    color: rgba(255, 255, 255, 0.8);
    border: 1rpx solid rgba(255, 255, 255, 0.8);
    margin-right: 10rpx;
    border-radius: 30rpx;
  }

  .settings {
    position: absolute;
    bottom: 0;
    right: 40rpx;
    font-size: 30rpx;
    color: #fff;
  }
}

/* 我的订单 */
.orders {
  position: relative;
  z-index: 99;
  padding: 30rpx;
  margin: 50rpx 20rpx 0;
  background-color: #fff;
  border-radius: 10rpx;
  box-shadow: 0 4rpx 6rpx rgba(240, 240, 240, 0.6);

  .title {
    height: 40rpx;
    line-height: 40rpx;
    font-size: 28rpx;
    color: #1e1e1e;

    .navigator {
      font-size: 24rpx;
      color: #939393;
      float: right;
    }
  }

  .section {
    width: 100%;
    display: flex;
    justify-content: space-between;
    padding: 40rpx 20rpx 10rpx;
    position: relative;
    .navigator {
      position: relative; /* 为badge定位提供参考 */
      flex: 1;
      display: flex;
      flex-direction: column; /* 改为垂直布局 */
      align-items: center; /* 水平居中 */
      justify-content: center; /* 垂直居中 */
      text-align: center;
      /* 图标样式调整 */
      &::before {
        display: block;
        margin-bottom: 10rpx; /* 增加图标和文字间距 */
        font-size: 60rpx; /* 保持图标大小 */
      }

      /* 文字样式调整 */
      font-size: 24rpx;
      color: #333;
      .badge {
        position: absolute;
        top: -10rpx;
        right: 10rpx;
        min-width: 32rpx;
        height: 32rpx;
        line-height: 32rpx;
        padding: 0 8rpx;
        background-color: #ff4444;
        color: #fff;
        font-size: 20rpx;
        border-radius: 40rpx;
        text-align: center;
        box-shadow: 0 2rpx 4rpx rgba(0, 0, 0, 0.15);
        transform: translate(50%, 0); /* 微调定位 */
      }
    }
  }
}

/* 猜你喜欢 */
.guess {
  background-color: #f7f7f8;
  margin-top: 20rpx;
}
</style>
