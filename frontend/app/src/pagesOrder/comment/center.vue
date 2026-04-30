<script setup lang="ts">
import { computed, ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { defCommentService } from '@/api/app/comment'
import { defBaseDictService } from '@/api/app/base_dict'
import type { CommentItem, PendingCommentGoodsItem } from '@/rpc/app/v1/comment'
import type { BaseDictForm_DictItem } from '@/rpc/app/v1/base_dict'
import { formatSrc } from '@/utils'
import { orderCommentWriteUrl } from '@/utils/navigation'

const query = defineProps<{
  tab?: string
}>()

const { safeAreaInsets } = uni.getSystemInfoSync()

type CommentCenterTab = 'pending' | 'done'

type PendingCommentItem = PendingCommentGoodsItem & {
  id: string
  goods_picture: string
}

type DoneCommentItem = CommentItem & {
  goods_picture: string
  date: string
  content: string
  images: string[]
}

const activeTab = ref<CommentCenterTab>(query.tab === 'pending' ? 'pending' : 'done')
const page_size = 10
const pendingCommentList = ref<PendingCommentGoodsItem[]>([])
const doneCommentList = ref<CommentItem[]>([])
const pendingLoading = ref(false)
const pendingLoadingMore = ref(false)
const pendingHasMore = ref(false)
const pendingPageNum = ref(1)
const doneLoading = ref(false)
const doneLoadingMore = ref(false)
const doneHasMore = ref(false)
const donePageNum = ref(1)
const expandedDoneMap = ref<Record<number, boolean>>({})
const quickSubmittingId = ref('')
const deletingCommentId = ref<number>()
const commentStatusList = ref<BaseDictForm_DictItem[]>([])
const collapsedContentLength = 82
const commentStatusDictCode = 'comment_status'

const pendingItems = computed<PendingCommentItem[]>(() => {
  return (pendingCommentList.value || []).map((item) => ({
    ...item,
    id: `${item.order_id}-${item.goods_id}-${item.sku_code}`,
    goods_picture: formatSrc(item.goods_picture),
  }))
})

const doneComments = computed<DoneCommentItem[]>(() => {
  return (doneCommentList.value || []).map((item) => ({
    ...item,
    goods_picture: formatSrc(item.goods_picture),
    date: item.date_text,
    content: (item.content_segments || []).map((segment) => segment.text).join(''),
    images: (item.img || []).map((image) => formatSrc(image)),
  }))
})

const onNavigateBack = () => {
  const pages = getCurrentPages()
  if (pages.length > 1) {
    uni.navigateBack()
    return
  }
  uni.switchTab({ url: '/pages/my/my' })
}

const onSwitchTab = (tab: CommentCenterTab) => {
  activeTab.value = tab
}

const getCommentStatusText = (status: number) => {
  return commentStatusList.value.find((item) => item.value === String(status))?.label || ''
}

const buildWriteUrl = (item: PendingCommentItem) => {
  return orderCommentWriteUrl({
    order_id: item.order_id,
    goods_id: item.goods_id,
    goods_name: item.goods_name,
    goods_picture: item.goods_picture,
    sku_code: item.sku_code,
    sku_desc: item.sku_desc,
  })
}

// 一键好评只提交分数，内容和图片留空，并默认匿名。
const onQuickPraise = async (item: PendingCommentItem, score: number) => {
  if (quickSubmittingId.value) {
    return
  }

  quickSubmittingId.value = item.id
  try {
    await defCommentService.CreateComment({
      order_id: item.order_id,
      goods_id: item.goods_id,
      sku_code: item.sku_code,
      content: '',
      img: [],
      is_anonymous: true,
      goods_score: score,
      package_score: score,
      delivery_score: score,
    })
    await Promise.all([loadPendingCommentList(true), loadMyCommentList(true)])
    activeTab.value = 'done'
  } finally {
    quickSubmittingId.value = ''
  }
}

const isDoneExpanded = (id: number) => {
  return expandedDoneMap.value[id] === true
}

const getDoneContent = (item: DoneCommentItem) => {
  if (item.content.length <= collapsedContentLength || isDoneExpanded(item.id)) {
    return item.content
  }
  return `${item.content.slice(0, collapsedContentLength)}...`
}

const onToggleDoneContent = (id: number) => {
  expandedDoneMap.value = {
    ...expandedDoneMap.value,
    [id]: !isDoneExpanded(id),
  }
}

const onPreviewImages = (images: string[], index: number) => {
  if (!images.length) {
    return
  }
  uni.previewImage({
    current: images[index],
    urls: images,
  })
}

// 查询待评价商品列表，真实数据来自评论待评价接口。
const loadPendingCommentList = async (reset: boolean) => {
  if (reset) {
    pendingLoading.value = true
  } else {
    if (pendingLoadingMore.value || !pendingHasMore.value) {
      return
    }
    pendingLoadingMore.value = true
  }

  const nextPageNum = reset ? 1 : pendingPageNum.value + 1
  try {
    const res = await defCommentService.PagePendingCommentGoods({
      page_num: nextPageNum,
      page_size,
    })
    pendingCommentList.value = reset
      ? res.pending_comment_goods || []
      : [...pendingCommentList.value, ...(res.pending_comment_goods || [])]
    pendingPageNum.value = res.page_num || nextPageNum
    pendingHasMore.value = Boolean(res.has_more)
  } catch (_error) {
    if (reset) {
      pendingCommentList.value = []
      pendingPageNum.value = 1
      pendingHasMore.value = false
    }
  } finally {
    pendingLoading.value = false
    pendingLoadingMore.value = false
  }
}

// 查询我的评价列表，并回填真实星级、正文和图片数据。
const loadMyCommentList = async (reset: boolean) => {
  if (reset) {
    doneLoading.value = true
  } else {
    if (doneLoadingMore.value || !doneHasMore.value) {
      return
    }
    doneLoadingMore.value = true
  }

  const nextPageNum = reset ? 1 : donePageNum.value + 1
  try {
    const res = await defCommentService.PageMyComment({
      page_num: nextPageNum,
      page_size,
    })
    doneCommentList.value = reset
      ? res.comments || []
      : [...doneCommentList.value, ...(res.comments || [])]
    donePageNum.value = res.page_num || nextPageNum
    doneHasMore.value = Boolean(res.has_more)
  } catch (_error) {
    if (reset) {
      doneCommentList.value = []
      donePageNum.value = 1
      doneHasMore.value = false
    }
  } finally {
    doneLoading.value = false
    doneLoadingMore.value = false
  }
}

const loadCommentStatusDict = async () => {
  const dict = await defBaseDictService.GetBaseDict({ value: commentStatusDictCode })
  commentStatusList.value = dict.items || []
}

const onCommentCenterToLower = () => {
  if (activeTab.value === 'pending') {
    void loadPendingCommentList(false)
    return
  }
  void loadMyCommentList(false)
}

const getDoneBadgeText = (item: DoneCommentItem) => {
  if (item.goods_score >= 5) {
    return '超赞'
  }
  if (item.goods_score >= 4) {
    return '不错'
  }
  if (item.goods_score >= 3) {
    return '一般'
  }
  return '待提升'
}

const getDoneStars = (score: number) => {
  const normalizedScore = Math.max(0, Math.min(score || 0, 5))
  return `${'★'.repeat(normalizedScore)}${'☆'.repeat(5 - normalizedScore)}`
}

const onDeleteComment = (item: DoneCommentItem) => {
  if (deletingCommentId.value) {
    return
  }

  uni.showModal({
    title: '删除评价',
    content: '确认删除这条评价吗？',
    confirmText: '删除',
    confirmColor: '#f03246',
    success: (res) => {
      if (!res.confirm) {
        return
      }

      deletingCommentId.value = item.id
      void defCommentService
        .DeleteComment({ id: item.id })
        .then(async () => {
          void uni.showToast({
            title: '删除成功',
            icon: 'success',
          })
          await loadMyCommentList(true)
        })
        .finally(() => {
          deletingCommentId.value = undefined
        })
    },
  })
}

onShow(() => {
  void Promise.all([loadPendingCommentList(true), loadMyCommentList(true), loadCommentStatusDict()])
})
</script>

<template>
  <view class="comment-center-page">
    <view class="comment-center-header" :style="{ paddingTop: `${safeAreaInsets?.top || 0}px` }">
      <view class="comment-center-nav">
        <view class="back-button icon-left" @tap="onNavigateBack"></view>
        <view class="page-title">我的评价</view>
      </view>
      <view class="comment-tabs">
        <view
          class="comment-tab"
          :class="{ active: activeTab === 'pending' }"
          @tap="onSwitchTab('pending')"
        >
          待评价
        </view>
        <view
          class="comment-tab"
          :class="{ active: activeTab === 'done' }"
          @tap="onSwitchTab('done')"
        >
          已评价
        </view>
      </view>
    </view>

    <scroll-view scroll-y class="comment-center-body" @scrolltolower="onCommentCenterToLower">
      <template v-if="activeTab === 'pending'">
        <view v-if="pendingLoading" class="loading-card">评价商品加载中...</view>
        <XtxEmptyState
          v-else-if="!pendingItems.length"
          image="/static/images/empty_comment.png"
          text="暂无待评价商品"
          padding="110rpx 48rpx 0"
        />
        <template v-else>
          <view v-for="item in pendingItems" :key="item.id" class="pending-card">
            <view class="pending-goods">
              <image class="goods-cover" :src="item.goods_picture" mode="aspectFill" />
              <view class="goods-main">
                <view class="goods-name ellipsis-2">{{ item.goods_name }}</view>
                <view class="goods-desc">{{ item.desc }}</view>
              </view>
            </view>
            <view class="pending-actions">
              <view class="quick-stars" :class="{ disabled: quickSubmittingId === item.id }">
                <text class="quick-label">一键好评</text>
                <text
                  v-for="star in 5"
                  :key="star"
                  class="quick-star"
                  @tap.stop="onQuickPraise(item, star)"
                  >★</text
                >
              </view>
              <navigator class="write-button" :url="buildWriteUrl(item)" hover-class="none">
                去评价
              </navigator>
            </view>
          </view>
        </template>
      </template>

      <template v-else>
        <view v-if="doneLoading" class="loading-card">评价记录加载中...</view>
        <XtxEmptyState
          v-else-if="!doneComments.length"
          image="/static/images/empty_comment.png"
          text="暂无已评价内容"
          padding="110rpx 48rpx 0"
        />
        <template v-else>
          <view v-for="item in doneComments" :key="item.id" class="done-card">
            <view class="done-head">
              <view class="praise-badge">{{ getDoneBadgeText(item) }}</view>
              <view class="done-stars">{{ getDoneStars(item.goods_score) }}</view>
              <view class="done-date"
                >发布于{{ item.date }}{{ item.anonymous_for_owner ? ' 已匿名' : '' }}</view
              >
            </view>

            <view class="done-goods">
              <image class="done-goods-cover" :src="item.goods_picture" mode="aspectFill" />
              <view class="done-goods-name ellipsis">{{ item.goods_name }}</view>
              <text class="icon-right"></text>
            </view>

            <view v-if="item.content" class="done-content">
              {{ getDoneContent(item) }}
              <text
                v-if="item.content.length > collapsedContentLength"
                class="content-toggle"
                @tap="onToggleDoneContent(item.id)"
              >
                {{ isDoneExpanded(item.id) ? '收起' : '展开' }}
              </text>
            </view>
            <view
              v-if="item.images.length"
              class="done-images"
              :class="`done-images--${Math.min(item.images.length, 3)}`"
            >
              <image
                v-for="(image, imageIndex) in item.images"
                :key="`${image}-${imageIndex}`"
                class="done-image"
                :src="image"
                mode="aspectFill"
                @tap="onPreviewImages(item.images, imageIndex)"
              />
            </view>

            <view class="done-footer">
              <view class="done-status">{{ getCommentStatusText(item.status) }}</view>
              <view
                class="delete-comment-button"
                :class="{ disabled: deletingCommentId === item.id }"
                @tap="onDeleteComment(item)"
              >
                {{ deletingCommentId === item.id ? '删除中' : '删除评价' }}
              </view>
            </view>
          </view>
        </template>
      </template>
    </scroll-view>
  </view>
</template>

<style lang="scss">
page {
  height: 100%;
  overflow: hidden;
  background-color: #f4f4f4;
}

.comment-center-page {
  height: 100%;
  display: flex;
  flex-direction: column;
  color: #333;
  background-color: #f4f4f4;
}

.comment-center-header {
  position: relative;
  z-index: 2;
  flex-shrink: 0;
  background-color: #fff;
}

.comment-center-nav {
  height: 96rpx;
  display: flex;
  align-items: center;
  padding: 0 28rpx;
  box-sizing: border-box;
}

.back-button {
  width: 56rpx;
  height: 56rpx;
  display: flex;
  align-items: center;
  font-size: 42rpx;
  color: #333;
}

.page-title {
  flex: 1;
  margin-left: 16rpx;
  font-size: 36rpx;
  color: #111827;
  font-weight: 700;
}

.comment-tabs {
  display: flex;
  align-items: center;
  height: 88rpx;
  padding: 0 30rpx;
  border-top: 1rpx solid #f6f6f6;
  box-sizing: border-box;
}

.comment-tab {
  position: relative;
  flex: 1;
  height: 88rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 29rpx;
  color: #333;

  &.active {
    color: #27ba9b;
    font-weight: 600;

    &::after {
      content: '';
      position: absolute;
      left: 50%;
      bottom: 0;
      width: 48rpx;
      height: 6rpx;
      border-radius: 6rpx;
      background-color: #27ba9b;
      transform: translateX(-50%);
    }
  }
}

.comment-center-body {
  flex: 1;
  min-height: 0;
  background-color: #f4f4f4;
}

.pending-card,
.done-card,
.loading-card {
  margin: 20rpx;
  border-radius: 10rpx;
  background-color: #fff;
  box-shadow: 0 8rpx 24rpx rgba(15, 23, 42, 0.03);
}

.loading-card {
  padding: 36rpx 0;
  text-align: center;
  font-size: 26rpx;
  color: #898b94;
}

.pending-card {
  padding: 24rpx;
}

.pending-goods {
  display: flex;
}

.goods-cover {
  width: 140rpx;
  height: 140rpx;
  margin-right: 20rpx;
  flex-shrink: 0;
  border-radius: 10rpx;
  background-color: #f7f7f8;
}

.goods-main {
  flex: 1;
  min-width: 0;
}

.goods-name {
  min-height: 72rpx;
  font-size: 29rpx;
  line-height: 1.35;
  color: #111827;
  font-weight: 600;
}

.goods-desc {
  margin-top: 8rpx;
  font-size: 24rpx;
  color: #898b94;
}

.pending-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20rpx;
  margin-top: 24rpx;
}

