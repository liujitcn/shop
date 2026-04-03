<template>
  <AnalyticsChartCard title="分类销量分布" description="查看核心类目在当前周期内的销量结构。">
    <ECharts :option="option" />
  </AnalyticsChartCard>
</template>

<script setup lang="ts">
import { computed, reactive, watch } from "vue";
import ECharts from "@/components/ECharts/index.vue";
import type { ECOption } from "@/components/ECharts/config";
import { defGoodsAnalyticsService } from "@/api/admin/goods_analytics";
import AnalyticsChartCard from "../../components/AnalyticsChartCard.vue";
import type { AnalyticsPieResponse, AnalyticsTimeType } from "@/rpc/common/analytics";

const props = defineProps<{
  timeType: AnalyticsTimeType;
}>();

const pieData = reactive<AnalyticsPieResponse>({
  items: []
});

const option = computed<ECOption>(() => ({
  color: ["#15a87b", "#2d6cdf", "#f08c2e", "#d9485f", "#0ea5e9", "#7c3aed", "#ef4444", "#84cc16"],
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
  const data = await defGoodsAnalyticsService.GetGoodsAnalyticsPie({ timeType });
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
