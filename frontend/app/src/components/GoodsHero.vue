<script setup lang="ts">
import { computed, ref } from 'vue'
import { formatPrice, formatSrc } from '@/utils'

const props = withDefaults(
  defineProps<{
    pictures: string[]
    price: number
    saleNum: number
    name: string
    desc?: string
    imageHeight?: string
    imageMode?: 'aspectFill' | 'aspectFit' | 'widthFix' | 'heightFix'
    previewable?: boolean
  }>(),
  {
    desc: '',
    imageHeight: '750rpx',
    imageMode: 'aspectFill',
    previewable: true,
  },
)

const emit = defineEmits<{
  nameTap: []
}>()

const activeIndex = ref(0)
const pictureList = computed(() => props.pictures.filter(Boolean))

const trimZeroDecimal = (value: string) => value.replace(/\.0$/, '')

const saleText = computed(() => {
  if (!props.saleNum) return '销量 0'
  if (props.saleNum >= 10000) {
    return `销量 ${trimZeroDecimal((props.saleNum / 10000).toFixed(1))}万+`
  }
  return `销量 ${props.saleNum}`
})

const onChange: UniHelper.SwiperOnChange = (event) => {
  activeIndex.value = event.detail.current
}

const onTapImage = () => {
  if (!props.previewable || !pictureList.value.length) return

  const urls = pictureList.value.map((item) => formatSrc(item))
  uni.previewImage({
    current: urls[activeIndex.value] || urls[0],
    urls,
  })
}
</script>

<template>
  <view class="goods-hero">
    <view class="goods-hero__preview" :style="{ height: props.imageHeight }">
      <swiper
        v-if="pictureList.length"
        class="goods-hero__swiper"
        :circular="pictureList.length > 1"
        @change="onChange"
      >
        <swiper-item v-for="picture in pictureList" :key="picture">
          <image
            class="goods-hero__image"
            :mode="props.imageMode"
            :src="formatSrc(picture)"
            @tap="onTapImage"
          />
        </swiper-item>
      </swiper>
      <view v-else class="goods-hero__placeholder">暂无图片</view>
      <view v-if="pictureList.length" class="goods-hero__indicator">
        <text class="goods-hero__current">{{ activeIndex + 1 }}</text>
        <text class="goods-hero__split">/</text>
        <text class="goods-hero__total">{{ pictureList.length }}</text>
      </view>
    </view>

    <view class="goods-hero__meta">
      <view class="goods-hero__price">
        <text class="goods-hero__symbol">¥</text>
        <text class="goods-hero__number">{{ formatPrice(props.price) }}</text>
        <text class="goods-hero__sales">{{ saleText }}</text>
      </view>
      <view class="goods-hero__name ellipsis" @tap="emit('nameTap')">{{ props.name }}</view>
      <view v-if="props.desc" class="goods-hero__desc">{{ props.desc }}</view>
    </view>
  </view>
</template>

<style lang="scss">
.goods-hero {
  background-color: #fff;
}

.goods-hero__preview {
  position: relative;
  background-color: #f7f7f7;
}

.goods-hero__swiper,
.goods-hero__image,
.goods-hero__placeholder {
  width: 100%;
  height: 100%;
}

.goods-hero__placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  color: #999;
  font-size: 28rpx;
}

.goods-hero__indicator {
  position: absolute;
  right: 30rpx;
  bottom: 30rpx;
  height: 40rpx;
  padding: 0 24rpx;
  border-radius: 30rpx;
  color: #fff;
  font-family: Arial, Helvetica, sans-serif;
  line-height: 40rpx;
  background-color: rgba(0, 0, 0, 0.3);
}

.goods-hero__current {
  font-size: 26rpx;
}

.goods-hero__split {
  margin: 0 1rpx 0 2rpx;
  font-size: 24rpx;
}

.goods-hero__total {
  font-size: 24rpx;
}

.goods-hero__meta {
  position: relative;
  border-bottom: 1rpx solid #eaeaea;
}

.goods-hero__price {
  position: relative;
  display: flex;
  align-items: center;
  height: 104rpx;
  padding: 0 30rpx;
  box-sizing: border-box;
  color: #fff;
  font-size: 30rpx;
  background-color: #35c8a9;
}

.goods-hero__number {
  font-size: 48rpx;
}

.goods-hero__sales {
  position: absolute;
  top: 45rpx;
  right: 30rpx;
  color: rgba(255, 255, 255, 0.9);
  font-size: 22rpx;
}

.goods-hero__name {
  max-height: 88rpx;
  margin: 20rpx;
  color: #333;
  font-size: 32rpx;
  line-height: 1.4;
}

.goods-hero__desc {
  padding: 0 20rpx 30rpx;
  color: #cf4444;
  font-size: 24rpx;
  line-height: 1;
}
</style>
