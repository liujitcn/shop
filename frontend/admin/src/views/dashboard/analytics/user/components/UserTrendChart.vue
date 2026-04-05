<template>
  <DataPanelCard title="用户增长趋势" description="注册用户与下单用户按时间维度对比。" primary>
    <ECharts :option="option" />
  </DataPanelCard>
</template>

<script setup lang="ts">
import { computed, reactive, watch } from "vue";
import ECharts from "@/components/ECharts/index.vue";
import type { ECOption } from "@/components/ECharts/config";
import { defUserAnalyticsService } from "@/api/admin/user_analytics";
import DataPanelCard from "@/components/Card/DataPanelCard.vue";
import type { AnalyticsTimeType, AnalyticsTrendResponse } from "@/rpc/common/analytics";

const props = defineProps<{
  timeType: AnalyticsTimeType;
}>();

const trendData = reactive<AnalyticsTrendResponse>({
  axis: [],
  series: [],
  yAxisNames: []
});

const option = computed<ECOption>(() => ({
  color: ["#2d6cdf", "#15a87b", "#f08c2e"],
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
      name: trendData.yAxisNames[0] || "人数",
      axisLabel: {
        color: "#6d7b8f"
      },
      splitLine: {
        lineStyle: {
          color: "#edf2f7"
        }
      }
    }
  ],
  series: trendData.series.map(item => ({
    name: item.name,
    type: item.type === 1 ? "line" : "bar",
    smooth: item.type === 1,
    barMaxWidth: item.type === 0 ? 18 : undefined,
    itemStyle: item.type === 0 ? { borderRadius: [8, 8, 0, 0] } : undefined,
    data: item.data
  }))
}));

async function loadData(timeType: AnalyticsTimeType) {
  const data = await defUserAnalyticsService.GetUserAnalyticsTrend({ timeType });
  Object.assign(trendData, data);
}

watch(
  () => props.timeType,
  value => {
    loadData(value);
  },
  { immediate: true }
);
</script>
