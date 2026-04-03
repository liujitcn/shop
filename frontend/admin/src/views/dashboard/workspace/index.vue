<template>
  <div class="workspace-page">
    <el-card class="workspace-card workspace-card--welcome" shadow="never">
      <div class="workspace-welcome">
        <div class="workspace-user">
          <el-avatar class="workspace-avatar" :src="avatarUrl" :size="64">
            {{ avatarFallback }}
          </el-avatar>
          <div class="workspace-copy">
            <span class="workspace-copy__label">今日工作概览</span>
            <h1>{{ greetingText }}</h1>
            <p>{{ subtitleText }}</p>
          </div>
        </div>

        <div class="workspace-summary">
          <article v-for="item in overviewCards" :key="item.label" class="summary-item">
            <span class="summary-item__label">{{ item.label }}</span>
            <strong class="summary-item__value">{{ item.value }}</strong>
            <span class="summary-item__meta">{{ item.meta }}</span>
          </article>
        </div>
      </div>
    </el-card>

    <section class="workspace-content">
      <el-card class="workspace-card" shadow="never">
        <template #header>
          <div class="panel-header">
            <div>
              <h3>今日关注</h3>
              <p>先处理最容易影响运营节奏的事项。</p>
            </div>
          </div>
        </template>
        <div class="focus-list">
          <button v-for="item in focusItems" :key="item.title" type="button" class="focus-item" @click="navigateTo(item.path)">
            <div class="focus-item__left">
              <span class="focus-item__badge">{{ item.badge }}</span>
              <div>
                <strong>{{ item.title }}</strong>
                <p>{{ item.description }}</p>
              </div>
            </div>
            <el-icon><ArrowRight /></el-icon>
          </button>
        </div>
      </el-card>

      <div class="content-side">
        <el-card class="workspace-card" shadow="never">
          <template #header>
            <div class="panel-header">
              <div>
                <h3>工作建议</h3>
                <p>按固定节奏检查关键模块。</p>
              </div>
            </div>
          </template>
          <div class="schedule-list">
            <div v-for="item in scheduleItems" :key="item.title" class="schedule-item">
              <span class="schedule-item__time">{{ item.time }}</span>
              <div>
                <strong>{{ item.title }}</strong>
                <p>{{ item.description }}</p>
              </div>
            </div>
          </div>
        </el-card>

        <el-card class="workspace-card" shadow="never">
          <template #header>
            <div class="panel-header">
              <div>
                <h3>快捷入口</h3>
                <p>常用模块直接进入。</p>
              </div>
            </div>
          </template>
          <div class="quick-grid">
            <button
              v-for="item in quickEntries"
              :key="item.title"
              type="button"
              class="quick-item"
              @click="navigateTo(item.path)"
            >
              <span class="quick-item__title">{{ item.title }}</span>
              <span class="quick-item__desc">{{ item.description }}</span>
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

import { computed } from "vue";
import { useRouter } from "vue-router";
import { ArrowRight } from "@element-plus/icons-vue";
import { useUserStore } from "@/stores/modules/user";

/** 工作台概览卡片。 */
interface OverviewCard {
  /** 标题。 */
  label: string;
  /** 主值。 */
  value: string;
  /** 辅助说明。 */
  meta: string;
}

/** 工作台待办项。 */
interface WorkspaceActionItem {
  /** 标题。 */
  title: string;
  /** 描述。 */
  description: string;
  /** 跳转路径。 */
  path: string;
  /** 徽标文案。 */
  badge?: string;
  /** 时间文案。 */
  time?: string;
}

const router = useRouter();
const userStore = useUserStore();

/** 当前显示名称，优先昵称。 */
const displayName = computed(() => {
  return userStore.userInfo.nickName || userStore.userInfo.userName || "管理员";
});

/** 头像兜底文案。 */
const avatarFallback = computed(() => {
  return displayName.value.slice(0, 1);
});

/** 当前头像地址。 */
const avatarUrl = computed(() => {
  return userStore.userInfo.avatar || "";
});

/** 根据当前时段生成问候语。 */
const greetingText = computed(() => {
  const hour = new Date().getHours();
  if (hour < 6) return `凌晨好，${displayName.value}`;
  if (hour < 11) return `上午好，${displayName.value}`;
  if (hour < 14) return `中午好，${displayName.value}`;
  if (hour < 18) return `下午好，${displayName.value}`;
  return `晚上好，${displayName.value}`;
});

/** 工作台副标题。 */
const subtitleText = computed(() => {
  return "先处理订单、商品和店铺运营，再回看系统配置与账号安全。";
});

/** 工作台概览数据。 */
const overviewCards = computed<OverviewCard[]>(() => {
  return [
    {
      label: "今日重点",
      value: "订单与商品",
      meta: "优先检查发货与库存"
    },
    {
      label: "处理节奏",
      value: "先订单后运营",
      meta: "上午看履约，下午看投放"
    },
    {
      label: "今日建议",
      value: "先巡检关键模块",
      meta: "避免问题积压到收尾阶段"
    }
  ];
});

/** 今日关注项。 */
const focusItems: WorkspaceActionItem[] = [
  {
    title: "订单管理",
    description: "查看待处理订单与发货进度。",
    path: "/order/info",
    badge: "订单"
  },
  {
    title: "商品管理",
    description: "检查商品资料、库存和上架状态。",
    path: "/goods/info",
    badge: "商品"
  },
  {
    title: "店铺运营",
    description: "维护轮播图、热门推荐和服务配置。",
    path: "/shop/banner",
    badge: "运营"
  }
];

