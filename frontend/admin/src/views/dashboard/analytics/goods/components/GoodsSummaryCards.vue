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

/** 按后端统计口径生成商品摘要卡片，并补充指标说明。 */
const cards = computed<AnalyticsMetricCardItem[]>(() => [
  {
    key: "newGoods",
    label: "新增商品",
    labelTooltip: "按当前时间范围统计创建的商品数量。",
    value: String(summary.newGoodsCount),
    footLabel: "上架率",
    footTooltip: "上架率 = 当前上架商品数 / 商品总数。后端按千分比返回，前端固定展示 1 位小数百分比。",
    footValue: `${(summary.putOnGoodsRate / 10).toFixed(1)}%`,
    color: "#15a87b",
    icon: Tickets
  },
  {
    key: "activeGoods",
    label: "动销商品",
    labelTooltip: "按当前时间范围统计产生过销量的商品数量，同一商品会去重后再统计。",
    value: String(summary.activeGoodsCount),
    footLabel: "动销率",
    footTooltip: "动销率 = 当前周期动销商品数 / 商品总数。后端按千分比返回，前端固定展示 1 位小数百分比。",
    footValue: `${(summary.activeGoodsRate / 10).toFixed(1)}%`,
    color: "#2d6cdf",
    icon: Document
  },
  {
    key: "saleCount",
    label: "商品销量",
    labelTooltip: "按当前时间范围汇总已售出的商品件数。",
    value: String(summary.saleCount),
    footLabel: "较上期",
    footTooltip:
      "较上期 = (本期商品销量 - 上一统计周期商品销量) / 上一统计周期商品销量。周看上周，月看上月，年看上一年；若上期为 0 且本期大于 0，后端固定返回 100%。",
    footValue: `${summary.saleGrowthRate}%`,
    color: "#f08c2e",
    icon: TrendCharts
  }
]);

/** 加载商品摘要数据，并同步覆盖本地展示状态。 */
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
