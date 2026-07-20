import { http } from '@/utils/http'
import type {
  AiService,
  CreateAiSessionBranchRequest,
  CreateAiSessionBranchResponse,
  CreateAiSessionRequest,
  CreateAiSessionResponse,
  DeleteAiSessionRequest,
  DeleteAiSessionResponse,
  ListAiMessageRequest,
  ListAiMessageResponse,
  ListAiShortcutRequest,
  ListAiShortcutResponse,
  ListAiSessionRequest,
  ListAiSessionResponse,
  UpdateAiSessionRequest,
  UpdateAiSessionResponse,
} from '@/rpc/base/v1/ai_session'

const AI_SHORTCUT_URL = '/v1/base/ai/shortcut'
const AI_SESSION_URL = '/v1/base/ai/session'

/** AI 助手会话服务。 */
export class AiSessionServiceImpl implements AiService {
  /** 查询 AI 助手快捷入口列表。 */
  ListAiShortcut(request: ListAiShortcutRequest): Promise<ListAiShortcutResponse> {
    return http<ListAiShortcutResponse>({
      url: AI_SHORTCUT_URL,
      method: 'GET',
      authMode: 'required',
      data: request,
    })
  }

  /** 查询 AI 助手会话列表。 */
  ListAiSession(request: ListAiSessionRequest): Promise<ListAiSessionResponse> {
    return http<ListAiSessionResponse>({
      url: AI_SESSION_URL,
      method: 'GET',
      authMode: 'required',
      data: request,
    })
  }

  /** 创建 AI 助手会话。 */
  CreateAiSession(request: CreateAiSessionRequest): Promise<CreateAiSessionResponse> {
    return http<CreateAiSessionResponse>({
      url: AI_SESSION_URL,
      method: 'POST',
      authMode: 'required',
      data: request,
    })
  }

  /** 更新 AI 助手会话。 */
  UpdateAiSession(request: UpdateAiSessionRequest): Promise<UpdateAiSessionResponse> {
    return http<UpdateAiSessionResponse>({
      url: `${AI_SESSION_URL}/${request.id}`,
      method: 'PUT',
      authMode: 'required',
      data: request,
    })
  }

  /** 删除 AI 助手会话。 */
  DeleteAiSession(request: DeleteAiSessionRequest): Promise<DeleteAiSessionResponse> {
    return http<DeleteAiSessionResponse>({
      url: `${AI_SESSION_URL}/${request.id}`,
      method: 'DELETE',
      authMode: 'required',
      data: request,
    })
  }

  /** 查询 AI 助手消息列表。 */
  ListAiMessage(request: ListAiMessageRequest): Promise<ListAiMessageResponse> {
    return http<ListAiMessageResponse>({
      url: `${AI_SESSION_URL}/${request.session_id}/message`,
      method: 'GET',
      authMode: 'required',
      data: request,
    })
  }

  /** 从指定消息创建 AI 助手分支会话。 */
  CreateAiSessionBranch(
    request: CreateAiSessionBranchRequest,
  ): Promise<CreateAiSessionBranchResponse> {
    return http<CreateAiSessionBranchResponse>({
      url: `${AI_SESSION_URL}/${request.source_session_id}/branch`,
      method: 'POST',
      authMode: 'required',
      data: request,
    })
  }
}

export const defAiSessionService = new AiSessionServiceImpl()
