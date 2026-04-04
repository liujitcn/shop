<template>
  <div v-loading="loading" class="workspace-page">
    <el-card class="workspace-card workspace-card--hero" shadow="never">
      <div class="workspace-hero">
        <div class="workspace-hero__intro">
          <div class="workspace-user">
            <div class="workspace-avatar">
              <img :src="avatarSrc" alt="avatar" @error="handleAvatarError" />
            </div>
            <div class="workspace-copy">
              <h1>{{ greetingText }}</h1>
            </div>
          </div>
        </div>

        <div class="metric-grid">
          <article v-for="item in metricCards" :key="item.key" class="metric-card">
            <button type="button" class="metric-card__main" @click="handleNavigate(item.analysisPath)">
              <div class="metric-card__header">
                <span class="metric-card__label">{{ item.label }}</span>
                <span class="metric-card__trend" :class="`metric-card__trend--${item.trendTone}`">{{ item.trend }}</span>
              </div>
              <strong class="metric-card__value">{{ item.value }}</strong>
            </button>
            <div class="metric-card__footer">
              <span>{{ item.subLabel }}</span>
              <button type="button" class="metric-card__link" @click="handleNavigate(item.actionPath)">
                {{ item.subValue }}
              </button>
            </div>
          </article>
        </div>
      </div>
    </el-card>

    <section class="workspace-main">
      <div class="workspace-primary">
        <el-card class="workspace-card workspace-card--todo" shadow="never">
          <template #header>
            <div class="panel-header">
              <div><h3>待处理事项</h3></div>
            </div>
          </template>

          <div class="todo-list">
            <button v-for="item in todoItems" :key="item.key" type="button" class="todo-item" @click="handleNavigate(item.path)">
              <div class="todo-item__main">
                <div class="todo-item__badge">{{ item.badge }}</div>
                <div>
                  <strong>{{ item.title }}</strong>
                  <p class="workspace-item__desc">{{ item.description }}</p>
                </div>
              </div>
              <div class="todo-item__side">
                <span class="todo-item__count">{{ item.count }}</span>
                <span class="todo-item__unit">{{ item.unit }}</span>
              </div>
            </button>
          </div>
        </el-card>
      </div>

      <div class="workspace-side">
        <el-card class="workspace-card workspace-card--risk" shadow="never">
          <template #header>
            <div class="panel-header">
              <div><h3>风险提醒</h3></div>
            </div>
          </template>

          <div class="risk-list">
            <button v-for="item in riskItems" :key="item.key" type="button" class="risk-item" @click="handleNavigate(item.path)">
              <div class="risk-item__main">
                <span class="risk-item__tag" :class="`risk-item__tag--${item.level}`">{{ item.levelLabel }}</span>
                <div class="risk-item__content">
                  <div class="risk-item__title">{{ item.title }}</div>
                  <p class="workspace-item__desc">{{ item.description }}</p>
                </div>
              </div>
              <div class="risk-item__side">
                <strong class="risk-item__count">{{ item.count }}</strong>
                <span class="risk-item__unit">{{ item.unit }}</span>
              </div>
            </button>
          </div>
        </el-card>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
defineOptions({
  name: "Workspace",
  inheritAttrs: false
});

import { computed, onMounted, reactive, ref, watch } from "vue";
import { useRouter, type RouteLocationRaw } from "vue-router";
import { defWorkspaceService } from "@/api/admin/workspace";
import type { WorkspaceMetricsResponse, WorkspaceRiskListResponse, WorkspaceTodoListResponse } from "@/rpc/admin/workspace";
import { useUserStore } from "@/stores/modules/user";
import { GoodsStatus, OrderStatus, PayBillStatus } from "@/rpc/common/enum";
import { navigateTo } from "@/utils/router";
import { formatPrice, formatSrc } from "@/utils/utils";
import defaultAvatar from "@/assets/images/avatar.png";

/** 工作台指标卡片。 */
interface WorkspaceMetricCard {
  /** 唯一标识。 */
  key: string;
  /** 指标标题。 */
  label: string;
  /** 指标主值。 */
  value: string;
  /** 趋势文案。 */
  trend: string;
  /** 趋势语义。 */
  trendTone: "up" | "flat" | "down";
  /** 次级标签。 */
  subLabel: string;
  /** 次级值。 */
  subValue: string;
  /** 分析页跳转路径。 */
  analysisPath: string;
  /** 业务页跳转路径。 */
  actionPath: string;
}

