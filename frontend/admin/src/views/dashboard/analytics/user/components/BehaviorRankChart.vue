<template>
  <DataPanelCard title="用户行为覆盖" description="查看用户沉淀与转化行为的覆盖情况。">
    <ECharts :option="option" />
  </DataPanelCard>
</template>

<script setup lang="ts">
import { computed, reactive, watch } from "vue";
import ECharts from "@/components/ECharts/index.vue";
import type { ECOption } from "@/components/ECharts/config";
import { defUserAnalyticsService } from "@/api/admin/user_analytics";
import DataPanelCard from "@/components/Card/DataPanelCard.vue";
import type { AnalyticsRankResponse, AnalyticsTimeType } from "@/rpc/common/v1/analytics";

const props = defineProps<{
  timeType: AnalyticsTimeType;
}>();

const rankData = reactive<AnalyticsRankResponse>({
  items: []
});

const option = computed<ECOption>(() => ({
  color: ["#2d6cdf"],
  tooltip: {
    trigger: "axis",
    axisPointer: {
      type: "shadow"
    }
  },
  grid: {
    top: 16,
    left: 18,
    right: 18,
    bottom: 12,
    containLabel: true
  },
  xAxis: {
    type: "value",
    axisLabel: {
      color: "#6d7b8f"
    },
    splitLine: {
      lineStyle: {
        color: "#edf2f7"
      }
    }
  },
  yAxis: {
    type: "category",
    inverse: true,
    axisLabel: {
      color: "#4f5d73"
    },
    data: rankData.items.map(item => item.name)
  },
  series: [
    {
      type: "bar",
      barWidth: 18,
      data: rankData.items.map(item => item.value),
      label: {
        show: true,
        position: "right",
        color: "#4f5d73"
      },
      itemStyle: {
        borderRadius: [0, 10, 10, 0]
      }
    }
  ]
}));

/** 加载用户行为覆盖排行数据。 */
async function loadData(timeType: AnalyticsTimeType) {
  const data = await defUserAnalyticsService.RankUserAnalytics({ time_type: timeType });
  Object.assign(rankData, data);
}

watch(
  () => props.timeType,
  value => {
    loadData(value);
  },
  { immediate: true }
);
</script>
