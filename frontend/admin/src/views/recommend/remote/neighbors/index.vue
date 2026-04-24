<template>
  <div class="remote-page remote-neighbors-page">
    <el-card class="remote-hero-card" shadow="never">
      <div class="remote-hero-card__content">
        <p>Gorse API</p>
        <h2>相似内容</h2>
        <span>查询商品相似、用户相似以及命名 Item/User to Item 推荐器结果。</span>
      </div>
      <div class="remote-hero-card__actions">
        <el-button :loading="loading" @click="loadNeighbors">刷新</el-button>
      </div>
    </el-card>

    <el-card class="remote-toolbar-card" shadow="never">
      <el-form :inline="true" :model="query" @submit.prevent>
        <el-form-item label="相似类型">
          <el-select v-model="query.type" filterable placeholder="请选择相似类型" @change="handleTypeChange">
            <el-option v-for="item in neighborTypeOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </el-form-item>
        <el-form-item :label="idLabel">
          <el-input v-model.trim="query.id" clearable :placeholder="idPlaceholder" @keyup.enter="loadNeighbors" />
        </el-form-item>
        <el-form-item v-if="supportsCategory" label="分类">
          <el-select v-model="query.category" clearable filterable placeholder="全部分类">
            <el-option v-for="item in categoryOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="数量">
          <el-input-number v-model="query.n" :min="1" :max="200" :step="10" controls-position="right" />
        </el-form-item>
        <el-form-item label="偏移">
          <el-input-number v-model="query.offset" :min="0" :max="10000" :step="10" controls-position="right" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="loading" @click="loadNeighbors">查询</el-button>
          <el-button @click="resetQuery">重置</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card class="remote-section-card" shadow="never">
      <template #header>
        <div class="remote-section-card__header">
          <strong>Neighbors</strong>
          <span>当前 {{ list.length }} 条结果</span>
        </div>
      </template>
      <el-table v-loading="loading" :data="list" :row-key="getRowKey" border>
        <el-table-column label="序号" type="index" width="80" />
        <el-table-column label="编号" min-width="180">
          <template #default="{ row }">{{ getRemoteId(row) || formatRemoteCell(row) }}</template>
        </el-table-column>
        <el-table-column label="分数" min-width="120" align="right">
          <template #default="{ row }">{{ formatRemoteCell(resolveRemoteValue(row, scoreKeys)) }}</template>
        </el-table-column>
        <el-table-column label="分类" min-width="180">
          <template #default="{ row }">
            <div v-if="getCategories(row).length" class="remote-tag-list">
              <el-tag v-for="category in getCategories(row)" :key="String(category)" effect="plain" type="info">
                {{ formatRemoteCell(category) }}
              </el-tag>
            </div>
            <span v-else>--</span>
          </template>
        </el-table-column>
        <el-table-column label="原始记录" min-width="320">
          <template #default="{ row }">
            <span class="remote-mono-text">{{ foldRemoteValue(row) }}</span>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <JsonPanel
      title="原始响应"
      description="远程相似内容接口返回的原始 JSON。"
      :json="rawJson"
      :loading="loading"
      @refresh="loadNeighbors"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { useRoute } from "vue-router";
import { ElMessage } from "element-plus";
import JsonPanel from "../components/JsonPanel.vue";
import { defRecommendRemoteService } from "@/api/admin/recommend_remote";
import {
  foldRemoteValue,
  formatRemoteCell,
  formatRemoteJson,
  parseRemoteCategories,
  parseRemoteRecordList,
  resolveRemoteArray,
  resolveRemoteId,
  resolveRemoteValue,
  type RemoteRecord
} from "../utils";

defineOptions({ name: "RecommendRemoteNeighbors" });

interface SelectOption {
  label: string;
  value: string;
}

const idKeys = ["ItemId", "itemId", "item_id", "UserId", "userId", "user_id", "Id", "id"];
const scoreKeys = ["Score", "score", "Value", "value"];
const categoryKeys = ["Categories", "categories", "Category", "category"];
const neighborTypeOptions = ref<SelectOption[]>([
  { label: "商品相似", value: "item" },
  { label: "用户相似", value: "user" }
]);
const categoryOptions = ref<SelectOption[]>([{ label: "全部分类", value: "" }]);
const query = reactive({
  type: "item",
  id: "",
  category: "",
  n: 20,
  offset: 0
});
const route = useRoute();
const loading = ref(false);
const list = ref<RemoteRecord[]>([]);
const rawJson = ref("[]");

