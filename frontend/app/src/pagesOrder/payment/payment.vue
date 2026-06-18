<script setup lang="ts">
import { computed, ref } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import { defOrderService } from '@/api/app/order_info'
import { useGuessList } from '@/composables'
import type { OrderInfoResponse } from '@/rpc/app/v1/order_info'
import { OrderPayType, OrderStatus, RecommendScene } from '@/rpc/common/v1/enum'
import { homeTabPage, orderDetailUrl } from '@/utils/navigation'

// 获取页面参数
const query = defineProps<{
  id: string
}>()

const orderData = ref<OrderInfoResponse>()
const loadError = ref(false)

// 根据后端订单状态生成支付结果，避免仅凭前端支付回调展示成功。
const paymentState = computed(() => {
  const order = orderData.value?.order
  if (loadError.value) {
    return {
      title: '支付结果查询失败',
      tips: '暂时无法确认支付结果，请进入订单详情查看最新状态',
      tone: 'unknown',
      success: false,
      showRecommendation: false,
    }
  }
  if (!order) {
    return {
      title: '支付结果确认中',
      tips: '正在查询订单最新状态，请稍候',
      tone: 'pending',
      success: false,
      showRecommendation: false,
    }
  }
  if (order.status === OrderStatus.CANCELED || order.status === OrderStatus.DELETED) {
    return {
      title: '订单已取消',
      tips: '当前订单已取消，如有疑问请进入订单详情查看',
      tone: 'unknown',
      success: false,
      showRecommendation: false,
    }
  }
  if (order.pay_type === OrderPayType.CASH_ON_DELIVERY) {
    return {
      title: '订单提交成功',
      tips: '订单将按货到付款方式处理，请后续关注发货信息',
      tone: 'success',
      success: true,
      showRecommendation: false,
    }
  }
  if (order.status === OrderStatus.CREATED) {
    return {
      title: '支付结果确认中',
      tips: '支付结果可能存在短暂延迟，请稍后进入订单详情查看',
      tone: 'pending',
      success: false,
      showRecommendation: false,
    }
  }
  const paidStatuses = [
    OrderStatus.PAID,
    OrderStatus.SHIPPED,
    OrderStatus.WAIT_REVIEW,
    OrderStatus.COMPLETED,
    OrderStatus.REFUNDING,
  ]
  if (paidStatuses.includes(order.status)) {
    return {
      title: '支付成功',
      tips: '请后续关注发货信息，有问题及时联系',
      tone: 'success',
      success: true,
      showRecommendation: true,
    }
  }
  return {
    title: '支付状态异常',
    tips: '当前订单状态无法确认，请进入订单详情查看',
    tone: 'unknown',
    success: false,
    showRecommendation: false,
  }
})

// loadPaymentResult 查询订单真实状态并刷新支付结果展示。
const loadPaymentResult = async () => {
  loadError.value = false
  try {
    orderData.value = await defOrderService.GetOrderInfoById({ id: Number(query.id) })
  } catch {
    loadError.value = true
  }
}

// 猜你喜欢
const { guessRef, onScrollToLower } = useGuessList()

onLoad(() => {
  void loadPaymentResult()
})
</script>

<template>
  <scroll-view enable-back-to-top class="viewport" scroll-y @scrolltolower="onScrollToLower">
    <!-- 订单状态 -->
    <view class="overview" :class="paymentState.tone">
      <view class="status" :class="{ 'icon-checked': paymentState.success }">
        {{ paymentState.title }}
      </view>
      <view class="tips">提示: {{ paymentState.tips }}</view>
      <view class="buttons">
        <navigator
          hover-class="none"
          class="button navigator"
          :url="homeTabPage"
          open-type="switchTab"
        >
          返回首页
        </navigator>
        <navigator
          hover-class="none"
          class="button navigator"
          :url="orderDetailUrl({ id: query.id, internal: true })"
          open-type="redirect"
        >
          查看订单
        </navigator>
      </view>
    </view>

    <!-- 猜你喜欢 -->
    <GoodsGuess
      v-if="paymentState.showRecommendation"
      ref="guessRef"
      title="顺手再带两件"
      :scene="RecommendScene.ORDER_PAID"
      :order-id="Number(query.id)"
    />
  </scroll-view>
</template>

<style lang="scss">
page {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

.viewport {
  background-color: #f7f7f8;
}

.overview {
  line-height: 1;
  padding: 50rpx 0;
  color: #fff;
  background-color: #27ba9b;

  &.pending {
    background-color: #e6a23c;
  }

  &.unknown {
    background-color: #909399;
  }

  .tips {
    width: 70%;
    font-size: 24rpx;
    text-align: center;
    line-height: 1.5;
    margin: 60rpx auto;
  }

  .status {
    font-size: 36rpx;
    font-weight: 500;
    text-align: center;
  }

  .status::before {
    display: block;
    font-size: 110rpx;
    margin-bottom: 20rpx;
  }

  .buttons {
    height: 60rpx;
    line-height: 60rpx;
    display: flex;
    justify-content: center;
    align-items: center;
    margin-top: 60rpx;
  }

  .button {
    text-align: center;
    margin: 0 10rpx;
    font-size: 28rpx;
    color: #fff;

    &:first-child {
      width: 200rpx;
      border-radius: 64rpx;
      border: 1rpx solid #fff;
    }
  }
}
</style>
