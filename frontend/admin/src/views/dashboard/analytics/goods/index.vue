<template>
  <AnalyticsPageLayout
    title="商品分析"
    description="按时间维度查看商品供给、行为转化与分类成交结构的汇总和趋势变化。"
    :period-label="activeTimeLabel"
    content-ratio="minmax(0, 1fr) minmax(0, 1fr)"
  >
    <template #toolbar>
      <AnalyticsTimeTabs v-model="activeTime" />
    </template>

    <template #metrics>
      <GoodsSummaryCards :time-type="activeTime" />
    </template>

    <GoodsTrendChart class="goods-analytics__trend" :time-type="activeTime" />
    <GoodsSidePanels class="goods-analytics__panels" :time-type="activeTime" />
  </AnalyticsPageLayout>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { AnalyticsTimeType } from "@/rpc/common/analytics";
import AnalyticsPageLayout from "../components/AnalyticsPageLayout.vue";
import AnalyticsTimeTabs from "../components/AnalyticsTimeTabs.vue";
import GoodsSidePanels from "./components/GoodsSidePanels.vue";
import GoodsSummaryCards from "./components/GoodsSummaryCards.vue";
import GoodsTrendChart from "./components/GoodsTrendChart.vue";

defineOptions({
  name: "GoodsAnalytics"
});

const activeTime = ref<AnalyticsTimeType>(AnalyticsTimeType.WEEK);

const activeTimeLabel = computed(() => {
  switch (activeTime.value) {
    case AnalyticsTimeType.MONTH:
      return "本月";
    case AnalyticsTimeType.YEAR:
      return "本年";
    default:
      return "本周";
  }
});
</script>

<style scoped lang="scss">
/* 商品行为趋势独占首行，底部两个图表保持同排均分。 */
.goods-analytics__trend,
.goods-analytics__panels {
  grid-column: 1 / -1;
}
</style>
