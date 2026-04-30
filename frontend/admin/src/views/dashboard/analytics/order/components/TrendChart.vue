<template>
  <DataPanelCard title="订单与销售趋势" primary>
    <ECharts :option="option" />
  </DataPanelCard>
</template>

<script setup lang="ts">
import { computed, reactive, watch } from "vue";
import ECharts from "@/components/ECharts/index.vue";
import type { ECOption } from "@/components/ECharts/config";
import { defOrderAnalyticsService } from "@/api/admin/order_analytics";
import DataPanelCard from "@/components/Card/DataPanelCard.vue";
import type { AnalyticsTimeType, AnalyticsTrendResponse } from "@/rpc/common/v1/analytics";

const props = defineProps<{
  timeType: AnalyticsTimeType;
}>();

const trendData = reactive<AnalyticsTrendResponse>({
  axis: [],
  series: [],
  y_axis_names: []
});

const option = computed<ECOption>(() => ({
  color: ["#f08c2e", "#d9485f", "#2d6cdf"],
  tooltip: {
    trigger: "axis",
    axisPointer: {
      type: "cross"
    }
  },
  legend: {
    bottom: 0,
    textStyle: {
      color: "#6d7b8f"
    }
  },
  grid: {
    top: 28,
    left: 20,
    right: 20,
    bottom: 44,
    containLabel: true
  },
  xAxis: {
    type: "category",
    data: trendData.axis,
    axisLabel: {
      color: "#6d7b8f"
    },
    axisLine: {
      lineStyle: {
        color: "#dbe4ee"
      }
    }
  },
  yAxis: [
    {
      type: "value",
      name: trendData.y_axis_names[0] || "订单量",
      axisLabel: {
        color: "#6d7b8f"
      },
      splitLine: {
        lineStyle: {
          color: "#edf2f7"
        }
      }
    },
    {
      type: "value",
      name: trendData.y_axis_names[1] || "销售额",
      axisLabel: {
        color: "#6d7b8f"
      }
    }
  ],
  series: trendData.series.map(item => ({
    name: item.name,
    type: item.type === 1 ? "line" : "bar",
    smooth: item.type === 1,
    yAxisIndex: item.y_axis_index,
    barMaxWidth: item.type === 0 ? 18 : undefined,
    itemStyle: item.type === 0 ? { borderRadius: [8, 8, 0, 0] } : undefined,
    data: item.data
  }))
}));

/** 加载订单趋势数据。 */
async function loadData(timeType: AnalyticsTimeType) {
  try {
    const data = await defOrderAnalyticsService.TrendOrderAnalytics({ time_type: timeType });
    Object.assign(trendData, data);
  } catch (_error) {
    // 接口异常时清空趋势数据，避免未捕获 Promise 在控制台产生阻塞性错误。
    trendData.axis = [];
    trendData.series = [];
    trendData.y_axis_names = [];
  }
}

watch(
  () => props.timeType,
  value => {
    loadData(value);
  },
  { immediate: true }
);
</script>
