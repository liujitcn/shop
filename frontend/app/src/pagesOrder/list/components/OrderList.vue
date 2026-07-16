<script setup lang="ts">
import { OrderInfoStatus, OrderRefundStatus, OrderTradeStatus } from '@/rpc/common/v1/enum'
import { defOrderService } from '@/api/app/order_info.ts'
import type { PageOrderInfoRequest, OrderInfo } from '@/rpc/app/v1/order_info'
import { computed, onMounted, ref } from 'vue'
import type { BaseDictForm_DictItem } from '@/rpc/app/v1/base_dict'
import { defBaseDictService } from '@/api/app/base_dict'
import { onLoad } from '@dcloudio/uni-app'
import { defPayService } from '@/api/app/pay'
import { formatSrc, formatPrice } from '@/utils'
import {
  orderCommentWriteUrl,
  orderCreateUrl,
  orderDetailUrl,
  redirectToOrderPayment,
  tenantStoreUrl,
} from '@/utils/navigation'
import RefundOrderPopup from '../../components/RefundOrderPopup.vue'
import {
  canDeleteOrder,
  canRefundOrder,
  getOrderDisplayStatus,
  isPayableTrade,
} from '@/utils/order'
import type { OrderListFilter } from '@/utils/order'

// 获取屏幕边界到安全区域距离
const { safeAreaInsets } = uni.getSystemInfoSync()

// 订单列表由父页面传入后端筛选条件，每个标签独立分页。
const props = defineProps<{
  filter: OrderListFilter
  title: string
}>()

// 请求参数
const queryParams: PageOrderInfoRequest = {
  page_num: 1,
  page_size: 10,
  ...props.filter,
}

// 弹出层组件
const popup = ref<UniHelper.UniPopupInstance>()
const refundPopup = ref<InstanceType<typeof RefundOrderPopup>>()
const reasonList = ref<BaseDictForm_DictItem[]>([])
// 取消原因列表
const cancelReasonList = ref<BaseDictForm_DictItem[]>([])
// 订单取消原因
const reason = ref('')
// 待取消订单
const orderItem = ref<OrderInfo>()
// 标题
const dialogTitle = ref('')
// tips
const tips = ref('')

const getDictData = async () => {
  const cancelReasonCode = 'order_cancel_reason'
  const cancelReasonDict = await defBaseDictService.GetBaseDict({ value: cancelReasonCode })
  cancelReasonList.value = cancelReasonDict.items || []
}

const buildOrderCommentWriteUrl = (order: OrderInfo) => {
  const firstGoods = order.order_goods_stores[0]?.goods[0]
  return orderCommentWriteUrl({
    order_id: order.id,
    goods_id: firstGoods?.goods_id,
    goods_name: firstGoods?.name,
    goods_picture: firstGoods?.picture ? formatSrc(firstGoods.picture) : undefined,
    sku_code: firstGoods?.sku_code,
    sku_desc: firstGoods?.spec_item?.join(' / '),
  })
}

const buildOrderDetailUrl = (order: OrderInfo) => {
  return order.is_trade
    ? orderDetailUrl({ trade_id: order.trade_id })
    : orderDetailUrl({ id: order.id, internal: true })
}

// 获取订单列表
const orderInfoList = ref<OrderInfo[]>([])
// 是否加载中标记，用于防止滚动触底触发多次请求
const isLoading = ref(false)
const getUserOrderData = async () => {
  // 如果数据出于加载中，退出函数
  if (isLoading.value) return
  // 退出分页判断
  if (isFinish.value === true) {
    if (orderInfoList.value.length > 0) {
      return uni.showToast({ icon: 'none', title: '没有更多数据~' })
    }
    return
  }
  // 发送请求前，标记为加载中
  isLoading.value = true
  try {
    // 发送请求
    const res = await defOrderService.PageOrderInfo(queryParams)
    // 数组追加
    const list = res.order_infos || []
    orderInfoList.value.push(...list)
    // 分页条件
    if (orderInfoList.value.length < res.total) {
      // 页码累加
      queryParams.page_num++
    } else {
      // 分页已结束
      isFinish.value = true
    }
  } finally {
    // 发送请求后，重置标记
    isLoading.value = false
  }
}
onLoad(() => {
  getDictData()
})
onMounted(() => {
  getUserOrderData()
})

