import type { AiAssistantStreamEvent, AiAssistantStreamEventName, AiAssistantStreamPayload } from "./types";

const STREAM_EVENT_NAMES = new Set<AiAssistantStreamEventName>(["delta", "finish", "error"]);

/** useXStream 解析后的 SSE 字段结构。 */
type XStreamSseOutput = Partial<Record<"data" | "event" | "id" | "retry", unknown>>;

/** 判断 SSE 事件名称是否为 AI 助手 direct stream 支持的事件。 */
function isAiAssistantStreamEventName(event?: unknown): event is AiAssistantStreamEventName {
  return STREAM_EVENT_NAMES.has(String(event ?? "").trim() as AiAssistantStreamEventName);
}

/** 解析 useXStream 产出的 SSE data 字段，兼容前导空格和空消息。 */
function parseStreamPayload(data?: unknown): AiAssistantStreamPayload | null {
  const rawData = String(data ?? "").trimStart();
  if (!rawData) return null;

  try {
    return JSON.parse(rawData) as AiAssistantStreamPayload;
  } catch {
    return null;
  }
}

/** 将 useXStream 的原始 SSE 项收敛为业务事件，避免页面直接处理字符串 JSON。 */
export function normalizeAiAssistantStreamItem(item?: XStreamSseOutput): AiAssistantStreamEvent | null {
  if (!item || !isAiAssistantStreamEventName(item.event)) return null;

  const payload = parseStreamPayload(item.data);
  if (!payload?.session_id || !payload.client_message_id) return null;

  return {
    event: String(item.event).trim() as AiAssistantStreamEventName,
    payload
  };
}
