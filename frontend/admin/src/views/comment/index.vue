<!-- 评论管理 -->
<template>
  <div class="table-box comment-page">
    <ProTable ref="proTable" row-key="id" :columns="columns" :request-api="requestCommentTable" :init-param="commentInitParam">
      <template #goods_name_snapshot="scope">
        <el-link v-if="BUTTONS['goods:info:detail']" type="primary" @click.stop="handleOpenGoodsDetail(scope.row)">
          {{ scope.row.goods_name_snapshot || "未命名商品" }}
        </el-link>
        <span v-else>{{ scope.row.goods_name_snapshot || "未命名商品" }}</span>
      </template>
      <template #discussion_count="scope">
        <el-space>
          <span>{{ scope.row.discussion_count || 0 }}</span>
          <el-link
            v-if="BUTTONS['comment:detail'] && scope.row.pending_discussion_count > 0"
            type="danger"
            @click.stop="handleOpenPendingDiscussion(scope.row)"
          >
            待审 {{ scope.row.pending_discussion_count }}
          </el-link>
          <el-tag v-else-if="scope.row.pending_discussion_count > 0" type="danger">
            待审 {{ scope.row.pending_discussion_count }}
          </el-tag>
        </el-space>
      </template>
    </ProTable>

    <el-dialog v-model="approveDialog.visible" title="评论审核" width="560px" destroy-on-close @closed="handleResetApproveDialog">
      <div v-if="approveDialog.row" class="approve-preview">
        <el-descriptions :column="1" border>
          <el-descriptions-item label="商品名称">{{ approveDialog.row.goods_name_snapshot }}</el-descriptions-item>
          <el-descriptions-item label="用户昵称">{{ approveDialog.row.user_name_snapshot }}</el-descriptions-item>
          <el-descriptions-item label="评论内容">
            <div class="approve-content">{{ approveDialog.row.content || "暂无评论内容" }}</div>
          </el-descriptions-item>
        </el-descriptions>

        <section class="comment-section">
          <h3>评论图片</h3>
          <div v-if="approveImageList.length" class="image-list">
            <el-image
              v-for="item in approveImageList"
              :key="item"
              :src="item"
              :preview-src-list="approveImageList"
              fit="cover"
              class="comment-image"
            />
          </div>
          <el-empty v-else description="暂无图片" :image-size="80" />
        </section>
      </div>

      <el-form label-position="top" class="approve-form">
        <el-form-item label="审核备注 / 不通过原因">
          <el-input
            v-model="approveDialog.reason"
            type="textarea"
            :rows="3"
            maxlength="200"
            show-word-limit
            placeholder="通过可选填备注；不通过请填写原因"
          />
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="handleCancelApprove">取消</el-button>
        <el-button type="danger" :loading="approveDialog.loading" @click="handleConfirmReject">不通过</el-button>
        <el-button type="success" :loading="approveDialog.loading" @click="handleConfirmApprove">通过</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import type { ColumnProps, ProTableInstance } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
import { useAuthButtons } from "@/hooks/useAuthButtons";
import { defCommentInfoService } from "@/api/admin/comment_info";
import type { CommentInfo, PageCommentInfosRequest } from "@/rpc/admin/v1/comment_info";
import { CommentStatus } from "@/rpc/common/v1/enum";
import { buildPageRequest } from "@/utils/proTable";
import { navigateTo } from "@/utils/router";

/** 评论审核确认弹窗状态。 */
type ApproveDialogState = {
  /** 弹窗是否显示 */
  visible: boolean;
  /** 审核请求提交状态 */
  loading: boolean;
  /** 当前待审核评论 */
  row?: CommentInfo;
  /** 审核备注或不通过原因 */
  reason: string;
};

defineOptions({
  name: "CommentInfo",
  inheritAttrs: false
});

const { BUTTONS } = useAuthButtons();
const proTable = ref<ProTableInstance>();
const route = useRoute();
const router = useRouter();

/** 工作台跳转评论列表时支持同步的查询参数。 */
const workspaceQueryKeys = ["status", "has_pending_discussion", "min_goods_score", "max_goods_score", "goods_score"] as const;

/** 评论列表工作台过滤参数，作为 ProTable 首次渲染的初始搜索条件。 */
const commentInitParam = computed(() => buildCommentQueryParam());

