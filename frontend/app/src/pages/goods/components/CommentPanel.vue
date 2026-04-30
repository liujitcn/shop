<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { defCommentService } from '@/api/app/comment'
import type { CommentFilterItem, CommentItem } from '@/rpc/app/v1/comment'
import { formatSrc } from '@/utils'
import { goodsCommentListUrl } from '@/utils/navigation'
import avatarImage from '@/static/images/avatar.png'
import { useUserStore } from '@/stores'

const props = defineProps<{
  goods_id: number
  goods_picture?: string
  sku_code?: string
}>()

const ANONYMOUS_USER_NAME = '匿名用户'
const userStore = useUserStore()
const isLoading = ref(false)
const totalCount = ref(0)
const recentDays = ref(90)
const recentGoodRateText = ref('0%')
const aiSummaryContent = ref('')
const commentTags = ref<CommentFilterItem[]>([])
const previewList = ref<CommentItem[]>([])

const hasCommentStats = computed(() => totalCount.value > 0)

// 评价数量展示文案由前端根据数值转换，后端只返回原始数量。
const totalCountText = computed(() => {
  if (totalCount.value >= 10000) {
    return `${Math.floor(totalCount.value / 10000)}万+`
  }
  return `${totalCount.value}`
})

const commentList = computed(() => {
  return previewList.value.map((item) => ({
    ...item,
    imageList: (item.img || []).map((image) => formatSrc(image)),
    previewImage: formatSrc(item.img?.[0] || ''),
  }))
})

/** 重置评价摘要，未登录或请求失败时保持商品详情页可正常浏览。 */
const resetOverview = () => {
  totalCount.value = 0
  recentDays.value = 90
  recentGoodRateText.value = '0%'
  aiSummaryContent.value = ''
  commentTags.value = []
  previewList.value = []
}

// 拉取商品评论摘要，用于商品详情页首屏展示。
const loadOverview = async () => {
  if (!props.goods_id || props.goods_id <= 0) {
    return
  }
  if (!userStore.userInfo) {
    // 当前评价摘要接口需要登录态，游客访问商品详情时不发起请求，避免触发重新登录弹窗。
    resetOverview()
    return
  }

  isLoading.value = true
  try {
    const res = await defCommentService.GoodsCommentOverview({
      goods_id: props.goods_id,
      preview_limit: 2,
    })
    totalCount.value = res.total_count || 0
    recentDays.value = res.recent_days || 90
    recentGoodRateText.value = `${res.recent_good_rate || 0}%`
    aiSummaryContent.value = res.ai_summary?.content?.[0]?.content || ''
    commentTags.value = res.comment_filters || []
    previewList.value = res.preview_comments || []
  } catch (_error) {
    resetOverview()
  } finally {
    isLoading.value = false
  }
}

watch(
  [() => props.goods_id, () => userStore.userInfo],
  () => {
    void loadOverview()
  },
  { immediate: true },
)

const getCommentUserName = (item: CommentItem) => {
  return item.user?.user_name || ANONYMOUS_USER_NAME
}

const getCommentAvatar = (item: CommentItem) => {
  if (!item.user?.avatar) {
    return avatarImage
  }
  return formatSrc(item.user.avatar)
}

const getCommentRole = (item: CommentItem) => {
  return item.user?.user_tag_text || ''
}

const onPreviewImage = (images: string[], index: number) => {
  if (!images.length) {
    return
  }
  uni.previewImage({
    current: images[index],
    urls: images,
  })
}

const onOpenCommentPage = () => {
  void uni.navigateTo({
    url: goodsCommentListUrl({
      goods_id: props.goods_id,
      goods_picture: props.goods_picture,
      sku_code: props.sku_code,
    }),
  })
}
</script>

