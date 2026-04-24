<template>
  <div v-loading="loading" class="goods-month-report">
    <AnalyticsPageLayout
      title="商品月报"
      description="按月份查看商品浏览、加购、下单与支付表现，支持下钻到日报。"
      period-label=""
      content-ratio="minmax(0, 1fr)"
    >
      <template #toolbar>
        <div class="report-toolbar">
          <el-date-picker
            v-model="monthRange"
            type="monthrange"
            unlink-panels
            range-separator="~"
            start-placeholder="开始月份"
            end-placeholder="结束月份"
            value-format="YYYY-MM"
          />
          <el-button type="primary" @click="loadData">查询</el-button>
        </div>
      </template>

      <template #metrics>
        <AnalyticsMetricCards :items="metricItems" />
      </template>

      <article class="report-card report-card--tabs">
        <div class="report-card__header report-card__header--tabs">
          <div class="report-card__tabs">
            <button
              type="button"
              class="report-tab"
              :class="{ 'report-tab--active': activePanel === 'trend' }"
              @click="handlePanelChange('trend')"
            >
              行为趋势
            </button>
            <button
              type="button"
              class="report-tab"
              :class="{ 'report-tab--active': activePanel === 'summary' }"
              @click="handlePanelChange('summary')"
            >
              月度汇总
            </button>
          </div>
          <el-button v-if="activePanel === 'summary'" @click="handleExport">导出 Excel</el-button>
        </div>

        <div v-show="activePanel === 'trend'" class="report-panel report-panel--chart">
          <ECharts :option="chartOption" :on-click="handleChartClick" />
        </div>

        <div v-show="activePanel === 'summary'" class="report-panel">
          <el-table :data="report.items" border class="report-table">
            <el-table-column prop="month" label="月份" min-width="120">
              <template #default="{ row }">
                <el-button link type="primary" @click="openDayReport(row.month)">{{ row.month }}</el-button>
              </template>
            </el-table-column>
            <el-table-column prop="viewCount" label="浏览次数" min-width="110" align="right" />
            <el-table-column prop="collectCount" label="收藏次数" min-width="110" align="right" />
            <el-table-column prop="cartCount" label="加购件数" min-width="110" align="right" />
            <el-table-column prop="orderCount" label="下单次数" min-width="110" align="right" />
            <el-table-column prop="payCount" label="支付次数" min-width="110" align="right" />
            <el-table-column prop="payGoodsNum" label="支付件数" min-width="110" align="right" />
            <el-table-column prop="payAmount" label="支付金额（元）" min-width="140" align="right">
              <template #default="{ row }">{{ formatPrice(row.payAmount) }}</template>
            </el-table-column>
            <el-table-column prop="payConversionRate" label="浏览支付率" min-width="120" align="right">
              <template #default="{ row }">{{ formatRatio(row.payConversionRate) }}</template>
            </el-table-column>
            <el-table-column label="操作" width="120" fixed="right" align="center">
              <template #default="{ row }">
                <el-button link type="primary" @click="openDayReport(row.month)">查看日报</el-button>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </article>
    </AnalyticsPageLayout>
  </div>
</template>

<script setup lang="ts">
defineOptions({
  name: "GoodsMonthReport"
});

import { computed, reactive, ref } from "vue";
import dayjs from "dayjs";
import { ElMessage } from "element-plus";
import { Box, Goods, Money, Tickets, TrendCharts } from "@element-plus/icons-vue";
import type { ECElementEvent } from "echarts/core";
import ECharts from "@/components/ECharts/index.vue";
import type { ECOption } from "@/components/ECharts/config";
import AnalyticsMetricCards, {
  type AnalyticsMetricCardItem
} from "@/views/dashboard/analytics/components/AnalyticsMetricCards.vue";
import AnalyticsPageLayout from "@/views/dashboard/analytics/components/AnalyticsPageLayout.vue";
import { defGoodsReportService } from "@/api/admin/goods_report";
import type { GoodsMonthReportItem, GoodsMonthReportSummaryResponse } from "@/rpc/admin/goods_report";
import router from "@/routers";
import { formatPrice } from "@/utils/utils";

/** 月报内容面板类型。 */
type ReportPanelType = "trend" | "summary";

const loading = ref(false);
const activePanel = ref<ReportPanelType>("trend");
const monthRange = ref<[string, string]>(getDefaultMonthRange());

const emptySummary = (): GoodsMonthReportSummaryResponse => ({
  viewCount: 0,
  collectCount: 0,
  cartCount: 0,
  orderCount: 0,
  payCount: 0,
  payGoodsNum: 0,
  payAmount: 0,
  cartConversionRate: 0,
  orderConversionRate: 0,
  payConversionRate: 0,
  payUnitPrice: 0
});

