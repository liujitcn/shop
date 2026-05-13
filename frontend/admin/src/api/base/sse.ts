import { EventStreamContentType, fetchEventSource, type EventSourceMessage } from "@microsoft/fetch-event-source";
import type { SubscribeSseRequest } from "@/rpc/base/v1/sse";
import { SseEvent, SseRefreshReason, SseRefreshTarget, SseStream } from "@/rpc/common/v1/enum";
import pinia from "@/stores";
import { useUserStore } from "@/stores/modules/user";

const SSE_URL = "/events";

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

/** AI 助手流式事件负载。 */
export interface SseAiAssistantPayload {
  /** 事件名称。 */
  event: SseEvent;
  /** 会话 ID。 */
  session_id: string;
  /** 前端本地消息 ID，用于关联流式占位消息。 */
  client_message_id: string;
  /** 本次新增的文本分片。 */
  delta?: string;
  /** 流式完成后的最终消息列表。 */
  messages?: unknown[];
  /** 流式完成后的最新会话。 */
  session?: unknown;
  /** 异常提示。 */
  error_message?: string;
  /** 事件发生时间。 */
  occurred_at: string;
}

/** SSE 刷新事件处理函数。 */
export type SseRefreshHandler = (payload: SseRefreshPayload) => void;

/** 同一条 SSE 流的共享连接记录。 */
interface SharedSseConnection {
  /** 关闭当前 SSE 连接。 */
  close: () => void;
  /** 已注册的事件监听器。 */
  listeners: Map<string, Set<(message: EventSourceMessage) => void>>;
  /** 当前流上已绑定的事件监听器数量。 */
  refCount: number;
}

/** SSE 不可恢复异常。 */
class SseFatalError extends Error {}

/** SSE 可重试异常。 */
class SseRetriableError extends Error {}

/** Base SSE 服务。 */
export class SseServiceImpl {
  private readonly sharedConnections = new Map<string, SharedSseConnection>();

  /** 创建或复用 SSE 订阅连接。 */
  SubscribeSse(request: SubscribeSseRequest): SharedSseConnection | null {
    if (typeof window === "undefined" || typeof AbortController === "undefined") {
      return null;
    }

    const url = this.buildSubscribeURL(request);
    if (!url) {
      return null;
    }

    const connectionKey = this.buildConnectionKey(request);
    const cachedConnection = this.sharedConnections.get(connectionKey);
    if (cachedConnection) {
      return cachedConnection;
    }

    const controller = new AbortController();
    const listeners = new Map<string, Set<(message: EventSourceMessage) => void>>();
    const connection: SharedSseConnection = {
      close: () => controller.abort(),
      listeners,
      refCount: 0
    };
    this.sharedConnections.set(connectionKey, connection);
    void this.openFetchEventSource(connectionKey, url, controller, connection);
    return connection;
  }

  /** 释放 SSE 共享连接。 */
  ReleaseSse(request: SubscribeSseRequest) {
    const connectionKey = this.buildConnectionKey(request);
    const cachedConnection = this.sharedConnections.get(connectionKey);
    if (!cachedConnection) {
      return;
    }
    cachedConnection.refCount -= 1;
    if (cachedConnection.refCount > 0) {
      return;
    }
    cachedConnection.close();
    this.sharedConnections.delete(connectionKey);
  }

  /** 构建 SSE 订阅地址。 */
  private buildSubscribeURL(request: SubscribeSseRequest) {
    const url = new URL(`${SSE_URL}/${request.stream}`, window.location.origin);
    return url.toString();
  }

  /** 构建 SSE 共享连接键。 */
  private buildConnectionKey(request: SubscribeSseRequest) {
    return String(request.stream);
  }

  /** 打开基于 fetch 的 SSE 长连接。 */
  private async openFetchEventSource(
    connectionKey: string,
    url: string,
    controller: AbortController,
    connection: SharedSseConnection
  ) {
    const accessToken = this.getAccessToken();
    if (!accessToken) {
      this.sharedConnections.delete(connectionKey);
      return;
    }

    try {
      await fetchEventSource(url, {
        method: "GET",
        signal: controller.signal,
        openWhenHidden: true,
        headers: {
          Accept: EventStreamContentType,
          Authorization: accessToken
        },
        async onopen(response) {
          const contentType = response.headers.get("content-type") ?? "";
          if (response.ok && contentType.startsWith(EventStreamContentType)) {
            return;
          }
          if (response.status === 401 || response.status === 403) {
            throw new SseFatalError("SSE 认证已失效");
          }
          throw new SseRetriableError(`SSE 连接失败: ${response.status}`);
        },
        onmessage: message => {
          const eventName = message.event || "";
          if (!eventName) {
            return;
          }
          const eventListeners = connection.listeners.get(eventName);
          if (!eventListeners || eventListeners.size === 0) {
            return;
          }
          eventListeners.forEach(listener => listener(message));
        },
        onclose: () => {
          throw new SseRetriableError("SSE 连接已关闭");
        },
        onerror: error => {
          if (controller.signal.aborted) {
            return;
          }
          if (error instanceof SseFatalError) {
            throw error;
          }
          return 1000;
        }
      });
    } catch {
      if (controller.signal.aborted) {
        return;
      }
      this.sharedConnections.delete(connectionKey);
    }
  }

