<script setup lang="ts">
import type { GoodsCategory } from '@/rpc/app/goods_category'
import { formatSrc } from '@/utils'
import defaultCategoryIcon from '@/static/images/logo_icon.png'
import { ref } from 'vue'

// 定义 props 接收数据
defineProps<{
  list: GoodsCategory[]
}>()

const failedImageIds = ref<number[]>([])

const getCategoryIcon = (item: GoodsCategory) => {
  if (!item.picture || failedImageIds.value.includes(item.id)) {
    return defaultCategoryIcon
  }
  return formatSrc(item.picture)
}

const onImageError = (id: number) => {
  if (!failedImageIds.value.includes(id)) {
    failedImageIds.value = [...failedImageIds.value, id]
  }
}

// H5 可依赖 title 展示全称，其它端补一个长按提示兜底。
const showCategoryName = (name: string) => {
  uni.showToast({
    title: name,
    icon: 'none',
  })
}
</script>

<template>
  <view class="panel category-panel">
    <view class="category">
      <navigator
        class="category-item"
        hover-class="none"
        :url="`/pages/search/index?categoryId=${item.id}&categoryName=${encodeURIComponent(
          item.name,
        )}`"
        v-for="item in list"
        :key="item.id"
        @longpress="showCategoryName(item.name)"
      >
        <view class="icon-box">
          <image
            class="icon"
            :src="getCategoryIcon(item)"
            mode="aspectFit"
            @error="onImageError(item.id)"
          ></image>
        </view>
        <view class="text" :title="item.name">{{ item.name }}</view>
      </navigator>
    </view>
  </view>
</template>

<style lang="scss">
@import '../styles/category.scss';
</style>
