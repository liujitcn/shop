<script setup lang="ts">
import { defRecommendService } from '@/api/app/recommend'
import { defOrderService } from '@/api/app/order_info.ts'
import { useAddressStore, useRecommendStore } from '@/stores'
import type {
  ConfirmOrderInfoResponse,
  CreateOrderInfoGoods,
  OrderGoods,
} from '@/rpc/app/order_info'
import type { RecommendContext } from '@/rpc/app/recommend'
import { onLoad } from '@dcloudio/uni-app'
import { computed, ref } from 'vue'
import type { UserAddress } from '@/rpc/app/user_address'
import { defUserAddressService } from '@/api/app/user_address'
import type { ListBaseDictResponse_DictItem } from '@/rpc/app/base_dict'
import { defBaseDictService } from '@/api/app/base_dict'
import { formatSrc, formatPrice } from '@/utils'
import { RecommendGoodsActionType } from '@/rpc/common/enum'

const addressStore = useAddressStore()
const recommendStore = useRecommendStore()

// 获取屏幕边界到安全区域距离
const { safeAreaInsets } = uni.getSystemInfoSync()
// 订单备注
const buyerMessage = ref('')
// 支付方式
const payTypeList = ref<ListBaseDictResponse_DictItem[]>([])
// 当前支付方式下标
const payTypeActiveIndex = ref(0)
// 当前支付方式
const activePayType = computed(() => payTypeList.value[payTypeActiveIndex.value])
// 修改支付方式
const onChangePayType: UniHelper.SelectorPickerOnChange = (ev) => {
  payTypeActiveIndex.value = ev.detail.value
}
// 支付渠道
const payChannelList = ref<ListBaseDictResponse_DictItem[]>([])
// 当前支付渠道下标
const payChannelActiveIndex = ref(0)
// 当前支付渠道
const activePayChannel = computed(() => payChannelList.value[payChannelActiveIndex.value])
// 修改支付渠道
const onChangePayChannel: UniHelper.SelectorPickerOnChange = (ev) => {
  payChannelActiveIndex.value = ev.detail.value
}

// 配送时间
const deliveryList = ref<ListBaseDictResponse_DictItem[]>([])
// 当前配送时间下标
const deliveryActiveIndex = ref(0)
// 当前配送时间
const activeDelivery = computed(() => deliveryList.value[deliveryActiveIndex.value])
// 修改配送时间
const onChangeDelivery: UniHelper.SelectorPickerOnChange = (ev) => {
  deliveryActiveIndex.value = ev.detail.value
}

// 页面参数
const query = defineProps<{
  goodsId?: string
  skuCode?: string
  num?: string
  orderId?: string
  scene?: string
  requestId?: string
  index?: string
}>()
const recommendContext = {
  scene: query.scene,
  requestId: query.requestId,
  position: query.index,
} as unknown as RecommendContext

/** 构建订单提交商品项。 */
const buildOrderRequestGoods = (item: OrderGoods): CreateOrderInfoGoods => {
  return {
    goodsId: item.goodsId,
    skuCode: item.skuCode,
    num: item.num,
    recommendContext: {
      scene: item.scene,
      requestId: item.requestId,
      position: item.position,
    } as unknown as RecommendContext,
  }
}

// 获取订单信息
const orderPre = ref<ConfirmOrderInfoResponse>()
const getUserOrderPreData = async () => {
  if (query.goodsId && query.skuCode && query.num) {
    orderPre.value = await defOrderService.OrderInfoBuy({
      goodsId: Number(query.goodsId),
      skuCode: query.skuCode,
      num: Number(query.num),
      recommendContext,
    })
  } else if (query.orderId) {
    // 再次购买
    orderPre.value = await defOrderService.OrderInfoRepurchase({
      orderId: Number(query.orderId),
    })
  } else {
    orderPre.value = await defOrderService.OrderInfoPre({})
  }
}

const addressList = ref<UserAddress[]>([])
const getUserAddressData = async () => {
  const res = await defUserAddressService.ListUserAddress({})
  addressList.value = res.list || []
}

const getDictData = async () => {
  const pay_type = 'order_pay_type'
  const pay_channel = 'order_pay_channel'
  const delivery_time_type = 'order_delivery_time'
  const res = await defBaseDictService.ListBaseDict({
    value: `${pay_type},${pay_channel},${delivery_time_type}`,
  })
  const list = res.list || []
  list.map((item) => {
    switch (item.code) {
      case pay_type:
        payTypeList.value = item.items || []
        break
      case pay_channel:
        payChannelList.value = item.items || []
        break
      case delivery_time_type:
        deliveryList.value = item.items || []
        break
    }
  })
}

onLoad(() => {
  Promise.all([getUserAddressData(), getUserOrderPreData(), getDictData()])
})

// 收货地址
const selectAddress = computed(() => {
  if (addressStore.selectedAddress) {
    return addressStore.selectedAddress
  } else {
    if (addressList.value) {
      return addressList.value.find((v) => v.isDefault)
    } else {
      return undefined
    }
  }
})

