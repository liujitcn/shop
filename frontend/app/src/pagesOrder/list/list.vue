<script setup lang="ts">
import { ref } from 'vue'
import OrderList from './components/OrderList.vue'
import { onLoad } from '@dcloudio/uni-app'
import { defBaseDictService } from '@/api/app/base_dict'
import { OrderStatus } from '@/rpc/common/enum.ts'

// 获取页面参数
const query = defineProps<{
  status: string
}>()

// tabs 数据
const orderTabs = ref([
  { status: OrderStatus.UNKNOWN_OS, title: '全部', isRender: false },
  { status: OrderStatus.CREATED, isRender: false },
  { status: OrderStatus.PAID, isRender: false },
  { status: OrderStatus.SHIPPED, isRender: false },
  { status: OrderStatus.RECEIVED, isRender: false },
])
const orderStatusMap: Map<number, string> = new Map()

onLoad(async () => {
  const code = 'order_status'
  const orderStatus = await defBaseDictService.ListBaseDict({
    value: code,
  })
  if (orderStatus && orderStatus.list) {
    orderStatus.list.map((dict) => {
      if (dict.code === code)
        if (dict.items) {
          dict.items.map((dictItem) => {
            orderStatusMap.set(Number(dictItem.value), dictItem.label)
          })
        }
    })
  }

  orderTabs.value.map((item) => {
    if (orderStatusMap.has(item.status)) {
      item.title = orderStatusMap.get(item.status)
    }
  })
})

// 高亮下标
const activeIndex = ref(orderTabs.value.findIndex((v) => v.status === Number(query.status)))
// 默认渲染容器
orderTabs.value[activeIndex.value].isRender = true
</script>

<template>
  <view class="viewport">
    <!-- tabs -->
    <view class="tabs">
      <text
        v-for="(item, index) in orderTabs"
        :key="item.title"
        class="item"
        @tap="
          () => {
            activeIndex = index
            item.isRender = true
          }
        "
      >
        {{ item.title }}
      </text>
      <!-- 游标 -->
      <view class="cursor" :style="{ left: activeIndex * 20 + '%' }" />
    </view>
    <!-- 滑动容器 -->
    <swiper class="swiper" :current="activeIndex" @change="activeIndex = $event.detail.current">
      <!-- 滑动项 -->
      <swiper-item v-for="item in orderTabs" :key="item.title">
        <!-- 订单列表 -->
        <OrderList v-if="item.isRender" :status="item.status" :status-map="orderStatusMap" />
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
  display: flex;
  justify-content: space-around;
  line-height: 60rpx;
  margin: 0 20rpx;
  background-color: #fff;
  border-radius: 20rpx;
  box-shadow: 0 8rpx 24rpx rgba(15, 23, 42, 0.04);
  position: relative;
  z-index: 9;
  overflow: hidden;

  .item {
    flex: 1;
    text-align: center;
    padding: 20rpx;
    font-size: 28rpx;
    color: #262626;
  }

  .cursor {
    position: absolute;
    left: 0;
    bottom: 0;
    width: 20%;
    height: 6rpx;
    padding: 0 50rpx;
    background-color: #27ba9b;
    /* 过渡效果 */
    transition: all 0.4s;
  }
}

// swiper
.swiper {
  background-color: #f7f7f8;
}
</style>