// 打开 H5 或 App 支付外链
const openH5PayUrl = (url: string) => {
  // H5 端直接跳转到微信支付链接
  // #ifdef H5
  window.location.href = url
  // #endif

  // App 端通过系统能力打开外部支付链接
  // #ifdef APP-PLUS

  plus.runtime.openURL(url)
  // #endif
}

// 发起订单支付
const onOrderPay = async (tradeID: number) => {
  // #ifdef MP-WEIXIN
  // 正式环境微信支付
  const jsapiRes = await defPayService.JsapiPay({ trade_id: tradeID })
  uni.requestPayment({
    provider: 'wxpay',
    /** 随机字符串，长度为32个字符以下 */
    nonceStr: jsapiRes.nonce_str,
    /** 统一下单接口返回的 prepay_id 参数值，提交格式如：prepay_id=*** */
    package: jsapiRes.package,
    /** 签名，具体见微信支付文档 */
    paySign: jsapiRes.pay_sign,
    /** 时间戳，从 1970 年 1 月 1 日 00:00:00 至今的秒数，即当前的时间 */
    timeStamp: jsapiRes.time_stamp,
    /** 接口调用结束的回调函数（调用成功、失败都会执行） */
    complete: () => {},
    /** 接口调用失败的回调函数 */
    fail: () => {},
    /** 签名算法，应与后台下单时的值一致
     *
     * 可选值：
     * - 'MD5': 仅在 v2 版本接口适用;
     * - 'HMAC-SHA256': 仅在 v2 版本接口适用;
     * - 'RSA': 仅在 v3 版本接口适用; */
    signType: 'RSA',
    /** 接口调用成功的回调函数 */
    success: () => {
      // 关闭当前页，再跳转支付结果页
      void redirectToOrderPayment(tradeID)
    },
  })
  // #endif

  // #ifdef H5 || APP-PLUS
  const h5Res = await defPayService.H5Pay({ trade_id: tradeID })
  openH5PayUrl(h5Res.h5_url)
  // #endif
}

// 确认收货
const onOrderConfirm = (id: number) => {
  uni.showModal({
    content: '为保障您的权益，请收到货并确认无误后，再确认收货',
    confirmColor: '#27BA9B',
    success: async (res) => {
      if (res.confirm) {
        await defOrderService.ReceiveOrderInfo({
          order_id: id,
        })
        await uni.showToast({ icon: 'success', title: '确认收货成功' })
        // 确认成功，更新为待评价
        updateStatusById(id, OrderInfoStatus.WAIT_REVIEW_OIS)
      }
    },
  })
}

// 打开取消订单弹窗
const onOpenPopup = async (order: OrderInfo) => {
  // 确保数据已加载
  if (cancelReasonList.value.length === 0) {
    await getDictData()
  }
  orderItem.value = order
  dialogTitle.value = '订单取消'
  tips.value = '请选择订单取消的原因：'
  reasonList.value = cancelReasonList.value
  popup.value?.open!()
}

// 关闭取消订单弹窗
const onClosePopup = () => {
  orderItem.value = undefined
  dialogTitle.value = ''
  tips.value = ''
  reasonList.value = []
  reason.value = ''
  // 关闭弹窗
  popup.value?.close!()
}

// 取消订单
const onConfirmPopup = async () => {
  if (!orderItem.value) {
    return uni.showToast({ icon: 'none', title: '请选择订单' })
  }
  if (!reason.value) {
    return uni.showToast({
      icon: 'none',
      title: '请选择订单取消的原因',
    })
  }
  await defOrderService.CancelOrderInfo({
    trade_id: orderItem.value.trade_id,
    reason: Number(reason.value),
  })
  await uni.showToast({
    icon: 'none',
    title: '订单取消成功',
  })
  orderItem.value.trade_status = OrderTradeStatus.CLOSED_OTS
  orderItem.value.status = OrderInfoStatus.CANCELED_OIS

  // 关闭弹窗
  onClosePopup()
}

const onOpenRefundPopup = (order: OrderInfo) => {
  refundPopup.value?.open(order)
}