const approveDialog = reactive<ApproveDialogState>({
  visible: false,
  loading: false,
  row: undefined,
  reason: ""
});

/** 当前审核弹窗图片列表。 */
const approveImageList = computed<string[]>(() => {
  const imgList = approveDialog.row?.img;
  // 兼容列表行图片字段为空或被省略的场景。
  if (!Array.isArray(imgList)) return [];
  return imgList;
});

/** 评论审核表格列配置。 */
const columns: ColumnProps[] = [
  { prop: "goods_picture_snapshot", label: "商品图", width: 90, cellType: "image", imageProps: { width: 48, height: 48 } },
  { prop: "goods_name_snapshot", label: "商品名称", minWidth: 220, search: { el: "input", key: "goods_name" } },
  { prop: "user_name_snapshot", label: "用户昵称", minWidth: 120, search: { el: "input", key: "user_name" } },
  { prop: "goods_score", label: "评分", width: 90, search: { el: "input-number", props: { min: 1, max: 5, precision: 0 } } },
  { prop: "content", label: "评论内容", minWidth: 260, showOverflowTooltip: true },
  { prop: "discussion_count", label: "讨论数", width: 130 },
  { prop: "status", label: "审核状态", width: 120, dictCode: "comment_status", search: { el: "select" } },
  { prop: "created_at", label: "评价时间", minWidth: 170 },
  {
    prop: "operation",
    label: "操作",
    width: 150,
    fixed: "right",
    cellType: "actions",
    actions: [
      {
        label: "详情",
        type: "primary",
        link: true,
        icon: View,
        hidden: () => !BUTTONS.value["comment:detail"],
        onClick: scope => handleOpenCommentDetail((scope.row as CommentInfo).id)
      },
      {
        label: "通过",
        type: "success",
        link: true,
        icon: CircleCheck,
        hidden: scope => !BUTTONS.value["comment:status"] || (scope.row as CommentInfo).status === CommentStatus.APPROVED_CS,
        onClick: scope => handleApproveComment(scope.row as CommentInfo)
      },
      {
        label: "不通过",
        type: "danger",
        link: true,
        icon: CircleClose,
        hidden: scope => !BUTTONS.value["comment:status"] || (scope.row as CommentInfo).status === CommentStatus.REJECTED_CS,
        onClick: scope => handleRejectComment(scope.row as CommentInfo)
      }
    ]
  }
];

/**
 * 将路由 query 单值化，兼容 vue-router 数组参数。
 */
function getRouteQueryValue(value: unknown) {
  if (Array.isArray(value)) return value[0];
  return value;
}

/**
 * 解析大于 0 的数字 query，避免空字符串或非法值污染列表查询。
 */
function parsePositiveNumberQuery(value: unknown) {
  const rawValue = getRouteQueryValue(value);
  const numberValue = Number(rawValue ?? 0);
  return numberValue > 0 ? numberValue : undefined;
}

/**
 * 从工作台跳转 query 构建评论列表过滤参数。
 */
function buildCommentQueryParam() {
  const numberParam = workspaceQueryKeys.reduce<Record<string, number | boolean | undefined>>((params, key) => {
    params[key] = parsePositiveNumberQuery(route.query[key]);
    return params;
  }, {});

  // 后端该字段是 bool，工作台用 1 表示只看存在待审核讨论的评价。
  if (numberParam.has_pending_discussion !== undefined) {
    numberParam.has_pending_discussion = Boolean(numberParam.has_pending_discussion);
  }

  return numberParam;
}

/**
 * 同步工作台 query 到 ProTable 搜索参数，保证初始化和路由变化都能触发表格过滤。
 */
function syncWorkspaceQuery() {
  const queryParam = buildCommentQueryParam();
  if (!proTable.value) return;

  Object.assign(proTable.value.searchParam, queryParam);
  Object.assign(proTable.value.searchInitParam, queryParam);
}

watch(
  () => [route.query, proTable.value],
  () => {
    syncWorkspaceQuery();
    if (!proTable.value) return;

    // 工作台查询条件变化后回到第一页，避免沿用旧分页导致结果为空。
    proTable.value.pageable.pageNum = 1;
    proTable.value.search();
  },
  { immediate: true }
);

