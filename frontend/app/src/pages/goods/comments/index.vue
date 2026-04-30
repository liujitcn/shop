<script setup lang="ts">
import { computed, ref } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import { defCommentService } from '@/api/app/comment'
import type {
  CommentAi,
  CommentFilterItem,
  CommentItem,
  CommentTextSegment,
} from '@/rpc/app/v1/comment'
import {
  CommentFilterType,
  CommentReactionTargetType,
  CommentReactionType,
  CommentSortType,
} from '@/rpc/common/v1/enum'
import { formatSrc } from '@/utils'
import { navigateToLogin } from '@/utils/navigation'
import { useUserStore } from '@/stores'
import defaultAvatar from '@/static/images/avatar.png'
import DiscussionPopup from './components/DiscussionPopup.vue'

const props = defineProps<{
  goods_id: string
  goods_picture?: string
  sku_code?: string
}>()

type ReviewFilter = CommentFilterItem & {
  key: string
}

type SummaryItem = {
  label: string
  content: string
}

const { safeAreaInsets } = uni.getSystemInfoSync()
const userStore = useUserStore()
const goodsId = Number(props.goods_id)
const pageSize = 10
const activeFilter = ref('')
const activeSort = ref(CommentSortType.COMMENT_SORT_DEFAULT)
const currentGoodsOnly = ref(false)
const activeReviewId = ref<number>()
const commentPopupVisible = ref(false)
const filterExpanded = ref(false)
const filterPinned = ref(false)
const expandedReviewMap = ref<Record<number, boolean>>({})
const collapsedFilterCount = 5
const collapsedPinnedThreshold = 72
const expandedPinnedThreshold = 220
const collapsedReviewTextLength = 72
const maxPreviewImageCount = 6
const ANONYMOUS_USER_NAME = '匿名用户'
const isLoading = ref(false)
const isLoadingMore = ref(false)
const filterList = ref<CommentFilterItem[]>([])
const aiSummary = ref<CommentAi>()
const buyerReviews = ref<CommentItem[]>([])
const currentPageNum = ref(1)
const hasMore = ref(false)

const filters = computed<ReviewFilter[]>(() => {
  return (filterList.value || []).map((item) => ({
    ...item,
    key: `${item.filter_type}-${item.tag_id}`,
  }))
})

const selectedFilter = computed(() => {
  return (
    filters.value.find((item) => item.key === activeFilter.value) || {
      key: '',
      filter_type: CommentFilterType.COMMENT_FILTER_ALL,
      tag_id: 0,
      label: '全部',
      value: '',
    }
  )
})

const visibleGridFilters = computed(() => {
  return filterExpanded.value ? filters.value : filters.value.slice(0, collapsedFilterCount)
})

const hasHiddenFilters = computed(() => {
  return filters.value.length > collapsedFilterCount
})

const aiSummaryList = computed<SummaryItem[]>(() => {
  return (aiSummary.value?.content || []).filter((item) => item.label || item.content)
})

const hasActiveReviewCondition = computed(() => {
  return (
    currentGoodsOnly.value ||
    selectedFilter.value.filter_type !== CommentFilterType.COMMENT_FILTER_ALL ||
    selectedFilter.value.tag_id > 0
  )
})

const reviewEmptyText = computed(() => {
  return hasActiveReviewCondition.value ? '暂无符合条件的评价' : '暂无评价'
})

// 评论页允许匿名浏览，但互动和发布讨论需要登录。
const isAiReactionActive = (reaction_type: CommentReactionType) => {
  return aiSummary.value?.reaction_type === reaction_type
}

const isReviewReactionActive = (item: CommentItem, reaction_type: CommentReactionType) => {
  return item.reaction_type === reaction_type
}

const ensureLogin = () => {
  if (userStore.userInfo) {
    return true
  }
  navigateToLogin()
  return false
}

