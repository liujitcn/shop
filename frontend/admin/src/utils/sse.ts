import { SseEvent, SseRefreshReason, SseRefreshTarget, SseStream } from "@/rpc/common/v1/enum";
import { useUserStore } from "@/stores/modules/user";

/** SSE 取消订阅函数。 */
export type SseStop = () => void;

/** SSE 刷新事件负载。 */
export interface SseRefreshPayload {
  /** 事件名称。 */
  event: SseEvent;
  /** 需要刷新的页面目标。 */
  targets: SseRefreshTarget[];
  /** 触发刷新原因。 */
  reason?: SseRefreshReason;
  /** 事件发生时间。 */
  occurred_at: string;
}

/** SSE 刷新事件处理函数。 */
export type SseRefreshHandler = (payload: SseRefreshPayload) => void;

const DEFAULT_SSE_PATH = "/events";

/** 订阅 SSE 页面刷新事件。 */
export function subscribeSseRefresh(stream: SseStream, handler: SseRefreshHandler): SseStop {
  return subscribeSseEvent(stream, SseEvent.SSE_EVENT_PAGE_REFRESH, raw => parseSseRefreshPayload(raw), handler);
}

/** 订阅指定 SSE 事件。 */
export function subscribeSseEvent<T>(
  stream: SseStream,
  event: SseEvent,
  parser: (raw: string) => T | null,
  handler: (payload: T) => void
): SseStop {
  if (typeof window === "undefined" || typeof EventSource === "undefined") {
    return () => undefined;
  }

  const url = buildSseURL(stream);
  if (!url) {
    return () => undefined;
  }

  const source = new EventSource(url);
  const listener = (message: MessageEvent<string>) => {
    const payload = parser(message.data);
    if (!payload) return;
    handler(payload);
  };
  const eventName = toSseEventName(event);
  source.addEventListener(eventName, listener);

  return () => {
    source.removeEventListener(eventName, listener);
    source.close();
  };
}

/** 构建 SSE 订阅地址。 */
function buildSseURL(stream: SseStream) {
  const userStore = useUserStore();
  const token = stripBearerPrefix(userStore.token);
  if (!token) {
    return "";
  }

  const url = new URL(DEFAULT_SSE_PATH, window.location.origin);
  url.searchParams.set("stream", toSseStreamID(stream));
  url.searchParams.set("access_token", token);
  return url.toString();
}

/** 解析 SSE 刷新事件负载。 */
function parseSseRefreshPayload(raw: string): SseRefreshPayload | null {
  if (!raw) {
    return null;
  }

  try {
    const payload = JSON.parse(raw) as SseRefreshPayload;
    if (payload.event !== SseEvent.SSE_EVENT_PAGE_REFRESH || !Array.isArray(payload.targets)) {
      return null;
    }
    return payload;
  } catch {
    return null;
  }
}

/** 将 SSE 流枚举转换为传输层流标识。 */
function toSseStreamID(stream: SseStream) {
  return String(stream);
}

/** 将 SSE 事件枚举转换为 EventSource 事件名称。 */
function toSseEventName(event: SseEvent) {
  return String(event);
}

/** 去除令牌中的 Bearer 前缀，适配 EventSource 只能通过 URL 传参的限制。 */
function stripBearerPrefix(token: string) {
  const value = token.trim();
  return value.replace(/^Bearer\s+/i, "");
}
