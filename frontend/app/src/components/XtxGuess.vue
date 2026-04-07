<script setup lang="ts">
import { defRecommendService, reportRecommendExposure } from '@/api/app/recommend'
import { useUserStore } from '@/stores'
import { formatPrice, formatSrc } from '@/utils'
import { getCurrentInstance, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import type { GoodsInfo } from '@/rpc/app/goods_info'
import { RecommendScene } from '@/rpc/common/enum'

const userStore = useUserStore()

// 组件属性
const props = withDefaults(
  defineProps<{
    title?: string
    scene: RecommendScene
    orderId?: number
  }>(),
  {
    title: '猜你喜欢',
    orderId: 0,
  },
)

// 分页参数
const pageParams = {
  scene: props.scene,
  orderId: props.orderId,
  pageNum: 1,
  pageSize: 10,
}
// 猜你喜欢的列表
const guessList = ref<GoodsInfo[]>([])
// 已结束标记
const finish = ref(false)
// 推荐请求ID
const requestId = ref('')
// 是否已经进入可视区
const isVisible = ref(false)
// 是否已经记录曝光
const exposed = ref(false)
// 交叉观察器
let exposureObserver: any

// 获取猜你喜欢数据
const getHomeGoodsGuessLikeData = async () => {
  // 退出分页判断
  if (finish.value === true) {
    return uni.showToast({ icon: 'none', title: '没有更多数据~' })
  }
  pageParams.scene = props.scene
  pageParams.orderId = props.orderId
  const res = await defRecommendService.RecommendGoods(pageParams)
  if (!requestId.value) {
    requestId.value = res.requestId
  }
  // 数组追加
  const list = res.list || []
  guessList.value.push(...list)
  // 分页条件
  if (guessList.value.length < res.total) {
    // 页码累加
    pageParams.pageNum++
  } else {
    finish.value = true
  }
  await nextTick()
  await reportExposure()
}
// 重置数据
const resetData = () => {
  pageParams.pageNum = 1
  guessList.value = []
  finish.value = false
  requestId.value = ''
  exposed.value = false
}

// 上报曝光埋点
const reportExposure = async () => {
  if (
    exposed.value ||
    !userStore.userInfo ||
    !isVisible.value ||
    !requestId.value ||
    guessList.value.length === 0
  ) {
    return
  }
  exposed.value = true
  try {
    await reportRecommendExposure({
      requestId: requestId.value,
      scene: String(props.scene),
      goodsIds: guessList.value.map((item) => item.id),
    })
  } catch (error) {
    exposed.value = false
    console.error(error)
  }
}

// 初始化曝光观察器
const initExposureObserver = () => {
  const instance = getCurrentInstance()
  if (!instance) {
    return
  }
  exposureObserver = uni.createIntersectionObserver(instance)
  exposureObserver.relativeToViewport().observe('.guess-root', (res: { intersectionRatio: number }) => {
    if (res.intersectionRatio <= 0) {
      return
    }
    isVisible.value = true
    void reportExposure()
  })
}
// 组件挂载完毕
onMounted(() => {
  initExposureObserver()
  void getHomeGoodsGuessLikeData()
})

// 组件卸载
onBeforeUnmount(() => {
  exposureObserver?.disconnect?.()
})
// 暴露方法
defineExpose({
  resetData,
  getMore: getHomeGoodsGuessLikeData,
})
</script>

<template>
  <!-- 猜你喜欢 -->
  <view class="guess-root">
    <view class="caption">
      <text class="text">{{ props.title }}</text>
    </view>
    <view class="guess">
      <navigator
        v-for="(item, index) in guessList"
        :key="item.id"
        class="guess-item"
        :url="`/pages/goods/goods?id=${item.id}&source=recommend&scene=${props.scene}&requestId=${requestId}&index=${index}`"
      >
        <image class="image" mode="aspectFill" :src="formatSrc(item.picture)" />
        <view class="name"> {{ item.name }} </view>
        <view class="price">
          <text class="small">¥</text>
          <text>{{ formatPrice(item.price) }}</text>
        </view>
      </navigator>
    </view>
    <view class="loading-text">
      {{ finish ? '没有更多数据~' : '正在加载...' }}
    </view>
  </view>
</template>

<style lang="scss">
:host {
  display: block;
}
/* 分类标题 */
.caption {
  display: flex;
  justify-content: center;
  line-height: 1;
  padding: 36rpx 0 40rpx;
  font-size: 32rpx;
  color: #262626;
  .text {
    display: flex;
    justify-content: center;
    align-items: center;
    padding: 0 28rpx 0 30rpx;

    &::before,
    &::after {
      content: '';
      width: 20rpx;
      height: 20rpx;
      background-image: url(@/static/images/bubble.png);
      background-size: contain;
      margin: 0 10rpx;
    }
  }
}

/* 猜你喜欢 */
.guess {
  display: flex;
  flex-wrap: wrap;
  justify-content: space-between;
  padding: 0 20rpx;
  .guess-item {
    width: 345rpx;
    padding: 24rpx 20rpx 20rpx;
    margin-bottom: 20rpx;
    border-radius: 10rpx;
    overflow: hidden;
    background-color: #fff;
  }
  .image {
    width: 304rpx;
    height: 304rpx;
  }
  .name {
    height: 75rpx;
    margin: 10rpx 0;
    font-size: 26rpx;
    color: #262626;
    overflow: hidden;
    text-overflow: ellipsis;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
  }
  .price {
    line-height: 1;
    padding-top: 4rpx;
    color: #cf4444;
    font-size: 26rpx;
  }
  .small {
    font-size: 80%;
  }
}
// 加载提示文字
.loading-text {
  text-align: center;
  font-size: 28rpx;
  color: #666;
  padding: 20rpx 0;
}
</style>
