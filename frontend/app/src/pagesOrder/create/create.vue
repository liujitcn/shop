<script setup lang="ts">
import { defOrderService } from '@/api/app/order_info.ts'
import { useAddressStore, useUserStore } from '@/stores'
import type {
  ConfirmOrderInfoResponse,
  BuyNowOrderInfoRequest,
  OrderGoods,
  RepurchaseOrderInfoRequest,
  BuyNowOrderInfoResponse,
  RepurchaseOrderInfoResponse,
  CreateOrderInfoGoods,
  CreateOrderInfoResponse,
} from '@/rpc/app/v1/order_info'
import type { BaseDictForm_DictItem } from '@/rpc/app/v1/base_dict'
import type { RecommendContext } from '@/rpc/app/v1/recommend'
import { onLoad } from '@dcloudio/uni-app'
import { computed, reactive, ref } from 'vue'
import type { UserAddress } from '@/rpc/app/v1/user_address'
import { defUserAddressService } from '@/api/app/user_address'
import { defBaseDictService } from '@/api/app/base_dict'
import { formatSrc, formatPrice } from '@/utils'
import { OrderPayChannel, OrderPayType, RecommendScene } from '@/rpc/common/v1/enum'
import {
  goodsDetailUrl,
  orderDetailUrl,
  parseRecommendRouteQuery,
  navigateToLogin,
  redirectToOrderPayment,
  tenantStoreUrl,
} from '@/utils/navigation'
import { startOrderPayment } from '@/utils/payment'

const addressStore = useAddressStore()
const userStore = useUserStore()
const isSubmitting = ref(false)

// 获取屏幕边界到安全区域距离
const { safeAreaInsets } = uni.getSystemInfoSync()
// 支付方式
const payTypeList = ref<BaseDictForm_DictItem[]>([])
// 当前支付方式下标
const payTypeActiveIndex = ref(0)
// 当前支付方式
const activePayType = computed(() => payTypeList.value[payTypeActiveIndex.value])
// 修改支付方式
const onChangePayType: UniHelper.SelectorPickerOnChange = (ev) => {
  payTypeActiveIndex.value = ev.detail.value
}
// 支付渠道
const payChannelList = ref<BaseDictForm_DictItem[]>([])
// 当前支付渠道下标
const payChannelActiveIndex = ref(0)
// 当前支付渠道
const activePayChannel = computed(() => payChannelList.value[payChannelActiveIndex.value])
// 修改支付渠道
const onChangePayChannel: UniHelper.SelectorPickerOnChange = (ev) => {
  payChannelActiveIndex.value = ev.detail.value
}

// 配送时间
const deliveryList = ref<BaseDictForm_DictItem[]>([])

type StoreOptionState = {
  deliveryIndex: number
  remark: string
}

const storeOptionState = reactive<Record<number, StoreOptionState>>({})

/** 初始化每个门店独立的配送时间和备注。 */
const initializeStoreOptions = () => {
  orderPre.value?.order_goods_stores.forEach((group) => {
    const storeID = group.store?.id
    if (storeID && !storeOptionState[storeID]) {
      storeOptionState[storeID] = { deliveryIndex: 0, remark: '' }
    }
  })
}

const getStoreDelivery = (storeID?: number) => {
  return storeID ? deliveryList.value[storeOptionState[storeID]?.deliveryIndex ?? 0] : undefined
}

const onChangeStoreDelivery = (storeID: number, index: number) => {
  storeOptionState[storeID].deliveryIndex = index
}

// 页面参数
const query = defineProps<{
  goods_id?: string
  sku_code?: string
  num?: string
  order_id?: string
  scene?: string
  request_id?: string
  index?: string
}>()
const routeRecommendQuery = parseRecommendRouteQuery(query)
const routeScene = routeRecommendQuery.scene
// 下单页统一使用解析后的推荐上下文，避免把路由字符串直接透传到接口层。
const recommend_context: RecommendContext = {
  scene: routeScene ?? RecommendScene.UNKNOWN_RS,
  request_id: routeRecommendQuery.request_id ?? 0,
  position: routeRecommendQuery.index ?? 0,
}

/** 构建订单提交商品项。 */
const buildOrderRequestGoods = (item: OrderGoods): CreateOrderInfoGoods => {
  return {
    goods_id: item.goods_id,
    sku_code: item.sku_code,
    num: item.num,
    recommend_context: item.recommend_context,
  }
}

// 获取订单信息
const orderPre = ref<
  ConfirmOrderInfoResponse | BuyNowOrderInfoResponse | RepurchaseOrderInfoResponse
