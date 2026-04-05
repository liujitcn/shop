<script setup lang="ts">
import { defGoodsInfoService } from '@/api/app/goods'
import { formatPrice, formatSrc } from '@/utils'
import { onMounted, ref } from 'vue'
import type { GoodsInfo, PageGoodsInfoRequest } from '@/rpc/app/goods_info'

withDefaults(
  defineProps<{
    flush?: boolean
  }>(),
  {
    flush: false,
  },
)

// 分页参数
const pageParams: Required<PageGoodsInfoRequest> = {
  /** 商品名 */
  name: '',
  /** 分类id */
  categoryId: 0,
  /** 猜你喜欢 */
  guessLike: true,
  pageNum: 1,
  pageSize: 10,
}
// 猜你喜欢的列表
const guessList = ref<GoodsInfo[]>([])
// 已结束标记
const finish = ref(false)
// 获取猜你喜欢数据
const getHomeGoodsGuessLikeData = async () => {
  // 退出分页判断
  if (finish.value === true) {
    return uni.showToast({ icon: 'none', title: '没有更多数据~' })
  }
  const res = await defGoodsInfoService.PageGoodsInfo(pageParams)
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
}
// 重置数据
const resetData = () => {
  pageParams.pageNum = 1
  guessList.value = []
  finish.value = false
}
// 组件挂载完毕
onMounted(() => {
  getHomeGoodsGuessLikeData()
})
// 暴露方法
defineExpose({
  resetData,
  getMore: getHomeGoodsGuessLikeData,
})
</script>

<template>
  <view class="guess-panel">
    <!-- 猜你喜欢 -->
    <view class="caption">
      <text class="text">猜你喜欢</text>
    </view>
    <view class="guess" :class="{ 'guess-flush': flush }">
      <navigator
        v-for="item in guessList"
        :key="item.id"
        class="guess-item"
        :url="`/pages/goods/goods?id=${item.id}`"
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

.guess-panel {
  width: 100%;
}

/* 分类标题 */
.caption {
  display: flex;
  justify-content: center;
  width: 100%;
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

  &.guess-flush {
    padding: 0;
  }

  .guess-item {
    width: calc((100% - 20rpx) / 2);
    padding: 24rpx 20rpx 20rpx;
    margin-bottom: 20rpx;
    border-radius: 20rpx;
    box-sizing: border-box;
    overflow: hidden;
    background-color: #fff;
  }
  .image {
    width: 100%;
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
