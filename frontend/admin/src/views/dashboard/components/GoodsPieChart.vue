<!-- 饼图 -->
<template>
  <el-card>
    <template #header>商品分类饼图</template>
    <div :id="id" :class="className" :style="{ height, width }" />
  </el-card>
</template>

<script setup lang="ts">
import * as echarts from "echarts";
import type { DashboardPieResponse } from "@/rpc/admin/dashboard";
import { DashboardTimeType } from "@/rpc/admin/dashboard";
import { defDashboardService } from "@/api/admin/dashboard";
import { useUserStore } from "@/store";

const props = defineProps({
  id: {
    type: String,
    default: "goodsPieChart",
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

const sourceData = reactive<DashboardPieResponse>({
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
    title: {
      text: "商品分类",
      left: "center",
    },
    tooltip: {
      trigger: "item",
      formatter: "{a} <br/>{b} : {c} ({d}%)",
    },
    legend: {
      top: "bottom",
      textStyle: {
        color: "#999",
      },
    },
    series: [
      {
        name: "商品数量",
        type: "pie",
        radius: [50, 130],
        center: ["50%", "50%"],
        roseType: "area",
        itemStyle: {
          borderRadius: 1,
        },
        data: sourceData.seriesData,
        emphasis: {
          itemStyle: {
            shadowBlur: 10,
            shadowOffsetX: 0,
            shadowColor: "rgba(0, 0, 0, 0.5)",
          },
        },
        label: {
          show: true, // 显示标签
          formatter: "{b}", // {b}代表数据项名称，{c}代表数据项的值
        },
      },
    ],
  };
};

onMounted(async () => {
  const chart = echarts.init(document.getElementById(props.id) as HTMLDivElement);
  if (useUserStore().hasPerm("dashboard:pie:goods")) {
    const res = await defDashboardService.DashboardPieGoods({
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
