<script setup lang="ts">
import { computed, ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { defOrderService } from '@/api/app/order_info'
import type { OrderGoods, OrderInfo } from '@/rpc/app/v1/order_info'
import { OrderStatus } from '@/rpc/common/v1/enum'
import { formatPrice, formatSrc } from '@/utils'
import { orderDetailUrl } from '@/utils/navigation'
import RefundOrderPopup from '../components/RefundOrderPopup.vue'

const query = defineProps<{
  tab?: string
}>()

const { safeAreaInsets } = uni.getSystemInfoSync()

type AfterSaleTab = 'apply' | 'record'

type AfterSaleTabItem = {
  key: AfterSaleTab
  title: string
  status: OrderStatus
}

const tabs: AfterSaleTabItem[] = [
  { key: 'apply', title: '售后申请', status: OrderStatus.PAID },
  { key: 'record', title: '申请记录', status: OrderStatus.REFUNDING },
]

const normalizeTab = (tab?: string): AfterSaleTab => {
  return tab === 'record' ? 'record' : 'apply'
}

const activeTab = ref<AfterSaleTab>(normalizeTab(query.tab))
const orderMap = ref<Record<AfterSaleTab, OrderInfo[]>>({
  apply: [],
  record: [],
})
const loadedMap = ref<Record<AfterSaleTab, boolean>>({
  apply: false,
  record: false,
})
const loadingMap = ref<Record<AfterSaleTab, boolean>>({
  apply: false,
  record: false,
})
const refundPopup = ref<InstanceType<typeof RefundOrderPopup>>()

const activeTabItem = computed(() => tabs.find((item) => item.key === activeTab.value) || tabs[0])
const activeOrders = computed(() => orderMap.value[activeTab.value])
const isLoading = computed(() => loadingMap.value[activeTab.value])
const emptyInfo = computed(() => {
  if (activeTab.value === 'apply') {
    return {
      image: '/static/images/empty_after_sale.png',
      text: '暂无可申请售后的订单',
    }
  }
  return {
    image: '/static/images/empty_after_sale.png',
    text: '暂无申请记录',
  }
})

const onNavigateBack = () => {
  const pages = getCurrentPages()
  if (pages.length > 1) {
    uni.navigateBack()
    return
  }
  uni.switchTab({ url: '/pages/my/my' })
}

const onSwitchTab = (tab: AfterSaleTab) => {
  activeTab.value = tab
  if (!loadedMap.value[tab]) {
    void loadOrders(tab)
  }
}

const loadOrders = async (tab: AfterSaleTab = activeTab.value) => {
  if (loadingMap.value[tab]) {
    return
  }
  loadingMap.value = { ...loadingMap.value, [tab]: true }
  try {
    const res = await defOrderService.PageOrderInfo({
      page_num: 1,
      page_size: 20,
      status: tabs.find((item) => item.key === tab)?.status || OrderStatus.PAID,
    })
    orderMap.value = { ...orderMap.value, [tab]: res.order_infos || [] }
    loadedMap.value = { ...loadedMap.value, [tab]: true }
  } catch (_error) {
    orderMap.value = { ...orderMap.value, [tab]: [] }
  } finally {
    loadingMap.value = { ...loadingMap.value, [tab]: false }
  }
}

const getGoodsSpec = (goods: OrderGoods) => {
  return goods.spec_item?.length ? goods.spec_item.join(' / ') : '标准规格'
}

const getRecordStatus = (order: OrderInfo) => {
  if (activeTab.value === 'apply') {
    return '可申请售后'
  }
  return order.refund_time ? '已退款' : '退款/售后处理中'
}

const onOpenRefundPopup = (order: OrderInfo) => {
  refundPopup.value?.open(order)
}

const onRefundSuccess = (order_id: number) => {
  orderMap.value = {
    ...orderMap.value,
    apply: orderMap.value.apply.filter((order) => order.id !== order_id),
  }
  loadedMap.value = { ...loadedMap.value, record: false }
}

onShow(() => {
  void loadOrders(activeTab.value)
})
</script>

<template>
  <view class="after-sale-page">
    <view class="after-sale-header" :style="{ paddingTop: `${safeAreaInsets?.top || 0}px` }">
      <view class="after-sale-nav">
        <view class="back-button" @tap="onNavigateBack"></view>
        <view class="page-title">退款/售后</view>
      </view>
      <view class="after-sale-tabs">
        <view
          v-for="item in tabs"
          :key="item.key"
          class="after-sale-tab"
          :class="{ active: activeTab === item.key }"
          @tap="onSwitchTab(item.key)"
        >
          {{ item.title }}
        </view>
      </view>
    </view>

    <scroll-view scroll-y class="after-sale-body">
      <view v-if="isLoading" class="state-card">售后订单加载中...</view>
      <XtxEmptyState
        v-else-if="activeOrders.length === 0"
        :image="emptyInfo.image"
        :text="emptyInfo.text"
        padding="110rpx 48rpx 0"
      />

      <template v-else>
        <view v-for="order in activeOrders" :key="order.id" class="after-sale-card">
          <view class="card-head">
            <view class="order-no">订单号 {{ order.order_no }}</view>
            <view class="status-tag">{{ getRecordStatus(order) }}</view>
          </view>

          <view
            v-for="goods in order.goods"
            :key="`${order.id}-${goods.goods_id}-${goods.sku_code}`"
            class="goods-row"
          >
            <image class="goods-cover" :src="formatSrc(goods.picture)" mode="aspectFill" />
            <view class="goods-main">
              <view class="goods-name ellipsis-2">{{ goods.name }}</view>
              <view class="goods-spec ellipsis">{{ getGoodsSpec(goods) }}</view>
              <view class="goods-price">
                <text>¥{{ formatPrice(goods.pay_price) }}</text>
                <text class="goods-num">x{{ goods.num }}</text>
              </view>
            </view>
          </view>

          <view class="order-summary">
            <text>{{ activeTabItem.title }}</text>
            <text class="summary-amount">实付 ¥{{ formatPrice(order.pay_money) }}</text>
          </view>

          <view class="card-actions">
            <navigator
              class="action-button"
              :url="orderDetailUrl({ id: order.id })"
              hover-class="none"
            >
              查看订单
            </navigator>
            <view
              v-if="activeTab === 'apply'"
              class="action-button primary"
              @tap="onOpenRefundPopup(order)"
            >
              退款/售后
            </view>
          </view>
        </view>
      </template>
    </scroll-view>

    <RefundOrderPopup ref="refundPopup" @success="onRefundSuccess" />
  </view>
</template>

<style lang="scss">
page {
  height: 100%;
  background-color: #f4f4f4;
}

.after-sale-page {
  height: 100%;
  display: flex;
  flex-direction: column;
  color: #333;
  background-color: #f4f4f4;
}

.after-sale-header {
  flex-shrink: 0;
  background-color: #fff;
  box-shadow: 0 4rpx 10rpx rgba(0, 0, 0, 0.03);
  position: relative;
  z-index: 2;
}

.after-sale-nav {
  height: 88rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  border-bottom: 1rpx solid #f1f1f1;
}

.back-button {
  position: absolute;
  left: 24rpx;
  top: 0;
  width: 72rpx;
  height: 88rpx;
  line-height: 88rpx;
  font-size: 34rpx;
  color: #333;
}

.back-button::before {
  content: '';
  position: absolute;
  left: 26rpx;
  top: 50%;
  width: 22rpx;
  height: 22rpx;
  border-left: 4rpx solid #333;
  border-bottom: 4rpx solid #333;
  transform: translateY(-50%) rotate(45deg);
}

.page-title {
  font-size: 34rpx;
  font-weight: 600;
  color: #222;
}

.after-sale-tabs {
  height: 88rpx;
  display: flex;
  align-items: center;
  background-color: #fff;
}

.after-sale-tab {
  flex: 1;
  height: 88rpx;
  line-height: 88rpx;
  text-align: center;
  font-size: 28rpx;
  color: #333;
  position: relative;

  &.active {
    color: #27ba9b;
    font-weight: 600;

    &::after {
      content: '';
      position: absolute;
      left: 50%;
      bottom: 0;
      width: 48rpx;
      height: 6rpx;
      border-radius: 6rpx;
      background-color: #27ba9b;
      transform: translateX(-50%);
    }
  }
}

.after-sale-body {
  flex: 1;
  min-height: 0;
  padding: 20rpx 0 32rpx;
  box-sizing: border-box;
}

.after-sale-card,
.state-card {
  margin: 0 20rpx 20rpx;
  border-radius: 10rpx;
  background-color: #fff;
}

.state-card {
  padding: 70rpx 30rpx;
  text-align: center;
  color: #888;
  font-size: 26rpx;
}

.after-sale-card {
  overflow: hidden;
}

.card-head {
  height: 78rpx;
  padding: 0 24rpx;
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-bottom: 1rpx solid #f3f3f3;
}

.order-no {
  flex: 1;
  font-size: 24rpx;
  color: #999;
}

.status-tag {
  margin-left: 16rpx;
  font-size: 24rpx;
  color: #27ba9b;
}

.goods-row {
  display: flex;
  padding: 22rpx 24rpx 0;
}

.goods-cover {
  width: 150rpx;
  height: 150rpx;
  margin-right: 20rpx;
  border-radius: 10rpx;
  background-color: #f4f4f4;
}

.goods-main {
  flex: 1;
  min-width: 0;
}

.goods-name {
  min-height: 72rpx;
  font-size: 27rpx;
  line-height: 36rpx;
  color: #333;
}

.goods-spec {
  display: inline-block;
  max-width: 100%;
  margin-top: 12rpx;
  padding: 4rpx 14rpx;
  box-sizing: border-box;
  border-radius: 6rpx;
  background-color: #f7f7f8;
  font-size: 23rpx;
  color: #888;
}

.goods-price {
  margin-top: 12rpx;
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 26rpx;
  color: #333;
}

.goods-num {
  margin-left: 16rpx;
  color: #999;
  font-size: 24rpx;
}

.order-summary {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin: 22rpx 24rpx 0;
  padding: 18rpx 20rpx;
  border-radius: 10rpx;
  background-color: #f7f7f8;
  font-size: 24rpx;
  color: #888;
}

.summary-amount {
  color: #333;
  font-size: 26rpx;
}

.card-actions {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  padding: 22rpx 24rpx 24rpx;
}

.action-button {
  min-width: 168rpx;
  height: 60rpx;
  line-height: 58rpx;
  margin-left: 18rpx;
  padding: 0 24rpx;
  box-sizing: border-box;
  border: 1rpx solid #ccc;
  border-radius: 60rpx;
  text-align: center;
  font-size: 26rpx;
  color: #444;

  &.primary {
    color: #27ba9b;
    border-color: #27ba9b;
  }
}
</style>