>()
const getUserOrderPreData = async () => {
  if (query.goods_id && query.sku_code && query.num) {
    const request: BuyNowOrderInfoRequest = {
      goods_id: Number(query.goods_id),
      sku_code: query.sku_code,
      num: Number(query.num),
      recommend_context,
    }
    orderPre.value = await defOrderService.BuyNowOrderInfo(request)
  } else if (query.order_id) {
    // 再次购买
    const request: RepurchaseOrderInfoRequest = {
      order_id: Number(query.order_id),
    }
    orderPre.value = await defOrderService.RepurchaseOrderInfo(request)
  } else {
    orderPre.value = await defOrderService.ConfirmOrderInfo({})
  }
}

const addressList = ref<UserAddress[]>([])
const getUserAddressData = async () => {
  const res = await defUserAddressService.ListUserAddresses({})
  addressList.value = res.user_addresses || []
}

const getDictData = async () => {
  const payTypeCode = 'order_pay_type'
  const payChannelCode = 'order_pay_channel'
  const deliveryTimeCode = 'order_delivery_time'
  // 新接口每次只返回一个字典，这里并发拉取三个字典并分别写入页面状态。
  const [payTypeDict, payChannelDict, deliveryDict] = await Promise.all([
    defBaseDictService.GetBaseDict({ value: payTypeCode }),
    defBaseDictService.GetBaseDict({ value: payChannelCode }),
    defBaseDictService.GetBaseDict({ value: deliveryTimeCode }),
  ])
  payTypeList.value = payTypeDict.items || []
  payChannelList.value = (payChannelDict.items || []).filter(
    (item) => Number(item.value) === OrderPayChannel.WX_PAY,
  )
  deliveryList.value = deliveryDict.items || []
}

// 页面初始化前先校验登录态，避免匿名直达时并发请求多个强登录接口。
onLoad(() => {
  if (!userStore.ensureAuthenticated()) {
    navigateToLogin()
    return
  }

  Promise.all([getUserAddressData(), getUserOrderPreData(), getDictData()])
    .then(initializeStoreOptions)
    .catch(() => {
      uni.showToast({ icon: 'none', title: '订单信息加载失败，请稍后重试' })
    })
})

// 收货地址
const selectAddress = computed(() => {
  if (addressStore.selectedAddress) {
    return addressStore.selectedAddress
  } else {
    if (addressList.value) {
      return addressList.value.find((v) => v.is_default)
    } else {
      return undefined
    }
  }
})

// 提交订单
const onOrderSubmit = async () => {
  if (isSubmitting.value) {
    return
  }
  if (!userStore.ensureAuthenticated()) {
    navigateToLogin()
    return
  }
  // 没有收货地址提醒
  if (!selectAddress.value) {
    return uni.showToast({ icon: 'none', title: '请选择收货地址' })
  }
  if (!activePayType.value?.value) {
    return uni.showToast({ icon: 'none', title: '请选择支付方式' })
  }
  if (
    Number(activePayType.value.value) === OrderPayType.ONLINE_PAY &&
    !activePayChannel.value?.value
  ) {
    return uni.showToast({ icon: 'none', title: '请选择支付渠道' })
  }
  const orderGoodsStores = orderPre.value?.order_goods_stores || []
  if (
    !orderGoodsStores.length ||
    orderGoodsStores.some(
      (group) =>
        !group.store?.id || !group.goods.length || !getStoreDelivery(group.store.id)?.value,
    )
  ) {
    return uni.showToast({ icon: 'none', title: '请完善门店商品和配送信息' })
  }
  const orderStoreOptions = orderGoodsStores.map((group) => {
    const storeID = group.store!.id
    return {
      tenant_store_id: storeID,
      delivery_time: Number(getStoreDelivery(storeID)!.value),
      remark: storeOptionState[storeID].remark,
    }
  })
  const requestGoods = orderGoodsStores
    .flatMap((store) => store.goods)
    .map((item) => buildOrderRequestGoods(item))
  isSubmitting.value = true
  let res: CreateOrderInfoResponse
  try {
    res = await defOrderService.CreateOrderInfo({
      /** 地址id */
      address_id: selectAddress.value!.id,
      /** 是否清空购物车 */
      clear_cart: orderPre.value?.clear_cart || false,
      /** 支付方式：枚举【OrderPayType】 */
      pay_type: Number(activePayType.value.value),
      /** 支付渠道：枚举【OrderPayChannel】 */
      pay_channel: Number(activePayChannel.value?.value || 0),
      /** 每个门店独立的配送时间和备注 */
      order_store_options: orderStoreOptions,
      /** 商品信息 */
      goods: requestGoods,
    })
  } catch (error) {
    // 只有交易尚未创建时才释放提交锁，允许用户修正或重试下单。
    isSubmitting.value = false
    throw error
  }
  // 在线订单创建成功后立即使用交易单编号调起支付，支付结果页只负责查询真实状态。
  if (Number(activePayType.value.value) === OrderPayType.ONLINE_PAY) {
    try {
      await startOrderPayment(res.trade_id)
    } catch (error) {
      // 调起支付失败时保留接口错误提示，不进入支付结果页伪装为支付确认中。
      isSubmitting.value = false
      throw error
    }
  } else {
    await redirectToOrderPayment(res.trade_id)
  }
}

