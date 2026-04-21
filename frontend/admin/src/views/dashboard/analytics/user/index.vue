<template>
  <AnalyticsPageLayout
    title=""
    description="按时间维度查看用户增长、转化与行为覆盖的汇总和趋势变化。"
    :period-label="activeTimeLabel"
    content-ratio="minmax(0, 1.25fr) minmax(320px, 0.9fr)"
  >
    <template #toolbar>
      <AnalyticsTimeTabs v-model="activeTime" />
    </template>

    <template #metrics>
      <UserSummaryCards :time-type="activeTime" />
    </template>

    <UserTrendChart :time-type="activeTime" />
    <UserBehaviorRankChart :time-type="activeTime" />
  </AnalyticsPageLayout>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { AnalyticsTimeType } from "@/rpc/common/analytics";
import AnalyticsPageLayout from "../components/AnalyticsPageLayout.vue";
import AnalyticsTimeTabs from "../components/AnalyticsTimeTabs.vue";
import UserBehaviorRankChart from "./components/UserBehaviorRankChart.vue";
import UserSummaryCards from "./components/UserSummaryCards.vue";
import UserTrendChart from "./components/UserTrendChart.vue";

defineOptions({
  name: "UserAnalytics"
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