.quick-stars {
  height: 68rpx;
  display: flex;
  align-items: center;
  flex: 1;
  min-width: 0;
  padding: 0 22rpx;
  border-radius: 10rpx;
  box-sizing: border-box;
  color: #d7dbe2;
  background-color: #f7f7f8;

  &.disabled {
    opacity: 0.6;
  }
}

.quick-label {
  margin-right: 18rpx;
  font-size: 26rpx;
  color: #898b94;
}

.quick-star {
  margin-right: 10rpx;
  padding: 8rpx 0;
  font-size: 36rpx;
  line-height: 1;
}

.write-button {
  width: 150rpx;
  height: 68rpx;
  flex-shrink: 0;
  border-radius: 10rpx;
  text-align: center;
  line-height: 68rpx;
  font-size: 28rpx;
  color: #fff;
  background-color: #27ba9b;
}

.done-card {
  padding: 28rpx 24rpx 24rpx;
}

.done-head {
  display: flex;
  align-items: center;
}

.praise-badge {
  margin-right: 10rpx;
  font-size: 27rpx;
  line-height: 1;
  color: #f03246;
  font-weight: 700;
}

.done-stars {
  margin-right: 18rpx;
  color: #f03246;
  font-size: 27rpx;
  letter-spacing: 2rpx;
}

