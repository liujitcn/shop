<template>
  <div v-if="canOpenAssistant" class="agent-assistant">
    <button
      ref="floatButtonRef"
      class="agent-float-button"
      :style="floatButtonStyle"
      type="button"
      aria-label="打开智能助手"
      @pointerdown="handlePointerDown"
      @click="handleFloatButtonClick"
    >
      <el-icon :size="22"><ChatDotRound /></el-icon>
      <span>AI</span>
    </button>

    <el-dialog
      v-model="dialogVisible"
      class="agent-assistant-dialog"
      width="1120px"
      :append-to-body="true"
      :close-on-click-modal="false"
      :destroy-on-close="false"
    >
      <template #header>
        <div class="agent-dialog-header">
          <div>
            <div class="agent-dialog-title">智能助手</div>
            <div class="agent-dialog-subtitle">统一 Agent 接口 · Element Plus X 会话</div>
          </div>
          <div class="agent-dialog-tags">
            <el-tag effect="plain" type="primary">admin</el-tag>
            <el-tag effect="plain" type="success">online</el-tag>
          </div>
        </div>
      </template>

      <div class="agent-dialog-body">
        <aside class="agent-session-panel">
          <div class="agent-panel-title">会话</div>
          <el-input v-model="sessionKeyword" placeholder="搜索" clearable :prefix-icon="Search" />
          <Conversations
            v-model:active="activeSessionID"
            class="agent-conversations"
            :items="filteredSessions"
            row-key="id"
            label-key="label"
            :show-tooltip="true"
            :label-height="72"
            @change="handleSessionChange"
          >
            <template #label="{ item }">
              <div class="agent-session-item">
                <div class="agent-session-name">{{ item.label }}</div>
                <div class="agent-session-meta">{{ item.scene }} · {{ item.toolCount }} tools</div>
              </div>
            </template>
          </Conversations>
        </aside>

        <main class="agent-chat-panel">
          <div class="agent-chat-header">
            <div>
              <div class="agent-chat-title">{{ activeSession?.label }}</div>
              <div class="agent-chat-desc">{{ activeSession?.summary }}</div>
            </div>
            <div class="agent-chat-tags">
              <el-tag effect="plain">{{ activeSession?.scene }}</el-tag>
              <el-tag effect="plain" type="success">context</el-tag>
            </div>
          </div>

          <BubbleList class="agent-message-list" :list="currentMessages" max-height="492px" :auto-scroll="true">
            <template #content="{ item }">
              <div v-if="item.kind === 'tool'" class="agent-tool-card">
                <div class="agent-card-title">
                  <el-icon><Operation /></el-icon>
                  工具调用
                </div>
                <div v-for="tool in item.tools" :key="tool.name" class="agent-tool-row">
                  <span class="agent-tool-status"></span>
                  <span class="agent-tool-name">{{ tool.name }}</span>
                  <span class="agent-tool-time">{{ tool.elapsed }}</span>
                </div>
              </div>
              <div v-else class="agent-message-content">{{ item.content }}</div>
            </template>
          </BubbleList>

          <div class="agent-sender-wrap">
            <XSender
              ref="senderRef"
              placeholder="输入问题或要执行的操作"
              submit-type="enter"
              :loading="sending"
              :clearable="true"
              @submit="handleSubmit"
            />
          </div>
        </main>

        <aside class="agent-context-panel">
          <section class="agent-side-section">
            <div class="agent-panel-title">上下文</div>
            <div class="agent-context-card">
              <span>terminal</span>
              <strong>admin</strong>
            </div>
            <div class="agent-context-card">
              <span>scene</span>
              <strong>{{ activeSession?.scene }}</strong>
            </div>
          </section>

          <section class="agent-confirm-card">
            <div class="agent-card-title">
              <el-icon><Warning /></el-icon>
              确认执行
            </div>
            <div class="agent-confirm-tool">execute_job</div>
            <div class="agent-confirm-risk">HIGH_RISK</div>
            <div class="agent-confirm-actions">
              <el-button size="small" plain>拒绝</el-button>
              <el-button size="small" type="warning">确认</el-button>
            </div>
          </section>

          <section class="agent-version-card">
            <div class="agent-card-title">第一版</div>
            <div>/api/v1/agent</div>
            <div>内存会话</div>
            <div>全量工具</div>
          </section>
        </aside>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts" name="AgentAssistant">
