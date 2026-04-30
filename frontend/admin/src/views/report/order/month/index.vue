<template>
  <div v-loading="loading" class="order-month-report">
    <PageLayout
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
        <MetricCards :items="metricItems" />
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
            <el-table-column prop="paid_order_count" label="支付订单数" min-width="120" align="right" />
            <el-table-column prop="paid_order_amount" label="支付金额（元）" min-width="150" align="right">
              <template #default="{ row }">{{ formatPrice(row.paid_order_amount) }}</template>
            </el-table-column>
            <el-table-column prop="refund_order_count" label="退款订单数" min-width="120" align="right" />
            <el-table-column prop="refund_order_amount" label="退款金额（元）" min-width="150" align="right">
              <template #default="{ row }">{{ formatPrice(row.refund_order_amount) }}</template>
            </el-table-column>
            <el-table-column prop="net_order_amount" label="净销售额（元）" min-width="150" align="right">
              <template #default="{ row }">{{ formatPrice(row.net_order_amount) }}</template>
            </el-table-column>
            <el-table-column prop="paid_user_count" label="支付用户数" min-width="120" align="right" />
            <el-table-column prop="goods_count" label="商品件数" min-width="120" align="right" />
            <el-table-column prop="customer_unit_price" label="客单价（元）" min-width="130" align="right">
              <template #default="{ row }">{{ formatPrice(row.customer_unit_price) }}</template>
            </el-table-column>
            <el-table-column label="操作" width="120" fixed="right" align="center">
              <template #default="{ row }">
                <el-button link type="primary" @click="openOrderDetail(row.month)">查看明细</el-button>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </article>
    </PageLayout>
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
import MetricCards, { type MetricCardItem } from "@/views/dashboard/analytics/components/MetricCards.vue";
import PageLayout from "@/views/dashboard/analytics/components/PageLayout.vue";
import { defOrderReportService } from "@/api/admin/order_report";
import type { OrderMonthReportItem, SummaryOrderMonthReportResponse } from "@/rpc/admin/v1/order_report";
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

const emptySummary = (): SummaryOrderMonthReportResponse => ({
  paid_order_count: 0,
  paid_order_amount: 0,
  refund_order_count: 0,
  refund_order_amount: 0,
  net_order_amount: 0,
  paid_user_count: 0,
  goods_count: 0,
  customer_unit_price: 0
});

const report = reactive<{
  summary: SummaryOrderMonthReportResponse;
  items: OrderMonthReportItem[];
}>({
  summary: emptySummary(),
  items: []
});

