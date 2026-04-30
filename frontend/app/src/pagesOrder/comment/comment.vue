<script setup lang="ts">
import { computed, ref } from 'vue'
import { defCommentService } from '@/api/app/comment'
import { uploadFileList } from '@/utils/file'

const COMMENT_CENTER_DONE_PAGE = '/pagesOrder/comment/center?tab=done'
const query = defineProps<{
  order_id?: string | number
  goods_id?: string | number
  goods_name?: string
  goods_picture?: string
  sku_code?: string
  sku_desc?: string
}>()
const { safeAreaInsets } = uni.getSystemInfoSync()
const reviewText = ref('')
const isAnonymous = ref(false)
const reviewImages = ref<string[]>([])
const isSubmitting = ref(false)
const minRating = 1
const maxRating = 5
const starList = [1, 2, 3, 4, 5]
const ratingItems = [
  { key: 'goods', label: '商品评价' },
  { key: 'package', label: '包装评价' },
  { key: 'delivery', label: '送货评价' },
]
const maxImageCount = 6
const ratingMap = ref<Record<string, number>>({
  goods: 5,
  package: 5,
  delivery: 5,
})

const reviewLength = computed(() => reviewText.value.length)
const canAddImage = computed(() => reviewImages.value.length < maxImageCount)

const toNumber = (value?: string | number) => {
  const result = Number(value)
  return Number.isFinite(result) ? result : 0
}

const normalizeRating = (value?: number) => {
  const result = Number(value)
  if (!Number.isFinite(result)) {
    return minRating
  }
  return Math.max(minRating, Math.min(result, maxRating))
}

const canPublish = computed(() => {
  return ratingItems.every((item) => normalizeRating(ratingMap.value[item.key]) >= minRating)
})

const routeOrderId = computed(() => toNumber(query.order_id))
const routeGoodsId = computed(() => toNumber(query.goods_id))
const onNavigateBack = () => {
  const pages = getCurrentPages()
  if (pages.length > 1) {
    uni.navigateBack()
    return
  }
  uni.switchTab({ url: '/pages/index/index' })
}

const onPublish = async () => {
  if (!canPublish.value) {
    void uni.showToast({
      title: '评分最低 1 星',
      icon: 'none',
    })
    return
  }

  if (isSubmitting.value) {
    return
  }

  // 路由缺少订单商品关联信息时，无法继续提交真实评价请求。
  if (routeOrderId.value <= 0 || routeGoodsId.value <= 0 || !String(query.sku_code || '').trim()) {
    void uni.showToast({
      title: '待评价商品不存在',
      icon: 'none',
    })
    return
  }

  isSubmitting.value = true
  try {
    await defCommentService.CreateComment({
      order_id: routeOrderId.value,
      goods_id: routeGoodsId.value,
      sku_code: String(query.sku_code || '').trim(),
      content: reviewText.value.trim(),
      img: reviewImages.value,
      is_anonymous: isAnonymous.value,
      goods_score: normalizeRating(ratingMap.value.goods),
      package_score: normalizeRating(ratingMap.value.package),
      delivery_score: normalizeRating(ratingMap.value.delivery),
    })
    void uni.redirectTo({ url: COMMENT_CENTER_DONE_PAGE })
  } finally {
    isSubmitting.value = false
  }
}

const onToggleAnonymous = () => {
  isAnonymous.value = !isAnonymous.value
}

const onSelectRating = (key: string, value: number) => {
  ratingMap.value = {
    ...ratingMap.value,
    [key]: normalizeRating(value),
  }
}

const getRatingText = (value: number) => {
  if (value >= 5) {
    return '非常好'
  }
  if (value >= 4) {
    return '比较好'
  }
  if (value >= 3) {
    return '一般'
  }
  return '待提升'
}

const onChooseImages = () => {
  const remainCount = maxImageCount - reviewImages.value.length
  if (remainCount <= 0) {
    return
  }

  uni.chooseImage({
    count: remainCount,
    sizeType: ['compressed'],
    sourceType: ['album', 'camera'],
    success: async (res) => {
      const tempFilePaths = Array.isArray(res.tempFilePaths)
        ? res.tempFilePaths
        : [res.tempFilePaths]
      const fileList = await uploadFileList('comment', tempFilePaths.slice(0, remainCount))
      reviewImages.value = [...reviewImages.value, ...fileList.map((item) => item.url)].slice(
        0,
        maxImageCount,
      )
    },
  })
}

