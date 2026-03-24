<script setup lang="ts">
import { onLoad } from '@dcloudio/uni-app'
import { useSettingStore } from '@/stores'
import { ref } from 'vue'

const settingStore = useSettingStore()
const content = ref('')

// 加载协议内容
const loadProtocol = async (type?: string) => {
  if (!settingStore.getData('serviceProtocol') && !settingStore.getData('privacyProtocol')) {
    await settingStore.loadData()
  }

  const isPrivacy = type === 'privacy'
  const title = isPrivacy ? '隐私协议' : '服务条款'
  const key = isPrivacy ? 'privacyProtocol' : 'serviceProtocol'

  uni.setNavigationBarTitle({
    title,
  })
  content.value = settingStore.getData(key) || ''
}

onLoad((query) => {
  loadProtocol(query?.type)
})
</script>
<template>
  <view class="container">
    <rich-text :nodes="content" />
  </view>
</template>
