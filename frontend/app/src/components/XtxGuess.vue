<script setup lang="ts">
import {
  buildRecommendGoodsActionItem,
  defRecommendService,
  formatRecommendSource,
  normalizeRecommendScene,
  reportRecommendExposure,
  reportRecommendGoodsAction,
} from '@/api/app/recommend'
import { formatPrice, formatSrc } from '@/utils'
import { getCurrentInstance, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import type { GoodsInfo } from '@/rpc/app/goods_info'
import { RecommendGoodsActionType, RecommendScene, RecommendSource } from '@/rpc/common/enum'

interface GuessGoods extends GoodsInfo {
  recommendRequestId: string
  recommendScene: string
  recommendIndex: number
}

interface RecommendExposureBatch {
  requestId: string
  scene: string
  goodsIds: number[]
  exposed: boolean
}

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
const guessList = ref<GuessGoods[]>([])
// 已结束标记
const finish = ref(false)
// 是否已经进入可视区
const isVisible = ref(false)
// 未曝光的分页批次
const exposureBatches = ref<RecommendExposureBatch[]>([])
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
  const startIndex = guessList.value.length
  const sceneName = normalizeRecommendScene(props.scene)
  const list = (res.list || []).map((item, index) => ({
    ...item,
    recommendRequestId: res.requestId,
    recommendScene: sceneName,
    recommendIndex: startIndex + index,
  }))
  guessList.value.push(...list)
  if (res.requestId && list.length > 0) {
    exposureBatches.value.push({
      requestId: res.requestId,
      scene: sceneName,
      goodsIds: list.map((item) => item.id),
      exposed: false,
    })
  }
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
  exposureBatches.value = []
}

// 上报曝光埋点
const reportExposure = async () => {
  if (!isVisible.value || exposureBatches.value.length === 0) {
    return
  }
  for (const batch of exposureBatches.value) {
    if (batch.exposed || !batch.requestId || batch.goodsIds.length === 0) {
      continue
    }
    batch.exposed = true
    try {
      await reportRecommendExposure({
        requestId: batch.requestId,
        scene: batch.scene,
        goodsIds: batch.goodsIds,
      })
    } catch (error) {
      batch.exposed = false
      console.error(error)
    }
  }
}

const onTapGoods = async (item: GuessGoods) => {
  try {
    await reportRecommendGoodsAction(RecommendGoodsActionType.RECOMMEND_GOODS_ACTION_CLICK, [
      buildRecommendGoodsActionItem({
        goodsId: item.id,
        goodsNum: 1,
        source: RecommendSource.RECOMMEND,
        scene: item.recommendScene,
        requestId: item.recommendRequestId,
        index: item.recommendIndex,
      }),
    ])
  } catch (error) {
    console.error(error)
  }
  void uni.navigateTo({
    url: `/pages/goods/goods?id=${item.id}&source=${formatRecommendSource(RecommendSource.RECOMMEND)}&scene=${item.recommendScene}&requestId=${item.recommendRequestId}&index=${item.recommendIndex}`,
  })
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
      <view
        v-for="item in guessList"
        :key="item.id"
        class="guess-item"
        @tap="onTapGoods(item)"
      >
        <image class="image" mode="aspectFill" :src="formatSrc(item.picture)" />
        <view class="name"> {{ item.name }} </view>
        <view class="price">
          <text class="small">¥</text>
          <text>{{ formatPrice(item.price) }}</text>
        </view>
      </view>
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
