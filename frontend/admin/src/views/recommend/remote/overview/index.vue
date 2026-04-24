<template>
  <div class="remote-overview-page">
    <el-card class="remote-hero-card" shadow="never">
      <div class="remote-hero-card__content">
        <p>Gorse Overview</p>
        <h2>远程推荐概览</h2>
        <span>对齐原始 Gorse Overview 的统计、推荐性能查询与非个性化推荐列表，展示风格沿用当前管理端。</span>
      </div>
      <el-button type="primary" :loading="pageLoading" @click="loadOverview">刷新</el-button>
    </el-card>

    <section v-loading="statsLoading" class="remote-metric-grid">
      <article v-for="item in statCards" :key="item.name" class="remote-metric-card">
        <div class="remote-metric-card__header">
          <span>{{ item.label }}</span>
          <el-tag :type="item.increase ? 'success' : 'danger'" effect="light" round>
            {{ item.increase ? "+" : "" }}{{ formatRemoteNumber(item.diff) }}
          </el-tag>
        </div>
        <strong>{{ formatRemoteNumber(item.current) }}</strong>
        <p>{{ item.description }}</p>
      </article>
    </section>

    <DataPanelCard
      title="Recommendation Performance"
      description="支持与原始 Gorse 页面一致的开始日期、结束日期和指标下拉查询。"
      primary
    >
      <div class="remote-chart-panel">
        <div class="remote-filter-row">
          <div class="remote-date-group">
            <el-date-picker
              v-model="performanceQuery.begin"
              type="date"
              placeholder="Start Date"
              value-format="YYYY-MM-DD"
              clearable
              @change="loadPerformance"
            />
            <el-date-picker
              v-model="performanceQuery.end"
              type="date"
              placeholder="End Date"
              value-format="YYYY-MM-DD"
              clearable
              @change="loadPerformance"
            />
          </div>
          <el-select v-model="performanceQuery.metric" class="remote-performance-select" @change="handlePerformanceMetricChange">
            <el-option v-for="item in performanceOptions" :key="item.name" :label="item.title" :value="item.name" />
          </el-select>
        </div>
        <div v-loading="performanceLoading" class="remote-chart-panel__chart">
          <ECharts :option="performanceOption" />
        </div>
      </div>
    </DataPanelCard>

    <el-card class="remote-table-card" shadow="never">
      <template #header>
        <div class="remote-table-card__header">
          <div>
            <strong>Non-personalized Recommendations</strong>
            <p>支持与原始 Gorse 页面一致的 Recommender 和 Categories 筛选。</p>
          </div>
          <span v-if="lastModified">Last Update: {{ formatGorseDateTime(lastModified) }}</span>
        </div>
      </template>

      <div class="remote-filter-row remote-filter-row--table">
        <div class="remote-input-group">
          <span>Recommender</span>
          <el-select v-model="recommendQuery.recommender" @change="loadRecommendedItems">
            <el-option v-for="item in recommenders" :key="item" :label="item" :value="item" />
          </el-select>
        </div>
        <div class="remote-input-group">
          <span>Categories</span>
          <el-select v-model="recommendQuery.category" clearable @change="loadRecommendedItems">
            <el-option label="" value="" />
            <el-option v-for="item in categories" :key="item" :label="item" :value="item" />
          </el-select>
        </div>
      </div>

      <el-table v-loading="recommendLoading" :data="recommendedItems" border>
        <el-table-column label="ID" min-width="100">
          <template #default="{ row }">{{ getItemId(row) }}</template>
        </el-table-column>
        <el-table-column label="Categories" min-width="150">
          <template #default="{ row }">
            <div class="remote-tag-list">
              <el-tag v-for="category in getItemCategories(row)" :key="String(category)" effect="plain" type="info">
                {{ category }}
              </el-tag>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="Timestamp" min-width="150">
          <template #default="{ row }">{{ formatGorseDateTime(resolveRemoteValue(row, ["Timestamp", "timestamp"])) }}</template>
        </el-table-column>
        <el-table-column label="Labels" min-width="300">
          <template #default="{ row }">
            <span class="remote-mono-text">{{ foldRemoteValue(resolveRemoteValue(row, ["Labels", "labels"])) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="Description" min-width="180">
          <template #default="{ row }">{{ formatRemoteCell(resolveRemoteValue(row, ["Comment", "comment"])) }}</template>
        </el-table-column>
        <el-table-column label="Score" min-width="140" align="right">
          <template #default="{ row }">{{ formatScore(resolveRemoteValue(row, ["Score", "score"])) }}</template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import dayjs from "dayjs";
import { computed, onMounted, reactive, ref } from "vue";
import { ElMessage } from "element-plus";
import DataPanelCard from "@/components/Card/DataPanelCard.vue";
import ECharts from "@/components/ECharts/index.vue";
import type { ECOption } from "@/components/ECharts/config";
import { defRecommendRemoteService } from "@/api/admin/recommend_remote";
import {
  buildRemoteTimeseriesMetric,
  foldRemoteValue,
  formatRemoteCell,
  formatRemoteNumber,
  parseRemoteCategories,
  parseRemoteJson,
  parseRemoteRecordList,
  remoteOverviewMetrics,
  resolveRemoteArray,
  resolveRemoteId,
  resolveRemoteNumber,
  resolveRemoteValue,
  type RemoteRecord,
  type RemoteTimeseriesMetric
} from "../utils";

/** 推荐性能指标选项。 */
interface PerformanceOption {
  /** 远程指标名称。 */
  name: string;
  /** 下拉展示标题。 */
  title: string;
  /** 图例标签。 */
  label: string;
}

defineOptions({
  name: "RecommendRemoteOverview"
});

const statsLoading = ref(false);
const performanceLoading = ref(false);
const recommendLoading = ref(false);
const statCards = ref<RemoteTimeseriesMetric[]>(remoteOverviewMetrics.map(item => buildRemoteTimeseriesMetric(item, "[]")));
const positiveFeedbackTypes = ref<string[]>([]);
const recommenders = ref<string[]>(["latest"]);
const categories = ref<string[]>([]);
const recommendedItems = ref<RemoteRecord[]>([]);
const lastModified = ref("");
const cacheSize = ref(100);

const performanceQuery = reactive({
  begin: "",
  end: "",
  metric: "positive_feedback_ratio"
});

const recommendQuery = reactive({
  recommender: "latest",
  category: ""
});

/** 页面任一模块加载中时，顶部刷新按钮进入加载状态。 */
const pageLoading = computed(() => statsLoading.value || performanceLoading.value || recommendLoading.value);

/** Gorse Overview 推荐性能指标下拉选项。 */
const performanceOptions = computed<PerformanceOption[]>(() => {
  const options: PerformanceOption[] = [
    {
      name: "positive_feedback_ratio",
      title: "Positive Feedback Ratio - All",
      label: "All"
    }
  ];
  positiveFeedbackTypes.value.forEach(type => {
    const label = type.charAt(0).toUpperCase() + type.slice(1);
    options.push({
      name: `positive_feedback_ratio_${type}`,
      title: `Positive Feedback Ratio - ${label}`,
      label
    });
  });
  options.push(
    { name: "cf_ndcg", title: "Collaborative Filtering - NDCG", label: "NDCG" },
    { name: "cf_precision", title: "Collaborative Filtering - Precision", label: "Precision" },
    { name: "cf_recall", title: "Collaborative Filtering - Recall", label: "Recall" },
    { name: "ctr_auc", title: "Click-Through Rate - AUC", label: "AUC" },
    { name: "ctr_precision", title: "Click-Through Rate - Precision", label: "Precision" },
    { name: "ctr_recall", title: "Click-Through Rate - Recall", label: "Recall" }
  );
  return options;
});

const performanceData = reactive({
  axis: [] as string[],
  values: [] as string[],
  label: "All"
});

/** Gorse Recommendation Performance 图表配置。 */
const performanceOption = computed<ECOption>(() => ({
  color: ["#2d6cdf"],
  tooltip: {
    trigger: "axis"
  },
  legend: {
    top: 0,
    data: [performanceData.label],
    textStyle: {
      color: "#6d7b8f"
    }
  },
  grid: {
    top: 36,
    left: 24,
    right: 20,
    bottom: 28,
    containLabel: true
  },
  xAxis: {
    type: "category",
    data: performanceData.axis,
    boundaryGap: false,
    axisLabel: {
      color: "#6d7b8f"
    }
  },
  yAxis: {
    type: "value",
    axisLabel: {
      color: "#6d7b8f"
    },
    splitLine: {
      lineStyle: {
        color: "#edf2f7"
      }
    }
  },
  series: [
    {
      name: performanceData.label,
      type: "line",
      smooth: true,
      showSymbol: false,
      areaStyle: {
        color: "rgba(45, 108, 223, 0.1)"
      },
      data: performanceData.values
    }
  ]
}));

/** 页面初始化，按 Gorse Overview 加载配置、统计、推荐性能和非个性化推荐。 */
async function loadOverview() {
  // 各区块互不阻塞，避免单个远程接口异常导致概览页整体无法打开。
  await Promise.allSettled([loadConfig(), loadCategories(), loadStats()]);
  await Promise.allSettled([loadPerformance(), loadRecommendedItems()]);
}

/** 加载 Gorse Dashboard 配置，用于反馈类型、缓存数量和非个性化推荐器。 */
async function loadConfig() {
  try {
    const data = await defRecommendRemoteService.GetRecommendRemoteConfig({});
    const config = parseRemoteJson(data.json) as RemoteRecord;
    const recommend = resolveRemoteValue(config, ["recommend"]);
    const database = resolveRemoteValue(config, ["database"]);
    if (typeof database === "object" && database !== null) {
      cacheSize.value = resolveRemoteNumber(database as RemoteRecord, ["cache_size", "cacheSize"]) || 100;
    }
    if (typeof recommend === "object" && recommend !== null) {
      const recommendRecord = recommend as RemoteRecord;
      const dataSource = resolveRemoteValue(recommendRecord, ["data_source", "dataSource"]);
      if (typeof dataSource === "object" && dataSource !== null) {
        positiveFeedbackTypes.value = resolveRemoteArray(dataSource as RemoteRecord, [
          "positive_feedback_types",
          "positiveFeedbackTypes"
        ]).map(String);
      }
      const nonPersonalized = resolveRemoteArray(recommendRecord, ["non-personalized", "nonPersonalized"]);
      recommenders.value = ["latest"].concat(
        nonPersonalized
          .filter(item => typeof item === "object" && item !== null)
          .map(item => `non-personalized/${String(resolveRemoteValue(item as RemoteRecord, ["name", "Name"]) ?? "")}`)
          .filter(item => item !== "non-personalized/")
      );
    }
  } catch (error) {
    ElMessage.error("加载推荐配置失败");
    throw error;
  }
}

/** 加载 Gorse Overview 顶部五个统计卡。 */
async function loadStats() {
  statsLoading.value = true;
  try {
    const responses = await Promise.all(
      remoteOverviewMetrics.map(item =>
        defRecommendRemoteService.GetRecommendRemoteTimeseries({ name: item.name, begin: "", end: "" })
      )
    );
    statCards.value = remoteOverviewMetrics.map((item, index) =>
      buildRemoteTimeseriesMetric(item, responses[index]?.json ?? "[]")
    );
  } catch (error) {
    ElMessage.error("加载概览统计失败");
    throw error;
  } finally {
    statsLoading.value = false;
  }
}

/** 加载 Gorse Dashboard 分类下拉。 */
async function loadCategories() {
  try {
    const data = await defRecommendRemoteService.GetRecommendRemoteCategories({});
    categories.value = parseRemoteCategories(data.json).map(item => item.name);
  } catch (error) {
    ElMessage.error("加载推荐分类失败");
    throw error;
  }
}

/** 加载 Gorse Recommendation Performance 图表。 */
async function loadPerformance() {
  performanceLoading.value = true;
  try {
    const selected = performanceOptions.value.find(item => item.name === performanceQuery.metric) ?? performanceOptions.value[0];
    const begin = buildPerformanceTime(performanceQuery.begin, dayjs().subtract(7, "day").toISOString());
    const end = buildPerformanceTime(performanceQuery.end, dayjs().toISOString());
    const data = await defRecommendRemoteService.GetRecommendRemoteTimeseries({ name: selected.name, begin, end });
    const list = parseRemoteRecordList(data.json, ["Timeseries", "timeseries", "Items", "items", "Values", "values"]);
    performanceData.axis = list.map((item, index) =>
      formatPerformanceAxis(resolveRemoteValue(item, ["Timestamp", "timestamp"]), index)
    );
    performanceData.values = list.map(item => resolveRemoteNumber(item, ["Value", "value"]).toFixed(5));
    performanceData.label = selected.label;
  } catch (error) {
    ElMessage.error("加载推荐性能失败");
    throw error;
  } finally {
    performanceLoading.value = false;
  }
}

/** 加载 Gorse Non-personalized Recommendations 表格。 */
async function loadRecommendedItems() {
  recommendLoading.value = true;
  try {
    const data = await defRecommendRemoteService.GetRecommendRemoteDashboardItems({
      recommender: recommendQuery.recommender,
      category: recommendQuery.category,
      end: cacheSize.value || 100
    });
    recommendedItems.value = parseRemoteRecordList(data.json);
    lastModified.value = data.lastModified;
  } catch (error) {
    ElMessage.error("加载非个性化推荐失败");
    throw error;
  } finally {
    recommendLoading.value = false;
  }
}

/** 切换推荐性能指标后重新加载图表。 */
function handlePerformanceMetricChange() {
  loadPerformance();
}

/** 生成推荐性能查询时间。 */
function buildPerformanceTime(value: string, fallback: string) {
  if (!value) return fallback;
  const date = dayjs(value);
  if (!date.isValid()) return fallback;
  return date.toISOString();
}

/** 格式化推荐性能横轴。 */
function formatPerformanceAxis(value: unknown, index: number) {
  const date = dayjs(String(value ?? ""));
  if (!date.isValid()) return String(index + 1);
  return date.format("MMM DD HH:mm");
}

/** 格式化 Gorse 页面时间。 */
function formatGorseDateTime(value: unknown) {
  const text = String(value ?? "");
  if (!text) return "";
  const date = dayjs(text);
  if (!date.isValid()) return text;
  return date.format("YYYY/MM/DD HH:mm");
}

/** 读取推荐商品 ID。 */
function getItemId(row: RemoteRecord) {
  return resolveRemoteId(row, ["ItemId", "itemId", "item_id", "Id", "id"]);
}

/** 读取推荐商品分类。 */
function getItemCategories(row: RemoteRecord) {
  return resolveRemoteArray(row, ["Categories", "categories"]);
}

/** 格式化 Gorse 推荐分数。 */
function formatScore(value: unknown) {
  const numberValue = Number(value ?? 0);
  if (!Number.isFinite(numberValue)) return "0.00000";
  return numberValue.toFixed(5);
}

onMounted(() => {
  loadOverview();
});
</script>

<style scoped lang="scss">
.remote-overview-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.remote-hero-card,
.remote-table-card,
.remote-metric-card {
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

.remote-metric-grid {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 16px;
}

.remote-metric-card {
  padding: 18px;
  border: 1px solid var(--admin-page-card-border);
  border-radius: 16px;

  &__header {
    display: flex;
    gap: 8px;
    align-items: center;
    justify-content: space-between;
    color: var(--admin-page-text-secondary);
    font-size: 13px;
  }

  strong {
    display: block;
    margin-top: 12px;
    color: var(--admin-page-text-primary);
    font-size: 28px;
    line-height: 1;
  }

  p {
    margin: 10px 0 0;
    color: var(--admin-page-text-secondary);
    font-size: 13px;
  }
}

.remote-chart-panel {
  display: flex;
  height: 100%;
  min-height: 360px;
  flex-direction: column;
  gap: 14px;
}

.remote-chart-panel__chart {
  flex: 1;
  min-height: 260px;
}

.remote-filter-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(280px, 0.9fr);
  gap: 16px;
  align-items: center;

  &--table {
    padding: 16px;
    background: var(--el-fill-color-lighter);
    border-bottom: 1px solid var(--el-border-color-lighter);
  }
}

.remote-date-group {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.remote-performance-select {
  width: 100%;
}

.remote-table-card__header {
  display: flex;
  gap: 16px;
  align-items: flex-start;
  justify-content: space-between;

  strong {
    color: var(--admin-page-text-primary);
    font-size: 16px;
  }

  p,
  span {
    margin: 6px 0 0;
    color: var(--admin-page-text-secondary);
    font-size: 13px;
  }
}

.remote-input-group {
  display: grid;
  grid-template-columns: 130px minmax(0, 1fr);

  > span {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-height: 32px;
    color: var(--admin-page-text-secondary);
    background: var(--admin-page-card-bg);
    border: 1px solid var(--el-border-color);
    border-right: 0;
    border-radius: 4px 0 0 4px;
  }

  :deep(.el-select__wrapper) {
    border-radius: 0 4px 4px 0;
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

@media (max-width: 1200px) {
  .remote-metric-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}

@media (max-width: 900px) {
  .remote-hero-card :deep(.el-card__body) {
    align-items: flex-start;
    flex-direction: column;
  }

  .remote-metric-grid,
  .remote-filter-row,
  .remote-date-group {
    grid-template-columns: 1fr;
  }
}
</style>