const reportSummary = computed<SummaryOrderMonthReportResponse>(() => {
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

/** 统一整理月报明细项，兼容蛇形和驼峰字段。 */
function normalizeReportItem(payload: Partial<OrderMonthReportItem> | undefined): OrderMonthReportItem {
  const source = (payload ?? {}) as Partial<OrderMonthReportItem> & Record<string, unknown>;
  return {
    month: String(source.month ?? ""),
    paid_order_count: normalizeNumber(source.paid_order_count ?? source["paidOrderCount"]),
    paid_order_amount: normalizeNumber(source.paid_order_amount ?? source["paidOrderAmount"]),
    refund_order_count: normalizeNumber(source.refund_order_count ?? source["refundOrderCount"]),
    refund_order_amount: normalizeNumber(source.refund_order_amount ?? source["refundOrderAmount"]),
    net_order_amount: normalizeNumber(source.net_order_amount ?? source["netOrderAmount"]),
    paid_user_count: normalizeNumber(source.paid_user_count ?? source["paidUserCount"]),
    goods_count: normalizeNumber(source.goods_count ?? source["goodsCount"]),
    customer_unit_price: normalizeNumber(source.customer_unit_price ?? source["customerUnitPrice"])
  };
}

/** 统一整理月报汇总响应，兼容网关包装结构。 */
function normalizeSummaryResponse(payload: unknown): SummaryOrderMonthReportResponse {
  const source = ((payload as { data?: Partial<SummaryOrderMonthReportResponse> } | undefined)?.data ??
    payload ??
    {}) as Partial<SummaryOrderMonthReportResponse> & Record<string, unknown>;

  return {
    paid_order_count: normalizeNumber(source.paid_order_count ?? source["paidOrderCount"]),
    paid_order_amount: normalizeNumber(source.paid_order_amount ?? source["paidOrderAmount"]),
    refund_order_count: normalizeNumber(source.refund_order_count ?? source["refundOrderCount"]),
    refund_order_amount: normalizeNumber(source.refund_order_amount ?? source["refundOrderAmount"]),
    net_order_amount: normalizeNumber(source.net_order_amount ?? source["netOrderAmount"]),
    paid_user_count: normalizeNumber(source.paid_user_count ?? source["paidUserCount"]),
    goods_count: normalizeNumber(source.goods_count ?? source["goodsCount"]),
    customer_unit_price: normalizeNumber(source.customer_unit_price ?? source["customerUnitPrice"])
  };
}

/** 统一整理月报明细列表响应。 */
function normalizeListResponse(payload: unknown): OrderMonthReportItem[] {
  const source =
    (payload as
      | {
          data?: { order_month_reports?: Partial<OrderMonthReportItem>[]; items?: Partial<OrderMonthReportItem>[] };
          order_month_reports?: Partial<OrderMonthReportItem>[];
          items?: Partial<OrderMonthReportItem>[];
        }
      | undefined) ?? {};
  const rawItems = source.data?.order_month_reports ?? source.order_month_reports ?? source.data?.items ?? source.items ?? [];
  return rawItems.map(item => normalizeReportItem(item));
}

const metricItems = computed<MetricCardItem[]>(() => [
  {
    key: "paid_order_count",
    label: "支付订单数",
    labelTooltip: "按支付成功时间统计的订单数量。",
    value: String(reportSummary.value.paid_order_count),
    footLabel: "支付用户数",
    footValue: String(reportSummary.value.paid_user_count),
    color: "#d9485f",
    icon: CreditCard
  },
  {
    key: "paid_order_amount",
    label: "支付金额",
    labelTooltip: "按支付成功时间汇总的实付金额。",
    value: `${formatPrice(reportSummary.value.paid_order_amount)} 元`,
    footLabel: "商品件数",
    footValue: String(reportSummary.value.goods_count),
    color: "#f08c2e",
    icon: Money
  },
  {
    key: "refund_order_amount",
    label: "退款金额",
    labelTooltip: "按退款成功时间汇总的退款金额。",
    value: `${formatPrice(reportSummary.value.refund_order_amount)} 元`,
    footLabel: "退款订单数",
    footValue: String(reportSummary.value.refund_order_count),
    color: "#2d6cdf",
    icon: RefreshLeft
  },
  {
    key: "net_order_amount",
    label: "净销售额",
    labelTooltip: "支付金额减去退款金额后的净额。",
    value: `${formatPrice(reportSummary.value.net_order_amount)} 元`,
    footLabel: "客单价",
    footValue: `${formatPrice(reportSummary.value.customer_unit_price)} 元`,
    color: "#1f9d55",
    icon: Box
  },
  {
    key: "paid_user_count",
    label: "支付用户数",
    labelTooltip: "统计区间内支付成功的用户数量。",
    value: String(reportSummary.value.paid_user_count),
    footLabel: "商品件数",
    footValue: String(reportSummary.value.goods_count),
    color: "#7c4dff",
    icon: User
  },
  {
    key: "customer_unit_price",
    label: "客单价",
    labelTooltip: "支付金额除以支付订单数。",
    value: `${formatPrice(reportSummary.value.customer_unit_price)} 元`,
    footLabel: "支付订单数",
    footValue: String(reportSummary.value.paid_order_count),
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
      data: report.items.map(item => Number(formatPrice(item.paid_order_amount)))
    },
    {
      name: "退款金额",
      type: "bar",
      barMaxWidth: 18,
      itemStyle: {
        borderRadius: [8, 8, 0, 0]
      },
      data: report.items.map(item => Number(formatPrice(item.refund_order_amount)))
    },
    {
      name: "支付订单数",
      type: "line",
      smooth: true,
      yAxisIndex: 1,
      data: report.items.map(item => item.paid_order_count)
    }
  ]
}));

/** 切换月报展示面板。 */
function handlePanelChange(panel: ReportPanelType) {
  activePanel.value = panel;
}

/** 按当前筛选条件加载月报汇总和列表。 */
async function loadData() {
  loading.value = true;
  try {
    const [startMonth, endMonth] = monthRange.value;
    const request = {
      start_month: startMonth,
      end_month: endMonth,
      pay_type: filters.payType ?? 0,
      pay_channel: filters.payChannel ?? 0
    };
    const [summaryData, listData] = await Promise.all([
      defOrderReportService.SummaryOrderMonthReport(request),
      defOrderReportService.ListOrderMonthReports(request)
    ]);
    const summary = normalizeSummaryResponse(summaryData);
    const items = normalizeListResponse(listData);
    report.summary = {
      ...emptySummary(),
      ...summary,
      net_order_amount: summary.net_order_amount ?? summary.paid_order_amount - summary.refund_order_amount
    };
    report.items = items.map(item => ({
      ...item,
      net_order_amount: item.net_order_amount ?? item.paid_order_amount - item.refund_order_amount
    }));
  } catch (error) {
    report.summary = emptySummary();
    report.items = [];
    throw error;
  } finally {
    loading.value = false;
  }
}

/** 图表点击后跳转到日报页查看对应月份明细。 */
function handleChartClick(event: ECElementEvent) {
  if (!event.name || typeof event.name !== "string") return;
  openOrderDetail(event.name);
}

/** 跳转到日报页查看指定月份的订单明细。 */
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

/** 导出当前月报表格数据。 */
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
    item.paid_order_count,
    formatPrice(item.paid_order_amount),
    item.refund_order_count,
    formatPrice(item.refund_order_amount),
    formatPrice(item.net_order_amount),
    item.paid_user_count,
    item.goods_count,
    formatPrice(item.customer_unit_price)
  ]);
  const summaryRow = [
    "合计",
    reportSummary.value.paid_order_count,
    formatPrice(reportSummary.value.paid_order_amount),
    reportSummary.value.refund_order_count,
    formatPrice(reportSummary.value.refund_order_amount),
    formatPrice(reportSummary.value.net_order_amount),
    reportSummary.value.paid_user_count,
    reportSummary.value.goods_count,
    formatPrice(reportSummary.value.customer_unit_price)
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
