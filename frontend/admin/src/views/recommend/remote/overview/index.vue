<template>
  <div v-loading="loading" class="remote-page remote-overview-page">
    <el-card class="remote-hero-card" shadow="never">
      <div class="remote-hero-card__content">
        <p>Gorse Dashboard</p>
        <h2>远程推荐概览</h2>
        <span>参照 Gorse 管理端展示用户、商品、反馈、分类与运行统计，数据仍实时来自远程推荐引擎。</span>
      </div>
      <div class="remote-hero-card__actions">
        <el-button type="primary" :loading="loading" @click="loadOverview">刷新概览</el-button>
      </div>
    </el-card>

    <section class="remote-metric-grid">
      <article v-for="item in metricCards" :key="item.name" class="remote-metric-card">
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

    <section class="remote-overview-page__charts">
      <DataPanelCard title="核心指标趋势" description="对应 Gorse Overview 顶部 Users、Items、Feedback 等时间序列指标。" primary>
        <ECharts :option="trendOption" />
      </DataPanelCard>
      <DataPanelCard title="商品分类分布" description="对应 Gorse Dashboard 的 Categories 统计结果。">
        <ECharts :option="categoryOption" />
      </DataPanelCard>
    </section>

    <section class="remote-overview-page__sections">
      <el-card class="remote-section-card" shadow="never">
        <template #header>
          <div class="remote-section-card__header">
            <strong>运行统计</strong>
            <span>远程推荐引擎 /api/dashboard/stats 摘要</span>
          </div>
        </template>
        <el-descriptions v-if="overviewEntries.length" :column="2" border>
          <el-descriptions-item v-for="item in overviewEntries" :key="item.name" :label="item.name">
            {{ item.text }}
          </el-descriptions-item>
        </el-descriptions>
        <el-empty v-else description="暂无运行统计" />
      </el-card>

      <el-card class="remote-section-card" shadow="never">
        <template #header>
          <div class="remote-section-card__header">
            <strong>分类数据</strong>
            <span>远程推荐商品分类列表</span>
          </div>
        </template>
        <el-table :data="categoryRows" border>
          <el-table-column prop="name" label="分类名称" min-width="180" />
          <el-table-column prop="count" label="数量/指标值" min-width="140" align="right" />
        </el-table>
      </el-card>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { ElMessage } from "element-plus";
import DataPanelCard from "@/components/Card/DataPanelCard.vue";
import ECharts from "@/components/ECharts/index.vue";
import type { ECOption } from "@/components/ECharts/config";
import { defRecommendRemoteService } from "@/api/admin/recommend_remote";
import {
  buildRemoteTimeseriesMetric,
  formatRemoteCell,
  formatRemoteNumber,
  isRemoteRecord,
  parseRemoteCategories,
  parseRemoteJson,
  remoteOverviewMetrics,
  type RemoteCategoryRow,
  type RemoteRecord,
  type RemoteTimeseriesMetric
} from "../utils";

/** 运行统计描述项。 */
interface OverviewEntry {
  /** 统计字段名称。 */
  name: string;
  /** 统计字段展示值。 */
  text: string;
}

defineOptions({
  name: "RecommendRemoteOverview"
});

const loading = ref(false);
const overviewData = ref<RemoteRecord>({});
const categoryRows = ref<RemoteCategoryRow[]>([]);
const metricCards = ref<RemoteTimeseriesMetric[]>(remoteOverviewMetrics.map(item => buildRemoteTimeseriesMetric(item, "[]")));

/** 运行统计条目。 */
const overviewEntries = computed<OverviewEntry[]>(() => {
  return Object.entries(overviewData.value).map(([name, value]) => ({
    name,
    text: formatRemoteCell(value)
  }));
});

