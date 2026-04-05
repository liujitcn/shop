<script setup lang="ts">
import type { GoodsCategory } from '@/rpc/app/goods_category'
import { formatSrc } from '@/utils'
import { computed } from 'vue'

// 定义 props 接收数据
const props = defineProps<{
  list: GoodsCategory[]
}>()

type CategoryDisplayItem = Pick<GoodsCategory, 'id' | 'name' | 'picture'> &
  Partial<GoodsCategory> & {
    isMore?: boolean
    pictures?: string[]
  }

const MAX_VISIBLE_COUNT = 8
const MAX_NAME_LENGTH = 5

const visibleList = computed<CategoryDisplayItem[]>(() => {
  if (props.list.length <= MAX_VISIBLE_COUNT) {
    return props.list
  }

  const hiddenList = props.list.slice(MAX_VISIBLE_COUNT - 1)
  const fallbackList = props.list.slice(0, MAX_VISIBLE_COUNT - 1)
  const morePictures = [...hiddenList, ...fallbackList]
    .map((item) => item.picture)
    .filter(Boolean)
    .slice(0, 4)

  return [
    ...props.list.slice(0, MAX_VISIBLE_COUNT - 1),
    {
      id: -1,
      name: '全部分类',
      picture: '',
      isMore: true,
      pictures: morePictures,
    },
  ]
})

const formatCategoryName = (name: string) => {
  if (name.length <= MAX_NAME_LENGTH) {
    return name
  }
  return `${name.slice(0, MAX_NAME_LENGTH)}...`
}

const shouldShowFullName = (name: string) => {
  return name.length > MAX_NAME_LENGTH
}

const showCategoryName = (name: string) => {
  uni.showToast({
    title: name,
    icon: 'none',
  })
}

const onTapCategory = (item: CategoryDisplayItem) => {
  if (item.isMore) {
    uni.switchTab({
      url: '/pages/category/category',
    })
    return
  }

  uni.navigateTo({
    url: `/pages/search/index?categoryId=${item.id}&categoryName=${item.name}`,
  })
}
</script>

<template>
  <view class="category">
    <view
      class="category-item"
      v-for="item in visibleList"
      :key="item.id"
      hover-class="none"
      @tap="onTapCategory(item)"
      @longpress="shouldShowFullName(item.name) && showCategoryName(item.name)"
    >
      <view v-if="item.isMore" class="icon icon-grid">
        <image
          v-for="(picture, index) in item.pictures"
          :key="`${item.id}-${index}`"
          class="icon-grid-item"
          :src="formatSrc(picture)"
          mode="aspectFill"
        />
      </view>
      <image v-else class="icon" :src="formatSrc(item.picture)" mode="aspectFit"></image>
      <view class="text" :title="shouldShowFullName(item.name) ? item.name : ''">
        {{ formatCategoryName(item.name) }}
      </view>
    </view>
  </view>
</template>

<style lang="scss">
@use '../styles/category.scss';
</style>