/** 工作台待处理事项。 */
interface WorkspaceTodoItem {
  /** 唯一标识。 */
  key: string;
  /** 事项标题。 */
  title: string;
  /** 数量值。 */
  count: number;
  /** 数值单位。 */
  unit: string;
  /** 辅助说明。 */
  description: string;
  /** 徽标文案。 */
  badge: string;
  /** 跳转路径。 */
  path: RouteLocationRaw;
}

/** 风险等级。 */
type WorkspaceRiskLevel = "warning" | "danger" | "info";

/** 工作台风险提醒。 */
interface WorkspaceRiskItem {
  /** 唯一标识。 */
  key: string;
  /** 风险标题。 */
  title: string;
  /** 风险数量。 */
  count: number;
  /** 数值单位。 */
  unit: string;
  /** 辅助说明。 */
  description: string;
  /** 风险等级。 */
  level: WorkspaceRiskLevel;
  /** 风险等级文案。 */
  levelLabel: string;
  /** 跳转路径。 */
  path: RouteLocationRaw;
}

const router = useRouter();
const userStore = useUserStore();
const loading = ref(false);
const avatarSrc = ref(defaultAvatar);

const metrics = reactive<WorkspaceMetricsResponse>({
  todayOrderCount: 0,
  todayOrderGrowthRate: 0,
  todaySaleAmount: 0,
  averageOrderAmount: 0,
  payConversionRate: 0,
  todayOrderUserCount: 0,
  repurchaseRate: 0,
  todayNewUserCount: 0,
  todaySaleCount: 0,
  activeGoodsCount: 0,
  todayNewGoodsCount: 0,
  todaySaleGrowthRate: 0
});

const todoSummary = reactive<WorkspaceTodoListResponse>({
  pendingPayOrderCount: 0,
  pendingShippedOrderCount: 0,
  lowInventorySkuCount: 0,
  pendingPutOnGoodsCount: 0
});

const riskSummary = reactive<WorkspaceRiskListResponse>({
  abnormalPayBillCount: 0,
  zeroInventoryPutOnSkuCount: 0,
  abnormalPriceSkuCount: 0
});

/** 当前显示名称，优先取昵称。 */
const displayName = computed(() => {
  return userStore.userInfo.nickName || userStore.userInfo.userName || "管理员";
});

/** 同步工作台头像展示，优先使用用户头像，为空时回退默认头像。 */
function syncAvatarSrc(avatar?: string) {
  avatarSrc.value = formatSrc(avatar || "") || defaultAvatar;
}

/** 头像加载失败时回退默认头像，避免出现破图。 */
function handleAvatarError() {
  avatarSrc.value = defaultAvatar;
}

/** 根据当前时段生成问候语。 */
const greetingText = computed(() => {
  const hour = new Date().getHours();
  if (hour < 6) return `凌晨好，${displayName.value}`;
  if (hour < 11) return `上午好，${displayName.value}`;
  if (hour < 14) return `中午好，${displayName.value}`;
  if (hour < 18) return `下午好，${displayName.value}`;
  return `晚上好，${displayName.value}`;
});

/** 将分转成带货币符号的金额文本。 */
function formatPriceLabel(value: number) {
  return formatPrice(value);
}

/** 将千分比转成 1 位小数百分比。 */
function formatRatioLabel(value: number) {
  return `${(value / 10).toFixed(1)}%`;
}

