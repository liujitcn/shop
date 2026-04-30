<template>
  <div class="table-box gorse-user-page">
    <ProTable
      ref="proTable"
      row-key="user_id"
      :data="filteredUserList"
      :columns="columns"
      :pagination="false"
      :tool-button="['refresh', 'setting', 'search']"
      @refresh="handleRefresh"
      @search="handleSearch"
      @reset="handleReset"
    >
      <template #user_id="{ row }">{{ row.user_id || "--" }}</template>
      <template #dept_id="{ row }">{{ formatUserLabelValue(row, "dept_id") }}</template>
      <template #role_id="{ row }">{{ formatUserLabelValue(row, "role_id") }}</template>
      <template #gender="{ row }">
        <DictLabel
          v-if="formatUserLabelValue(row, 'gender') !== '--'"
          :model-value="formatUserLabelValue(row, 'gender')"
          code="base_user_gender"
        />
        <span v-else>--</span>
      </template>
      <template #status="{ row }">
        <DictLabel
          v-if="formatUserLabelValue(row, 'status') !== '--'"
          :model-value="formatUserLabelValue(row, 'status')"
          code="status"
        />
        <span v-else>--</span>
      </template>
      <template #last_active_time="{ row }">{{ formatTimestamp(row.last_active_time) }}</template>
      <template #last_update_time="{ row }">{{ formatTimestamp(row.last_update_time) }}</template>
    </ProTable>

    <div class="gorse-cursor-pagination">
      <el-button :disabled="!cursorStack.length || loading" @click="handlePrevPage">上一页</el-button>
      <el-button type="primary" :disabled="!nextCursor || loading" @click="handleNextPage">下一页</el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ChatLineRound, Delete, Star, View } from "@element-plus/icons-vue";
import { onMounted, ref, watch } from "vue";
import { useRouter } from "vue-router";
import dayjs from "dayjs";
import { ElMessage, ElMessageBox } from "element-plus";
import ProTable from "@/components/ProTable/index.vue";
import type { ColumnProps, ProTableInstance } from "@/components/ProTable/interface";
import DictLabel from "@/components/Dict/DictLabel.vue";
import { defBaseDeptService } from "@/api/admin/base_dept";
import { defBaseRoleService } from "@/api/admin/base_role";
import { defRecommendGorseService } from "@/api/admin/recommend_gorse";
import { navigateTo } from "@/utils/router";
import type { UserResponse } from "@/rpc/admin/v1/recommend_gorse";
import type { TreeOptionResponse_Option } from "@/rpc/common/v1/common";

const router = useRouter();
const loading = ref(false);
const proTable = ref<ProTableInstance>();
const pageSize = ref(10);
const currentCursor = ref("");
const nextCursor = ref("");
const cursorStack = ref<string[]>([]);
const userList = ref<UserResponse[]>([]);
const filteredUserList = ref<UserResponse[]>([]);
const deptNameMap = ref<Record<string, string>>({});
const roleNameMap = ref<Record<string, string>>({});

/** 推荐用户表格列配置。 */
const columns: ColumnProps[] = [
  {
    prop: "user_id",
    label: "用户ID",
    minWidth: 160,
    showOverflowTooltip: true,
    search: { el: "input", order: 1, props: { placeholder: "请输入用户ID" } }
  },
  { prop: "comment", label: "用户名称", minWidth: 160, showOverflowTooltip: true },
  {
    label: "标签",
    align: "center",
    _children: [
      { prop: "dept_id", label: "部门", minWidth: 160, showOverflowTooltip: true },
      { prop: "role_id", label: "角色", minWidth: 160, showOverflowTooltip: true },
      { prop: "gender", label: "性别", minWidth: 100, align: "center" },
      { prop: "status", label: "状态", minWidth: 100, align: "center" }
    ]
  },
  { prop: "last_active_time", label: "最后活跃时间", minWidth: 180 },
  { prop: "last_update_time", label: "最后更新时间", minWidth: 180 },
  {
    prop: "operation",
    label: "操作",
    width: 320,
    fixed: "right",
    cellType: "actions",
    actions: [
      {
        label: "查看",
        type: "primary",
        link: true,
        icon: View,
        onClick: scope => handleOpenSimilar(scope.row as UserResponse)
      },
      {
        label: "推荐",
        type: "primary",
        link: true,
        icon: Star,
        onClick: scope => handleOpenRecommend(scope.row as UserResponse)
      },
      {
        label: "反馈",
        type: "primary",
        link: true,
        icon: ChatLineRound,
        onClick: scope => handleOpenFeedback(scope.row as UserResponse)
      },
      {
        label: "删除",
        type: "danger",
        link: true,
        icon: Delete,
        onClick: scope => handleDelete(scope.row as UserResponse)
      }
    ]
  }
];

watch(
  userList,
  () => {
    applyUserFilter();
  },
  { deep: true, immediate: true }
);

/** 加载推荐用户游标分页数据。 */
async function loadUserPage(cursor = currentCursor.value) {
  loading.value = true;
  try {
    const data = await defRecommendGorseService.PageUsers({ cursor, n: pageSize.value });
    currentCursor.value = cursor;
    nextCursor.value = data.cursor || "";
    userList.value = data.users ?? [];
  } finally {
    loading.value = false;
  }
}

