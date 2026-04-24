<template>
  <div class="remote-page remote-users-page">
    <el-card class="remote-hero-card" shadow="never">
      <div class="remote-hero-card__content">
        <p>Gorse Dashboard</p>
        <h2>推荐用户</h2>
        <span>按 Gorse Users 页面展示用户编号、标签、描述、活跃时间和更新时间。</span>
      </div>
      <div class="remote-hero-card__actions">
        <el-button :loading="loading" @click="loadUsers(false)">刷新</el-button>
      </div>
    </el-card>

    <el-card class="remote-toolbar-card" shadow="never">
      <el-form :inline="true" :model="query" @submit.prevent>
        <el-form-item label="用户编号">
          <el-input v-model.trim="query.id" clearable placeholder="输入完整用户编号查询" @keyup.enter="loadUsers(true)" />
        </el-form-item>
        <el-form-item label="返回数量">
          <el-input-number v-model="query.n" :min="1" :max="200" :step="10" controls-position="right" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="loading" @click="loadUsers(true)">查询</el-button>
          <el-button :disabled="!hasPreviousPage" :loading="loading" @click="loadPreviousPage">上一页</el-button>
          <el-button :disabled="!nextCursor" :loading="loading" @click="loadNextPage">下一页</el-button>
          <el-button @click="resetQuery">重置</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card class="remote-section-card" shadow="never">
      <template #header>
        <div class="remote-section-card__header">
          <strong>Users</strong>
          <span>当前页 {{ list.length }} 条</span>
        </div>
      </template>

      <el-table v-loading="loading" :data="list" :row-key="getUserId" border>
        <el-table-column label="用户编号" min-width="180">
          <template #default="{ row }">{{ getUserId(row) || "--" }}</template>
        </el-table-column>
        <el-table-column label="标签" min-width="240">
          <template #default="{ row }">
            <div v-if="getUserLabels(row).length" class="remote-tag-list">
              <el-tag v-for="label in getUserLabels(row)" :key="String(label)" effect="plain">{{
                formatRemoteCell(label)
              }}</el-tag>
            </div>
            <span v-else>--</span>
          </template>
        </el-table-column>
        <el-table-column label="描述" min-width="180">
          <template #default="{ row }">{{
            formatRemoteCell(resolveRemoteValue(row, ["Comment", "comment", "Description", "description"]))
          }}</template>
        </el-table-column>
        <el-table-column label="最后活跃" min-width="180">
          <template #default="{ row }">{{
            formatRemoteDateTime(resolveRemoteValue(row, ["LastActiveTime", "lastActiveTime", "last_active_time"]))
          }}</template>
        </el-table-column>
        <el-table-column label="最后更新" min-width="180">
          <template #default="{ row }">{{
            formatRemoteDateTime(resolveRemoteValue(row, ["LastUpdateTime", "lastUpdateTime", "last_update_time"]))
          }}</template>
        </el-table-column>
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openDetail(row)">详情</el-button>
            <el-button link type="danger" @click="deleteUser(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-drawer v-model="detailVisible" title="用户详情" size="50%">
      <div v-loading="detailLoading" class="remote-detail-drawer">
        <el-descriptions :column="1" border>
          <el-descriptions-item label="用户编号">{{ getUserId(detailData) || "--" }}</el-descriptions-item>
          <el-descriptions-item label="描述">
            {{ formatRemoteCell(resolveRemoteValue(detailData, ["Comment", "comment", "Description", "description"])) }}
          </el-descriptions-item>
          <el-descriptions-item label="最后活跃">
            {{ formatRemoteDateTime(resolveRemoteValue(detailData, ["LastActiveTime", "lastActiveTime", "last_active_time"])) }}
          </el-descriptions-item>
          <el-descriptions-item label="最后更新">
            {{ formatRemoteDateTime(resolveRemoteValue(detailData, ["LastUpdateTime", "lastUpdateTime", "last_update_time"])) }}
          </el-descriptions-item>
          <el-descriptions-item label="标签">
            <div v-if="getUserLabels(detailData).length" class="remote-tag-list">
              <el-tag v-for="label in getUserLabels(detailData)" :key="String(label)" effect="plain">{{
                formatRemoteCell(label)
              }}</el-tag>
            </div>
            <span v-else>--</span>
          </el-descriptions-item>
        </el-descriptions>

        <el-card class="remote-sub-card" shadow="never">
          <template #header><strong>用户记录</strong></template>
          <pre class="remote-code-block">{{ stringifyRemoteValue(detailData) }}</pre>
        </el-card>
      </div>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { defRecommendRemoteService } from "@/api/admin/recommend_remote";
import {
  formatRemoteCell,
  formatRemoteDateTime,
  parseRemoteCursorList,
  parseRemoteRecord,
  parseRemoteJson,
  resolveRemoteArray,
  resolveRemoteId,
  resolveRemoteValue,
  stringifyRemoteValue,
  type RemoteRecord
} from "../utils";

defineOptions({
  name: "RecommendRemoteUsers"
});

const userIdKeys = ["UserId", "userId", "user_id", "Id", "id"];

const query = reactive({
  id: "",
  cursor: "",
  n: 20
});

const loading = ref(false);
const detailLoading = ref(false);
const detailVisible = ref(false);
const nextCursor = ref("");
const cursorStack = ref<string[]>([]);
const currentDetailId = ref("");
const list = ref<RemoteRecord[]>([]);
const detailData = ref<RemoteRecord>({});

/** 是否存在上一页游标。 */
const hasPreviousPage = computed(() => cursorStack.value.length > 0);

