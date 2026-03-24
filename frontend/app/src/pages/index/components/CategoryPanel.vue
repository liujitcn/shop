<script setup lang="ts">
import type { GoodsCategory } from '@/rpc/app/goods_category'
import { formatSrc } from '@/utils'

// 定义 props 接收数据
defineProps<{
  list: GoodsCategory[]
}>()
</script>

<template>
  <view class="category">
    <navigator
      class="category-item"
      hover-class="none"
      :url="`/pages/search/index?categoryId=${item.id}&categoryName=${encodeURIComponent(
        item.name,
      )}`"
      v-for="item in list"
      :key="item.id"
    >
      <image class="icon" :src="formatSrc(item.picture)"></image>
      <text class="text">{{ item.name }}</text>
    </navigator>
  </view>
</template>

<style lang="scss">
@import '../styles/category.scss';
</style>
