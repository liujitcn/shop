<!-- 评论详情 -->
<template>
  <div v-loading="loading" class="app-container comment-detail-page">
    <el-card v-if="comment" class="comment-hero-card" shadow="never">
      <div class="comment-hero">
        <div class="comment-cover-panel">
          <el-image class="comment-cover-image" :src="formatImage(comment.goods_picture_snapshot)" fit="cover" preview-teleported>
            <template #error>
              <div class="image-placeholder">暂无主图</div>
            </template>
          </el-image>
        </div>

        <div class="comment-summary-panel">
          <div class="comment-summary-toolbar">
            <el-button
              v-if="BUTTONS['comment:status'] && comment.status !== CommentStatus.APPROVED_CS"
              :loading="approveLoading"
              type="success"
              @click="handleApproveComment"
            >
              通过评论
            </el-button>
            <el-button
              v-if="BUTTONS['comment:status'] && comment.status !== CommentStatus.REJECTED_CS"
              :loading="approveLoading"
              type="danger"
              @click="handleRejectComment"
            >
              不通过评论
            </el-button>
          </div>

          <div class="comment-title">{{ comment.goods_name_snapshot || "未命名商品" }}</div>
          <div class="comment-subtitle">{{ comment.sku_desc_snapshot || "暂无规格" }}</div>
          <el-space wrap>
            <el-tag type="warning">评分：{{ formatScore(comment.goods_score) }}</el-tag>
            <DictLabel :model-value="comment.status" code="comment_status" size="default" />
            <el-tag v-if="comment.pending_discussion_count > 0" type="danger">
              待审讨论 {{ comment.pending_discussion_count }}
            </el-tag>
          </el-space>
        </div>
      </div>
    </el-card>

    <el-card v-if="comment" class="comment-detail-panel" shadow="never">
      <el-tabs v-model="activeTabName" class="comment-detail-tabs">
        <el-tab-pane label="基本信息" name="basic">
          <div class="detail-tab-panel detail-tab-panel--basic">
            <el-descriptions :column="2" border class="comment-descriptions">
              <el-descriptions-item label="审核状态">
                <DictLabel :model-value="comment.status" code="comment_status" size="default" />
              </el-descriptions-item>
              <el-descriptions-item label="评价时间">{{ comment.created_at || "-" }}</el-descriptions-item>
              <el-descriptions-item label="商品名称" :span="2">{{ comment.goods_name_snapshot || "-" }}</el-descriptions-item>
              <el-descriptions-item label="商品规格" :span="2">{{
                comment.sku_desc_snapshot || "暂无规格"
              }}</el-descriptions-item>
              <el-descriptions-item label="用户昵称">{{ comment.user_name_snapshot || "-" }}</el-descriptions-item>
              <el-descriptions-item label="匿名展示">{{ comment.is_anonymous ? "是" : "否" }}</el-descriptions-item>
              <el-descriptions-item label="商品评分">{{ formatScore(comment.goods_score) }}</el-descriptions-item>
              <el-descriptions-item label="包装评分">{{ formatScore(comment.package_score) }}</el-descriptions-item>
              <el-descriptions-item label="配送评分">{{ formatScore(comment.delivery_score) }}</el-descriptions-item>
              <el-descriptions-item label="讨论数">{{ comment.discussion_count }}</el-descriptions-item>
              <el-descriptions-item label="待审核讨论">{{ comment.pending_discussion_count }}</el-descriptions-item>
              <el-descriptions-item label="最后更新">{{ comment.updated_at || "-" }}</el-descriptions-item>
              <el-descriptions-item label="评论内容" :span="2">
                <div class="comment-content">{{ comment.content || "暂无评论内容" }}</div>
              </el-descriptions-item>
            </el-descriptions>
          </div>
        </el-tab-pane>

        <el-tab-pane label="图片标签" name="media">
          <div class="detail-tab-panel">
            <div class="detail-section">
              <div class="detail-section-header">评价图片</div>
              <div class="detail-media-list">
                <el-image
                  v-for="(img, index) in commentImageList"
                  :key="`comment-img-${index}`"
                  class="detail-media-item"
                  :src="img"
                  :preview-src-list="commentImageList"
                  :initial-index="index"
                  fit="cover"
                  preview-teleported
                >
                  <template #error>
                    <div class="detail-media-item__placeholder">图片加载失败</div>
                  </template>
                </el-image>
                <div v-if="!commentImageList.length" class="detail-media-empty">暂无评价图片</div>
              </div>
            </div>

            <div class="detail-section">
              <div class="detail-section-header">评价标签</div>
              <el-space v-if="matchedTagList.length" wrap>
                <el-tag v-for="tag in matchedTagList" :key="tag.name" type="info">{{ tag.name }}</el-tag>
              </el-space>
              <div v-else class="detail-text-empty">暂无评价标签</div>
            </div>
          </div>
        </el-tab-pane>

        <el-tab-pane label="AI 摘要" name="ai">
          <div class="detail-tab-panel">
            <template v-if="aiSummaryList.length">
              <div v-for="item in aiSummaryList" :key="`${item.scene}-${item.created_at}`" class="ai-summary-card">
                <div class="ai-summary-header">
                  <span class="ai-summary-title">AI摘要</span>
                  <DictLabel :model-value="item.scene" code="comment_ai_scene" size="default" />
                  <span class="ai-summary-time">{{ item.updated_at || item.created_at || "-" }}</span>
                </div>

                <div v-if="item.content?.length" class="ai-content-list">
                  <div
                    v-for="(contentItem, index) in item.content"
                    :key="`${item.scene}-${index}-${contentItem.label}`"
                    class="ai-content-item"
                  >
                    <span class="ai-content-label">{{ contentItem.label || "摘要" }}</span>
                    <span class="ai-content-text">{{ contentItem.content || "-" }}</span>
                  </div>
                </div>
                <div v-else class="detail-text-empty">暂无摘要内容</div>
              </div>
            </template>
            <div v-else class="detail-text-empty">暂无AI摘要</div>
          </div>
        </el-tab-pane>

        <el-tab-pane label="审核记录" name="review">
          <div class="detail-tab-panel">
            <el-timeline v-if="reviewList.length" class="review-timeline">
              <el-timeline-item
                v-for="item in reviewList"
                :key="item.id"
                :timestamp="item.created_at || '-'"
                :type="reviewTimelineType(item.status)"
              >
                <div class="review-card">
                  <div class="review-card__title">
                    <el-tag size="small" :type="item.type === 1 ? 'primary' : 'warning'">
                      {{ item.type === 1 ? "AI审核" : "人工审核" }}
                    </el-tag>
                    <el-tag size="small" :type="reviewStatusTagType(item.status)">{{ reviewStatusText(item.status) }}</el-tag>
                    <span class="review-card__operator">{{ item.operator_name || "-" }}</span>
                  </div>
                  <div v-if="item.tags?.length" class="review-card__tags">
                    <el-tag v-for="tag in item.tags" :key="`${item.id}-${tag}`" size="small" type="info">{{ tag }}</el-tag>
                  </div>
                  <div class="review-card__reason">{{ item.reason || "无备注" }}</div>
                </div>
              </el-timeline-item>
            </el-timeline>
            <div v-else class="detail-text-empty">暂无审核记录</div>
          </div>
        </el-tab-pane>

        <el-tab-pane label="讨论列表" name="discussion">
          <DiscussionList
            :key="discussionListKey"
            class="detail-table-content"
            :comment-id="comment.id"
            :default-status="defaultDiscussionStatus"
            @audited="handleReloadCommentDetail"
          />
        </el-tab-pane>
      </el-tabs>
    </el-card>

    <el-empty v-else-if="!loading" description="暂无评论详情" />
  </div>
