import service from "@/utils/request";
import type {
  AiAssistantService,
  AiAssistantSession,
  CreateAiAssistantSessionRequest,
  DeleteAiAssistantSessionRequest,
  OperateAiAssistantConfirmRequest,
  OperateAiAssistantConfirmResponse,
  ListAiAssistantMessagesRequest,
  ListAiAssistantMessagesResponse,
  ListAiAssistantSessionsRequest,
  ListAiAssistantSessionsResponse,
  SendAiAssistantMessageRequest,
  SendAiAssistantMessageResponse,
  UpdateAiAssistantSessionRequest
} from "@/rpc/base/v1/ai_assistant";
import type { Empty } from "@/rpc/google/protobuf/empty";

const AI_ASSISTANT_SESSION_URL = "/v1/base/ai/assistant/session";

/** AI 助手公共服务。 */
export class AiAssistantServiceImpl implements AiAssistantService {
  /** 查询 AI 助手会话列表。 */
  ListAiAssistantSessions(request: ListAiAssistantSessionsRequest): Promise<ListAiAssistantSessionsResponse> {
    return service<ListAiAssistantSessionsRequest, ListAiAssistantSessionsResponse>({
      url: `${AI_ASSISTANT_SESSION_URL}`,
      method: "get",
      params: request
    });
  }

  /** 创建 AI 助手会话。 */
  CreateAiAssistantSession(request: CreateAiAssistantSessionRequest): Promise<AiAssistantSession> {
    return service<CreateAiAssistantSessionRequest, AiAssistantSession>({
      url: `${AI_ASSISTANT_SESSION_URL}`,
      method: "post",
      data: request
    });
  }

  /** 更新 AI 助手会话。 */
  UpdateAiAssistantSession(request: UpdateAiAssistantSessionRequest): Promise<AiAssistantSession> {
    return service<UpdateAiAssistantSessionRequest, AiAssistantSession>({
      url: `${AI_ASSISTANT_SESSION_URL}/${request.id}`,
      method: "put",
      data: request
    });
  }

  /** 删除 AI 助手会话。 */
  DeleteAiAssistantSession(request: DeleteAiAssistantSessionRequest): Promise<Empty> {
    return service<DeleteAiAssistantSessionRequest, Empty>({
      url: `${AI_ASSISTANT_SESSION_URL}/${request.id}`,
      method: "delete",
      params: request
    });
  }

  /** 查询 AI 助手消息列表。 */
  ListAiAssistantMessages(request: ListAiAssistantMessagesRequest): Promise<ListAiAssistantMessagesResponse> {
    return service<ListAiAssistantMessagesRequest, ListAiAssistantMessagesResponse>({
      url: `${AI_ASSISTANT_SESSION_URL}/${request.session_id}/message`,
      method: "get",
      params: request
    });
  }

  /** 发送 AI 助手消息。 */
  SendAiAssistantMessage(request: SendAiAssistantMessageRequest): Promise<SendAiAssistantMessageResponse> {
    return service<SendAiAssistantMessageRequest, SendAiAssistantMessageResponse>({
      url: `${AI_ASSISTANT_SESSION_URL}/${request.session_id}/message`,
      method: "post",
      data: request
    });
  }

  /** 处理 AI 助手确认卡动作。 */
  OperateAiAssistantConfirm(request: OperateAiAssistantConfirmRequest): Promise<OperateAiAssistantConfirmResponse> {
    return service<OperateAiAssistantConfirmRequest, OperateAiAssistantConfirmResponse>({
      url: `${AI_ASSISTANT_SESSION_URL}/${request.session_id}/confirm/${request.message_id}`,
      method: "post",
      data: request
    });
  }
}

export const defAiAssistantService = new AiAssistantServiceImpl();