import "vue-element-plus-x/styles/index.css";
import { computed, onBeforeUnmount, reactive, ref } from "vue";
import { BubbleList, Conversations, XSender } from "vue-element-plus-x";
import { ChatDotRound, Operation, Search, Warning } from "@element-plus/icons-vue";
import { useAuthStore } from "@/stores/modules/auth";
import { useUserStore } from "@/stores/modules/user";

const ASSISTANT_OPEN_PERMISSION = "agent:assistant:open";
const FLOAT_POSITION_KEY = "agent-assistant-float-position";

/** Agent 会话列表项，用于驱动 Element Plus X Conversations。 */
interface AgentSessionItem {
  /** 会话唯一标识。 */
  id: string;
  /** 会话标题。 */
  label: string;
  /** 当前业务场景。 */
  scene: string;
  /** 会话摘要。 */
  summary: string;
  /** 示例工具数量。 */
  toolCount: number;
}

/** 工具调用展示项。 */
interface AgentToolCall {
  /** 工具名称。 */
  name: string;
  /** 工具耗时。 */
  elapsed: string;
}

/** 聊天消息展示项，兼容 BubbleList 的基础字段。 */
interface AgentMessageItem {
  /** 消息唯一标识。 */
  id: string;
  /** 消息内容。 */
  content: string;
  /** 气泡位置。 */
  placement: "start" | "end";
  /** 消息展示类型。 */
  kind?: "text" | "tool";
  /** 工具调用列表。 */
  tools?: AgentToolCall[];
  /** 气泡样式。 */
  variant?: "filled" | "borderless" | "outlined" | "shadow";
  /** 气泡圆角。 */
  shape?: "round" | "corner";
  /** 最大宽度。 */
  maxWidth?: string;
}

/** 悬浮按钮位置。 */
interface FloatPosition {
  /** 距离窗口左侧。 */
  left: number;
  /** 距离窗口顶部。 */
  top: number;
}

const authStore = useAuthStore();
const userStore = useUserStore();
const dialogVisible = ref(false);
const sessionKeyword = ref("");
const activeSessionID = ref("workspace-risk");
const sending = ref(false);
const floatButtonRef = ref<HTMLButtonElement>();
const senderRef = ref<InstanceType<typeof XSender>>();
const pointerStart = reactive({ x: 0, y: 0, left: 0, top: 0, moved: false, dragging: false });

const sessions = ref<AgentSessionItem[]>([
  {
    id: "workspace-risk",
    label: "今日经营风险",
    scene: "workspace",
    summary: "工作台 · 订单 · 评价",
    toolCount: 3
  },
  {
    id: "recommend-check",
    label: "推荐链路排查",
    scene: "recommend",
    summary: "Gorse · 推荐请求",
    toolCount: 4
  },
  {
    id: "comment-review",
    label: "评价审核建议",
    scene: "comment",
    summary: "评价 · 讨论 · 审核",
    toolCount: 2
  }
]);

const messages = ref<Record<string, AgentMessageItem[]>>({
  "workspace-risk": [
    {
      id: "m1",
      content: "看一下今天有哪些订单和评价风险",
      placement: "end",
      kind: "text",
      variant: "filled",
      shape: "round",
      maxWidth: "380px"
    },
    {
      id: "m2",
      content: "已查询工作台、订单分析和待审核评价。正在整理风险优先级。",
      placement: "start",
      kind: "text",
      variant: "filled",
      shape: "round",
      maxWidth: "420px"
    },
    {
      id: "m3",
      content: "",
      placement: "start",
      kind: "tool",
      tools: [
        { name: "workspace_summary_risk", elapsed: "82ms" },
        { name: "order_analytics_summary", elapsed: "104ms" }
      ],
      variant: "outlined",
      shape: "round",
      maxWidth: "430px"
    },
    {
      id: "m4",
      content: "待发货超时 3 单，退款中 2 单，待审核评价 8 条。建议先处理超时订单，再复核异常评价。",
      placement: "start",
      kind: "text",
      variant: "filled",
      shape: "round",
      maxWidth: "440px"
    }
  ],
  "recommend-check": [
    {
      id: "r1",
      content: "排查首页推荐曝光下降的原因",
      placement: "end",
      kind: "text",
      variant: "filled",
      shape: "round",
      maxWidth: "380px"
    },
    {
      id: "r2",
      content: "可以从推荐请求、Gorse 状态和热门兜底三个方向检查。",
      placement: "start",
      kind: "text",
      variant: "filled",
      shape: "round",
      maxWidth: "420px"
    }
  ],
  "comment-review": [
    {
      id: "c1",
      content: "汇总待审核评价里的高风险内容",
      placement: "end",
      kind: "text",
      variant: "filled",
      shape: "round",
      maxWidth: "380px"
    },
    {
      id: "c2",
      content: "待审核评价中优先关注连续差评、售后相关和疑似违规内容。",
      placement: "start",
      kind: "text",
      variant: "filled",
      shape: "round",
      maxWidth: "420px"
    }
  ]
});