</template>

<script setup lang="ts">
import { computed, onActivated, ref, watch } from "vue";
import { useRoute } from "vue-router";
import DictLabel from "@/components/Dict/DictLabel.vue";
import { useAuthButtons } from "@/hooks/useAuthButtons";
import { useTabsStore } from "@/stores/modules/tabs";
import { defCommentInfoService } from "@/api/admin/comment_info";
import type {
  CommentAi,
  CommentDiscussion,
  CommentInfo,
  CommentInfoDetail,
  CommentReview,
  CommentTag
} from "@/rpc/admin/v1/comment_info";
import { CommentStatus } from "@/rpc/common/v1/enum";
import { formatSrc } from "@/utils/utils";
import DiscussionList from "../components/discussion/index.vue";

defineOptions({
  name: "CommentDetail",
  inheritAttrs: false
});

/** 评论详情页标签名称。 */
type CommentDetailTabName = "basic" | "media" | "ai" | "review" | "discussion";

/** 评论详情响应兼容结构，用于迁移期兼容旧字段名。 */
type CommentInfoDetailCompat = Partial<CommentInfoDetail> & {
  /** 旧协议商品评论标签字段 */
  commentTags?: CommentTag[];
  /** 旧协议商品评论标签字段 */
  tagList?: CommentTag[];
  /** 旧协议评论讨论列表字段 */
  commentDiscussions?: CommentDiscussion[];
  /** 旧协议评论讨论列表字段 */
  discussionList?: CommentDiscussion[];
  /** 旧协议商品评论 AI 摘要字段 */
  commentAis?: CommentAi[];
  /** 旧协议商品评论 AI 摘要字段 */
  aiList?: CommentAi[];
  /** 旧协议评论审核记录字段 */
  reviewList?: CommentReview[];
};