// 加载评论分页数据，并同步更新筛选项与 AI 摘要。
const loadCommentData = async (reset: boolean) => {
  if (!Number.isFinite(goodsId) || goodsId <= 0) {
    return
  }
  if (reset) {
    isLoading.value = true
  } else {
    if (isLoadingMore.value || !hasMore.value) {
      return
    }
    isLoadingMore.value = true
  }

  const nextPageNum = reset ? 1 : currentPageNum.value + 1
  try {
    const res = await defCommentService.PageGoodsComment({
      goods_id: goodsId,
      sku_code: props.sku_code || '',
      filter_type: selectedFilter.value.filter_type,
      tag_id: selectedFilter.value.tag_id,
      current_goods_only: currentGoodsOnly.value,
      sort_type: activeSort.value,
      page_num: nextPageNum,
      page_size: pageSize,
    })

    filterList.value = res.comment_filters || []
    if (filterList.value.length) {
      const hasCurrentFilter = filterList.value.some(
        (item) => `${item.filter_type}-${item.tag_id}` === activeFilter.value,
      )
      if (!hasCurrentFilter) {
        const firstFilter = filterList.value[0]
        activeFilter.value = `${firstFilter.filter_type}-${firstFilter.tag_id}`
      }
    }

    aiSummary.value = res.ai_summary
    buyerReviews.value = reset
      ? res.comments || []
      : [...buyerReviews.value, ...(res.comments || [])]
    currentPageNum.value = res.page_num || nextPageNum
    hasMore.value = Boolean(res.has_more)
  } catch (_error) {
    if (reset) {
      filterList.value = []
      aiSummary.value = undefined
      buyerReviews.value = []
      currentPageNum.value = 1
      hasMore.value = false
    }
  } finally {
    isLoading.value = false
    isLoadingMore.value = false
  }
}

const getReviewUserName = (item: CommentItem) => {
  return item.user?.user_name || ANONYMOUS_USER_NAME
}

const getReviewAvatar = (item: CommentItem) => {
  if (!item.user?.avatar) {
    return defaultAvatar
  }
  return formatSrc(item.user.avatar)
}

const getReviewRole = (item: CommentItem) => {
  return item.user?.user_tag_text || ''
}

const getReviewTextLength = (content: CommentTextSegment[]) => {
  return content.reduce((total, segment) => total + segment.text.length, 0)
}

const shouldCollapseReview = (item: CommentItem) => {
  return getReviewTextLength(item.content_segments || []) > collapsedReviewTextLength
}

const isReviewExpanded = (reviewId: number) => {
  return expandedReviewMap.value[reviewId] === true
}

const getVisibleReviewContent = (item: CommentItem) => {
  if (!shouldCollapseReview(item) || isReviewExpanded(item.id)) {
    return item.content_segments || []
  }

  let remainingLength = collapsedReviewTextLength
  const content: CommentTextSegment[] = []
  for (const segment of item.content_segments || []) {
    if (remainingLength <= 0) {
      break
    }
    if (segment.text.length <= remainingLength) {
      content.push(segment)
      remainingLength -= segment.text.length
      continue
    }
    content.push({
      text: `${segment.text.slice(0, remainingLength)}...`,
      highlight: segment.highlight,
    })
    break
  }
  return content
}

const getReviewImageList = (item: CommentItem) => {
  return (item.img || []).map((image) => formatSrc(image))
}

const getVisibleReviewImages = (images: string[]) => {
  return images.slice(0, maxPreviewImageCount)
}

const getReviewImageLayoutClass = (image_count: number) => {
  if (image_count === 1) {
    return 'review-images--one'
  }
  if (image_count === 2) {
    return 'review-images--two'
  }
  if (image_count === 3) {
    return 'review-images--three'
  }
  if (image_count === 4) {
    return 'review-images--four'
  }
  return 'review-images--multi'
}

const isReviewImageMoreMaskVisible = (images: string[], imageIndex: number) => {
  return images.length > maxPreviewImageCount && imageIndex === maxPreviewImageCount - 1
}

const onSelectFilter = (key: string) => {
  activeFilter.value = key
  void loadCommentData(true)
}

const onSelectSortLatest = () => {
  activeSort.value =
    activeSort.value === CommentSortType.COMMENT_SORT_LATEST
      ? CommentSortType.COMMENT_SORT_DEFAULT
      : CommentSortType.COMMENT_SORT_LATEST
  void loadCommentData(true)
}