/** 工作台指标卡片。 */
const metricCards = computed<WorkspaceMetricCard[]>(() => {
  return [
    {
      key: "today-order",
      label: "今日订单",
      value: String(metrics.todayOrderCount),
      trend: `较昨日 ${metrics.todayOrderGrowthRate >= 0 ? "+" : ""}${metrics.todayOrderGrowthRate}%`,
      trendTone: metrics.todayOrderGrowthRate >= 0 ? "up" : "down",
      subLabel: "客单价",
      subValue: formatPriceLabel(metrics.averageOrderAmount),
      analysisPath: "/dashboard/analytics/order",
      actionPath: "/dashboard/analytics/order"
    },
    {
      key: "today-sales",
      label: "今日成交额",
      value: formatPriceLabel(metrics.todaySaleAmount),
      trend: `较昨日 ${metrics.todaySaleGrowthRate >= 0 ? "+" : ""}${metrics.todaySaleGrowthRate}%`,
      trendTone: metrics.todaySaleGrowthRate >= 0 ? "up" : "down",
      subLabel: "支付转化",
      subValue: formatRatioLabel(metrics.payConversionRate),
      analysisPath: "/dashboard/analytics/order",
      actionPath: "/dashboard/analytics/order"
    },
    {
      key: "today-users",
      label: "今日下单用户",
      value: String(metrics.todayOrderUserCount),
      trend: `复购占比 ${formatRatioLabel(metrics.repurchaseRate)}`,
      trendTone: "flat",
      subLabel: "新增用户",
      subValue: `${metrics.todayNewUserCount} 人`,
      analysisPath: "/dashboard/analytics/user",
      actionPath: "/dashboard/analytics/user"
    },
    {
      key: "today-goods",
      label: "今日商品销量",
      value: String(metrics.todaySaleCount),
      trend: `动销商品 ${metrics.activeGoodsCount} 个`,
      trendTone: "flat",
      subLabel: "新增商品",
      subValue: `${metrics.todayNewGoodsCount} 个`,
      analysisPath: "/dashboard/analytics/goods",
      actionPath: "/dashboard/analytics/goods"
    }
  ];
});

/** 工作台待处理事项。 */
const todoItems = computed<WorkspaceTodoItem[]>(() => {
  return [
    {
      key: "todo-pay",
      title: "待支付订单",
      count: todoSummary.pendingPayOrderCount,
      unit: "单",
      description: "继续观察支付转化情况。",
      badge: "支付",
      path: { path: "/order/order", query: { status: String(OrderStatus.CREATED) } }
    },
    {
      key: "todo-shipped",
      title: "待发货订单",
      count: todoSummary.pendingShippedOrderCount,
      unit: "单",
      description: "优先处理已支付未发货订单。",
      badge: "履约",
      path: { path: "/order/order", query: { status: String(OrderStatus.PAID) } }
    },
    {
      key: "todo-stock",
      title: "低库存商品",
      count: todoSummary.lowInventorySkuCount,
      unit: "个",
      description: "需要尽快补货或调整售卖策略。",
      badge: "库存",
      path: { path: "/goods/goods", query: { status: String(GoodsStatus.PUT_ON), inventoryAlert: "1" } }
    },
    {
      key: "todo-put-on",
      title: "待上架商品",
      count: todoSummary.pendingPutOnGoodsCount,
      unit: "个",
      description: "资料已齐，适合统一回看上架。",
      badge: "商品",
      path: { path: "/goods/goods", query: { status: String(GoodsStatus.PULL_OFF) } }
    }
  ];
});

/** 工作台风险提醒。 */
const riskItems = computed<WorkspaceRiskItem[]>(() => {
  return [
    {
      key: "risk-bill",
      title: "对账单异常",
      count: riskSummary.abnormalPayBillCount,
      unit: "项",
      description: "优先核对对账结果，尽快排查差异原因。",
      level: "danger",
      levelLabel: "高风险",
      path: { path: "/pay/bill", query: { status: String(PayBillStatus.HAS_ERROR) } }
    },
    {
      key: "risk-zero-stock",
      title: "库存为 0 但仍上架",
      count: riskSummary.zeroInventoryPutOnSkuCount,
      unit: "项",
      description: "继续曝光会直接影响转化。",
      level: "danger",
      levelLabel: "高风险",
      path: { path: "/goods/goods", query: { status: String(GoodsStatus.PUT_ON), inventoryAlert: "2" } }
    },
    {
      key: "risk-price",
      title: "价格配置异常",
      count: riskSummary.abnormalPriceSkuCount,
      unit: "项",
      description: "需要复核售价与折扣价关系。",
      level: "warning",
      levelLabel: "需核对",
      path: { path: "/goods/goods", query: { priceAlert: "1" } }
    }
  ];
});

/**
 * 统一处理工作台入口跳转。
 * @param path 目标路由路径。
 */
