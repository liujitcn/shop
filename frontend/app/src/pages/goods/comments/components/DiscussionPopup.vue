<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { defCommentService } from '@/api/app/comment'
import type { CommentDiscussionItem } from '@/rpc/app/v1/comment'
import { CommentReactionTargetType, CommentReactionType } from '@/rpc/common/v1/enum'
import { formatSrc } from '@/utils'
import { navigateToLogin } from '@/utils/navigation'
import { useUserStore } from '@/stores'
import defaultAvatar from '@/static/images/avatar.png'

const props = defineProps<{
  visible: boolean
  reviewId?: number
}>()

const emit = defineEmits<{
  close: []
  countChange: [payload: { comment_id: number; discussion_count: number }]
}>()

const { safeAreaInsets } = uni.getSystemInfoSync()
const userStore = useUserStore()
const ANONYMOUS_USER_NAME = '匿名用户'
const pageSize = 20
const discussionList = ref<CommentDiscussionItem[]>([])
const discussionTotal = ref(0)
const currentPageNum = ref(1)
const hasMore = ref(false)
const isLoading = ref(false)
const isLoadingMore = ref(false)
const isSubmitting = ref(false)
const commentDraft = ref('')
const replyTargetName = ref('')
const replyTargetDiscussionId = ref(0)
const commentInputFocus = ref(false)

const currentReviewId = computed(() => props.reviewId || 0)
const discussionCount = computed(() => discussionTotal.value)

const commentPlaceholder = computed(() => {
  return replyTargetName.value ? `回复 ${replyTargetName.value}` : '说说你的想法～'
})

const ensureLogin = () => {
  if (userStore.userInfo) {
    return true
  }
  navigateToLogin()
  return false
}

const getDiscussionUserName = (item: CommentDiscussionItem) => {
  return item.user?.user_name || ANONYMOUS_USER_NAME
}

const getDiscussionAvatar = (item: CommentDiscussionItem) => {
  if (!item.user?.avatar) {
    return defaultAvatar
  }
  return formatSrc(item.user.avatar)
}

const getDiscussionRole = (item: CommentDiscussionItem) => {
  return item.user?.user_tag_text || ''
}

const isDiscussionReactionActive = (
  item: CommentDiscussionItem,
  reaction_type: CommentReactionType,
) => {
  return item.reaction_type === reaction_type
}

// 打开弹层时拉取真实讨论列表，并在翻页时继续追加。
const loadDiscussionData = async (reset: boolean) => {
  if (currentReviewId.value <= 0) {
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
    const res = await defCommentService.PageCommentDiscussion({
      comment_id: currentReviewId.value,
      page_num: nextPageNum,
      page_size: pageSize,
    })
    discussionList.value = reset
      ? res.comment_discussions || []
      : [...discussionList.value, ...(res.comment_discussions || [])]
    discussionTotal.value = res.total || 0
    currentPageNum.value = res.page_num || nextPageNum
    hasMore.value = Boolean(res.has_more)
  } catch (_error) {
    if (reset) {
      discussionList.value = []
      discussionTotal.value = 0
      currentPageNum.value = 1
      hasMore.value = false
    }
  } finally {
    isLoading.value = false
    isLoadingMore.value = false
  }
}

watch(
  () => [props.visible, props.reviewId] as const,
  ([visible]) => {
    if (!visible) {
      return
    }
    replyTargetName.value = ''
    replyTargetDiscussionId.value = 0
    commentDraft.value = ''
    commentInputFocus.value = false
    void loadDiscussionData(true)
  },
)

const onClose = () => {
  emit('close')
}

const focusCommentInput = () => {
  commentInputFocus.value = false
  nextTick(() => {
    commentInputFocus.value = true
  })
}

const onReplyTo = (item: CommentDiscussionItem) => {
  replyTargetDiscussionId.value = item.id
  replyTargetName.value = getDiscussionUserName(item)
  focusCommentInput()
}

const onDiscussionToLower = () => {
  void loadDiscussionData(false)
}

const onSubmitDiscussion = async () => {
  if (!ensureLogin()) {
    return
  }
  if (isSubmitting.value) {
    return
  }

  const content = commentDraft.value.trim()
  if (!content) {
    void uni.showToast({ title: '先说点什么吧', icon: 'none' })
    return
  }

  isSubmitting.value = true
  try {
    const res = await defCommentService.CreateCommentDiscussion({
      comment_id: currentReviewId.value,
      content,
      parent_id: replyTargetDiscussionId.value || 0,
      reply_to_discussion_id: replyTargetDiscussionId.value || 0,
      is_anonymous: false,
    })
    discussionList.value = res.item ? [res.item, ...discussionList.value] : discussionList.value
    discussionTotal.value = res.discussion_count ?? discussionTotal.value
    emit('countChange', {
      comment_id: currentReviewId.value,
      discussion_count: discussionTotal.value,
    })
    commentDraft.value = ''
    replyTargetName.value = ''
    replyTargetDiscussionId.value = 0
    commentInputFocus.value = false
    void uni.showToast({ title: '讨论已提交，审核通过后展示', icon: 'none' })
    onClose()
  } finally {
    isSubmitting.value = false
  }
}

