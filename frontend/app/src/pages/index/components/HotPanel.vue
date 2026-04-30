<script setup lang="ts">
import type { ShopHot } from '@/rpc/app/v1/shop_hot'
import { formatSrc } from '@/utils'

// 定义 props 接收数据
defineProps<{
  list: ShopHot[]
}>()
</script>

<template>
  <!-- 推荐专区 -->
  <view class="panel hot">
    <view v-for="item in list" :key="item.id" class="item">
      <view class="title">
        <text class="title-text">{{ item.title }}</text>
        <text class="title-desc">{{ item.desc }}</text>
      </view>
      <navigator hover-class="none" :url="`/pages/hot/hot?id=${item.id}`" class="cards">
        <image
          v-for="src in item.picture"
          :key="src"
          class="image"
          mode="aspectFit"
          :src="formatSrc(src)"
        />
      </navigator>
    </view>
  </view>
</template>

<style lang="scss">
@use '../styles/hot.scss';
</style>