const onToggleCurrentGoodsOnly = () => {
  if (!props.sku_code) {
    void uni.showToast({
      title: '当前规格暂不可筛选',
      icon: 'none',
    })
    return
  }
  currentGoodsOnly.value = !currentGoodsOnly.value
  void loadCommentData(true)
}

const onToggleFilterExpanded = () => {
  filterExpanded.value = !filterExpanded.value
}

const onCommentsScroll = (event: { detail: { scrollTop: number } }) => {
  const pinnedThreshold = filterExpanded.value ? expandedPinnedThreshold : collapsedPinnedThreshold
  filterPinned.value = event.detail.scrollTop > pinnedThreshold
}

const onCommentsToLower = () => {
  void loadCommentData(false)
}

const onToggleReviewContent = (reviewId: number) => {
  expandedReviewMap.value = {
    ...expandedReviewMap.value,
    [reviewId]: !isReviewExpanded(reviewId),
  }
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

const onNavigateBack = () => {
  const pages = getCurrentPages()
  if (pages.length > 1) {
    uni.navigateBack()
    return
  }
  uni.switchTab({ url: '/pages/index/index' })
}

const onOpenCommentPopup = (reviewId: number) => {
  activeReviewId.value = reviewId
  commentPopupVisible.value = true
}

const onCloseCommentPopup = () => {
  commentPopupVisible.value = false
}

// 发布讨论后同步回写评论项上的讨论总数，避免关闭弹层前列表数据滞后。
const onDiscussionCountChange = (payload: { comment_id: number; discussion_count: number }) => {
  buyerReviews.value = buyerReviews.value.map((item) => {
    if (item.id !== payload.comment_id) {
      return item
    }
    return {
      ...item,
      discussion_count: payload.discussion_count,
    }
  })
}

// 保存 AI 总结赞踩状态，并用后端实时统计数量刷新当前卡片。
const onSaveAiReaction = async (reaction_type: CommentReactionType) => {
  if (!ensureLogin() || !aiSummary.value?.id) {
    return
  }

  const active = !isAiReactionActive(reaction_type)
  const res = await defCommentService.SaveCommentReaction({
    target_type: CommentReactionTargetType.AI,
    target_id: aiSummary.value.id,
    reaction_type,
    active,
  })
  aiSummary.value = {
    ...aiSummary.value,
    like_count: res.like_count,
    dislike_count: res.dislike_count,
    reaction_type: res.reaction_type,
  }
}

// 保存评价赞踩状态，并用后端实时统计数量刷新当前评价。
const onSaveReviewReaction = async (item: CommentItem, reaction_type: CommentReactionType) => {
  if (!ensureLogin()) {
    return
  }

  const active = !isReviewReactionActive(item, reaction_type)
  const res = await defCommentService.SaveCommentReaction({
    target_type: CommentReactionTargetType.COMMENT,
    target_id: item.id,
    reaction_type,
    active,
  })
  buyerReviews.value = buyerReviews.value.map((review) => {
    if (review.id !== item.id) {
      return review
    }
    return {
      ...review,
      like_count: res.like_count,
      dislike_count: res.dislike_count,
      reaction_type: res.reaction_type,
    }
  })
}

const onStaticToast = (title: string) => {
  void uni.showToast({
    title,
    icon: 'none',
  })
}

onLoad(() => {
  void loadCommentData(true)
})
</script>

<template>
  <view class="comments-page">
    <view class="comments-header" :style="{ paddingTop: `${safeAreaInsets?.top || 0}px` }">
      <view class="comments-nav">
        <view class="back-button" @tap="onNavigateBack">‹</view>
        <view class="comments-title">评价</view>
      </view>
    </view>

    <view
      v-show="filterPinned"
      class="filter-bar filter-bar--pinned"
      :style="{ top: `calc(${safeAreaInsets?.top || 0}px + 88rpx)` }"
    >
      <scroll-view scroll-x class="filter-scroll" :show-scrollbar="false">
        <view class="filter-list filter-list--scroll">
          <view
            v-for="item in filters"
            :key="item.key"
            class="filter-chip"
            :class="{ active: activeFilter === item.key }"
            @tap="onSelectFilter(item.key)"
          >
            <text>{{ item.label }}</text>
            <text v-if="item.value" class="filter-value">{{ item.value }}</text>
          </view>
        </view>
      </scroll-view>
    </view>

    <scroll-view
      scroll-y
      class="comments-scroll"
      @scroll="onCommentsScroll"
      @scrolltolower="onCommentsToLower"
    >
      <view class="comments-overview-card">
        <view class="filter-grid-card">
          <view class="filter-grid">
            <view
              v-for="item in visibleGridFilters"
              :key="item.key"
              class="filter-chip filter-chip--grid"
              :class="{ active: activeFilter === item.key }"
              @tap="onSelectFilter(item.key)"
            >
              <text>{{ item.label }}</text>
              <text v-if="item.value" class="filter-value">{{ item.value }}</text>
            </view>
            <view
              v-if="hasHiddenFilters"
              class="filter-toggle"
              :class="{ expanded: filterExpanded }"
              @tap="onToggleFilterExpanded"
            />
          </view>
        </view>

        <view class="review-sort">
          <view class="review-sort-trust">真实、有用的买家评价</view>
          <view class="review-sort-actions">
            <view
              class="review-sort-option"
              :class="{ active: activeSort === CommentSortType.COMMENT_SORT_LATEST }"
              @tap="onSelectSortLatest"
            >
              最新
            </view>
            <view class="review-sort-divider" />
            <view
              class="review-sort-option review-sort-option--check"
              :class="{ active: currentGoodsOnly }"
              @tap="onToggleCurrentGoodsOnly"
            >
              <view class="review-sort-check" />
              当前商品
            </view>
          </view>
        </view>

        <view v-if="aiSummaryList.length" class="summary-card">
          <view v-for="item in aiSummaryList" :key="item.label" class="summary-line">
            <text>{{ item.label }}：</text>{{ item.content }}
          </view>
          <view class="summary-actions">
            <view
              class="summary-action action-item"
              :class="{ active: isAiReactionActive(CommentReactionType.LIKE) }"
              @tap="onSaveAiReaction(CommentReactionType.LIKE)"
            >
              <view class="action-icon action-icon--like" />
              <text>赞 {{ aiSummary?.like_count || 0 }}</text>
            </view>
            <view
              class="summary-action action-item"
              :class="{ active: isAiReactionActive(CommentReactionType.DISLIKE) }"
              @tap="onSaveAiReaction(CommentReactionType.DISLIKE)"
            >
              <view class="action-icon action-icon--dislike" />
              <text>点踩 {{ aiSummary?.dislike_count || 0 }}</text>
            </view>
          </view>
        </view>
      </view>

      <view class="review-list">
        <view v-if="isLoading" class="review-item">
          <view class="review-sort-trust">评价加载中...</view>
        </view>
        <XtxEmptyState
          v-else-if="!buyerReviews.length"
          image="/static/images/empty_comment.png"
          :text="reviewEmptyText"
          image-width="180rpx"
          image-height="150rpx"
          min-height="340rpx"
          padding="72rpx 0 88rpx"
        />
        <view v-for="item in buyerReviews" :key="item.id" class="review-item">
          <view class="review-user">
            <image class="review-avatar" :src="getReviewAvatar(item)" mode="aspectFill" />
            <view class="review-user-main">
              <view class="review-user-line">
                <text class="review-user-name">{{ getReviewUserName(item) }}</text>
                <text v-if="getReviewRole(item)" class="review-role">{{
                  getReviewRole(item)
                }}</text>
              </view>
            </view>
            <text class="review-date">{{ item.date_text }}</text>
          </view>

          <view class="review-content">
            <text
              v-for="(segment, index) in getVisibleReviewContent(item)"
              :key="index"
              :class="{ 'review-keyword': segment.highlight }"
            >
              {{ segment.text }}
            </text>
            <text
              v-if="shouldCollapseReview(item)"
              class="review-expand"
              :class="{ expanded: isReviewExpanded(item.id) }"
              @tap.stop="onToggleReviewContent(item.id)"
            >
              {{ isReviewExpanded(item.id) ? '收起' : '展开' }}
            </text>
          </view>

          <view
            v-if="getReviewImageList(item).length"
            class="review-images"
            :class="getReviewImageLayoutClass(getReviewImageList(item).length)"
          >
            <view
              v-for="(image, imageIndex) in getVisibleReviewImages(getReviewImageList(item))"
              :key="`${item.id}-${imageIndex}`"
              class="review-image-wrap"
              @tap="onPreviewImage(getReviewImageList(item), imageIndex)"
            >
              <image class="review-image" :src="image" mode="aspectFill" />
              <view
                v-if="isReviewImageMoreMaskVisible(getReviewImageList(item), imageIndex)"
                class="review-image-more"
              >
                +{{ getReviewImageList(item).length - maxPreviewImageCount }}
              </view>
            </view>
          </view>

          <view class="review-actions">
            <view
              class="review-action action-item"
              :class="{ active: isReviewReactionActive(item, CommentReactionType.LIKE) }"
              @tap="onSaveReviewReaction(item, CommentReactionType.LIKE)"
            >
              <view class="action-icon action-icon--like" />
              <text>赞 {{ item.like_count || 0 }}</text>
            </view>
            <view
              class="review-action action-item"
              :class="{ active: isReviewReactionActive(item, CommentReactionType.DISLIKE) }"
              @tap="onSaveReviewReaction(item, CommentReactionType.DISLIKE)"
            >
              <view class="action-icon action-icon--dislike" />
              <text>点踩 {{ item.dislike_count || 0 }}</text>
            </view>
            <view class="review-action action-item" @tap="onOpenCommentPopup(item.id)">
              <view class="action-icon action-icon--comment" />
              <text>评论 {{ item.discussion_count || 0 }}</text>
            </view>
          </view>
        </view>
        <view v-if="isLoadingMore" class="review-item">
          <view class="review-sort-trust">更多评价加载中...</view>
        </view>
      </view>
    </scroll-view>

    <view class="toolbar" :style="{ paddingBottom: `${safeAreaInsets?.bottom || 0}px` }">
      <view class="icons">
        <button class="icons-button" @tap="onStaticToast('请返回商品详情收藏')">
          <text class="icon-heart" />收藏
        </button>
        <!-- #ifdef MP-WEIXIN -->
        <button class="icons-button" open-type="contact"><text class="icon-handset" />客服</button>
        <!-- #endif -->
        <navigator class="icons-button" url="/pages/cart/cart2" open-type="navigate">
          <text class="icon-cart" />购物车
        </navigator>
      </view>
      <view class="buttons">
        <view class="addcart" @tap="onStaticToast('请返回商品详情选择规格')"> 加入购物车 </view>
        <view class="payment" @tap="onStaticToast('请返回商品详情选择规格')"> 立即购买 </view>
      </view>
    </view>

    <DiscussionPopup
      :visible="commentPopupVisible"
      :review-id="activeReviewId"
      @close="onCloseCommentPopup"
      @count-change="onDiscussionCountChange"
    />
  </view>
</template>

<style lang="scss">
page {
  height: 100%;
  overflow: hidden;
  background-color: #f4f4f4;
}

.comments-page {
  height: 100%;
  display: flex;
  flex-direction: column;
  color: #333;
  background-color: #f4f4f4;
}

.comments-header {
  position: relative;
  z-index: 13;
  flex-shrink: 0;
  background-color: #fff;
}

.comments-nav {
  height: 88rpx;
  display: flex;
  align-items: center;
  padding: 0 24rpx;
  box-sizing: border-box;
}

.back-button {
  width: 56rpx;
  margin-right: 16rpx;
  line-height: 1;
  font-size: 56rpx;
  color: #333;
  font-weight: 300;
}

.comments-title {
  flex: 1;
  font-size: 34rpx;
  color: #333;
  font-weight: 600;
}

.filter-bar {
  background-color: #fff;
  border-bottom: 1rpx solid #f0f0f0;
}

.filter-bar--pinned {
  position: fixed;
  left: 0;
  right: 0;
  z-index: 12;
  box-shadow: 0 8rpx 20rpx rgba(15, 23, 42, 0.04);
}

.filter-scroll {
  width: 100%;
  white-space: nowrap;
}

.filter-list--scroll {
  min-width: 100%;
  display: inline-flex;
  align-items: center;
  padding: 12rpx 20rpx 18rpx;
  box-sizing: border-box;
}

.comments-overview-card {
  margin: 14rpx 20rpx 20rpx;
  overflow: hidden;
  border-radius: 14rpx;
  background-color: #fff;
}

.filter-grid-card {
  padding: 16rpx 16rpx 2rpx;
  background-color: transparent;
}

.filter-grid {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
}

.filter-chip {
  height: 64rpx;
  display: inline-flex;
  align-items: center;
  margin-right: 16rpx;
  padding: 0 22rpx;
  flex-shrink: 0;
  border-radius: 10rpx;
  font-size: 26rpx;
  color: #333;
  background-color: #f7f7f8;

  &.active {
    color: #168f78;
    background-color: #e9f8f5;
    font-weight: 600;
  }
}

.filter-chip--grid {
  margin-right: 14rpx;
  margin-bottom: 14rpx;
}

.filter-toggle {
  width: 64rpx;
  height: 64rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-left: auto;
  margin-bottom: 14rpx;
  flex-shrink: 0;
  color: #898b94;

  &::before {
    content: '\e6c0';
    font-family: 'erabbit' !important;
    font-size: 28rpx;
  }

  &.expanded::before {
    content: '\e6bf';
  }
}

.filter-value {
  margin-left: 8rpx;
  color: #898b94;
  font-weight: 400;
}

.comments-scroll {
  flex: 1;
  min-height: 0;
  background-color: #f4f4f4;
}

.review-sort {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin: 2rpx 20rpx 0;
  padding: 18rpx 0;
  border-top: 1rpx solid #f0f0f0;
  border-bottom: 1rpx solid #f0f0f0;
  background-color: transparent;
}

.review-sort-trust {
  position: relative;
  padding-left: 34rpx;
  font-size: 26rpx;
  color: #898b94;

  &::before {
    content: '✓';
    position: absolute;
    left: 0;
    top: 50%;
    width: 24rpx;
    height: 24rpx;
    border: 2rpx solid #c7cbd2;
    border-radius: 50%;
    line-height: 22rpx;
    text-align: center;
    font-size: 18rpx;
    color: #9ca3af;
    transform: translateY(-50%);
  }
}

.review-sort-actions {
  display: flex;
  align-items: center;
  flex-shrink: 0;
}

.review-sort-option {
  display: flex;
  align-items: center;
  font-size: 26rpx;
  color: #333;

  &.active {
    color: #111827;
    font-weight: 600;
  }
}

.review-sort-divider {
  width: 1rpx;
  height: 24rpx;
  margin: 0 18rpx;
  background-color: #e5e7eb;
}

.review-sort-option--check {
  color: #333;
}

.review-sort-check {
  width: 28rpx;
  height: 28rpx;
  margin-right: 8rpx;
  border: 3rpx solid #8f96a3;
  border-radius: 50%;
  box-sizing: border-box;
}

.review-sort-option--check.active {
  color: #168f78;

  .review-sort-check {
    position: relative;
    border-color: #27ba9b;

    &::after {
      content: '';
      position: absolute;
      left: 50%;
      top: 50%;
      width: 12rpx;
      height: 12rpx;
      border-radius: 50%;
      background-color: #27ba9b;
      transform: translate(-50%, -50%);
    }
  }
}

.summary-card {
  padding: 20rpx;
  background-color: transparent;
}

.summary-line {
  position: relative;
  margin-top: 14rpx;
  padding-left: 20rpx;
  font-size: 26rpx;
  line-height: 1.62;
  color: #4b5563;

  &::before {
    content: '';
    position: absolute;
    left: 0;
    top: 17rpx;
    width: 8rpx;
    height: 8rpx;
    border-radius: 50%;
    background-color: #27ba9b;
  }

  text {
    color: #333;
    font-weight: 600;
  }

  &:first-child {
    margin-top: 0;
  }
}

.summary-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  margin-top: 18rpx;
}

