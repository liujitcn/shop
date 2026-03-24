<!-- 漏斗图 -->
<template>
  <el-card>
    <template #header>商品销量排行榜</template>
    <div :id="id" :class="className" :style="{ height, width }" />
  </el-card>
</template>

<script setup lang="ts">
import * as echarts from "echarts";
import type { DashboardBarResponse } from "@/rpc/admin/dashboard";
import { DashboardTimeType } from "@/rpc/admin/dashboard";
import { useUserStore } from "@/store";
import { defDashboardService } from "@/api/admin/dashboard";

const props = defineProps({
  id: {
    type: String,
    default: "goodsBarChart",
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
const sourceData = reactive<DashboardBarResponse>({
  /** 图例的数据数组 */
  axisData: [],
  /** 数据内容数组 */
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
      trigger: "axis",
      axisPointer: {
        type: "cross",
        crossStyle: {
          color: "#999",
        },
      },
    },
    xAxis: [
      {
        type: "value",
      },
    ],
    yAxis: [
      {
        type: "category",
        data: sourceData.axisData,
      },
    ],

    series: [
      {
        name: "销售数量",
        type: "bar",
        data: sourceData.seriesData[0].value,
        barWidth: 20,
        itemStyle: {
          color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: "#83bff6" },
            { offset: 0.5, color: "#188df0" },
            { offset: 1, color: "#188df0" },
          ]),
        },
      },
    ],
  };
};

onMounted(async () => {
  const chart = echarts.init(document.getElementById(props.id) as HTMLDivElement);
  if (useUserStore().hasPerm("dashboard:bar:goods")) {
    const res = await defDashboardService.DashboardBarGoods({
      timeType: DashboardTimeType.DAY,
      top: 15,
    });
    Object.assign(sourceData, res);
  }
  console.log(getChartOption());
  chart.setOption(getChartOption());

  window.addEventListener("resize", () => {
    chart.resize();
  });
});
</script>