const route = useRoute();
const tabsStore = useTabsStore();
const { BUTTONS } = useAuthButtons();
const loading = ref(false);
const approveLoading = ref(false);
const commentId = ref(0);
const detailRequestId = ref(0);
const activeTabName = ref<CommentDetailTabName>("basic");
const defaultDiscussionStatus = ref<CommentStatus | undefined>();
const commentDetail = ref<CommentInfoDetailCompat>();
const workspaceTitle = "评论详情";

/** 当前评论内容。 */
const comment = computed<CommentInfo | undefined>(() => commentDetail.value?.comment);

/** 当前评论命中的标签列表。 */
const matchedTagList = computed<CommentTag[]>(() => {
  const detail = commentDetail.value;
  // 评论详情为空时，命中标签为空集合。
  if (!detail?.comment) return [];
  const tagIdList = Array.isArray(detail.comment.tag_id) ? detail.comment.tag_id : [];
  const tagList = Array.isArray(detail.comment_tags) ? detail.comment_tags : (detail.commentTags ?? detail.tagList ?? []);
  // 后端省略空数组字段时，前端统一按空集合处理，避免详情读取 length 报错。
  if (!tagIdList.length || !tagList.length) return [];
  const tagIdSet = new Set(tagIdList);
  return tagList.filter(item => tagIdSet.has(item.id));
});

/** 当前评论详情图片列表。 */
const commentImageList = computed<string[]>(() => {
  const imgList = comment.value?.img;
  // 兼容接口返回空数组字段被省略的场景。
  if (!Array.isArray(imgList)) return [];
  return imgList.map(item => formatImage(item)).filter(Boolean);
});

/** 当前评论 AI 摘要列表。 */
const aiSummaryList = computed<CommentAi[]>(() => {
  const detail = commentDetail.value;
  const list = detail?.comment_ais ?? detail?.commentAis ?? detail?.aiList;
  // 兼容接口返回空数组字段被省略的场景。
  if (!Array.isArray(list)) return [];
  return list;
});

/** 当前评论审核记录列表。 */
const reviewList = computed<CommentReview[]>(() => {
  const detail = commentDetail.value;
  const list = detail?.comment_reviews ?? detail?.reviewList;
  // 兼容接口返回空数组字段被省略的场景。
  if (!Array.isArray(list)) return [];
  return list;
});