.summary-action {
  padding: 0 0 0 18rpx;
  font-size: 24rpx;
  color: #898b94;

  &.active {
    color: #27ba9b;
  }

  &:first-child {
    padding-left: 0;
  }

  & + .summary-action {
    margin-left: 18rpx;
    border-left: 1rpx solid #e5e7eb;
  }
}

.action-item {
  display: flex;
  align-items: center;
  line-height: 1;
}

.action-icon {
  width: 40rpx;
  height: 40rpx;
  margin-right: 8rpx;
  flex-shrink: 0;
  background-position: center;
  background-repeat: no-repeat;
  background-size: 100% 100%;
}

.action-icon--like {
  background-image: url("data:image/svg+xml,%3Csvg viewBox='0 0 40 40' xmlns='http://www.w3.org/2000/svg' fill='none' stroke='%238f96a3' stroke-width='2.8' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpath d='M13 18v16H8.5A3.5 3.5 0 0 1 5 30.5v-9A3.5 3.5 0 0 1 8.5 18H13Z'/%3E%3Cpath d='M13 18l6.2-11.5c.7-1.3 2.5-1.5 3.5-.5.8.8 1 2 .5 3L21 15h10.2c2.4 0 4.1 2.2 3.6 4.5l-1.8 9.2A6.5 6.5 0 0 1 26.6 34H13V18Z'/%3E%3C/svg%3E");
}

