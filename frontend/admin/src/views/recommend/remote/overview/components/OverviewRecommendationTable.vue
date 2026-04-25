<template>
  <el-card class="remote-recommendation-card" shadow="never">
    <div class="remote-recommendation-card__header">
      <div>
        <strong>非个性化推荐</strong>
      </div>
      <span v-if="lastModified">最后更新：{{ formatRemoteOverviewDateTime(lastModified) }}</span>
    </div>

    <div class="remote-recommendation-card__query no-card">
      <SearchForm
        :columns="queryColumns"
        :search-param="queryForm"
        :search-col="{ xs: 1, sm: 2, md: 3, lg: 6, xl: 6 }"
        :show-operation="false"
        :search="refresh"
        :reset="refresh"
      />
    </div>

    <div class="remote-recommendation-card__table no-card">
      <ProTable
        ref="proTable"
        row-key="__rowKey"
        :columns="columns"
        :request-api="requestRecommendedItemsTable"
        :request-auto="false"
        :pagination="false"
        :tool-button="false"
      >
        <template #categories="{ row }">
          <div class="remote-tag-list">
            <el-tag v-for="category in row.categories" :key="String(category)" effect="plain" type="info">
              {{ category }}
            </el-tag>
          </div>
        </template>
        <template #timestamp="{ row }">
          {{ formatRemoteOverviewDateTime(row.timestamp) }}
        </template>
        <template #labels="{ row }">
          <span class="remote-mono-text">{{ foldRemoteValue(row.labels) }}</span>
        </template>
        <template #score="{ row }">
          {{ formatScore(row.score) }}
        </template>
      </ProTable>
    </div>
  </el-card>
</template>

<script setup lang="ts">
import dayjs from "dayjs";
import { computed, onMounted, provide, reactive, ref, watch } from "vue";
import { ElMessage } from "element-plus";
import ProTable from "@/components/ProTable/index.vue";
import SearchForm from "@/components/SearchForm/index.vue";
import type { ColumnProps, ProTableInstance } from "@/components/ProTable/interface";
import { defRecommendRemoteService } from "@/api/admin/recommend_remote";
import { useDictStore } from "@/stores/modules/dict";
import {
  foldRemoteValue,
  formatRemoteCell,
  resolveRemoteArray,
  resolveRemoteId,
  resolveRemoteValue,
  type RemoteRecord
} from "../../utils";

/** 非个性化推荐表格入参。 */
interface OverviewRecommendationTableProps {
  /** 远程非个性化推荐器原始值。 */
  recommenders: string[];
  /** 远程分类原始值。 */
  categories: string[];
  /** 远程推荐缓存数量。 */
  cacheSize: number;
}

/** 推荐结果查询参数。 */
interface RecommendationQueryForm {
  /** 推荐器原始值。 */
  recommender: string;
  /** 分类原始值。 */
  category: string;
}

/** 推荐器下拉选项。 */
interface RecommenderOption {
  /** 远程推荐器原始值，请求时透传。 */
  value: string;
  /** 页面展示名称。 */
  label: string;
}

/** 分类下拉选项。 */
interface CategoryOption {
  /** 远程分类原始值，请求时透传。 */
  value: string;
  /** 页面展示名称。 */
  label: string;
}

/** 推荐商品表格行。 */
interface RecommendationItemRow {
  /** 表格稳定行键。 */
  __rowKey: string;
  /** 商品编号。 */
  itemId: string;
  /** 商品分类。 */
  categories: unknown[];
  /** 更新时间。 */
  timestamp: unknown;
  /** 标签。 */
  labels: unknown;
  /** 描述。 */
  comment: string;
  /** 推荐分数。 */
  score: unknown;
}

const props = defineProps<OverviewRecommendationTableProps>();
const dictStore = useDictStore();
const proTable = ref<ProTableInstance>();
const lastModified = ref("");
const queryForm = reactive<RecommendationQueryForm>({
  recommender: "latest",
  category: ""
});

const RECOMMEND_PROVIDER_DICT_CODE = "recommend_provider";

