<!-- 评论管理 -->
<template>
  <div class="table-box comment-page">
    <ProTable
      ref="proTable"
      :key="isDefaultTenant ? 'default-tenant' : 'normal-tenant'"
      row-key="id"
      :columns="columns"
      :request-api="requestCommentTable"
      :request-auto="false"
    >
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

    <ProDialog v-model="approveDialog.visible" title="评论审核" width="560px" destroy-on-close @closed="handleResetApproveDialog">
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

      <ProForm :model="approveDialog" :fields="approveFormFields" label-position="top" class="approve-form" />

      <template #footer>
        <el-button @click="handleCancelApprove">取消</el-button>
        <el-button type="danger" :loading="approveDialog.loading" @click="handleConfirmReject">不通过</el-button>
        <el-button type="success" :loading="approveDialog.loading" @click="handleConfirmApprove">通过</el-button>
      </template>
    </ProDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import ProDialog from "@/components/Dialog/ProDialog.vue";
import ProForm from "@/components/ProForm/index.vue";
import type { ProFormField } from "@/components/ProForm/interface";
import type { ColumnProps, ProTableInstance } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
import { useAuthButtons } from "@/hooks/useAuthButtons";
import { defCommentInfoService } from "@/api/shop/admin/comment_info";
import { defTenantStoreService } from "@/api/shop/admin/tenant_store";
import type { CommentInfo, PageCommentInfoRequest } from "@/rpc/shop/admin/v1/comment_info";
import { CommentStatus } from "@/rpc/shop/common/v1/enum";
import { buildPageRequest } from "@/utils/proTable";
import { navigateTo } from "@/utils/router";
import { useUserStore } from "@/stores/modules/user";
import {
  buildTenantStoreDisplayMap,
  buildTenantStoreDisplayMapFromOptions,
  DEFAULT_TENANT_CODE,
  formatTenantStoreDisplay,
  parseTenantStoreTreeValue,
  transformTenantStoreTreeOptions,
  type TenantStoreDisplayInfo
} from "@/utils/tenant";

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
const userStore = useUserStore();
const proTable = ref<ProTableInstance>();
const route = useRoute();
const router = useRouter();
const tenantStoreDisplayMap = ref(new Map<number, TenantStoreDisplayInfo>());

/** 评论列表搜索参数，兼容租户门店树筛选展示值。 */
type CommentInfoSearchParams = PageCommentInfoRequest & {
  /** 默认租户的租户门店树筛选值。 */
  tenant_store_tree_value?: string;
};

/** 工作台跳转评论列表时支持同步的查询参数。 */
const workspaceQueryKeys = ["status", "has_pending_discussion", "min_goods_score", "max_goods_score", "goods_score"] as const;

const approveDialog = reactive<ApproveDialogState>({
  visible: false,
  loading: false,
  row: undefined,
  reason: ""
});

/** 评论审核备注表单字段。 */
const approveFormFields: ProFormField[] = [
  {
    prop: "reason",
    label: "审核备注 / 不通过原因",
    component: "textarea",
    props: {
      rows: 3,
      maxlength: 200,
      showWordLimit: true,
      placeholder: "通过可选填备注；不通过请填写原因"
    }
  }
];

/** 当前登录账号是否默认租户。 */
const isDefaultTenant = computed(() => userStore.userInfo.tenant_code === DEFAULT_TENANT_CODE);

/** 当前审核弹窗图片列表。 */
const approveImageList = computed<string[]>(() => {
  const imgList = approveDialog.row?.img;
  // 兼容列表行图片字段为空或被省略的场景。
  if (!Array.isArray(imgList)) return [];
  return imgList;
});

