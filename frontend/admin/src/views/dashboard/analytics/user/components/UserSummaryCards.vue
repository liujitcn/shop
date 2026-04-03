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

/** 按后端统计口径生成用户摘要卡片，并补充指标说明。 */
const cards = computed<AnalyticsMetricCardItem[]>(() => [
  {
    key: "newUser",
    label: "新增用户",
    labelTooltip: "按当前时间范围统计新注册的用户数量。",
    value: String(summary.newUserCount),
    footLabel: "较上期",
    footTooltip:
      "较上期 = (本期新增用户数 - 上一统计周期新增用户数) / 上一统计周期新增用户数。周看上周，月看上月，年看上一年；若上期为 0 且本期大于 0，后端固定返回 100%。",
    footValue: `${summary.newUserGrowthRate}%`,
    color: "#2d6cdf",
    icon: User
  },
  {
    key: "orderUser",
    label: "下单用户",
    labelTooltip: "按当前时间范围统计下过单的用户数量，同一用户会去重后再统计。",
    value: String(summary.orderUserCount),
    footLabel: "转化占比",
    footTooltip: "转化占比 = 当前周期下单用户数 / 当前周期新增用户数。当前实现严格按后端现有口径计算，并非全量用户转化率。",
    footValue: `${(summary.orderUserConversionRate / 10).toFixed(1)}%`,
    color: "#15a87b",
    icon: Document
  },
  {
    key: "activeUser",
    label: "活跃行为用户",
    labelTooltip: "按当前时间范围统计发生过地址填写、收藏、加购、门店申请或下单行为的用户数量，同一用户会去重后再统计。",
    value: String(summary.activeUserCount),
    footLabel: "覆盖用户",
    footTooltip: "覆盖用户 = 当前周期活跃行为用户数 / 当前周期新增用户数。当前实现严格按后端现有口径计算，并非全量用户覆盖率。",
    footValue: `${(summary.activeUserCoverageRate / 10).toFixed(1)}%`,
    color: "#f08c2e",
    icon: Position
  }
]);

/** 加载用户摘要数据，并同步覆盖本地展示状态。 */
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