/** 推荐器字典列表，用于把远程推荐器标识转换为当前系统中文名称。 */
const recommenderDictList = computed(() => dictStore.getDictionary(RECOMMEND_PROVIDER_DICT_CODE));

/** 非个性化推荐器下拉选项，value 保留远程接口可识别的原始值。 */
const recommenderOptions = computed<RecommenderOption[]>(() =>
  props.recommenders.map(item => ({
    value: item,
    label: resolveRecommenderLabel(item)
  }))
);

/** 推荐分类下拉选项，value 保留远程接口可识别的原始值。 */
const categoryOptions = computed<CategoryOption[]>(() => [
  { value: "", label: "全部分类" },
  ...props.categories.map(item => ({
    value: item,
    label: item
  }))
]);

/** 查询组件枚举映射，复用系统 SearchForm 的下拉渲染方式。 */
const queryEnumMap = computed(() => {
  const enumMap = new Map<string, RecommenderOption[] | CategoryOption[]>();
  enumMap.set("recommender", recommenderOptions.value);
  enumMap.set("category", categoryOptions.value);
  return enumMap;
});

provide("enumMap", queryEnumMap);

/** 非个性化推荐查询列配置，选择变更后直接驱动表格查询。 */
const queryColumns = computed<ColumnProps[]>(() => [
  {
    prop: "recommender",
    label: "推荐器",
    search: {
      el: "select",
      span: 2,
      props: {
        placeholder: "请选择推荐器",
        filterable: true,
        clearable: false,
        onChange: handleQueryChange
      }
    }
  },
  {
    prop: "category",
    label: "分类",
    search: {
      el: "select",
      span: 2,
      props: {
        placeholder: "全部分类",
        filterable: true,
        clearable: true,
        onChange: handleQueryChange
      }
    }
  }
]);

/** 非个性化推荐表格列配置，表格查询由上方筛选组件直接驱动。 */
const columns = computed<ColumnProps[]>(() => [
  { prop: "itemId", label: "商品编号", minWidth: 120 },
  { prop: "categories", label: "分类", minWidth: 160 },
  { prop: "timestamp", label: "更新时间", minWidth: 160 },
  { prop: "labels", label: "标签", minWidth: 300, align: "left" },
  { prop: "comment", label: "描述", minWidth: 180, align: "left" },
  { prop: "score", label: "分数", minWidth: 140, align: "right" }
]);

/** 推荐器列表变化时校正当前选中值，避免保留已经不存在的远程推荐器。 */
watch(
  () => props.recommenders,
  value => {
    if (!value.length) return;
    if (value.includes(queryForm.recommender)) return;
    queryForm.recommender = value[0];
  },
  { deep: true }
);

/** 分类列表变化时校正当前分类，避免继续传递已不存在的分类值。 */
watch(
  () => props.categories,
  value => {
    if (!queryForm.category) return;
    if (value.includes(queryForm.category)) return;
    queryForm.category = "";
  },
  { deep: true }
);

/** 加载系统字典，用于把远程推荐器标识展示为当前系统内的中文名称。 */
async function loadDictionaryOptions() {
  try {
    await dictStore.loadDictionaries();
  } catch {
    // 字典失败不影响远程数据查询，推荐器下拉会回退展示可读名称。
  }
}

/** 查询非个性化推荐表格数据。 */
async function requestRecommendedItemsTable() {
  try {
    const data = await defRecommendRemoteService.GetDashboardItems({
      recommender: queryForm.recommender || "latest",
      category: queryForm.category || "",
      end: props.cacheSize || 100
    });
    lastModified.value = data.lastModified;
    return {
      data: data.list.map(item => normalizeRecommendedItem((item.raw ?? item) as RemoteRecord, data.list.indexOf(item)))
    };
  } catch (error) {
    ElMessage.error("加载非个性化推荐失败");
    throw error;
  }
}

/** 查询条件变化后直接刷新表格，不再依赖搜索和重置按钮。 */
function handleQueryChange() {
  refresh();
}