/** 讨论列表组件 key，确保从待审核入口进入时搜索默认值即时生效。 */
const discussionListKey = computed(() => `${commentId.value}-${defaultDiscussionStatus.value ?? 0}`);

/** 判断当前是否仍停留在评论详情页，避免离开后继续刷新其他页面状态。 */
function isCurrentCommentDetailRoute() {
  return route.name === "CommentDetail" || route.path.includes("/admin/comment/detail/");
}

/** 从路由中同步当前评论记录。 */
function syncCommentIdFromRoute() {
  commentId.value = Number(route.params.commentId ?? 0);
  return commentId.value;
}

/** 从路由查询参数同步默认标签页和讨论筛选条件。 */
function syncViewStateFromRoute() {
  const tab = String(route.query.tab || "");
  activeTabName.value =
    tab === "discussion" || tab === "media" || tab === "ai" || tab === "review" ? (tab as CommentDetailTabName) : "basic";

  const discussionStatus = Number(route.query.discussionStatus ?? 0);
  // 只有明确携带有效状态时，才默认填充讨论列表搜索条件。
  if (
    discussionStatus === CommentStatus.PENDING_REVIEW_CS ||
    discussionStatus === CommentStatus.APPROVED_CS ||
    discussionStatus === CommentStatus.REJECTED_CS
  ) {
    defaultDiscussionStatus.value = discussionStatus;
    return;
  }
  defaultDiscussionStatus.value = undefined;
}

/** 同步当前页签和浏览器标题。 */
function syncWorkspaceTitle() {
  tabsStore.setTabsTitle(workspaceTitle);
  document.title = `${workspaceTitle} - ${import.meta.env.VITE_GLOB_APP_TITLE}`;
}

watch(
  () => [route.params.commentId, route.query.tab, route.query.discussionStatus],
  () => {
    if (!isCurrentCommentDetailRoute()) return;
    const oldCommentId = commentId.value;
    const currentCommentId = syncCommentIdFromRoute();
    syncViewStateFromRoute();
    syncWorkspaceTitle();
    if (!currentCommentId) {
      commentDetail.value = undefined;
      return;
    }
    // 只有评论记录变化或详情为空时才重新拉取详情，切换 tab 不重复请求。
    if (oldCommentId !== currentCommentId || !commentDetail.value) {
      handleQuery(currentCommentId);
    }
  },
  { immediate: true }
);

/** 查询评论详情。 */
function handleQuery(targetCommentId = commentId.value) {
  if (!targetCommentId) return;
  const requestId = ++detailRequestId.value;
  loading.value = true;
  defCommentInfoService
    .GetCommentInfo({ id: targetCommentId })
    .then(data => {
      if (requestId !== detailRequestId.value) return;
      commentDetail.value = data;
    })
    .catch(() => {
      if (requestId !== detailRequestId.value) return;
      commentDetail.value = undefined;
    })
    .finally(() => {
      if (requestId !== detailRequestId.value) return;
      loading.value = false;
    });
}

/** 重新加载当前评论详情。 */
function handleReloadCommentDetail() {
  handleQuery(commentId.value);
}

/** 审核通过当前评论。 */
async function handleApproveComment() {
  const currentComment = comment.value;
  // 当前详情没有评论时，不发起审核请求。
  if (!currentComment) return;
  try {
    await ElMessageBox.confirm(`是否确认通过该评论？\n评论内容：${currentComment.content || "暂无评论内容"}`, "提示", {
      confirmButtonText: "确认",
      cancelButtonText: "取消",
      type: "warning"
    });
    approveLoading.value = true;
    await defCommentInfoService.SetCommentInfoStatus({ id: currentComment.id, status: CommentStatus.APPROVED_CS, reason: "" });
    ElMessage.success("评论审核通过");
    handleReloadCommentDetail();
  } catch {
    // 用户主动取消时不需要额外处理。
  } finally {
    approveLoading.value = false;
  }
}