const onRefundSuccess = (order_id: number) => {
  const order = orderInfoList.value.find((item) => item.id === order_id)
  if (order) {
    order.refund_status = OrderRefundStatus.PROCESSING_ORS
  }
}

// 删除交易聚合记录或门店子订单
const onOrderDelete = (order: OrderInfo) => {
  uni.showModal({
    content: '你确定要删除该订单？',
    confirmColor: '#27BA9B',
    success: async (res) => {
      if (res.confirm) {
        if (order.is_trade) {
          await defOrderService.DeleteOrderTrade({ trade_id: order.trade_id })
        } else {
          await defOrderService.DeleteOrderInfo({ id: order.id })
        }
        // 删除成功，界面中删除订单
        const index = orderInfoList.value.indexOf(order)
        if (index >= 0) {
          orderInfoList.value.splice(index, 1)
        }
      }
    },
  })
}

// 更新状态的函数
const updateStatusById = (id: number, status: OrderInfoStatus): void => {
  const index = orderInfoList.value.findIndex((v) => v.id === id)
  if (index < 0) {
    console.error(`未找到 ID 为 ${id} 的订单`)
  } else {
    orderInfoList.value[index].status = status
  }
}

// 是否分页结束
const isFinish = ref(false)
// 是否触发下拉刷新
const isTriggered = ref(false)
// 空状态只在当前状态列表真正无订单时展示，避免误显示“没有更多数据”。
const isEmpty = computed(
  () => isFinish.value && orderInfoList.value.length === 0 && !isLoading.value,
)
const emptyText = computed(() => {
  if (Object.keys(props.filter).length === 0) {
    return '暂无订单，去首页挑选好货吧'
  }
  return `暂无${props.title}订单`
})

const emptyImage = computed(() => {
  if (props.filter.trade_status === OrderTradeStatus.PENDING_PAYMENT_OTS) {
    return '/static/images/empty_payment.png'
  }
  if (
    props.filter.status === OrderInfoStatus.WAIT_SHIPMENT_OIS ||
    props.filter.status === OrderInfoStatus.SHIPPED_OIS
  ) {
    return '/static/images/empty_delivery.png'
  }
  if (props.filter.status === OrderInfoStatus.WAIT_REVIEW_OIS) {
    return '/static/images/empty_comment.png'
  }
  if (props.filter.has_refund || props.filter.refund_status) {
    return '/static/images/empty_after_sale.png'
  }
  return '/static/images/empty_order.png'
})
const showListFooter = computed(() => {
  if (isFinish.value) {
    return orderInfoList.value.length > 0
  }
  return isLoading.value || orderInfoList.value.length > 0
})
const listFooterText = computed(() => (isFinish.value ? '没有更多数据~' : '正在加载...'))
// 自定义下拉刷新被触发
const onRefresherRefresh = async () => {
  // 开始动画
  isTriggered.value = true
  // 重置数据
  queryParams.page_num = 1
  orderInfoList.value = []
  isFinish.value = false
  // 加载数据
  await getUserOrderData()
  // 关闭动画
  isTriggered.value = false
}
</script>

