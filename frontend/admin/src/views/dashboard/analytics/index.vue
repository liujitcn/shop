<template>
  <div class="analytics-page">
    <el-card class="analytics-card analytics-card--summary" shadow="never">
      <div class="analytics-toolbar">
        <div>
          <h2 class="analytics-title">经营数据</h2>
          <p class="analytics-desc">按时间维度查看用户、商品、订单与销售额的汇总和趋势变化。</p>
        </div>
        <span class="analytics-period">{{ activeTimeLabel }}</span>

        <el-tabs v-model="activeTimeType" class="analytics-tabs">
          <el-tab-pane v-for="item in timeOptions" :key="item.value" :label="item.label" :name="item.value" />
        </el-tabs>
      </div>

      <div class="summary-grid">
        <article v-for="item in summaryCards" :key="item.key" class="summary-card" :style="{ '--card-accent': item.color }">
          <div class="summary-card__meta">
            <div>
              <span class="summary-card__label">{{ item.label }}</span>
              <div class="summary-card__value">{{ item.value }}</div>
            </div>
            <div class="summary-card__icon">
              <el-icon :size="20">
                <component :is="item.icon" />
              </el-icon>
            </div>
          </div>
          <div class="summary-card__foot">
            <span>{{ item.footLabel }}</span>
            <strong>{{ item.newValue }}</strong>
          </div>
        </article>
      </div>
    </el-card>

    <section class="chart-grid">
      <OrderBarChart :time-type="activeTimeType" />
      <SalesBarChart :time-type="activeTimeType" />
      <GoodsPieChart :time-type="activeTimeType" />
      <OrderPieChart :time-type="activeTimeType" />
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from "vue";
import OrderBarChart from "./components/OrderBarChart.vue";
import SalesBarChart from "./components/SalesBarChart.vue";
import GoodsPieChart from "./components/GoodsPieChart.vue";
import OrderPieChart from "./components/OrderPieChart.vue";
import { defAnalyticsService } from "@/api/admin/analytics";
import type { AnalyticsCountResponse, AnalyticsTimeType } from "@/rpc/admin/analytics";
import { AnalyticsTimeType as AnalyticsTimeTypeEnum } from "@/rpc/admin/analytics";
import { formatPrice } from "@/utils/utils";

defineOptions({
  name: "Analytics",
  inheritAttrs: false
});

/** 时间维度选项，需与后端枚举定义保持一致。 */
const timeOptions = [
  { label: "本周", value: AnalyticsTimeTypeEnum.WEEK },
  { label: "本月", value: AnalyticsTimeTypeEnum.MONTH },
  { label: "本年", value: AnalyticsTimeTypeEnum.YEAR }
];

const activeTimeType = ref<AnalyticsTimeType>(AnalyticsTimeTypeEnum.WEEK);

const activeTimeLabel = computed(() => {
  return timeOptions.find(item => item.value === activeTimeType.value)?.label ?? "当前";
});

const dashboardCountUser = reactive<AnalyticsCountResponse>({
  /** 新增数量 */
  newNum: 0,
  /** 总数量 */
  totalNum: 0
});

const dashboardCountGoods = reactive<AnalyticsCountResponse>({
  /** 新增数量 */
  newNum: 0,
  /** 总数量 */
  totalNum: 0
});

const dashboardCountOrder = reactive<AnalyticsCountResponse>({
  /** 新增数量 */
  newNum: 0,
  /** 总数量 */
  totalNum: 0
});

const dashboardCountSale = reactive<AnalyticsCountResponse>({
  /** 新增数量 */
  newNum: 0,
  /** 总数量 */
  totalNum: 0
});

/** 顶部概览卡片配置项。 */
interface SummaryCardItem {
  /** 唯一键。 */
  key: string;
  /** 卡片标题。 */
  label: string;
  /** 主值。 */
  value: string;
  /** 辅助值。 */
  newValue: string;
  /** 底部说明。 */
  footLabel: string;
  /** 对应图标组件。 */
  icon: any;
  /** 强调色。 */
  color: string;
}

