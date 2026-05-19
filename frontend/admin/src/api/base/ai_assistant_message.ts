import service, { handleAuthExpired } from "@/utils/request";
import type { ListAiAssistantMessagesRequest, ListAiAssistantMessagesResponse } from "@/rpc/base/v1/ai_assistant_session";
import type { SendAiAssistantMessageRequest } from "@/rpc/base/v1/ai_assistant_message";
import pinia from "@/stores";
import { useUserStore } from "@/stores/modules/user";

const AI_ASSISTANT_SESSION_URL = "/v1/base/ai/assistant/session";
const apiBasePath = import.meta.env.VITE_APP_BASE_API || "";
const apiTargetUrl = import.meta.env.VITE_API_URL || import.meta.env.VITE_APP_API_URL || "";
const baseURL = `${apiTargetUrl}${apiBasePath}`;

/** 读取当前访问令牌，必要时先刷新，保持 direct stream 与 axios 请求一致的认证行为。 */
async function getAccessToken(): Promise<string> {
  const userStore = useUserStore(pinia);
  const expiresAt = userStore.tokenExpiresAt;
  const remainingTime = expiresAt - Date.now();
  if (expiresAt && remainingTime <= 5 * 60 * 1000) {
    await userStore.refreshAccessToken();
  }
  return userStore.token.trim();
}

/** 使用 direct stream 发送 AI 助手消息，并返回原始 Fetch Response 供 useXStream 消费。 */
export async function SendAiAssistantMessageStream(request: SendAiAssistantMessageRequest): Promise<Response> {
  const accessToken = await getAccessToken();
  const headers: Record<string, string> = {
    Accept: "text/event-stream",
    "Content-Type": "application/json;charset=utf-8"
  };
  if (accessToken) headers.Authorization = accessToken;

  const response = await fetch(`${baseURL}${AI_ASSISTANT_SESSION_URL}/${request.session_id}/message`, {
    method: "POST",
    headers,
    body: JSON.stringify(request)
  });

  // direct stream 不经过 axios 响应拦截器，需要在这里补齐登录失效处理。
  if (response.status === 401 || response.status === 403) {
    handleAuthExpired();
    throw new Error("登录状态已失效，请重新登录");
  }

  return response;
}

/** AI 助手消息服务。 */
export class AiAssistantMessageServiceImpl {
  /** 查询 AI 助手消息列表。 */
  ListAiAssistantMessages(request: ListAiAssistantMessagesRequest): Promise<ListAiAssistantMessagesResponse> {
    return service<ListAiAssistantMessagesRequest, ListAiAssistantMessagesResponse>({
      url: `${AI_ASSISTANT_SESSION_URL}/${request.session_id}/message`,
      method: "get",
      params: request
    });
  }

  /** 使用 direct stream 发送 AI 助手消息，并返回原始 Fetch Response 供 useXStream 消费。 */
  async SendAiAssistantMessage(request: SendAiAssistantMessageRequest): Promise<Response> {
    return SendAiAssistantMessageStream(request);
  }
}

export const defAiAssistantMessageService = new AiAssistantMessageServiceImpl();
