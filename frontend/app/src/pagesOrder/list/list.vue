<script setup lang="ts">
import { computed, ref } from 'vue'
import OrderList from './components/OrderList.vue'
import { OrderStatus } from '@/rpc/common/v1/enum.ts'
import { useUserStore } from '@/stores'
import { navigateToLogin } from '@/utils/navigation'
import { onLoad } from '@dcloudio/uni-app'

type OrderTab = {
  status: OrderStatus
  title: string
  isRender: boolean
}

// 获取页面参数
const query = defineProps<{
  status?: string
}>()
const userStore = useUserStore()
const canRenderOrderList = computed(() => Boolean(userStore.userInfo))

// tabs 数据
const orderTabs = ref<OrderTab[]>([
  { status: OrderStatus.UNKNOWN_OS, title: '全部', isRender: false },
  { status: OrderStatus.CREATED, title: '待付款', isRender: false },
  { status: OrderStatus.PAID, title: '待发货', isRender: false },
  { status: OrderStatus.SHIPPED, title: '待收货', isRender: false },
  { status: OrderStatus.WAIT_REVIEW, title: '待评价', isRender: false },
  { status: OrderStatus.COMPLETED, title: '已完成', isRender: false },
  { status: OrderStatus.CANCELED, title: '已取消', isRender: false },
])
const cursorWidth = 100 / orderTabs.value.length

// 订单卡片状态文案优先复用顶部 tab 标题，避免状态文案重复维护。
const orderStatusTitleMap = new Map<OrderStatus, string>([
  ...orderTabs.value.map((item) => [item.status, item.title] as [OrderStatus, string]),
  // 退款/售后状态不作为顶部 tab 展示，但订单卡片仍需要展示对应状态文案。
  [OrderStatus.REFUNDING, '退款/售后'],
])

// 高亮下标
const defaultActiveIndex = orderTabs.value.findIndex((v) => v.status === Number(query.status))
const activeIndex = ref(defaultActiveIndex >= 0 ? defaultActiveIndex : 0)
// 默认渲染容器
orderTabs.value[activeIndex.value].isRender = true

onLoad(() => {
  if (!canRenderOrderList.value) {
    // 订单列表需要登录态，未登录时不渲染子列表，避免直接请求订单接口产生 401 噪声。
    navigateToLogin()
  }
})

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
          :status="item.status"
          :status-title-map="orderStatusTitleMap"
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
