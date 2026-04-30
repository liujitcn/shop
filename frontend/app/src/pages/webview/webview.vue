<template>
  <view class="webview-container">
    <!-- 小程序/APP 使用 web-view 组件 -->
    <!-- #ifdef MP-WEIXIN || MP-ALIPAY || APP-PLUS -->
    <web-view :src="url" @message="handleMessage"></web-view>
    <!-- #endif -->

    <!-- H5 直接跳转 -->
    <!-- #ifdef H5 -->
    <iframe v-if="isH5" :src="url" frameborder="0" class="h5-iframe"></iframe>
    <!-- #endif -->
  </view>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { onLoad } from '@dcloudio/uni-app'

const url = ref('')
const isH5 = ref(false)

onLoad((query) => {
  url.value = decodeURIComponent(query?.url || '')

  // #ifdef H5
  isH5.value = true
  // 直接跳转（可选）
  // window.location.href = url.value
  // #endif
})

// 小程序接收消息
const handleMessage = (e: any) => {
  console.log('收到H5消息:', e.detail)
}
</script>

<style scoped>
.webview-container {
  flex: 1;
  height: 100vh;
}
.h5-iframe {
  width: 100%;
  height: 100vh;
}
</style>
