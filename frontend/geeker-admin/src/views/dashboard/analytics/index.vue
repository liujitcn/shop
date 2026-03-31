<template>
  <div class="analytics-page">
    <section class="hero-card">
      <div class="hero-card__head">
        <div>
          <h2 class="hero-card__title">{{ activeTimeLabel }}经营数据</h2>
          <p class="hero-card__desc">展示用户、商品、订单和销售额汇总，以及对应的趋势和分布情况。</p>
        </div>

        <el-tabs v-model="activeTimeType" class="hero-card__tabs">
          <el-tab-pane v-for="item in timeOptions" :key="item.value" :label="item.label" :name="item.value" />
        </el-tabs>
      </div>

      <div class="hero-card__summary">
        <article v-for="item in summaryCards" :key="item.key" class="summary-card" :style="{ '--card-accent': item.color }">
          <div class="summary-card__meta">
            <span class="summary-card__label">{{ item.label }}</span>
            <div class="summary-card__icon">
              <el-icon :size="20">
                <component :is="item.icon" />
              </el-icon>
            </div>
          </div>
          <div class="summary-card__value">{{ item.value }}</div>
          <div class="summary-card__foot">
            <span>{{ item.footLabel }}</span>
            <strong>{{ item.newValue }}</strong>
          </div>
        </article>
      </div>
    </section>

    <section class="chart-grid">
      <OrderBarChart :time-type="activeTimeType" />
      <GoodsBarChart :time-type="activeTimeType" />
      <GoodsPieChart :time-type="activeTimeType" />
      <OrderRadarChart :time-type="activeTimeType" />
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from "vue";
import { Document, Goods, User, Wallet } from "@element-plus/icons-vue";
import OrderBarChart from "./components/OrderBarChart.vue";
import GoodsBarChart from "./components/GoodsBarChart.vue";
import GoodsPieChart from "./components/GoodsPieChart.vue";
import OrderRadarChart from "./components/OrderRadarChart.vue";
import { defDashboardService } from "@/api/admin/dashboard";
import type { DashboardCountResponse, DashboardTimeType } from "@/rpc/admin/dashboard";
import { DashboardTimeType as DashboardTimeTypeEnum } from "@/rpc/admin/dashboard";
import { formatPrice } from "@/utils/utils";

defineOptions({
  name: "Analytics",
  inheritAttrs: false
});

/** 时间维度选项，需与后端枚举定义保持一致。 */
const timeOptions = [
  { label: "今日", value: DashboardTimeTypeEnum.DAY },
  { label: "本周", value: DashboardTimeTypeEnum.WEEK },
  { label: "本月", value: DashboardTimeTypeEnum.MONTH }
];

const activeTimeType = ref<DashboardTimeType>(DashboardTimeTypeEnum.DAY);

const activeTimeLabel = computed(() => {
  return timeOptions.find(item => item.value === activeTimeType.value)?.label ?? "当前";
});

const dashboardCountUser = reactive<DashboardCountResponse>({
  /** 新增数量 */
  newNum: 0,
  /** 总数量 */
  totalNum: 0
});

const dashboardCountGoods = reactive<DashboardCountResponse>({
  /** 新增数量 */
  newNum: 0,
  /** 总数量 */
  totalNum: 0
});

const dashboardCountOrder = reactive<DashboardCountResponse>({
  /** 新增数量 */
  newNum: 0,
  /** 总数量 */
  totalNum: 0
});

const dashboardCountSale = reactive<DashboardCountResponse>({
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
async function loadSummaryData(timeType: DashboardTimeType) {
  const [user, goods, order, sale] = await Promise.all([
    defDashboardService.DashboardCountUser({ timeType }),
    defDashboardService.DashboardCountGoods({ timeType }),
    defDashboardService.DashboardCountOrder({ timeType }),
    defDashboardService.DashboardCountSale({ timeType })
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
  padding: 24px;
  background:
    radial-gradient(circle at top left, rgb(45 108 223 / 10%), transparent 26%), linear-gradient(180deg, #f5f7fb 0%, #eef3f8 100%);
}

.hero-card {
  padding: 24px;
  border: 1px solid rgb(255 255 255 / 70%);
  border-radius: 24px;
  background: linear-gradient(135deg, rgb(255 255 255 / 95%), rgb(246 249 253 / 92%)), #fff;
  box-shadow: 0 20px 40px rgb(31 45 61 / 8%);
}

.hero-card__head {
  display: flex;
  gap: 24px;
  align-items: flex-start;
  justify-content: space-between;
  margin-bottom: 24px;
}

.hero-card__title {
  margin: 0;
  font-size: 28px;
  font-weight: 700;
  color: #1f2d3d;
}

.hero-card__desc {
  max-width: 620px;
  margin: 8px 0 0;
  color: #6b7a90;
  line-height: 1.7;
}

.hero-card__tabs {
  min-width: 280px;
}

.hero-card__summary {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 18px;
}

.summary-card {
  position: relative;
  padding: 20px;
  overflow: hidden;
  border-radius: 22px;
  background: linear-gradient(180deg, #fff 0%, #f8fbff 100%);
  box-shadow: inset 0 1px 0 rgb(255 255 255 / 70%);
}

.summary-card::after {
  position: absolute;
  top: -28px;
  right: -18px;
  width: 112px;
  height: 112px;
  content: "";
  background: radial-gradient(circle, rgb(255 255 255 / 0%), var(--card-accent) 100%);
  opacity: 0.12;
  filter: blur(6px);
}

.summary-card__meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.summary-card__label {
  font-size: 14px;
  color: #6b7a90;
}

.summary-card__icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 42px;
  height: 42px;
  color: #fff;
  border-radius: 14px;
  background: var(--card-accent);
  box-shadow: 0 12px 24px rgb(0 0 0 / 10%);
}

.summary-card__value {
  margin: 20px 0 12px;
  font-size: 30px;
  font-weight: 700;
  color: #1f2d3d;
}

.summary-card__foot {
  display: flex;
  gap: 8px;
  align-items: center;
  font-size: 13px;
  color: #7f8ea3;
}

.summary-card__foot strong {
  color: var(--card-accent);
}

.chart-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 18px;
  margin-top: 20px;
}

:deep(.hero-card__tabs .el-tabs__header) {
  margin: 0;
}

:deep(.hero-card__tabs .el-tabs__nav-wrap::after) {
  display: none;
}

@media (max-width: 1200px) {
  .hero-card__summary,
  .chart-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 768px) {
  .analytics-page {
    padding: 16px;
  }

  .hero-card {
    padding: 18px;
    border-radius: 18px;
  }

  .hero-card__head {
    flex-direction: column;
  }

  .hero-card__tabs {
    width: 100%;
    min-width: 0;
  }

  .hero-card__summary,
  .chart-grid {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