<template>
  <scroll-view
    enable-back-to-top
    scroll-y
    class="orders"
    refresher-enabled
    :refresher-triggered="isTriggered"
    @refresherrefresh="onRefresherRefresh"
    @scrolltolower="getUserOrderData"
  >
    <view
      v-for="order in orderInfoList"
      :key="order.is_trade ? `trade-${order.trade_id}` : `order-${order.id}`"
      class="card"
    >
      <!-- 订单信息 -->
      <view class="status">
        <view class="order-number">
          {{ order.is_trade ? '交易单' : '订单' }}
          {{ order.is_trade ? order.trade_no : order.order_no }}
        </view>
        <view class="status-info">
          <text class="date" v-if="order.cancel_time">{{ order.cancel_time }}</text>
          <text>{{ getOrderDisplayStatus(order) }}</text>
          <text v-if="canDeleteOrder(order)" class="icon-delete" @tap="onOrderDelete(order)"></text>
        </view>
      </view>

      <!-- 聚合交易包含多个门店，门店子订单只包含当前门店。 -->
      <view
        v-for="group in order.order_goods_stores"
        :key="`${order.trade_id}-${group.store?.id}`"
        class="store-group"
      >
        <navigator
          v-if="group.store?.id"
          class="store-info"
          :url="tenantStoreUrl(group.store.id)"
          hover-class="none"
        >
          <image
            v-if="group.store.logo"
            class="store-logo"
            :src="formatSrc(group.store.logo)"
            mode="aspectFill"
          />
          <text class="store-name">{{ group.store.name || '门店信息' }}</text>
          <text class="store-arrow">&gt;</text>
        </navigator>
        <view v-else class="store-info">
          <text class="store-name">门店信息</text>
        </view>
        <navigator
          v-for="item in group.goods"
          :key="`${item.goods_id}-${item.sku_code}`"
          class="goods"
          :url="buildOrderDetailUrl(order)"
          hover-class="none"
        >
          <view class="cover">
            <image class="image" mode="aspectFit" :src="formatSrc(item.picture)"></image>
          </view>
          <view class="meta">
            <view class="name ellipsis">{{ item.name }}</view>
            <view class="type">{{ item.spec_item.join('/') }}</view>
          </view>
        </navigator>
      </view>
      <!-- 支付信息 -->
      <view class="payment">
        <text class="quantity">共{{ order.goods_num }}件商品</text>
        <text>实付</text>
        <text class="amount"> <text class="symbol">¥</text>{{ formatPrice(order.pay_money) }}</text>
      </view>
      <!-- 订单操作按钮 -->
      <view class="action">
        <view v-if="isPayableTrade(order)" class="button" @tap="onOpenPopup(order)">
          取消订单
        </view>
        <view v-if="isPayableTrade(order)" class="button primary" @tap="onOrderPay(order.trade_id)"
          >去支付</view
        >
        <navigator
          v-if="!order.is_trade"
          class="button secondary"
          :url="orderCreateUrl({ order_id: order.id })"
          hover-class="none"
        >
          再次购买
        </navigator>
        <view v-if="canRefundOrder(order)" class="button" @tap="onOpenRefundPopup(order)">
          申请退款
        </view>
        <view
          v-if="!order.is_trade && order.status === OrderInfoStatus.SHIPPED_OIS"
          class="button primary"
          @tap="onOrderConfirm(order.id)"
        >
          确认收货
        </view>
        <navigator
          v-if="!order.is_trade && order.status === OrderInfoStatus.WAIT_REVIEW_OIS"
          class="button primary"
          :url="buildOrderCommentWriteUrl(order)"
          hover-class="none"
        >
          去评价
        </navigator>
      </view>
    </view>
    <!-- 当前状态无订单时展示空状态，不再使用分页结束提示代替。 -->
    <EmptyState
      v-if="isEmpty"
      :image="emptyImage"
      :text="emptyText"
      min-height="640rpx"
      padding="150rpx 48rpx 80rpx"
    />
    <!-- 底部提示文字 -->
    <view
      v-if="showListFooter"
      class="loading-text"
      :style="{ paddingBottom: safeAreaInsets?.bottom + 'px' }"
    >
      {{ listFooterText }}
    </view>
  </scroll-view>
  <!-- 取消订单弹窗 -->
  <uni-popup ref="popup" type="bottom" background-color="#fff">
    <view class="popup-root">
      <view class="title">{{ dialogTitle }}</view>
      <view class="description">
        <view class="tips">{{ tips }}</view>
        <view class="cell" v-for="item in reasonList" :key="item.value" @tap="reason = item.value">
          <text class="text">{{ item.label }}</text>
          <text class="icon" :class="{ checked: item.value === reason }"></text>
        </view>
      </view>
      <view class="footer">
        <view class="button" @tap="onClosePopup">取消</view>
        <view class="button primary" @tap="onConfirmPopup">确认</view>
      </view>
    </view>
  </uni-popup>
  <RefundOrderPopup ref="refundPopup" @success="onRefundSuccess" />
</template>