/** 读取远程用户编号。 */
function getUserId(row: RemoteRecord) {
  return resolveRemoteId(row, userIdKeys);
}

/** 读取远程用户标签。 */
function getUserLabels(row: RemoteRecord) {
  return resolveRemoteArray(row, ["Labels", "labels"]);
}

/** 加载远程推荐用户列表或单个用户。 */
async function loadUsers(resetCursor = false) {
  if (resetCursor) {
    query.cursor = "";
    cursorStack.value = [];
  }
  loading.value = true;
  try {
    // 输入用户编号时直接查询单个用户，保持与 Gorse Users 搜索一致。
    if (query.id) {
      const data = await defRecommendRemoteService.GetRecommendRemoteUser({ id: query.id });
      const record = parseRemoteJson(data.json);
      list.value = typeof record === "object" && record !== null && !Array.isArray(record) ? [record as RemoteRecord] : [];
      nextCursor.value = "";
      return;
    }

    const data = await defRecommendRemoteService.PageRecommendRemoteUsers({
      id: "",
      cursor: query.cursor,
      n: query.n
    });
    const page = parseRemoteCursorList(data.json, ["Users", "users"]);
    list.value = page.list;
    nextCursor.value = page.cursor;
  } catch (error) {
    ElMessage.error("加载远程推荐用户失败");
    throw error;
  } finally {
    loading.value = false;
  }
}

/** 加载下一页远程用户。 */
async function loadNextPage() {
  if (!nextCursor.value) {
    ElMessage.warning("暂无下一页数据");
    return;
  }
  cursorStack.value.push(query.cursor);
  query.cursor = nextCursor.value;
  await loadUsers(false);
}

/** 加载上一页远程用户。 */
async function loadPreviousPage() {
  const previousCursor = cursorStack.value.pop();
  // 没有上一页游标时，当前已经是第一页。
  if (previousCursor === undefined) {
    ElMessage.warning("暂无上一页数据");
    return;
  }
  query.cursor = previousCursor;
  await loadUsers(false);
}

/** 重置查询条件并重新加载用户列表。 */
function resetQuery() {
  query.id = "";
  query.cursor = "";
  query.n = 20;
  cursorStack.value = [];
  loadUsers(true);
}

/** 打开远程用户详情。 */
async function openDetail(row: RemoteRecord) {
  const id = getUserId(row);
  if (!id) {
    ElMessage.warning("用户编号为空，无法查看详情");
    return;
  }
  currentDetailId.value = id;
  detailVisible.value = true;
  await reloadDetail();
}

/** 重新加载当前远程用户详情。 */
async function reloadDetail() {
  if (!currentDetailId.value) return;

  detailLoading.value = true;
  try {
    const data = await defRecommendRemoteService.GetRecommendRemoteUser({ id: currentDetailId.value });
    detailData.value = parseRemoteRecord(data.json);
  } catch (error) {
    ElMessage.error("加载用户详情失败");
    throw error;
  } finally {
    detailLoading.value = false;
  }
}

/** 删除远程推荐用户。 */
async function deleteUser(row: RemoteRecord) {
  const id = getUserId(row);
  if (!id) {
    ElMessage.warning("用户编号为空，无法删除");
    return;
  }
  await ElMessageBox.prompt(`请输入用户编号 ${id} 以确认删除`, "删除远程推荐用户", {
    confirmButtonText: "确认删除",
    cancelButtonText: "取消",
    inputPattern: new RegExp(`^${escapeRegExp(id)}$`),
    inputErrorMessage: "用户编号不匹配",
    type: "warning"
  });
  await defRecommendRemoteService.DeleteRecommendRemoteUser({ id });
  ElMessage.success("删除远程推荐用户成功");
  await loadUsers(false);
}

/** 转义确认输入使用的正则特殊字符。 */
function escapeRegExp(value: string) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

onMounted(() => {
  loadUsers(true);
});
</script>

<style scoped lang="scss">
.remote-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.remote-hero-card,
.remote-toolbar-card,
.remote-section-card,
.remote-sub-card {
  border-color: var(--admin-page-card-border);
  background: var(--admin-page-card-bg);
  box-shadow: var(--admin-page-shadow);
}

.remote-hero-card {
  background: radial-gradient(circle at top right, var(--el-color-primary-light-9), transparent 38%), var(--admin-page-card-bg);

  :deep(.el-card__body) {
    display: flex;
    gap: 16px;
    align-items: center;
    justify-content: space-between;
  }

  &__content p {
    margin: 0 0 6px;
    color: var(--el-color-primary);
    font-weight: 600;
  }

  &__content h2 {
    margin: 0 0 8px;
    color: var(--admin-page-text-primary);
    font-size: 26px;
  }

  &__content span {
    color: var(--admin-page-text-secondary);
  }
}

.remote-toolbar-card :deep(.el-form-item) {
  margin-bottom: 0;
}

.remote-section-card__header {
  display: flex;
  gap: 8px;
  align-items: baseline;
  justify-content: space-between;

  strong {
    color: var(--admin-page-text-primary);
  }

  span {
    color: var(--admin-page-text-secondary);
    font-size: 13px;
  }
}

.remote-tag-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.remote-detail-drawer {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.remote-code-block {
  max-height: 360px;
  padding: 14px;
  overflow: auto;
  color: var(--admin-page-text-primary);
  font-size: 13px;
  line-height: 1.7;
  white-space: pre-wrap;
  word-break: break-word;
  background: var(--el-fill-color-lighter);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 10px;
}

@media (max-width: 900px) {
  .remote-hero-card :deep(.el-card__body) {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
