<template>
  <div class="ai-assistant-page" :class="{ 'is-session-collapsed': sessionPanelCollapsed }">
    <SessionPanel
      v-if="!sessionPanelCollapsed"
      v-model:active="activeSessionID"
      v-model:keyword="sessionKeyword"
      :sessions="filteredSessions"
      @change="handleSessionChange"
      @action="handleSessionAction"
      @toggle-collapse="toggleSessionPanel"
    />
    <div v-else class="agent-session-collapsed">
      <button class="agent-session-collapsed__toggle" type="button" aria-label="展开会话栏" @click="toggleSessionPanel">
        <el-icon><DArrowRight /></el-icon>
      </button>
      <span class="agent-session-collapsed__label">最近对话</span>
    </div>
    <ChatPanel
      :active-session="activeSession"
      :messages="currentMessages"
      :sending="sending"
      @submit="handleSubmit"
      @confirm-action="handleConfirmAction"
    />
  </div>
</template>

<script setup lang="ts">
import "vue-element-plus-x/styles/index.css";
import { computed, onMounted, ref } from "vue";
import { DArrowRight } from "@element-plus/icons-vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { defAiAssistantService } from "@/api/base/ai_assistant";
import type { AiAssistantAttachment, AiAssistantMessage, AiAssistantSession } from "@/rpc/base/v1/ai_assistant";
import ChatPanel from "./components/ChatPanel.vue";
import type { ChatMessageItem } from "./components/ChatPanel.vue";
import SessionPanel from "./components/SessionPanel.vue";

defineOptions({
  name: "DashboardAssistant"
});

type SessionAction = "rename" | "delete";

type SessionListItem = AiAssistantSession & {
  label: string;
};

type SubmitPayload = {
  text: string;
  attachments: AiAssistantAttachment[];
};

type ConfirmActionPayload = {
  action: "confirm" | "reject";
  message: ChatMessageItem;
  formValues: Record<string, string>;
};

const sessionKeyword = ref("");
const activeSessionID = ref("");
const sessionPanelCollapsed = ref(false);
const sending = ref(false);
const loadingSessions = ref(false);
const loadingMessages = ref(false);
const sessions = ref<AiAssistantSession[]>([]);
const messages = ref<Record<string, ChatMessageItem[]>>({});

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

/** 提交用户输入并同步消息流。 */
async function handleSubmit(payload: SubmitPayload) {
  const sessionID = await ensureActiveSession();
  if (!sessionID) return;

  sending.value = true;
  try {
    const response = await defAiAssistantService.SendAiAssistantMessage({
      session_id: sessionID,
      content: payload.text,
      attachments: payload.attachments.map(item => ({
        id: item.id,
        name: item.name,
        size: item.size,
        url: item.url,
        mime_type: item.mime_type
      }))
    });

    const nextMessages = normalizeMessageList(response?.messages);
    messages.value[sessionID] = [...(messages.value[sessionID] ?? []), ...nextMessages];
    if (response?.session) upsertSession(response.session);
  } catch {
    // 错误提示已由统一请求拦截器处理，这里仅拦截未捕获 Promise，避免页面继续抛红。
  } finally {
    sending.value = false;
  }
}

/** 处理确认卡动作，并继续复用现有消息发送链路。 */
async function handleConfirmAction(payload: ConfirmActionPayload) {
  const sessionID = activeSessionID.value;
  if (!sessionID || sending.value) return;
  if (payload.action === "confirm" && !validateConfirmForm(payload.message, payload.formValues)) return;

  updateConfirmMessageState(sessionID, payload.message.id, "processing");
  try {
    const response = await defAiAssistantService.OperateAiAssistantConfirm({
      session_id: sessionID,
      message_id: payload.message.id,
      action: payload.action === "confirm" ? "approve" : "reject",
      form_json: JSON.stringify(payload.formValues ?? {})
    });

    mergeSessionMessages(sessionID, normalizeMessageList(response?.messages));
    if (response?.session) upsertSession(response.session);
  } catch {
    updateConfirmMessageState(sessionID, payload.message.id, "pending");
  }
}

