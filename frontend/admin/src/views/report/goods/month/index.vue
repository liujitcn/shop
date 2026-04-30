<template>
  <div v-loading="loading" class="goods-month-report">
    <PageLayout
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
            <el-table-column prop="view_count" label="浏览次数" min-width="110" align="right" />
            <el-table-column prop="collect_count" label="收藏次数" min-width="110" align="right" />
            <el-table-column prop="cart_count" label="加购件数" min-width="110" align="right" />
            <el-table-column prop="order_count" label="下单次数" min-width="110" align="right" />
            <el-table-column prop="pay_count" label="支付次数" min-width="110" align="right" />
            <el-table-column prop="pay_goods_num" label="支付件数" min-width="110" align="right" />
            <el-table-column prop="pay_amount" label="支付金额（元）" min-width="140" align="right">
              <template #default="{ row }">{{ formatPrice(row.pay_amount) }}</template>
            </el-table-column>
            <el-table-column prop="pay_conversion_rate" label="浏览支付率" min-width="120" align="right">
              <template #default="{ row }">{{ formatRatio(row.pay_conversion_rate) }}</template>
            </el-table-column>
            <el-table-column label="操作" width="120" fixed="right" align="center">
              <template #default="{ row }">
                <el-button link type="primary" @click="openDayReport(row.month)">查看日报</el-button>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </article>
    </PageLayout>
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
import MetricCards, { type MetricCardItem } from "@/views/dashboard/analytics/components/MetricCards.vue";
import PageLayout from "@/views/dashboard/analytics/components/PageLayout.vue";
import { defGoodsReportService } from "@/api/admin/goods_report";
import type { GoodsMonthReportItem, SummaryGoodsMonthReportResponse } from "@/rpc/admin/v1/goods_report";
import router from "@/routers";
import { formatPrice } from "@/utils/utils";

/** 月报内容面板类型。 */
type ReportPanelType = "trend" | "summary";

const loading = ref(false);
const activePanel = ref<ReportPanelType>("trend");
const monthRange = ref<[string, string]>(getDefaultMonthRange());

const emptySummary = (): SummaryGoodsMonthReportResponse => ({
  view_count: 0,
  collect_count: 0,
  cart_count: 0,
  order_count: 0,
  pay_count: 0,
  pay_goods_num: 0,
  pay_amount: 0,
  cart_conversion_rate: 0,
  order_conversion_rate: 0,
  pay_conversion_rate: 0,
  pay_unit_price: 0
});

const report = reactive<{
  summary: SummaryGoodsMonthReportResponse;
  items: GoodsMonthReportItem[];
}>({
  summary: emptySummary(),
  items: []
});

const reportSummary = computed<SummaryGoodsMonthReportResponse>(() => {
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
    view_count: normalizeNumber(source.view_count ?? source["viewCount"]),
    collect_count: normalizeNumber(source.collect_count ?? source["collectCount"]),
    cart_count: normalizeNumber(source.cart_count ?? source["cartCount"]),
    order_count: normalizeNumber(source.order_count ?? source["orderCount"]),
    pay_count: normalizeNumber(source.pay_count ?? source["payCount"]),
    pay_goods_num: normalizeNumber(source.pay_goods_num ?? source["payGoodsNum"]),
    pay_amount: normalizeNumber(source.pay_amount ?? source["payAmount"]),
    cart_conversion_rate: normalizeNumber(source.cart_conversion_rate ?? source["cartConversionRate"]),
    order_conversion_rate: normalizeNumber(source.order_conversion_rate ?? source["orderConversionRate"]),
    pay_conversion_rate: normalizeNumber(source.pay_conversion_rate ?? source["payConversionRate"]),
    pay_unit_price: normalizeNumber(source.pay_unit_price ?? source["payUnitPrice"])
  };
}

/** 统一整理商品月报汇总响应，兼容网关包装结构。 */
function normalizeSummaryResponse(payload: unknown): SummaryGoodsMonthReportResponse {
  const source = ((payload as { data?: Partial<SummaryGoodsMonthReportResponse> } | undefined)?.data ??
    payload ??
    {}) as Partial<SummaryGoodsMonthReportResponse> & Record<string, unknown>;

  return {
    view_count: normalizeNumber(source.view_count ?? source["viewCount"]),
    collect_count: normalizeNumber(source.collect_count ?? source["collectCount"]),
    cart_count: normalizeNumber(source.cart_count ?? source["cartCount"]),
    order_count: normalizeNumber(source.order_count ?? source["orderCount"]),
    pay_count: normalizeNumber(source.pay_count ?? source["payCount"]),
    pay_goods_num: normalizeNumber(source.pay_goods_num ?? source["payGoodsNum"]),
    pay_amount: normalizeNumber(source.pay_amount ?? source["payAmount"]),
    cart_conversion_rate: normalizeNumber(source.cart_conversion_rate ?? source["cartConversionRate"]),
    order_conversion_rate: normalizeNumber(source.order_conversion_rate ?? source["orderConversionRate"]),
    pay_conversion_rate: normalizeNumber(source.pay_conversion_rate ?? source["payConversionRate"]),
    pay_unit_price: normalizeNumber(source.pay_unit_price ?? source["payUnitPrice"])
  };
}

