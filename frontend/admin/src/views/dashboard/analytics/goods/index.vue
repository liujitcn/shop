<template>
  <PageLayout
    title="商品分析"
    description="按时间维度查看商品供给、行为转化与分类成交结构的汇总和趋势变化。"
    :period-label="activeTimeLabel"
    content-ratio="minmax(0, 1fr) minmax(0, 1fr)"
  >
    <template #toolbar>
      <TimeTabs v-model="activeTime" />
    </template>

    <template #metrics>
      <SummaryCards :time-type="activeTime" />
    </template>

    <TrendChart class="goods-analytics__trend" :time-type="activeTime" />
    <SidePanels class="goods-analytics__panels" :time-type="activeTime" />
  </PageLayout>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { AnalyticsTimeType } from "@/rpc/common/v1/analytics";
import PageLayout from "../components/PageLayout.vue";
import TimeTabs from "../components/TimeTabs.vue";
import SidePanels from "./components/SidePanels.vue";
import SummaryCards from "./components/SummaryCards.vue";
import TrendChart from "./components/TrendChart.vue";

defineOptions({
  name: "GoodsAnalytics"
});

// 默认以周维度展示，枚举值与 proto v1 生成成员保持一致。
const activeTime = ref<AnalyticsTimeType>(AnalyticsTimeType.ANALYTICS_TIME_TYPE_WEEK);

const activeTimeLabel = computed(() => {
  switch (activeTime.value) {
    case AnalyticsTimeType.ANALYTICS_TIME_TYPE_MONTH:
      return "本月";
    case AnalyticsTimeType.ANALYTICS_TIME_TYPE_YEAR:
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