/** 审核不通过当前评论。 */
async function handleRejectComment() {
  const currentComment = comment.value;
  if (!currentComment) return;
  try {
    const { value } = await ElMessageBox.prompt(
      `请输入不通过原因\n评论内容：${currentComment.content || "暂无评论内容"}`,
      "评论审核不通过",
      {
        confirmButtonText: "确认",
        cancelButtonText: "取消",
        inputType: "textarea",
        inputPlaceholder: "请填写不通过原因",
        inputValidator: value => Boolean(String(value || "").trim()),
        inputErrorMessage: "请填写不通过原因"
      }
    );
    approveLoading.value = true;
    await defCommentInfoService.SetCommentInfoStatus({
      id: currentComment.id,
      status: CommentStatus.REJECTED_CS,
      reason: String(value || "").trim()
    });
    ElMessage.success("评论审核不通过");
    handleReloadCommentDetail();
  } catch {
    // 用户主动取消时不需要额外处理。
  } finally {
    approveLoading.value = false;
  }
}

/** 审核记录状态文案。 */
function reviewStatusText(status: number) {
  if (status === 1) return "通过";
  if (status === 2) return "不通过";
  if (status === 3) return "异常";
  return "未知";
}

/** 审核记录状态标签样式。 */
function reviewStatusTagType(status: number) {
  if (status === 1) return "success";
  if (status === 2) return "danger";
  if (status === 3) return "warning";
  return "info";
}

/** 审核记录时间线样式。 */
function reviewTimelineType(status: number) {
  if (status === 1) return "success";
  if (status === 2) return "danger";
  if (status === 3) return "warning";
  return "info";
}

/** 格式化图片地址。 */
function formatImage(src: string) {
  return formatSrc(src || "");
}

/** 格式化评分，统一补充单位。 */
function formatScore(score: number) {
  if (!score) return "-";
  return `${score} 分`;
}

onActivated(() => {
  if (!isCurrentCommentDetailRoute()) return;
  syncWorkspaceTitle();
  const currentCommentId = syncCommentIdFromRoute();
  syncViewStateFromRoute();
  if (!currentCommentId || loading.value) return;
  handleQuery(currentCommentId);
});
</script>

<style scoped lang="scss">
.comment-hero-card,
.comment-detail-panel {
  border: 1px solid var(--admin-page-card-border);
  border-radius: var(--admin-page-radius);
  background: var(--admin-page-card-bg);
  box-shadow: var(--admin-page-shadow);
}

.comment-hero-card {
  margin-bottom: 18px;
}

:deep(.comment-hero-card .el-card__body) {
  padding: 16px;
}

:deep(.comment-detail-panel .el-card__body) {
  padding: 0;
}

.comment-hero {
  display: grid;
  grid-template-columns: 112px minmax(0, 1fr);
  gap: 16px;
  align-items: stretch;
}

.comment-cover-panel {
  display: flex;
  box-sizing: border-box;
  min-width: 0;
  padding: 8px;
  border: 1px solid var(--admin-page-card-border-soft);
  border-radius: calc(var(--admin-page-radius) + 2px);
  background: var(--admin-page-card-bg-soft);
}

.comment-cover-image {
  width: 96px;
  height: 96px;
  overflow: hidden;
  border-radius: var(--admin-page-radius);
  background: var(--admin-page-card-bg-muted);
}

.image-placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
  padding: 8px;
  font-size: 12px;
  color: var(--admin-page-text-placeholder);
  background: var(--admin-page-card-bg-muted);
}

.comment-summary-panel {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 10px;
}

.comment-summary-toolbar {
  display: flex;
  justify-content: flex-end;
  min-height: 32px;
  gap: 8px;
}