/** 统一整理商品月报名细列表响应。 */
function normalizeListResponse(payload: unknown): GoodsMonthReportItem[] {
  const source =
    (payload as
      | {
          data?: { goods_month_reports?: Partial<GoodsMonthReportItem>[]; items?: Partial<GoodsMonthReportItem>[] };
          goods_month_reports?: Partial<GoodsMonthReportItem>[];
          items?: Partial<GoodsMonthReportItem>[];
        }
      | undefined) ?? {};
  const rawItems = source.data?.goods_month_reports ?? source.goods_month_reports ?? source.data?.items ?? source.items ?? [];
  return rawItems.map(item => normalizeReportItem(item));
}

/** 统一将千分比指标格式化成 1 位小数百分比。 */
function formatRatio(value: number) {
  return `${(value / 10).toFixed(1)}%`;
}

const metricItems = computed<MetricCardItem[]>(() => [
  {
    key: "view_count",
    label: "浏览次数",
    labelTooltip: "按当前月报区间汇总商品详情页浏览次数。",
    value: String(reportSummary.value.view_count),
    footLabel: "收藏次数",
    footValue: String(reportSummary.value.collect_count),
    color: "#2d6cdf",
    icon: Goods
  },
  {
    key: "cart_count",
    label: "加购件数",
    labelTooltip: "按当前月报区间汇总加入购物车的商品件数。",
    value: String(reportSummary.value.cart_count),
    footLabel: "浏览加购率",
    footValue: formatRatio(reportSummary.value.cart_conversion_rate),
    color: "#f08c2e",
    icon: Box
  },
  {
    key: "order_count",
    label: "下单次数",
    labelTooltip: "按当前月报区间汇总下单次数。",
    value: String(reportSummary.value.order_count),
    footLabel: "加购下单率",
    footValue: formatRatio(reportSummary.value.order_conversion_rate),
    color: "#d9485f",
    icon: Tickets
  },
  {
    key: "pay_count",
    label: "支付次数",
    labelTooltip: "按当前月报区间汇总支付次数。",
    value: String(reportSummary.value.pay_count),
    footLabel: "浏览支付率",
    footValue: formatRatio(reportSummary.value.pay_conversion_rate),
    color: "#15a87b",
    icon: TrendCharts
  },
  {
    key: "pay_amount",
    label: "支付金额",
    labelTooltip: "按当前月报区间汇总支付成功金额。",
    value: `${formatPrice(reportSummary.value.pay_amount)} 元`,
    footLabel: "支付件数",
    footValue: String(reportSummary.value.pay_goods_num),
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
      data: report.items.map(item => item.view_count)
    },
    {
      name: "支付件数",
      type: "bar",
      barMaxWidth: 18,
      itemStyle: {
        borderRadius: [8, 8, 0, 0]
      },
      data: report.items.map(item => item.pay_goods_num)
    },
    {
      name: "支付金额（元）",
      type: "bar",
      yAxisIndex: 1,
      barMaxWidth: 18,
      itemStyle: {
        borderRadius: [8, 8, 0, 0]
      },
      data: report.items.map(item => Number(formatPrice(item.pay_amount)))
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
      start_month: startMonth,
      end_month: endMonth
    };
    const [summaryData, listData] = await Promise.all([
      defGoodsReportService.SummaryGoodsMonthReport(request),
      defGoodsReportService.ListGoodsMonthReports(request)
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
    item.view_count,
    item.collect_count,
    item.cart_count,
    item.order_count,
    item.pay_count,
    item.pay_goods_num,
    formatPrice(item.pay_amount),
    formatRatio(item.pay_conversion_rate)
  ]);
  const summaryRow = [
    "合计",
    reportSummary.value.view_count,
    reportSummary.value.collect_count,
    reportSummary.value.cart_count,
    reportSummary.value.order_count,
    reportSummary.value.pay_count,
    reportSummary.value.pay_goods_num,
    formatPrice(reportSummary.value.pay_amount),
    formatRatio(reportSummary.value.pay_conversion_rate)
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
