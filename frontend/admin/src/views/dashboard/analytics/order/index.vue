<template>
  <AnalyticsPageLayout
    title=""
    description="按时间维度查看订单规模、成交结果与状态结构的汇总和趋势变化。"
    :period-label="activeTimeLabel"
    content-ratio="minmax(0, 1.25fr) minmax(320px, 0.9fr)"
  >
    <template #toolbar>
      <AnalyticsTimeTabs v-model="activeTime" />
    </template>

    <template #metrics>
      <OrderSummaryCards :time-type="activeTime" />
    </template>

    <OrderTrendChart :time-type="activeTime" />
    <OrderStatusChart :time-type="activeTime" />
  </AnalyticsPageLayout>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { AnalyticsTimeType } from "@/rpc/common/analytics";
import AnalyticsPageLayout from "../components/AnalyticsPageLayout.vue";
import AnalyticsTimeTabs from "../components/AnalyticsTimeTabs.vue";
import OrderStatusChart from "./components/OrderStatusChart.vue";
import OrderSummaryCards from "./components/OrderSummaryCards.vue";
import OrderTrendChart from "./components/OrderTrendChart.vue";

defineOptions({
  name: "OrderAnalytics"
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