/** 将远程推荐商品记录转换为 ProTable 行数据。 */
function normalizeRecommendedItem(row: RemoteRecord, index: number): RecommendationItemRow {
  const itemId = resolveRemoteId(row, ["ItemId", "itemId", "item_id", "Id", "id"]);
  return {
    __rowKey: `${itemId || "item"}-${index}`,
    itemId,
    categories: resolveRemoteArray(row, ["Categories", "categories"]),
    timestamp: resolveRemoteValue(row, ["Timestamp", "timestamp"]),
    labels: resolveRemoteValue(row, ["Labels", "labels"]),
    comment: formatRemoteCell(resolveRemoteValue(row, ["Comment", "comment"])),
    score: resolveRemoteValue(row, ["Score", "score"])
  };
}

/** 格式化远程推荐概览时间。 */
function formatRemoteOverviewDateTime(value: unknown) {
  const text = String(value ?? "");
  if (!text) return "";
  const date = dayjs(text);
  if (!date.isValid()) return text;
  return date.format("YYYY/MM/DD HH:mm");
}

/** 根据系统字典把远程推荐器标识转换为中文名称。 */
function resolveRecommenderLabel(recommender: string) {
  const dictValue = buildRecommenderDictValue(recommender);
  const matched = recommenderDictList.value.find(item => item.value === dictValue);
  return matched?.label || formatRecommenderFallback(recommender);
}

/** 构建推荐器字典值，保持与 recommend_provider 字典值一致。 */
function buildRecommenderDictValue(recommender: string) {
  const normalized = String(recommender ?? "").trim();
  if (normalized === "latest") return "remote:latest";
  if (normalized.startsWith("non-personalized/")) {
    return `remote:non_personalized.${normalized.replace("non-personalized/", "")}`;
  }
  return `remote:${normalized.replace(/\//g, ".")}`;
}

/** 推荐器未命中字典时的中文兜底名称。 */
function formatRecommenderFallback(recommender: string) {
  const normalized = String(recommender ?? "").trim();
  if (!normalized) return "未命名推荐器";
  if (normalized === "latest") return "最新商品";
  if (normalized.startsWith("non-personalized/")) {
    return `非个性化推荐（${normalized.replace("non-personalized/", "")}）`;
  }
  return normalized;
}

/** 格式化推荐分数。 */
function formatScore(value: unknown) {
  const numberValue = Number(value ?? 0);
  if (!Number.isFinite(numberValue)) return "0.00000";
  return numberValue.toFixed(5);
}

/** 刷新当前查询条件下的推荐结果。 */
async function refresh() {
  await proTable.value?.getTableList();
}

onMounted(() => {
  loadDictionaryOptions();
});

defineExpose({
  /** 暴露给父组件，用于顶部刷新按钮统一刷新表格。 */
  refresh
});
</script>

<style scoped lang="scss">
.remote-recommendation-card {
  border-color: var(--admin-page-card-border);
  border-radius: var(--admin-page-radius);
  background: var(--admin-page-card-bg);
  box-shadow: var(--admin-page-shadow);

  :deep(.el-card__body) {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }
}

.remote-recommendation-card__header {
  display: flex;
  gap: 16px;
  align-items: flex-start;
  justify-content: space-between;

  strong {
    color: var(--admin-page-text-primary);
    font-size: 16px;
  }

  span {
    margin: 0;
    color: var(--admin-page-text-secondary);
    font-size: 13px;
  }
}

.remote-recommendation-card__query {
  :deep(.table-search) {
    padding-top: 0 !important;
    margin-bottom: 0 !important;
  }
}

.remote-recommendation-card__table {
  :deep(.table-main) {
    min-height: 420px;
  }
}

.remote-tag-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.remote-mono-text {
  font-family:
    Consolas, Menlo, Monaco, "Lucida Console", "Liberation Mono", "DejaVu Sans Mono", "Bitstream Vera Sans Mono", "Courier New",
    monospace, serif;
  white-space: pre-wrap;
  word-break: break-word;
}

@media (max-width: 900px) {
  .remote-recommendation-card__header {
    flex-direction: column;
  }
}
</style>
