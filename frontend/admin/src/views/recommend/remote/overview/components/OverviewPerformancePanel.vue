<template>
  <DataPanelCard title="推荐性能" primary>
    <div class="remote-chart-panel">
      <div class="remote-chart-panel__query no-card">
        <SearchForm
          :columns="queryColumns"
          :search-param="performanceQuery"
          :search-col="{ xs: 1, sm: 2, md: 3, lg: 8, xl: 8 }"
          :show-operation="false"
          :search="loadPerformance"
          :reset="loadPerformance"
        />
      </div>
      <div v-loading="performanceLoading" class="remote-chart-panel__chart">
        <ECharts :option="performanceOption" />
      </div>
    </div>
  </DataPanelCard>
</template>

<script setup lang="ts">
import dayjs from "dayjs";
import { computed, provide, reactive, ref } from "vue";
import { ElMessage } from "element-plus";
import DataPanelCard from "@/components/Card/DataPanelCard.vue";
import ECharts from "@/components/ECharts/index.vue";
import SearchForm from "@/components/SearchForm/index.vue";
import type { ECOption } from "@/components/ECharts/config";
import type { ColumnProps } from "@/components/ProTable/interface";
import { defRecommendRemoteService } from "@/api/admin/recommend_remote";
import { parseRemoteRecordList, resolveRemoteNumber, resolveRemoteValue } from "../../utils";

/** 推荐性能图表入参。 */
interface OverviewPerformancePanelProps {
  /** 远程推荐配置中的正向反馈类型。 */
  positiveFeedbackTypes: string[];
}

/** 推荐性能指标选项。 */
interface PerformanceOption {
  /** 远程指标名称，请求时透传。 */
  name: string;
  /** 页面展示标题。 */
  title: string;
  /** 图例标签。 */
  label: string;
}

/** 推荐性能查询下拉选项。 */
interface PerformanceSelectOption {
  /** 页面展示名称。 */
  label: string;
  /** 远程指标名称，请求时透传。 */
  value: string;
}

/** 推荐性能查询参数。 */
interface PerformanceQueryForm {
  /** 日期范围，第一项为开始日期，第二项为结束日期。 */
  dateRange: string[] | null;
  /** 远程指标名称。 */
  metric: string;
}

const props = defineProps<OverviewPerformancePanelProps>();

const performanceLoading = ref(false);
const performanceQuery = reactive<PerformanceQueryForm>({
  dateRange: [],
  metric: "positive_feedback_ratio"
});

const feedbackTypeLabelMap: Record<string, string> = {
  click: "点击",
  read: "浏览",
  view: "浏览",
  favorite: "收藏",
  collect: "收藏",
  cart: "加购",
  add_cart: "加购",
  order: "下单",
  order_create: "下单",
  pay: "支付",
  order_pay: "支付",
  like: "点赞"
};

/** 推荐性能指标下拉选项，name 保留远程接口可识别的原始指标名。 */
const performanceOptions = computed<PerformanceOption[]>(() => {
  const options: PerformanceOption[] = [
    {
      name: "positive_feedback_ratio",
      title: "正向反馈占比（全部）",
      label: "全部"
    }
  ];
  props.positiveFeedbackTypes.forEach(type => {
    const label = resolveFeedbackTypeLabel(type);
    options.push({
      name: `positive_feedback_ratio_${type}`,
      title: `正向反馈占比（${label}）`,
      label
    });
  });
  options.push(
    { name: "cf_ndcg", title: "协同过滤 NDCG", label: "NDCG" },
    { name: "cf_precision", title: "协同过滤准确率", label: "准确率" },
    { name: "cf_recall", title: "协同过滤召回率", label: "召回率" },
    { name: "ctr_auc", title: "点击率 AUC", label: "AUC" },
    { name: "ctr_precision", title: "点击率准确率", label: "准确率" },
    { name: "ctr_recall", title: "点击率召回率", label: "召回率" }
  );
  return options;
});

/** 推荐性能查询组件枚举映射，复用系统 SearchForm 的下拉渲染方式。 */
const queryEnumMap = computed(() => {
  const enumMap = new Map<string, PerformanceSelectOption[]>();
  enumMap.set(
    "metric",
    performanceOptions.value.map(item => ({
      label: item.title,
      value: item.name
    }))
  );
  return enumMap;
});

provide("enumMap", queryEnumMap);

/** 推荐性能查询列配置，选择日期或指标后直接刷新图表。 */
const queryColumns = computed<ColumnProps[]>(() => [
  {
    prop: "dateRange",
    label: "日期范围",
    search: {
      el: "date-picker",
      span: 2,
      props: {
        type: "daterange",
        valueFormat: "YYYY-MM-DD",
        startPlaceholder: "开始日期",
        endPlaceholder: "结束日期",
        unlinkPanels: true,
        clearable: true,
        onChange: loadPerformance
      }
    }
  },
  {
    prop: "metric",
    label: "指标",
    search: {
      el: "select",
      props: {
        placeholder: "请选择指标",
        filterable: true,
        clearable: false,
        onChange: handlePerformanceMetricChange
      }
    }
  }
]);

const performanceData = reactive({
  axis: [] as string[],
  values: [] as string[],
  label: "全部"
});

/** 推荐性能图表配置。 */
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

/** 加载推荐性能图表。 */
async function loadPerformance() {
  performanceLoading.value = true;
  try {
    const selected = performanceOptions.value.find(item => item.name === performanceQuery.metric) ?? performanceOptions.value[0];
    const { begin, end } = buildPerformanceDateRange();
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

/** 切换推荐性能指标后重新加载图表。 */
function handlePerformanceMetricChange() {
  loadPerformance();
}

/** 生成推荐性能日期范围查询参数。 */
function buildPerformanceDateRange() {
  const [beginValue, endValue] = Array.isArray(performanceQuery.dateRange) ? performanceQuery.dateRange : [];
  return {
    begin: buildPerformanceTime(beginValue, dayjs().subtract(7, "day").toISOString()),
    end: buildPerformanceTime(endValue, dayjs().toISOString())
  };
}

/** 生成推荐性能查询时间。 */
function buildPerformanceTime(value: string | undefined, fallback: string) {
  if (!value) return fallback;
  const date = dayjs(value);
  if (!date.isValid()) return fallback;
  return date.toISOString();
}

/** 格式化推荐性能横轴。 */
function formatPerformanceAxis(value: unknown, index: number) {
  const date = dayjs(String(value ?? ""));
  if (!date.isValid()) return String(index + 1);
  return date.format("MM-DD HH:mm");
}

/** 将远程反馈类型转换为页面中文名称。 */
function resolveFeedbackTypeLabel(type: string) {
  const normalized = String(type ?? "")
    .trim()
    .toLowerCase();
  if (!normalized) return "未命名反馈";
  return feedbackTypeLabelMap[normalized] ?? normalized;
}

defineExpose({
  /** 暴露给父组件，用于顶部刷新按钮统一刷新图表。 */
  refresh: loadPerformance
});
</script>

<style scoped lang="scss">
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

.remote-chart-panel__query {
  :deep(.table-search) {
    padding-top: 0 !important;
    margin-bottom: 0 !important;
  }
}
</style>