/** 首次打开时加载会话列表，并拉取当前会话消息。 */
async function ensureSessionsLoaded() {
  if (loadingSessions.value || sessions.value.length > 0) return;

  loadingSessions.value = true;
  try {
    const response = await defAiAssistantService.ListAiAssistantSessions({ terminal: "admin" });
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
  if (!sessionID || loadingMessages.value) return;

  loadingMessages.value = true;
  try {
    const response = await defAiAssistantService.ListAiAssistantMessages({ session_id: sessionID });
    messages.value[sessionID] = normalizeMessageList(response?.messages);
  } catch {
    messages.value[sessionID] = [];
  } finally {
    loadingMessages.value = false;
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
  upsertSession(session);
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

  const session = await defAiAssistantService.CreateAiAssistantSession({
    title: "新对话",
    scene: "workspace",
    terminal: "admin"
  });
  const normalizedSession = normalizeSession(session);
  upsertSession(normalizedSession);
  activeSessionID.value = normalizedSession.id;
  return activeSessionID.value;
}

/** 映射后端消息结构到页面气泡结构。 */
function mapMessageItem(message: AiAssistantMessage): ChatMessageItem {
  const normalizedContent = String(message.content ?? "");
  return {
    ...message,
    key: message.id,
    content: normalizedContent,
    placement: message.role === "user" ? "end" : "start",
    confirmTitle: message.confirm_title,
    confirmLines: message.confirm_lines,
    confirmFormFields: resolveConfirmFormFields(message),
    confirmFormValues: {},
    confirmState: resolveConfirmState(message),
    reply_source: String(message.reply_source ?? ""),
    model: String(message.model ?? ""),
    fallback: Boolean(message.fallback),
    fallback_reason: String(message.fallback_reason ?? ""),
    variant: message.role === "user" ? "filled" : message.kind === "tool" || message.kind === "confirm" ? "outlined" : "filled",
    shape: "round",
    maxWidth:
      message.kind === "confirm" ? "360px" : message.kind === "tool" ? "430px" : message.role === "user" ? "380px" : "440px"
  };
}

/** 根据后端消息状态映射确认卡展示状态。 */
function resolveConfirmState(message: AiAssistantMessage): ChatMessageItem["confirmState"] {
  if (message.kind !== "confirm") return undefined;
  switch (String(message.confirm_status ?? "")) {
    case "approved":
      return "confirmed";
    case "rejected":
      return "rejected";
    default:
      return "pending";
  }
}

/** 从工具输入里提取确认卡表单字段，兼容当前消息协议未单独暴露表单结构的阶段。 */
function resolveConfirmFormFields(message: AiAssistantMessage) {
  if (message.kind !== "confirm") return [];
  if (!message.confirm_action.includes("shipment")) return [];
  return [
    { prop: "name", label: "物流公司名称", placeholder: "请输入物流公司名称", required: true },
    { prop: "no", label: "物流单号", placeholder: "请输入物流单号", required: true },
    { prop: "contact", label: "联系方式", placeholder: "请输入联系方式", required: true }
  ];
}

/** 提交确认前校验必填表单，避免空值请求直达后端。 */
function validateConfirmForm(message: ChatMessageItem, formValues: Record<string, string>) {
  const requiredFields = message.confirmFormFields?.filter(field => field.required) ?? [];
  for (const field of requiredFields) {
    if (!String(formValues?.[field.prop] ?? "").trim()) {
      ElMessage.warning(`请填写${field.label}`);
      return false;
    }
  }
  return true;
}

/** 更新当前会话中指定确认消息的状态，保持交互反馈及时可见。 */
function updateConfirmMessageState(sessionID: string, messageID: string, confirmState: ChatMessageItem["confirmState"]) {
  const sessionMessages = messages.value[sessionID] ?? [];
  messages.value[sessionID] = sessionMessages.map(item =>
    item.id === messageID
      ? {
          ...item,
          confirmState
        }
      : item
  );
}

/** 合并后端返回的最新消息，优先用新消息覆盖同 ID 的旧状态。 */
function mergeSessionMessages(sessionID: string, nextMessages: ChatMessageItem[]) {
  if (!nextMessages.length) return;
  const mergedMap = new Map<string, ChatMessageItem>();
  (messages.value[sessionID] ?? []).forEach(item => {
    mergedMap.set(item.id, item);
  });
  nextMessages.forEach(item => {
    mergedMap.set(item.id, item);
  });
  messages.value[sessionID] = Array.from(mergedMap.values()).sort((left, right) => {
    const leftTime = resolveTimestamp(left.created_at);
    const rightTime = resolveTimestamp(right.created_at);
    if (leftTime === rightTime) return Number(left.id) - Number(right.id);
    return leftTime - rightTime;
  });
}

/** 兜底清洗会话数据，避免接口空值导致页面初始化报错。 */
function normalizeSession(session?: Partial<AiAssistantSession> | null): AiAssistantSession {
  return {
    id: String(session?.id ?? ""),
    title: String(session?.title ?? "新对话"),
    scene: String(session?.scene ?? "workspace"),
    summary: String(session?.summary ?? ""),
    tool_count: Number(session?.tool_count ?? 0),
    updated_at: session?.updated_at,
    terminal: String(session?.terminal ?? "admin")
  };
}

/** 将会话列表统一收敛为可安全渲染的数组。 */
function normalizeSessionList(list?: AiAssistantSession[] | null) {
  if (!Array.isArray(list)) return [];
  return list.map(item => normalizeSession(item)).filter(item => item.id);
}

/** 将消息列表统一收敛为可安全渲染的数组。 */
function normalizeMessageList(list?: AiAssistantMessage[] | null) {
  if (!Array.isArray(list)) return [];
  return list.filter(Boolean).map(item => mapMessageItem(item));
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

/** 将 protobuf 时间戳转为毫秒时间。 */
function resolveTimestamp(timestamp?: { seconds?: number; nanos?: number }) {
  if (!timestamp) return 0;
  const seconds = Number(timestamp.seconds ?? 0);
  const nanos = Number(timestamp.nanos ?? 0);
  return seconds * 1000 + Math.floor(nanos / 1_000_000);
}

/** 页面加载后主动准备首个会话，避免进入菜单后仍需额外点击。 */
onMounted(() => {
  void ensureSessionsLoaded();
});
</script>

<style scoped lang="scss">
.ai-assistant-page {
  display: grid;
  grid-template-columns: 320px minmax(0, 1fr);
  min-height: calc(100vh - 128px);
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
    min-height: calc(100vh - 126px);
  }

  .agent-session-collapsed {
    display: none;
  }
}
</style>
