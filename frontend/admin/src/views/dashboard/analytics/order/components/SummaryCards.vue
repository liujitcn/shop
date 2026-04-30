<template>
  <MetricCards :items="cards" />
</template>

<script setup lang="ts">
import { computed, reactive, watch } from "vue";
import { Document, Money, User } from "@element-plus/icons-vue";
import { defOrderAnalyticsService } from "@/api/admin/order_analytics";
import type { SummaryOrderAnalyticsResponse } from "@/rpc/admin/v1/order_analytics";
import type { AnalyticsTimeType } from "@/rpc/common/v1/analytics";
import { formatPrice } from "@/utils/utils";
import MetricCards, { type MetricCardItem } from "../../components/MetricCards.vue";

const props = defineProps<{
  timeType: AnalyticsTimeType;
}>();

const summary = reactive<SummaryOrderAnalyticsResponse>({
  new_order_count: 0,
  new_order_growth_rate: 0,
  sale_amount: 0,
  average_order_amount: 0,
  order_user_count: 0,
  repurchase_rate: 0
});

/** 按后端统计口径生成订单摘要卡片，并补充指标说明。 */
const cards = computed<MetricCardItem[]>(() => [
  {
    key: "newOrder",
    label: "新增订单",
    labelTooltip: "按当前时间范围统计创建的订单数量。",
    value: String(summary.new_order_count),
    footLabel: "较上期",
    footTooltip:
      "较上期 = (本期订单数 - 上一统计周期订单数) / 上一统计周期订单数。周看上周，月看上月，年看上一年；若上期为 0 且本期大于 0，后端固定返回 100%。",
    footValue: `${summary.new_order_growth_rate}%`,
    color: "#f08c2e",
    icon: Document
  },
  {
    key: "sale_amount",
    label: "成交金额",
    labelTooltip: "按当前时间范围汇总订单成交金额。",
    value: formatPrice(summary.sale_amount),
    footLabel: "客单价",
    footTooltip: "客单价 = 当前周期成交金额 / 当前周期订单数。后端按整数除法计算，前端按金额格式展示。",
    footValue: formatPrice(summary.average_order_amount),
    color: "#d9485f",
    icon: Money
  },
  {
    key: "orderUser",
    label: "下单用户",
    labelTooltip: "按当前时间范围统计下过单的用户数量，同一用户会去重后再统计。",
    value: String(summary.order_user_count),
    footLabel: "复购占比",
    footTooltip:
      "复购占比 = 当前周期内下单次数大于等于 2 次的用户数 / 当前周期下单用户数。后端按千分比返回，前端固定展示 1 位小数百分比。",
    footValue: `${(summary.repurchase_rate / 10).toFixed(1)}%`,
    color: "#2d6cdf",
    icon: User
  }
]);

/** 加载订单摘要数据，并同步覆盖本地展示状态。 */
async function loadData(timeType: AnalyticsTimeType) {
  try {
    const data = await defOrderAnalyticsService.SummaryOrderAnalytics({ time_type: timeType });
    Object.assign(summary, data);
  } catch (_error) {
    // 接口异常时保留默认 0 值展示，避免未捕获 Promise 影响页面后续渲染。
  }
}

watch(
  () => props.timeType,
  value => {
    loadData(value);
  },
  { immediate: true }
);
</script>