/** 顶部概览卡片配置。 */
const summaryCards = computed<SummaryCardItem[]>(() => [
  {
    key: "user",
    label: "用户总数",
    value: String(dashboardCountUser.totalNum),
    newValue: String(dashboardCountUser.newNum),
    footLabel: `${activeTimeLabel.value}新增`,
    icon: User,
    color: "#2d6cdf"
  },
  {
    key: "goods",
    label: "商品总数",
    value: String(dashboardCountGoods.totalNum),
    newValue: String(dashboardCountGoods.newNum),
    footLabel: `${activeTimeLabel.value}新增`,
    icon: Goods,
    color: "#15a87b"
  },
  {
    key: "order",
    label: "订单总量",
    value: String(dashboardCountOrder.totalNum),
    newValue: String(dashboardCountOrder.newNum),
    footLabel: `${activeTimeLabel.value}新增`,
    icon: Document,
    color: "#f08c2e"
  },
  {
    key: "sale",
    label: "销售总额",
    value: formatPrice(dashboardCountSale.totalNum),
    newValue: formatPrice(dashboardCountSale.newNum),
    footLabel: activeTimeLabel.value,
    icon: Wallet,
    color: "#d9485f"
  }
]);

/**
 * 按当前时间维度加载顶部汇总数据。
 */
async function loadSummaryData(timeType: AnalyticsTimeType) {
  const [user, goods, order, sale] = await Promise.all([
    defAnalyticsService.AnalyticsCountUser({ timeType }),
    defAnalyticsService.AnalyticsCountGoods({ timeType }),
    defAnalyticsService.AnalyticsCountOrder({ timeType }),
    defAnalyticsService.AnalyticsCountSale({ timeType })
  ]);

  Object.assign(dashboardCountUser, user);
  Object.assign(dashboardCountGoods, goods);
  Object.assign(dashboardCountOrder, order);
  Object.assign(dashboardCountSale, sale);
}

watch(
  activeTimeType,
  value => {
    loadSummaryData(value);
  },
  { immediate: true }
);
</script>

<style scoped lang="scss">
.analytics-page {
  padding: 20px;
  background: #f5f7fb;
}

.analytics-card {
  border: 1px solid #e5eaf1;
  border-radius: 16px;
  box-shadow: 0 8px 24px rgb(15 23 42 / 4%);
}

:deep(.analytics-card .el-card__body) {
  padding: 18px;
}

.analytics-card--summary {
  margin-bottom: 16px;
}

.analytics-toolbar {
  display: flex;
  gap: 24px;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}

.analytics-title {
  margin: 0;
  font-size: 22px;
  font-weight: 700;
  color: #1f2937;
}

.analytics-desc {
  max-width: 560px;
  margin: 8px 0 0;
  color: #64748b;
  line-height: 1.7;
}

.analytics-period {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 72px;
  height: 32px;
  padding: 0 12px;
  font-size: 12px;
  font-weight: 600;
  color: #2563eb;
  background: #eff6ff;
  border: 1px solid #dbeafe;
  border-radius: 999px;
}

.analytics-tabs {
  min-width: 280px;
}

:deep(.analytics-tabs .el-tabs__nav) {
  padding: 4px;
  background: #f8fafc;
  border: 1px solid #e8edf4;
  border-radius: 10px;
}

:deep(.analytics-tabs .el-tabs__item) {
  height: 32px;
  padding: 0 14px;
  border-radius: 8px;
  color: #64748b;
}

:deep(.analytics-tabs .el-tabs__item.is-active) {
  color: #1f2937;
  background: #fff;
}

.summary-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}

.summary-card {
  padding: 14px;
  border: 1px solid #e8edf4;
  border-radius: 12px;
  background: #fff;
}

.summary-card__meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.summary-card__label {
  display: block;
  font-size: 14px;
  color: #64748b;
}

.summary-card__icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  color: #fff;
  border-radius: 10px;
  background: var(--card-accent);
}

.summary-card__value {
  margin: 8px 0 0;
  font-size: 24px;
  font-weight: 700;
  color: #1f2937;
}

.summary-card__foot {
  display: flex;
  gap: 8px;
  align-items: center;
  font-size: 13px;
  color: #94a3b8;
  margin-top: 10px;
  padding-top: 10px;
  border-top: 1px solid #eef2f7;
}

.summary-card__foot strong {
  color: var(--card-accent);
}

.chart-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
  margin-top: 16px;
}

:deep(.analytics-tabs .el-tabs__header) {
  margin: 0;
}

:deep(.analytics-tabs .el-tabs__nav-wrap::after) {
  display: none;
}

@media (max-width: 1200px) {
  .summary-grid,
  .chart-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 768px) {
  .analytics-page {
    padding: 16px;
  }

  .analytics-toolbar {
    flex-direction: column;
    align-items: stretch;
  }

  .analytics-tabs {
    width: 100%;
    min-width: 0;
  }

  .summary-grid,
  .chart-grid {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