// 提交订单
const onOrderSubmit = async () => {
  // 没有收货地址提醒
  if (!selectAddress.value) {
    return uni.showToast({ icon: 'none', title: '请选择收货地址' })
  }
  if (!activePayType.value?.value) {
    return uni.showToast({ icon: 'none', title: '请选择支付方式' })
  }
  if (Number(activePayType.value?.value) === 1 && !activePayChannel.value.value) {
    return uni.showToast({ icon: 'none', title: '请选择支付渠道' })
  }
  if (!activeDelivery.value?.value) {
    return uni.showToast({ icon: 'none', title: '请选择配送时间类型' })
  }
  const requestGoods = orderPre.value!.goods.map((item) => buildOrderRequestGoods(item))
  // 发送请求
  const res = await defOrderService.CreateOrderInfo({
    /** 地址id */
    addressId: selectAddress.value!.id,
    /** 是否清空购物车 */
    clearCart: orderPre.value?.clearCart || false,
    /** 支付方式：枚举【OrderPayType】 */
    payType: Number(activePayType.value.value),
    /** 支付渠道：枚举【OrderPayChannel】 */
    payChannel: Number(activePayChannel.value.value),
    /** 配送时间：枚举【OrderDeliveryTime】 */
    deliveryTime: Number(activeDelivery.value.value),
    /** 订单备注 */
    remark: buyerMessage.value,
    /** 商品信息 */
    goods: requestGoods,
  })
  const goodsItems = requestGoods.map((item) => ({
    goodsId: item.goodsId,
    goodsNum: item.num || 1,
    recommendContext: item.recommendContext,
  }))
  await recommendStore.getAnonymousId()
  await defRecommendService.RecommendGoodsActionReport({
    eventType: RecommendGoodsActionType.RECOMMEND_GOODS_ACTION_ORDER_CREATE,
    goodsItems,
  })
  // 关闭当前页面，跳转到订单详情，传递订单id
  if (Number(activePayType.value.value) === 2) {
    await uni.redirectTo({ url: `/pagesOrder/payment/payment?id=${res.orderId}` })
  } else {
    await uni.redirectTo({ url: `/pagesOrder/detail/detail?id=${res.orderId}&internal=true` })
  }
}

// 计算提交按钮是否应置灰
const onOrderSubmitOk = computed(() => {
  if (!selectAddress.value?.id) {
    return true
  }
  if (!activePayType.value?.value) {
    return true
  }
  if (!activeDelivery.value?.value) {
    return true
  }
  if (Number(activePayType.value?.value) === 1 && !activePayChannel.value?.value) {
    return true
  }
  return false
})
</script>

<template>
  <scroll-view enable-back-to-top scroll-y class="viewport">
    <!-- 收货地址 -->
    <navigator
      v-if="selectAddress"
      class="shipment"
      hover-class="none"
      url="/pagesMember/address/address?from=order"
    >
      <view class="user"> {{ selectAddress.receiver }} {{ selectAddress.contact }} </view>
      <view class="address">
        {{ selectAddress.address.join('-') }}-{{ selectAddress.detail }}
      </view>
      <text class="icon icon-right"></text>
    </navigator>
    <navigator
      v-else
      class="shipment"
      hover-class="none"
      url="/pagesMember/address/address?from=order"
    >
      <view class="address"> 请选择收货地址 </view>
      <text class="icon icon-right"></text>
    </navigator>

    <!-- 商品信息 -->
    <view class="goods" v-if="orderPre?.goods">
      <navigator
        v-for="item in orderPre!.goods"
        :key="item.skuCode"
        :url="
          `/pages/goods/goods?id=${item.goodsId}&scene=${encodeURIComponent(String(item.scene))}&requestId=${encodeURIComponent(item.requestId || '')}&index=${encodeURIComponent(String(item.position))}`
        "
        class="item"
        hover-class="none"
      >
        <image class="picture" :src="formatSrc(item.picture)" />
        <view class="meta">
          <view class="name ellipsis"> {{ item.name }} </view>
          <view class="attrs">{{ item.specItem.join('/') }}</view>
          <view class="prices">
            <view class="pay-price symbol">{{ formatPrice(item.payPrice) }}</view>
            <view class="price symbol">{{ formatPrice(item.price) }}</view>
          </view>
          <view class="count">x{{ item.num }}</view>
        </view>
      </navigator>
    </view>

    <!-- 配送及支付方式 -->
    <view class="related">
      <view class="item">
        <text class="text">配送时间</text>
        <picker :range="deliveryList" range-key="label" @change="onChangeDelivery">
          <view class="icon-fonts picker">{{ activeDelivery?.label }}</view>
        </picker>
      </view>
      <view class="item">
        <text class="text">支付方式</text>
        <picker :range="payTypeList" range-key="label" @change="onChangePayType">
          <view class="icon-fonts picker">{{ activePayType?.label }}</view>
        </picker>
      </view>
      <view class="item" v-if="Number(activePayType?.value) === 1">
        <text class="text">支付渠道</text>
        <picker :range="payChannelList" range-key="label" @change="onChangePayChannel">
          <view class="icon-fonts picker">{{ activePayChannel?.label }}</view>
        </picker>
      </view>
      <view class="item">
        <text class="text">订单备注</text>
        <input
          class="input"
          :cursor-spacing="30"
          placeholder="选题，建议留言前先与商家沟通确认"
          v-model="buyerMessage"
        />
      </view>
    </view>

    <!-- 支付金额 -->
    <view class="settlement" v-if="orderPre?.summary">
      <view class="item">
        <text class="text">商品总价: </text>
        <text class="number symbol">{{ formatPrice(orderPre!.summary!.totalMoney) }}</text>
      </view>
      <view class="item">
        <text class="text">运费: </text>
        <text class="number symbol"> {{ formatPrice(orderPre!.summary?.postFee) }}</text>
      </view>
    </view>
  </scroll-view>

  <!-- 吸底工具栏 -->
  <view class="toolbar" :style="{ paddingBottom: safeAreaInsets!.bottom + 'px' }">
    <view class="total-pay symbol" v-if="orderPre?.summary">
      <text class="number">{{ formatPrice(orderPre!.summary!.payMoney) }}</text>
    </view>
    <view class="button" :class="{ disabled: onOrderSubmitOk }" @tap="onOrderSubmit">
      提交订单
    </view>
  </view>
