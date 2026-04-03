<template>
  <article class="chart-card">
    <div class="chart-card__header">
      <div>
        <h3 class="chart-card__title">订单销量趋势</h3>
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
import type { AnalyticsBarResponse, AnalyticsTimeType } from "@/rpc/admin/analytics";

const props = defineProps<{
  timeType: AnalyticsTimeType;
}>();

const sourceData = reactive<AnalyticsBarResponse>({
  /** 图例的数据数组 */
  axisData: [],
  /** 数据内容数组 */
  seriesData: []
});

/** 订单趋势图的系列名称。 */
const seriesNames = {
  orderCount: "订单量",
  orderGrowth: "订单量增长率"
};

/** 订单趋势图表配置。 */
const option = computed<ECOption>(() => ({
  color: ["#2d6cdf", "#f08c2e"],
  tooltip: {
    trigger: "axis",
    axisPointer: {
      type: "cross"
    }
  },
  legend: {
    bottom: 0,
    textStyle: {
      color: "#7f8ea3"
    },
    data: Object.values(seriesNames)
  },
  toolbox: {
    right: 8,
    feature: {
      saveAsImage: {}
    }
  },
  grid: {
    top: 36,
    left: 18,
    right: 18,
    bottom: 48,
    containLabel: true
  },
  xAxis: {
    type: "category",
    data: sourceData.axisData,
    axisLabel: {
      color: "#7f8ea3"
    },
    axisLine: {
      lineStyle: {
        color: "#d9e2ef"
      }
    }
  },
  yAxis: [
    {
      type: "value",
      name: "订单量",
      axisLabel: {
        color: "#7f8ea3"
      },
      splitLine: {
        lineStyle: {
          color: "#eef2f8"
        }
      }
    },
    {
      type: "value",
      name: "增长率",
      axisLabel: {
        color: "#7f8ea3",
        formatter: "{value}%"
      }
    }
  ],
  series: [
    {
      name: seriesNames.orderCount,
      type: "bar",
      barMaxWidth: 18,
      data: sourceData.seriesData[0]?.value ?? [],
      itemStyle: {
        borderRadius: [8, 8, 0, 0]
      }
    },
    {
      name: seriesNames.orderGrowth,
      type: "line",
      yAxisIndex: 1,
      smooth: true,
      data: sourceData.seriesData[1]?.value ?? []
    }
  ]
}));

/**
 * 根据时间维度加载订单趋势图数据。
 */
async function loadChartData(timeType: AnalyticsTimeType) {
  const data = await defAnalyticsService.AnalyticsBarOrder({ timeType });
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
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.chart-card__title {
  margin: 0;
  font-size: 16px;
  color: #1f2937;
}
</style>
