<!-- 雷达图 -->
<template>
  <el-card>
    <template #header>订单状态雷达图</template>
    <div :id="id" :class="className" :style="{ height, width }" />
  </el-card>
</template>

<script setup lang="ts">
import * as echarts from "echarts";
import type { AnalyticsRadarResponse } from "@/rpc/admin/analytics";
import { AnalyticsTimeType } from "@/rpc/admin/analytics";
import { defAnalyticsService } from "@/api/admin/analytics";

const props = defineProps({
  id: {
    type: String,
    default: "orderRadarChart",
  },
  className: {
    type: String,
    default: "",
  },
  width: {
    type: String,
    default: "100%",
  },
  height: {
    type: String,
    default: "400px",
  },
});
const sourceData = reactive<AnalyticsRadarResponse>({
  legendData: [],
  radarIndicator: [],
  seriesData: [],
});

const getChartOption = () => {
  return {
    grid: {
      left: "2%",
      right: "2%",
      bottom: "10%",
      containLabel: true,
    },
    tooltip: {
      trigger: "item",
    },
    legend: {
      x: "center",
      y: "bottom",
      data: sourceData.legendData,
      textStyle: {
        color: "#999",
      },
    },
    radar: {
      radius: "60%",
      indicator: sourceData.radarIndicator,
    },
    series: [
      {
        name: "分类销量对比",
        type: "radar",
        itemStyle: {
          borderRadius: 6,
        },
        data: sourceData.seriesData,
      },
    ],
  };
};
onMounted(async () => {
  const chart = echarts.init(document.getElementById(props.id) as HTMLDivElement);
  const res = await defAnalyticsService.AnalyticsRadarOrder({
    timeType: AnalyticsTimeType.DAY,
  });
  Object.assign(sourceData, res);
  chart.setOption(getChartOption());

  window.addEventListener("resize", () => {
    chart.resize();
  });
});
</script>
