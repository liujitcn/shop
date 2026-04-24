<template>
  <div v-loading="loading" class="order-month-report">
    <AnalyticsPageLayout
      title="订单月报"
      description="按月份查看成交、退款、净销售额与客单价变化，支持下钻到日报。"
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
          <el-select v-model="filters.payType" clearable placeholder="支付方式" class="report-toolbar__select">
            <el-option v-for="item in payTypeOptions" :key="String(item.value)" :label="item.label" :value="Number(item.value)" />
          </el-select>
          <el-select v-model="filters.payChannel" clearable placeholder="支付渠道" class="report-toolbar__select">
            <el-option
              v-for="item in payChannelOptions"
              :key="String(item.value)"
              :label="item.label"
              :value="Number(item.value)"
            />
          </el-select>
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
              成交与退款趋势
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
                <el-button link type="primary" @click="openOrderDetail(row.month)">{{ row.month }}</el-button>
              </template>
            </el-table-column>
            <el-table-column prop="paidOrderCount" label="支付订单数" min-width="120" align="right" />
            <el-table-column prop="paidOrderAmount" label="支付金额（元）" min-width="150" align="right">
              <template #default="{ row }">{{ formatPrice(row.paidOrderAmount) }}</template>
            </el-table-column>
            <el-table-column prop="refundOrderCount" label="退款订单数" min-width="120" align="right" />
            <el-table-column prop="refundOrderAmount" label="退款金额（元）" min-width="150" align="right">
              <template #default="{ row }">{{ formatPrice(row.refundOrderAmount) }}</template>
            </el-table-column>
            <el-table-column prop="netOrderAmount" label="净销售额（元）" min-width="150" align="right">
              <template #default="{ row }">{{ formatPrice(row.netOrderAmount) }}</template>
            </el-table-column>
            <el-table-column prop="paidUserCount" label="支付用户数" min-width="120" align="right" />
            <el-table-column prop="goodsCount" label="商品件数" min-width="120" align="right" />
            <el-table-column prop="customerUnitPrice" label="客单价（元）" min-width="130" align="right">
              <template #default="{ row }">{{ formatPrice(row.customerUnitPrice) }}</template>
            </el-table-column>
            <el-table-column label="操作" width="120" fixed="right" align="center">
              <template #default="{ row }">
                <el-button link type="primary" @click="openOrderDetail(row.month)">查看明细</el-button>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </article>
    </AnalyticsPageLayout>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from "vue";
import dayjs from "dayjs";
import { ElMessage } from "element-plus";
import { Box, CreditCard, Goods, Money, RefreshLeft, User } from "@element-plus/icons-vue";
import type { ECElementEvent } from "echarts/core";
import ECharts from "@/components/ECharts/index.vue";
import type { ECOption } from "@/components/ECharts/config";
import type { EnumProps } from "@/components/ProTable/interface";
import AnalyticsMetricCards, {
  type AnalyticsMetricCardItem
} from "@/views/dashboard/analytics/components/AnalyticsMetricCards.vue";
import AnalyticsPageLayout from "@/views/dashboard/analytics/components/AnalyticsPageLayout.vue";
import { defOrderReportService } from "@/api/admin/order_report";
import type { OrderMonthReportItem, OrderMonthReportSummaryResponse } from "@/rpc/admin/order_report";
import router from "@/routers";
import { buildDictEnum } from "@/utils/proTable";
import { formatPrice } from "@/utils/utils";

defineOptions({
  name: "OrderMonthReport"
});

/** 月报内容面板类型。 */
type ReportPanelType = "trend" | "summary";

const loading = ref(false);
const activePanel = ref<ReportPanelType>("trend");
const monthRange = ref<[string, string]>(getDefaultMonthRange());
const payTypeOptions = ref<EnumProps[]>([]);
const payChannelOptions = ref<EnumProps[]>([]);
const filters = reactive({
  payType: undefined as number | undefined,
  payChannel: undefined as number | undefined
});