const isUserType = computed(() => query.type === "user" || query.type.startsWith("user-to-user/"));
const supportsCategory = computed(() => !isUserType.value);
const idLabel = computed(() => (isUserType.value ? "用户编号" : "商品编号"));
const idPlaceholder = computed(() => (isUserType.value ? "输入用户编号" : "输入商品编号"));

/** 加载远程配置和分类选项。 */
async function loadOptions() {
  await Promise.allSettled([loadConfigOptions(), loadCategoryOptions()]);
}

/** 从远程配置中提取命名相似推荐器。 */
async function loadConfigOptions() {
  const data = await defRecommendRemoteService.GetRecommendRemoteConfig({});
  const config = JSON.parse(data.json || "{}");
  const recommend = config.recommend ?? config.Recommend ?? {};
  const itemToItem = Array.isArray(recommend["item-to-item"]) ? recommend["item-to-item"] : [];
  const userToUser = Array.isArray(recommend["user-to-user"]) ? recommend["user-to-user"] : [];
  const namedOptions: SelectOption[] = [];
  itemToItem.forEach((item: RemoteRecord) => {
    const name = String(resolveRemoteValue(item, ["name", "Name"]) ?? "");
    if (name) namedOptions.push({ label: `Item to Item / ${name}`, value: `item-to-item/${name}` });
  });
  userToUser.forEach((item: RemoteRecord) => {
    const name = String(resolveRemoteValue(item, ["name", "Name"]) ?? "");
    if (name) namedOptions.push({ label: `User to User / ${name}`, value: `user-to-user/${name}` });
  });
  neighborTypeOptions.value = neighborTypeOptions.value.concat(namedOptions);
}

/** 加载远程分类选项。 */
async function loadCategoryOptions() {
  const data = await defRecommendRemoteService.GetRecommendRemoteCategories({});
  categoryOptions.value = [{ label: "全部分类", value: "" }].concat(
    parseRemoteCategories(data.json).map(item => ({ label: item.name, value: item.name }))
  );
}

/** 查询远程相似内容。 */
async function loadNeighbors() {
  if (!query.id) {
    ElMessage.warning(`请先输入${idLabel.value}`);
    return;
  }
  loading.value = true;
  try {
    const data = await defRecommendRemoteService.GetRecommendRemoteNeighbors({ ...query });
    rawJson.value = formatRemoteJson(data.json || "[]");
    list.value = parseRemoteRecordList(data.json, ["Items", "items", "Results", "results", "Neighbors", "neighbors"]);
  } catch (error) {
    ElMessage.error("查询远程相似内容失败");
    throw error;
  } finally {
    loading.value = false;
  }
}

/** 相似类型变化时清理不适用条件。 */
function handleTypeChange() {
  if (!supportsCategory.value) query.category = "";
}

/** 重置查询条件。 */
function resetQuery() {
  query.type = "item";
  query.id = "";
  query.category = "";
  query.n = 20;
  query.offset = 0;
}

/** 获取结果行键。 */
function getRowKey(row: RemoteRecord, index: number) {
  return `${getRemoteId(row) || index}-${index}`;
}

/** 获取远程编号。 */
function getRemoteId(row: RemoteRecord) {
  return resolveRemoteId(row, idKeys);
}

/** 获取分类集合。 */
function getCategories(row: RemoteRecord) {
  return resolveRemoteArray(row, categoryKeys);
}

onMounted(async () => {
  applyRouteQuery();
  await loadOptions();
});

/** 应用其他页面跳转携带的查询参数。 */
function applyRouteQuery() {
  const type = route.query.type;
  const id = route.query.id;
  if (typeof type === "string" && type) query.type = type;
  if (typeof id === "string" && id) query.id = id;
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

.remote-tag-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.remote-mono-text {
  font-family: var(--admin-page-font-mono);
  word-break: break-all;
}
</style>