/** 评论审核表格列配置。 */
const columns = computed<ColumnProps[]>(() => [
  {
    prop: "tenant_store_id",
    label: isDefaultTenant.value ? "租户/门店" : "门店",
    minWidth: isDefaultTenant.value ? 220 : 150,
    showOverflowTooltip: true,
    render: scope => getTenantStoreText(scope.row as CommentInfo),
    search: isDefaultTenant.value
      ? {
          el: "tree-select",
          key: "tenant_store_tree_value",
          order: 1,
          props: {
            clearable: true,
            filterable: true,
            checkStrictly: true,
            renderAfterExpand: false,
            placeholder: "请选择租户/门店",
            style: { width: "100%" }
          }
        }
      : {
          el: "select",
          key: "tenant_store_id",
          order: 1,
          props: {
            clearable: true,
            filterable: true,
            placeholder: "请选择门店",
            style: { width: "100%" }
          }
        },
    enum: isDefaultTenant.value ? requestTenantStoreTreeOptions : requestTenantStoreOptions
  },
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
        hidden: scope =>
          !BUTTONS.value["comment:status"] || (scope.row as CommentInfo).status !== CommentStatus.PENDING_REVIEW_CS,
        onClick: scope => handleApproveComment(scope.row as CommentInfo)
      },
      {
        label: "不通过",
        type: "danger",
        link: true,
        icon: CircleClose,
        hidden: scope =>
          !BUTTONS.value["comment:status"] || (scope.row as CommentInfo).status !== CommentStatus.PENDING_REVIEW_CS,
        onClick: scope => handleRejectComment(scope.row as CommentInfo)
      }
    ]
  }
]);

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
  if (rawValue === undefined || rawValue === null || rawValue === "") return undefined;
  const numberValue = Number(rawValue);
  return Number.isFinite(numberValue) && numberValue > 0 ? numberValue : undefined;
}

/**
 * 从工作台跳转 query 构建评论列表过滤参数。
 */
function buildCommentQueryParam() {
  const params: Record<string, number | boolean> = {};
  workspaceQueryKeys.forEach(key => {
    const numberValue = parsePositiveNumberQuery(route.query[key]);
    if (numberValue === undefined) return;

    // 后端该字段是 bool，工作台用 1 表示只看存在待审核讨论的评价。
    params[key] = key === "has_pending_discussion" ? true : numberValue;
  });
  return params;
}

/**
 * 清理表格中的工作台查询字段，避免路由变化或手动清空搜索后沿用旧条件。
 */
function clearWorkspaceQueryParam(params: Record<string, unknown>) {
  workspaceQueryKeys.forEach(key => {
    delete params[key];
  });
}

/**
 * 同步工作台 query 到 ProTable 搜索参数，保证初始化和路由变化都能触发表格过滤。
 */
function syncWorkspaceQuery() {
  if (!proTable.value) return;

  const queryParam = buildCommentQueryParam();
  clearWorkspaceQueryParam(proTable.value.searchParam);
  clearWorkspaceQueryParam(proTable.value.searchInitParam);
  Object.assign(proTable.value.searchParam, queryParam);
  Object.assign(proTable.value.searchInitParam, queryParam);
}

watch(
  () => [route.query, proTable.value],
  () => {
    syncWorkspaceQuery();
    if (!proTable.value) return;

    // 工作台查询条件变化后回到第一页，避免沿用旧分页导致结果为空。
    proTable.value.pageable.page_num = 1;
    proTable.value.search();
  },
  { immediate: true }
);

/** 请求评论列表，并由 ProTable 统一维护分页与搜索参数。 */
async function requestCommentTable(params: Record<string, any>) {
  const searchParams = params as CommentInfoSearchParams;
  // 默认租户按树节点解析租户或门店，普通租户直接传下拉选择的门店编号。
  const tenantStoreSelection = isDefaultTenant.value
    ? parseTenantStoreTreeValue(searchParams.tenant_store_tree_value)
    : { tenant_store_id: searchParams.tenant_store_id };
  const { tenant_store_tree_value: _tenantStoreTreeValue, tenant_id: _tenantId, tenant_store_id: _tenantStoreId, ...rawParams } = searchParams;
  const data = await defCommentInfoService.PageCommentInfo(
    buildPageRequest({
      ...rawParams,
      tenant_id: tenantStoreSelection.tenant_id,
      tenant_store_id: tenantStoreSelection.tenant_store_id
    }) as PageCommentInfoRequest
  );
  return { data: { list: data.comment_infos ?? [], total: data.total } };
}

/**
 * 请求租户门店树筛选数据。
 */
async function requestTenantStoreTreeOptions() {
  const response = await defTenantStoreService.TreeTenantStore({ keyword: "" });
  tenantStoreDisplayMap.value = buildTenantStoreDisplayMap(response.list ?? []);
  return { data: transformTenantStoreTreeOptions(response.list ?? []) };
}

/** 请求普通租户的门店下拉筛选数据。 */
async function requestTenantStoreOptions() {
  const response = await defTenantStoreService.OptionTenantStore({ keyword: "" });
  tenantStoreDisplayMap.value = buildTenantStoreDisplayMapFromOptions(response.list ?? []);
  return { data: response.list ?? [] };
}

/**
 * 读取评论列表租户门店展示文本，默认租户显示租户/门店。
 */
function getTenantStoreText(row: CommentInfo) {
  return formatTenantStoreDisplay(row.tenant_store_id, tenantStoreDisplayMap.value);
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
