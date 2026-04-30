<template>
  <DataPanelCard title="商品行为趋势" description="按时间维度观察浏览、加购、成交件数和支付金额变化。">
    <ECharts :option="option" />
  </DataPanelCard>
</template>

<script setup lang="ts">
import { computed, reactive, watch } from "vue";
import type { YAXisComponentOption } from "echarts";
import ECharts from "@/components/ECharts/index.vue";
import type { ECOption } from "@/components/ECharts/config";
import { defGoodsAnalyticsService } from "@/api/admin/goods_analytics";
import DataPanelCard from "@/components/Card/DataPanelCard.vue";
import type { AnalyticsTimeType, AnalyticsTrendResponse } from "@/rpc/common/v1/analytics";

const props = defineProps<{
  timeType: AnalyticsTimeType;
}>();

const trendData = reactive<AnalyticsTrendResponse>({
  axis: [],
  series: [],
  y_axis_names: []
});

/** 根据后端返回的 Y 轴名称动态构造图表坐标轴。 */
const yAxisList = computed<YAXisComponentOption[]>(() => {
  const axisNames = trendData.y_axis_names.length ? trendData.y_axis_names : ["次数 / 件数"];
  return axisNames.map((name, index) => ({
    type: "value" as const,
    name,
    position: index % 2 === 0 ? ("left" as const) : ("right" as const),
    axisLabel: {
      color: "#6d7b8f"
    },
    splitLine: {
      lineStyle: {
        color: index === 0 ? "#edf2f7" : "transparent"
      }
    }
  }));
});

const option = computed<ECOption>(() => ({
  color: ["#2d6cdf", "#f08c2e", "#15a87b", "#d9485f"],
  tooltip: {
    trigger: "axis",
    axisPointer: {
      type: "cross"
    }
  },
  legend: {
    bottom: 0,
    textStyle: {
      color: "#6d7b8f"
    }
  },
  grid: {
    top: 28,
    left: 20,
    right: 20,
    bottom: 44,
    containLabel: true
  },
  xAxis: {
    type: "category",
    data: trendData.axis,
    axisLabel: {
      color: "#6d7b8f"
    },
    axisLine: {
      lineStyle: {
        color: "#dbe4ee"
      }
    }
  },
  yAxis: yAxisList.value,
  series: trendData.series.map(item => ({
    name: item.name,
    type: item.type === 1 ? "line" : "bar",
    smooth: item.type === 1,
    yAxisIndex: item.y_axis_index ?? 0,
    barMaxWidth: item.type === 0 ? 18 : undefined,
    itemStyle: item.type === 0 ? { borderRadius: [8, 8, 0, 0] } : undefined,
    data: item.data
  }))
}));

/** 加载商品行为趋势数据。 */
async function loadData(timeType: AnalyticsTimeType) {
  const data = await defGoodsAnalyticsService.TrendGoodsAnalytics({ time_type: timeType });
  Object.assign(trendData, data);
}

watch(
  () => props.timeType,
  value => {
    loadData(value);
  },
  { immediate: true }
);
</script>
