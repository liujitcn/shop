<template>
  <div class="remote-page remote-feedback-page">
    <el-card class="remote-hero-card" shadow="never">
      <div class="remote-hero-card__content">
        <p>Gorse API</p>
        <h2>反馈管理</h2>
        <span>查询、写入和删除远程推荐反馈，适用于验证行为同步与画像更新。</span>
      </div>
      <div class="remote-hero-card__actions">
        <el-button :loading="loading" @click="loadFeedback(false)">刷新</el-button>
      </div>
    </el-card>

    <el-card class="remote-toolbar-card" shadow="never">
      <el-form :inline="true" :model="query" @submit.prevent>
        <el-form-item label="反馈类型">
          <el-input v-model.trim="query.feedbackType" clearable placeholder="如 CLICK" @keyup.enter="loadFeedback(true)" />
        </el-form-item>
        <el-form-item label="用户编号">
          <el-input v-model.trim="query.userId" clearable placeholder="可选" @keyup.enter="loadFeedback(true)" />
        </el-form-item>
        <el-form-item label="商品编号">
          <el-input v-model.trim="query.itemId" clearable placeholder="可选" @keyup.enter="loadFeedback(true)" />
        </el-form-item>
        <el-form-item label="返回数量">
          <el-input-number v-model="query.n" :min="1" :max="200" :step="10" controls-position="right" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="loading" @click="loadFeedback(true)">查询</el-button>
          <el-button :disabled="!hasPreviousPage" :loading="loading" @click="loadPreviousPage">上一页</el-button>
          <el-button :disabled="!nextCursor" :loading="loading" @click="loadNextPage">下一页</el-button>
          <el-button @click="resetQuery">重置</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card class="remote-section-card" shadow="never">
      <template #header>
        <div class="remote-section-card__header">
          <strong>Feedback</strong>
          <span>当前页 {{ list.length }} 条</span>
        </div>
      </template>
      <el-table v-loading="loading" :data="list" :row-key="getRowKey" border>
        <el-table-column label="反馈类型" min-width="140">
          <template #default="{ row }">{{ getFeedbackType(row) || "--" }}</template>
        </el-table-column>
        <el-table-column label="用户编号" min-width="180">
          <template #default="{ row }">{{ getUserId(row) || "--" }}</template>
        </el-table-column>
        <el-table-column label="商品编号" min-width="180">
          <template #default="{ row }">{{ getItemId(row) || "--" }}</template>
        </el-table-column>
        <el-table-column label="时间" min-width="180">
          <template #default="{ row }">{{ formatRemoteDateTime(resolveRemoteValue(row, timeKeys)) }}</template>
        </el-table-column>
        <el-table-column label="描述" min-width="240">
          <template #default="{ row }">
            <span class="remote-mono-text">{{ foldRemoteValue(resolveRemoteValue(row, detailKeys) ?? row) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="120" fixed="right">
          <template #default="{ row }">
            <el-button link type="danger" @click="deleteFeedback(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <section class="remote-feedback-grid">
      <el-card class="remote-section-card" shadow="never">
        <template #header>
          <div class="remote-section-card__header">
            <strong>导入反馈</strong>
            <span>写入远程推荐引擎</span>
          </div>
        </template>
        <el-input v-model="importJson" type="textarea" :rows="12" placeholder="粘贴 Gorse feedback JSON 数组" />
        <div class="remote-card-actions">
          <el-button type="success" :loading="importLoading" @click="importFeedback">导入反馈</el-button>
          <el-button @click="fillImportExample">填入示例</el-button>
        </div>
      </el-card>

      <JsonPanel
        title="原始响应"
        description="当前反馈查询接口返回的原始 JSON。"
        :json="rawJson"
        :loading="loading"
        @refresh="loadFeedback(false)"
      />
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import JsonPanel from "../components/JsonPanel.vue";
import { defRecommendRemoteService } from "@/api/admin/recommend_remote";
import {
  foldRemoteValue,
  formatRemoteDateTime,
  formatRemoteJson,
  parseRemoteCursorList,
  resolveRemoteId,
  resolveRemoteValue,
  stringifyRemoteValue,
  type RemoteRecord
} from "../utils";

defineOptions({ name: "RecommendRemoteFeedback" });

const feedbackTypeKeys = ["FeedbackType", "feedbackType", "feedback_type", "Type", "type"];
const userIdKeys = ["UserId", "userId", "user_id"];
const itemIdKeys = ["ItemId", "itemId", "item_id"];
const timeKeys = ["Timestamp", "timestamp", "Time", "time", "CreatedAt", "createdAt", "created_at"];
const detailKeys = ["Comment", "comment", "Value", "value"];
const query = reactive({
  cursor: "",
  n: 20,
  feedbackType: "",
  userId: "",
  itemId: ""
});
const loading = ref(false);
const importLoading = ref(false);
const list = ref<RemoteRecord[]>([]);
const nextCursor = ref("");
const cursorStack = ref<string[]>([]);
const rawJson = ref("[]");
const importJson = ref("[]");

const hasPreviousPage = computed(() => cursorStack.value.length > 0);

/** 加载远程推荐反馈列表或详情。 */
async function loadFeedback(resetCursor: boolean) {
  if (resetCursor) {
    query.cursor = "";
    cursorStack.value = [];
  }
  loading.value = true;
  try {
    const data = await defRecommendRemoteService.PageRecommendRemoteFeedback({ ...query });
    rawJson.value = formatRemoteJson(data.json || "[]");
    const parsed = parseRemoteCursorList(data.json, ["Feedback", "feedback", "Items", "items", "List", "list"]);
    list.value = parsed.list;
    nextCursor.value = parsed.cursor;
  } catch (error) {
    ElMessage.error("加载远程推荐反馈失败");
    throw error;
  } finally {
    loading.value = false;
  }
}

/** 加载下一页反馈。 */
async function loadNextPage() {
  if (!nextCursor.value) return;
  cursorStack.value.push(query.cursor);
  query.cursor = nextCursor.value;
  await loadFeedback(false);
}

/** 加载上一页反馈。 */
async function loadPreviousPage() {
  if (!hasPreviousPage.value) return;
  query.cursor = cursorStack.value.pop() ?? "";
  await loadFeedback(false);
}

/** 重置查询条件。 */
function resetQuery() {
  query.cursor = "";
  query.n = 20;
  query.feedbackType = "";
  query.userId = "";
  query.itemId = "";
  nextCursor.value = "";
  cursorStack.value = [];
}

/** 导入远程推荐反馈。 */
async function importFeedback() {
  const body = importJson.value.trim();
  if (!body) {
    ElMessage.warning("请先填写反馈 JSON");
    return;
  }
  try {
    JSON.parse(body);
  } catch {
    ElMessage.error("反馈 JSON 格式不正确");
    return;
  }
  await ElMessageBox.confirm("是否确定导入反馈到远程推荐？该操作会直接写入 Gorse。", "警告", {
    confirmButtonText: "确认",
    cancelButtonText: "取消",
    type: "warning"
  });
  importLoading.value = true;
  try {
    await defRecommendRemoteService.ImportRecommendRemoteFeedback({ json: body });
    ElMessage.success("导入远程推荐反馈成功");
    await loadFeedback(true);
  } catch (error) {
    ElMessage.error("导入远程推荐反馈失败");
    throw error;
  } finally {
    importLoading.value = false;
  }
}

/** 删除单条远程推荐反馈。 */
async function deleteFeedback(row: RemoteRecord) {
  const feedbackType = getFeedbackType(row);
  const userId = getUserId(row);
  const itemId = getItemId(row);
  if (!userId || !itemId) {
    ElMessage.warning("当前反馈缺少用户编号或商品编号，无法删除");
    return;
  }
  await ElMessageBox.confirm(`是否确定删除用户 ${userId} 与商品 ${itemId} 的反馈？`, "删除远程推荐反馈", {
    confirmButtonText: "确认",
    cancelButtonText: "取消",
    type: "warning"
  });
  await defRecommendRemoteService.DeleteRecommendRemoteFeedback({ feedbackType, userId, itemId });
  ElMessage.success("删除远程推荐反馈成功");
  await loadFeedback(false);
}

/** 填入反馈导入示例。 */
function fillImportExample() {
  importJson.value = stringifyRemoteValue([
    {
      FeedbackType: "CLICK",
      UserId: "user-id",
      ItemId: "item-id",
      Timestamp: new Date().toISOString()
    }
  ]);
}

/** 获取反馈表格行键。 */
function getRowKey(row: RemoteRecord, index: number) {
  return `${getFeedbackType(row)}-${getUserId(row)}-${getItemId(row)}-${index}`;
}

/** 获取反馈类型。 */
function getFeedbackType(row: RemoteRecord) {
  return resolveRemoteId(row, feedbackTypeKeys);
}

/** 获取用户编号。 */
function getUserId(row: RemoteRecord) {
  return resolveRemoteId(row, userIdKeys);
}

/** 获取商品编号。 */
function getItemId(row: RemoteRecord) {
  return resolveRemoteId(row, itemIdKeys);
}
</script>

<style scoped lang="scss">
.remote-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.remote-hero-card,
.remote-toolbar-card,
.remote-section-card {
  border-color: var(--admin-page-card-border);
  background: var(--admin-page-card-bg);
  box-shadow: var(--admin-page-shadow);
}

.remote-hero-card {
  background: radial-gradient(circle at top right, var(--el-color-primary-light-9), transparent 38%), var(--admin-page-card-bg);

  :deep(.el-card__body) {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
  }
}

.remote-hero-card__content {
  display: flex;
  flex-direction: column;
  gap: 8px;

  p {
    margin: 0;
    color: var(--admin-page-text-secondary);
  }

  h2 {
    margin: 0;
    color: var(--admin-page-text-primary);
  }

  span {
    color: var(--admin-page-text-secondary);
  }
}

.remote-section-card__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  color: var(--admin-page-text-primary);

  span {
    color: var(--admin-page-text-secondary);
    font-size: 13px;
  }
}

.remote-feedback-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 16px;
}

.remote-card-actions {
  display: flex;
  gap: 12px;
  margin-top: 12px;
}

.remote-mono-text {
  font-family: var(--admin-page-font-mono);
  word-break: break-all;
}

@media (max-width: 1200px) {
  .remote-feedback-grid {
    grid-template-columns: 1fr;
  }
}
</style>
