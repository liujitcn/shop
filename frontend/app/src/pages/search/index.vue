<script setup lang="ts">
import { defGoodsInfoService } from '@/api/app/goods'
import { ref } from 'vue'
import type { GoodsInfo, PageGoodsInfoRequest } from '@/rpc/app/goods_info'
import { onLoad } from '@dcloudio/uni-app'
import { formatSrc, formatPrice } from '@/utils'

// 分类名、搜索词可能会经过路由层二次编码，这里统一做安全解码。
const decodeQueryText = (value?: string) => {
  if (!value) return ''
  let result = value
  for (let i = 0; i < 2; i++) {
    try {
      const decoded = decodeURIComponent(result)
      if (decoded === result) break
      result = decoded
    } catch {
      break
    }
  }
  return result
}

// 接收页面参数
const query = defineProps<{
  name: string
  categoryId: string
  categoryName: string
}>()

const decodedName = decodeQueryText(query.name)
const decodedCategoryName = decodeQueryText(query.categoryName)
// 分页参数
const pageParams: Required<PageGoodsInfoRequest> = {
  /** 商品名 */
  name: decodedName,
  /** 分类id */
  categoryId: query.categoryId ? Number(query.categoryId) : 0,
  /** 猜你喜欢 */
  guessLike: false,
  pageNum: 1,
  pageSize: 10,
}
// 猜你喜欢的列表
const goodsList = ref<GoodsInfo[]>([])
// 已结束标记
const finish = ref(false)
// 获取数据
const getGoodsData = async () => {
  // 退出分页判断
  if (finish.value === true) {
    return uni.showToast({ icon: 'none', title: '没有更多数据~' })
  }
  const res = await defGoodsInfoService.PageGoodsInfo(pageParams)
  // 数组追加
  const list = res.list || []
  goodsList.value.push(...list)
  // 分页条件
  if (goodsList.value.length < res.total) {
    // 页码累加
    pageParams.pageNum++
  } else {
    finish.value = true
  }
}

// 组件挂载完毕
onLoad(async () => {
  let title = '搜索结果'
  if (query.categoryId && decodedCategoryName) {
    title = decodedCategoryName
  }
  if (decodedName) {
    title = decodedName
  }
  // 动态设置标题
  await uni.setNavigationBarTitle({ title: title })
  await getGoodsData()
})

// 滚动触底
const onScrollToLower = async () => {
  await getGoodsData()
}
</script>

<template>
  <scroll-view enable-back-to-top scroll-y class="scroll-view" @scrolltolower="onScrollToLower">
    <view class="goods">
      <navigator
        v-for="item in goodsList"
        :key="item.id"
        class="goods-item"
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
  </scroll-view>
</template>

<style lang="scss">
page {
  height: 100%;
  background-color: #f4f4f4;
}
.goods {
  display: flex;
  flex-wrap: wrap;
  justify-content: space-between;
  padding: 0 20rpx;
  .goods-item {
    width: 345rpx;
    padding: 24rpx 20rpx 20rpx;
    margin-bottom: 20rpx;
    border-radius: 20rpx;
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
