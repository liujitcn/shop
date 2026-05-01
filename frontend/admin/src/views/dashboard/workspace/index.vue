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
              <p>今日重点：订单履约、评价审核与库存风险</p>
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
                <el-icon><ArrowRight /></el-icon>
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

        <el-card class="workspace-card workspace-card--pending-comment" shadow="never">
          <template #header>
            <div class="panel-header panel-header--action">
              <div><h3>待审核评价</h3></div>
              <button type="button" class="panel-header__link" @click="handleNavigate(pendingCommentListPath)">
                查看更多
                <el-icon><ArrowRight /></el-icon>
              </button>
            </div>
          </template>

          <ProTable
            row-key="id"
            class="pending-comment-table no-card"
            :data="pendingComments"
            :columns="pendingCommentColumns"
            :pagination="false"
            :tool-button="false"
            :border="false"
          />
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

        <el-card class="workspace-card workspace-card--reputation" shadow="never">
          <template #header>
            <div class="panel-header">
              <div><h3>口碑洞察</h3></div>
            </div>
          </template>

          <div class="reputation-list">
            <article class="reputation-item">
              <span>平均评分</span>
              <strong>{{ formatScoreLabel(reputationSummary.average_comment_score) }}</strong>
              <p>近 7 日评价口径</p>
            </article>
            <article class="reputation-item">
              <span>高频标签</span>
              <strong>{{ hotTagText }}</strong>
              <p>按评价标签提及次数排序</p>
            </article>
            <article class="reputation-item">
              <span>AI 摘要</span>
              <strong>{{ reputationSummary.ai_summary || "暂无评价摘要" }}</strong>
              <p>来自商品评价摘要</p>
            </article>
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

import { computed, h, onBeforeUnmount, onMounted, reactive, ref, resolveComponent, watch } from "vue";
import { useRouter, type RouteLocationRaw } from "vue-router";
import type { ColumnProps, RenderScope } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
import { defWorkspaceService } from "@/api/admin/workspace";
import type {
  SummaryWorkspaceMetricsResponse,
  SummaryWorkspaceReputationResponse,
  SummaryWorkspaceRiskResponse,
  SummaryWorkspaceTodoResponse,
  WorkspacePendingComment
} from "@/rpc/admin/v1/workspace";
import { useUserStore } from "@/stores/modules/user";
import { CommentStatus, GoodsStatus, OrderStatus, PayBillStatus, SseRefreshTarget, SseStream } from "@/rpc/common/v1/enum";
import { navigateTo } from "@/utils/router";
import { formatPrice } from "@/utils/utils";
import { subscribeSseRefresh, type SseStop } from "@/utils/sse";
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
  analysisPath: RouteLocationRaw;
  /** 业务页跳转路径。 */
  actionPath: RouteLocationRaw;
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
const pendingComments = ref<WorkspacePendingComment[]>([]);
const workspaceRefreshTargets: SseRefreshTarget[] = [
  SseRefreshTarget.SSE_REFRESH_TARGET_ADMIN_WORKSPACE_METRICS,
  SseRefreshTarget.SSE_REFRESH_TARGET_ADMIN_WORKSPACE_TODO,
  SseRefreshTarget.SSE_REFRESH_TARGET_ADMIN_WORKSPACE_RISK,
  SseRefreshTarget.SSE_REFRESH_TARGET_ADMIN_WORKSPACE_REPUTATION,
  SseRefreshTarget.SSE_REFRESH_TARGET_ADMIN_WORKSPACE_PENDING_COMMENTS
];
let stopWorkspaceSse: SseStop | null = null;
let workspaceRefreshTimer: ReturnType<typeof setTimeout> | null = null;
let queuedRefreshTargets = new Set<SseRefreshTarget>();

const metrics = reactive<SummaryWorkspaceMetricsResponse>({
  today_order_count: 0,
  today_order_growth_rate: 0,
  today_sale_amount: 0,
  average_order_amount: 0,
  pay_conversion_rate: 0,
  today_order_user_count: 0,
  repurchase_rate: 0,
  today_new_user_count: 0,
  today_sale_count: 0,
  active_goods_count: 0,
  today_new_goods_count: 0,
  today_sale_growth_rate: 0,
  today_comment_count: 0,
  average_comment_score: 0
});

const todoSummary = reactive<SummaryWorkspaceTodoResponse>({
  pending_pay_order_count: 0,
  pending_shipped_order_count: 0,
  low_inventory_sku_count: 0,
  pending_put_on_goods_count: 0,
  pending_comment_count: 0,
  pending_comment_discussion_count: 0
});