const report = reactive<{
  summary: GoodsMonthReportSummaryResponse;
  items: GoodsMonthReportItem[];
}>({
  summary: emptySummary(),
  items: []
});

const reportSummary = computed<GoodsMonthReportSummaryResponse>(() => {
  return report.summary ?? emptySummary();
});

/** 统一将接口返回的数值字段转成数字。 */
function normalizeNumber(value: unknown) {
  if (typeof value === "number") return Number.isFinite(value) ? value : 0;
  if (typeof value === "string") {
    const parsedValue = Number(value);
    return Number.isFinite(parsedValue) ? parsedValue : 0;
  }
  return 0;
}

/** 统一整理商品月报项，兼容蛇形和驼峰字段。 */
function normalizeReportItem(payload: Partial<GoodsMonthReportItem> | undefined): GoodsMonthReportItem {
  const source = (payload ?? {}) as Partial<GoodsMonthReportItem> & Record<string, unknown>;
  return {
    month: String(source.month ?? ""),
    viewCount: normalizeNumber(source.viewCount ?? source.view_count),
    collectCount: normalizeNumber(source.collectCount ?? source.collect_count),
    cartCount: normalizeNumber(source.cartCount ?? source.cart_count),
    orderCount: normalizeNumber(source.orderCount ?? source.order_count),
    payCount: normalizeNumber(source.payCount ?? source.pay_count),
    payGoodsNum: normalizeNumber(source.payGoodsNum ?? source.pay_goods_num),
    payAmount: normalizeNumber(source.payAmount ?? source.pay_amount),
    cartConversionRate: normalizeNumber(source.cartConversionRate ?? source.cart_conversion_rate),
    orderConversionRate: normalizeNumber(source.orderConversionRate ?? source.order_conversion_rate),
    payConversionRate: normalizeNumber(source.payConversionRate ?? source.pay_conversion_rate),
    payUnitPrice: normalizeNumber(source.payUnitPrice ?? source.pay_unit_price)
  };
}

/** 统一整理商品月报汇总响应，兼容网关包装结构。 */
function normalizeSummaryResponse(payload: unknown): GoodsMonthReportSummaryResponse {
  const source = ((payload as { data?: Partial<GoodsMonthReportSummaryResponse> } | undefined)?.data ??
    payload ??
    {}) as Partial<GoodsMonthReportSummaryResponse> & Record<string, unknown>;

  return {
    viewCount: normalizeNumber(source.viewCount ?? source.view_count),
    collectCount: normalizeNumber(source.collectCount ?? source.collect_count),
    cartCount: normalizeNumber(source.cartCount ?? source.cart_count),
    orderCount: normalizeNumber(source.orderCount ?? source.order_count),
    payCount: normalizeNumber(source.payCount ?? source.pay_count),
    payGoodsNum: normalizeNumber(source.payGoodsNum ?? source.pay_goods_num),
    payAmount: normalizeNumber(source.payAmount ?? source.pay_amount),
    cartConversionRate: normalizeNumber(source.cartConversionRate ?? source.cart_conversion_rate),
    orderConversionRate: normalizeNumber(source.orderConversionRate ?? source.order_conversion_rate),
    payConversionRate: normalizeNumber(source.payConversionRate ?? source.pay_conversion_rate),
    payUnitPrice: normalizeNumber(source.payUnitPrice ?? source.pay_unit_price)
  };
}

/** 统一整理商品月报名细列表响应。 */
function normalizeListResponse(payload: unknown): GoodsMonthReportItem[] {
  const source =
    (payload as { data?: { items?: Partial<GoodsMonthReportItem>[] }; items?: Partial<GoodsMonthReportItem>[] } | undefined) ??
    {};
  const rawItems = source.data?.items ?? source.items ?? [];
  return rawItems.map(item => normalizeReportItem(item));
}

/** 统一将千分比指标格式化成 1 位小数百分比。 */
function formatRatio(value: number) {
  return `${(value / 10).toFixed(1)}%`;
}

