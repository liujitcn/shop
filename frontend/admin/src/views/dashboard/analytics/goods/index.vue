<template>
  <AnalyticsPageLayout
    title="商品分析"
    description="按时间维度查看商品供给、动销效率与分类表现的汇总和趋势变化。"
    :period-label="activeTimeLabel"
    content-ratio="minmax(0, 1.25fr) minmax(320px, 0.9fr)"
  >
    <template #toolbar>
      <AnalyticsTimeTabs v-model="activeTime" />
    </template>

    <template #metrics>
      <GoodsSummaryCards :time-type="activeTime" />
    </template>

    <GoodsTrendChart :time-type="activeTime" />
    <GoodsCategoryChart :time-type="activeTime" />
  </AnalyticsPageLayout>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { AnalyticsTimeType } from "@/rpc/common/analytics";
import AnalyticsPageLayout from "../components/AnalyticsPageLayout.vue";
import AnalyticsTimeTabs from "../components/AnalyticsTimeTabs.vue";
import GoodsCategoryChart from "./components/GoodsCategoryChart.vue";
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