.action-icon--dislike {
  transform: rotate(180deg);
  background-image: url("data:image/svg+xml,%3Csvg viewBox='0 0 40 40' xmlns='http://www.w3.org/2000/svg' fill='none' stroke='%238f96a3' stroke-width='2.8' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpath d='M13 18v16H8.5A3.5 3.5 0 0 1 5 30.5v-9A3.5 3.5 0 0 1 8.5 18H13Z'/%3E%3Cpath d='M13 18l6.2-11.5c.7-1.3 2.5-1.5 3.5-.5.8.8 1 2 .5 3L21 15h10.2c2.4 0 4.1 2.2 3.6 4.5l-1.8 9.2A6.5 6.5 0 0 1 26.6 34H13V18Z'/%3E%3C/svg%3E");
}

.action-icon--comment {
  background-image: url("data:image/svg+xml,%3Csvg viewBox='0 0 40 40' xmlns='http://www.w3.org/2000/svg' fill='none' stroke='%238f96a3' stroke-width='2.8' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpath d='M8 8h24a4 4 0 0 1 4 4v13a4 4 0 0 1-4 4H18l-8 6v-6H8a4 4 0 0 1-4-4V12a4 4 0 0 1 4-4Z'/%3E%3Cpath d='M14 18h.01M26 18h.01M15.5 22.5c2.4 2.1 6.6 2.1 9 0'/%3E%3C/svg%3E");
}