const onPreviewImage = (index: number) => {
  uni.previewImage({
    current: reviewImages.value[index],
    urls: reviewImages.value,
  })
}

const onRemoveImage = (index: number) => {
  reviewImages.value = reviewImages.value.filter((_, imageIndex) => imageIndex !== index)
}
</script>

<template>
  <view class="review-page">
    <view class="review-header" :style="{ paddingTop: `${safeAreaInsets?.top || 0}px` }">
      <view class="review-nav">
        <view class="back-button" @tap="onNavigateBack">‹</view>
        <view class="review-title-wrap">
          <text class="review-title">写评价</text>
        </view>
        <button
          class="publish-button"
          :class="{ disabled: !canPublish || isSubmitting }"
          @tap="onPublish"
        >
          {{ isSubmitting ? '发布中' : '发布' }}
        </button>
      </view>
    </view>

    <scroll-view scroll-y class="review-body">
      <view class="editor-card">
        <view class="editor-head">
          <view class="editor-title">分享真实体验</view>
          <view class="word-count">{{ reviewLength }}/500</view>
        </view>
        <textarea
          v-model="reviewText"
          class="review-textarea"
          maxlength="500"
          placeholder="说说商品质量、包装、物流或使用感受"
          placeholder-class="review-placeholder"
        />

        <view class="image-section">
          <view class="image-list">
            <view
              v-for="(image, index) in reviewImages"
              :key="`${image}-${index}`"
              class="image-item"
              @tap="onPreviewImage(index)"
            >
              <image class="review-image" :src="image" mode="aspectFill" />
              <view class="image-remove" @tap.stop="onRemoveImage(index)">×</view>
            </view>
            <view v-if="canAddImage" class="image-add" @tap="onChooseImages">
              <text class="icon-camera-plus" />
              <view>上传图片</view>
            </view>
          </view>
        </view>
      </view>

      <view class="rating-card">
        <view class="rating-card-head">
          <view class="rating-card-title">评分</view>
          <view class="rating-card-tip">内容图片选填，最低 1 星</view>
        </view>
        <view v-for="item in ratingItems" :key="item.key" class="rating-row">
          <text class="rating-name">{{ item.label }}</text>
          <text class="rating-text">{{ getRatingText(ratingMap[item.key]) }}</text>
          <view class="rating-stars">
            <text
              v-for="star in starList"
              :key="star"
              class="rating-star"
              :class="{ active: star <= ratingMap[item.key] }"
              @tap="onSelectRating(item.key, star)"
              >★</text
            >
          </view>
        </view>
      </view>

      <view class="anonymous-card" @tap="onToggleAnonymous">
        <view class="checkbox" :class="{ checked: isAnonymous }">
          <text v-if="isAnonymous">✓</text>
        </view>
        <view>
          <view class="anonymous-title">匿名评价</view>
          <view class="anonymous-desc">匿名后其他买家不会看到你的昵称</view>
        </view>
      </view>
    </scroll-view>
  </view>
</template>

<style lang="scss">
page {
  height: 100%;
  background-color: #f4f4f4;
}

.review-page {
  height: 100%;
  display: flex;
  flex-direction: column;
  color: #333;
  background-color: #f4f4f4;
}

.review-header {
  flex-shrink: 0;
  background-color: #fff;
}

.review-nav {
  height: 88rpx;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 24rpx;
  border-bottom: 1rpx solid #eee;
}

.back-button {
  width: 64rpx;
  height: 64rpx;
  line-height: 56rpx;
  font-size: 56rpx;
  color: #333;
  font-weight: 300;
}

.review-title-wrap {
  position: absolute;
  left: 50%;
  display: flex;
  align-items: center;
  transform: translateX(-50%);
}

.review-title {
  font-size: 32rpx;
  color: #333;
  font-weight: 600;
}

.publish-button {
  width: 108rpx;
  height: 60rpx;
  margin: 0;
  border-radius: 60rpx;
  line-height: 60rpx;
  font-size: 26rpx;
  color: #fff;
  background-color: #27ba9b;
  box-shadow: 0 8rpx 18rpx rgba(39, 186, 155, 0.2);

  &::after {
    border: 0;
  }

  &.disabled {
    color: #fff;
    background-color: #b8ded5;
    box-shadow: none;
  }
}