function handleNavigate(path: RouteLocationRaw) {
  if (!path) return;
  if (typeof path === "string") {
    navigateTo(router, path);
    return;
  }
  navigateTo(router, String(path.path ?? ""), (path.query ?? {}) as Record<string, string | number>);
}

/** 加载工作台三块数据。 */
async function loadWorkspaceData() {
  loading.value = true;
  try {
    const [metricsData, todoData, riskData] = await Promise.all([
      defWorkspaceService.GetWorkspaceMetrics({}),
      defWorkspaceService.GetWorkspaceTodoList({}),
      defWorkspaceService.GetWorkspaceRiskList({})
    ]);
    Object.assign(metrics, metricsData);
    Object.assign(todoSummary, todoData);
    Object.assign(riskSummary, riskData);
  } finally {
    loading.value = false;
  }
}

onMounted(() => {
  loadWorkspaceData();
});

watch(
  () => userStore.userInfo.avatar,
  avatar => {
    syncAvatarSrc(avatar);
  },
  { immediate: true }
);
</script>

<style lang="scss" scoped>
.workspace-page {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.workspace-card {
  border: 1px solid var(--admin-page-card-border);
  border-radius: 16px;
  background: var(--admin-page-card-bg);
  box-shadow: var(--admin-page-shadow);
}

.workspace-card--hero {
  overflow: hidden;
}

:deep(.workspace-card .el-card__header) {
  position: relative;
  padding: 18px 20px 0;
  border-bottom: 0;
}

:deep(.workspace-card .el-card__header)::after {
  position: absolute;
  right: 20px;
  bottom: 0;
  left: 20px;
  height: 1px;
  content: "";
  background: var(--admin-page-divider);
}

:deep(.workspace-card .el-card__body) {
  padding: 18px 20px 20px;
}

.workspace-hero {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.workspace-hero__intro {
  display: block;
}

.workspace-user {
  display: flex;
  gap: 16px;
  align-items: center;
  min-width: 0;
}

.workspace-avatar {
  width: 64px;
  height: 64px;
  flex-shrink: 0;
  overflow: hidden;
  border: 1px solid var(--admin-page-card-border);
  border-radius: 50%;
  background: #edf2f7;
  box-shadow: 0 8px 18px rgba(15, 23, 42, 0.12);

  img {
    display: block;
    width: 100%;
    height: 100%;
    object-fit: cover;
  }
}

.workspace-copy h1 {
  margin: 0;
  font-size: 24px;
  line-height: 1.2;
  color: var(--admin-page-text-primary);
}

.metric-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 14px;
}

.metric-card__main,
.todo-item,
.risk-item {
  cursor: pointer;
  background: transparent;
  border: 0;
}

.metric-card {
  border: 1px solid var(--admin-page-card-border-soft);
  border-radius: 14px;
  background: var(--admin-page-card-bg-soft);
  transition:
    border-color 0.2s ease,
    transform 0.2s ease;
}

.metric-card:hover,
.todo-item:hover,
.risk-item:hover {
  border-color: var(--admin-page-card-border-muted);
  transform: translateY(-1px);
}

.metric-card__main {
  display: flex;
  flex-direction: column;
  gap: 12px;
  width: 100%;
  padding: 16px 16px 12px;
  text-align: left;
}

.metric-card__header,
.metric-card__footer {
  display: flex;
  gap: 12px;
  align-items: center;
  justify-content: space-between;
}

.metric-card__label {
  font-size: 13px;
  color: var(--admin-page-text-secondary);
}

.metric-card__trend {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 28px;
  padding: 0 10px;
  font-size: 12px;
  font-weight: 600;
  border-radius: 999px;
}

.metric-card__trend--up {
  color: var(--el-color-success);
  background: rgb(103 194 58 / 12%);
}

.metric-card__trend--flat {
  color: var(--admin-page-accent-soft-text);
  background: var(--admin-page-accent-soft-bg);
}

.metric-card__trend--down {
  color: var(--el-color-danger);
  background: rgb(245 108 108 / 12%);
}

.metric-card__value {
  font-size: 28px;
  line-height: 1.1;
  color: var(--admin-page-text-primary);
}

.metric-card__footer {
  padding: 12px 16px 16px;
  font-size: 13px;
  color: var(--admin-page-text-secondary);
  border-top: 1px solid var(--admin-page-divider);
}

.metric-card__link {
  padding: 0;
  font-size: 13px;
  font-weight: 600;
  color: var(--el-color-primary);
  cursor: pointer;
  background: transparent;
  border: 0;
}

.workspace-main {
  display: grid;
  grid-template-columns: minmax(0, 1.35fr) minmax(320px, 0.85fr);
  gap: 20px;
}

.workspace-primary,
.workspace-side {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.panel-header h3 {
  margin: 0;
  font-size: 17px;
  font-weight: 700;
  color: var(--admin-page-text-primary);
}

.todo-list,
.risk-list {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.todo-item {
  display: flex;
  gap: 16px;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  padding: 16px 18px;
  text-align: left;
  border: 1px solid var(--admin-page-card-border-soft);
  border-radius: 14px;
  background: var(--admin-page-card-bg-soft);
  transition:
    border-color 0.2s ease,
    transform 0.2s ease;
}

.todo-item__main {
  display: flex;
  gap: 14px;
  align-items: flex-start;
  min-width: 0;
}

.todo-item__badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 44px;
  height: 44px;
  padding: 0 10px;
  font-size: 12px;
  font-weight: 700;
  color: var(--admin-page-accent-soft-text);
  background: var(--admin-page-badge-bg);
  border-radius: 12px;
}

.todo-item strong,
.risk-item__title {
  display: block;
  color: var(--admin-page-text-primary);
}

.todo-item strong {
  font-size: 16px;
  line-height: 1.35;
}

.workspace-item__desc {
  margin: 6px 0 0;
  font-size: 12px;
  line-height: 1.6;
  color: var(--admin-page-text-secondary);
}

.todo-item__side {
  display: flex;
  flex-direction: column;
  gap: 4px;
  align-items: flex-end;
  flex-shrink: 0;
}

.todo-item__count {
  font-size: 28px;
  font-weight: 700;
  line-height: 1;
  color: var(--admin-page-text-primary);
}

.todo-item__unit {
  font-size: 12px;
  color: var(--admin-page-text-placeholder);
}

.risk-item {
  display: flex;
  gap: 16px;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  padding: 16px 18px;
  text-align: left;
  border: 1px solid var(--admin-page-card-border-soft);
  border-radius: 14px;
  background: var(--admin-page-card-bg-soft);
  transition:
    border-color 0.2s ease,
    transform 0.2s ease;
}

.risk-item__main {
  display: flex;
  gap: 14px;
  align-items: flex-start;
  min-width: 0;
}

.risk-item__tag {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 56px;
  height: 44px;
  padding: 0 10px;
  font-size: 12px;
  font-weight: 600;
  border-radius: 12px;
}

.risk-item__tag--danger {
  color: var(--el-color-danger);
  background: rgb(245 108 108 / 12%);
}

.risk-item__tag--warning {
  color: var(--el-color-warning);
  background: rgb(230 162 60 / 12%);
}

.risk-item__tag--info {
  color: var(--admin-page-accent-soft-text);
  background: var(--admin-page-accent-soft-bg);
}

.risk-item__content {
  min-width: 0;
}

.risk-item__side {
  display: flex;
  flex-direction: column;
  gap: 4px;
  align-items: flex-end;
  flex-shrink: 0;
}

.risk-item__count {
  font-size: 28px;
  font-weight: 700;
  line-height: 1;
  color: var(--admin-page-text-primary);
}

.risk-item__unit {
  font-size: 12px;
  color: var(--admin-page-text-placeholder);
}

.risk-item__title {
  font-size: 16px;
  line-height: 1.35;
  color: var(--admin-page-text-primary);
}

@media (width <= 1200px) {
  .metric-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .workspace-main {
    display: flex;
    flex-direction: column;
  }

  .workspace-side {
    order: -1;
  }
}

@media (width <= 768px) {
  .metric-grid {
    grid-template-columns: 1fr;
  }

  .todo-item {
    flex-direction: column;
    align-items: stretch;
  }

  .risk-item {
    flex-direction: column;
    align-items: stretch;
  }

  .todo-item__side {
    flex-direction: row;
    align-items: baseline;
    justify-content: space-between;
  }

  .risk-item__side {
    flex-direction: row;
    align-items: baseline;
    justify-content: space-between;
  }
}
</style>
