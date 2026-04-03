<template>
  <article class="chart-card">
    <div class="chart-card__header">
      <div>
        <h3 class="chart-card__title">订单状态分布</h3>
      </div>
    </div>
    <ECharts :option="option" :height="360" />
  </article>
</template>

<script setup lang="ts">
import { computed, reactive, watch } from "vue";
import ECharts from "@/components/ECharts/index.vue";
import type { ECOption } from "@/components/ECharts/config";
import { defAnalyticsService } from "@/api/admin/analytics";
import type { AnalyticsPieResponse, AnalyticsTimeType } from "@/rpc/admin/analytics";

const props = defineProps<{
  timeType: AnalyticsTimeType;
}>();

const sourceData = reactive<AnalyticsPieResponse>({
  /** 数据内容数组 */
  seriesData: []
});

/** 订单状态饼图配置。 */
const option = computed<ECOption>(() => ({
  color: ["#2d6cdf", "#15a87b", "#f08c2e", "#d9485f", "#7a5af8", "#0ea5e9", "#ef4444", "#84cc16"],
  tooltip: {
    trigger: "item",
    formatter: "{b}<br/>{c} ({d}%)"
  },
  legend: {
    bottom: 0,
    left: "center",
    textStyle: {
      color: "#7f8ea3"
    }
  },
  toolbox: {
    right: 8,
    feature: {
      saveAsImage: {}
    }
  },
  series: [
    {
      name: "订单状态",
      type: "pie",
      radius: ["34%", "72%"],
      center: ["50%", "45%"],
      roseType: "radius",
      itemStyle: {
        borderRadius: 8
      },
      label: {
        color: "#4f5d73"
      },
      data: sourceData.seriesData
    }
  ]
}));

/**
 * 根据时间维度加载订单状态分布数据。
 */
async function loadChartData(timeType: AnalyticsTimeType) {
  const data = await defAnalyticsService.AnalyticsPieOrder({ timeType });
  Object.assign(sourceData, data);
}

watch(
  () => props.timeType,
  value => {
    loadChartData(value);
  },
  { immediate: true }
);
</script>

<style scoped lang="scss">
.chart-card {
  padding: 18px;
  border: 1px solid #e5eaf1;
  border-radius: 16px;
  background: #fff;
  box-shadow: 0 8px 24px rgb(15 23 42 / 4%);
}

.chart-card__header {
  margin-bottom: 12px;
}

.chart-card__title {
  margin: 0;
  font-size: 16px;
  color: #1f2937;
}
</style>