.review-list {
  padding: 0 20rpx 160rpx;
}

.review-item {
  padding: 28rpx 0;
  border-bottom: 1rpx solid #f0f0f0;
  background-color: #fff;

  &:first-child {
    border-radius: 10rpx 10rpx 0 0;
    padding-top: 24rpx;
  }

  &:last-child {
    border-bottom: 0;
    border-radius: 0 0 10rpx 10rpx;
  }
}

.review-user,
.review-content,
.review-images,
.review-actions {
  margin-left: 24rpx;
  margin-right: 24rpx;
}

.review-user {
  display: flex;
  align-items: center;
}

.review-avatar {
  width: 64rpx;
  height: 64rpx;
  margin-right: 16rpx;
  flex-shrink: 0;
  border-radius: 50%;
  background-color: #f2f3f5;
}

.review-user-main {
  flex: 1;
  min-width: 0;
}

.review-user-line {
  display: flex;
  align-items: center;
}

.review-user-name {
  max-width: 260rpx;
  margin-right: 10rpx;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 28rpx;
  color: #333;
  font-weight: 600;
}

.review-role {
  padding: 2rpx 8rpx;
  border-radius: 4rpx;
  font-size: 20rpx;
  color: #898b94;
  background-color: #f7f7f8;
}

.review-date {
  margin-left: 16rpx;
  font-size: 22rpx;
  color: #999;
}

