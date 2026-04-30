<script setup lang="ts">
import { useGuessList } from '@/composables'
import { defOrderService } from '@/api/app/order_info.ts'
import type { OrderInfoResponse } from '@/rpc/app/v1/order_info'
import { onLoad, onReady } from '@dcloudio/uni-app'
import { ref } from 'vue'
import PageSkeleton from './components/PageSkeleton.vue'
import { OrderCancelReason, OrderStatus, RecommendScene } from '@/rpc/common/v1/enum'
import type { BaseDictForm_DictItem } from '@/rpc/app/v1/base_dict'
import { defBaseDictService } from '@/api/app/base_dict'
import { defPayService } from '@/api/app/pay'
import { formatPrice, formatSrc } from '@/utils'
import {
  goodsDetailUrl,
  homeTabPage,
  orderCommentWriteUrl,
  orderCreateUrl,
  orderListUrl,
  redirectToOrderPayment,
} from '@/utils/navigation'
import RefundOrderPopup from '../components/RefundOrderPopup.vue'
// 获取屏幕边界到安全区域距离
const { safeAreaInsets } = uni.getSystemInfoSync()
// 猜你喜欢
const { guessRef, onScrollToLower } = useGuessList()
// 弹出层组件
const popup = ref<UniHelper.UniPopupInstance>()
const refundPopup = ref<InstanceType<typeof RefundOrderPopup>>()
// 原因列表
const reasonList = ref<BaseDictForm_DictItem[]>([])
// 取消原因列表
const cancelReasonList = ref<BaseDictForm_DictItem[]>([])
const orderStatusMap: Map<number, string> = new Map()
// 订单取消原因
const reason = ref('')
// 标题
const title = ref('')
// tips
const tips = ref('')

const buildGoodsDetailUrl = (
  goods_id: number,
  query: { scene?: RecommendScene; request_id?: number; index?: number },
) => {
  return goodsDetailUrl({
    id: goods_id,
    scene: query.scene,
    request_id: query.request_id,
    index: query.index,
  })
}

const buildOrderCommentWriteUrl = () => {
  const firstGoods = orderData.value?.order?.goods?.[0]
  return orderCommentWriteUrl({
    order_id: order_id.value,
    goods_id: firstGoods?.goods_id,
    goods_name: firstGoods?.name,
    goods_picture: firstGoods?.picture ? formatSrc(firstGoods.picture) : undefined,
    sku_code: firstGoods?.sku_code,
    sku_desc: firstGoods?.spec_item?.join(' / '),
  })
}

const getOrderStatusText = (status?: OrderStatus) => {
  if (status === OrderStatus.WAIT_REVIEW) {
    return '待评价'
  }
  if (status === OrderStatus.COMPLETED) {
    return '已完成'
  }
  return status === undefined ? '' : orderStatusMap.get(status)
}

// 复制内容
const onCopy = (id: string) => {
  // 设置系统剪贴板的内容
  uni.setClipboardData({ data: id })
}
// 获取页面参数
const query = defineProps<{
  id: string | number
  internal: boolean
}>()

const order_id = ref(0)

// 获取页面栈
const pages = getCurrentPages()

// 基于小程序的 Page 类型扩展 uni-app 的 Page
type PageInstance = Page.PageInstance & WechatMiniprogram.Page.InstanceMethods<any>

// #ifdef MP-WEIXIN
// 获取当前页面实例，数组最后一项
const pageInstance = pages.at(-1) as PageInstance

