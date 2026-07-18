<script setup lang="ts">
import { useGuessList } from '@/composables'
import { useUserStore } from '@/stores'
import { onShow } from '@dcloudio/uni-app'
import { defOrderService } from '@/api/app/order_info.ts'
import { defCommentService } from '@/api/app/comment'
import { computed, ref } from 'vue'
import { formatSrc } from '@/utils'
import { navigateToLogin, orderListUrl } from '@/utils/navigation'
import { OrderInfoStatus, OrderTradeStatus, RecommendScene } from '@/rpc/common/v1/enum'
import type { OrderListFilter } from '@/utils/order'
// 获取屏幕边界到安全区域距离
const { safeAreaInsets } = uni.getSystemInfoSync()
const COMMENT_CENTER_PENDING_PAGE = '/pagesOrder/comment/center?tab=pending'
const AFTERSALE_APPLY_PAGE = '/pagesOrder/aftersale/aftersale?tab=apply'
const AI_ASSISTANT_PAGE = '/pagesMember/ai-assistant/index'

/** 我的页面订单入口展示项。 */
type OrderCountEntry = {
  key: string
  icon: string
  text: string
  filter?: OrderListFilter
  refund: boolean
  url?: string
  num: number
}

const orderCount = ref<OrderCountEntry[]>([
  {
    key: 'pending-payment',
    icon: '/static/images/order_pay_ref.png',
    text: '待支付',
    filter: { trade_status: OrderTradeStatus.PENDING_PAYMENT_OTS },
    refund: false,
    num: 0,
  },
  {
    key: 'wait-shipment',
    icon: '/static/images/order_deliver_ref.png',
    text: '待发货',
    filter: { status: OrderInfoStatus.WAIT_SHIPMENT_OIS },
    refund: false,
    num: 0,
  },
  {
    key: 'shipped',
    icon: '/static/images/order_receive_ref.png',
    text: '待收货',
    filter: { status: OrderInfoStatus.SHIPPED_OIS },
    refund: false,
    num: 0,
  },
  {
    key: 'wait-review',
    icon: '/static/images/order_review_ref.png',
    text: '待评价',
    refund: false,
    url: COMMENT_CENTER_PENDING_PAGE,
    num: 0,
  },
  {
    key: 'refund',
    icon: '/static/images/order_aftersale_ref.png',
    text: '退款/售后',
    filter: { refundable: true },
    refund: true,
    url: AFTERSALE_APPLY_PAGE,
    num: 0,
  },
])
// 获取会员信息
const userStore = useUserStore()
const isLoggedIn = computed(() => userStore.isAuthenticated())
const profile = computed(() => userStore.userInfo)
// 判断当前是否仍具备加载个人订单数据的登录态。
const canLoadOrderData = () => userStore.isAuthenticated()
/** 加载我的页面订单数量，并在登录态仍有效时回写入口文案。 */
const getOrderData = async () => {
  if (!canLoadOrderData()) {
    return
  }

  const [orderResponse, pendingCommentResponse] = await Promise.all([
    defOrderService.CountOrderInfo({}),
    defCommentService.PagePendingCommentGoods({ page_num: 1, page_size: 1 }),
  ])
  if (!canLoadOrderData()) {
    return
  }

  orderCount.value.forEach((entry) => {
    if (entry.key === 'wait-review') {
      entry.num = pendingCommentResponse.total
      return
    }
    // 服务端 JSON 会省略零值字段，必须归一化默认值后按全部维度精确匹配统计项。
    entry.num = orderResponse.counts.reduce((total, item) => {
      const matched =
        (entry.filter?.trade_status ?? OrderTradeStatus.UNKNOWN_OTS) ===
          (item.trade_status ?? OrderTradeStatus.UNKNOWN_OTS) &&
        (entry.filter?.status ?? OrderInfoStatus.UNKNOWN_OIS) ===
          (item.status ?? OrderInfoStatus.UNKNOWN_OIS) &&
        Boolean(entry.filter?.refundable) === Boolean(item.refundable)
      return matched ? total + (item.num ?? 0) : total
    }, 0)
  })
}

const { guessRef, onScrollToLower } = useGuessList()
const guessTitle = computed(() => {
  if (isLoggedIn.value) {
    return '根据你的偏好推荐'
  }
  return '热门好物推荐'
})

