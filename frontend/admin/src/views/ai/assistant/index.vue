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
    <ChatPanel
      :active-session="activeSession"
      :messages="currentMessages"
      :sending="sending"
      @submit="handleSubmit"
      @message-action="handleMessageAction"
    />
  </div>
</template>

<script setup lang="ts">
import "vue-element-plus-x/styles/index.css";
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { useXStream } from "vue-element-plus-x";
import { DArrowRight } from "@element-plus/icons-vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { defAiAssistantMessageService } from "@/api/base/ai_assistant_message";
import { defAiAssistantSessionService } from "@/api/base/ai_assistant_session";
import { type AiAssistantSession } from "@/rpc/base/v1/ai_assistant_session";
import { AiAssistantMessageStatus, Terminal } from "@/rpc/common/v1/enum";
import ChatPanel from "./components/ChatPanel.vue";
import SessionPanel from "./components/SessionPanel.vue";
import {
  appendStreamingDelta,
  createLocalUserMessage,
  createThinkingMessage,
  ensureStreamingMessage,
  hasStreamingDelta,
  markAssistantMessageRegenerating,
  markStreamingError,
  markThinkingMessageFailed,
  markSpeakingMessage,
  normalizeMessageList,
  normalizeSession,
  normalizeSessionList,
  resolveTimestamp,
  replacePendingMessages,
  sortMessages
} from "./message";
import { normalizeAiAssistantStreamItem } from "./stream";
import type { AiAssistantStreamEvent, AiAssistantStreamPayload, ChatMessageAction, ChatMessageItem, SessionAction, SubmitPayload } from "./types";

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
const pendingDeltaMap = new Map<string, AiAssistantStreamPayload>();
const { startStream, cancel: cancelStream, data: streamData, error: streamError } = useXStream();
let pendingDeltaFrame = 0;
let consumedStreamItemCount = 0;
let streamFinished = false;
let speakingMessageID = "";

const filteredSessions = computed(() => {
  const keyword = sessionKeyword.value.trim();
  if (!keyword) return sessions.value;
  return sessions.value.filter(item => item.title.includes(keyword) || item.summary.includes(keyword));
});

const activeSession = computed(() => sessions.value.find(item => item.id === activeSessionID.value) ?? sessions.value[0]);

const currentMessages = computed(() => messages.value[activeSessionID.value] ?? []);

/** 切换会话时同步当前活动会话。 */
function handleSessionChange(item: AiAssistantSession) {
  activeSessionID.value = item.id;
  void loadMessages(item.id);
}

/** 切换会话侧栏折叠状态。 */
function toggleSessionPanel() {
  sessionPanelCollapsed.value = !sessionPanelCollapsed.value;
}

/** 处理会话项菜单动作，接入真实重命名与删除接口。 */
async function handleSessionAction(payload: { action: SessionAction; item: AiAssistantSession }) {
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
  try {
    await sendAiAssistantPayload(payload);
  } finally {
    sending.value = false;
  }
}

