<script setup lang="ts">
import { defShopHotService } from '@/api/app/shop_hot'
import type { ShopHotItem } from '@/rpc/app/shop_hot'
import type { GoodsInfo } from '@/rpc/app/goods_info'
import { onLoad } from '@dcloudio/uni-app'
import { ref } from 'vue'
import { formatSrc, formatPrice } from '@/utils'

// uniapp 获取页面参数
const query = defineProps<{
  id: string
}>()
// 推荐封面图
const bannerPicture = ref('')
// 推荐选项
const subTypes = ref<
  (ShopHotItem & {
    finish?: boolean
    goodsItems?: GoodsInfo[]
    total?: number
    pageNum?: number
    pageSize?: number
  })[]
>([])
// 高亮的下标
const activeIndex = ref(0)
// 获取热门推荐数据
const getShopHotItem = async () => {
  const res = await defShopHotService.ListShopHotItem({
    value: Number(query.id),
  })
  bannerPicture.value = res.banner
  // 动态设置标题
  await uni.setNavigationBarTitle({ title: res.title })
  subTypes.value = res.list || []
  subTypes.value.map((item) => {
    item.pageNum = 1
    item.pageSize = 10
    defShopHotService
      .PageShopHotGoods({
        /** 选项id */
        hotItemId: item.id,
        /** 当前页码 */
        pageNum: item.pageNum,
        /** 每一页的行数 */
        pageSize: item.pageSize,
      })
      .then((res) => {
        item.goodsItems = res.list || []
        item.total = res.total || 0
        item.finish = item.total === 0 || item.goodsItems.length < 10
        console.log(item)
      })
  })
}

// 页面加载
onLoad(() => {
  getShopHotItem()
})

// 滚动触底
const onScrollToLower = async () => {
  // 获取当前选项
  const currSubTypes = subTypes.value[activeIndex.value]
  // 分页条件
  if (currSubTypes.goodsItems!.length < currSubTypes.total!) {
    // 当前页码累加
    currSubTypes.pageNum!++
  } else {
    // 标记已结束
    currSubTypes.finish = true
    // 退出并轻提示
    return uni.showToast({ icon: 'none', title: '没有更多数据了~' })
  }

  // 调用API传参
  const res = await defShopHotService.PageShopHotGoods({
    /** 选项id */
    hotItemId: currSubTypes.id,
    /** 当前页码 */
    pageNum: currSubTypes.pageNum!,
    /** 每一页的行数 */
    pageSize: currSubTypes.pageSize!,
  })
  const list = res.list || []
  // 数组追加
  currSubTypes.goodsItems!.push(...list)
}
</script>

<template>
  <view class="viewport">
    <!-- 推荐封面图 -->
    <view class="cover">
      <image class="image" mode="widthFix" :src="formatSrc(bannerPicture)" />
    </view>
    <!-- 推荐选项 -->
    <view class="tabs">
      <text
        v-for="(item, index) in subTypes"
        :key="item.id"
        class="text"
        :class="{ active: index === activeIndex }"
        @tap="activeIndex = index"
        >{{ item.title }}</text
      >
    </view>
    <!-- 推荐列表 -->
    <scroll-view
      v-for="(item, index) in subTypes"
      v-show="activeIndex === index"
      :key="item.id"
      enable-back-to-top
      scroll-y
      class="scroll-view"
      @scrolltolower="onScrollToLower"
    >
      <view class="goods">
        <navigator
          v-for="goods in item.goodsItems"
          :key="goods.id"
          hover-class="none"
          class="navigator"
          :url="`/pages/goods/goods?id=${goods.id}`"
        >
          <image class="thumb" :src="formatSrc(goods.picture)" />
          <view class="name ellipsis">{{ goods.name }}</view>
          <view class="price">
            <text class="symbol">¥</text>
            <text class="number">{{ formatPrice(goods.price) }}</text>
          </view>
        </navigator>
      </view>
      <view class="loading-text">
        {{ item.finish ? '没有更多数据了~' : '正在加载...' }}
      </view>
    </scroll-view>
  </view>
</template>

<style lang="scss">
page {
  height: 100%;
  background-color: #f4f4f4;
}
.viewport {
  display: flex;
  flex-direction: column;
  height: 100%;
  padding: 180rpx 0 0;
  position: relative;
}
.cover {
  width: 750rpx;
  height: 225rpx;
  border-radius: 0 0 40rpx 40rpx;
  overflow: hidden;
  position: absolute;
  left: 0;
  top: 0;
  .image {
    width: 750rpx;
  }
}
.scroll-view {
  flex: 1;
}
.tabs {
  display: flex;
  justify-content: space-evenly;
  height: 100rpx;
  line-height: 90rpx;
  margin: 0 20rpx;
  font-size: 28rpx;
  border-radius: 10rpx;
  box-shadow: 0 4rpx 5rpx rgba(200, 200, 200, 0.3);
  color: #333;
  background-color: #fff;
  position: relative;
  z-index: 9;
  .text {
    margin: 0 20rpx;
    position: relative;
  }
  .active {
    &::after {
      content: '';
      width: 40rpx;
      height: 4rpx;
      transform: translate(-50%);
      background-color: #27ba9b;
      position: absolute;
      left: 50%;
      bottom: 24rpx;
    }
  }
}
.goods {
  display: flex;
  flex-wrap: wrap;
  justify-content: space-between;
  padding: 0 20rpx 20rpx;
  .navigator {
    width: 342rpx;
    padding: 20rpx;
    margin-top: 20rpx;
    border-radius: 10rpx;
    background-color: #fff;
  }
  .thumb {
    width: 305rpx;
    height: 305rpx;
  }
  .name {
    height: 88rpx;
    font-size: 26rpx;
  }
  .price {
    line-height: 1;
    color: #cf4444;
    font-size: 30rpx;
  }
  .symbol {
    font-size: 70%;
  }
  .decimal {
    font-size: 70%;
  }
}

.loading-text {
  text-align: center;
  font-size: 28rpx;
  color: #666;
  padding: 20rpx 0 50rpx;
}
</style>