const floatPosition = ref<FloatPosition>(loadFloatPosition());

const canOpenAssistant = computed(() => {
  const globalButtons = authStore.authButtonListGet.__global__ ?? [];
  const routeButtons = authStore.authButtonListGet[authStore.routeName] ?? [];
  const roleCode = userStore.userInfo.role_code;
  const isAdminRole = ["admin", "super_admin", "root"].includes(roleCode);

  // 首版权限数据可能还没有初始化按钮编码，管理员角色先允许验收入口。
  return isAdminRole || [...globalButtons, ...routeButtons].includes(ASSISTANT_OPEN_PERMISSION);
});

const filteredSessions = computed(() => {
  const keyword = sessionKeyword.value.trim();
  if (!keyword) return sessions.value;
  return sessions.value.filter(item => item.label.includes(keyword) || item.scene.includes(keyword));
});

const activeSession = computed(() => sessions.value.find(item => item.id === activeSessionID.value) ?? sessions.value[0]);

const currentMessages = computed(() => messages.value[activeSessionID.value] ?? []);

const floatButtonStyle = computed(() => ({
  left: `${floatPosition.value.left}px`,
  top: `${floatPosition.value.top}px`
}));

/** 读取悬浮入口本地位置，默认落在右下角。 */
function loadFloatPosition(): FloatPosition {
  try {
    const cachedPosition = localStorage.getItem(FLOAT_POSITION_KEY);
    if (cachedPosition) {
      const parsedPosition = JSON.parse(cachedPosition) as FloatPosition;
      if (Number.isFinite(parsedPosition.left) && Number.isFinite(parsedPosition.top)) return clampFloatPosition(parsedPosition);
    }
  } catch {
    // 本地缓存异常时使用默认位置，避免影响后台主流程。
  }

  return clampFloatPosition({
    left: window.innerWidth - 96,
    top: window.innerHeight - 112
  });
}

/** 限制悬浮按钮位置，避免拖到窗口外。 */
function clampFloatPosition(position: FloatPosition): FloatPosition {
  const buttonSize = 64;
  const pagePadding = 16;
  return {
    left: Math.min(Math.max(position.left, pagePadding), window.innerWidth - buttonSize - pagePadding),
    top: Math.min(Math.max(position.top, pagePadding), window.innerHeight - buttonSize - pagePadding)
  };
}

/** 保存悬浮入口位置。 */
function saveFloatPosition(position: FloatPosition) {
  localStorage.setItem(FLOAT_POSITION_KEY, JSON.stringify(position));
}

/** 记录拖动起点，区分拖动与点击打开。 */
function handlePointerDown(event: PointerEvent) {
  pointerStart.x = event.clientX;
  pointerStart.y = event.clientY;
  pointerStart.left = floatPosition.value.left;
  pointerStart.top = floatPosition.value.top;
  pointerStart.moved = false;
  pointerStart.dragging = true;
  floatButtonRef.value?.setPointerCapture(event.pointerId);
  window.addEventListener("pointermove", handlePointerMove);
  window.addEventListener("pointerup", handlePointerUp, { once: true });
}

/** 拖动悬浮按钮并实时限制边界。 */
function handlePointerMove(event: PointerEvent) {
  if (!pointerStart.dragging) return;

  const nextPosition = clampFloatPosition({
    left: pointerStart.left + event.clientX - pointerStart.x,
    top: pointerStart.top + event.clientY - pointerStart.y
  });
  pointerStart.moved = Math.abs(event.clientX - pointerStart.x) > 4 || Math.abs(event.clientY - pointerStart.y) > 4;
  floatPosition.value = nextPosition;
}

