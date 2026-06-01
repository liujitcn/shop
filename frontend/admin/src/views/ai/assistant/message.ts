import { type AiAssistantAttachment, type AiAssistantMessage, type AiAssistantSession } from "@/rpc/base/v1/ai_assistant_session";
import { AiAssistantMessageStatus } from "@/rpc/common/v1/enum";
import { Terminal } from "@/rpc/common/v1/enum";
import type { AiAssistantStreamPayload, ChatMessageItem, ReplySourceTag } from "./types";

const THINKING_MESSAGE_ID_PREFIX = "assistant-thinking";
const LOCAL_USER_MESSAGE_ID_PREFIX = "assistant-user-local";
const PENDING_STREAM_MESSAGE_ID = "pending";
const THINKING_MESSAGE_CONTENT = "正在整理回复...";

/** 生成流式消息分组键，确保同一轮回复只更新当前占位气泡。 */
export function buildStreamMessageKey(sessionID: string, messageID: string) {
  return [sessionID, messageID].join(":");
}

/** 生成待绑定真实消息编号前的本地流式分组键。 */
export function buildPendingStreamMessageKey(sessionID: string) {
  return buildStreamMessageKey(sessionID, PENDING_STREAM_MESSAGE_ID);
}

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
    status: Number(message.status ?? AiAssistantMessageStatus.SUCCESS_AAMS),
    variant: message.role === "user" ? "filled" : "outlined",
    shape: "corner",
    progressState:
      message.status === AiAssistantMessageStatus.GENERATING_AAMS
        ? "streaming"
        : message.status === AiAssistantMessageStatus.FAILED_AAMS
          ? "failed"
          : "idle",
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
export function createLocalUserMessage(payload: {
  text: string;
  attachments: AiAssistantAttachment[];
}) {
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
    fallback_reason: "",
    status: AiAssistantMessageStatus.GENERATING_AAMS
  });
  // 本地回显消息只用于等待服务端响应，收到正式消息后需要被替换掉，避免同一问题展示两遍。
  message.localOnly = true;
  return message;
}

/** 创建助手思考中的占位消息。 */
export function createThinkingMessage(options?: { sessionID?: string; messageID?: string }) {
  const now = new Date();
  const streamKey = options?.sessionID
    ? buildStreamMessageKey(options.sessionID, options.messageID || PENDING_STREAM_MESSAGE_ID)
    : undefined;
  const message = mapMessageItem({
    id: streamKey || `${THINKING_MESSAGE_ID_PREFIX}-${now.getTime()}`,
    role: "assistant",
    kind: "text",
    content: THINKING_MESSAGE_CONTENT,
    attachments: [],
    created_at: {
      seconds: Math.floor(now.getTime() / 1000),
      nanos: (now.getTime() % 1000) * 1_000_000
    },
    reply_source: "",
    model: "",
    fallback: false,
    fallback_reason: "",
    status: AiAssistantMessageStatus.GENERATING_AAMS
  });
  message.progressState = "streaming";
  message.localOnly = true;
  message.streamKey = streamKey;
  message.replySourceTag = { text: "思考中", tone: "info" };
  return message;
}

/** 确保当前轮次存在流式占位消息。 */
export function ensureStreamingMessage(current: ChatMessageItem[], payload: AiAssistantStreamPayload) {
  const sessionID = String(payload.session_id ?? "");
  const messageID = String(payload.message_id ?? "");
  if (!sessionID || !messageID) return current;

  const streamKey = buildStreamMessageKey(sessionID, messageID);
  if (current.some(item => item.streamKey === streamKey)) return current;

  const pendingStreamKey = buildPendingStreamMessageKey(sessionID);
  const next = current.map<ChatMessageItem>(item => (item.streamKey === pendingStreamKey ? { ...item, streamKey } : item));
  if (next.some(item => item.streamKey === streamKey)) return next;

  return sortMessages([...next, createThinkingMessage({ sessionID, messageID })]);
}