.review-content {
  margin-top: 18rpx;
  font-size: 29rpx;
  line-height: 1.72;
  color: #222;
}

.review-expand {
  margin-left: 8rpx;
  white-space: nowrap;
  color: #898b94;

  &::after {
    content: '\e6c0';
    margin-left: 4rpx;
    font-family: 'erabbit' !important;
    font-size: 22rpx;
  }

  &.expanded::after {
    content: '\e6bf';
  }
}

.review-keyword {
  font-weight: 600;
  text-decoration-line: underline;
  text-decoration-style: wavy;
  text-decoration-color: rgba(39, 186, 155, 0.38);
}

.review-images {
  display: grid;
  gap: 12rpx;
  margin-top: 18rpx;
}

.review-image-wrap {
  position: relative;
  overflow: hidden;
  border-radius: 10rpx;
  background-color: #f7f7f8;
}

.review-image {
  width: 100%;
  height: 100%;
}

.review-images--one {
  width: 430rpx;
  grid-template-columns: 430rpx;

  .review-image-wrap {
    height: 430rpx;
  }
}

.review-images--two {
  width: 488rpx;
  grid-template-columns: repeat(2, 1fr);

  .review-image-wrap {
    height: 238rpx;
  }
}

.review-images--three {
  grid-template-columns: minmax(0, 1fr) 236rpx;
  grid-template-rows: repeat(2, 172rpx);

  .review-image-wrap:first-child {
    grid-row: span 2;
  }
}

