<template>
  <div class="table-box gorse-detail-page">
    <ProTable
      ref="proTable"
      row-key="user_id"
      :data="similarList"
      :columns="columns"
      :pagination="false"
      :tool-button="['refresh', 'setting', 'search']"
      @refresh="handleRefresh"
      @search="handleSearch"
      @reset="handleReset"
    >
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
    </ProTable>
  </div>
</template>

<script setup lang="ts">
import { ChatLineRound, Star } from "@element-plus/icons-vue";
import { computed, onMounted, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { ElMessage } from "element-plus";
import ProTable from "@/components/ProTable/index.vue";
import type { ColumnProps, ProTableInstance } from "@/components/ProTable/interface";
import DictLabel from "@/components/Dict/DictLabel.vue";
import { defBaseDeptService } from "@/api/admin/base_dept";
import { defBaseRoleService } from "@/api/admin/base_role";
import { defRecommendGorseService } from "@/api/admin/recommend_gorse";
import { useRecommendGorseStore } from "@/stores/modules/recommendGorse";
import { navigateTo } from "@/utils/router";
import type { UserResponse } from "@/rpc/admin/v1/recommend_gorse";
import type { TreeOptionResponse_Option } from "@/rpc/common/v1/common";

const route = useRoute();
const router = useRouter();
const userId = computed(() => String(route.params.userId ?? ""));
const recommendGorseStore = useRecommendGorseStore();
const proTable = ref<ProTableInstance>();
const similarList = ref<UserResponse[]>([]);
const deptNameMap = ref<Record<string, string>>({});
const roleNameMap = ref<Record<string, string>>({});
const similarPageInitialized = ref(false);

/** 用户相似推荐器下拉项。 */
const recommenderOptions = computed(() => {
  const options = recommendGorseStore.userToUserRecommenderOptions;
  if (options.length) return options;
  return [{ label: "相似用户推荐", value: "similar_users" }];
});

watch(
  recommenderOptions,
  options => {
    const values = options.map(item => item.value);
    const currentValue = String(proTable.value?.searchParam?.recommender ?? "").trim();
    if (!values.length || !proTable.value) return;
    if (currentValue && values.includes(currentValue)) return;
    proTable.value.searchParam.recommender = values[0];
    proTable.value.searchInitParam.recommender = values[0];
    if (!similarPageInitialized.value) return;
    loadSimilarUser().catch(() => {
      ElMessage.error("加载相似用户失败");
    });
  },
  { immediate: true }
);

/** 相似用户表格列配置。 */
const columns: ColumnProps[] = [
  {
    prop: "recommender",
    label: "推荐器",
    isShow: false,
    isSetting: false,
    enum: recommenderOptions,
    search: {
      el: "select",
      order: 1,
      defaultValue: "similar_users",
      props: {
        filterable: true,
        placeholder: "请选择推荐器"
      }
    }
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
  { prop: "score", label: "相似分数", minWidth: 120, align: "right" },
  {
    prop: "operation",
    label: "操作",
    width: 220,
    fixed: "right",
    cellType: "actions",
    actions: [
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
      }
    ]
  }
];

/** 加载当前用户与相似用户列表。 */
async function loadSimilarUser() {
  if (!userId.value) {
    ElMessage.warning("缺少用户ID");
    return;
  }
  const recommender = resolveSelectedRecommender();
  const similar = await defRecommendGorseService.GetUserSimilar({
    id: userId.value,
    recommender,
    category: ""
  });
  similarList.value = similar.users ?? [];
}

/** 读取当前选中的推荐器。 */
function resolveSelectedRecommender() {
  const value = String(proTable.value?.searchParam?.recommender ?? "").trim();
  if (value) return value;
  return recommenderOptions.value[0]?.value || "similar_users";
}

/** 刷新当前相似用户列表。 */
function handleRefresh() {
  loadSimilarUser().catch(() => {
    ElMessage.error("加载相似用户失败");
  });
}

/** 响应标准搜索事件。 */
function handleSearch() {
  loadSimilarUser().catch(() => {
    ElMessage.error("加载相似用户失败");
  });
}

/** 响应标准重置事件。 */
function handleReset() {
  loadSimilarUser().catch(() => {
    ElMessage.error("加载相似用户失败");
  });
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

/** 打开当前相似用户的用户推荐页。 */
function handleOpenRecommend(row: UserResponse) {
  if (!row.user_id) {
    ElMessage.warning("当前用户缺少用户ID");
    return;
  }
  navigateTo(router, `/recommend/gorse/user/recommend/${row.user_id}`);
}

/** 打开当前相似用户的用户反馈页。 */
function handleOpenFeedback(row: UserResponse) {
  if (!row.user_id) {
    ElMessage.warning("当前用户缺少用户ID");
    return;
  }
  navigateTo(router, `/recommend/gorse/user/feedback/${row.user_id}`);
}

onMounted(() => {
  Promise.all([recommendGorseStore.loadConfig(), loadLabelNameMap()])
    .then(() => {
      similarPageInitialized.value = true;
      if (proTable.value && !String(proTable.value.searchParam.recommender || "").trim()) {
        const defaultRecommender = recommenderOptions.value[0]?.value || "similar_users";
        proTable.value.searchParam.recommender = defaultRecommender;
        proTable.value.searchInitParam.recommender = defaultRecommender;
      }
      return loadSimilarUser();
    })
    .catch(() => {
      ElMessage.error("加载相似用户失败");
    });
});
</script>

<style scoped lang="scss">
.gorse-detail-page {
  min-height: 100%;
}
</style>
