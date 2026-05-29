import service, { getRequestAccessToken, handleAuthExpired, requestBaseURL } from "@/utils/request";
import type { ListAiAssistantMessagesRequest, ListAiAssistantMessagesResponse } from "@/rpc/base/v1/ai_assistant_session";
import type {
  AiAssistantMessageService,
  DeleteAiAssistantMessageRequest,
  DeleteAiAssistantMessageResponse,
  RegenerateAiAssistantMessageRequest,
  RetryAiAssistantUserMessageRequest,
  SendAiAssistantMessageRequest,
  SendAiAssistantMessageResponse
} from "@/rpc/base/v1/ai_assistant_message";

const AI_ASSISTANT_SESSION_URL = "/v1/base/ai/assistant/session";

/** 从 direct stream 错误响应中提取后端业务提示。 */
async function resolveStreamErrorMessage(response: Response): Promise<string> {
  const fallbackMessage = `AI 助手请求失败（${response.status}）`;
  const contentType = response.headers.get("Content-Type") ?? "";
  if (contentType.includes("application/json")) {
    try {
      const payload = await response.json();
      return String(payload?.message || payload?.error || fallbackMessage);
    } catch {
      return fallbackMessage;
    }
  }
  try {
    const text = (await response.text()).trim();
    return text || fallbackMessage;
  } catch {
    return fallbackMessage;
  }
}

/** 使用 direct stream 发送 AI 助手消息，并返回原始 Fetch Response 供 useXStream 消费。 */
export async function SendAiAssistantMessageStream(request: SendAiAssistantMessageRequest): Promise<Response> {
  const accessToken = await getRequestAccessToken();
  const headers: Record<string, string> = {
    Accept: "text/event-stream",
    "Content-Type": "application/json;charset=utf-8"
  };
  if (accessToken) headers.Authorization = accessToken;

  const response = await fetch(`${requestBaseURL}${AI_ASSISTANT_SESSION_URL}/${request.session_id}/message`, {
    method: "POST",
    headers,
    body: JSON.stringify(request)
  });

  // direct stream 不经过 axios 响应拦截器，需要在这里补齐登录失效处理。
  if (response.status === 401 || response.status === 403) {
    handleAuthExpired();
    throw new Error("登录状态已失效，请重新登录");
  }
  if (!response.ok) {
    throw new Error(await resolveStreamErrorMessage(response));
  }

  return response;
}

/** AI 助手消息服务。 */
export class AiAssistantMessageServiceImpl implements AiAssistantMessageService {
  /** 查询 AI 助手消息列表。 */
  ListAiAssistantMessages(request: ListAiAssistantMessagesRequest): Promise<ListAiAssistantMessagesResponse> {
    return service<ListAiAssistantMessagesRequest, ListAiAssistantMessagesResponse>({
      url: `${AI_ASSISTANT_SESSION_URL}/${request.session_id}/message`,
      method: "get",
      params: request
    });
  }

  /** 发送 AI 助手消息并等待完整响应。 */
  SendAiAssistantMessage(request: SendAiAssistantMessageRequest): Promise<SendAiAssistantMessageResponse> {
    return service<SendAiAssistantMessageRequest, SendAiAssistantMessageResponse>({
      url: `${AI_ASSISTANT_SESSION_URL}/${request.session_id}/message`,
      method: "post",
      data: request
    });
  }

  /** 删除 AI 助手消息。 */
  DeleteAiAssistantMessage(request: DeleteAiAssistantMessageRequest): Promise<DeleteAiAssistantMessageResponse> {
    return service<DeleteAiAssistantMessageRequest, DeleteAiAssistantMessageResponse>({
      url: `${AI_ASSISTANT_SESSION_URL}/${request.session_id}/message/${request.message_id}`,
      method: "delete",
      params: request
    });
  }

  /** 重试失败的用户消息。 */
  RetryAiAssistantUserMessage(request: RetryAiAssistantUserMessageRequest): Promise<SendAiAssistantMessageResponse> {
    return service<RetryAiAssistantUserMessageRequest, SendAiAssistantMessageResponse>({
      url: `${AI_ASSISTANT_SESSION_URL}/${request.session_id}/message/${request.message_id}/retry`,
      method: "post",
      data: request
    });
  }

  /** 重新生成助手回复。 */
  RegenerateAiAssistantMessage(request: RegenerateAiAssistantMessageRequest): Promise<SendAiAssistantMessageResponse> {
    return service<RegenerateAiAssistantMessageRequest, SendAiAssistantMessageResponse>({
      url: `${AI_ASSISTANT_SESSION_URL}/${request.session_id}/message/${request.message_id}/regeneration`,
      method: "post",
      data: request
    });
  }

  /** 使用 direct stream 发送 AI 助手消息，并返回原始 Fetch Response 供 useXStream 消费。 */
  async StreamAiAssistantMessage(request: SendAiAssistantMessageRequest): Promise<Response> {
    return SendAiAssistantMessageStream(request);
  }
}

export const defAiAssistantMessageService = new AiAssistantMessageServiceImpl();
