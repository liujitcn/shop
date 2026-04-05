<template>
  <DataPanelCard title="订单状态分布" description="关注履约状态与取消结构。">
    <ECharts :option="option" />
  </DataPanelCard>
</template>

<script setup lang="ts">
import { computed, reactive, watch } from "vue";
import ECharts from "@/components/ECharts/index.vue";
import type { ECOption } from "@/components/ECharts/config";
import { defOrderAnalyticsService } from "@/api/admin/order_analytics";
import DataPanelCard from "@/components/Card/DataPanelCard.vue";
import type { AnalyticsPieResponse, AnalyticsTimeType } from "@/rpc/common/analytics";

const props = defineProps<{
  timeType: AnalyticsTimeType;
}>();

const pieData = reactive<AnalyticsPieResponse>({
  items: []
});

const option = computed<ECOption>(() => ({
  color: ["#2d6cdf", "#15a87b", "#f08c2e", "#d9485f", "#7c3aed", "#0ea5e9", "#ef4444", "#84cc16"],
  tooltip: {
    trigger: "item",
    formatter: "{b}<br/>{c} ({d}%)"
  },
  legend: {
    bottom: 0,
    left: "center",
    textStyle: {
      color: "#6d7b8f"
    }
  },
  series: [
    {
      type: "pie",
      radius: ["34%", "74%"],
      center: ["50%", "42%"],
      roseType: "radius",
      itemStyle: {
        borderRadius: 10
      },
      label: {
        color: "#4f5d73"
      },
      data: pieData.items
    }
  ]
}));

async function loadData(timeType: AnalyticsTimeType) {
  const data = await defOrderAnalyticsService.GetOrderAnalyticsPie({ timeType });
  Object.assign(pieData, data);
}

watch(
  () => props.timeType,
  value => {
    loadData(value);
  },
  { immediate: true }
);
</script>
