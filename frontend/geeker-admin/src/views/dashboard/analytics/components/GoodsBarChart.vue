<template>
  <article class="chart-card">
    <div class="chart-card__header">
      <div>
        <p class="chart-card__eyebrow">商品分析</p>
        <h3 class="chart-card__title">商品销量排行</h3>
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

/** 商品销量排行图表配置。 */
const option = computed<ECOption>(() => ({
  color: ["#2d6cdf"],
  tooltip: {
    trigger: "axis",
    axisPointer: {
      type: "shadow"
    }
  },
  toolbox: {
    right: 8,
    feature: {
      saveAsImage: {}
    }
  },
  grid: {
    top: 30,
    left: 10,
    right: 12,
    bottom: 8,
    containLabel: true
  },
  xAxis: {
    type: "value",
    axisLabel: {
      color: "#7f8ea3"
    },
    splitLine: {
      lineStyle: {
        color: "#eef2f8"
      }
    }
  },
  yAxis: {
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
  series: [
    {
      name: "销量",
      type: "bar",
      barMaxWidth: 16,
      data: sourceData.seriesData[0]?.value ?? [],
      showBackground: true,
      backgroundStyle: {
        color: "#edf3fb",
        borderRadius: 8
      },
      itemStyle: {
        borderRadius: [0, 8, 8, 0]
      }
    }
  ]
}));

/**
 * 根据时间维度加载商品销量排行数据。
 */
async function loadChartData(timeType: DashboardTimeType) {
  const data = await defDashboardService.DashboardBarGoods({
    timeType,
    top: 15
  });
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
  margin-bottom: 12px;
}

.chart-card__eyebrow {
  margin: 0 0 6px;
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.06em;
  color: #7f8ea3;
}

.chart-card__title {
  margin: 0;
  font-size: 20px;
  color: #1f2d3d;
}
</style>