.done-date {
  flex: 1;
  min-width: 0;
  font-size: 26rpx;
  color: #898b94;
}

.done-content {
  margin-top: 26rpx;
  font-size: 29rpx;
  line-height: 1.72;
  color: #333;
}

.content-toggle {
  margin-left: 8rpx;
  color: #898b94;
}

.done-images {
  display: grid;
  gap: 12rpx;
  margin-top: 18rpx;
}

.done-images--1 {
  width: 420rpx;
  grid-template-columns: 420rpx;
}

.done-images--2 {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.done-images--3 {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.done-image {
  width: 100%;
  height: 220rpx;
  border-radius: 10rpx;
  background-color: #f7f7f8;
}

.done-images--1 .done-image {
  height: 420rpx;
}

.done-goods {
  display: flex;
  align-items: center;
  margin-top: 26rpx;
  padding: 14rpx 16rpx;
  border-radius: 8rpx;
  background-color: #f6f7f9;
}

.done-goods-cover {
  width: 84rpx;
  height: 84rpx;
  margin-right: 16rpx;
  border-radius: 8rpx;
  background-color: #fff;
}

.done-goods-name {
  flex: 1;
  min-width: 0;
  font-size: 30rpx;
  color: #111827;
}

.done-goods .icon-right {
  color: #898b94;
  font-size: 28rpx;
}

.done-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18rpx;
  margin-top: 26rpx;
}

.done-status {
  display: flex;
  align-items: center;
  min-width: 0;
  font-size: 26rpx;
  color: #898b94;
}

.done-status .icon-right {
  margin-left: 4rpx;
  color: #898b94;
  font-size: 26rpx;
}

.delete-comment-button {
  width: 150rpx;
  height: 58rpx;
  flex-shrink: 0;
  border-radius: 8rpx;
  box-sizing: border-box;
  text-align: center;
  line-height: 58rpx;
  font-size: 27rpx;
  color: #f03246;
  font-weight: 600;
  background-color: #fff0f3;
}

.delete-comment-button.disabled {
  color: #c7cbd2;
  background-color: #f7f7f8;
}

.ellipsis {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ellipsis-2 {
  display: -webkit-box;
  overflow: hidden;
  text-overflow: ellipsis;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}
</style>