/** 核心时间序列折线图配置。 */
const trendOption = computed<ECOption>(() => ({
  color: ["#2d6cdf", "#15a87b", "#f08c2e", "#d9485f", "#7c3aed"],
  tooltip: {
    trigger: "axis"
  },
  legend: {
    bottom: 0,
    textStyle: {
      color: "#6d7b8f"
    }
  },
  grid: {
    top: 24,
    left: 24,
    right: 24,
    bottom: 48,
    containLabel: true
  },
  xAxis: {
    type: "category",
    data: metricCards.value[0]?.axis ?? [],
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
  series: metricCards.value.map(item => ({
    name: item.label,
    type: "line",
    smooth: true,
    showSymbol: false,
    data: item.values
  }))
}));

/** 分类分布饼图配置。 */
const categoryOption = computed<ECOption>(() => ({
  color: ["#2d6cdf", "#15a87b", "#f08c2e", "#d9485f", "#7c3aed", "#0ea5e9", "#84cc16", "#ef4444"],
  tooltip: {
    trigger: "item",
    formatter: "{b}<br/>{c} ({d}%)"
  },
  legend: {
    bottom: 0,
    left: "center",
    textStyle: {
      color: "#6d7b8f"
    }
  },
  series: [
    {
      type: "pie",
      radius: ["36%", "72%"],
      center: ["50%", "42%"],
      itemStyle: {
        borderRadius: 8
      },
      label: {
        color: "#4f5d73"
      },
      data: categoryRows.value.map(item => ({
        name: item.name,
        value: Number(item.count) || 1
      }))
    }
  ]
}));

/** 加载推荐概览、分类和 Gorse 时间序列。 */
async function loadOverview() {
  loading.value = true;
  try {
    const [overview, categories, ...timeseriesList] = await Promise.all([
      defRecommendRemoteService.GetRecommendRemoteOverview({}),
      defRecommendRemoteService.GetRecommendRemoteCategories({}),
      ...remoteOverviewMetrics.map(item => defRecommendRemoteService.GetRecommendRemoteTimeseries({ name: item.name }))
    ]);
    const rawOverview = parseRemoteJson(overview.json);
    overviewData.value = isRemoteRecord(rawOverview) ? rawOverview : {};
    categoryRows.value = parseRemoteCategories(categories.json);
    metricCards.value = remoteOverviewMetrics.map((item, index) =>
      buildRemoteTimeseriesMetric(item, timeseriesList[index]?.json ?? "[]")
    );
  } catch (error) {
    ElMessage.error("加载推荐概览失败");
    throw error;
  } finally {
    loading.value = false;
  }
}

onMounted(() => {
  loadOverview();
});
</script>

<style scoped lang="scss">
.remote-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.remote-hero-card {
  border-color: var(--admin-page-card-border);
  background: radial-gradient(circle at top right, var(--el-color-primary-light-9), transparent 38%), var(--admin-page-card-bg);
  box-shadow: var(--admin-page-shadow);

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

  &__actions {
    flex-shrink: 0;
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
  background: var(--admin-page-card-bg);
  box-shadow: var(--admin-page-shadow);

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

.remote-overview-page {
  &__charts {
    display: grid;
    grid-template-columns: minmax(0, 1.3fr) minmax(0, 1fr);
    gap: 16px;
  }

  &__sections {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 16px;
  }
}

.remote-section-card {
  border-color: var(--admin-page-card-border);
  background: var(--admin-page-card-bg);
  box-shadow: var(--admin-page-shadow);

  &__header {
    display: flex;
    gap: 8px;
    align-items: baseline;
    justify-content: space-between;
  }

  &__header strong {
    color: var(--admin-page-text-primary);
  }

  &__header span {
    color: var(--admin-page-text-secondary);
    font-size: 13px;
  }
}

@media (max-width: 1200px) {
  .remote-metric-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }

  .remote-overview-page__charts,
  .remote-overview-page__sections {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 700px) {
  .remote-hero-card :deep(.el-card__body) {
    align-items: flex-start;
    flex-direction: column;
  }

  .remote-metric-grid {
    grid-template-columns: 1fr;
  }
}
</style>