/** 建议检查节奏。 */
const scheduleItems: WorkspaceActionItem[] = [
  {
    time: "09:30",
    title: "核对订单",
    description: "确认新单、支付状态与异常单。",
    path: ""
  },
  {
    time: "14:00",
    title: "巡检商品",
    description: "检查价格、库存和上下架状态。",
    path: ""
  },
  {
    time: "17:30",
    title: "回看运营位",
    description: "确认首页推荐和服务配置是否生效。",
    path: ""
  }
];

/** 快捷入口。 */
const quickEntries: WorkspaceActionItem[] = [
  {
    title: "用户管理",
    description: "账号、角色、部门",
    path: "/base/user"
  },
  {
    title: "系统配置",
    description: "站点与基础配置",
    path: "/base/config"
  },
  {
    title: "轮播图",
    description: "店铺首页投放",
    path: "/shop/banner"
  },
  {
    title: "热门推荐",
    description: "商品推荐位",
    path: "/shop/hot"
  }
];

/** 跳转到目标页面。 */
function navigateTo(path: string) {
  if (!path) return;
  router.push(path);
}
</script>

<style lang="scss" scoped>
.workspace-page {
  min-height: 100%;
  padding: 24px;
  background: #f5f7fb;
}

.workspace-card {
  border: 1px solid #e5eaf1;
  border-radius: 16px;
  box-shadow: 0 8px 24px rgb(15 23 42 / 4%);
}

.workspace-card--welcome {
  margin-bottom: 20px;
}

:deep(.workspace-card .el-card__header) {
  padding: 18px 20px 0;
  border-bottom: 0;
}

:deep(.workspace-card .el-card__body) {
  padding: 18px 20px 20px;
}

.workspace-welcome {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
}

.workspace-user {
  display: flex;
  gap: 16px;
  align-items: center;
  min-width: 0;
}

.workspace-avatar {
  flex-shrink: 0;
  border: 1px solid #e5eaf1;
}

.workspace-copy h1 {
  margin: 0 0 8px;
  font-size: 24px;
  line-height: 1.3;
  color: #1f2937;
}

.workspace-copy__label {
  display: inline-flex;
  margin-bottom: 6px;
  font-size: 12px;
  font-weight: 600;
  color: #64748b;
}

.workspace-copy p {
  margin: 0;
  font-size: 14px;
  line-height: 1.7;
  color: #64748b;
}

.workspace-summary {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
  min-width: 420px;
}

.summary-item {
  padding: 16px;
  background: #f8fafc;
  border: 1px solid #e8edf4;
  border-radius: 12px;
}

.summary-item__label {
  display: block;
  margin-bottom: 8px;
  font-size: 12px;
  color: #64748b;
}

.summary-item__value {
  display: block;
  font-size: 18px;
  color: #1f2937;
}

.summary-item__meta {
  display: block;
  margin-top: 8px;
  font-size: 13px;
  color: #94a3b8;
}

.workspace-content {
  display: grid;
  grid-template-columns: minmax(0, 1.25fr) minmax(320px, 0.9fr);
  gap: 20px;
}

.content-side {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.panel-header h3 {
  margin: 0;
  font-size: 16px;
  color: #1f2937;
}

.panel-header p {
  margin: 8px 0 0;
  font-size: 13px;
  line-height: 1.7;
  color: #64748b;
}

.focus-list,
.schedule-list {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.focus-item,
.quick-item {
  cursor: pointer;
  background: transparent;
  border: 0;
}

.focus-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
  width: 100%;
  padding: 16px 18px;
  text-align: left;
  background: #f8fafc;
  border: 1px solid #e8edf4;
  border-radius: 12px;
  transition: border-color 0.2s ease;
}

.focus-item:hover,
.quick-item:hover {
  border-color: #cdd7e5;
}

.focus-item__left {
  display: flex;
  gap: 14px;
  align-items: flex-start;
}

.focus-item__badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  font-size: 12px;
  font-weight: 700;
  color: #2563eb;
  background: #eaf2ff;
  border-radius: 10px;
}

.focus-item strong,
.schedule-item strong,
.quick-item__title {
  display: block;
  font-size: 15px;
  color: #1f2937;
}

.focus-item p,
.schedule-item p,
.quick-item__desc {
  margin: 8px 0 0;
  font-size: 13px;
  line-height: 1.7;
  color: #64748b;
}

.schedule-item {
  display: flex;
  gap: 14px;
  align-items: flex-start;
  padding: 16px 18px;
  background: #f8fafc;
  border: 1px solid #e8edf4;
  border-radius: 12px;
}

.schedule-item__time {
  min-width: 56px;
  padding-top: 2px;
  font-size: 13px;
  font-weight: 700;
  color: #2563eb;
}

.quick-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
}

.workspace-card :deep(.el-card__header) {
  position: relative;
}

.workspace-card :deep(.el-card__header)::after {
  position: absolute;
  right: 20px;
  bottom: 0;
  left: 20px;
  height: 1px;
  content: "";
  background: #eef2f7;
}

.quick-item {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  justify-content: center;
  min-height: 96px;
  padding: 16px;
  text-align: left;
  background: #f8fafc;
  border: 1px solid #e8edf4;
  border-radius: 12px;
  transition: border-color 0.2s ease;
}

@media screen and (width <= 1080px) {
  .workspace-welcome,
  .workspace-content {
    flex-direction: column;
  }

  .workspace-summary {
    width: 100%;
    min-width: 0;
  }
}

@media screen and (width <= 840px) {
  .workspace-summary,
  .quick-grid {
    grid-template-columns: 1fr;
  }
}

@media screen and (width <= 640px) {
  .workspace-page {
    padding: 16px;
  }

  .workspace-user {
    align-items: flex-start;
  }

  .workspace-copy h1 {
    font-size: 22px;
  }
}
</style>