  /** 读取请求头使用的访问令牌。 */
  private getAccessToken() {
    const userStore = useUserStore(pinia);
    return userStore.token.trim();
  }
}

export const defSseService = new SseServiceImpl();

/** 订阅 SSE 页面刷新事件。 */
export function subscribeSseRefresh(stream: SseStream, handler: SseRefreshHandler): SseStop {
  return subscribeSseEvent(stream, SseEvent.SSE_EVENT_PAGE_REFRESH, raw => parseSseRefreshPayload(raw), handler);
}

/** 订阅 AI 助手流式文本增量事件。 */
export function subscribeAiAssistantDelta(handler: (payload: SseAiAssistantPayload) => void): SseStop {
  return subscribeSseEvent(
    SseStream.SSE_STREAM_ADMIN_AI_ASSISTANT,
    SseEvent.SSE_EVENT_AI_ASSISTANT_DELTA,
    raw => parseAiAssistantPayload(raw, SseEvent.SSE_EVENT_AI_ASSISTANT_DELTA),
    handler
  );
}

/** 订阅 AI 助手流式完成事件。 */
export function subscribeAiAssistantFinish(handler: (payload: SseAiAssistantPayload) => void): SseStop {
  return subscribeSseEvent(
    SseStream.SSE_STREAM_ADMIN_AI_ASSISTANT,
    SseEvent.SSE_EVENT_AI_ASSISTANT_FINISH,
    raw => parseAiAssistantPayload(raw, SseEvent.SSE_EVENT_AI_ASSISTANT_FINISH),
    handler
  );
}

/** 订阅 AI 助手流式异常事件。 */
export function subscribeAiAssistantError(handler: (payload: SseAiAssistantPayload) => void): SseStop {
  return subscribeSseEvent(
    SseStream.SSE_STREAM_ADMIN_AI_ASSISTANT,
    SseEvent.SSE_EVENT_AI_ASSISTANT_ERROR,
    raw => parseAiAssistantPayload(raw, SseEvent.SSE_EVENT_AI_ASSISTANT_ERROR),
    handler
  );
}

/** 订阅指定 SSE 事件。 */
export function subscribeSseEvent<T>(
  stream: SseStream,
  event: SseEvent,
  parser: (raw: string) => T | null,
  handler: (payload: T) => void
): SseStop {
  const request = { stream };
  const connection = defSseService.SubscribeSse(request);
  if (!connection) return () => undefined;

  connection.refCount += 1;
  const eventName = toSseEventName(event);
  let eventListeners = connection.listeners.get(eventName);
  if (!eventListeners) {
    eventListeners = new Set();
    connection.listeners.set(eventName, eventListeners);
  }

  const listener = (message: EventSourceMessage) => {
    const payload = parser(message.data);
    if (!payload) return;
    handler(payload);
  };
  eventListeners.add(listener);

  return () => {
    const currentListeners = connection.listeners.get(eventName);
    currentListeners?.delete(listener);
    if (currentListeners && currentListeners.size === 0) {
      connection.listeners.delete(eventName);
    }
    defSseService.ReleaseSse(request);
  };
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

/** 解析 AI 助手流式事件负载。 */
function parseAiAssistantPayload(raw: string, event: SseEvent): SseAiAssistantPayload | null {
  if (!raw) {
    return null;
  }

  try {
    const payload = JSON.parse(raw) as SseAiAssistantPayload;
    if (payload.event !== event || !payload.session_id || !payload.client_message_id) {
      return null;
    }
    return payload;
  } catch {
    return null;
  }
}

/** 将 SSE 事件枚举转换为 EventSource 事件名称。 */
function toSseEventName(event: SseEvent) {
  return String(event);
}