<template>
  <view class="comment-panel panel">
    <view class="comment-header" @tap="hasCommentStats ? onOpenCommentPage() : undefined">
      <view class="comment-title"
        >买家评价 <text>{{ totalCountText }}</text></view
      >
      <view v-if="hasCommentStats" class="comment-rate">
        近 {{ recentDays }} 天好评率 <text>{{ recentGoodRateText }}</text> ›
      </view>
    </view>

    <view v-if="hasCommentStats && aiSummaryContent" class="ai-review" @tap="onOpenCommentPage">
      <view class="ai-mascot">
        <text class="ai-mascot-eye">›•</text>
      </view>
      <view class="ai-bubble">
        <text class="summary-label">✦ AI 全网评</text>
        <text class="summary-content">{{ aiSummaryContent }}</text>
      </view>
    </view>

    <view v-if="hasCommentStats && commentTags.length" class="comment-tags">
      <view
        v-for="item in commentTags"
        :key="`${item.filter_type}-${item.tag_id}`"
        class="comment-tag"
      >
        <view class="comment-tag-name">{{ item.label }}</view>
        <view class="comment-tag-count">{{ item.value }}</view>
      </view>
    </view>

    <view class="comment-list">
      <view v-if="isLoading" class="comment-tag-count">评价加载中...</view>
      <XtxEmptyState
        v-else-if="!commentList.length"
        image="/static/images/empty_comment.png"
        text="暂无评价"
        image-width="180rpx"
        image-height="150rpx"
        min-height="260rpx"
        padding="28rpx 0 30rpx"
      />
      <view v-for="item in commentList" :key="item.id" class="comment-item">
        <view class="comment-user">
          <image class="comment-avatar" :src="getCommentAvatar(item)" mode="aspectFill" />
          <view class="comment-user-main">
            <view class="comment-user-line">
              <text class="comment-user-name">{{ getCommentUserName(item) }}</text>
              <text v-if="getCommentRole(item)" class="comment-role">{{
                getCommentRole(item)
              }}</text>
            </view>
          </view>
          <text class="comment-date">{{ item.date_text }}</text>
        </view>

        <view class="comment-body" :class="{ 'comment-body--media': item.previewImage }">
          <view class="comment-content">
            <text
              v-for="(segment, index) in item.content_segments"
              :key="index"
              :class="{ 'comment-keyword': segment.highlight }"
            >
              {{ segment.text }}
            </text>
          </view>
          <view
            v-if="item.previewImage"
            class="comment-image-wrap"
            @tap="onPreviewImage(item.imageList, 0)"
          >
            <image class="comment-image" :src="item.previewImage" mode="aspectFill" />
            <view v-if="item.image_count" class="comment-image-count">{{ item.image_count }}</view>
          </view>
        </view>
      </view>
    </view>
  </view>
</template>

<style lang="scss">
.comment-panel {
  padding: 0 20rpx 28rpx;
  background-color: #fff;
}

.comment-header {
  height: 90rpx;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 30rpx 0;
  box-sizing: border-box;
}

.comment-title {
  flex-shrink: 0;
  padding-left: 10rpx;
  border-left: 4rpx solid #27ba9b;
  line-height: 1;
  font-size: 28rpx;
  color: #333;
  font-weight: 600;

  text {
    margin-left: 8rpx;
  }
}

.comment-rate {
  font-size: 24rpx;
  color: #898b94;

  text {
    margin-left: 6rpx;
    color: #333;
    font-weight: 600;
  }
}

.ai-review {
  display: flex;
  align-items: center;
  margin-bottom: 18rpx;
}

