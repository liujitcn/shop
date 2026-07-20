<script setup lang="ts">
import { ref } from 'vue'
import { defBaseDictService } from '@/api/system/app/base_dict'
import { defOrderService } from '@/api/shop/app/order_info'
import type { BaseDictForm_DictItem } from '@/rpc/system/app/v1/base_dict'
import type { OrderInfo } from '@/rpc/shop/app/v1/order_info'

/** 退款弹窗只需要订单编号，避免弹窗依赖完整订单结构。 */
type RefundOrderTarget = Pick<OrderInfo, 'id'>

const emit = defineEmits<{
  success: [order_id: number]
}>()

const popup = ref<UniHelper.UniPopupInstance>()
const reasonList = ref<BaseDictForm_DictItem[]>([])
const reason = ref('')
const currentOrder = ref<RefundOrderTarget>()

/** 懒加载退款原因字典。 */
const getReasonList = async () => {
  if (reasonList.value.length) {
    return
  }
  const refundReasonDict = await defBaseDictService.GetBaseDict({ value: 'order_refund_reason' })
  reasonList.value = refundReasonDict.items || []
}

/** 打开退款弹窗并初始化当前订单。 */
const open = async (order: RefundOrderTarget) => {
  await getReasonList()
  currentOrder.value = order
  reason.value = ''
  popup.value?.open?.()
}

/** 关闭退款弹窗并清理临时状态。 */
const close = () => {
  currentOrder.value = undefined
  reason.value = ''
  popup.value?.close?.()
}

/** 提交退款申请并通知父组件刷新列表。 */
const onConfirmRefund = async () => {
  if (!currentOrder.value) {
    void uni.showToast({ icon: 'none', title: '请选择订单' })
    return
  }
  if (!reason.value) {
    void uni.showToast({ icon: 'none', title: '请选择订单退款的原因' })
    return
  }

  await defOrderService.RefundOrderInfo({
    order_id: currentOrder.value.id,
    reason: Number(reason.value),
  })
  const order_id = currentOrder.value.id
  void uni.showToast({ icon: 'none', title: '退款申请已提交' })
  close()
  emit('success', order_id)
}

defineExpose({
  open,
  close,
})
</script>

<template>
  <uni-popup ref="popup" type="bottom" background-color="#fff">
    <view class="refund-popup-root">
      <view class="title">订单退款</view>
      <view class="description">
        <view class="tips">请选择订单退款的原因：</view>
        <view v-for="item in reasonList" :key="item.value" class="cell" @tap="reason = item.value">
          <text class="text">{{ item.label }}</text>
          <text class="icon" :class="{ checked: item.value === reason }" />
        </view>
      </view>
      <view class="footer">
        <view class="button" @tap="close">取消</view>
        <view class="button primary" @tap="onConfirmRefund">确认</view>
      </view>
    </view>
  </uni-popup>
</template>

<style lang="scss">
.refund-popup-root {
  padding: 30rpx 30rpx 0;
  border-radius: 10rpx 10rpx 0 0;
  overflow: hidden;

  .title {
    font-size: 30rpx;
    text-align: center;
    margin-bottom: 30rpx;
  }

  .description {
    font-size: 28rpx;
    padding: 0 20rpx;

    .tips {
      color: #444;
      margin-bottom: 12rpx;
    }

    .cell {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 15rpx 0;
      color: #666;
    }

    .icon::before {
      content: '\e6cd';
      font-family: 'erabbit' !important;
      font-size: 38rpx;
      color: #999;
    }

    .icon.checked::before {
      content: '\e6cc';
      font-size: 38rpx;
      color: #27ba9b;
    }
  }

  .footer {
    display: flex;
    justify-content: space-between;
    padding: 30rpx 0 40rpx;
    font-size: 28rpx;
    color: #444;

    .button {
      flex: 1;
      height: 72rpx;
      text-align: center;
      line-height: 72rpx;
      margin: 0 20rpx;
      color: #444;
      border-radius: 72rpx;
      border: 1rpx solid #ccc;
    }

    .primary {
      color: #fff;
      background-color: #27ba9b;
      border: none;
    }
  }
}
</style>