const onToggleLike = async (item: CommentDiscussionItem) => {
  if (!ensureLogin()) {
    return
  }

  const res = await defCommentService.SaveCommentReaction({
    target_type: CommentReactionTargetType.DISCUSSION,
    target_id: item.id,
    reaction_type: CommentReactionType.LIKE,
    active: !isDiscussionReactionActive(item, CommentReactionType.LIKE),
  })
  discussionList.value = discussionList.value.map((discussion) => {
    if (discussion.id !== item.id) {
      return discussion
    }
    return {
      ...discussion,
      reaction_type: res.reaction_type,
      like_count: res.like_count,
    }
  })
}
</script>

<template>
  <view v-if="visible" class="discussion-popup-mask" @tap="onClose">
    <view
      class="discussion-popup"
      :style="{ paddingBottom: `${safeAreaInsets?.bottom || 0}px` }"
      @tap.stop
    >
      <view class="discussion-header">
        <view class="discussion-title">
          全部讨论
          <text v-if="discussionCount">{{ discussionCount }}</text>
        </view>
        <view class="discussion-close" @tap="onClose">×</view>
      </view>

      <scroll-view scroll-y class="discussion-scroll" @scrolltolower="onDiscussionToLower">
        <view v-if="discussionList.length" class="discussion-list">
          <view v-for="item in discussionList" :key="item.id" class="discussion-item">
            <image class="discussion-avatar" :src="getDiscussionAvatar(item)" mode="aspectFill" />
            <view class="discussion-main">
              <view class="discussion-head">
                <view class="discussion-user">
                  <text class="discussion-name">{{ getDiscussionUserName(item) }}</text>
                  <text v-if="getDiscussionRole(item)" class="discussion-role">{{
                    getDiscussionRole(item)
                  }}</text>
                </view>
                <view
                  class="discussion-like"
                  :class="{ active: isDiscussionReactionActive(item, CommentReactionType.LIKE) }"
                  @tap="onToggleLike(item)"
                >
                  <view class="discussion-like-icon" />
                  <text>{{ item.like_count || '赞' }}</text>
                </view>
              </view>

              <view class="discussion-content">
                <text v-if="item.reply_to_display_name" class="discussion-reply-prefix"
                  >回复 {{ item.reply_to_display_name }}：</text
                >{{ item.content }}
              </view>
              <view class="discussion-meta">
                {{ item.date_text }} <text @tap="onReplyTo(item)">回复</text>
              </view>
            </view>
          </view>
          <view v-if="isLoadingMore" class="discussion-loading-more">更多讨论加载中...</view>
        </view>

        <view v-else-if="isLoading" class="discussion-loading-more">讨论加载中...</view>
        <XtxEmptyState
          v-else
          image="/static/images/empty_discussion.png"
          text="暂无讨论"
          image-width="180rpx"
          image-height="150rpx"
          min-height="460rpx"
          padding="72rpx 0 88rpx"
        />
      </scroll-view>

      <view class="discussion-input-bar">
        <input
          v-model="commentDraft"
          class="discussion-input"
          :focus="commentInputFocus"
          :placeholder="commentPlaceholder"
          confirm-type="send"
          placeholder-class="discussion-input-placeholder"
          @confirm="onSubmitDiscussion"
        />
        <view
          class="discussion-send"
          :class="{ active: commentDraft.trim() && !isSubmitting }"
          @tap="onSubmitDiscussion"
        >
          {{ isSubmitting ? '发送中' : '发送' }}
        </view>
      </view>
    </view>
  </view>
</template>

<style lang="scss">
.discussion-popup-mask {
  position: fixed;
  left: 0;
  right: 0;
  top: 0;
  bottom: 0;
  z-index: 30;
  display: flex;
  align-items: flex-end;
  background-color: rgba(0, 0, 0, 0.58);
}

.discussion-popup {
  width: 100%;
  max-height: 74%;
  min-height: 56%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border-radius: 28rpx 28rpx 0 0;
  background-color: #fff;
}

