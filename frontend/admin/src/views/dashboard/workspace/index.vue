<template>
  <div class="workspace-page">
    <section class="workspace-hero">
      <div class="hero-main">
        <div class="hero-user">
          <el-avatar class="hero-avatar" :src="avatarUrl" :size="88">
            {{ avatarFallback }}
          </el-avatar>
          <div class="hero-copy">
            <h1>{{ greetingText }}</h1>
            <p>{{ subtitleText }}</p>
          </div>
        </div>
      </div>
      <div class="hero-grid">
        <article v-for="item in overviewCards" :key="item.label" class="overview-card">
          <span class="overview-card__label">{{ item.label }}</span>
          <strong class="overview-card__value">{{ item.value }}</strong>
          <span class="overview-card__meta">{{ item.meta }}</span>
        </article>
      </div>
    </section>

    <section class="workspace-content">
      <el-card class="content-card focus-card" shadow="never">
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
        <el-card class="content-card schedule-card" shadow="never">
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

        <el-card class="content-card quick-card" shadow="never">
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
  background:
    radial-gradient(circle at top left, rgb(224 238 255 / 88%), transparent 26%),
    radial-gradient(circle at bottom right, rgb(255 237 223 / 72%), transparent 22%),
    linear-gradient(180deg, #f5f8fc 0%, #f2f6fb 100%);
}

.workspace-hero {
  padding: 28px;
  margin-bottom: 20px;
  background: linear-gradient(135deg, rgb(255 255 255 / 94%) 0%, rgb(246 250 255 / 96%) 50%, rgb(255 244 236 / 94%) 100%);
  border: 1px solid rgb(226 234 245 / 92%);
  border-radius: 28px;
  box-shadow: 0 22px 52px rgb(32 64 104 / 9%);
}

.hero-main {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 20px;
}

.hero-user {
  display: flex;
  gap: 18px;
  align-items: center;
}

.hero-avatar {
  flex-shrink: 0;
  border: 3px solid rgb(255 255 255 / 92%);
  box-shadow: 0 12px 28px rgb(30 64 175 / 14%);
}

.hero-copy h1 {
  margin: 0 0 10px;
  font-size: 34px;
  line-height: 1.2;
  color: #1d3150;
}

.hero-copy p {
  margin: 0;
  font-size: 14px;
  line-height: 1.8;
  color: #697c99;
}

.hero-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 16px;
  margin-top: 24px;
}

.overview-card {
  padding: 20px;
  background: rgb(255 255 255 / 82%);
  border: 1px solid rgb(229 236 246 / 92%);
  border-radius: 20px;
}

.overview-card__label {
  display: block;
  margin-bottom: 10px;
  font-size: 12px;
  color: #7990ad;
}

.overview-card__value {
  display: block;
  font-size: 20px;
  color: #213552;
}

.overview-card__meta {
  display: block;
  margin-top: 8px;
  font-size: 13px;
  color: #657991;
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

.content-card {
  border: 1px solid #e7eef7;
  border-radius: 24px;
  box-shadow: 0 18px 42px rgb(34 64 102 / 8%);
}

:deep(.content-card .el-card__header) {
  padding: 22px 24px 0;
  border-bottom: 0;
}

:deep(.content-card .el-card__body) {
  padding: 20px 24px 24px;
}

.panel-header h3 {
  margin: 0;
  font-size: 18px;
  color: #1f3251;
}

.panel-header p {
  margin: 8px 0 0;
  font-size: 13px;
  line-height: 1.7;
  color: #70819b;
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
  padding: 18px 20px;
  text-align: left;
  background: #f8fbff;
  border: 1px solid #ebf1f8;
  border-radius: 20px;
  transition: all 0.2s ease;
}

.focus-item:hover,
.quick-item:hover {
  transform: translateY(-1px);
  box-shadow: 0 14px 26px rgb(46 95 160 / 10%);
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
  width: 48px;
  height: 48px;
  font-size: 12px;
  font-weight: 700;
  color: #1d4ed8;
  background: #dce9ff;
  border-radius: 16px;
}

.focus-item strong,
.schedule-item strong,
.quick-item__title {
  display: block;
  font-size: 16px;
  color: #243754;
}

.focus-item p,
.schedule-item p,
.quick-item__desc {
  margin: 8px 0 0;
  font-size: 13px;
  line-height: 1.7;
  color: #6f839f;
}

.schedule-item {
  display: flex;
  gap: 14px;
  align-items: flex-start;
  padding: 18px 20px;
  background: #f8fbff;
  border: 1px solid #ebf1f8;
  border-radius: 20px;
}

.schedule-item__time {
  min-width: 56px;
  padding-top: 2px;
  font-size: 13px;
  font-weight: 700;
  color: #1d4ed8;
}

.quick-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
}

.quick-item {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  justify-content: center;
  min-height: 108px;
  padding: 18px;
  text-align: left;
  background: linear-gradient(180deg, #f8fbff 0%, #fff8f2 100%);
  border: 1px solid #ebf1f8;
  border-radius: 20px;
  transition: all 0.2s ease;
}

@media screen and (width <= 1080px) {
  .hero-main,
  .workspace-content {
    grid-template-columns: 1fr;
    flex-direction: column;
  }
}

@media screen and (width <= 840px) {
  .hero-grid,
  .quick-grid {
    grid-template-columns: 1fr;
  }
}

@media screen and (width <= 640px) {
  .workspace-page {
    padding: 16px;
  }

  .workspace-hero,
  :deep(.content-card .el-card__body) {
    padding: 20px;
  }

  .hero-user {
    align-items: flex-start;
  }

  .hero-copy h1 {
    font-size: 28px;
  }
}
</style>