const riskSummary = reactive<SummaryWorkspaceRiskResponse>({
  abnormal_pay_bill_count: 0,
  zero_inventory_put_on_sku_count: 0,
  abnormal_price_sku_count: 0,
  low_score_comment_count: 0
});

const reputationSummary = reactive<SummaryWorkspaceReputationResponse>({
  average_comment_score: 0,
  hot_tags: [],
  ai_summary: ""
});

/** 跳转待审核评价列表时携带固定审核状态。 */
const pendingCommentListPath = computed<RouteLocationRaw>(() => {
  return { path: "/admin/comment/info", query: { status: String(CommentStatus.PENDING_REVIEW_CS) } };
});

/** 口碑洞察高频标签文案。 */
const hotTagText = computed(() => {
  const names = reputationSummary.hot_tags.map(item => item.name).filter(Boolean);
  return names.length ? names.join("、") : "暂无标签";
});

/** 当前显示名称，优先取昵称。 */
const displayName = computed(() => {
  return userStore.userInfo.nick_name || userStore.userInfo.user_name || "管理员";
});

/** 同步工作台头像展示，优先使用用户头像，为空时回退默认头像。 */
function syncAvatarSrc(avatar?: string) {
  // 工作台头像与头部、个人中心统一使用用户资料中的原始地址，避免重复拼接静态域名导致地址失效。
  avatarSrc.value = avatar || defaultAvatar;
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

/** 将十分位评分转成 1 位小数评分。 */
function formatScoreLabel(value: number) {
  return `${(value / 10).toFixed(1)} 分`;
}

/** 工作台指标卡片。 */
const metricCards = computed<WorkspaceMetricCard[]>(() => {
  return [
    {
      key: "today-order",
      label: "今日订单",
      value: String(metrics.today_order_count),
      trend: `较昨日 ${metrics.today_order_growth_rate >= 0 ? "+" : ""}${metrics.today_order_growth_rate}%`,
      trendTone: metrics.today_order_growth_rate >= 0 ? "up" : "down",
      subLabel: "客单价",
      subValue: formatPriceLabel(metrics.average_order_amount),
      analysisPath: "/dashboard/analytics/order",
      actionPath: "/dashboard/analytics/order"
    },
    {
      key: "today-sales",
      label: "今日成交额",
      value: formatPriceLabel(metrics.today_sale_amount),
      trend: `较昨日 ${metrics.today_sale_growth_rate >= 0 ? "+" : ""}${metrics.today_sale_growth_rate}%`,
      trendTone: metrics.today_sale_growth_rate >= 0 ? "up" : "down",
      subLabel: "支付转化",
      subValue: formatRatioLabel(metrics.pay_conversion_rate),
      analysisPath: "/dashboard/analytics/order",
      actionPath: "/dashboard/analytics/order"
    },
    {
      key: "today-users",
      label: "今日下单用户",
      value: String(metrics.today_order_user_count),
      trend: `复购占比 ${formatRatioLabel(metrics.repurchase_rate)}`,
      trendTone: "flat",
      subLabel: "新增用户",
      subValue: `${metrics.today_new_user_count} 人`,
      analysisPath: "/dashboard/analytics/user",
      actionPath: "/dashboard/analytics/user"
    },
    {
      key: "today-goods",
      label: "今日商品销量",
      value: String(metrics.today_sale_count),
      trend: `动销商品 ${metrics.active_goods_count} 个`,
      trendTone: "flat",
      subLabel: "新增商品",
      subValue: `${metrics.today_new_goods_count} 个`,
      analysisPath: "/dashboard/analytics/goods",
      actionPath: { path: "/goods/info" }
    },
    {
      key: "today-comment",
      label: "今日评价",
      value: String(metrics.today_comment_count),
      trend: `平均评分 ${formatScoreLabel(metrics.average_comment_score)}`,
      trendTone: "flat",
      subLabel: "待审核",
      subValue: `${todoSummary.pending_comment_count} 条`,
      analysisPath: { path: "/admin/comment/info" },
      actionPath: pendingCommentListPath.value
    }
  ];
});

/** 工作台待处理事项。 */
const todoItems = computed<WorkspaceTodoItem[]>(() => {
  return [
    {
      key: "todo-pay",
      title: "待支付订单",
      count: todoSummary.pending_pay_order_count,
      unit: "单",
      description: "继续观察支付转化情况。",
      badge: "支付",
      path: { path: "/order/info", query: { status: String(OrderStatus.CREATED) } }
    },
    {
      key: "todo-shipped",
      title: "待发货订单",
      count: todoSummary.pending_shipped_order_count,
      unit: "单",
      description: "优先处理已支付未发货订单。",
      badge: "履约",
      path: { path: "/order/info", query: { status: String(OrderStatus.PAID) } }
    },
    {
      key: "todo-comment",
      title: "待审核评论",
      count: todoSummary.pending_comment_count,
      unit: "条",
      description: "新评价提交后需要审核展示。",
      badge: "评论",
      path: pendingCommentListPath.value
    },
    {
      key: "todo-discussion",
      title: "待审核讨论",
      count: todoSummary.pending_comment_discussion_count,
      unit: "条",
      description: "评论回复内容需要尽快处理。",
      badge: "互动",
      path: { path: "/admin/comment/info", query: { has_pending_discussion: "1" } }
    },
    {
      key: "todo-stock",
      title: "低库存商品",
      count: todoSummary.low_inventory_sku_count,
      unit: "个",
      description: "需要尽快补货或调整售卖策略。",
      badge: "库存",
      path: { path: "/goods/info", query: { status: String(GoodsStatus.PUT_ON), inventoryAlert: "1" } }
    },
    {
      key: "todo-put-on",
      title: "待上架商品",
      count: todoSummary.pending_put_on_goods_count,
      unit: "个",
      description: "资料已齐，适合统一回看上架。",
      badge: "商品",
      path: { path: "/goods/info", query: { status: String(GoodsStatus.PULL_OFF) } }
    }
  ];
});

/** 工作台风险提醒。 */
const riskItems = computed<WorkspaceRiskItem[]>(() => {
  return [
    {
      key: "risk-bill",
      title: "对账单异常",
      count: riskSummary.abnormal_pay_bill_count,
      unit: "项",
      description: "优先核对对账结果，尽快排查差异原因。",
      level: "danger",
      levelLabel: "高风险",
      path: { path: "/pay/bill", query: { status: String(PayBillStatus.HAS_ERROR) } }
    },
    {
      key: "risk-zero-stock",
      title: "库存为 0 但仍上架",
      count: riskSummary.zero_inventory_put_on_sku_count,
      unit: "项",
      description: "继续曝光会直接影响转化。",
      level: "danger",
      levelLabel: "高风险",
      path: { path: "/goods/info", query: { status: String(GoodsStatus.PUT_ON), inventoryAlert: "2" } }
    },
    {
      key: "risk-price",
      title: "价格配置异常",
      count: riskSummary.abnormal_price_sku_count,
      unit: "项",
      description: "需要复核售价与折扣价关系。",
      level: "warning",
      levelLabel: "需核对",
      path: { path: "/goods/info", query: { priceAlert: "1" } }
    },
    {
      key: "risk-low-score-comment",
      title: "低分评价提醒",
      count: riskSummary.low_score_comment_count,
      unit: "条",
      description: "低评分会影响商品转化，建议优先跟进。",
      level: "warning",
      levelLabel: "需跟进",
      path: { path: "/admin/comment/info", query: { max_goods_score: "2" } }
    }
  ];
});

/** 待审核评价表格列配置。 */
const pendingCommentColumns: ColumnProps[] = [
  {
    prop: "goods_name",
    label: "商品",
    minWidth: 160,
    showOverflowTooltip: true,
    render: scope => renderPendingCommentGoods(scope)
  },
  { prop: "user_name", label: "用户", width: 96, showOverflowTooltip: true },
  {
    prop: "goods_score",
    label: "评分",
    width: 112,
    render: scope => renderPendingCommentScore(scope)
  },
  { prop: "content", label: "内容摘要", minWidth: 190, showOverflowTooltip: true },
  {
    prop: "created_at",
    label: "时间",
    width: 90,
    render: scope => formatPendingCommentTime(scope.row.created_at)
  },
  {
    prop: "operation",
    label: "操作",
    width: 88,
    render: (scope: RenderScope) => {
      return h(
        resolveComponent("el-button"),
        {
          type: "primary",
          link: true,
          onClick: () => handleOpenCommentDetail(scope.row.id)
        },
        () => "审核"
      );
    }
  }
];

/** 渲染待审核评价商品名称，点击后进入评论详情审核。 */
function renderPendingCommentGoods(scope: RenderScope) {
  const row = scope.row as WorkspacePendingComment;
  return h(
    resolveComponent("el-link"),
    {
      type: "primary",
      onClick: (event: MouseEvent) => {
        event.stopPropagation();
        handleOpenCommentDetail(row.id);
      }
    },
    () => row.goods_name || "未命名商品"
  );
}

/** 渲染待审核评价评分，保持与评价列表的星级展示一致。 */
function renderPendingCommentScore(scope: RenderScope) {
  const row = scope.row as WorkspacePendingComment;
  return h(resolveComponent("el-rate"), { modelValue: row.goods_score, disabled: true, size: "small" });
}

/** 格式化待审核评价时间，工作台只展示短时间。 */
function formatPendingCommentTime(value: string) {
  if (!value) return "--";
  const parts = value.split(" ");
  return parts.length > 1 ? parts[1].slice(0, 5) : value;
}

/** 打开评论详情页面。 */
function handleOpenCommentDetail(commentId: number) {
  if (!commentId) {
    ElMessage.warning("评论记录不存在");
    return;
  }
  void navigateTo(router, `/admin/comment/detail/${commentId}`);
}

/**
 * 统一处理工作台入口跳转。
 * @param path 目标路由路径。
 */
function handleNavigate(path: RouteLocationRaw) {
  if (!path) return;
  if (typeof path === "string") {
    void navigateTo(router, path);
    return;
  }
  void navigateTo(router, String(path.path ?? ""), (path.query ?? {}) as Record<string, string | number>);
}

/** 加载工作台顶部指标。 */
async function loadWorkspaceMetrics() {
  const response = await defWorkspaceService.SummaryWorkspaceMetrics({});
  Object.assign(metrics, response);
}

/** 加载工作台待处理事项。 */
async function loadWorkspaceTodo() {
  const response = await defWorkspaceService.SummaryWorkspaceTodo({});
  Object.assign(todoSummary, response);
}

/** 加载工作台风险提醒。 */
async function loadWorkspaceRisk() {
  const response = await defWorkspaceService.SummaryWorkspaceRisk({});
  Object.assign(riskSummary, response);
}

/** 加载工作台口碑洞察。 */
async function loadWorkspaceReputation() {
  const response = await defWorkspaceService.SummaryWorkspaceReputation({});
  Object.assign(reputationSummary, response);
}

/** 加载工作台待审核评价。 */
async function loadWorkspacePendingComments() {
  const response = await defWorkspaceService.ListWorkspacePendingComments({ limit: 5 });
  pendingComments.value = response.pending_comments ?? [];
}

/** 加载工作台全部数据。 */
async function loadWorkspaceData() {
  loading.value = true;
  try {
    const results = await refreshWorkspaceTargets(workspaceRefreshTargets);
    // 任一接口失败时保留已成功数据，避免工作台整页空白。
    if (results.some(item => item.status === "rejected")) {
      ElMessage.warning("部分工作台数据加载失败，请稍后重试");
    }
  } finally {
    loading.value = false;
  }
}

/** 按目标静默刷新工作台数据。 */
async function refreshWorkspaceTargets(targets: SseRefreshTarget[]) {
  const normalizedTargets = normalizeRefreshTargets(targets);
  const tasks = normalizedTargets.map(target => loadWorkspaceTarget(target));
  return Promise.allSettled(tasks);
}

/** 加载单个工作台刷新目标。 */
function loadWorkspaceTarget(target: SseRefreshTarget) {
  switch (target) {
    case SseRefreshTarget.SSE_REFRESH_TARGET_ADMIN_WORKSPACE_METRICS:
      return loadWorkspaceMetrics();
    case SseRefreshTarget.SSE_REFRESH_TARGET_ADMIN_WORKSPACE_TODO:
      return loadWorkspaceTodo();
    case SseRefreshTarget.SSE_REFRESH_TARGET_ADMIN_WORKSPACE_RISK:
      return loadWorkspaceRisk();
    case SseRefreshTarget.SSE_REFRESH_TARGET_ADMIN_WORKSPACE_REPUTATION:
      return loadWorkspaceReputation();
    case SseRefreshTarget.SSE_REFRESH_TARGET_ADMIN_WORKSPACE_PENDING_COMMENTS:
      return loadWorkspacePendingComments();
    default:
      return Promise.resolve();
  }
}

/** 去重并过滤需要刷新的工作台目标。 */
function normalizeRefreshTargets(targets: SseRefreshTarget[]) {
  return [...new Set(targets.filter(target => workspaceRefreshTargets.includes(target)))];
}

/** 订阅工作台刷新事件。 */
function startWorkspaceSse() {
  stopWorkspaceSse = subscribeSseRefresh(SseStream.SSE_STREAM_ADMIN, payload => {
    queueWorkspaceRefresh(payload.targets.filter(isWorkspaceRefreshTarget));
  });
}

/** 判断推送目标是否属于工作台当前支持的刷新目标。 */
function isWorkspaceRefreshTarget(target: SseRefreshTarget) {
  return workspaceRefreshTargets.includes(target);
}

/** 将短时间内的刷新目标合并，避免连续事件造成接口抖动。 */
function queueWorkspaceRefresh(targets: SseRefreshTarget[]) {
  normalizeRefreshTargets(targets).forEach(target => queuedRefreshTargets.add(target));
  if (queuedRefreshTargets.size === 0) {
    return;
  }

  if (workspaceRefreshTimer) {
    clearTimeout(workspaceRefreshTimer);
  }
  workspaceRefreshTimer = setTimeout(() => {
    const targets = [...queuedRefreshTargets];
    queuedRefreshTargets = new Set<SseRefreshTarget>();
    workspaceRefreshTimer = null;
    void refreshWorkspaceTargets(targets);
  }, 300);
}

onMounted(() => {
  startWorkspaceSse();
  void loadWorkspaceData();
});

onBeforeUnmount(() => {
  if (workspaceRefreshTimer) {
    clearTimeout(workspaceRefreshTimer);
    workspaceRefreshTimer = null;
  }
  stopWorkspaceSse?.();
  stopWorkspaceSse = null;
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
  border-radius: var(--admin-page-radius);
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
  background: var(--admin-page-card-bg-soft);
  box-shadow: 0 8px 18px rgb(15 23 42 / 12%);

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

.workspace-copy p {
  margin: 8px 0 0;
  font-size: 13px;
  color: var(--admin-page-text-secondary);
}

.metric-grid {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
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
  border-radius: var(--admin-page-radius);
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

.metric-card__link,
.panel-header__link {
  display: inline-flex;
  gap: 4px;
  align-items: center;
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

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
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

.todo-item,
.risk-item {
  display: flex;
  gap: 16px;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  padding: 16px 18px;
  text-align: left;
  border: 1px solid var(--admin-page-card-border-soft);
  border-radius: var(--admin-page-radius);
  background: var(--admin-page-card-bg-soft);
  transition:
    border-color 0.2s ease,
    transform 0.2s ease;
}

.todo-item__main,
.risk-item__main {
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
  border-radius: var(--admin-page-radius);
}

.todo-item strong,
.risk-item__title {
  display: block;
  color: var(--admin-page-text-primary);
}

.todo-item strong,
.risk-item__title {
  font-size: 16px;
  line-height: 1.35;
}

.workspace-item__desc {
  margin: 6px 0 0;
  font-size: 12px;
  line-height: 1.6;
  color: var(--admin-page-text-secondary);
}

.todo-item__side,
.risk-item__side {
  display: flex;
  flex-direction: column;
  gap: 4px;
  align-items: flex-end;
  flex-shrink: 0;
}

.todo-item__count,
.risk-item__count {
  font-size: 28px;
  font-weight: 700;
  line-height: 1;
  color: var(--admin-page-text-primary);
}

.todo-item__unit,
.risk-item__unit {
  font-size: 12px;
  color: var(--admin-page-text-placeholder);
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
  border-radius: var(--admin-page-radius);
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

.pending-comment-table {
  :deep(.el-table) {
    background: transparent;
  }

  :deep(.el-table__header-wrapper th.el-table__cell) {
    background: var(--admin-page-card-bg-soft);
  }
}

.reputation-list {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.reputation-item {
  padding: 16px 18px;
  border: 1px solid var(--admin-page-card-border-soft);
  border-radius: var(--admin-page-radius);
  background: var(--admin-page-card-bg-soft);

  span {
    font-size: 13px;
    color: var(--admin-page-text-secondary);
  }

  strong {
    display: block;
    margin-top: 8px;
    font-size: 18px;
    line-height: 1.4;
    color: var(--admin-page-text-primary);
  }

  p {
    margin: 6px 0 0;
    font-size: 12px;
    color: var(--admin-page-text-placeholder);
  }
}

@media (width <= 1400px) {
  .metric-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
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

  .todo-item,
  .risk-item {
    flex-direction: column;
    align-items: stretch;
  }

  .todo-item__side,
  .risk-item__side {
    flex-direction: row;
    align-items: baseline;
    justify-content: space-between;
  }
}
</style>
