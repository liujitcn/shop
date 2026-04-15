<template>
  <AnalyticsMetricCards :items="cards" />
</template>

<script setup lang="ts">
import { computed, reactive, watch } from "vue";
import { Box, Document, Goods, Money, Tickets, TrendCharts } from "@element-plus/icons-vue";
import { defGoodsAnalyticsService } from "@/api/admin/goods_analytics";
import type { GoodsAnalyticsSummaryResponse } from "@/rpc/admin/goods_analytics";
import type { AnalyticsTimeType } from "@/rpc/common/analytics";
import AnalyticsMetricCards, { type AnalyticsMetricCardItem } from "../../components/AnalyticsMetricCards.vue";
import { formatPrice } from "@/utils/utils";

const props = defineProps<{
  timeType: AnalyticsTimeType;
}>();

const summary = reactive<GoodsAnalyticsSummaryResponse>({
  newGoodsCount: 0,
  putOnGoodsRate: 0,
  activeGoodsCount: 0,
  activeGoodsRate: 0,
  saleCount: 0,
  saleGrowthRate: 0,
  viewCount: 0,
  collectCount: 0,
  cartCount: 0,
  orderCount: 0,
  payCount: 0,
  payAmount: 0,
  cartConversionRate: 0,
  orderConversionRate: 0,
  payConversionRate: 0,
  payUnitPrice: 0
});

/** 统一将千分比指标格式化成 1 位小数百分比。 */
function formatRatio(value: number) {
  return `${(value / 10).toFixed(1)}%`;
}

/** 按商品分析口径生成摘要卡片。 */
const cards = computed<AnalyticsMetricCardItem[]>(() => [
  {
    key: "newGoods",
    label: "新增商品",
    labelTooltip: "按当前时间范围统计创建的商品数量。",
    value: String(summary.newGoodsCount),
    footLabel: "上架率",
    footTooltip: "上架率 = 当前上架商品数 / 商品总数。后端按千分比返回，前端固定展示 1 位小数百分比。",
    footValue: formatRatio(summary.putOnGoodsRate),
    color: "#15a87b",
    icon: Tickets
  },
  {
    key: "activeGoods",
    label: "动销商品",
    labelTooltip: "按当前时间范围统计产生过支付件数的商品数量，同一商品会去重后再统计。",
    value: String(summary.activeGoodsCount),
    footLabel: "动销率",
    footTooltip: "动销率 = 当前周期动销商品数 / 商品总数。后端按千分比返回，前端固定展示 1 位小数百分比。",
    footValue: formatRatio(summary.activeGoodsRate),
    color: "#2d6cdf",
    icon: Document
  },
  {
    key: "viewCount",
    label: "商品浏览",
    labelTooltip: "按当前时间范围统计商品详情页浏览次数。",
    value: String(summary.viewCount),
    footLabel: "收藏次数",
    footTooltip: "收藏次数按用户收藏事件累计，未做用户去重。",
    footValue: String(summary.collectCount),
    color: "#f08c2e",
    icon: Goods
  },
  {
    key: "cartCount",
    label: "加购件数",
    labelTooltip: "按当前时间范围累计加入购物车的商品件数。",
    value: String(summary.cartCount),
    footLabel: "加购下单率",
    footTooltip: "加购下单率 = 下单次数 / 加购件数。后端按千分比返回，前端固定展示 1 位小数百分比。",
    footValue: formatRatio(summary.orderConversionRate),
    color: "#d9485f",
    icon: Box
  },
  {
    key: "saleCount",
    label: "商品销量",
    labelTooltip: "按当前时间范围汇总支付成功的商品件数。",
    value: String(summary.saleCount),
    footLabel: "浏览支付率",
    footTooltip: "浏览支付率 = 支付次数 / 浏览次数。后端按千分比返回，前端固定展示 1 位小数百分比。",
    footValue: formatRatio(summary.payConversionRate),
    color: "#7c4dff",
    icon: TrendCharts
  },
  {
    key: "payAmount",
    label: "支付金额",
    labelTooltip: "按当前时间范围汇总支付成功的商品金额。",
    value: `${formatPrice(summary.payAmount)} 元`,
    footLabel: "件均成交价",
    footTooltip: "件均成交价 = 支付金额 / 支付件数。",
    footValue: `${formatPrice(summary.payUnitPrice)} 元`,
    color: "#00838f",
    icon: Money
  }
]);

/** 加载商品摘要数据，并同步覆盖本地展示状态。 */
async function loadData(timeType: AnalyticsTimeType) {
  const data = await defGoodsAnalyticsService.GetGoodsAnalyticsSummary({ timeType });
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
