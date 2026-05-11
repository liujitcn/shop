import type { AiAssistantAttachment, AiAssistantMessage, AiAssistantSession } from "@/rpc/base/v1/ai_assistant";
import type { SseAiAssistantPayload } from "@/api/base/sse";
import { Terminal } from "@/rpc/common/v1/enum";
import type { ChatMessageItem, ReplySourceTag } from "./types";

const THINKING_MESSAGE_ID_PREFIX = "assistant-thinking";
const LOCAL_USER_MESSAGE_ID_PREFIX = "assistant-user-local";

/** 计算消息角色排序权重，确保同一轮对话里用户问题先于助手回答。 */
function resolveRoleOrder(role?: string) {
  return role === "user" ? 0 : 1;
}

/** 解析 protobuf 时间戳为毫秒。 */
export function resolveTimestamp(timestamp?: { seconds?: number; nanos?: number }) {
  if (!timestamp) return 0;
  const seconds = Number(timestamp.seconds ?? 0);
  const nanos = Number(timestamp.nanos ?? 0);
  return seconds * 1000 + Math.floor(nanos / 1_000_000);
}

/** 归一化会话对象，避免空值影响渲染。 */
export function normalizeSession(session?: Partial<AiAssistantSession> | null): AiAssistantSession {
  return {
    id: String(session?.id ?? ""),
    title: String(session?.title ?? "新对话"),
    scene: String(session?.scene ?? "workspace"),
    summary: String(session?.summary ?? ""),
    tool_count: Number(session?.tool_count ?? 0),
    updated_at: session?.updated_at,
    terminal: Number(session?.terminal ?? Terminal.TERMINAL_ADMIN)
  };
}

/** 将会话列表收敛成可安全渲染的数组。 */
export function normalizeSessionList(list?: AiAssistantSession[] | null) {
  if (!Array.isArray(list)) return [];
  return list.map(item => normalizeSession(item)).filter(item => item.id);
}

/** 生成回复来源标签。 */
export function resolveReplySourceTag(message: AiAssistantMessage): ReplySourceTag | undefined {
  if (message.role === "user") return undefined;
  if (message.fallback) return { text: "降级回复", tone: "warning" };
  switch (String(message.reply_source ?? "")) {
    case "network":
      return { text: "网络数据", tone: "success" };
    case "llm":
      return { text: "模型回答", tone: "primary" };
    case "fallback":
      return { text: "降级回复", tone: "warning" };
    default:
      return message.model ? { text: "模型回答", tone: "primary" } : undefined;
  }
}

/** 将后端消息映射到聊天气泡结构。 */
export function mapMessageItem(message: AiAssistantMessage): ChatMessageItem {
  return {
    ...message,
    key: String(message.id),
    content: String(message.content ?? ""),
    placement: message.role === "user" ? "end" : "start",
    reply_source: String(message.reply_source ?? ""),
    model: String(message.model ?? ""),
    fallback: Boolean(message.fallback),
    fallback_reason: String(message.fallback_reason ?? ""),
    variant: message.role === "user" ? "filled" : "outlined",
    shape: "round",
    progressState: "idle",
    replySourceTag: resolveReplySourceTag(message),
    maxWidth: message.role === "user" ? "380px" : "460px"
  };
}

/** 收敛消息数组。 */
export function normalizeMessageList(list?: AiAssistantMessage[] | null) {
  if (!Array.isArray(list)) return [];
  return list.filter(Boolean).map(item => mapMessageItem(item));
}

/** 创建本地用户回显消息。 */
export function createLocalUserMessage(payload: { text: string; attachments: AiAssistantAttachment[] }) {
  const now = new Date();
  const message = mapMessageItem({
    id: `${LOCAL_USER_MESSAGE_ID_PREFIX}-${now.getTime()}`,
    role: "user",
    kind: "text",
    content: payload.text,
    attachments: payload.attachments,
    created_at: {
      seconds: Math.floor(now.getTime() / 1000),
      nanos: (now.getTime() % 1000) * 1_000_000
    },
    reply_source: "",
    model: "",
    fallback: false,
    fallback_reason: ""
  });
  // 本地回显消息只用于等待服务端响应，收到正式消息后需要被替换掉，避免同一问题展示两遍。
  message.localOnly = true;
  return message;
}

/** 创建助手思考中的占位消息。 */
export function createThinkingMessage(clientMessageId?: string) {
  const now = new Date();
  const message = mapMessageItem({
    id: clientMessageId || `${THINKING_MESSAGE_ID_PREFIX}-${now.getTime()}`,
    role: "assistant",
    kind: "text",
    content: "正在整理回复...",
    attachments: [],
    created_at: {
      seconds: Math.floor(now.getTime() / 1000),
      nanos: (now.getTime() % 1000) * 1_000_000
    },
    reply_source: "",
    model: "",
    fallback: false,
    fallback_reason: ""
  });
  message.progressState = "streaming";
  message.localOnly = true;
  message.replySourceTag = { text: "思考中", tone: "info" };
  return message;
}

/** 将消息按创建时间排序。 */
export function sortMessages(list: ChatMessageItem[]) {
  return [...list].sort((left, right) => {
    const leftTime = resolveTimestamp(left.created_at);
    const rightTime = resolveTimestamp(right.created_at);
    if (leftTime === rightTime) {
      const roleOrder = resolveRoleOrder(left.role) - resolveRoleOrder(right.role);
      if (roleOrder !== 0) return roleOrder;
      return String(left.id).localeCompare(String(right.id), "zh-Hans-CN", { numeric: true });
    }
    return leftTime - rightTime;
  });
}

/** 去掉本地占位消息，并拼入服务端返回。 */
export function replacePendingMessages(current: ChatMessageItem[], nextMessages: ChatMessageItem[]) {
  const stableMessages = current.filter(item => !item.localOnly);
  const messageMap = new Map<string, ChatMessageItem>();

  for (const item of stableMessages) {
    messageMap.set(String(item.id), item);
  }
  for (const item of nextMessages) {
    messageMap.set(String(item.id), item);
  }

  return sortMessages(Array.from(messageMap.values()));
}

/** 标记思考中消息失败。 */
export function markThinkingMessageFailed(current: ChatMessageItem[]) {
  return current.map<ChatMessageItem>(item => {
    if (!item.localOnly || item.progressState !== "streaming") return item;
    return {
      ...item,
      progressState: "failed",
      content: "这次回复没有成功返回，你可以直接重试刚才的问题。",
      replySourceTag: { text: "发送失败", tone: "warning" }
    };
  });
}

/** 将流式文本增量追加到本地占位消息。 */
export function appendStreamingDelta(current: ChatMessageItem[], payload: SseAiAssistantPayload) {
  return current.map<ChatMessageItem>(item => {
    if (item.id !== payload.client_message_id || !item.localOnly) return item;
    const baseContent = item.content === "正在整理回复..." ? "" : item.content;
    const nextContent = `${baseContent}${payload.delta ?? ""}`;
    return {
      ...item,
      content: nextContent || item.content,
      progressState: "streaming",
      replySourceTag: { text: "回答中", tone: "info" }
    };
  });
}

/** 根据流式异常事件更新本地占位消息。 */
export function markStreamingError(current: ChatMessageItem[], payload: SseAiAssistantPayload) {
  return current.map<ChatMessageItem>(item => {
    if (item.id !== payload.client_message_id || !item.localOnly) return item;
    return {
      ...item,
      progressState: "failed",
      content: payload.error_message || "这次回复没有成功返回，你可以直接重试刚才的问题。",
      replySourceTag: { text: "发送失败", tone: "warning" }
    };
  });
}
