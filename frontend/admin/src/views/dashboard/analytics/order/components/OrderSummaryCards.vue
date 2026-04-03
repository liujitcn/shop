<template>
  <AnalyticsMetricCards :items="cards" />
</template>

<script setup lang="ts">
import { computed, reactive, watch } from "vue";
import { Document, Money, User } from "@element-plus/icons-vue";
import { defOrderAnalyticsService } from "@/api/admin/order_analytics";
import type { OrderAnalyticsSummaryResponse } from "@/rpc/admin/order_analytics";
import type { AnalyticsTimeType } from "@/rpc/common/analytics";
import { formatPrice } from "@/utils/utils";
import AnalyticsMetricCards, { type AnalyticsMetricCardItem } from "../../components/AnalyticsMetricCards.vue";

const props = defineProps<{
  timeType: AnalyticsTimeType;
}>();

const summary = reactive<OrderAnalyticsSummaryResponse>({
  newOrderCount: 0,
  newOrderGrowthRate: 0,
  saleAmount: 0,
  averageOrderAmount: 0,
  orderUserCount: 0,
  repurchaseRate: 0
});

const cards = computed<AnalyticsMetricCardItem[]>(() => [
  {
    key: "newOrder",
    label: "新增订单",
    value: String(summary.newOrderCount),
    footLabel: "较上期",
    footValue: `${summary.newOrderGrowthRate}%`,
    color: "#f08c2e",
    icon: Document
  },
  {
    key: "saleAmount",
    label: "成交金额",
    value: formatPrice(summary.saleAmount),
    footLabel: "客单价",
    footValue: formatPrice(summary.averageOrderAmount),
    color: "#d9485f",
    icon: Money
  },
  {
    key: "orderUser",
    label: "下单用户",
    value: String(summary.orderUserCount),
    footLabel: "复购占比",
    footValue: `${(summary.repurchaseRate / 10).toFixed(1)}%`,
    color: "#2d6cdf",
    icon: User
  }
]);

async function loadData(timeType: AnalyticsTimeType) {
  const data = await defOrderAnalyticsService.GetOrderAnalyticsSummary({ timeType });
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
