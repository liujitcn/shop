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

/** SSE 刷新事件处理函数。 */
export type SseRefreshHandler = (payload: SseRefreshPayload) => void;

/** SSE 订阅连接配置。 */
export interface SubscribeSseOptions {
  /** 是否携带跨域凭据。 */
  withCredentials?: boolean;
}

/** Base SSE 服务。 */
export class SseServiceImpl {
  /** 创建 SSE 订阅连接。 */
  SubscribeSse(request: SubscribeSseRequest, options?: SubscribeSseOptions): EventSource | null {
    if (typeof window === "undefined" || typeof EventSource === "undefined") {
      return null;
    }

    const url = this.buildSubscribeURL(request);
    if (!url) {
      return null;
    }

    return new EventSource(url, {
      withCredentials: options?.withCredentials
    });
  }

  /** 构建 SSE 订阅地址。 */
  private buildSubscribeURL(request: SubscribeSseRequest) {
    const token = this.getAccessToken();
    if (!token) {
      return "";
    }

    const url = new URL(SSE_URL, window.location.origin);
    url.searchParams.set("stream", String(request.stream));
    url.searchParams.set("access_token", token);
    return url.toString();
  }

  /** 读取适配 EventSource 查询参数传递的访问令牌。 */
  private getAccessToken() {
    const userStore = useUserStore(pinia);
    const value = userStore.token.trim();
    return value.replace(/^Bearer\s+/i, "");
  }
}

export const defSseService = new SseServiceImpl();

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
  const source = defSseService.SubscribeSse({ stream });
  if (!source) return () => undefined;

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

/** 将 SSE 事件枚举转换为 EventSource 事件名称。 */
function toSseEventName(event: SseEvent) {
  return String(event);
}
