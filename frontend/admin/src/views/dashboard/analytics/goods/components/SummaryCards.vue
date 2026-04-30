<template>
  <MetricCards :items="cards" />
</template>

<script setup lang="ts">
import { computed, reactive, watch } from "vue";
import { Box, Document, Goods, Money, Tickets, TrendCharts } from "@element-plus/icons-vue";
import { defGoodsAnalyticsService } from "@/api/admin/goods_analytics";
import type { SummaryGoodsAnalyticsResponse } from "@/rpc/admin/v1/goods_analytics";
import type { AnalyticsTimeType } from "@/rpc/common/v1/analytics";
import MetricCards, { type MetricCardItem } from "../../components/MetricCards.vue";
import { formatPrice } from "@/utils/utils";

const props = defineProps<{
  timeType: AnalyticsTimeType;
}>();

const summary = reactive<SummaryGoodsAnalyticsResponse>({
  new_goods_count: 0,
  put_on_goods_rate: 0,
  active_goods_count: 0,
  active_goods_rate: 0,
  sale_count: 0,
  sale_growth_rate: 0,
  view_count: 0,
  collect_count: 0,
  cart_count: 0,
  order_count: 0,
  pay_count: 0,
  pay_amount: 0,
  cart_conversion_rate: 0,
  order_conversion_rate: 0,
  pay_conversion_rate: 0,
  pay_unit_price: 0
});

/** 统一将千分比指标格式化成 1 位小数百分比。 */
function formatRatio(value: number) {
  return `${(value / 10).toFixed(1)}%`;
}

/** 按商品分析口径生成摘要卡片。 */
const cards = computed<MetricCardItem[]>(() => [
  {
    key: "newGoods",
    label: "新增商品",
    labelTooltip: "按当前时间范围统计创建的商品数量。",
    value: String(summary.new_goods_count),
    footLabel: "上架率",
    footTooltip: "上架率 = 当前上架商品数 / 商品总数。后端按千分比返回，前端固定展示 1 位小数百分比。",
    footValue: formatRatio(summary.put_on_goods_rate),
    color: "#15a87b",
    icon: Tickets
  },
  {
    key: "activeGoods",
    label: "动销商品",
    labelTooltip: "按当前时间范围统计产生过支付件数的商品数量，同一商品会去重后再统计。",
    value: String(summary.active_goods_count),
    footLabel: "动销率",
    footTooltip: "动销率 = 当前周期动销商品数 / 商品总数。后端按千分比返回，前端固定展示 1 位小数百分比。",
    footValue: formatRatio(summary.active_goods_rate),
    color: "#2d6cdf",
    icon: Document
  },
  {
    key: "view_count",
    label: "商品浏览",
    labelTooltip: "按当前时间范围统计商品详情页浏览次数。",
    value: String(summary.view_count),
    footLabel: "收藏次数",
    footTooltip: "收藏次数按用户收藏事件累计，未做用户去重。",
    footValue: String(summary.collect_count),
    color: "#f08c2e",
    icon: Goods
  },
  {
    key: "cart_count",
    label: "加购件数",
    labelTooltip: "按当前时间范围累计加入购物车的商品件数。",
    value: String(summary.cart_count),
    footLabel: "加购下单率",
    footTooltip: "加购下单率 = 下单次数 / 加购件数。后端按千分比返回，前端固定展示 1 位小数百分比。",
    footValue: formatRatio(summary.order_conversion_rate),
    color: "#d9485f",
    icon: Box
  },
  {
    key: "sale_count",
    label: "商品销量",
    labelTooltip: "按当前时间范围汇总支付成功的商品件数。",
    value: String(summary.sale_count),
    footLabel: "浏览支付率",
    footTooltip: "浏览支付率 = 支付次数 / 浏览次数。后端按千分比返回，前端固定展示 1 位小数百分比。",
    footValue: formatRatio(summary.pay_conversion_rate),
    color: "#7c4dff",
    icon: TrendCharts
  },
  {
    key: "pay_amount",
    label: "支付金额",
    labelTooltip: "按当前时间范围汇总支付成功的商品金额。",
    value: `${formatPrice(summary.pay_amount)} 元`,
    footLabel: "件均成交价",
    footTooltip: "件均成交价 = 支付金额 / 支付件数。",
    footValue: `${formatPrice(summary.pay_unit_price)} 元`,
    color: "#00838f",
    icon: Money
  }
]);

/** 加载商品摘要数据，并同步覆盖本地展示状态。 */
async function loadData(timeType: AnalyticsTimeType) {
  const data = await defGoodsAnalyticsService.SummaryGoodsAnalytics({ time_type: timeType });
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
