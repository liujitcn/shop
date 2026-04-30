<template>
  <el-card class="gorse-overview-chart-card" shadow="never">
    <template #header>
      <div class="gorse-overview-chart-card__header">
        <div class="gorse-overview-chart-card__title">推荐性能</div>
        <div class="gorse-overview-chart-card__filters">
          <el-date-picker
            v-model="selectedTimeRange"
            type="daterange"
            :editable="false"
            :clearable="false"
            class="gorse-overview-chart-card__range"
            range-separator="~"
            start-placeholder="开始时间"
            end-placeholder="截止时间"
            value-format="YYYY-MM-DD"
            @change="handleTimeRangeChange"
          />
          <el-select v-model="selectedMetric" class="gorse-overview-chart-card__select" filterable>
            <el-option v-for="option in performanceOptions" :key="option.value" :label="option.label" :value="option.value" />
          </el-select>
        </div>
      </div>
    </template>
    <div v-loading="chartLoading" class="gorse-overview-chart">
      <ECharts v-if="chartPoints.length" :option="chartOption" />
      <el-empty v-else description="暂无趋势数据" />
    </div>
  </el-card>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import dayjs from "dayjs";
import { ElMessage } from "element-plus";
import ECharts from "@/components/ECharts/index.vue";
import type { ECOption } from "@/components/ECharts/config";
import { defRecommendGorseService } from "@/api/admin/recommend_gorse";
import { useRecommendGorseStore } from "@/stores/modules/recommendGorse";
import type { TimeSeriesPoint } from "@/rpc/admin/v1/recommend_gorse";

const recommendGorseStore = useRecommendGorseStore();
const chartLoading = ref(false);
const isInitialized = ref(false);
const selectedMetric = ref("positive_feedback_ratio");
const selectedTimeRange = ref<[string, string]>(buildDefaultTimeRange());
const chartPoints = ref<TimeSeriesPoint[]>([]);

const performanceOptions = computed(() => recommendGorseStore.performanceOptions);
const selectedMetricLabel = computed(
  () => performanceOptions.value.find(item => item.value === selectedMetric.value)?.label ?? selectedMetric.value
);

const chartOption = computed<ECOption>(() => {
  const yAxisMax = resolveYAxisMax(selectedMetric.value, chartPoints.value);
  return {
    color: ["#2d6cdf"],
    tooltip: {
      trigger: "axis"
    },
    grid: {
      top: 30,
      left: 24,
      right: 24,
      bottom: 28,
      containLabel: true
    },
    xAxis: {
      type: "category",
      data: chartPoints.value.map(point => formatAxisTime(point.timestamp)),
      axisLabel: {
        color: "#6d7b8f"
      },
      axisLine: {
        lineStyle: {
          color: "#dbe4ee"
        }
      }
    },
    yAxis: {
      type: "value",
      min: 0,
      max: yAxisMax,
      axisLabel: {
        color: "#6d7b8f",
        formatter: value => formatYAxisLabel(Number(value), selectedMetric.value)
      },
      splitLine: {
        lineStyle: {
          color: "#edf2f7"
        }
      }
    },
    series: [
      {
        name: selectedMetricLabel.value,
        type: "line",
        smooth: true,
        showSymbol: false,
        areaStyle: {
          opacity: 0.12
        },
        data: chartPoints.value.map(point => point.value)
      }
    ]
  };
});

watch(selectedMetric, value => {
  // 初始化完成后，用户切换指标才立即重新拉取趋势数据。
  if (!isInitialized.value) return;
  loadTimeSeries(value).catch(() => {
    ElMessage.error("加载推荐性能趋势失败");
  });
});

/** 初始化推荐性能组件所需配置和趋势数据。 */
async function loadPerformanceData() {
  try {
    await recommendGorseStore.loadConfig();
    normalizeSelectedMetric();
    await loadTimeSeries(selectedMetric.value);
  } finally {
    isInitialized.value = true;
  }
}

/** 根据当前下拉指标加载趋势数据。 */
async function loadTimeSeries(name: string) {
  chartLoading.value = true;
  try {
    const [beginValue, endValue] = selectedTimeRange.value;
    const data = await defRecommendGorseService.GetTimeSeries({
      name,
      begin: dayjs(beginValue).startOf("day").toISOString(),
      end: dayjs(endValue).endOf("day").toISOString()
    });
    chartPoints.value = normalizeTimeSeriesPoints(data);
  } finally {
    chartLoading.value = false;
  }
}

/** 切换时间区间后重新加载当前推荐性能指标。 */
function handleTimeRangeChange() {
  loadTimeSeries(selectedMetric.value).catch(() => {
    ElMessage.error("加载推荐性能趋势失败");
  });
}

/** 校正当前推荐性能指标，确保选中值来自配置下拉。 */
function normalizeSelectedMetric() {
  const options = performanceOptions.value;
  // 配置加载后如果当前指标不在 store 下拉中，则回退到第一项可用指标。
  if (options.length && !options.some(option => option.value === selectedMetric.value)) selectedMetric.value = options[0].value;
}

/** 格式化趋势图横轴时间。 */
function formatAxisTime(value: string) {
  if (!value) return "--";
  return dayjs(value).format("MM-DD HH:mm");
}