// 页面渲染完毕，绑定动画效果
onReady(() => {
  // 动画效果,导航栏背景色
  pageInstance.animate(
    '.navbar',
    [{ backgroundColor: 'transparent' }, { backgroundColor: '#f8f8f8' }],
    1000,
    {
      scrollSource: '#scroller',
      timeRange: 1000,
      startScrollOffset: 0,
      endScrollOffset: 50,
    },
  )
  // 动画效果,导航栏标题
  pageInstance.animate('.navbar .title', [{ color: 'transparent' }, { color: '#000' }], 1000, {
    scrollSource: '#scroller',
    timeRange: 1000,
    startScrollOffset: 0,
    endScrollOffset: 50,
  })
  // 动画效果,导航栏返回按钮
  pageInstance.animate('.navbar .back', [{ color: '#fff' }, { color: '#000' }], 1000, {
    scrollSource: '#scroller',
    timeRange: 1000,
    startScrollOffset: 0,
    endScrollOffset: 50,
  })
})
// #endif

// 获取订单详情
const orderData = ref<OrderInfoResponse>()
const getUserOrderById = async () => {
  if (!query.internal) {
    const res = await defOrderService.GetOrderInfoIdByOrderNo({
      order_no: String(query.id),
    })
    order_id.value = res.order_id
  } else {
    order_id.value = Number(query.id)
  }
  orderData.value = await defOrderService.GetOrderInfoById({
    id: order_id.value,
  })
}
const getDictData = async () => {
  const orderStatusCode = 'order_status'
  const cancelReasonCode = 'order_cancel_reason'
  // 新协议按字典编码单查，这里并发加载详情页依赖的字典。
  const [orderStatusDict, cancelReasonDict] = await Promise.all([
    defBaseDictService.GetBaseDict({ value: orderStatusCode }),
    defBaseDictService.GetBaseDict({ value: cancelReasonCode }),
  ])
  orderStatusDict.items.map((dictItem) => {
    orderStatusMap.set(Number(dictItem.value), dictItem.label)
  })
  cancelReasonList.value = cancelReasonDict.items || []
}
onLoad(() => {
  Promise.all([getUserOrderById(), getDictData()])
})

const onTimeUpFlag = ref(false)

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

// 倒计时结束事件
const onTimeUp = async () => {
  // 添加状态检查：只有待支付状态才执行取消
  if (orderData.value?.order?.status !== OrderStatus.CREATED) return
  // 订单真实 ID 未就绪时不触发取消，避免按订单号误取消
  if (!order_id.value) return
  // 添加加载状态锁
  if (onTimeUpFlag.value) return
  onTimeUpFlag.value = true
  // 修改订单状态为已取消
  try {
    await defOrderService.CancelOrderInfo({
      order_id: order_id.value,
      reason: OrderCancelReason.UNKNOWN_OCR,
    })
    await getUserOrderById()
  } finally {
    onTimeUpFlag.value = false
  }
}

// 发起订单支付
const onOrderPay = async () => {
  // #ifdef MP-WEIXIN
  // 正式环境微信支付
  const jsapiRes = await defPayService.JsapiPay({ order_id: order_id.value })
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
      void redirectToOrderPayment(order_id.value)
    },
  })
  // #endif

  // #ifdef H5 || APP-PLUS
  const h5Res = await defPayService.H5Pay({ order_id: order_id.value })
  openH5PayUrl(h5Res.h5_url)
  // #endif
}

// 确认收货
const onOrderConfirm = () => {
  // 二次确认弹窗
  uni.showModal({
    content: '为保障您的权益，请收到货并确认无误后，再确认收货',
    confirmColor: '#27BA9B',
    success: async (success) => {
      if (success.confirm) {
        await defOrderService.ReceiveOrderInfo({
          order_id: order_id.value,
        })
        await getUserOrderById()
      }
    },
  })
}
// 删除订单
const onOrderDelete = () => {
  // 二次确认
  uni.showModal({
    content: '是否删除订单',
    confirmColor: '#27BA9B',
    success: async (success) => {
      if (success.confirm) {
        await defOrderService.DeleteOrderInfo({ id: order_id.value })
        uni.redirectTo({ url: orderListUrl(0) })
      }
    },
  })
}

