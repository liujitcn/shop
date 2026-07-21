<script setup lang="ts">
import type { ShopBanner } from '@/rpc/shop/app/v1/shop_banner'
import { ref } from 'vue'
import { formatSrc } from '@/utils'
import { ShopBannerType } from '@/rpc/shop/common/v1/enum.ts'
import { goodsDetailUrl, searchPageUrl } from '@/utils/navigation'

const activeIndex = ref(0)

// 当 swiper 下标发生变化时触发
const onChange: UniHelper.SwiperOnChange = (ev) => {
  activeIndex.value = ev.detail.current
}
// 定义 props 接收
defineProps<{
  list: ShopBanner[]
}>()

/** 从结构化 query 或兼容旧路径中读取正整数目标编号。 */
const resolveTargetID = (href: string, queryKeys: string[], legacySegment: string) => {
  const normalizedHref = href.trim()
  if (/^\d+$/.test(normalizedHref)) {
    return Number(normalizedHref)
  }

  const query = normalizedHref.includes('?')
    ? normalizedHref.slice(normalizedHref.indexOf('?') + 1)
    : normalizedHref.replace(/^[#?&]+/, '')
  for (const item of query.split('&')) {
    const [key, value] = item.split('=', 2)
    if (queryKeys.includes(key) && /^\d+$/.test(value || '')) {
      return Number(value)
    }
  }

  const legacyMatch = normalizedHref.match(new RegExp(`/${legacySegment}/(\\d+)(?:[/?#]|$)`))
  return legacyMatch ? Number(legacyMatch[1]) : 0
}

/** 提示轮播配置无效，避免打开缺少业务参数的页面。 */
const showInvalidTarget = () => {
  void uni.showToast({ title: '轮播跳转配置无效', icon: 'none' })
}

/** 按轮播类型解析目标并进入对应页面。 */
const handleClick = (item: ShopBanner) => {
  if (!item.type || !item.href) {
    return
  }

  switch (item.type) {
    case ShopBannerType.BANNER_GOODS_DETAIL: {
      const goodsID = resolveTargetID(item.href, ['id', 'goods_id'], 'goods')
      if (!goodsID) {
        showInvalidTarget()
        return
      }
      void uni.navigateTo({ url: goodsDetailUrl(goodsID) })
      return
    }
    case ShopBannerType.CATEGORY_DETAIL: {
      const categoryID = resolveTargetID(item.href, ['category_id'], 'category')
      if (!categoryID) {
        showInvalidTarget()
        return
      }
      void uni.navigateTo({ url: searchPageUrl({ category_id: categoryID }) })
      return
    }
    case ShopBannerType.WEB_VIEW:
      void uni.navigateTo({ url: `/pages/webview/webview?url=${encodeURIComponent(item.href)}` })
      return
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
      return
    default:
      return
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
@use './styles/ShopSwiper.scss';
</style>