const getOrderEntryUrl = (entry: OrderCountEntry) => {
  return entry.url || orderListUrl(entry.filter || {})
}

/** 打开移动端 AI 助手静态页，未登录时先进入登录流程。 */
const navigateToAiAssistant = () => {
  if (!userStore.ensureAuthenticated()) {
    navigateToLogin()
    return
  }
  uni.navigateTo({
    url: AI_ASSISTANT_PAGE,
  })
}

// 初始化调用: 页面显示触发
onShow(() => {
  if (canLoadOrderData()) {
    getOrderData()
  }
})
</script>

<template>
  <scroll-view enable-back-to-top @scrolltolower="onScrollToLower" class="viewport" scroll-y>
    <!-- 个人资料 -->
    <view class="profile" :style="{ paddingTop: safeAreaInsets!.top + 'px' }">
      <!-- 情况1：已登录 -->
      <view class="overview" v-if="isLoggedIn && profile">
        <navigator url="/pagesMember/profile/profile" hover-class="none">
          <image
            v-if="profile.avatar"
            class="avatar"
            :src="formatSrc(profile.avatar)"
            mode="aspectFill"
          ></image>
          <image v-else class="avatar" src="@/static/images/avatar.png" mode="aspectFill"></image>
        </navigator>
        <view class="meta">
          <view class="nickname">
            {{ profile.nick_name }}
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
        <navigator v-if="isLoggedIn" class="navigator" :url="orderListUrl(0)" hover-class="none">
          查看全部订单<text class="icon-right"></text>
        </navigator>
        <view v-else class="navigator" @tap="navigateToLogin"
          >查看全部订单<text class="icon-right"></text
        ></view>
      </view>
      <view class="section">
        <!-- 订单 -->
        <template v-if="isLoggedIn">
          <navigator
            v-for="item in orderCount"
            :key="item.key"
            :url="getOrderEntryUrl(item)"
            :class="['navigator', item.refund ? 'order-aftersale-item' : '']"
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
            :key="item.key"
            :class="['navigator', item.refund ? 'order-aftersale-item' : '']"
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
    <!-- AI 助手入口 -->
    <view class="ai-assistant-entry" @tap="navigateToAiAssistant">
      <view class="ai-assistant-entry__icon">AI</view>
      <view class="ai-assistant-entry__content">
        <view class="ai-assistant-entry__title">商城 AI 助手</view>
        <view class="ai-assistant-entry__desc">帮你找商品、查订单、看售后和整理推荐</view>
      </view>
      <view class="ai-assistant-entry__action">去提问</view>
    </view>
    <!-- 猜你喜欢 -->
    <view class="guess">
      <GoodsGuess ref="guessRef" :title="guessTitle" :scene="RecommendScene.PROFILE" />
    </view>
  </scroll-view>
</template>

<style lang="scss">
page {
  height: 100%;
  overflow: hidden;
  background-color: #f7f7f8;
}

/* AI 助手入口 */
.ai-assistant-entry {
  position: relative;
  z-index: 99;
  display: flex;
  align-items: center;
  margin: 20rpx 20rpx 0;
  padding: 26rpx 24rpx;
  border-radius: 10rpx;
  background-color: #fff;
  box-shadow: 0 4rpx 6rpx rgba(240, 240, 240, 0.6);
}

.ai-assistant-entry__icon {
  flex-shrink: 0;
  width: 92rpx;
  height: 92rpx;
  border-radius: 24rpx;
  color: #fff;
  font-size: 28rpx;
  font-weight: 700;
  line-height: 92rpx;
  text-align: center;
  background-color: #27ba9b;
}

.ai-assistant-entry__content {
  flex: 1;
  min-width: 0;
  margin-left: 18rpx;
}

.ai-assistant-entry__title {
  color: #1e1e1e;
  font-size: 30rpx;
  font-weight: 600;
  line-height: 40rpx;
}

.ai-assistant-entry__desc {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  margin-top: 8rpx;
  color: #747f7c;
  font-size: 24rpx;
  line-height: 32rpx;
}

.ai-assistant-entry__action {
  flex-shrink: 0;
  margin-left: 16rpx;
  padding: 10rpx 18rpx;
  border-radius: 999rpx;
  color: #16806d;
  font-size: 24rpx;
  background-color: #e8f8f4;
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
