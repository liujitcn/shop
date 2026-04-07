<script setup lang="ts">
import { defGoodsCategoryService } from '@/api/app/goods_category'
import { defShopBannerService } from '@/api/app/shop_banner'
import { defShopHotService } from '@/api/app/shop_hot'
import type { ShopBanner } from '@/rpc/app/shop_banner'
import type { GoodsCategory } from '@/rpc/app/goods_category'
import type { ShopHot } from '@/rpc/app/shop_hot'
import { onLoad, onShow } from '@dcloudio/uni-app'
import { computed, ref } from 'vue'
import CustomNavbar from './components/CustomNavbar.vue'
import CategoryPanel from './components/CategoryPanel.vue'
import HotPanel from './components/HotPanel.vue'
import PageSkeleton from './components/PageSkeleton.vue'
import { useGuessList } from '@/composables'
import { useSettingStore } from '@/stores'
import { ShopBannerSite } from '@/rpc/common/enum.ts'
import { RecommendScene } from '@/rpc/app/recommend'

const settingStore = useSettingStore()
// 获取轮播图数据
const bannerList = ref<ShopBanner[]>([])
const getHomeBannerData = async () => {
  const res = await defShopBannerService.ListShopBanner({
    site: ShopBannerSite.INDEX,
  })
  bannerList.value = res.list || []
}

// 获取前台分类数据
const categoryList = ref<GoodsCategory[]>([])
const getHomeCategoryData = async () => {
  const res = await defGoodsCategoryService.ListGoodsCategory({
    parentId: 0,
  })
  categoryList.value = res.list || []
}

// 获取热门推荐数据
const hotList = ref<ShopHot[]>([])
const getHomeHotData = async () => {
  const res = await defShopHotService.ListShopHot({})
  hotList.value = res.list || []
}

// 是否加载中标记
const isLoading = ref(false)

// 页面加载
onLoad(async () => {
  const switchTabIndex = uni.getStorageSync('SwitchTabIndex')
  if (!switchTabIndex) {
    isLoading.value = true
    await Promise.all([getHomeBannerData(), getHomeCategoryData(), getHomeHotData()])
    isLoading.value = false
  }
})

onShow(async () => {
  await settingStore.loadData()
  const switchTabIndex = uni.getStorageSync('SwitchTabIndex')
  if (switchTabIndex) {
    isLoading.value = true
    await Promise.all([getHomeBannerData(), getHomeCategoryData(), getHomeHotData()])
    isLoading.value = false
    uni.removeStorageSync('SwitchTabIndex')
  }
})

// 猜你喜欢组合式函数调用
const { guessRef, onScrollToLower } = useGuessList()
// 当前下拉刷新状态
const isTriggered = ref(false)
// 自定义下拉刷新被触发
const onRefresh = async () => {
  // 开始动画
  isTriggered.value = true
  // 重置猜你喜欢组件数据
  guessRef.value?.resetData()
  await Promise.all([
    getHomeBannerData(),
    getHomeCategoryData(),
    getHomeHotData(),
    guessRef.value?.getMore(),
  ])
  // 关闭动画
  isTriggered.value = false
}

// 定义分享配置
const shareConfig = computed(() => {
  return {
    title: '',
    path: `/pages/index/index`,
    imageUrl: '',
  }
})

// 分享给朋友
const onShareAppMessage = () => {
  return shareConfig.value
}

// 分享到朋友圈
const onShareTimeline = () => {
  return shareConfig.value
}
</script>

<template>
  <view class="viewport">
    <!-- 自定义导航栏 -->
    <CustomNavbar />
    <!-- 滚动容器 -->
    <scroll-view
      enable-back-to-top
      refresher-enabled
      :refresher-triggered="isTriggered"
      class="scroll-view"
      scroll-y
      @refresherrefresh="onRefresh"
      @scrolltolower="onScrollToLower"
    >
      <PageSkeleton v-if="isLoading" />
      <template v-else>
        <!-- 自定义轮播图 -->
        <XtxSwiper v-if="bannerList.length" :list="bannerList" />
        <!-- 分类面板 -->
        <CategoryPanel v-if="categoryList.length" :list="categoryList" />
        <!-- 热门推荐 -->
        <HotPanel v-if="hotList.length" :list="hotList" />
        <!-- 猜你喜欢 -->
        <XtxGuess ref="guessRef" title="为你推荐" :scene="RecommendScene.HOME" />
      </template>
    </scroll-view>
  </view>
</template>

<style lang="scss">
page {
  background-color: #f7f7f7;
  height: 100%;
  overflow: hidden;
}

.viewport {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.scroll-view {
  flex: 1;
  overflow: hidden;
}
</style>
