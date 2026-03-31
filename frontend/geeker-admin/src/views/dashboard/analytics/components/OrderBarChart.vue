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
import { defDashboardService } from "@/api/admin/dashboard";
import type { DashboardBarResponse, DashboardTimeType } from "@/rpc/admin/dashboard";

const props = defineProps<{
  timeType: DashboardTimeType;
}>();

const sourceData = reactive<DashboardBarResponse>({
  /** 图例的数据数组 */
  axisData: [],
  /** 数据内容数组 */
  seriesData: []
});

/** 订单趋势图的系列名称，顺序需和后端返回保持一致。 */
const seriesNames = {
  orderCount: "订单量",
  saleAmount: "销售额",
  orderGrowth: "订单量增长率",
  saleGrowth: "销售额增长率"
};

/** 订单趋势图表配置。 */
const option = computed<ECOption>(() => ({
  color: ["#2d6cdf", "#15a87b", "#f08c2e", "#d9485f"],
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
      name: seriesNames.orderCount,
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
      name: seriesNames.saleAmount,
      axisLabel: {
        color: "#7f8ea3",
        formatter: "{value} 元"
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
      name: seriesNames.saleAmount,
      type: "bar",
      yAxisIndex: 1,
      barMaxWidth: 18,
      data: (sourceData.seriesData[1]?.value ?? []).map(item => item / 100),
      itemStyle: {
        borderRadius: [8, 8, 0, 0]
      }
    },
    {
      name: seriesNames.orderGrowth,
      type: "line",
      yAxisIndex: 2,
      smooth: true,
      data: (sourceData.seriesData[2]?.value ?? []).map(item => item / 100)
    },
    {
      name: seriesNames.saleGrowth,
      type: "line",
      yAxisIndex: 2,
      smooth: true,
      data: (sourceData.seriesData[3]?.value ?? []).map(item => item / 100)
    }
  ]
}));

/**
 * 根据时间维度加载订单趋势图数据。
 */
async function loadChartData(timeType: DashboardTimeType) {
  const data = await defDashboardService.DashboardBarOrder({ timeType });
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
  padding: 20px;
  border: 1px solid rgb(255 255 255 / 70%);
  border-radius: 24px;
  background: linear-gradient(180deg, rgb(255 255 255 / 96%), rgb(246 249 253 / 92%));
  box-shadow: 0 18px 36px rgb(31 45 61 / 8%);
}

.chart-card__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.chart-card__title {
  margin: 0;
  font-size: 20px;
  color: #1f2d3d;
}
</style>