/** 刷新当前游标页数据。 */
function handleRefresh() {
  loadUserPage().catch(() => {
    ElMessage.error("刷新推荐用户失败");
  });
}

/** 按当前搜索条件过滤推荐用户列表。 */
function applyUserFilter() {
  const keyword = String(proTable.value?.searchParam?.user_id ?? "").trim();
  if (!keyword) {
    filteredUserList.value = [...userList.value];
    return;
  }
  filteredUserList.value = userList.value.filter(user => String(user.user_id || "").includes(keyword));
}

/** 响应公共搜索事件，按用户ID过滤当前页数据。 */
function handleSearch() {
  applyUserFilter();
}

/** 响应公共重置事件，清空用户ID筛选结果。 */
function handleReset() {
  applyUserFilter();
}

/** 读取用户标签展示值。 */
function formatUserLabelValue(row: UserResponse, key: keyof NonNullable<UserResponse["labels"]>) {
  const value = row.labels?.[key];
  if (value === undefined || value === null || value === 0) return "--";
  // 部门与角色标签统一转换成后台基础资料名称，避免页面展示原始编号。
  if (key === "dept_id") return deptNameMap.value[String(value)] || String(value);
  if (key === "role_id") return roleNameMap.value[String(value)] || String(value);
  return value;
}

/** 加载部门与角色名称映射。 */
async function loadLabelNameMap() {
  const [deptResponse, roleResponse] = await Promise.all([
    defBaseDeptService.OptionBaseDepts({}),
    defBaseRoleService.OptionBaseRoles({})
  ]);

  const nextDeptNameMap: Record<string, string> = {};
  const deptStack: TreeOptionResponse_Option[] = [...(deptResponse.list ?? [])];
  while (deptStack.length) {
    const current = deptStack.shift();
    // 部门节点为空时直接跳过，避免异常数据影响映射。
    if (!current) continue;
    nextDeptNameMap[String(current.value)] = String(current.label ?? "").trim();
    deptStack.push(...(current.children ?? []));
  }
  deptNameMap.value = nextDeptNameMap;

  const nextRoleNameMap: Record<string, string> = {};
  for (const option of roleResponse.list ?? []) {
    nextRoleNameMap[String(option.value)] = String(option.label ?? "").trim();
  }
  roleNameMap.value = nextRoleNameMap;
}

/** 打开相似用户页面。 */
function handleOpenSimilar(row: UserResponse) {
  if (!row.user_id) {
    ElMessage.warning("当前用户缺少用户ID");
    return;
  }
  navigateTo(router, `/recommend/gorse/user/similar/${row.user_id}`);
}

/** 打开用户推荐页面。 */
function handleOpenRecommend(row: UserResponse) {
  if (!row.user_id) {
    ElMessage.warning("当前用户缺少用户ID");
    return;
  }
  navigateTo(router, `/recommend/gorse/user/recommend/${row.user_id}`);
}

/** 打开用户反馈页面。 */
function handleOpenFeedback(row: UserResponse) {
  if (!row.user_id) {
    ElMessage.warning("当前用户缺少用户ID");
    return;
  }
  navigateTo(router, `/recommend/gorse/user/feedback/${row.user_id}`);
}

/** 删除 Gorse 推荐用户。 */
function handleDelete(row: UserResponse) {
  const userText = row.comment || row.user_id;
  ElMessageBox.confirm(`是否确定删除 Gorse 推荐用户？\n用户：${userText}`, "警告", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning"
  }).then(
    () => {
      defRecommendGorseService.DeleteUser({ id: row.user_id }).then(() => {
        ElMessage.success("删除 Gorse 推荐用户成功");
        loadUserPage().catch(() => {
          ElMessage.error("刷新推荐用户失败");
        });
      });
    },
    () => {
      ElMessage.info("已取消删除 Gorse 推荐用户");
    }
  );
}

/** 跳转下一页游标数据。 */
function handleNextPage() {
  if (!nextCursor.value) return;
  cursorStack.value.push(currentCursor.value);
  loadUserPage(nextCursor.value).catch(() => {
    ElMessage.error("加载推荐用户失败");
  });
}

/** 返回上一页游标数据。 */
function handlePrevPage() {
  const previousCursor = cursorStack.value.pop();
  if (previousCursor === undefined) return;
  loadUserPage(previousCursor).catch(() => {
    ElMessage.error("加载推荐用户失败");
  });
}

/** 格式化Gorse 推荐用户时间。 */
function formatTimestamp(value: string) {
  if (!value || value.startsWith("0001-01-01")) return "--";
  return dayjs(value).format("YYYY-MM-DD HH:mm:ss");
}

onMounted(() => {
  loadLabelNameMap().catch(() => {
    ElMessage.error("加载用户标签映射失败");
  });
  loadUserPage("").catch(() => {
    ElMessage.error("加载推荐用户失败");
  });
});
</script>

<style scoped lang="scss">
.gorse-cursor-pagination {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 16px 0 0;
}
</style>