.comment-title {
  min-width: 0;
  overflow: hidden;
  font-size: 18px;
  font-weight: 700;
  color: var(--admin-page-text-primary);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.comment-subtitle,
.comment-content {
  line-height: 1.6;
  color: var(--admin-page-text-secondary);
  white-space: pre-wrap;
}

.comment-detail-tabs :deep(.el-tabs__header) {
  margin: 0;
  padding: 0 16px;
}

.comment-detail-tabs :deep(.el-tabs__nav-wrap::after) {
  height: 1px;
  background-color: var(--admin-page-divider-strong);
}

.comment-detail-tabs :deep(.el-tabs__item) {
  height: 36px;
  padding: 0 8px;
  font-size: 14px;
  font-weight: 600;
}

.comment-detail-tabs :deep(.el-tabs__content) {
  padding: 0;
}

.detail-tab-panel {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 16px;
}

.detail-section + .detail-section {
  margin-top: 2px;
}

.detail-section-header {
  margin-bottom: 12px;
  font-size: 16px;
  font-weight: 600;
  color: var(--admin-page-text-primary);
}

.detail-media-list {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
}

.detail-media-item,
.detail-media-empty {
  width: 112px;
  height: 112px;
  overflow: hidden;
  border: 1px solid var(--admin-page-card-border-muted);
  border-radius: var(--admin-page-radius);
  background: var(--admin-page-card-bg-muted);
}

.detail-media-item {
  display: block;
}

.detail-media-empty,
.detail-media-item__placeholder,
.detail-text-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 8px;
  font-size: 12px;
  line-height: 1.5;
  color: var(--admin-page-text-placeholder);
}

.detail-media-empty,
.detail-media-item__placeholder {
  width: 100%;
  height: 100%;
  text-align: center;
  background: var(--admin-page-card-bg-muted);
}

.detail-text-empty {
  justify-content: flex-start;
  min-height: 36px;
  padding: 0;
}

.comment-descriptions :deep(.el-descriptions__label) {
  width: 110px;
  font-weight: 600;
}

.comment-descriptions :deep(.el-descriptions__cell) {
  padding: 10px 14px;
}

.detail-table-content {
  display: block;
}

.ai-summary-card {
  padding: 14px;
  border: 1px solid var(--admin-page-card-border-soft);
  border-radius: var(--admin-page-radius);
  background: var(--admin-page-card-bg-soft);
}

.ai-summary-header {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
  margin-bottom: 12px;
}

.ai-summary-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--admin-page-text-primary);
}

.ai-summary-time {
  margin-left: auto;
  font-size: 12px;
  color: var(--admin-page-text-placeholder);
}

.ai-content-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.ai-content-item {
  display: grid;
  grid-template-columns: 120px minmax(0, 1fr);
  gap: 12px;
  line-height: 1.6;
}

.ai-content-label {
  font-weight: 600;
  color: var(--admin-page-text-secondary);
}

.ai-content-text {
  min-width: 0;
  color: var(--admin-page-text-primary);
  white-space: pre-wrap;
}

.review-timeline {
  padding-left: 4px;
}

.review-card {
  padding: 12px;
  border: 1px solid var(--admin-page-card-border-soft);
  border-radius: var(--admin-page-radius);
  background: var(--admin-page-card-bg-soft);
}

.review-card__title,
.review-card__tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.review-card__operator {
  color: var(--admin-page-text-secondary);
}

.review-card__tags {
  margin-top: 10px;
}

.review-card__reason {
  margin-top: 10px;
  line-height: 1.6;
  color: var(--admin-page-text-primary);
  white-space: pre-wrap;
}

@media (width <= 768px) {
  .comment-hero {
    grid-template-columns: 1fr;
  }

  .comment-cover-image {
    width: 100%;
    height: 180px;
  }

  .comment-summary-toolbar {
    justify-content: flex-start;
  }

  .ai-summary-time {
    width: 100%;
    margin-left: 0;
  }

  .ai-content-item {
    grid-template-columns: 1fr;
    gap: 4px;
  }
}
</style>