</template>

<style lang="scss">
page {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
  background-color: #f4f4f4;
}

.symbol::before {
  content: '¥';
  font-size: 80%;
  margin-right: 5rpx;
}

.shipment {
  margin: 20rpx;
  padding: 30rpx 30rpx 30rpx 84rpx;
  font-size: 26rpx;
  border-radius: 10rpx;
  background: url(@/static/images/locate.png) 20rpx center / 50rpx no-repeat #fff;
  position: relative;

  .icon {
    font-size: 36rpx;
    color: #333;
    transform: translateY(-50%);
    position: absolute;
    top: 50%;
    right: 20rpx;
  }

  .user {
    color: #333;
    margin-bottom: 5rpx;
  }

  .address {
    color: #666;
  }
}

.goods {
  margin: 20rpx;
  padding: 0 20rpx;
  border-radius: 10rpx;
  background-color: #fff;

  .item {
    display: flex;
    padding: 30rpx 0;
    border-top: 1rpx solid #eee;

    &:first-child {
      border-top: none;
    }

    .picture {
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

    .attrs {
      line-height: 1.8;
      padding: 0 15rpx;
      margin-top: 6rpx;
      font-size: 24rpx;
      align-self: flex-start;
      border-radius: 4rpx;
      color: #888;
      background-color: #f7f7f8;
    }

    .prices {
      display: flex;
      align-items: baseline;
      margin-top: 6rpx;
      font-size: 28rpx;

      .pay-price {
        margin-right: 10rpx;
        color: #cf4444;
      }

      .price {
        font-size: 24rpx;
        color: #999;
        text-decoration: line-through;
      }
    }

    .count {
      position: absolute;
      bottom: 0;
      right: 0;
      font-size: 26rpx;
      color: #444;
    }
  }
}

.related {
  margin: 20rpx;
  padding: 0 20rpx;
  border-radius: 10rpx;
  background-color: #fff;

  .item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    min-height: 80rpx;
    font-size: 26rpx;
    color: #333;
  }

  .input {
    flex: 1;
    text-align: right;
    margin: 20rpx 0;
    padding-right: 20rpx;
    font-size: 26rpx;
    color: #999;
  }

  .item .text {
    width: 125rpx;
  }

  .picker {
    color: #666;
  }

  .picker::after {
    content: '\e6c2';
  }
}

/* 结算清单 */
.settlement {
  margin: 20rpx;
  padding: 0 20rpx;
  border-radius: 10rpx;
  background-color: #fff;

  .item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    height: 80rpx;
    font-size: 26rpx;
    color: #333;
  }

  .danger {
    color: #cf4444;
  }
}

/* 吸底工具栏 */
.toolbar {
  position: fixed;
  left: 0;
  right: 0;
  bottom: calc(var(--window-bottom));
  z-index: 1;

  background-color: #fff;
  height: 100rpx;
  padding: 0 20rpx;
  border-top: 1rpx solid #eaeaea;
  display: flex;
  justify-content: space-between;
  align-items: center;
  box-sizing: content-box;

  .total-pay {
    font-size: 40rpx;
    color: #cf4444;

    .decimal {
      font-size: 75%;
    }
  }

  .button {
    width: 220rpx;
    text-align: center;
    line-height: 72rpx;
    font-size: 26rpx;
    color: #fff;
    border-radius: 72rpx;
    background-color: #27ba9b;
  }

  .disabled {
    opacity: 0.6;
  }
}
</style>