const metricItems = computed<AnalyticsMetricCardItem[]>(() => [
  {
    key: "viewCount",
    label: "浏览次数",
    labelTooltip: "按当前月报区间汇总商品详情页浏览次数。",
    value: String(reportSummary.value.viewCount),
    footLabel: "收藏次数",
    footValue: String(reportSummary.value.collectCount),
    color: "#2d6cdf",
    icon: Goods
  },
  {
    key: "cartCount",
    label: "加购件数",
    labelTooltip: "按当前月报区间汇总加入购物车的商品件数。",
    value: String(reportSummary.value.cartCount),
    footLabel: "浏览加购率",
    footValue: formatRatio(reportSummary.value.cartConversionRate),
    color: "#f08c2e",
    icon: Box
  },
  {
    key: "orderCount",
    label: "下单次数",
    labelTooltip: "按当前月报区间汇总下单次数。",
    value: String(reportSummary.value.orderCount),
    footLabel: "加购下单率",
    footValue: formatRatio(reportSummary.value.orderConversionRate),
    color: "#d9485f",
    icon: Tickets
  },
  {
    key: "payCount",
    label: "支付次数",
    labelTooltip: "按当前月报区间汇总支付次数。",
    value: String(reportSummary.value.payCount),
    footLabel: "浏览支付率",
    footValue: formatRatio(reportSummary.value.payConversionRate),
    color: "#15a87b",
    icon: TrendCharts
  },
  {
    key: "payAmount",
    label: "支付金额",
    labelTooltip: "按当前月报区间汇总支付成功金额。",
    value: `${formatPrice(reportSummary.value.payAmount)} 元`,
    footLabel: "支付件数",
    footValue: String(reportSummary.value.payGoodsNum),
    color: "#00838f",
    icon: Money
  }
]);

const chartOption = computed<ECOption>(() => ({
  color: ["#2d6cdf", "#15a87b", "#d9485f"],
  tooltip: {
    trigger: "axis",
    axisPointer: {
      type: "cross"
    }
  },
  legend: {
    bottom: 0,
    textStyle: {
      color: "#6d7b8f"
    }
  },
  grid: {
    top: 72,
    left: 20,
    right: 20,
    bottom: 44,
    containLabel: true
  },
  xAxis: {
    type: "category",
    data: report.items.map(item => item.month),
    axisLabel: {
      color: "#6d7b8f"
    },
    axisLine: {
      lineStyle: {
        color: "#dbe4ee"
      }
    }
  },
  yAxis: [
    {
      type: "value",
      name: "次数 / 件数",
      nameLocation: "end",
      nameGap: 28,
      axisLabel: {
        color: "#6d7b8f"
      },
      splitLine: {
        lineStyle: {
          color: "#edf2f7"
        }
      }
    },
    {
      type: "value",
      name: "金额（元）",
      nameLocation: "end",
      nameGap: 24,
      axisLabel: {
        color: "#6d7b8f"
      }
    }
  ],
  series: [
    {
      name: "浏览次数",
      type: "line",
      smooth: true,
      data: report.items.map(item => item.viewCount)
    },
    {
      name: "支付件数",
      type: "bar",
      barMaxWidth: 18,
      itemStyle: {
        borderRadius: [8, 8, 0, 0]
      },
      data: report.items.map(item => item.payGoodsNum)
    },
    {
      name: "支付金额（元）",
      type: "bar",
      yAxisIndex: 1,
      barMaxWidth: 18,
      itemStyle: {
        borderRadius: [8, 8, 0, 0]
      },
      data: report.items.map(item => Number(formatPrice(item.payAmount)))
    }
  ]
}));

/** 切换商品月报展示面板。 */
function handlePanelChange(panel: ReportPanelType) {
  activePanel.value = panel;
}

/** 按当前筛选条件加载商品月报汇总和列表。 */
async function loadData() {
  loading.value = true;
  try {
    const [startMonth, endMonth] = monthRange.value;
    const request = {
      startMonth,
      endMonth
    };
    const [summaryData, listData] = await Promise.all([
      defGoodsReportService.GoodsMonthReportSummary(request),
      defGoodsReportService.GoodsMonthReportList(request)
    ]);
    report.summary = {
      ...emptySummary(),
      ...normalizeSummaryResponse(summaryData)
    };
    report.items = normalizeListResponse(listData);
  } catch (error) {
    report.summary = emptySummary();
    report.items = [];
    throw error;
  } finally {
    loading.value = false;
  }
}

/** 图表点击后跳转到对应月份的商品日报。 */
function handleChartClick(event: ECElementEvent) {
  if (!event.name || typeof event.name !== "string") return;
  openDayReport(event.name);
}

/** 打开指定月份的商品日报页面。 */
function openDayReport(month: string) {
  router.push({
    path: "/report/goods/day",
    query: {
      startDate: dayjs(`${month}-01`).format("YYYY-MM-DD"),
      endDate: dayjs(`${month}-01`).endOf("month").format("YYYY-MM-DD"),
      periodLabel: month,
      source: "month-report"
    }
  });
}

