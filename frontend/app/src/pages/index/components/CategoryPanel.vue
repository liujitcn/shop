<script setup lang="ts">
import type { GoodsCategory } from '@/rpc/app/goods_category'
import { formatSrc } from '@/utils'
import defaultCategoryIcon from '@/static/images/logo_icon.png'
import { computed, ref } from 'vue'

// 定义 props 接收数据
const props = defineProps<{
  list: GoodsCategory[]
}>()

type CategoryDisplayItem = GoodsCategory & {
  isMore?: boolean
}

const MAX_VISIBLE_COUNT = 8
const MAX_NAME_LENGTH = 5

const failedImageIds = ref<number[]>([])

const visibleList = computed<CategoryDisplayItem[]>(() => {
  if (props.list.length <= MAX_VISIBLE_COUNT) {
    return props.list
  }

  return [
    ...props.list.slice(0, MAX_VISIBLE_COUNT - 1),
    {
      id: -1,
      name: '更多',
      picture: '',
      isMore: true,
    },
  ]
})

const getCategoryIcon = (item: CategoryDisplayItem) => {
  if (item.isMore || !item.picture || failedImageIds.value.includes(item.id)) {
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

const formatCategoryName = (name: string) => {
  if (name.length <= MAX_NAME_LENGTH) {
    return name
  }
  return `${name.slice(0, MAX_NAME_LENGTH)}...`
}

const shouldShowFullName = (name: string) => {
  return name.length > MAX_NAME_LENGTH
}

const onTapCategory = (item: CategoryDisplayItem) => {
  if (item.isMore) {
    uni.switchTab({
      url: '/pages/category/category',
    })
    return
  }

  uni.navigateTo({
    url: `/pages/search/index?categoryId=${item.id}&categoryName=${encodeURIComponent(item.name)}`,
  })
}
</script>

<template>
  <view class="panel category-panel">
    <view class="category">
      <view
        class="category-item"
        v-for="item in visibleList"
        :key="item.id"
        hover-class="none"
        @tap="onTapCategory(item)"
        @longpress="shouldShowFullName(item.name) && showCategoryName(item.name)"
      >
        <view class="icon-box" :class="{ 'icon-box-more': item.isMore }">
          <image
            class="icon"
            :src="getCategoryIcon(item)"
            mode="aspectFit"
            @error="onImageError(item.id)"
          ></image>
        </view>
        <view class="text" :title="shouldShowFullName(item.name) ? item.name : ''">
          {{ formatCategoryName(item.name) }}
        </view>
      </view>
    </view>
  </view>
</template>

<style lang="scss">
@import '../styles/category.scss';
</style>
