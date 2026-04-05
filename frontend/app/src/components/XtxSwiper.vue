<script setup lang="ts">
import type { ShopBanner } from '@/rpc/app/shop_banner'
import { ref } from 'vue'
import { formatSrc } from '@/utils'
import { ShopBannerType } from '@/rpc/common/enum.ts'

const activeIndex = ref(0)

// 当 swiper 下标发生变化时触发
const onChange: UniHelper.SwiperOnChange = (ev) => {
  activeIndex.value = ev.detail.current
}
// 定义 props 接收
defineProps<{
  list: ShopBanner[]
}>()

const handleClick = (item: ShopBanner) => {
  if (!item.type || !item.href) return
  switch (item.type) {
    case ShopBannerType.GOODS_DETAIL:
      uni.navigateTo({ url: `/pages/goods/goods?${item.href}` })
      break
    case ShopBannerType.CATEGORY_DETAIL:
      uni.navigateTo({ url: `/pages/search/search?${item.href}` })
      break
    case ShopBannerType.WEB_VIEW:
      uni.navigateTo({
        url: `/pages/webview/webview?url=${encodeURIComponent(item.href)}`,
      })
      break
    case ShopBannerType.MINI:
      // #ifdef MP-WEIXIN
      uni.navigateToMiniProgram({
        appId: item.href,
        success(res) {
          console.log('跳转成功', res)
        },
        fail(err) {
          console.error('跳转失败', err)
          uni.showToast({ title: '跳转小程序失败', icon: 'none' })
        },
      })
      // #endif
      break
    default:
      console.warn('Unhandled banner type:', item.type)
  }
}
</script>

<template>
  <view class="carousel">
    <swiper :circular="true" :autoplay="false" :interval="3000" @change="onChange">
      <swiper-item v-for="item in list" :key="item.id" @tap="handleClick(item)">
        <image mode="aspectFill" class="image" :src="formatSrc(item.picture)" />
      </swiper-item>
    </swiper>
    <!-- 指示点 -->
    <view class="indicator">
      <text
        v-for="(item, index) in list"
        :key="item.id"
        class="dot"
        :class="{ active: index === activeIndex }"
      />
    </view>
  </view>
</template>

<style lang="scss">
@use './styles/XtxSwiper.scss';
</style>
