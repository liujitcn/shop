<script setup lang="ts">
import type { AiAssistantAction } from '@/rpc/base/v1/ai_assistant_message'
import { formatPrice } from '@/utils/index'

type AssistantFlowBlock = {
  type: string
  [key: string]: any
}

const emit = defineEmits<{
  'flow-action': [action?: AiAssistantAction, label?: string]
}>()

const props = defineProps<{
  orders: AssistantFlowBlock[]
  reveal?: boolean
  activeFlowMessageId: string
}>()

function resolveFlowActionLabel(action?: AiAssistantAction) {
  const labelMap: Record<string, string> = {
    start_payment: '发起支付',
    view_order: '查看订单',
    receive_order: '确认收货',
  }
  return action?.type ? labelMap[action.type] || '继续' : '继续'
}

function isActionEnabled(action?: Partial<AiAssistantAction>) {
  if (!action?.type || !props.activeFlowMessageId) {
    return false
  }
  return (
    action.source_message_id === props.activeFlowMessageId &&
    Boolean(action.action_id) &&
    String(action.flow_version || '') === props.activeFlowMessageId
  )
}

function handleFlowAction(action?: AiAssistantAction) {
  if (!isActionEnabled(action)) {
    return
  }
  emit('flow-action', action, resolveFlowActionLabel(action))
}
</script>

<template>
  <view class="flow-order-list">
    <view v-if="!orders.length" class="flow-empty">暂时没有相关订单</view>
    <view
      v-for="order in orders"
      :key="order.id"
      class="flow-order-card"
      :class="{ 'flow-reveal-item': reveal, 'is-disabled': !isActionEnabled(order.action) }"
    >
      <view class="flow-order-head">
        <view class="flow-order-title">
          <text class="flow-order-label">订单</text>
          <text class="flow-order-no">{{ order.order_no || order.id }}</text>
        </view>
        <text class="flow-order-price">¥{{ formatPrice(Number(order.pay_money || 0)) }}</text>
      </view>
      <template v-for="group in order.order_goods_stores || []" :key="group.store?.id">
        <view v-for="goods in group.goods || []" :key="goods.sku_code" class="flow-order-goods">
          <text class="flow-order-goods-name">{{ goods.name }}</text>
          <text class="flow-order-goods-num">x{{ goods.num }}</text>
        </view>
      </template>
      <button
        class="flow-primary-button is-wide"
        :class="{ 'is-disabled': !isActionEnabled(order.action) }"
        hover-class="none"
        @tap="handleFlowAction(order.action)"
      >
        {{ order.action?.type === 'start_payment' ? '继续支付' : '查看详情' }}
      </button>
    </view>
  </view>
</template>

<style lang="scss" scoped>
.flow-empty {
  color: #898b94;
  font-size: 24rpx;
  line-height: 36rpx;
}

.flow-order-card {
  max-width: 100%;
  padding: 16rpx;
  border-radius: 10rpx;
  background-color: #fff;
  box-sizing: border-box;
  overflow: hidden;
}

.flow-order-card + .flow-order-card {
  margin-top: 14rpx;
}

.flow-order-card.is-disabled {
  opacity: 0.58;
}

.flow-reveal-item {
  animation: flow-reveal-in 180ms ease-out both;
}

@keyframes flow-reveal-in {
  from {
    opacity: 0;
    transform: translateY(10rpx);
  }

  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.flow-order-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16rpx;
  min-width: 0;
}

.flow-order-title {
  flex: 1;
  min-width: 0;
  color: #333;
  font-size: 23rpx;
  font-weight: 600;
  line-height: 32rpx;
}

.flow-order-label {
  display: block;
}

.flow-order-no {
  display: block;
  max-width: 100%;
  overflow-wrap: anywhere;
  word-break: break-all;
}

.flow-order-price {
  flex-shrink: 0;
  max-width: 180rpx;
  color: #333;
  font-size: 23rpx;
  font-weight: 600;
  line-height: 32rpx;
  text-align: right;
}

.flow-order-goods {
  display: flex;
  align-items: center;
  gap: 12rpx;
  min-width: 0;
  margin-top: 10rpx;
  color: #666;
  font-size: 22rpx;
  line-height: 32rpx;
}

.flow-order-goods-name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.flow-order-goods-num {
  flex-shrink: 0;
  color: #898b94;
}

.flow-primary-button {
  flex-shrink: 0;
  width: 112rpx;
  height: 56rpx;
  padding: 0;
  margin: 0;
  border-radius: 8rpx;
  color: #fff;
  font-size: 24rpx;
  line-height: 56rpx;
  background-color: #27ba9b;

  &::after {
    border: none;
  }
}

.flow-primary-button.is-wide {
  width: 100%;
  margin-top: 18rpx;
}

.flow-primary-button.is-disabled {
  color: #eef6f3;
  background-color: #9ddfcc;
}
</style>