.discussion-header {
  position: relative;
  height: 102rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.discussion-title {
  font-size: 32rpx;
  color: #222;
  font-weight: 600;

  text {
    margin-left: 8rpx;
    color: #898b94;
    font-size: 24rpx;
    font-weight: 400;
  }
}

.discussion-close {
  position: absolute;
  right: 30rpx;
  top: 50%;
  width: 52rpx;
  height: 52rpx;
  text-align: center;
  line-height: 48rpx;
  font-size: 52rpx;
  color: #333;
  font-weight: 300;
  transform: translateY(-50%);
}

.discussion-scroll {
  flex: 1;
  min-height: 0;
}

.discussion-list {
  padding-bottom: 16rpx;
}

.discussion-item {
  display: flex;
  padding: 22rpx 32rpx;
}

.discussion-avatar {
  width: 64rpx;
  height: 64rpx;
  margin-right: 18rpx;
  flex-shrink: 0;
  border-radius: 50%;
  background-color: #f2f3f5;
}

.discussion-main {
  flex: 1;
  min-width: 0;
}

.discussion-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 34rpx;
}

.discussion-user {
  display: flex;
  align-items: center;
  min-width: 0;
}

.discussion-name {
  max-width: 230rpx;
  margin-right: 10rpx;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 26rpx;
  color: #898b94;
}

.discussion-role {
  padding: 2rpx 10rpx;
  border-radius: 6rpx;
  font-size: 21rpx;
  color: #168f78;
  background-color: #e9f8f5;
}

.discussion-like {
  display: flex;
  align-items: center;
  margin-left: 16rpx;
  flex-shrink: 0;
  font-size: 24rpx;
  color: #898b94;

  &.active {
    color: #27ba9b;
  }
}

.discussion-like-icon {
  width: 34rpx;
  height: 34rpx;
  margin-right: 4rpx;
  background-position: center;
  background-repeat: no-repeat;
  background-size: 100% 100%;
  background-image: url("data:image/svg+xml,%3Csvg viewBox='0 0 40 40' xmlns='http://www.w3.org/2000/svg' fill='none' stroke='%238f96a3' stroke-width='2.8' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpath d='M13 18v16H8.5A3.5 3.5 0 0 1 5 30.5v-9A3.5 3.5 0 0 1 8.5 18H13Z'/%3E%3Cpath d='M13 18l6.2-11.5c.7-1.3 2.5-1.5 3.5-.5.8.8 1 2 .5 3L21 15h10.2c2.4 0 4.1 2.2 3.6 4.5l-1.8 9.2A6.5 6.5 0 0 1 26.6 34H13V18Z'/%3E%3C/svg%3E");
}

.discussion-like.active .discussion-like-icon {
  background-image: url("data:image/svg+xml,%3Csvg viewBox='0 0 40 40' xmlns='http://www.w3.org/2000/svg' fill='none' stroke='%2327ba9b' stroke-width='2.8' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpath d='M13 18v16H8.5A3.5 3.5 0 0 1 5 30.5v-9A3.5 3.5 0 0 1 8.5 18H13Z'/%3E%3Cpath d='M13 18l6.2-11.5c.7-1.3 2.5-1.5 3.5-.5.8.8 1 2 .5 3L21 15h10.2c2.4 0 4.1 2.2 3.6 4.5l-1.8 9.2A6.5 6.5 0 0 1 26.6 34H13V18Z'/%3E%3C/svg%3E");
}

.discussion-content {
  margin-top: 12rpx;
  font-size: 30rpx;
  line-height: 1.58;
  color: #222;
}

.discussion-reply-prefix {
  color: #168f78;
  font-weight: 600;
}

.discussion-meta {
  margin-top: 12rpx;
  font-size: 24rpx;
  line-height: 1.2;
  color: #999;

  text {
    color: #777;
  }
}

.discussion-loading-more {
  padding: 24rpx 0;
  text-align: center;
  font-size: 24rpx;
  color: #999;
}

.discussion-input-bar {
  display: flex;
  align-items: center;
  padding: 18rpx 32rpx 20rpx;
  flex-shrink: 0;
  border-top: 1rpx solid #f0f0f0;
  background-color: #fff;
}

.discussion-input {
  height: 72rpx;
  padding: 0 24rpx;
  flex: 1;
  border-radius: 14rpx;
  font-size: 27rpx;
  color: #333;
  background-color: #f7f7f8;
  box-sizing: border-box;
}

.discussion-input-placeholder {
  color: #999;
}

.discussion-send {
  width: 104rpx;
  height: 72rpx;
  margin-left: 18rpx;
  border-radius: 999rpx;
  text-align: center;
  line-height: 72rpx;
  font-size: 26rpx;
  color: #fff;
  background-color: #b7e5db;

  &.active {
    background-color: #27ba9b;
  }
}
</style>
