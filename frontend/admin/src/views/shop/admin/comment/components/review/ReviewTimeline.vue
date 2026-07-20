<template>
  <div v-loading="loading" class="comment-review-timeline">
    <el-timeline v-if="reviewList.length" class="review-timeline">
      <el-timeline-item
        v-for="item in reviewList"
        :key="item.id"
        :timestamp="item.created_at || '-'"
        :type="reviewTimelineType(item.status)"
      >
        <div class="review-card">
          <div class="review-card__title">
            <el-tag size="small" :type="item.type === CommentReviewType.COMMENT_REVIEW_TYPE_AI ? 'primary' : 'warning'">
              {{ item.type === CommentReviewType.COMMENT_REVIEW_TYPE_AI ? "AI审核" : "人工审核" }}
            </el-tag>
            <el-tag size="small" :type="reviewStatusTagType(item.status)">
              {{ reviewStatusText(item.status) }}
            </el-tag>
            <span class="review-card__operator">{{ item.operator_name || "-" }}</span>
          </div>
          <div v-if="item.tags?.length" class="review-card__tags">
            <el-tag v-for="tag in item.tags" :key="`${item.id}-${tag}`" size="small" type="info">
              {{ tag }}
            </el-tag>
          </div>
          <div class="review-card__reason">{{ item.reason || "无备注" }}</div>
        </div>
      </el-timeline-item>
    </el-timeline>
    <div v-else class="review-empty">{{ emptyText }}</div>
  </div>
</template>

<script setup lang="ts">
import type { CommentReview } from "@/rpc/shop/admin/v1/comment_info";
import { CommentReviewStatus, CommentReviewType } from "@/rpc/shop/common/v1/enum";

/** 审核记录时间线组件入参。 */
interface ReviewTimelineProps {
  /** 审核记录列表 */
  reviewList: CommentReview[];
  /** 审核记录加载状态 */
  loading?: boolean;
  /** 空数据提示文案 */
  emptyText?: string;
}

withDefaults(defineProps<ReviewTimelineProps>(), {
  loading: false,
  emptyText: "暂无审核记录"
});

/** 审核记录状态文案。 */
function reviewStatusText(status: number) {
  if (status === CommentReviewStatus.COMMENT_REVIEW_STATUS_APPROVED) return "通过";
  if (status === CommentReviewStatus.COMMENT_REVIEW_STATUS_REJECTED) return "不通过";
  if (status === CommentReviewStatus.COMMENT_REVIEW_STATUS_EXCEPTION) return "异常";
  return "未知";
}

/** 审核记录状态标签样式。 */
function reviewStatusTagType(status: number) {
  if (status === CommentReviewStatus.COMMENT_REVIEW_STATUS_APPROVED) return "success";
  if (status === CommentReviewStatus.COMMENT_REVIEW_STATUS_REJECTED) return "danger";
  if (status === CommentReviewStatus.COMMENT_REVIEW_STATUS_EXCEPTION) return "warning";
  return "info";
}

/** 审核记录时间线样式。 */
function reviewTimelineType(status: number) {
  if (status === CommentReviewStatus.COMMENT_REVIEW_STATUS_APPROVED) return "success";
  if (status === CommentReviewStatus.COMMENT_REVIEW_STATUS_REJECTED) return "danger";
  if (status === CommentReviewStatus.COMMENT_REVIEW_STATUS_EXCEPTION) return "warning";
  return "info";
}
</script>

<style scoped lang="scss">
.comment-review-timeline {
  min-height: 36px;
}
.review-timeline {
  padding-left: 4px;
}
.review-card {
  padding: 12px;
  background: var(--admin-page-card-bg-soft);
  border: 1px solid var(--admin-page-card-border-soft);
  border-radius: var(--admin-page-radius);
  &__title,
  &__tags {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    align-items: center;
  }
  &__operator {
    color: var(--admin-page-text-secondary);
  }
  &__tags {
    margin-top: 10px;
  }
  &__reason {
    margin-top: 10px;
    line-height: 1.6;
    color: var(--admin-page-text-primary);
    white-space: pre-wrap;
  }
}
.review-empty {
  display: flex;
  align-items: center;
  min-height: 36px;
  font-size: 12px;
  line-height: 1.5;
  color: var(--admin-page-text-placeholder);
}
</style>