<style lang="scss">
// 订单列表
.orders {
  height: 100%;

  .card {
    min-height: 100rpx;
    padding: 20rpx;
    margin: 20rpx 20rpx 0;
    border-radius: 10rpx;
    background-color: #fff;

    &:last-child {
      padding-bottom: 40rpx;
    }
  }

  .status {
    display: flex;
    align-items: center;
    justify-content: space-between;
    font-size: 28rpx;
    color: #999;
    margin-bottom: 15rpx;

    .store-info {
      display: flex;
      flex: 1;
      min-width: 0;
      align-items: center;
      margin-right: 20rpx;
      color: #444;
    }

    .store-name {
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    .store-arrow {
      flex-shrink: 0;
      margin-left: 8rpx;
      color: #999;
    }

    .status-info {
      display: flex;
      flex-shrink: 0;
      align-items: center;
    }

    .date {
      color: #666;
      margin-right: 16rpx;
    }

    .primary {
      color: #ff9240;
    }

    .icon-delete {
      line-height: 1;
      margin-left: 10rpx;
      padding-left: 10rpx;
      border-left: 1rpx solid #e3e3e3;
    }
  }

  .order-number {
    min-width: 0;
    margin-right: 20rpx;
    overflow: hidden;
    color: #666;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .store-group {
    padding-top: 12rpx;
    border-top: 1rpx solid #eee;
  }

  .store-info {
    display: flex;
    align-items: center;
    min-width: 0;
    height: 64rpx;
    color: #444;
  }

  .store-logo {
    flex-shrink: 0;
    width: 36rpx;
    height: 36rpx;
    margin-right: 10rpx;
    border-radius: 50%;
  }

  .store-name {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .store-arrow {
    flex-shrink: 0;
    margin-left: 8rpx;
    color: #999;
  }

  .goods {
    display: flex;
    margin-bottom: 20rpx;

    .cover {
      width: 170rpx;
      height: 170rpx;
      margin-right: 20rpx;
      border-radius: 10rpx;
      overflow: hidden;
      position: relative;
      .image {
        width: 170rpx;
        height: 170rpx;
      }
    }

    .quantity {
      position: absolute;
      bottom: 0;
      right: 0;
      line-height: 1;
      padding: 6rpx 4rpx 6rpx 8rpx;
      font-size: 24rpx;
      color: #fff;
      border-radius: 10rpx 0 0 0;
      background-color: rgba(0, 0, 0, 0.6);
    }

    .meta {
      flex: 1;
      display: flex;
      flex-direction: column;
      justify-content: center;
    }

    .name {
      height: 80rpx;
      font-size: 26rpx;
      color: #444;
    }

    .type {
      line-height: 1.8;
      padding: 0 15rpx;
      margin-top: 10rpx;
      font-size: 24rpx;
      align-self: flex-start;
      border-radius: 4rpx;
      color: #888;
      background-color: #f7f7f8;
    }

    .more {
      flex: 1;
      display: flex;
      align-items: center;
      justify-content: center;
      font-size: 22rpx;
      color: #333;
    }
  }

  .payment {
    display: flex;
    justify-content: flex-end;
    align-items: center;
    line-height: 1;
    padding: 20rpx 0;
    text-align: right;
    color: #999;
    font-size: 28rpx;
    border-bottom: 1rpx solid #eee;

    .quantity {
      font-size: 24rpx;
      margin-right: 16rpx;
    }

    .amount {
      color: #444;
      margin-left: 6rpx;
    }

    .symbol {
      font-size: 20rpx;
    }
  }

  .action {
    display: flex;
    flex-wrap: wrap;
    justify-content: flex-end;
    align-items: center;
    padding-top: 20rpx;

    .button {
      width: 180rpx;
      height: 60rpx;
      display: flex;
      justify-content: center;
      align-items: center;
      margin-left: 20rpx;
      border-radius: 60rpx;
      border: 1rpx solid #ccc;
      font-size: 26rpx;
      color: #444;
    }

    .secondary {
      color: #27ba9b;
      border-color: #27ba9b;
    }

    .primary {
      color: #fff;
      background-color: #27ba9b;
      border-color: #27ba9b;
    }
  }

  .loading-text {
    text-align: center;
    font-size: 28rpx;
    color: #666;
    padding: 20rpx 0;
  }
}

.popup-root {
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