/** 执行真实发送流程，复用于输入框提交和本地失败重发。 */
async function sendAiAssistantPayload(payload: SubmitPayload) {
  const sessionID = await ensureActiveSession();
  if (!sessionID) {
    return false;
  }

  const localUserMessage = createLocalUserMessage(payload);
  const thinkingMessage = createThinkingMessage({ sessionID });
  messages.value[sessionID] = sortMessages([...(messages.value[sessionID] ?? []), localUserMessage, thinkingMessage]);
  try {
    consumedStreamItemCount = 0;
    streamFinished = false;
    // useXStream 每次 startStream 会重置内部 data；这里提前对齐游标，避免旧数据长度影响本轮消费。
    streamData.value = [];
    const response = await defAiAssistantMessageService.StreamAiAssistantMessage({
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

    if (!response.body) {
      throw new Error("AI 助手流式响应为空");
    }
    await startStream({ readableStream: response.body });
    consumeStreamItems();
    if (streamError.value) {
      throw streamError.value;
    }
    if (!streamFinished) {
      throw new Error("AI 助手流式响应未完整返回");
    }
    return true;
  } catch (error) {
    messages.value[sessionID] = markThinkingMessageFailed(messages.value[sessionID] ?? [], {
      sessionID
    });
    const message = error instanceof Error ? error.message : "AI 助手请求失败";
    ElMessage.error(message);
    return false;
  }
}

/** 处理聊天气泡上的重试、分支、朗读、复制和删除操作。 */
async function handleMessageAction(payload: { action: ChatMessageAction; item: ChatMessageItem }) {
  if (payload.action === "copy") {
    await handleCopyMessage(payload.item);
    return;
  }
  if (payload.action === "delete") {
    await handleDeleteMessage(payload.item);
    return;
  }
  if (payload.action === "branch") {
    await handleBranchMessage(payload.item);
    return;
  }
  if (payload.action === "speak") {
    handleSpeakMessage(payload.item);
    return;
  }
  await handleRetryMessage(payload.item);
}

/** 复制当前消息正文，保留用户原始输入或助手 Markdown 文本。 */
async function handleCopyMessage(item: ChatMessageItem) {
  try {
    await navigator.clipboard.writeText(String(item.content ?? ""));
    ElMessage.success("消息已复制");
  } catch {
    ElMessage.error("当前浏览器不支持复制");
  }
}

/** 删除当前会话中的消息，并同步后端持久化状态。 */
async function handleDeleteMessage(item: ChatMessageItem) {
  const sessionID = activeSessionID.value;
  if (!sessionID) return;
  if (speakingMessageID === resolveMessageBubbleKey(item)) {
    stopSpeaking();
  }
  if (item.localOnly) {
    const messageKey = resolveMessageBubbleKey(item);
    messages.value[sessionID] = (messages.value[sessionID] ?? []).filter(message => {
      if (item.role === "user") return !message.localOnly;
      return resolveMessageBubbleKey(message) !== messageKey;
    });
    ElMessage.success("消息已删除");
    return;
  }
  await defAiAssistantMessageService.DeleteAiAssistantMessage({
    session_id: sessionID,
    message_id: String(item.id)
  });
  messages.value[sessionID] = (messages.value[sessionID] ?? []).filter(message => String(message.id) !== String(item.id));
  ElMessage.success("消息已删除");
}

/** 重试失败的一轮消息，或重新生成助手输出。 */
async function handleRetryMessage(item: ChatMessageItem) {
  const sessionID = activeSessionID.value;
  if (!sessionID || sending.value) return;

  sending.value = true;
  try {
    let response;
    if (item.localOnly) {
      const payload = resolveLocalRetryPayload(item);
      if (!payload) {
        ElMessage.warning("未找到可重新发送的本地消息");
        return;
      }
      messages.value[sessionID] = (messages.value[sessionID] ?? []).filter(message => !message.localOnly);
      if (await sendAiAssistantPayload(payload)) {
        ElMessage.success("已重新发送");
      }
      return;
    }
    if (item.role === "user") {
      if (item.status !== AiAssistantMessageStatus.FAILED_AAMS) {
        ElMessage.warning("只有发送失败的消息可以重新发送");
        return;
      }
      response = await defAiAssistantMessageService.RetryAiAssistantUserMessage({
        session_id: sessionID,
        message_id: String(item.id)
      });
    } else {
      messages.value[sessionID] = markAssistantMessageRegenerating(messages.value[sessionID] ?? [], sessionID, String(item.id));
      response = await defAiAssistantMessageService.RegenerateAiAssistantMessage({
        session_id: sessionID,
        message_id: String(item.id)
      });
    }
    const current = messages.value[sessionID] ?? [];
    messages.value[sessionID] = replacePendingMessages(current, normalizeMessageList(response.messages));
    if (response.session) upsertSession(normalizeSession(response.session));
    ElMessage.success(item.role === "user" ? "已重新发送" : "已重新生成");
  } catch (error) {
    if (item.role !== "user") await loadMessages(sessionID);
    const message = error instanceof Error ? error.message : "重新生成失败";
    ElMessage.error(message);
  } finally {
    sending.value = false;
  }
}

/** 从本地失败气泡还原可重新提交的输入内容。 */
function resolveLocalRetryPayload(item: ChatMessageItem): SubmitPayload | undefined {
  if (item.role === "user") {
    return {
      text: String(item.content ?? ""),
      attachments: item.attachments ?? []
    };
  }

  const sortedList = sortMessages(messages.value[activeSessionID.value] ?? []);
  const targetIndex = sortedList.findIndex(message => resolveMessageBubbleKey(message) === resolveMessageBubbleKey(item));
  const endIndex = targetIndex >= 0 ? targetIndex - 1 : sortedList.length - 1;
  for (let index = endIndex; index >= 0; index--) {
    const message = sortedList[index];
    if (message.localOnly && message.role === "user") {
      return {
        text: String(message.content ?? ""),
        attachments: message.attachments ?? []
      };
    }
  }
  return undefined;
}

/** 从当前消息处创建一个新的持久化分支会话。 */
async function handleBranchMessage(item: ChatMessageItem) {
  const sourceSessionID = activeSessionID.value;
  if (!sourceSessionID) return;
  stopSpeaking();
  const response = await defAiAssistantSessionService.CreateAiAssistantSessionBranch({
    source_session_id: sourceSessionID,
    anchor_message_id: String(item.id),
    title: buildBranchSessionTitle(item),
    terminal: Terminal.TERMINAL_ADMIN
  });
  const branchSession = normalizeSession(response.session);
  upsertSession(branchSession);
  messages.value[branchSession.id] = normalizeMessageList(response.messages);
  activeSessionID.value = branchSession.id;
  ElMessage.success("已创建分支会话");
}

/** 朗读或停止朗读当前助手输出。 */
function handleSpeakMessage(item: ChatMessageItem) {
  if (item.role === "user") return;
  if (!window.speechSynthesis) {
    ElMessage.warning("当前浏览器不支持朗读");
    return;
  }
  const messageKey = resolveMessageBubbleKey(item);
  if (speakingMessageID === messageKey) {
    stopSpeaking();
    return;
  }
  stopSpeaking();
  const utterance = new SpeechSynthesisUtterance(String(item.content ?? ""));
  utterance.lang = "zh-CN";
  utterance.onend = () => clearSpeakingState(messageKey);
  utterance.onerror = () => clearSpeakingState(messageKey);
  speakingMessageID = messageKey;
  markAllMessagesSpeaking(speakingMessageID);
  window.speechSynthesis.speak(utterance);
}

/** 生成当前气泡的前端稳定键，用于朗读态和渲染态关联。 */
function resolveMessageBubbleKey(item: ChatMessageItem) {
  return String(item.key ?? `${item.id}:${item.role}`);
}

/** 停止浏览器朗读，并清理气泡朗读态。 */
function stopSpeaking() {
  if (window.speechSynthesis) {
    window.speechSynthesis.cancel();
  }
  speakingMessageID = "";
  markAllMessagesSpeaking();
}

/** 朗读事件结束后，只清理当前朗读消息对应的状态。 */
function clearSpeakingState(messageID: string) {
  if (speakingMessageID && speakingMessageID !== messageID) return;
  speakingMessageID = "";
  markAllMessagesSpeaking();
}

/** 同步所有已加载会话中的朗读态，避免切换会话后状态残留。 */
function markAllMessagesSpeaking(messageID?: string) {
  Object.keys(messages.value).forEach(sessionID => {
    messages.value[sessionID] = markSpeakingMessage(messages.value[sessionID] ?? [], messageID);
  });
}

/** 处理 AI 助手流式文本增量。 */
function handleAiAssistantDelta(payload: AiAssistantStreamPayload) {
  if (!hasStreamingDelta(payload)) return;
  queueAiAssistantDelta(payload);
}

/** 处理 AI 助手流式结束事件。 */
function handleAiAssistantFinish(payload: AiAssistantStreamPayload) {
  const sessionID = String(payload.session_id ?? "");
  if (!sessionID) return;
  streamFinished = true;
  flushAiAssistantDelta();
  const nextMessages = normalizeMessageList(payload.messages as never[]);
  const current = messages.value[sessionID] ?? [];
  const messageID = String(payload.message_id ?? "");
  const streamKey = messageID ? `${sessionID}:${messageID}` : "";
  const hasLocalStreamingMessages = current.some(item => item.streamKey === streamKey && item.localOnly);
  messages.value[sessionID] =
    nextMessages.length || !hasLocalStreamingMessages ? replacePendingMessages(current, nextMessages, payload) : current;
  if (payload.session) {
    upsertSession(normalizeSession(payload.session as AiAssistantSession));
  }
}

/** 处理 AI 助手流式异常事件。 */
function handleAiAssistantError(payload: AiAssistantStreamPayload) {
  const sessionID = String(payload.session_id ?? "");
  if (!sessionID || !messages.value[sessionID]) return;
  streamFinished = true;
  flushAiAssistantDelta();
  const nextMessages = normalizeMessageList(payload.messages as never[]);
  if (nextMessages.length) {
    messages.value[sessionID] = replacePendingMessages(messages.value[sessionID] ?? [], nextMessages, payload);
    return;
  }
  const streamingMessages = ensureStreamingMessage(messages.value[sessionID] ?? [], payload);
  messages.value[sessionID] = markStreamingError(streamingMessages, payload);
}

/** 根据 useXStream 解析结果派发 AI 助手 direct stream 事件。 */
function handleAiAssistantStreamEvent(event: AiAssistantStreamEvent) {
  switch (event.event) {
    case "delta":
      handleAiAssistantDelta(event.payload);
      break;
    case "finish":
      handleAiAssistantFinish(event.payload);
      break;
    case "error":
      handleAiAssistantError(event.payload);
      break;
  }
}

/** 消费 useXStream 追加的数据项，避免同一个 SSE 片段被重复处理。 */
function consumeStreamItems() {
  const items = streamData.value.slice(consumedStreamItemCount);
  consumedStreamItemCount = streamData.value.length;
  for (const item of items) {
    const event = normalizeAiAssistantStreamItem(item);
    if (!event) continue;
    handleAiAssistantStreamEvent(event);
  }
}

/** 合并同一帧内的流式分片，减少频繁重排导致的卡顿。 */
function queueAiAssistantDelta(payload: AiAssistantStreamPayload) {
  const sessionID = String(payload.session_id ?? "");
  const messageID = String(payload.message_id ?? "");
  if (!sessionID || !messageID || !messages.value[sessionID]) return;

  const key = `${sessionID}:${messageID}`;
  const cachedPayload = pendingDeltaMap.get(key);
  pendingDeltaMap.set(key, {
    ...payload,
    delta: `${cachedPayload?.delta ?? ""}${payload.delta ?? ""}`
  });

  if (pendingDeltaFrame) return;
  pendingDeltaFrame = window.requestAnimationFrame(() => {
    pendingDeltaFrame = 0;
    flushAiAssistantDelta();
  });
}

/** 立即刷新已缓存的流式分片。 */
function flushAiAssistantDelta() {
  if (pendingDeltaFrame) {
    window.cancelAnimationFrame(pendingDeltaFrame);
    pendingDeltaFrame = 0;
  }
  if (pendingDeltaMap.size === 0) return;

  const payloadList = Array.from(pendingDeltaMap.values());
  pendingDeltaMap.clear();
  for (const payload of payloadList) {
    const sessionID = String(payload.session_id ?? "");
    if (!sessionID || !messages.value[sessionID]) continue;
    const streamingMessages = ensureStreamingMessage(messages.value[sessionID] ?? [], payload);
    messages.value[sessionID] = appendStreamingDelta(streamingMessages, payload);
  }
}

/** 首次打开时加载会话列表，并拉取当前会话消息。 */
async function ensureSessionsLoaded() {
  if (loadingSessions.value || sessions.value.length > 0) return;

  loadingSessions.value = true;
  try {
    const response = await defAiAssistantSessionService.ListAiAssistantSessions({ terminal: Terminal.TERMINAL_ADMIN });
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
    const response = await defAiAssistantMessageService.ListAiAssistantMessages({ session_id: sessionID });
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

  const response = await defAiAssistantSessionService.UpdateAiAssistantSession({ id: item.id, title: value.trim() });
  upsertSession(normalizeSession(response.session));
  ElMessage.success("会话已重命名");
}

/** 删除当前会话，并自动切换到剩余会话。 */
async function handleDeleteSession(item: AiAssistantSession) {
  await ElMessageBox.confirm(`是否删除该会话？\n会话名称：${item.title}`, "删除会话", {
    confirmButtonText: "确认删除",
    cancelButtonText: "取消",
    type: "warning"
  });

  await defAiAssistantSessionService.DeleteAiAssistantSession({ id: item.id });
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
async function createSession(options?: { title?: string }) {
  const response = await defAiAssistantSessionService.CreateAiAssistantSession({
    title: options?.title || "新对话",
    terminal: Terminal.TERMINAL_ADMIN
  });
  const normalizedSession = normalizeSession(response.session);
  upsertSession(normalizedSession);
  return normalizedSession.id;
}

/** 使用当前消息内容生成一个易识别的分支会话标题。 */
function buildBranchSessionTitle(item: ChatMessageItem) {
  const content = String(item.input_content?.content || item.content || "新对话").replace(/\s+/g, " ").trim();
  return `分支：${content.slice(0, 18) || "新对话"}`;
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
  void ensureSessionsLoaded();
});

/** 监听 useXStream 数据长度变化，逐条处理 direct stream SSE 事件。 */
watch(
  () => streamData.value.length,
  () => {
    consumeStreamItems();
  },
  { flush: "sync" }
);

onBeforeUnmount(() => {
  stopSpeaking();
  cancelStream();
  if (pendingDeltaFrame) {
    window.cancelAnimationFrame(pendingDeltaFrame);
    pendingDeltaFrame = 0;
  }
  pendingDeltaMap.clear();
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
  flex-direction: column;
  gap: 12px;
  align-items: center;
  min-height: 0;
  padding: 16px 8px;
  background: var(--admin-page-card-bg);
  border-right: 1px solid var(--admin-page-divider-strong);
}
.agent-session-collapsed__toggle {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  color: var(--admin-page-text-secondary);
  cursor: pointer;
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
  user-select: none;
  writing-mode: vertical-rl;
}

@media screen and (width <= 1200px) {
  .ai-assistant-page {
    grid-template-columns: 264px minmax(0, 1fr);
  }
  .ai-assistant-page.is-session-collapsed {
    grid-template-columns: 44px minmax(0, 1fr);
  }
}

@media screen and (width <= 768px) {
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
