<template>
  <AnalyticsMetricCards :items="cards" />
</template>

<script setup lang="ts">
import { computed, reactive, watch } from "vue";
import { Document, Position, User } from "@element-plus/icons-vue";
import { defUserAnalyticsService } from "@/api/admin/user_analytics";
import type { UserAnalyticsSummaryResponse } from "@/rpc/admin/user_analytics";
import type { AnalyticsTimeType } from "@/rpc/common/analytics";
import AnalyticsMetricCards, { type AnalyticsMetricCardItem } from "../../components/AnalyticsMetricCards.vue";

const props = defineProps<{
  timeType: AnalyticsTimeType;
}>();

const summary = reactive<UserAnalyticsSummaryResponse>({
  newUserCount: 0,
  newUserGrowthRate: 0,
  orderUserCount: 0,
  orderUserConversionRate: 0,
  activeUserCount: 0,
  activeUserCoverageRate: 0
});

const cards = computed<AnalyticsMetricCardItem[]>(() => [
  {
    key: "newUser",
    label: "新增用户",
    value: String(summary.newUserCount),
    footLabel: "较上期",
    footValue: `${summary.newUserGrowthRate}%`,
    color: "#2d6cdf",
    icon: User
  },
  {
    key: "orderUser",
    label: "下单用户",
    value: String(summary.orderUserCount),
    footLabel: "转化占比",
    footValue: `${(summary.orderUserConversionRate / 10).toFixed(1)}%`,
    color: "#15a87b",
    icon: Document
  },
  {
    key: "activeUser",
    label: "活跃行为用户",
    value: String(summary.activeUserCount),
    footLabel: "覆盖用户",
    footValue: `${(summary.activeUserCoverageRate / 10).toFixed(1)}%`,
    color: "#f08c2e",
    icon: Position
  }
]);

async function loadData(timeType: AnalyticsTimeType) {
  const data = await defUserAnalyticsService.GetUserAnalyticsSummary({ timeType });
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