const emptySummary = (): OrderMonthReportSummaryResponse => ({
  paidOrderCount: 0,
  paidOrderAmount: 0,
  refundOrderCount: 0,
  refundOrderAmount: 0,
  netOrderAmount: 0,
  paidUserCount: 0,
  goodsCount: 0,
  customerUnitPrice: 0
});

const report = reactive<{
  summary: OrderMonthReportSummaryResponse;
  items: OrderMonthReportItem[];
}>({
  summary: emptySummary(),
  items: []
});

const reportSummary = computed<OrderMonthReportSummaryResponse>(() => {
  return report.summary ?? emptySummary();
});

function normalizeNumber(value: unknown) {
  if (typeof value === "number") return Number.isFinite(value) ? value : 0;
  if (typeof value === "string") {
    const parsedValue = Number(value);
    return Number.isFinite(parsedValue) ? parsedValue : 0;
  }
  return 0;
}

function normalizeReportItem(payload: Partial<OrderMonthReportItem> | undefined): OrderMonthReportItem {
  const source = (payload ?? {}) as Partial<OrderMonthReportItem> & Record<string, unknown>;
  return {
    month: String(source.month ?? ""),
    paidOrderCount: normalizeNumber(source.paidOrderCount ?? source.paid_order_count),
    paidOrderAmount: normalizeNumber(source.paidOrderAmount ?? source.paid_order_amount),
    refundOrderCount: normalizeNumber(source.refundOrderCount ?? source.refund_order_count),
    refundOrderAmount: normalizeNumber(source.refundOrderAmount ?? source.refund_order_amount),
    netOrderAmount: normalizeNumber(source.netOrderAmount ?? source.net_order_amount),
    paidUserCount: normalizeNumber(source.paidUserCount ?? source.paid_user_count),
    goodsCount: normalizeNumber(source.goodsCount ?? source.goods_count),
    customerUnitPrice: normalizeNumber(source.customerUnitPrice ?? source.customer_unit_price)
  };
}

function normalizeSummaryResponse(payload: unknown): OrderMonthReportSummaryResponse {
  const source = ((payload as { data?: Partial<OrderMonthReportSummaryResponse> } | undefined)?.data ??
    payload ??
    {}) as Partial<OrderMonthReportSummaryResponse> & Record<string, unknown>;

  return {
    paidOrderCount: normalizeNumber(source.paidOrderCount ?? source.paid_order_count),
    paidOrderAmount: normalizeNumber(source.paidOrderAmount ?? source.paid_order_amount),
    refundOrderCount: normalizeNumber(source.refundOrderCount ?? source.refund_order_count),
    refundOrderAmount: normalizeNumber(source.refundOrderAmount ?? source.refund_order_amount),
    netOrderAmount: normalizeNumber(source.netOrderAmount ?? source.net_order_amount),
    paidUserCount: normalizeNumber(source.paidUserCount ?? source.paid_user_count),
    goodsCount: normalizeNumber(source.goodsCount ?? source.goods_count),
    customerUnitPrice: normalizeNumber(source.customerUnitPrice ?? source.customer_unit_price)
  };
}

function normalizeListResponse(payload: unknown): OrderMonthReportItem[] {
  const source =
    (payload as { data?: { items?: Partial<OrderMonthReportItem>[] }; items?: Partial<OrderMonthReportItem>[] } | undefined) ??
    {};
  const rawItems = source.data?.items ?? source.items ?? [];
  return rawItems.map(item => normalizeReportItem(item));
}

