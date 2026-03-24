<!--  线 + 柱混合图 -->
<template>
  <el-card>
    <template #header>
      <div class="title">
        销售数据统计
        <el-tooltip effect="dark" content="点击试试下载" placement="bottom">
          <i-ep-download class="download" @click="downloadEchart" />
        </el-tooltip>
      </div>
    </template>

    <div :id="id" :class="className" :style="{ height, width }" />
  </el-card>
</template>

<script setup lang="ts">
import * as echarts from "echarts";
import { defDashboardService } from "@/api/admin/dashboard";
import type { DashboardBarResponse } from "@/rpc/admin/dashboard";
import { DashboardTimeType } from "@/rpc/admin/dashboard";
import { useUserStore } from "@/store";

const props = defineProps({
  id: {
    type: String,
    default: "orderBarChart",
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
    legend: {
      x: "center",
      y: "bottom",
      data: ["销售金额", "订单数量", "销售金额增长率", "订单数量增长率"],
      textStyle: {
        color: "#999",
      },
    },
    xAxis: [
      {
        type: "category",
        data: sourceData.axisData,
        axisPointer: {
          type: "shadow",
        },
      },
    ],
    yAxis: [
      {
        type: "value",
        name: "销售金额",
        position: "left",
        alignTicks: true,
        axisLine: {
          show: true,
          color: "#83bff6",
        },
        axisLabel: {
          formatter: "{value}元",
        },
      },
      {
        type: "value",
        name: "订单数量",
        position: "left",
        alignTicks: true,
        offset: 80,
        axisLine: {
          show: true,
          color: "#25d73c",
        },
        axisLabel: {
          formatter: "{value}单",
        },
      },
      {
        type: "value",
        name: "增长率",
        position: "right",
        axisLine: {
          show: true,
          color: "#EE6666",
        },
        axisLabel: {
          formatter: "{value}%",
        },
      },
    ],

    series: [
      {
        name: "订单数量",
        type: "bar",
        yAxisIndex: 1,
        data: sourceData.seriesData[0].value,
        barWidth: 20,
        itemStyle: {
          color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: "#25d73c" },
            { offset: 0.5, color: "#1bc23d" },
            { offset: 1, color: "#179e61" },
          ]),
        },
      },
      {
        name: "销售金额",
        type: "bar",
        data: sourceData.seriesData[1].value.map((item) => item / 100),
        barWidth: 20,
        itemStyle: {
          color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: "#83bff6" },
            { offset: 0.5, color: "#188df0" },
            { offset: 1, color: "#188df0" },
          ]),
        },
      },
      {
        name: "订单数量增长率",
        type: "line",
        yAxisIndex: 2,
        data: sourceData.seriesData[2].value.map((item) => item / 100),
        itemStyle: {
          color: "#409EFF",
        },
      },
      {
        name: "销售金额增长率",
        type: "line",
        yAxisIndex: 2,
        data: sourceData.seriesData[3].value.map((item) => item / 100),
        itemStyle: {
          color: "#67C23A",
        },
      },
    ],
  };
};
const chart = ref<any>("");
onMounted(async () => {
  // 图表初始化
  chart.value = markRaw(echarts.init(document.getElementById(props.id) as HTMLDivElement));
  if (useUserStore().hasPerm("dashboard:bar:order")) {
    const res = await defDashboardService.DashboardBarOrder({
      timeType: DashboardTimeType.DAY,
    });
    Object.assign(sourceData, res);
  }

  chart.value.setOption(getChartOption());

  // 大小自适应
  window.addEventListener("resize", () => {
    chart.value.resize();
  });
});
const downloadEchart = () => {
  // 获取画布图表地址信息
  const img = new Image();
  img.src = chart.value.getDataURL({
    type: "png",
    pixelRatio: 1,
    backgroundColor: "#fff",
  });
  // 当图片加载完成后，生成 URL 并下载
  img.onload = () => {
    const canvas = document.createElement("canvas");
    canvas.width = img.width;
    canvas.height = img.height;
    const ctx = canvas.getContext("2d");
    if (ctx) {
      ctx.drawImage(img, 0, 0, img.width, img.height);
      const link = document.createElement("a");
      link.download = `销售数据统计.png`;
      link.href = canvas.toDataURL("image/png", 0.9);
      document.body.appendChild(link);
      link.click();
      link.remove();
    }
  };
};
</script>
<style lang="scss" scoped>
.title {
  display: flex;
  justify-content: space-between;

  .download {
    cursor: pointer;

    &:hover {
      color: #409eff;
    }
  }
}
</style>