.ai-mascot {
  position: relative;
  width: 72rpx;
  height: 72rpx;
  margin-right: 18rpx;
  flex-shrink: 0;
  border-radius: 36rpx 40rpx 36rpx 40rpx;
  background:
    radial-gradient(circle at 20% 75%, rgba(255, 142, 178, 0.45) 0, rgba(255, 142, 178, 0) 16rpx),
    radial-gradient(circle at 82% 70%, rgba(91, 214, 255, 0.4) 0, rgba(91, 214, 255, 0) 16rpx),
    linear-gradient(135deg, #dff7f2 0%, #ffffff 48%, #e9f8f5 100%);

  &::before,
  &::after {
    content: '✦';
    position: absolute;
    color: #5ed1bd;
  }

  &::before {
    top: -10rpx;
    right: -6rpx;
    font-size: 22rpx;
  }

  &::after {
    left: -6rpx;
    bottom: 8rpx;
    font-size: 16rpx;
    opacity: 0.65;
  }
}

.ai-mascot-eye {
  position: absolute;
  left: 19rpx;
  top: 22rpx;
  font-size: 28rpx;
  color: #111;
  font-weight: 700;
  letter-spacing: 2rpx;
}

.ai-bubble {
  position: relative;
  flex: 1;
  padding: 18rpx 20rpx;
  border: 1rpx solid #ebeef2;
  border-radius: 14rpx;
  background-color: #fff;

  &::before {
    content: '';
    position: absolute;
    left: -12rpx;
    top: 34rpx;
    width: 22rpx;
    height: 22rpx;
    border-left: 1rpx solid #ebeef2;
    border-bottom: 1rpx solid #ebeef2;
    background-color: #fff;
    transform: rotate(45deg);
  }
}

.summary-label {
  margin-right: 8rpx;
  font-size: 24rpx;
  line-height: 1.6;
  color: #27ba9b;
  font-weight: 600;
}

.summary-content {
  font-size: 26rpx;
  line-height: 1.6;
  color: #333;
}

.summary-highlight {
  font-weight: 600;
}

.comment-tags {
  display: flex;
  justify-content: space-between;
  margin: 18rpx 0;
}

.comment-tag {
  width: calc((100% - 30rpx) / 4);
  padding: 14rpx 4rpx;
  border-radius: 8rpx;
  text-align: center;
  background-color: #f7f7f8;
}

.comment-tag-name {
  font-size: 24rpx;
  line-height: 1.2;
  color: #333;
}

.comment-tag-count {
  margin-top: 8rpx;
  font-size: 20rpx;
  line-height: 1.2;
  color: #898b94;
}

.comment-list {
  border-top: 1rpx solid #f1f1f1;
}

.comment-item {
  padding: 20rpx 0;
  border-bottom: 1rpx solid #f1f1f1;

  &:last-child {
    border-bottom: 0;
  }
}

.comment-user {
  display: flex;
  align-items: center;
}

.comment-avatar {
  width: 50rpx;
  height: 50rpx;
  margin-right: 12rpx;
  flex-shrink: 0;
  border-radius: 50%;
  background-color: #f2f3f5;
}

.comment-user-main {
  flex: 1;
  min-width: 0;
}

.comment-user-line {
  display: flex;
  align-items: center;
}

.comment-user-name {
  margin-right: 8rpx;
  font-size: 24rpx;
  color: #333;
}

.comment-role {
  padding: 2rpx 8rpx;
  border-radius: 4rpx;
  font-size: 20rpx;
  color: #898b94;
  background-color: #f7f7f8;
}

.comment-date {
  margin-left: 16rpx;
  font-size: 22rpx;
  color: #999;
}

.comment-body {
  margin-top: 14rpx;
}

.comment-body--media {
  display: flex;
  align-items: flex-start;
}

.comment-content {
  flex: 1;
  min-width: 0;
  font-size: 26rpx;
  line-height: 1.6;
  color: #333;
}

.comment-keyword {
  font-weight: 600;
  text-decoration-line: underline;
  text-decoration-style: wavy;
  text-decoration-color: rgba(39, 186, 155, 0.36);
}

.comment-image-wrap {
  position: relative;
  width: 132rpx;
  height: 132rpx;
  margin-left: 18rpx;
  flex-shrink: 0;
  overflow: hidden;
  border-radius: 10rpx;
  background-color: #f7f7f8;
}

.comment-image {
  width: 100%;
  height: 100%;
}

.comment-image-count {
  position: absolute;
  right: 0;
  bottom: 0;
  min-width: 34rpx;
  height: 34rpx;
  padding: 0 6rpx;
  border-radius: 10rpx 0 10rpx 0;
  text-align: center;
  line-height: 34rpx;
  color: #fff;
  font-size: 20rpx;
  background-color: rgba(0, 0, 0, 0.6);
}

.comment-footer {
  margin-top: 12rpx;
  text-align: right;
  font-size: 22rpx;
  color: #999;
}
</style>
