<template>
  <div class="ai-assistant-page" :class="{ 'is-session-collapsed': sessionPanelCollapsed }">
    <SessionPanel
      v-if="!sessionPanelCollapsed"
      v-model:active="activeSessionID"
      v-model:keyword="sessionKeyword"
      :sessions="filteredSessions"
      @change="handleSessionChange"
      @action="handleSessionAction"
      @create="handleCreateSession"
      @toggle-collapse="toggleSessionPanel"
    />
    <div v-else class="agent-session-collapsed">
      <button class="agent-session-collapsed__toggle" type="button" aria-label="展开会话栏" @click="toggleSessionPanel">
        <el-icon><DArrowRight /></el-icon>
      </button>
      <span class="agent-session-collapsed__label">最近对话</span>
    </div>
    <ChatPanel :active-session="activeSession" :messages="currentMessages" :sending="sending" @submit="handleSubmit" />
  </div>
</template>

<script setup lang="ts">
import "vue-element-plus-x/styles/index.css";
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import { DArrowRight } from "@element-plus/icons-vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { defAiAssistantService } from "@/api/base/ai_assistant";
import {
  subscribeAiAssistantDelta,
  subscribeAiAssistantError,
  subscribeAiAssistantFinish,
  type SseAiAssistantPayload
} from "@/api/base/sse";
import type { AiAssistantSession } from "@/rpc/base/v1/ai_assistant";
import { Terminal } from "@/rpc/common/v1/enum";
import ChatPanel from "./components/ChatPanel.vue";
import SessionPanel from "./components/SessionPanel.vue";
import {
  appendStreamingDelta,
  createLocalUserMessage,
  createThinkingMessage,
  markStreamingError,
  markThinkingMessageFailed,
  normalizeMessageList,
  normalizeSession,
  normalizeSessionList,
  resolveTimestamp,
  replacePendingMessages,
  sortMessages
} from "./message";
import type { ChatMessageItem, SessionAction, SessionListItem, SubmitPayload } from "./types";

defineOptions({
  name: "AiAssistant"
});

const sessionKeyword = ref("");
const activeSessionID = ref("");
const sessionPanelCollapsed = ref(false);
const sending = ref(false);
const loadingSessions = ref(false);
const loadingSessionID = ref("");
const sessions = ref<AiAssistantSession[]>([]);
const messages = ref<Record<string, ChatMessageItem[]>>({});
const sseStops: Array<() => void> = [];

const sessionItems = computed<SessionListItem[]>(() =>
  sessions.value.map(item => ({
    ...item,
    label: item.title
  }))
);

const filteredSessions = computed(() => {
  const keyword = sessionKeyword.value.trim();
  if (!keyword) return sessionItems.value;
  return sessionItems.value.filter(item => item.title.includes(keyword) || item.scene.includes(keyword));
});

const activeSession = computed(() => sessions.value.find(item => item.id === activeSessionID.value) ?? sessions.value[0]);

const currentMessages = computed(() => messages.value[activeSessionID.value] ?? []);

/** 切换会话时同步当前活动会话。 */
function handleSessionChange(item: SessionListItem) {
  activeSessionID.value = item.id;
  void loadMessages(item.id);
}

/** 切换会话侧栏折叠状态。 */
function toggleSessionPanel() {
  sessionPanelCollapsed.value = !sessionPanelCollapsed.value;
}

/** 处理会话项菜单动作，接入真实重命名与删除接口。 */
async function handleSessionAction(payload: { action: SessionAction; item: SessionListItem }) {
  try {
    if (payload.action === "rename") {
      await handleRenameSession(payload.item);
      return;
    }
    await handleDeleteSession(payload.item);
  } catch (error) {
    if (error === "cancel" || error === "close") {
      ElMessage.info("已取消操作");
    }
  }
}

/** 主动创建新的会话，并切换到刚创建的会话。 */
async function handleCreateSession() {
  const sessionID = await createSession();
  if (!sessionID) return;
  activeSessionID.value = sessionID;
  messages.value[sessionID] = [];
  ElMessage.success("已创建新会话");
}