const metricItems = computed<AnalyticsMetricCardItem[]>(() => [
  {
    key: "paidOrderCount",
    label: "支付订单数",
    labelTooltip: "按支付成功时间统计的订单数量。",
    value: String(reportSummary.value.paidOrderCount),
    footLabel: "支付用户数",
    footValue: String(reportSummary.value.paidUserCount),
    color: "#d9485f",
    icon: CreditCard
  },
  {
    key: "paidOrderAmount",
    label: "支付金额",
    labelTooltip: "按支付成功时间汇总的实付金额。",
    value: `${formatPrice(reportSummary.value.paidOrderAmount)} 元`,
    footLabel: "商品件数",
    footValue: String(reportSummary.value.goodsCount),
    color: "#f08c2e",
    icon: Money
  },
  {
    key: "refundOrderAmount",
    label: "退款金额",
    labelTooltip: "按退款成功时间汇总的退款金额。",
    value: `${formatPrice(reportSummary.value.refundOrderAmount)} 元`,
    footLabel: "退款订单数",
    footValue: String(reportSummary.value.refundOrderCount),
    color: "#2d6cdf",
    icon: RefreshLeft
  },
  {
    key: "netOrderAmount",
    label: "净销售额",
    labelTooltip: "支付金额减去退款金额后的净额。",
    value: `${formatPrice(reportSummary.value.netOrderAmount)} 元`,
    footLabel: "客单价",
    footValue: `${formatPrice(reportSummary.value.customerUnitPrice)} 元`,
    color: "#1f9d55",
    icon: Box
  },
  {
    key: "paidUserCount",
    label: "支付用户数",
    labelTooltip: "统计区间内支付成功的用户数量。",
    value: String(reportSummary.value.paidUserCount),
    footLabel: "商品件数",
    footValue: String(reportSummary.value.goodsCount),
    color: "#7c4dff",
    icon: User
  },
  {
    key: "customerUnitPrice",
    label: "客单价",
    labelTooltip: "支付金额除以支付订单数。",
    value: `${formatPrice(reportSummary.value.customerUnitPrice)} 元`,
    footLabel: "支付订单数",
    footValue: String(reportSummary.value.paidOrderCount),
    color: "#00838f",
    icon: Goods
  }
]);

const chartOption = computed<ECOption>(() => ({
  color: ["#f08c2e", "#2d6cdf", "#d9485f"],
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
    // 顶部额外留白，避免坐标轴名称与卡片标题区发生视觉重叠，同时保留轴标题在顶部展示。
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
      name: "金额（元）",
      nameLocation: "end",
      nameGap: 28,
      axisLabel: {
        color: "#6d7b8f",
        formatter: (value: number) => Number(value).toFixed(0)
      },
      splitLine: {
        lineStyle: {
          color: "#edf2f7"
        }
      }
    },
    {
      type: "value",
      name: "订单数",
      nameLocation: "end",
      nameGap: 24,
      axisLabel: {
        color: "#6d7b8f"
      }
    }
  ],
  series: [
    {
      name: "支付金额",
      type: "bar",
      barMaxWidth: 18,
      itemStyle: {
        borderRadius: [8, 8, 0, 0]
      },
      data: report.items.map(item => Number(formatPrice(item.paidOrderAmount)))
    },
    {
      name: "退款金额",
      type: "bar",
      barMaxWidth: 18,
      itemStyle: {
        borderRadius: [8, 8, 0, 0]
      },
      data: report.items.map(item => Number(formatPrice(item.refundOrderAmount)))
    },
    {
      name: "支付订单数",
      type: "line",
      smooth: true,
      yAxisIndex: 1,
      data: report.items.map(item => item.paidOrderCount)
    }
  ]
}));

/** 切换月报展示面板。 */
function handlePanelChange(panel: ReportPanelType) {
  activePanel.value = panel;
}

async function loadData() {
  loading.value = true;
  try {
    const [startMonth, endMonth] = monthRange.value;
    const request = {
      startMonth,
      endMonth,
      payType: filters.payType ?? 0,
      payChannel: filters.payChannel ?? 0
    };
    const [summaryData, listData] = await Promise.all([
      defOrderReportService.OrderMonthReportSummary(request),
      defOrderReportService.OrderMonthReportList(request)
    ]);
    const summary = normalizeSummaryResponse(summaryData);
    const items = normalizeListResponse(listData);
    report.summary = {
      ...emptySummary(),
      ...summary,
      netOrderAmount: summary.netOrderAmount ?? summary.paidOrderAmount - summary.refundOrderAmount
    };
    report.items = items.map(item => ({
      ...item,
      netOrderAmount: item.netOrderAmount ?? item.paidOrderAmount - item.refundOrderAmount
    }));
  } catch (error) {
    report.summary = emptySummary();
    report.items = [];
    throw error;
  } finally {
    loading.value = false;
  }
}

