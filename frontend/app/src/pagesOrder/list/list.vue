<script setup lang="ts">
import { computed, ref } from 'vue'
import OrderList from './components/OrderList.vue'
import { OrderInfoStatus, OrderTradeStatus } from '@/rpc/shop/common/v1/enum.ts'
import { useUserStore } from '@/stores'
import { navigateToLogin } from '@/utils/navigation'
import type { OrderListFilter } from '@/utils/order'
import { onLoad } from '@dcloudio/uni-app'

/** 订单列表页顶部状态标签配置。 */
type OrderTab = {
  key: string
  title: string
  filter: OrderListFilter
  isRender: boolean
}

// 获取页面参数
const query = defineProps<{
  status?: string
  trade_status?: string
  refund_status?: string
  has_refund?: string
}>()
const userStore = useUserStore()
const canRenderOrderList = computed(() => userStore.isAuthenticated())

// tabs 数据
const orderTabs = ref<OrderTab[]>([
  { key: 'all', title: '全部', filter: {}, isRender: false },
  {
    key: 'pending-payment',
    title: '待支付',
    filter: { trade_status: OrderTradeStatus.PENDING_PAYMENT_OTS },
    isRender: false,
  },
  {
    key: 'wait-shipment',
    title: '待发货',
    filter: { status: OrderInfoStatus.WAIT_SHIPMENT_OIS },
    isRender: false,
  },
  {
    key: 'shipped',
    title: '待收货',
    filter: { status: OrderInfoStatus.SHIPPED_OIS },
    isRender: false,
  },
  {
    key: 'wait-review',
    title: '待评价',
    filter: { status: OrderInfoStatus.WAIT_REVIEW_OIS },
    isRender: false,
  },
  {
    key: 'completed',
    title: '已完成',
    filter: { status: OrderInfoStatus.COMPLETED_OIS },
    isRender: false,
  },
  {
    key: 'canceled',
    title: '已取消',
    filter: { status: OrderInfoStatus.CANCELED_OIS },
    isRender: false,
  },
])
const cursorWidth = 100 / orderTabs.value.length

// 高亮下标
const routeFilter: OrderListFilter = {
  status: query.status ? Number(query.status) : undefined,
  trade_status: query.trade_status ? Number(query.trade_status) : undefined,
  refund_status: query.refund_status ? Number(query.refund_status) : undefined,
  has_refund: query.has_refund === undefined ? undefined : query.has_refund === 'true',
}
const defaultActiveIndex = orderTabs.value.findIndex((item) => {
  const filter = item.filter
  return (
    filter.status === routeFilter.status &&
    filter.trade_status === routeFilter.trade_status &&
    filter.refund_status === routeFilter.refund_status &&
    filter.has_refund === routeFilter.has_refund
  )
})
const activeIndex = ref(defaultActiveIndex >= 0 ? defaultActiveIndex : 0)
// 默认渲染容器
orderTabs.value[activeIndex.value].isRender = true

onLoad(() => {
  if (!userStore.ensureAuthenticated()) {
    // 订单列表需要登录态，未登录时不渲染子列表，避免直接请求订单接口产生 401 噪声。
    navigateToLogin()
  }
})

/** 切换订单状态标签，并标记对应列表允许渲染。 */
const onChangeActiveIndex = (index: number) => {
  activeIndex.value = index
  orderTabs.value[index].isRender = true
}
</script>

<template>
  <view class="viewport">
    <!-- tabs -->
    <scroll-view scroll-x class="tabs" :show-scrollbar="false">
      <view class="tabs-inner">
        <text
          v-for="(item, index) in orderTabs"
          :key="item.title"
          class="item"
          :class="{ active: activeIndex === index }"
          @tap="onChangeActiveIndex(index)"
        >
          {{ item.title }}
        </text>
        <!-- 游标 -->
        <view
          class="cursor"
          :style="{ left: activeIndex * cursorWidth + '%', width: cursorWidth + '%' }"
        />
      </view>
    </scroll-view>
    <!-- 滑动容器 -->
    <swiper
      class="swiper"
      :current="activeIndex"
      @change="onChangeActiveIndex($event.detail.current)"
    >
      <!-- 滑动项 -->
      <swiper-item v-for="item in orderTabs" :key="item.title">
        <!-- 订单列表 -->
        <OrderList
          v-if="canRenderOrderList && item.isRender"
          :filter="item.filter"
          :title="item.title"
        />
      </swiper-item>
    </swiper>
  </view>
</template>

<style lang="scss">
page {
  height: 100%;
  overflow: hidden;
}

.viewport {
  height: 100%;
  display: flex;
  flex-direction: column;
  background-color: #fff;
}

// tabs
.tabs {
  width: 100%;
  height: 98rpx;
  flex: 0 0 98rpx;
  white-space: nowrap;
  margin: 0;
  background-color: #fff;
  box-shadow: 0 4rpx 6rpx rgba(240, 240, 240, 0.6);
  position: relative;
  z-index: 9;

  .tabs-inner {
    position: relative;
    min-width: 980rpx;
    height: 98rpx;
    display: flex;
    align-items: center;
  }

  .item {
    flex: 1;
    text-align: center;
    line-height: 98rpx;
    font-size: 28rpx;
    color: #262626;

    &.active {
      color: #27ba9b;
      font-weight: 600;
    }
  }

  .cursor {
    position: absolute;
    left: 0;
    bottom: 0;
    display: flex;
    justify-content: center;
    height: 6rpx;
    background-color: transparent;
    /* 过渡效果 */
    transition: all 0.4s;

    &::after {
      content: '';
      width: 48rpx;
      height: 6rpx;
      border-radius: 6rpx;
      background-color: #27ba9b;
    }
  }
}

// swiper
.swiper {
  flex: 1;
  min-height: 0;
  background-color: #f7f7f8;
}
</style>
