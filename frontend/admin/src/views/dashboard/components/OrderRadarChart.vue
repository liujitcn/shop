<!-- 雷达图 -->
<template>
  <el-card>
    <template #header>订单状态雷达图</template>
    <div :id="id" :class="className" :style="{ height, width }" />
  </el-card>
</template>

<script setup lang="ts">
import * as echarts from "echarts";
import type { DashboardRadarResponse } from "@/rpc/admin/dashboard";
import { DashboardTimeType } from "@/rpc/admin/dashboard";
import { defDashboardService } from "@/api/admin/dashboard";
import { useUserStore } from "@/store";

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
const sourceData = reactive<DashboardRadarResponse>({
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
  if (useUserStore().hasPerm("dashboard:radar:order")) {
    const res = await defDashboardService.DashboardRadarOrder({
      timeType: DashboardTimeType.DAY,
    });
    Object.assign(sourceData, res);
  }
  chart.setOption(getChartOption());

  window.addEventListener("resize", () => {
    chart.resize();
  });
});
</script>