/** 将已有助手消息清空并标记为重新生成中，保留消息 ID 与原位置。 */
export function markAssistantMessageRegenerating(current: ChatMessageItem[], sessionID: string, messageID: string) {
  const streamKey = buildStreamMessageKey(sessionID, messageID);
  return current.map<ChatMessageItem>(item => {
    if (String(item.id) !== messageID || item.role === "user") return item;
    return {
      ...item,
      content: THINKING_MESSAGE_CONTENT,
      fallback: false,
      fallback_reason: "",
      progressState: "streaming",
      replySourceTag: { text: "思考中", tone: "info" },
      status: AiAssistantMessageStatus.GENERATING_AAMS,
      streamKey
    };
  });
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

/** 去掉当前轮次的本地占位消息，并拼入服务端返回。 */
export function replacePendingMessages(
  current: ChatMessageItem[],
  nextMessages: ChatMessageItem[],
  payload?: AiAssistantStreamPayload
) {
  const sessionID = String(payload?.session_id ?? "");
  const streamKey = payload && buildStreamMessageKey(sessionID, String(payload.message_id ?? ""));
  const pendingStreamKey = payload?.message_id && sessionID ? buildPendingStreamMessageKey(sessionID) : "";
  const stableMessages = current.filter(item => {
    if (!item.localOnly) return true;
    if (payload?.message_id && item.role === "user") {
      return !nextMessages.some(message => message.role === "user");
    }
    if (!streamKey) return false;
    return item.streamKey !== streamKey && item.streamKey !== pendingStreamKey;
  });
  const messageMap = new Map<string, ChatMessageItem>();

  for (const item of stableMessages) {
    messageMap.set(String(item.id), item);
  }
  for (const item of nextMessages) {
    messageMap.set(String(item.id), item);
  }

  return sortMessages(Array.from(messageMap.values()));
}

/** 标记思考中消息失败，可按当前轮次限定失败范围。 */
export function markThinkingMessageFailed(
  current: ChatMessageItem[],
  options?: { sessionID?: string; messageID?: string }
) {
  const streamKey =
    options?.sessionID && options.messageID ? buildStreamMessageKey(options.sessionID, options.messageID) : "";
  const pendingStreamKey = options?.sessionID ? buildPendingStreamMessageKey(options.sessionID) : "";
  return current.map<ChatMessageItem>(item => {
    if (!item.localOnly || item.progressState !== "streaming") return item;
    if (streamKey && item.streamKey !== streamKey) return item;
    if (!streamKey && pendingStreamKey && item.streamKey !== pendingStreamKey) return item;
    return {
      ...item,
      progressState: "failed",
      content: "这次回复没有成功返回，你可以直接重试刚才的问题。",
      replySourceTag: { text: "发送失败", tone: "warning" }
    };
  });
}

/** 判断流式增量是否包含可追加内容。 */
export function hasStreamingDelta(payload: Pick<AiAssistantStreamPayload, "delta">) {
  return payload.delta !== undefined && payload.delta !== "";
}

/** 将流式文本增量追加到本地占位消息。 */
export function appendStreamingDelta(current: ChatMessageItem[], payload: AiAssistantStreamPayload) {
  if (!hasStreamingDelta(payload)) return current;
  const streamKey = buildStreamMessageKey(String(payload.session_id ?? ""), String(payload.message_id ?? ""));
  return current.map<ChatMessageItem>(item => {
    if (item.streamKey !== streamKey || (!item.localOnly && item.role === "user")) return item;
    const baseContent = item.content === THINKING_MESSAGE_CONTENT ? "" : item.content;
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
export function markStreamingError(current: ChatMessageItem[], payload: AiAssistantStreamPayload) {
  const streamKey = buildStreamMessageKey(String(payload.session_id ?? ""), String(payload.message_id ?? ""));
  return current.map<ChatMessageItem>(item => {
    if (item.streamKey !== streamKey || !item.localOnly) return item;
    return {
      ...item,
      progressState: "failed",
      content: "这次回复没有成功返回，你可以直接重试刚才的问题。",
      replySourceTag: { text: "发送失败", tone: "warning" }
    };
  });
}

/** 按消息位置查找最近一条用户消息，用于重新生成助手回复。 */
export function findPreviousUserMessage(current: ChatMessageItem[], target: ChatMessageItem) {
  const sortedList = sortMessages(current);
  const targetIndex = sortedList.findIndex(item => String(item.id) === String(target.id));
  const endIndex = targetIndex >= 0 ? targetIndex - 1 : sortedList.length - 1;
  for (let index = endIndex; index >= 0; index--) {
    const item = sortedList[index];
    if (item.role === "user") return item;
  }
  return undefined;
}

/** 标记正在朗读的消息，保证全局只有一个气泡处于朗读态。 */
export function markSpeakingMessage(current: ChatMessageItem[], messageID?: string) {
  return current.map<ChatMessageItem>(item => ({
    ...item,
    speaking: Boolean(messageID && String(item.id) === messageID)
  }));
}