/** 提交用户输入并同步消息流。 */
async function handleSubmit(payload: SubmitPayload) {
  // 已有发送中的请求时，直接忽略重复提交，避免同一轮输入被并发发送多次。
  if (sending.value) return;
  sending.value = true;

  const sessionID = await ensureActiveSession();
  if (!sessionID) {
    sending.value = false;
    return;
  }

  const clientMessageId = payload.clientMessageId || `assistant-stream-${Date.now()}`;
  const localUserMessage = createLocalUserMessage(payload);
  const thinkingMessage = createThinkingMessage(clientMessageId);
  messages.value[sessionID] = sortMessages([...(messages.value[sessionID] ?? []), localUserMessage, thinkingMessage]);
  try {
    const response = await defAiAssistantService.SendAiAssistantMessage({
      session_id: sessionID,
      content: payload.text,
      client_message_id: clientMessageId,
      attachments: payload.attachments.map(item => ({
        id: item.id,
        name: item.name,
        size: item.size,
        url: item.url,
        mime_type: item.mime_type
      }))
    });

    const nextMessages = normalizeMessageList(response?.messages);
    messages.value[sessionID] = replacePendingMessages(messages.value[sessionID] ?? [], nextMessages);
    if (response?.session) upsertSession(normalizeSession(response.session));
  } catch {
    messages.value[sessionID] = markThinkingMessageFailed(messages.value[sessionID] ?? []);
  } finally {
    sending.value = false;
  }
}

/** 处理 AI 助手流式文本增量。 */
function handleAiAssistantDelta(payload: SseAiAssistantPayload) {
  const sessionID = String(payload.session_id ?? "");
  if (!sessionID || !messages.value[sessionID]) return;
  messages.value[sessionID] = appendStreamingDelta(messages.value[sessionID] ?? [], payload);
}

/** 处理 AI 助手流式结束事件。 */
function handleAiAssistantFinish(payload: SseAiAssistantPayload) {
  const sessionID = String(payload.session_id ?? "");
  if (!sessionID) return;
  const nextMessages = normalizeMessageList(payload.messages as never[]);
  messages.value[sessionID] = replacePendingMessages(messages.value[sessionID] ?? [], nextMessages);
  if (payload.session) {
    upsertSession(normalizeSession(payload.session as AiAssistantSession));
  }
}

/** 处理 AI 助手流式异常事件。 */
function handleAiAssistantError(payload: SseAiAssistantPayload) {
  const sessionID = String(payload.session_id ?? "");
  if (!sessionID || !messages.value[sessionID]) return;
  messages.value[sessionID] = markStreamingError(messages.value[sessionID] ?? [], payload);
}

/** 首次打开时加载会话列表，并拉取当前会话消息。 */
async function ensureSessionsLoaded() {
  if (loadingSessions.value || sessions.value.length > 0) return;

  loadingSessions.value = true;
  try {
    const response = await defAiAssistantService.ListAiAssistantSessions({ terminal: Terminal.TERMINAL_ADMIN });
    sessions.value = normalizeSessionList(response?.sessions);
    const sessionID = await ensureActiveSession();
    if (sessionID) await loadMessages(sessionID);
  } catch {
    sessions.value = [];
  } finally {
    loadingSessions.value = false;
  }
}

/** 加载指定会话的消息记录。 */
async function loadMessages(sessionID: string) {
  if (!sessionID) return;

  loadingSessionID.value = sessionID;
  try {
    const response = await defAiAssistantService.ListAiAssistantMessages({ session_id: sessionID });
    if (loadingSessionID.value !== sessionID) return;
    messages.value[sessionID] = normalizeMessageList(response?.messages);
  } catch {
    if (loadingSessionID.value === sessionID) {
      messages.value[sessionID] = [];
    }
  } finally {
    if (loadingSessionID.value === sessionID) {
      loadingSessionID.value = "";
    }
  }
}

/** 重命名当前会话。 */
async function handleRenameSession(item: AiAssistantSession) {
  const { value } = await ElMessageBox.prompt(`请输入新的会话名称\n当前名称：${item.title}`, "重命名会话", {
    confirmButtonText: "确认",
    cancelButtonText: "取消",
    inputValue: item.title,
    inputPattern: /\S+/,
    inputErrorMessage: "请输入会话名称"
  });

  const session = await defAiAssistantService.UpdateAiAssistantSession({ id: item.id, title: value.trim() });
  upsertSession(normalizeSession(session));
  ElMessage.success("会话已重命名");
}

