<template>
  <article class="chart-card">
    <div class="chart-card__header">
      <div>
        <p class="chart-card__eyebrow">订单状态</p>
        <h3 class="chart-card__title">订单状态对比</h3>
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
import type { DashboardRadarResponse, DashboardTimeType } from "@/rpc/admin/dashboard";

const props = defineProps<{
  timeType: DashboardTimeType;
}>();

const sourceData = reactive<DashboardRadarResponse>({
  legendData: [],
  radarIndicator: [],
  seriesData: []
});

/** 订单状态雷达图配置。 */
const option = computed<ECOption>(() => ({
  color: ["#2d6cdf", "#d9485f", "#15a87b"],
  tooltip: {
    trigger: "item"
  },
  legend: {
    bottom: 0,
    textStyle: {
      color: "#7f8ea3"
    },
    data: sourceData.legendData
  },
  toolbox: {
    right: 8,
    feature: {
      saveAsImage: {}
    }
  },
  radar: {
    radius: "62%",
    splitNumber: 5,
    axisName: {
      color: "#4f5d73"
    },
    splitLine: {
      lineStyle: {
        color: "#e7edf7"
      }
    },
    splitArea: {
      areaStyle: {
        color: ["rgb(45 108 223 / 0.03)", "rgb(45 108 223 / 0.01)"]
      }
    },
    indicator: sourceData.radarIndicator
  },
  series: [
    {
      name: "订单状态",
      type: "radar",
      data: sourceData.seriesData,
      areaStyle: {
        opacity: 0.12
      }
    }
  ]
}));

/**
 * 根据时间维度加载订单状态雷达图数据。
 */
async function loadChartData(timeType: DashboardTimeType) {
  const data = await defDashboardService.DashboardRadarOrder({ timeType });
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
