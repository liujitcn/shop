<script setup lang="ts">
import { computed, useSlots } from 'vue'

const emit = defineEmits<{
  action: []
}>()

const props = withDefaults(
  defineProps<{
    image: string
    text: string
    buttonText?: string
    imageWidth?: string
    imageHeight?: string
    minHeight?: string
    padding?: string
    mode?: string
  }>(),
  {
    imageWidth: '220rpx',
    imageHeight: '220rpx',
    minHeight: '420rpx',
    padding: '96rpx 48rpx',
    mode: 'aspectFit',
  },
)

const slots = useSlots()
const hasAction = computed(() => Boolean(props.buttonText || slots.default))
</script>

<template>
  <view
    class="xtx-empty-state"
    :style="{
      minHeight: props.minHeight,
      padding: props.padding,
    }"
  >
    <image
      class="xtx-empty-state__image"
      :src="props.image"
      :mode="props.mode"
      :style="{
        width: props.imageWidth,
        height: props.imageHeight,
      }"
    />
    <view class="xtx-empty-state__text">{{ props.text }}</view>
    <view v-if="hasAction" class="xtx-empty-state__action">
      <button v-if="props.buttonText" class="xtx-empty-state__button" @tap="emit('action')">
        {{ props.buttonText }}
      </button>
      <slot v-else />
    </view>
  </view>
</template>

<style lang="scss">
:host {
  display: block;
}

.xtx-empty-state {
  display: flex;
  align-items: center;
  justify-content: center;
  flex-direction: column;
  box-sizing: border-box;
  text-align: center;
}

.xtx-empty-state__image {
  flex-shrink: 0;
}

.xtx-empty-state__text {
  max-width: 100%;
  margin-top: 18rpx;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 26rpx;
  line-height: 1.4;
  color: #666;
}

.xtx-empty-state__action {
  margin-top: 24rpx;
}

.xtx-empty-state__button {
  min-width: 240rpx;
  height: 60rpx;
  padding: 0 32rpx;
  border-radius: 60rpx;
  line-height: 60rpx;
  font-size: 26rpx;
  color: #fff;
  background-color: #27ba9b;
}
</style>