.review-body {
  flex: 1;
  min-height: 0;
  padding-bottom: 36rpx;
  box-sizing: border-box;
  background-color: #f4f4f4;
}

.editor-card,
.rating-card,
.anonymous-card {
  margin: 20rpx;
  border-radius: 10rpx;
  background-color: #fff;
  box-shadow: 0 8rpx 24rpx rgba(15, 23, 42, 0.03);
}

.editor-card {
  padding: 26rpx 24rpx 24rpx;
}

.editor-head,
.rating-card-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.editor-title,
.rating-card-title {
  position: relative;
  padding-left: 14rpx;
  font-size: 29rpx;
  line-height: 1;
  color: #333;
  font-weight: 600;

  &::before {
    content: '';
    position: absolute;
    left: 0;
    top: 50%;
    width: 5rpx;
    height: 24rpx;
    border-radius: 5rpx;
    background-color: #27ba9b;
    transform: translateY(-50%);
  }
}

.word-count {
  font-size: 24rpx;
  color: #999;
}

.review-textarea {
  width: 100%;
  height: 260rpx;
  margin-top: 24rpx;
  padding: 22rpx;
  border-radius: 10rpx;
  box-sizing: border-box;
  color: #333;
  font-size: 28rpx;
  line-height: 1.6;
  background-color: #f7f7f8;
}

.review-placeholder {
  color: #b8bcc5;
}

.image-section {
  margin-top: 24rpx;
  padding-top: 22rpx;
  border-top: 1rpx solid #f0f0f0;
}

.image-list {
  display: flex;
  flex-wrap: wrap;
  gap: 14rpx;
  margin-top: 18rpx;
}

.image-item,
.image-add {
  width: 148rpx;
  height: 148rpx;
  border-radius: 10rpx;
  box-sizing: border-box;
}

.image-item {
  position: relative;
  overflow: hidden;
  background-color: #f7f7f8;
}

.review-image {
  width: 100%;
  height: 100%;
}

.image-remove {
  position: absolute;
  right: 8rpx;
  top: 8rpx;
  width: 34rpx;
  height: 34rpx;
  border-radius: 50%;
  text-align: center;
  line-height: 30rpx;
  font-size: 30rpx;
  color: #fff;
  background-color: rgba(0, 0, 0, 0.45);
}

.image-add {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  border: 1rpx dashed rgba(39, 186, 155, 0.45);
  color: #27ba9b;
  font-size: 23rpx;
  background-color: #f2fffb;
}

.icon-camera-plus {
  margin-bottom: 10rpx;
  font-size: 42rpx;
  line-height: 1;
}

.rating-card {
  padding: 24rpx;
}

.rating-card-head {
  margin-bottom: 8rpx;
}

.rating-card-tip {
  font-size: 24rpx;
  color: #999;
}

.rating-row {
  min-height: 78rpx;
  display: flex;
  align-items: center;
  border-bottom: 1rpx solid #f5f5f5;

  &:last-child {
    border-bottom: 0;
  }
}

.rating-name {
  width: 150rpx;
  font-size: 27rpx;
  color: #333;
}

.rating-text {
  width: 108rpx;
  font-size: 25rpx;
  color: #898b94;
}

.rating-stars {
  flex: 1;
  display: flex;
  justify-content: flex-end;
}

.rating-star {
  margin-left: 18rpx;
  font-size: 40rpx;
  line-height: 1;
  color: #d7dbe2;

  &.active {
    color: #ffa868;
  }
}

.anonymous-card {
  display: flex;
  align-items: center;
  padding: 24rpx;
  margin-bottom: calc(24rpx + env(safe-area-inset-bottom));
}

.checkbox {
  width: 38rpx;
  height: 38rpx;
  margin-right: 18rpx;
  border: 2rpx solid #c7cbd2;
  border-radius: 50%;
  text-align: center;
  line-height: 34rpx;
  color: #fff;
  font-size: 22rpx;
  box-sizing: border-box;

  &.checked {
    border-color: #27ba9b;
    background-color: #27ba9b;
  }
}

.anonymous-title {
  font-size: 27rpx;
  color: #333;
  font-weight: 600;
}

.anonymous-desc {
  margin-top: 8rpx;
  font-size: 23rpx;
  color: #999;
}
</style>
