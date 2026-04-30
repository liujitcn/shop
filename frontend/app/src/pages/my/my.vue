<script setup lang="ts">
import { useGuessList } from '@/composables'
import { useUserStore } from '@/stores'
import { onShow } from '@dcloudio/uni-app'
import { defOrderService } from '@/api/app/order_info.ts'
import { defBaseDictService } from '@/api/app/base_dict'
import { computed, ref } from 'vue'
import type { CountOrderInfoResponse_Count } from '@/rpc/app/v1/order_info'
import { formatSrc } from '@/utils'
import { navigateToLogin, orderListUrl } from '@/utils/navigation'
import { OrderStatus, RecommendScene } from '@/rpc/common/v1/enum'
// 获取屏幕边界到安全区域距离
const { safeAreaInsets } = uni.getSystemInfoSync()
const COMMENT_CENTER_PENDING_PAGE = '/pagesOrder/comment/center?tab=pending'
const AFTERSALE_RECORD_PAGE = '/pagesOrder/aftersale/aftersale?tab=record'

type OrderCountEntry = CountOrderInfoResponse_Count & {
  icon: string
  text: string
}

const orderCount = ref<OrderCountEntry[]>([
  { status: OrderStatus.CREATED, icon: '/static/images/order_pay_ref.png', text: '待付款', num: 0 },
  {
    status: OrderStatus.PAID,
    icon: '/static/images/order_deliver_ref.png',
    text: '待发货',
    num: 0,
  },
  {
    status: OrderStatus.SHIPPED,
    icon: '/static/images/order_receive_ref.png',
    text: '待收货',
    num: 0,
  },
  {
    status: OrderStatus.WAIT_REVIEW,
    icon: '/static/images/order_review_ref.png',
    text: '待评价',
    num: 0,
  },
  {
    status: OrderStatus.REFUNDING,
    icon: '/static/images/order_aftersale_ref.png',
    text: '退款/售后',
    num: 0,
  },
])
// 获取会员信息
const userStore = useUserStore()
const getOrderData = async () => {
  const numMap = new Map<number, number>()
  const res = await defOrderService.CountOrderInfo({})
  if (res.counts) {
    res.counts.map((item) => {
      numMap.set(item.status, item.num)
    })
  }

  const code = 'order_status'
  const orderStatus = await defBaseDictService.GetBaseDict({
    value: code,
  })
  const textMap = new Map<number, string>()
  if (orderStatus && orderStatus.items) {
    orderStatus.items.map((dictItem) => {
      textMap.set(Number(dictItem.value), dictItem.label)
    })
  }

  orderCount.value.map((item) => {
    item.num = numMap.get(item.status) || 0
    item.text =
      item.status === OrderStatus.WAIT_REVIEW || item.status === OrderStatus.REFUNDING
        ? item.text
        : textMap.get(item.status) || item.text
  })
}

const { guessRef, onScrollToLower } = useGuessList()
const guessTitle = computed(() => {
  if (userStore.userInfo) {
    return '根据你的偏好推荐'
  }
  return '热门好物推荐'
})

const getOrderEntryUrl = (status: OrderStatus) => {
  if (status === OrderStatus.WAIT_REVIEW) {
    return COMMENT_CENTER_PENDING_PAGE
  }
  if (status === OrderStatus.REFUNDING) {
    return AFTERSALE_RECORD_PAGE
  }
  return orderListUrl(status)
}

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
            {{ userStore.userInfo.nick_name }}
          </view>
          <navigator class="extra" url="/pagesMember/profile/profile" hover-class="none">
            <text class="update">更新头像昵称</text>
          </navigator>
        </view>
      </view>
      <!-- 情况2：未登录 -->
      <view class="overview" v-else>
        <view @tap="navigateToLogin">
          <image class="avatar gray" mode="aspectFill" src="@/static/images/avatar.png"></image>
        </view>
        <view class="meta">
          <view @tap="navigateToLogin" class="nickname"> 未登录 </view>
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
          v-if="userStore.userInfo"
          class="navigator"
          :url="orderListUrl(0)"
          hover-class="none"
        >
          查看全部订单<text class="icon-right"></text>
        </navigator>
        <view v-else class="navigator" @tap="navigateToLogin"
          >查看全部订单<text class="icon-right"></text
        ></view>
      </view>
      <view class="section">
        <!-- 订单 -->
        <template v-if="userStore.userInfo">
          <navigator
            v-for="item in orderCount"
            :key="item.status"
            :url="getOrderEntryUrl(item.status)"
            :class="[
              'navigator',
              item.status === OrderStatus.REFUNDING ? 'order-aftersale-item' : '',
            ]"
            hover-class="none"
          >
            <view class="order-icon-wrap">
              <image class="order-icon" :src="item.icon" mode="aspectFit"></image>
              <text class="badge" v-if="item.num">{{ item.num > 99 ? '99+' : item.num }}</text>
            </view>
            <text class="order-text">{{ item.text }}</text>
          </navigator>
        </template>
        <template v-else>
          <view
            v-for="item in orderCount"
            :key="item.status"
            :class="[
              'navigator',
              item.status === OrderStatus.REFUNDING ? 'order-aftersale-item' : '',
            ]"
            @tap="navigateToLogin"
          >
            <view class="order-icon-wrap">
              <image class="order-icon" :src="item.icon" mode="aspectFit"></image>
            </view>
            <text class="order-text">{{ item.text }}</text>
          </view>
        </template>
      </view>
    </view>
    <!-- 猜你喜欢 -->
    <view class="guess">
      <XtxGuess ref="guessRef" :title="guessTitle" :scene="RecommendScene.PROFILE" />
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
    box-sizing: border-box;
    display: flex;
    justify-content: flex-start;
    padding: 40rpx 20rpx 10rpx;
    position: relative;

    .navigator {
      position: relative; /* 为badge定位提供参考 */
      flex: 0 0 20%;
      width: 20%;
      min-width: 0;
      display: flex;
      flex-direction: column; /* 改为垂直布局 */
      align-items: center; /* 水平居中 */
      justify-content: center; /* 垂直居中 */
      text-align: center;
      font-size: 24rpx;
      color: #333;
    }

    .order-icon-wrap {
      position: relative;
      display: flex;
      align-items: center;
      justify-content: center;
      width: 60rpx;
      height: 60rpx;
      margin-bottom: 10rpx;
    }

    .order-icon {
      display: block;
      width: 60rpx;
      height: 60rpx;
      opacity: 0.68;
    }

    .order-aftersale-item .order-icon-wrap {
      transform: translateX(22rpx);
    }

    .badge {
      position: absolute;
      top: -8rpx;
      right: -10rpx;
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
    }

    .order-text {
      width: 100%;
      line-height: 1.4;
      text-align: center;
      white-space: nowrap;
    }
  }
}

/* 猜你喜欢 */
.guess {
  background-color: #f7f7f8;
  margin-top: 20rpx;
}
</style>
