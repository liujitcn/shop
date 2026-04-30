<!-- 评论讨论审核列表 -->
<template>
  <ProTable
    ref="proTable"
    row-key="id"
    :columns="columns"
    :request-api="requestDiscussionTable"
    :init-param="initParam"
    :tool-button="false"
  />
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from "vue";
import type { ColumnProps, ProTableInstance } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
import { useAuthButtons } from "@/hooks/useAuthButtons";
import { defCommentInfoService } from "@/api/admin/comment_info";
import type { CommentDiscussion, PageCommentDiscussionsRequest } from "@/rpc/admin/v1/comment_info";
import { CommentStatus } from "@/rpc/common/v1/enum";
import { buildPageRequest } from "@/utils/proTable";

/** 评论讨论组件入参。 */
interface DiscussionProps {
  /** 当前评论记录 */
  commentId: number;
  /** 默认审核状态筛选 */
  defaultStatus?: CommentStatus;
}

const props = defineProps<DiscussionProps>();
const emit = defineEmits<{
  /** 讨论审核完成后通知父级刷新详情。 */
  audited: [];
}>();

const { BUTTONS } = useAuthButtons();
const proTable = ref<ProTableInstance>();

const initParam = computed<Partial<PageCommentDiscussionsRequest>>(() => {
  const params: Partial<PageCommentDiscussionsRequest> = {
    comment_id: props.commentId,
    page_num: 1,
    page_size: 10
  };
  // 从评论列表待审标记进入时，默认只展示待审核讨论。
  if (props.defaultStatus !== undefined) {
    params.status = props.defaultStatus;
  }
  return params;
});

watch(
  () => [props.commentId, props.defaultStatus],
  () => {
    void nextTick(() => {
      if (!proTable.value) return;
      syncDefaultStatusToSearch();
      proTable.value.pageable.pageNum = 1;
      proTable.value.search();
    });
  }
);

/** 同步默认状态到搜索表单，确保从待审入口进入时搜索条件可见。 */
function syncDefaultStatusToSearch() {
  if (!proTable.value) return;
  // 没有默认状态时，清理状态搜索条件，恢复为全部讨论。
  if (props.defaultStatus === undefined) {
    delete proTable.value.searchParam.status;
    delete proTable.value.searchInitParam.status;
    return;
  }
  proTable.value.searchParam.status = props.defaultStatus;
  proTable.value.searchInitParam.status = props.defaultStatus;
}

/** 评论讨论表格列配置。 */
const columns: ColumnProps[] = [
  { prop: "user_name_snapshot", label: "用户昵称", minWidth: 120, search: { el: "input", key: "user_name" } },
  { prop: "content", label: "讨论内容", minWidth: 240, search: { el: "input" } },
  { prop: "reply_to_display_name", label: "回复对象", minWidth: 120 },
  {
    prop: "status",
    label: "审核状态",
    minWidth: 120,
    dictCode: "comment_status",
    search: { el: "select", defaultValue: props.defaultStatus }
  },
  { prop: "created_at", label: "发布时间", minWidth: 170 },
  {
    prop: "operation",
    label: "操作",
    width: 170,
    fixed: "right",
    cellType: "actions",
    actions: [
      {
        label: "通过",
        type: "success",
        link: true,
        icon: CircleCheck,
        hidden: scope =>
          !BUTTONS.value["comment:status"] || (scope.row as CommentDiscussion).status === CommentStatus.APPROVED_CS,
        onClick: scope => handleApproveDiscussion(scope.row as CommentDiscussion)
      },
      {
        label: "不通过",
        type: "danger",
        link: true,
        icon: CircleClose,
        hidden: scope =>
          !BUTTONS.value["comment:status"] || (scope.row as CommentDiscussion).status === CommentStatus.REJECTED_CS,
        onClick: scope => handleRejectDiscussion(scope.row as CommentDiscussion)
      }
    ]
  }
];

/** 请求评论讨论列表，并固定附加当前评论记录。 */
async function requestDiscussionTable(params: Record<string, any>) {
  const pageParams: Record<string, any> = {
    ...params,
    comment_id: props.commentId
  };
  const { pageNum, pageSize, ...requestParams } = buildPageRequest(pageParams);
  const data = await defCommentInfoService.PageCommentDiscussions({
    ...requestParams,
    comment_id: props.commentId,
    page_num: Number(pageNum),
    page_size: Number(pageSize)
  } as PageCommentDiscussionsRequest);
  const compatData = data as typeof data & {
    commentDiscussions?: typeof data.comment_discussions;
    list?: typeof data.comment_discussions;
  };
  return { data: { ...data, list: compatData.comment_discussions ?? compatData.commentDiscussions ?? compatData.list ?? [] } };
}

/** 刷新评论讨论表格。 */
function refreshTable() {
  proTable.value?.getTableList();
}

/** 审核通过单条评论讨论。 */
async function handleApproveDiscussion(row: CommentDiscussion) {
  try {
    await ElMessageBox.confirm(`是否确认通过该讨论？\n讨论内容：${row.content}`, "提示", {
      confirmButtonText: "确认",
      cancelButtonText: "取消",
      type: "warning"
    });
    await defCommentInfoService.SetCommentDiscussionStatus({ id: row.id, status: CommentStatus.APPROVED_CS, reason: "" });
    ElMessage.success("讨论审核通过");
    refreshTable();
    emit("audited");
  } catch {
    ElMessage.info("已取消审核");
  }
}

/** 审核不通过单条评论讨论。 */
async function handleRejectDiscussion(row: CommentDiscussion) {
  try {
    const { value } = await ElMessageBox.prompt(
      `请输入不通过原因\n讨论内容：${row.content || "暂无讨论内容"}`,
      "讨论审核不通过",
      {
        confirmButtonText: "确认",
        cancelButtonText: "取消",
        inputType: "textarea",
        inputPlaceholder: "请填写不通过原因",
        inputValidator: value => Boolean(String(value || "").trim()),
        inputErrorMessage: "请填写不通过原因"
      }
    );
    await defCommentInfoService.SetCommentDiscussionStatus({
      id: row.id,
      status: CommentStatus.REJECTED_CS,
      reason: String(value || "").trim()
    });
    ElMessage.success("讨论审核不通过");
    refreshTable();
    emit("audited");
  } catch {
    ElMessage.info("已取消审核");
  }
}
</script>