/** 请求评论列表，并由 ProTable 统一维护分页与搜索参数。 */
async function requestCommentTable(params: Record<string, any>) {
  const { pageNum, pageSize, ...requestParams } = buildPageRequest(params);
  const data = await defCommentInfoService.PageCommentInfos({
    ...requestParams,
    page_num: Number(pageNum),
    page_size: Number(pageSize)
  } as PageCommentInfosRequest);
  const compatData = data as typeof data & { commentInfos?: typeof data.comment_infos; list?: typeof data.comment_infos };
  // ProTable 固定消费 list，优先使用新 snake_case 字段并兼容历史响应。
  const list = compatData.comment_infos ?? compatData.commentInfos ?? compatData.list ?? [];
  return { data: { ...data, list } };
}

/** 刷新评论审核列表。 */
function refreshTable() {
  proTable.value?.getTableList();
}

/** 打开评论详情页面。 */
function handleOpenCommentDetail(commentId: number) {
  if (!commentId) {
    ElMessage.warning("评论记录不存在");
    return;
  }
  void navigateTo(router, `/admin/comment/detail/${commentId}`);
}

/** 打开评论详情页的待审核讨论标签。 */
function handleOpenPendingDiscussion(row: CommentInfo) {
  if (!row.id) {
    ElMessage.warning("评论记录不存在");
    return;
  }
  void navigateTo(router, `/admin/comment/detail/${row.id}`, {
    tab: "discussion",
    discussionStatus: CommentStatus.PENDING_REVIEW_CS
  });
}

/** 跳转到商品详情页面，保持商品列表页一致的详情入口。 */
function handleOpenGoodsDetail(row: CommentInfo) {
  // 商品不存在时不跳转，避免进入无效详情页。
  if (!row.goods_id) {
    ElMessage.warning("商品记录不存在");
    return;
  }
  void navigateTo(router, `/goods/detail/${row.goods_id}`);
}

/** 打开单条评论审核弹窗。 */
function openReviewDialog(row: CommentInfo) {
  approveDialog.row = row;
  approveDialog.reason = "";
  approveDialog.visible = true;
}

/** 审核通过单条评论。 */
function handleApproveComment(row: CommentInfo) {
  openReviewDialog(row);
}

/** 审核不通过单条评论。 */
function handleRejectComment(row: CommentInfo) {
  openReviewDialog(row);
}

/** 取消当前评论审核确认。 */
function handleCancelApprove() {
  approveDialog.visible = false;
}

/** 弹窗关闭后重置当前审核记录。 */
function handleResetApproveDialog() {
  // 请求提交中不清理记录，避免确认请求读取不到当前评论。
  if (approveDialog.loading) return;
  approveDialog.row = undefined;
  approveDialog.reason = "";
}

/** 确认通过当前评论审核。 */
async function handleConfirmApprove() {
  const row = approveDialog.row;
  // 弹窗没有选中评论时，不发起审核请求。
  if (!row) return;

  approveDialog.loading = true;
  try {
    await defCommentInfoService.SetCommentInfoStatus({
      id: row.id,
      status: CommentStatus.APPROVED_CS,
      reason: approveDialog.reason.trim()
    });
    ElMessage.success("评论审核通过");
    approveDialog.visible = false;
    refreshTable();
  } finally {
    approveDialog.loading = false;
  }
}

/** 确认不通过当前评论审核。 */
async function handleConfirmReject() {
  const row = approveDialog.row;
  if (!row) return;
  const reason = approveDialog.reason.trim();
  // 不通过必须填写原因，便于客服和用户侧追溯。
  if (!reason) {
    ElMessage.warning("请填写不通过原因");
    return;
  }

  approveDialog.loading = true;
  try {
    await defCommentInfoService.SetCommentInfoStatus({ id: row.id, status: CommentStatus.REJECTED_CS, reason });
    ElMessage.success("评论审核不通过");
    approveDialog.visible = false;
    refreshTable();
  } finally {
    approveDialog.loading = false;
  }
}
</script>

<style scoped lang="scss">
.comment-page {
  .comment-section {
    margin-top: 18px;
  }

  .image-list {
    display: flex;
    flex-wrap: wrap;
    gap: 12px;
  }

  .comment-image {
    width: 88px;
    height: 88px;
    border: 1px solid var(--el-border-color-light);
    border-radius: 8px;
  }

  .approve-form {
    margin-top: 16px;
  }

  .approve-content {
    line-height: 1.6;
    color: var(--el-text-color-primary);
    white-space: pre-wrap;
  }
}
</style>
