<template>
  <PageLayout
    title="用户分析"
    description="按时间维度查看用户增长、转化与行为覆盖的汇总和趋势变化。"
    :period-label="activeTimeLabel"
    content-ratio="minmax(0, 1.25fr) minmax(320px, 0.9fr)"
  >
    <template #toolbar>
      <TimeTabs v-model="activeTime" />
    </template>

    <template #metrics>
      <SummaryCards :time-type="activeTime" />
    </template>

    <TrendChart :time-type="activeTime" />
    <BehaviorRankChart :time-type="activeTime" />
  </PageLayout>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { AnalyticsTimeType } from "@/rpc/common/v1/analytics";
import PageLayout from "../components/PageLayout.vue";
import TimeTabs from "../components/TimeTabs.vue";
import BehaviorRankChart from "./components/BehaviorRankChart.vue";
import SummaryCards from "./components/SummaryCards.vue";
import TrendChart from "./components/TrendChart.vue";

defineOptions({
  name: "UserAnalytics"
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