/** 导出当前商品月报表格数据。 */
function handleExport() {
  if (!report.items.length) {
    ElMessage.warning("暂无可导出数据");
    return;
  }

  const headers = [
    "月份",
    "浏览次数",
    "收藏次数",
    "加购件数",
    "下单次数",
    "支付次数",
    "支付件数",
    "支付金额（元）",
    "浏览支付率"
  ];
  const rows = report.items.map(item => [
    item.month,
    item.viewCount,
    item.collectCount,
    item.cartCount,
    item.orderCount,
    item.payCount,
    item.payGoodsNum,
    formatPrice(item.payAmount),
    formatRatio(item.payConversionRate)
  ]);
  const summaryRow = [
    "合计",
    reportSummary.value.viewCount,
    reportSummary.value.collectCount,
    reportSummary.value.cartCount,
    reportSummary.value.orderCount,
    reportSummary.value.payCount,
    reportSummary.value.payGoodsNum,
    formatPrice(reportSummary.value.payAmount),
    formatRatio(reportSummary.value.payConversionRate)
  ];

  const csvContent = [headers, ...rows, summaryRow]
    .map(row => row.map(cell => `"${String(cell ?? "").replaceAll('"', '""')}"`).join(","))
    .join("\n");
  const blob = new Blob([`\ufeff${csvContent}`], { type: "application/vnd.ms-excel;charset=utf-8;" });
  const fileName = `商品月报_${monthRange.value[0]}_${monthRange.value[1]}.xls`;
  const blobUrl = window.URL.createObjectURL(blob);
  const downloadLink = document.createElement("a");
  downloadLink.style.display = "none";
  downloadLink.href = blobUrl;
  downloadLink.download = fileName;
  document.body.appendChild(downloadLink);
  downloadLink.click();
  document.body.removeChild(downloadLink);
  window.URL.revokeObjectURL(blobUrl);
}

/** 默认展示近 6 个月的商品月报。 */
function getDefaultMonthRange(): [string, string] {
  const currentMonth = dayjs();
  return [currentMonth.subtract(5, "month").format("YYYY-MM"), currentMonth.format("YYYY-MM")];
}

/** 初始化页面并拉取商品月报数据。 */
async function initializePage() {
  await loadData().catch(() => undefined);
}

initializePage();
</script>

<style scoped lang="scss">
.goods-month-report {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.report-toolbar {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.report-card {
  padding: 18px;
  border: 1px solid var(--admin-page-card-border);
  border-radius: var(--admin-page-radius);
  background: var(--admin-page-card-bg);
  box-shadow: var(--admin-page-shadow);
}

.report-card--tabs {
  overflow: hidden;
}

.report-card__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 16px;
}

.report-card__header--tabs {
  margin-bottom: 18px;
}

.report-card__tabs {
  display: flex;
  gap: 12px;
  align-items: center;
  min-width: 0;
}

.report-tab {
  min-width: 0;
  padding: 10px 16px;
  color: var(--admin-page-text-secondary);
  cursor: pointer;
  background: var(--admin-page-card-bg-soft);
  border: 1px solid var(--admin-page-card-border-soft);
  border-radius: var(--admin-page-radius);
  transition:
    color 0.2s ease,
    background-color 0.2s ease,
    border-color 0.2s ease,
    box-shadow 0.2s ease;
}

.report-tab--active {
  color: var(--el-color-primary);
  background: color-mix(in srgb, var(--el-color-primary) 12%, var(--admin-page-card-bg));
  border-color: color-mix(in srgb, var(--el-color-primary) 36%, var(--admin-page-card-border));
  box-shadow: 0 8px 18px rgb(64 158 255 / 12%);
}

.report-panel {
  min-width: 0;
}

.report-panel--chart {
  height: 360px;
}

.report-table {
  width: 100%;
}

.goods-month-report :deep(.summary-grid) {
  grid-template-columns: repeat(3, minmax(0, 1fr)) !important;
}

.goods-month-report :deep(.summary-card__meta) {
  align-items: flex-start;
}

.goods-month-report :deep(.summary-card__label),
.goods-month-report :deep(.summary-card__foot-label) {
  line-height: 1.5;
  white-space: normal;
}

.goods-month-report :deep(.summary-card__value) {
  word-break: break-word;
}

@media (max-width: 768px) {
  .report-toolbar {
    flex-direction: column;
    align-items: stretch;
  }

  .report-card__header--tabs {
    flex-direction: column;
    align-items: stretch;
  }

  .report-card__tabs {
    flex-wrap: wrap;
  }

  .goods-month-report :deep(.summary-grid) {
    grid-template-columns: minmax(0, 1fr) !important;
  }
}
</style>