// 计算提交按钮是否应置灰
const onOrderSubmitOk = computed(() => {
  if (isSubmitting.value) {
    return true
  }
  if (!selectAddress.value?.id) {
    return true
  }
  if (!activePayType.value?.value) {
    return true
  }
  if (
    !orderPre.value?.order_goods_stores.length ||
    orderPre.value.order_goods_stores.some(
      (group) => !group.store?.id || !getStoreDelivery(group.store.id)?.value,
    )
  ) {
    return true
  }
  if (
    Number(activePayType.value?.value) === OrderPayType.ONLINE_PAY &&
    !activePayChannel.value?.value
  ) {
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
    <view v-for="group in orderPre?.order_goods_stores" :key="group.store?.id" class="goods">
      <navigator
        v-if="group.store?.id"
        class="store-header"
        :url="tenantStoreUrl(group.store.id)"
        hover-class="none"
      >
        <image
          v-if="group.store.logo"
          class="store-logo"
          :src="formatSrc(group.store.logo)"
          mode="aspectFill"
        />
        <text class="store-name">{{ group.store.name || '店铺' }}</text>
        <text class="store-arrow">&gt;</text>
      </navigator>
      <view v-else class="store-header">
        <text class="store-name">店铺</text>
      </view>
      <navigator
        v-for="item in group.goods"
        :key="item.sku_code"
        :url="
          goodsDetailUrl({
            id: item.goods_id,
            scene: item.recommend_context?.scene,
            request_id: item.recommend_context?.request_id,
            index: item.recommend_context?.position,
          })
        "
        class="item"
        hover-class="none"
      >
        <image class="picture" :src="formatSrc(item.picture)" />
        <view class="meta">
          <view class="name ellipsis"> {{ item.name }} </view>
          <view class="attrs">{{ item.spec_item.join('/') }}</view>
          <view class="prices">
            <view class="pay-price symbol">{{ formatPrice(item.pay_price) }}</view>
            <view class="price symbol">{{ formatPrice(item.price) }}</view>
          </view>
          <view class="count">x{{ item.num }}</view>
        </view>
      </navigator>
      <view v-if="group.store?.id" class="store-options">
        <view class="option-row">
          <text class="option-label">配送时间</text>
          <picker
            :range="deliveryList"
            range-key="label"
            @change="onChangeStoreDelivery(group.store.id, $event.detail.value)"
          >
            <view class="icon-fonts picker">{{ getStoreDelivery(group.store.id)?.label }}</view>
          </picker>
        </view>
        <view class="option-row">
          <text class="option-label">订单备注</text>
          <input
            class="option-input"
            :cursor-spacing="30"
            placeholder="给当前门店留言"
            v-model="storeOptionState[group.store.id].remark"
          />
        </view>
      </view>
    </view>

    <!-- 支付方式 -->
    <view class="related">
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
    </view>

    <!-- 支付金额 -->
    <view class="settlement" v-if="orderPre?.summary">
      <view class="item">
        <text class="text">商品总价: </text>
        <text class="number symbol">{{ formatPrice(orderPre!.summary!.total_money) }}</text>
      </view>
      <view class="item">
        <text class="text">运费: </text>
        <text class="number symbol"> {{ formatPrice(orderPre!.summary?.post_fee) }}</text>
      </view>
    </view>
    <view
      class="toolbar-placeholder"
      :style="{ height: `calc(100rpx + ${safeAreaInsets!.bottom}px)` }"
    />
  </scroll-view>

  <!-- 吸底工具栏 -->
  <view class="toolbar" :style="{ paddingBottom: safeAreaInsets!.bottom + 'px' }">
    <view class="total-pay symbol" v-if="orderPre?.summary">
      <text class="number">{{ formatPrice(orderPre!.summary!.pay_money) }}</text>
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

  .store-header {
    display: flex;
    align-items: center;
    height: 80rpx;
  }

  .store-logo {
    width: 40rpx;
    height: 40rpx;
    margin-right: 12rpx;
    border-radius: 50%;
  }

  .store-name {
    flex: 1;
    overflow: hidden;
    font-size: 26rpx;
    font-weight: 500;
    color: #333;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .store-arrow {
    margin-left: 12rpx;
    color: #999;
    font-size: 28rpx;
  }

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

  .store-options {
    border-top: 1rpx solid #eee;
  }

  .option-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    min-height: 80rpx;
    font-size: 26rpx;
    color: #333;
  }

  .option-label {
    flex-shrink: 0;
    width: 130rpx;
  }

  .option-input {
    flex: 1;
    margin: 20rpx 0;
    text-align: right;
    color: #666;
  }

  .picker {
    color: #666;
  }

  .picker::after {
    content: '\e6c2';
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