// 打开取消订单弹窗
const onOpenPopup = () => {
  title.value = '订单取消'
  tips.value = '请选择订单取消的原因：'
  reasonList.value = cancelReasonList.value
  popup.value?.open!()
}

// 关闭取消订单弹窗
const onClosePopup = () => {
  title.value = ''
  tips.value = ''
  reasonList.value = []
  reason.value = ''
  // 关闭弹窗
  popup.value?.close!()
}

// 取消订单
const onConfirmPopup = async () => {
  if (!reason.value) {
    return uni.showToast({
      icon: 'none',
      title: '请选择订单取消的原因',
    })
  }
  await defOrderService.CancelOrderInfo({
    order_id: order_id.value,
    reason: Number(reason.value),
  })
  await uni.showToast({
    icon: 'none',
    title: '订单取消成功',
  })
  await getUserOrderById()
  // 关闭弹窗
  onClosePopup()
}

const onOpenRefundPopup = () => {
  if (!orderData.value?.order) {
    return
  }
  refundPopup.value?.open(orderData.value.order)
}

const onRefundSuccess = async () => {
  await getUserOrderById()
}
</script>

<template>
  <!-- 自定义导航栏: 默认透明不可见, scroll-view 滚动到 50 时展示 -->
  <view class="navbar" :style="{ paddingTop: safeAreaInsets?.top + 'px' }">
    <view class="wrap">
      <navigator v-if="pages.length > 1" open-type="navigateBack" class="back icon-left" />
      <navigator v-else :url="homeTabPage" open-type="switchTab" class="back icon-home" />
      <view class="title">订单详情</view>
    </view>
  </view>
  <scroll-view
    id="scroller"
    enable-back-to-top
    scroll-y
    class="viewport"
    @scrolltolower="onScrollToLower"
  >
    <template v-if="orderData">
      <!-- 订单状态 -->
      <view class="overview" :style="{ paddingTop: safeAreaInsets!.top + 20 + 'px' }">
        <!-- 待付款状态:展示倒计时 -->
        <template v-if="orderData.order!.status === OrderStatus.CREATED">
          <view class="status icon-clock">等待付款</view>
          <view class="tips">
            <text class="money">应付金额: ¥ {{ formatPrice(orderData.order!.pay_money) }}</text>
            <text class="time">支付剩余</text>
            <uni-countdown
              :second="orderData.countdown"
              color="#fff"
              splitor-color="#fff"
              :show-day="false"
              :show-colon="false"
              @timeup="onTimeUp"
            />
          </view>
          <view class="button" @tap="onOrderPay">去支付</view>
        </template>
        <!-- 其他订单状态:展示再次购买按钮 -->
        <template v-else>
          <!-- 订单状态文字 -->
          <view class="status"> {{ getOrderStatusText(orderData.order!.status) }}</view>
          <view class="button-group">
            <navigator
              class="button"
              :url="orderCreateUrl({ order_id: query.id })"
              hover-class="none"
            >
              再次购买
            </navigator>
            <!-- 待收货状态: 展示确认收货按钮 -->
            <view
              v-if="orderData.order!.status === OrderStatus.SHIPPED"
              class="button"
              @tap="onOrderConfirm"
            >
              确认收货
            </view>
          </view>
        </template>
      </view>
      <!-- 配送状态 -->
      <view class="shipment">
        <!-- 订单物流信息 -->
        <view v-for="(item, idx) in orderData.logistics?.detail" :key="idx" class="item">
          <view class="message">
            {{ item.text }}
          </view>
          <view class="date"> {{ item.time }} </view>
        </view>
        <!-- 用户收货地址 -->
        <view class="locate">
          <view class="user">
            {{ orderData.address!.receiver }} {{ orderData.address!.contact }}</view
          >
          <view class="address">
            {{ orderData.address!.address.join('-') }}-{{ orderData.address!.detail }}</view
          >
        </view>
      </view>

      <!-- 商品信息 -->
      <view class="goods">
        <view class="item">
          <navigator
            v-for="item in orderData.order!.goods"
            :key="item.goods_id"
            class="navigator"
            :url="
              buildGoodsDetailUrl(item.goods_id, {
                scene: item.recommend_context?.scene,
                request_id: item.recommend_context?.request_id,
                index: item.recommend_context?.position,
              })
            "
            hover-class="none"
          >
            <image class="cover" :src="formatSrc(item.picture)" />
            <view class="meta">
              <view class="name ellipsis">{{ item.name }}</view>
              <view class="type">{{ item.spec_item.join('/') }}</view>
              <view class="price">
                <view class="actual">
                  <text class="symbol">¥</text>
                  <text>{{ formatPrice(item.pay_price) }}</text>
                </view>
              </view>
              <view class="quantity">x{{ item.num }}</view>
            </view>
          </navigator>
          <!-- 待评价状态:展示按钮 -->
          <view v-if="orderData.order!.status === OrderStatus.WAIT_REVIEW" class="action">
            <view class="button primary" @tap="onOpenRefundPopup">申请售后</view>
            <navigator :url="buildOrderCommentWriteUrl()" class="button"> 去评价 </navigator>
          </view>
        </view>
        <!-- 合计 -->
        <view class="total">
          <view class="row">
            <view class="text">商品总价: </view>
            <view class="symbol">{{ formatPrice(orderData.order!.total_money) }}</view>
          </view>
          <view class="row">
            <view class="text">运费: </view>
            <view class="symbol">{{ formatPrice(orderData.order!.post_fee) }}</view>
          </view>
          <view class="row">
            <view class="text">应付金额: </view>
            <view class="symbol primary">{{ formatPrice(orderData.order!.pay_money) }}</view>
          </view>
        </view>
      </view>

      <!-- 订单信息 -->
      <view class="detail">
        <view class="title">订单信息</view>
        <view class="row">
          <view class="item">
            订单编号: {{ orderData.order!.order_no }}
            <text class="copy" @tap="onCopy(orderData.order!.order_no)">复制</text>
          </view>
          <view class="item">下单时间: {{ orderData.order!.created_at }}</view>
          <view v-if="orderData.order!.payment_time" class="item"
            >支付时间: {{ orderData.order!.payment_time }}</view
          >
          <view v-if="orderData.order!.cancel_time" class="item"
            >取消时间: {{ orderData.order!.cancel_time }}</view
          >
          <view v-if="orderData.order!.refund_time" class="item"
            >退款时间: {{ orderData.order!.refund_time }}</view
          >
          <view class="item">订单备注: {{ orderData.order!.remark }}</view>
        </view>
      </view>

      <!-- 猜你喜欢 -->
      <XtxGuess
        ref="guessRef"
        title="买过这单的人还会买"
        :scene="RecommendScene.ORDER_DETAIL"
        :order-id="order_id"
      />

      <!-- 底部操作栏 -->
      <view class="toolbar-height" :style="{ paddingBottom: safeAreaInsets?.bottom + 'px' }" />
      <view class="toolbar" :style="{ paddingBottom: safeAreaInsets?.bottom + 'px' }">
        <view
          v-if="orderData.order!.status === OrderStatus.CREATED"
          class="button"
          @tap="onOpenPopup"
        >
          取消订单
        </view>
        <view
          v-if="orderData.order!.status === OrderStatus.CREATED"
          class="button primary"
          @tap="onOrderPay"
        >
          去支付
        </view>
        <navigator
          v-if="orderData.order!.status !== OrderStatus.CREATED"
          class="button secondary"
          :url="orderCreateUrl({ order_id: query.id })"
          hover-class="none"
        >
          再次购买
        </navigator>
        <view
          v-if="orderData.order!.status === OrderStatus.PAID"
          class="button"
          @tap="onOpenRefundPopup"
        >
          申请退款
        </view>
        <view
          v-if="orderData.order!.status === OrderStatus.SHIPPED"
          class="button primary"
          @tap="onOrderConfirm"
        >
          确认收货
        </view>
        <!-- 已完成/退款售后/已取消状态允许删除，待评价不能删除。 -->
        <view
          v-if="
            orderData.order!.status === OrderStatus.COMPLETED ||
            orderData.order!.status === OrderStatus.REFUNDING ||
            orderData.order!.status === OrderStatus.CANCELED
          "
          class="button delete"
          @tap="onOrderDelete"
        >
          删除订单
        </view>
      </view>
    </template>
    <template v-else>
      <!-- 骨架屏组件 -->
      <PageSkeleton />
    </template>
  </scroll-view>
  <!-- 取消订单弹窗 -->
  <uni-popup ref="popup" type="bottom" background-color="#fff">
    <view class="popup-root">
      <view class="title">{{ title }}</view>
      <view class="description">
        <view class="tips">{{ tips }}</view>
        <view v-for="item in reasonList" :key="item.value" class="cell" @tap="reason = item.value">
          <text class="text">{{ item.label }}</text>
          <text class="icon" :class="{ checked: item.value === reason }" />
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
page {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

.navbar {
  width: 750rpx;
  color: #000;
  position: fixed;
  top: 0;
  left: 0;
  z-index: 9;
  /* background-color: #f8f8f8; */
  background-color: transparent;

  .wrap {
    position: relative;

    .title {
      height: 44px;
      display: flex;
      justify-content: center;
      align-items: center;
      font-size: 32rpx;
      /* color: #000; */
      color: transparent;
    }

    .back {
      position: absolute;
      left: 0;
      height: 44px;
      width: 44px;
      font-size: 44rpx;
      display: flex;
      align-items: center;
      justify-content: center;
      /* color: #000; */
      color: #fff;
    }
  }
}

.viewport {
  background-color: #f7f7f8;
}

.overview {
  display: flex;
  flex-direction: column;
  align-items: center;

  line-height: 1;
  padding-bottom: 30rpx;
  color: #fff;
  background-image: url(@/static/images/order_bg.png);
  background-size: cover;

  .status {
    font-size: 36rpx;
  }

  .status::before {
    margin-right: 6rpx;
    font-weight: 500;
  }

  .tips {
    margin: 30rpx 0;
    display: flex;
    font-size: 14px;
    align-items: center;

    .money {
      margin-right: 30rpx;
    }
  }

  .button-group {
    margin-top: 30rpx;
    display: flex;
    justify-content: center;
    align-items: center;
  }

  .button {
    width: 260rpx;
    height: 64rpx;
    margin: 0 10rpx;
    text-align: center;
    line-height: 64rpx;
    font-size: 28rpx;
    color: #27ba9b;
    border-radius: 68rpx;
    background-color: #fff;
  }
}

.shipment {
  line-height: 1.4;
  padding: 0 20rpx;
  margin: 20rpx 20rpx 0;
  border-radius: 10rpx;
  background-color: #fff;

  .locate,
  .item {
    min-height: 120rpx;
    padding: 30rpx 30rpx 25rpx 75rpx;
    background-size: 50rpx;
    background-repeat: no-repeat;
    background-position: 6rpx center;
  }

  .locate {
    background-image: url(@/static/images/locate.png);

    .user {
      font-size: 26rpx;
      color: #444;
    }

    .address {
      font-size: 24rpx;
      color: #666;
    }
  }

  .item {
    background-image: url(@/static/images/car.png);
    border-bottom: 1rpx solid #eee;
    position: relative;

    .message {
      font-size: 26rpx;
      color: #444;
    }

    .date {
      font-size: 24rpx;
      color: #666;
    }
  }
}

.goods {
  margin: 20rpx 20rpx 0;
  padding: 0 20rpx;
  border-radius: 10rpx;
  background-color: #fff;

  .item {
    padding: 30rpx 0;
    border-bottom: 1rpx solid #eee;

    .navigator {
      display: flex;
      margin: 20rpx 0;
    }

    .cover {
      width: 170rpx;
      height: 170rpx;
      border-radius: 10rpx;
      margin-right: 20rpx;
    }

    .meta {
      flex: 1;
      display: flex;
      flex-direction: column;
      justify-content: center;
      position: relative;
    }

    .name {
      height: 80rpx;
      font-size: 26rpx;
      color: #444;
    }

    .type {
      line-height: 1.8;
      padding: 0 15rpx;
      margin-top: 6rpx;
      font-size: 24rpx;
      align-self: flex-start;
      border-radius: 4rpx;
      color: #888;
      background-color: #f7f7f8;
    }

    .price {
      display: flex;
      margin-top: 6rpx;
      font-size: 24rpx;
    }

    .symbol {
      font-size: 20rpx;
    }

    .original {
      color: #999;
      text-decoration: line-through;
    }

    .actual {
      margin-left: 10rpx;
      color: #444;
    }

    .text {
      font-size: 22rpx;
    }

    .quantity {
      position: absolute;
      bottom: 0;
      right: 0;
      font-size: 24rpx;
      color: #444;
    }

    .action {
      display: flex;
      flex-direction: row-reverse;
      justify-content: flex-start;
      padding: 30rpx 0 0;

      .button {
        width: 200rpx;
        height: 60rpx;
        text-align: center;
        justify-content: center;
        line-height: 60rpx;
        margin-left: 20rpx;
        border-radius: 60rpx;
        border: 1rpx solid #ccc;
        font-size: 26rpx;
        color: #444;
      }

      .primary {
        color: #27ba9b;
        border-color: #27ba9b;
      }
    }
  }

  .total {
    line-height: 1;
    font-size: 26rpx;
    padding: 20rpx 0;
    color: #666;

    .row {
      display: flex;
      align-items: center;
      justify-content: space-between;
      padding: 10rpx 0;
    }

    .symbol::before {
      content: '¥';
      font-size: 80%;
      margin-right: 3rpx;
    }

    .primary {
      color: #cf4444;
      font-size: 36rpx;
    }
  }
}

.detail {
  line-height: 1;
  padding: 30rpx 20rpx 0;
  margin: 20rpx 20rpx 0;
  font-size: 26rpx;
  color: #666;
  border-radius: 10rpx;
  background-color: #fff;

  .title {
    font-size: 30rpx;
    color: #444;
  }

  .row {
    padding: 20rpx 0;

    .item {
      padding: 10rpx 0;
      display: flex;
      align-items: center;
    }

    .copy {
      border-radius: 20rpx;
      font-size: 20rpx;
      border: 1px solid #ccc;
      padding: 5rpx 10rpx;
      margin-left: 10rpx;
    }
  }
}

.toolbar-height {
  height: 100rpx;
  box-sizing: content-box;
}

.toolbar {
  position: fixed;
  left: 0;
  right: 0;
  bottom: calc(var(--window-bottom));
  z-index: 1;

  height: 100rpx;
  padding: 0 20rpx;
  display: flex;
  align-items: center;
  flex-direction: row-reverse;
  border-top: 1rpx solid #ededed;
  border-bottom: 1rpx solid #ededed;
  background-color: #fff;
  box-sizing: content-box;

  .button {
    display: flex;
    justify-content: center;
    align-items: center;

    width: 200rpx;
    height: 72rpx;
    margin-left: 15rpx;
    font-size: 26rpx;
    border-radius: 72rpx;
    border: 1rpx solid #ccc;
    color: #444;
  }

  .delete {
    order: 4;
    color: #cf4444;
  }

  .button {
    order: 3;
  }

  .secondary {
    order: 2;
    color: #27ba9b;
    border-color: #27ba9b;
  }

  .primary {
    order: 1;
    color: #fff;
    background-color: #27ba9b;
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