/** 拖动结束后持久化悬浮按钮位置。 */
function handlePointerUp() {
  pointerStart.dragging = false;
  saveFloatPosition(floatPosition.value);
  window.removeEventListener("pointermove", handlePointerMove);
}

/** 点击悬浮按钮打开智能助手弹窗。 */
function handleFloatButtonClick() {
  if (pointerStart.moved) return;
  dialogVisible.value = true;
}

/** 切换会话时同步当前活动会话。 */
function handleSessionChange(item: AgentSessionItem) {
  activeSessionID.value = item.id;
}

/** 提交用户输入，第一版前端先生成本地回显，后续替换为统一 Agent 接口。 */
function handleSubmit() {
  const inputText = senderRef.value?.getModelValue().text.trim();
  if (!inputText) return;

  const activeID = activeSessionID.value;
  const nextMessages = messages.value[activeID] ?? [];
  nextMessages.push({
    id: `${activeID}-${Date.now()}`,
    content: inputText,
    placement: "end",
    kind: "text",
    variant: "filled",
    shape: "round",
    maxWidth: "380px"
  });
  senderRef.value?.clear();
  sending.value = true;

  window.setTimeout(() => {
    nextMessages.push({
      id: `${activeID}-${Date.now()}-reply`,
      content: "已收到，后续会通过 /api/v1/agent 统一接口调用 Blades Agent 和工具链。",
      placement: "start",
      kind: "text",
      variant: "filled",
      shape: "round",
      maxWidth: "440px"
    });
    messages.value[activeID] = [...nextMessages];
    sending.value = false;
  }, 500);
}

onBeforeUnmount(() => {
  window.removeEventListener("pointermove", handlePointerMove);
});
</script>

<style scoped lang="scss">
.agent-assistant {
  position: fixed;
  inset: 0;
  z-index: 3000;
  pointer-events: none;
}

.agent-float-button {
  position: fixed;
  z-index: 3001;
  display: flex;
  gap: 4px;
  align-items: center;
  justify-content: center;
  width: 64px;
  height: 64px;
  color: #ffffff;
  cursor: grab;
  pointer-events: auto;
  touch-action: none;
  background: var(--el-color-primary);
  border: none;
  border-radius: 50%;
  box-shadow: 0 12px 28px rgb(64 158 255 / 32%);
  transition:
    box-shadow 0.2s ease,
    transform 0.2s ease;

  span {
    font-size: 14px;
    font-weight: 700;
    line-height: 1;
  }

  &:hover {
    box-shadow: 0 16px 34px rgb(64 158 255 / 40%);
    transform: translateY(-1px);
  }

  &:active {
    cursor: grabbing;
    transform: translateY(0);
  }
}

:global(.agent-assistant-dialog) {
  max-width: calc(100vw - 48px);
  margin-top: 8vh;
  overflow: hidden;
  border-radius: 8px;

  .el-dialog__header {
    padding: 0;
  }

  .el-dialog__body {
    padding: 0;
  }
}

.agent-dialog-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 56px;
  padding: 10px 28px;
  background: var(--el-bg-color);
  border-bottom: 1px solid var(--admin-page-divider-strong);
}

.agent-dialog-title {
  font-size: 17px;
  font-weight: 700;
  line-height: 24px;
  color: var(--admin-page-text-primary);
}

.agent-dialog-subtitle {
  margin-top: 2px;
  font-size: 12px;
  color: var(--admin-page-text-secondary);
}

.agent-dialog-tags,
.agent-chat-tags {
  display: flex;
  gap: 8px;
  align-items: center;
}

.agent-dialog-body {
  display: grid;
  grid-template-columns: 276px minmax(0, 1fr) 224px;
  height: min(752px, calc(100vh - 144px));
  min-height: 620px;
  background: var(--el-bg-color);
}

.agent-session-panel {
  min-width: 0;
  padding: 28px;
  overflow: hidden;
  background: var(--admin-page-card-bg-soft);
  border-right: 1px solid var(--admin-page-divider-strong);
}

.agent-panel-title {
  margin-bottom: 14px;
  font-size: 15px;
  font-weight: 700;
  color: var(--admin-page-text-primary);
}

.agent-conversations {
  height: calc(100% - 70px);
  margin-top: 20px;
}

.agent-session-item {
  min-width: 0;
}

