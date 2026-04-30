<script setup lang="ts">
import { defRecommendService } from '@/api/app/recommend'
import { useRecommendStore } from '@/stores'
import { formatPrice, formatSrc } from '@/utils'
import { computed, getCurrentInstance, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import type { GoodsInfo } from '@/rpc/app/v1/goods_info'
import type {
  RecommendEventItem,
  RecommendGoodsRequest,
  RecommendGoodsResponse,
} from '@/rpc/app/v1/recommend'
import { RecommendEventType, RecommendScene } from '@/rpc/common/v1/enum'
import { goodsDetailUrl } from '@/utils/navigation'

type GuessGoods = GoodsInfo & {
  recommend_request_id: RecommendGoodsResponse['request_id']
  recommendScene: RecommendScene
  recommendIndex: number
}

type RecommendExposureBatch = {
  request_id: RecommendGoodsResponse['request_id']
  scene: RecommendScene
  items: RecommendEventItem[]
  /** 当前批次是否已经上报过曝光。 */
  exposed: boolean
}

// 组件属性
const props = withDefaults(
  defineProps<{
    title?: string
    scene: RecommendScene
    order_id?: number
    goods_id?: number
  }>(),
  {
    title: '猜你喜欢',
    order_id: 0,
    goods_id: 0,
  },
)

// 分页参数
const pageParams: RecommendGoodsRequest = {
  scene: props.scene,
  order_id: props.order_id,
  goods_id: props.goods_id,
  page_num: 1,
  page_size: 10,
  request_id: 0,
}
// 猜你喜欢的列表
const guessList = ref<GuessGoods[]>([])
// 已结束标记
const finish = ref(false)
const isEmpty = computed(() => finish.value && guessList.value.length === 0)
const emptyText = computed(() => {
  if (props.scene === RecommendScene.CART) {
    return '暂无搭配推荐'
  }
  if (props.scene === RecommendScene.PROFILE) {
    return '暂无专属推荐'
  }
  if (props.scene === RecommendScene.ORDER_DETAIL || props.scene === RecommendScene.ORDER_PAID) {
    return '暂无订单相关推荐'
  }
  return '暂无推荐商品'
})
// 是否已经进入可视区
const isVisible = ref(false)
// 未曝光的分页批次
const exposureBatches = ref<RecommendExposureBatch[]>([])
// 交叉观察器
let exposureObserver: any
const recommendStore = useRecommendStore()

// 获取猜你喜欢数据
const getHomeGoodsGuessLikeData = async () => {
  // 退出分页判断
  if (finish.value === true) {
    return uni.showToast({ icon: 'none', title: '没有更多数据~' })
  }
  const requestPageNum = pageParams.page_num
  const requestPageSize = pageParams.page_size
  // 推荐位序号需要和后端 request_id 会话内的分页位次保持一致，不能直接按当前已渲染条数累加。
  const requestStartIndex = (requestPageNum - 1) * requestPageSize
  pageParams.scene = props.scene
  pageParams.order_id = props.order_id
  pageParams.goods_id = props.goods_id
  await recommendStore.getAnonymousId()
  const res = await defRecommendService.RecommendGoods(pageParams)
  const sceneValue = props.scene
  const list = (res.goods_infos || []).map((item, index) => ({
    ...item,
    recommend_request_id: res.request_id,
    recommendScene: sceneValue,
    recommendIndex: requestStartIndex + index,
  }))
  // 首次返回请求编号后，后续翻页继续复用同一推荐会话。
  pageParams.request_id = res.request_id || pageParams.request_id
  guessList.value.push(...list)
  if (res.request_id > 0 && list.length > 0) {
    exposureBatches.value.push({
      request_id: res.request_id,
      scene: sceneValue,
      items: list.map((item) => ({
        goods_id: item.id,
        goods_num: 1,
        position: item.recommendIndex,
      })),
      exposed: false,
    })
  }
  const loadedCount = guessList.value.length
  // 后端返回了精确总数时，优先按总数分页；本地推荐未额外统计总数时，再退回到满页判断。
  if (res.total > 0) {
    if (loadedCount < res.total) {
      // 页码累加
      pageParams.page_num = requestPageNum + 1
    } else {
      finish.value = true
    }
  } else if (list.length >= requestPageSize) {
    // 页码累加
    pageParams.page_num = requestPageNum + 1
  } else {
    finish.value = true
  }
  await nextTick()
  await reportExposure()
}
// 重置数据
const resetData = () => {
  pageParams.page_num = 1
  pageParams.request_id = 0
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
    if (batch.exposed || !batch.request_id || batch.items.length === 0) {
      continue
    }
    batch.exposed = true
    try {
      await recommendStore.getAnonymousId()
      await defRecommendService.RecommendEventReport({
        event_type: RecommendEventType.EXPOSURE,
        recommend_context: {
          request_id: batch.request_id,
          scene: batch.scene,
        },
        items: batch.items,
      })
    } catch (error) {
      batch.exposed = false
      console.error(error)
    }
  }
}

const onTapGoods = async (item: GuessGoods) => {
  try {
    await recommendStore.getAnonymousId()
    await defRecommendService.RecommendEventReport({
      event_type: RecommendEventType.CLICK,
      recommend_context: {
        scene: item.recommendScene,
        request_id: item.recommend_request_id,
      },
      items: [
        {
          goods_id: item.id,
          goods_num: 1,
          position: item.recommendIndex,
        },
      ],
    })
  } catch (error) {
    console.error(error)
  }
  uni.navigateTo({
    url: goodsDetailUrl({
      id: item.id,
      scene: item.recommendScene,
      request_id: item.recommend_request_id,
      index: item.recommendIndex,
    }),
  })
}

// 初始化曝光观察器
const initExposureObserver = () => {
  const instance = getCurrentInstance()
  if (!instance) {
    return
  }
  exposureObserver = uni.createIntersectionObserver(instance)
  exposureObserver
    .relativeToViewport()
    .observe('.guess-root', (res: { intersectionRatio: number }) => {
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
    <view v-if="guessList.length" class="guess">
      <view v-for="item in guessList" :key="item.id" class="guess-item" @tap="onTapGoods(item)">
        <image class="image" mode="aspectFill" :src="formatSrc(item.picture)" />
        <view class="name"> {{ item.name }} </view>
        <view class="price">
          <text class="small">¥</text>
          <text>{{ formatPrice(item.price) }}</text>
        </view>
      </view>
    </view>
    <XtxEmptyState
      v-else-if="isEmpty"
      image="/static/images/empty_search.png"
      :text="emptyText"
      image-width="180rpx"
      image-height="150rpx"
      min-height="320rpx"
      padding="36rpx 0 56rpx"
    />
    <view v-if="!isEmpty" class="loading-text">
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