function handleChartClick(event: ECElementEvent) {
  if (!event.name || typeof event.name !== "string") return;
  openOrderDetail(event.name);
}

function openOrderDetail(month: string) {
  router.push({
    path: "/report/order/day",
    query: {
      startDate: dayjs(`${month}-01`).format("YYYY-MM-DD"),
      endDate: dayjs(`${month}-01`).endOf("month").format("YYYY-MM-DD"),
      payType: filters.payType,
      payChannel: filters.payChannel,
      source: "month-report",
      periodLabel: month
    }
  });
}

function handleExport() {
  if (!report.items.length) {
    ElMessage.warning("暂无可导出数据");
    return;
  }

  const headers = [
    "月份",
    "支付订单数",
    "支付金额（元）",
    "退款订单数",
    "退款金额（元）",
    "净销售额（元）",
    "支付用户数",
    "商品件数",
    "客单价（元）"
  ];
  const rows = report.items.map(item => [
    item.month,
    item.paidOrderCount,
    formatPrice(item.paidOrderAmount),
    item.refundOrderCount,
    formatPrice(item.refundOrderAmount),
    formatPrice(item.netOrderAmount),
    item.paidUserCount,
    item.goodsCount,
    formatPrice(item.customerUnitPrice)
  ]);
  const summaryRow = [
    "合计",
    reportSummary.value.paidOrderCount,
    formatPrice(reportSummary.value.paidOrderAmount),
    reportSummary.value.refundOrderCount,
    formatPrice(reportSummary.value.refundOrderAmount),
    formatPrice(reportSummary.value.netOrderAmount),
    reportSummary.value.paidUserCount,
    reportSummary.value.goodsCount,
    formatPrice(reportSummary.value.customerUnitPrice)
  ];

  const csvContent = [headers, ...rows, summaryRow]
    .map(row => row.map(cell => `"${String(cell ?? "").replaceAll('"', '""')}"`).join(","))
    .join("\n");
  const blob = new Blob([`\ufeff${csvContent}`], { type: "application/vnd.ms-excel;charset=utf-8;" });
  const fileName = `订单月报_${monthRange.value[0]}_${monthRange.value[1]}.xls`;
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

function getDefaultMonthRange(): [string, string] {
  const currentMonth = dayjs();
  return [currentMonth.subtract(5, "month").format("YYYY-MM"), currentMonth.format("YYYY-MM")];
}

async function loadFilterOptions() {
  const [payTypeEnum, payChannelEnum] = await Promise.all([buildDictEnum("order_pay_type"), buildDictEnum("order_pay_channel")]);
  payTypeOptions.value = payTypeEnum.data;
  payChannelOptions.value = payChannelEnum.data;
}

async function initializePage() {
  await loadFilterOptions().catch(() => {
    payTypeOptions.value = [];
    payChannelOptions.value = [];
  });
  await loadData().catch(() => undefined);
}

initializePage();
</script>

<style scoped lang="scss">
.order-month-report {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.report-toolbar {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.report-toolbar__select {
  width: 150px;
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

.order-month-report :deep(.summary-grid) {
  grid-template-columns: repeat(3, minmax(0, 1fr)) !important;
}

.order-month-report :deep(.summary-card__meta) {
  align-items: flex-start;
}

.order-month-report :deep(.summary-card__label),
.order-month-report :deep(.summary-card__foot-label) {
  line-height: 1.5;
  white-space: normal;
}

.order-month-report :deep(.summary-card__value) {
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

  .order-month-report :deep(.summary-grid) {
    grid-template-columns: minmax(0, 1fr) !important;
  }
}
</style>
