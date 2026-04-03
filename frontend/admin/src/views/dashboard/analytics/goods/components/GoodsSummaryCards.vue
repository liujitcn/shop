<template>
  <AnalyticsMetricCards :items="cards" />
</template>

<script setup lang="ts">
import { computed, reactive, watch } from "vue";
import { Document, Tickets, TrendCharts } from "@element-plus/icons-vue";
import { defGoodsAnalyticsService } from "@/api/admin/goods_analytics";
import type { GoodsAnalyticsSummaryResponse } from "@/rpc/admin/goods_analytics";
import type { AnalyticsTimeType } from "@/rpc/common/analytics";
import AnalyticsMetricCards, { type AnalyticsMetricCardItem } from "../../components/AnalyticsMetricCards.vue";

const props = defineProps<{
  timeType: AnalyticsTimeType;
}>();

const summary = reactive<GoodsAnalyticsSummaryResponse>({
  newGoodsCount: 0,
  putOnGoodsRate: 0,
  activeGoodsCount: 0,
  activeGoodsRate: 0,
  saleCount: 0,
  saleGrowthRate: 0
});

const cards = computed<AnalyticsMetricCardItem[]>(() => [
  {
    key: "newGoods",
    label: "新增商品",
    value: String(summary.newGoodsCount),
    footLabel: "上架完成率",
    footValue: `${(summary.putOnGoodsRate / 10).toFixed(1)}%`,
    color: "#15a87b",
    icon: Tickets
  },
  {
    key: "activeGoods",
    label: "动销商品",
    value: String(summary.activeGoodsCount),
    footLabel: "动销率",
    footValue: `${(summary.activeGoodsRate / 10).toFixed(1)}%`,
    color: "#2d6cdf",
    icon: Document
  },
  {
    key: "saleCount",
    label: "商品销量",
    value: String(summary.saleCount),
    footLabel: "较上期",
    footValue: `${(summary.saleGrowthRate / 10).toFixed(1)}%`,
    color: "#f08c2e",
    icon: TrendCharts
  }
]);

async function loadData(timeType: AnalyticsTimeType) {
  const data = await defGoodsAnalyticsService.GetGoodsAnalyticsSummary({ timeType });
  Object.assign(summary, data);
}

watch(
  () => props.timeType,
  value => {
    loadData(value);
  },
  { immediate: true }
);
</script>