/** 根据指标类型和当前数据计算 Y 轴上限，避免 ECharts 为零值数据生成负向刻度。 */
function resolveYAxisMax(metricName: string, points: TimeSeriesPoint[]) {
  const values = points.map(point => Number(point.value || 0)).filter(Number.isFinite);
  const maxValue = Math.max(0, ...values);
  // 推荐性能指标都是 0 到 1 的比率或模型评估值，按当前数据动态收敛上限但不超过 1。
  if (isRatioMetric(metricName)) {
    // 全零时给出 1 作为上限，保证坐标轴从 0 正向展开。
    if (maxValue <= 0) return 1;
    return Math.min(1, ceilNiceAxisMax(maxValue));
  }

  // 数量类指标保留原始仪表盘的非负坐标语义，并按数据最大值生成友好刻度。
  if (maxValue <= 0) return 1;
  return ceilNiceAxisMax(maxValue);
}

/** 判断当前指标是否属于推荐性能比率类指标。 */
function isRatioMetric(metricName: string) {
  return metricName.startsWith("positive_feedback_ratio") || metricName.startsWith("cf_") || metricName.startsWith("ctr_");
}

/** 按原始仪表盘思路把当前最大值抬升到友好刻度。 */
function ceilNiceAxisMax(value: number) {
  const normalizedValue = Number(value.toPrecision(6));
  // 异常或非正数上限统一兜底为 1，防止坐标轴反向展开。
  if (normalizedValue <= 0) return 1;

  const exponent = Math.floor(Math.log10(normalizedValue));
  const base = Math.pow(10, exponent);
  const normalized = normalizedValue / base;
  const step = [1, 2, 5, 10].find(item => normalized <= item) ?? 10;
  return step * base;
}

/** 根据指标类型格式化 Y 轴刻度。 */
function formatYAxisLabel(value: number, metricName: string) {
  // 比率类指标保留小数，便于区分 0.1、0.01 这类小数值。
  if (isRatioMetric(metricName)) return formatDecimalAxisValue(value);
  return formatCountAxisValue(value);
}

/** 格式化比率类刻度，去掉多余的末尾 0。 */
function formatDecimalAxisValue(value: number) {
  return value.toFixed(4).replace(/\.?0+$/, "") || "0";
}

/** 格式化数量类刻度，超过千时沿用原始仪表盘的 K 表达。 */
function formatCountAxisValue(value: number) {
  // 原始仪表盘超过 999 时用 K 缩写展示数量级。
  if (value > 999) return `${(value / 1000).toFixed(1).replace(/\.0$/, "")}K`;
  return formatDecimalAxisValue(value);
}

/** 兼容后台代理 points 包装和 Gorse 原生数组两种时间序列返回结构。 */
function normalizeTimeSeriesPoints(response: unknown): TimeSeriesPoint[] {
  const responseRecord =
    typeof response === "object" && response !== null && !Array.isArray(response) ? (response as Record<string, unknown>) : {};
  const rawPoints = (
    Array.isArray(response)
      ? response
      : Array.isArray(responseRecord.Points ?? responseRecord.points)
        ? (responseRecord.Points ?? responseRecord.points)
        : []
  ) as unknown[];
  return rawPoints
    .map(point => {
      const record =
        typeof point === "object" && point !== null && !Array.isArray(point) ? (point as Record<string, unknown>) : {};
      const timestamp = String(record.timestamp ?? record.Timestamp ?? "");
      const value = Number(record.value ?? record.Value ?? 0);
      return { name: String(record.name ?? record.Name ?? ""), timestamp, value };
    })
    .filter(point => point.timestamp);
}

/** 构建推荐性能默认时间区间。 */
function buildDefaultTimeRange(): [string, string] {
  const end = dayjs();
  const begin = end.subtract(7, "day");
  return [begin.format("YYYY-MM-DD"), end.format("YYYY-MM-DD")];
}

onMounted(() => {
  loadPerformanceData().catch(() => {
    ElMessage.error("加载推荐性能趋势失败");
  });
});
</script>

<style scoped lang="scss">
.gorse-overview-chart-card {
  border-color: var(--admin-page-card-border);
  background: var(--admin-page-card-bg);
  box-shadow: var(--admin-page-shadow);

  &__header {
    display: flex;
    gap: 16px;
    align-items: center;
    justify-content: space-between;
  }

  &__title {
    color: var(--admin-page-text-primary);
    font-size: 16px;
    font-weight: 700;
    line-height: 24px;
  }

  &__filters {
    display: flex;
    flex-wrap: wrap;
    gap: 12px;
    justify-content: flex-end;
  }

  &__range {
    width: 300px;
    max-width: 100%;
  }

  &__select {
    width: 280px;
    max-width: 100%;
  }
}

.gorse-overview-chart {
  height: 360px;
}

@media (max-width: 900px) {
  .gorse-overview-chart-card__header {
    align-items: flex-start;
    flex-direction: column;
  }

  .gorse-overview-chart-card__filters {
    justify-content: flex-start;
    width: 100%;
  }
}

@media (max-width: 560px) {
  .gorse-overview-chart-card__range,
  .gorse-overview-chart-card__select {
    width: 100%;
  }
}
</style>