.review-images--four {
  width: 500rpx;
  grid-template-columns: repeat(2, 1fr);

  .review-image-wrap {
    height: 244rpx;
  }
}

.review-images--multi {
  grid-template-columns: repeat(3, 1fr);

  .review-image-wrap {
    height: 204rpx;
  }
}

.review-image-more {
  position: absolute;
  left: 0;
  right: 0;
  top: 0;
  bottom: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 34rpx;
  font-weight: 600;
  background-color: rgba(0, 0, 0, 0.45);
}

.review-actions {
  display: flex;
  justify-content: flex-end;
  margin-top: 18rpx;
}

.review-action {
  margin-left: 22rpx;
  font-size: 24rpx;
  color: #898b94;

  &.active {
    color: #27ba9b;
  }
}

.summary-action.active .action-icon--like,
.summary-action.active .action-icon--dislike,
.review-action.active .action-icon--like,
.review-action.active .action-icon--dislike {
  background-image: url("data:image/svg+xml,%3Csvg viewBox='0 0 40 40' xmlns='http://www.w3.org/2000/svg' fill='none' stroke='%2327ba9b' stroke-width='2.8' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpath d='M13 18v16H8.5A3.5 3.5 0 0 1 5 30.5v-9A3.5 3.5 0 0 1 8.5 18H13Z'/%3E%3Cpath d='M13 18l6.2-11.5c.7-1.3 2.5-1.5 3.5-.5.8.8 1 2 .5 3L21 15h10.2c2.4 0 4.1 2.2 3.6 4.5l-1.8 9.2A6.5 6.5 0 0 1 26.6 34H13V18Z'/%3E%3C/svg%3E");
}

/* 底部工具栏，延续商品详情页样式 */
.toolbar {
  position: fixed;
  left: 0;
  right: 0;
  bottom: calc((var(--window-bottom)));
  z-index: 1;
  background-color: #fff;
  height: 100rpx;
  padding: 0 20rpx;
  border-top: 1rpx solid #eaeaea;
  display: flex;
  justify-content: space-between;
  align-items: center;
  box-sizing: content-box;

  .buttons {
    display: flex;

    & > view {
      width: 220rpx;
      text-align: center;
      line-height: 72rpx;
      font-size: 26rpx;
      color: #fff;
      border-radius: 72rpx;
    }

    .addcart {
      background-color: #ffa868;
    }

    .payment {
      background-color: #27ba9b;
      margin-left: 20rpx;
    }
  }

  .icons {
    padding-right: 20rpx;
    display: flex;
    align-items: center;
    flex: 1;

    .navigator-wrap,
    .icons-button {
      flex: 1;
      text-align: center;
      line-height: 1.4;
      padding: 0;
      margin: 0;
      border-radius: 0;
      font-size: 20rpx;
      color: #333;
      background-color: #fff;

      &::after {
        border: none;
      }
    }

    text {
      display: block;
      font-size: 34rpx;
      transition: color 0.3s ease;
    }
  }
}

.icons-button {
  position: relative;
}
</style>