/** 删除当前会话，并自动切换到剩余会话。 */
async function handleDeleteSession(item: AiAssistantSession) {
  await ElMessageBox.confirm(`是否删除该会话？\n会话名称：${item.title}`, "删除会话", {
    confirmButtonText: "确认删除",
    cancelButtonText: "取消",
    type: "warning"
  });

  await defAiAssistantService.DeleteAiAssistantSession({ id: item.id });
  sessions.value = sessions.value.filter(session => session.id !== item.id);
  delete messages.value[item.id];

  if (activeSessionID.value === item.id) {
    activeSessionID.value = sessions.value[0]?.id ?? "";
  }
  const nextSessionID = await ensureActiveSession();
  if (nextSessionID) await loadMessages(nextSessionID);

  ElMessage.success("会话已删除");
}

/** 保证当前存在可用会话；当列表为空时自动创建首个会话。 */
async function ensureActiveSession() {
  if (!activeSessionID.value && sessions.value.length > 0) {
    activeSessionID.value = sessions.value[0].id;
  }
  if (activeSessionID.value) return activeSessionID.value;

  activeSessionID.value = (await createSession()) ?? "";
  return activeSessionID.value;
}

/** 创建新的助手会话，并同步到本地列表。 */
async function createSession() {
  const session = await defAiAssistantService.CreateAiAssistantSession({
    title: "新对话",
    scene: "workspace",
    terminal: Terminal.TERMINAL_ADMIN
  });
  const normalizedSession = normalizeSession(session);
  upsertSession(normalizedSession);
  return normalizedSession.id;
}

/** 更新或插入会话，并按更新时间排序。 */
function upsertSession(session: AiAssistantSession) {
  const nextList = sessions.value.filter(item => item.id !== session.id);
  nextList.unshift(session);
  sessions.value = nextList.sort((left, right) => {
    const leftTime = resolveTimestamp(left.updated_at);
    const rightTime = resolveTimestamp(right.updated_at);
    return rightTime - leftTime;
  });
}

/** 页面加载后主动准备首个会话，避免进入菜单后仍需额外点击。 */
onMounted(() => {
  sseStops.push(subscribeAiAssistantDelta(handleAiAssistantDelta));
  sseStops.push(subscribeAiAssistantFinish(handleAiAssistantFinish));
  sseStops.push(subscribeAiAssistantError(handleAiAssistantError));
  void ensureSessionsLoaded();
});

onBeforeUnmount(() => {
  while (sseStops.length > 0) {
    const stop = sseStops.pop();
    stop?.();
  }
});
</script>

<style scoped lang="scss">
.ai-assistant-page {
  box-sizing: border-box;
  display: grid;
  grid-template-columns: 320px minmax(0, 1fr);
  height: 100%;
  min-height: 0;
  overflow: hidden;
  background: var(--admin-page-card-bg);
  border: 1px solid var(--admin-page-divider-strong);
  border-radius: var(--admin-page-radius);
  box-shadow: var(--admin-page-shadow);
}

.ai-assistant-page.is-session-collapsed {
  grid-template-columns: 44px minmax(0, 1fr);
}

.agent-session-collapsed {
  display: flex;
  gap: 12px;
  min-height: 0;
  padding: 16px 8px;
  align-items: center;
  flex-direction: column;
  background: var(--admin-page-card-bg);
  border-right: 1px solid var(--admin-page-divider-strong);
}

.agent-session-collapsed__toggle {
  display: inline-flex;
  width: 28px;
  height: 28px;
  color: var(--admin-page-text-secondary);
  cursor: pointer;
  align-items: center;
  justify-content: center;
  background: var(--el-fill-color-light);
  border: 0;
  border-radius: var(--admin-page-radius);
  transition:
    color 0.2s ease,
    background-color 0.2s ease;

  &:hover {
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
  }
}

.agent-session-collapsed__label {
  font-size: 12px;
  line-height: 16px;
  color: var(--admin-page-text-secondary);
  writing-mode: vertical-rl;
  user-select: none;
}

@media screen and (max-width: 1200px) {
  .ai-assistant-page {
    grid-template-columns: 264px minmax(0, 1fr);
  }

  .ai-assistant-page.is-session-collapsed {
    grid-template-columns: 44px minmax(0, 1fr);
  }
}

@media screen and (max-width: 768px) {
  .ai-assistant-page {
    grid-template-columns: 1fr;
    height: 100%;
    min-height: 0;
  }

  .agent-session-collapsed {
    display: none;
  }
}
</style>
