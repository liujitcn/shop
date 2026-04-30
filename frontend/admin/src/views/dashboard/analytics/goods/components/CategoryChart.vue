<template>
  <DataPanelCard title="分类成交件数分布" description="查看核心类目在当前周期内的支付件数结构。">
    <ECharts :option="option" />
  </DataPanelCard>
</template>

<script setup lang="ts">
import { computed, reactive, watch } from "vue";
import ECharts from "@/components/ECharts/index.vue";
import type { ECOption } from "@/components/ECharts/config";
import { defGoodsAnalyticsService } from "@/api/admin/goods_analytics";
import DataPanelCard from "@/components/Card/DataPanelCard.vue";
import type { AnalyticsPieResponse, AnalyticsTimeType } from "@/rpc/common/v1/analytics";

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

/** 加载商品分类分布数据。 */
async function loadData(timeType: AnalyticsTimeType) {
  const data = await defGoodsAnalyticsService.PieGoodsAnalytics({ time_type: timeType });
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
