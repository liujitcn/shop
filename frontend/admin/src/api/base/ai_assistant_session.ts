import service from "@/utils/request";
import type {
  AiAssistantService,
  CreateAiAssistantSessionBranchRequest,
  CreateAiAssistantSessionBranchResponse,
  CreateAiAssistantSessionRequest,
  CreateAiAssistantSessionResponse,
  DeleteAiAssistantSessionRequest,
  DeleteAiAssistantSessionResponse,
  ListAiAssistantMessageRequest,
  ListAiAssistantMessageResponse,
  ListAiAssistantShortcutRequest,
  ListAiAssistantShortcutResponse,
  ListAiAssistantSessionRequest,
  ListAiAssistantSessionResponse,
  UpdateAiAssistantSessionRequest,
  UpdateAiAssistantSessionResponse
} from "@/rpc/base/v1/ai_assistant_session";

const AI_ASSISTANT_SHORTCUT_URL = "/v1/base/ai/assistant/shortcut";
const AI_ASSISTANT_SESSION_URL = "/v1/base/ai/assistant/session";

/** AI 助手会话服务。 */
export class AiAssistantSessionServiceImpl implements AiAssistantService {
  /** 查询 AI 助手快捷入口列表。 */
  ListAiAssistantShortcut(request: ListAiAssistantShortcutRequest): Promise<ListAiAssistantShortcutResponse> {
    return service<ListAiAssistantShortcutRequest, ListAiAssistantShortcutResponse>({
      url: `${AI_ASSISTANT_SHORTCUT_URL}`,
      method: "get",
      params: request
    });
  }

  /** 查询 AI 助手会话列表。 */
  ListAiAssistantSession(request: ListAiAssistantSessionRequest): Promise<ListAiAssistantSessionResponse> {
    return service<ListAiAssistantSessionRequest, ListAiAssistantSessionResponse>({
      url: `${AI_ASSISTANT_SESSION_URL}`,
      method: "get",
      params: request
    });
  }

  /** 创建 AI 助手会话。 */
  CreateAiAssistantSession(request: CreateAiAssistantSessionRequest): Promise<CreateAiAssistantSessionResponse> {
    return service<CreateAiAssistantSessionRequest, CreateAiAssistantSessionResponse>({
      url: `${AI_ASSISTANT_SESSION_URL}`,
      method: "post",
      data: request
    });
  }

  /** 更新 AI 助手会话。 */
  UpdateAiAssistantSession(request: UpdateAiAssistantSessionRequest): Promise<UpdateAiAssistantSessionResponse> {
    return service<UpdateAiAssistantSessionRequest, UpdateAiAssistantSessionResponse>({
      url: `${AI_ASSISTANT_SESSION_URL}/${request.id}`,
      method: "put",
      data: request
    });
  }

  /** 删除 AI 助手会话。 */
  DeleteAiAssistantSession(request: DeleteAiAssistantSessionRequest): Promise<DeleteAiAssistantSessionResponse> {
    return service<DeleteAiAssistantSessionRequest, DeleteAiAssistantSessionResponse>({
      url: `${AI_ASSISTANT_SESSION_URL}/${request.id}`,
      method: "delete",
      params: request
    });
  }

  /** 查询 AI 助手消息列表。 */
  ListAiAssistantMessage(request: ListAiAssistantMessageRequest): Promise<ListAiAssistantMessageResponse> {
    return service<ListAiAssistantMessageRequest, ListAiAssistantMessageResponse>({
      url: `${AI_ASSISTANT_SESSION_URL}/${request.session_id}/message`,
      method: "get",
      params: request
    });
  }

  /** 从指定消息创建 AI 助手分支会话。 */
  CreateAiAssistantSessionBranch(request: CreateAiAssistantSessionBranchRequest): Promise<CreateAiAssistantSessionBranchResponse> {
    return service<CreateAiAssistantSessionBranchRequest, CreateAiAssistantSessionBranchResponse>({
      url: `${AI_ASSISTANT_SESSION_URL}/${request.source_session_id}/branch`,
      method: "post",
      data: request
    });
  }
}

export const defAiAssistantSessionService = new AiAssistantSessionServiceImpl();
