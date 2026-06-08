import { http } from '@/utils/http'
import type {
  AiAssistantService,
  CreateAiAssistantSessionBranchRequest,
  CreateAiAssistantSessionBranchResponse,
  CreateAiAssistantSessionRequest,
  CreateAiAssistantSessionResponse,
  DeleteAiAssistantSessionRequest,
  DeleteAiAssistantSessionResponse,
  ListAiAssistantMessagesRequest,
  ListAiAssistantMessagesResponse,
  ListAiAssistantSessionsRequest,
  ListAiAssistantSessionsResponse,
  UpdateAiAssistantSessionRequest,
  UpdateAiAssistantSessionResponse,
} from '@/rpc/base/v1/ai_assistant_session'

const AI_ASSISTANT_SESSION_URL = '/v1/base/ai/assistant/session'

/** AI 助手会话服务。 */
export class AiAssistantSessionServiceImpl implements AiAssistantService {
  /** 查询 AI 助手会话列表。 */
  ListAiAssistantSessions(
    request: ListAiAssistantSessionsRequest,
  ): Promise<ListAiAssistantSessionsResponse> {
    return http<ListAiAssistantSessionsResponse>({
      url: AI_ASSISTANT_SESSION_URL,
      method: 'GET',
      data: request,
    })
  }

  /** 创建 AI 助手会话。 */
  CreateAiAssistantSession(
    request: CreateAiAssistantSessionRequest,
  ): Promise<CreateAiAssistantSessionResponse> {
    return http<CreateAiAssistantSessionResponse>({
      url: AI_ASSISTANT_SESSION_URL,
      method: 'POST',
      data: request,
    })
  }

  /** 更新 AI 助手会话。 */
  UpdateAiAssistantSession(
    request: UpdateAiAssistantSessionRequest,
  ): Promise<UpdateAiAssistantSessionResponse> {
    return http<UpdateAiAssistantSessionResponse>({
      url: `${AI_ASSISTANT_SESSION_URL}/${request.id}`,
      method: 'PUT',
      data: request,
    })
  }

  /** 删除 AI 助手会话。 */
  DeleteAiAssistantSession(
    request: DeleteAiAssistantSessionRequest,
  ): Promise<DeleteAiAssistantSessionResponse> {
    return http<DeleteAiAssistantSessionResponse>({
      url: `${AI_ASSISTANT_SESSION_URL}/${request.id}`,
      method: 'DELETE',
      data: request,
    })
  }

  /** 查询 AI 助手消息列表。 */
  ListAiAssistantMessages(
    request: ListAiAssistantMessagesRequest,
  ): Promise<ListAiAssistantMessagesResponse> {
    return http<ListAiAssistantMessagesResponse>({
      url: `${AI_ASSISTANT_SESSION_URL}/${request.session_id}/message`,
      method: 'GET',
      data: request,
    })
  }

  /** 从指定消息创建 AI 助手分支会话。 */
  CreateAiAssistantSessionBranch(
    request: CreateAiAssistantSessionBranchRequest,
  ): Promise<CreateAiAssistantSessionBranchResponse> {
    return http<CreateAiAssistantSessionBranchResponse>({
      url: `${AI_ASSISTANT_SESSION_URL}/${request.source_session_id}/branch`,
      method: 'POST',
      data: request,
    })
  }
}

export const defAiAssistantSessionService = new AiAssistantSessionServiceImpl()