.agent-session-name {
  overflow: hidden;
  font-size: 14px;
  font-weight: 700;
  line-height: 22px;
  color: var(--admin-page-text-primary);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.agent-session-meta {
  margin-top: 4px;
  overflow: hidden;
  font-size: 12px;
  line-height: 18px;
  color: var(--admin-page-text-secondary);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.agent-chat-panel {
  display: flex;
  min-width: 0;
  padding: 28px 32px 30px;
  overflow: hidden;
  flex-direction: column;
  background: var(--el-bg-color);
}

.agent-chat-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  padding-bottom: 18px;
}

.agent-chat-title {
  font-size: 17px;
  font-weight: 700;
  line-height: 24px;
  color: var(--admin-page-text-primary);
}

.agent-chat-desc {
  margin-top: 4px;
  font-size: 12px;
  color: var(--admin-page-text-secondary);
}

.agent-message-list {
  flex: 1;
  min-height: 0;
  padding-right: 4px;
}

.agent-message-content {
  line-height: 24px;
  white-space: pre-wrap;
}

.agent-tool-card {
  min-width: 360px;
  padding: 6px 2px;
}

.agent-card-title {
  display: flex;
  gap: 6px;
  align-items: center;
  margin-bottom: 14px;
  font-size: 14px;
  font-weight: 700;
  color: var(--admin-page-text-primary);
}

.agent-tool-row {
  display: grid;
  grid-template-columns: 14px minmax(0, 1fr) auto;
  gap: 10px;
  align-items: center;
  min-height: 28px;
  font-size: 13px;
  color: var(--admin-page-text-secondary);
}

.agent-tool-status {
  width: 8px;
  height: 8px;
  background: var(--el-color-success);
  border-radius: 50%;
}

.agent-tool-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.agent-tool-time {
  color: var(--admin-page-text-placeholder);
}

.agent-sender-wrap {
  padding-top: 16px;
}

.agent-context-panel {
  min-width: 0;
  padding: 28px 24px;
  overflow: auto;
  background: var(--admin-page-card-bg-soft);
  border-left: 1px solid var(--admin-page-divider-strong);
}

.agent-side-section {
  margin-bottom: 26px;
}

.agent-context-card,
.agent-version-card,
.agent-confirm-card {
  padding: 16px;
  background: var(--el-bg-color);
  border: 1px solid var(--admin-page-card-border);
  border-radius: var(--admin-page-radius);
}

.agent-context-card {
  margin-bottom: 12px;

  span {
    display: block;
    margin-bottom: 8px;
    font-size: 12px;
    color: var(--admin-page-text-secondary);
  }

  strong {
    font-size: 14px;
    color: var(--admin-page-text-primary);
  }
}

.agent-confirm-card {
  margin-bottom: 26px;
  background: #fff7ed;
  border-color: #fed7aa;

  .agent-card-title,
  .agent-confirm-tool,
  .agent-confirm-risk {
    color: #9a3412;
  }
}

.agent-confirm-tool,
.agent-confirm-risk,
.agent-version-card {
  font-size: 12px;
  line-height: 24px;
  color: var(--admin-page-text-secondary);
}

.agent-confirm-actions {
  display: flex;
  gap: 8px;
  margin-top: 14px;
}

.agent-version-card {
  .agent-card-title {
    margin-bottom: 10px;
  }
}

html.dark {
  .agent-float-button {
    box-shadow: 0 12px 28px rgb(64 158 255 / 20%);
  }

  .agent-confirm-card {
    background: rgb(249 115 22 / 12%);
    border-color: rgb(249 115 22 / 30%);
  }
}

@media screen and (max-width: 1200px) {
  :global(.agent-assistant-dialog) {
    width: calc(100vw - 32px) !important;
    margin-top: 5vh;
  }

  .agent-dialog-body {
    grid-template-columns: 240px minmax(0, 1fr);
    height: min(760px, calc(100vh - 104px));
  }

  .agent-context-panel {
    display: none;
  }
}

@media screen and (max-width: 768px) {
  .agent-float-button {
    width: 56px;
    height: 56px;
  }

  .agent-dialog-body {
    grid-template-columns: 1fr;
    min-height: 560px;
  }

  .agent-session-panel {
    display: none;
  }

  .agent-chat-panel {
    padding: 20px 18px 22px;
  }

  .agent-dialog-subtitle,
  .agent-chat-tags {
    display: none;
  }
}
</style>
