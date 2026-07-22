import service from "@/utils/request";
import { subscribeSseEvent, type SseStop } from "@/api/base/sse";
import type { SseRefreshReason, SseRefreshTarget } from "@/rpc/shop/common/v1/enum";
import type {
  ListWorkspacePendingCommentRequest,
  ListWorkspacePendingCommentResponse,
  SummaryWorkspaceMetricsRequest,
  SummaryWorkspaceMetricsResponse,
  SummaryWorkspaceRiskRequest,
  SummaryWorkspaceReputationRequest,
  SummaryWorkspaceReputationResponse,
  SummaryWorkspaceRiskResponse,
  SummaryWorkspaceTodoRequest,
  SummaryWorkspaceTodoResponse,
  WorkspaceService
} from "@/rpc/shop/admin/v1/workspace";

const ADMIN_WORKSPACE = "/v1/admin/workspace";
const WORKSPACE_SSE_STREAM = "shop.admin.workspace";
const WORKSPACE_SSE_EVENT_REFRESH = "workspace.refresh";

/** 商城工作台 SSE 刷新事件负载。 */
interface WorkspaceSseRefreshPayload {
  /** 事件名称。 */
  event: string;
  /** 需要刷新的页面目标。 */
  targets: SseRefreshTarget[];
  /** 触发刷新原因。 */
  reason?: SseRefreshReason;
  /** 事件发生时间。 */
  occurred_at: string;
}

/** 商城工作台 SSE 刷新事件处理函数。 */
export type WorkspaceSseRefreshHandler = (payload: WorkspaceSseRefreshPayload) => void;

/** Admin 工作台服务 */
export class WorkspaceServiceImpl implements WorkspaceService {
  /** 查询工作台顶部指标 */
  SummaryWorkspaceMetrics(request: SummaryWorkspaceMetricsRequest): Promise<SummaryWorkspaceMetricsResponse> {
    return service<SummaryWorkspaceMetricsRequest, SummaryWorkspaceMetricsResponse>({
      url: `${ADMIN_WORKSPACE}/metrics/summary`,
      method: "get",
      params: request
    });
  }

  /** 查询工作台待处理事项 */
  SummaryWorkspaceTodo(request: SummaryWorkspaceTodoRequest): Promise<SummaryWorkspaceTodoResponse> {
    return service<SummaryWorkspaceTodoRequest, SummaryWorkspaceTodoResponse>({
      url: `${ADMIN_WORKSPACE}/todo/summary`,
      method: "get",
      params: request
    });
  }

  /** 查询工作台风险提醒 */
  SummaryWorkspaceRisk(request: SummaryWorkspaceRiskRequest): Promise<SummaryWorkspaceRiskResponse> {
    return service<SummaryWorkspaceRiskRequest, SummaryWorkspaceRiskResponse>({
      url: `${ADMIN_WORKSPACE}/risk/summary`,
      method: "get",
      params: request
    });
  }

  /** 查询工作台口碑洞察 */
  SummaryWorkspaceReputation(request: SummaryWorkspaceReputationRequest): Promise<SummaryWorkspaceReputationResponse> {
    return service<SummaryWorkspaceReputationRequest, SummaryWorkspaceReputationResponse>({
      url: `${ADMIN_WORKSPACE}/reputation/summary`,
      method: "get",
      params: request
    });
  }

  /** 查询工作台待审核评价 */
  ListWorkspacePendingComment(request: ListWorkspacePendingCommentRequest): Promise<ListWorkspacePendingCommentResponse> {
    return service<ListWorkspacePendingCommentRequest, ListWorkspacePendingCommentResponse>({
      url: `${ADMIN_WORKSPACE}/comment/pending`,
      method: "get",
      params: request
    });
  }
}

export const defWorkspaceService = new WorkspaceServiceImpl();

/** 订阅商城工作台局部刷新事件。 */
export function subscribeWorkspaceSseRefresh(handler: WorkspaceSseRefreshHandler): SseStop {
  return subscribeSseEvent(
    { stream: WORKSPACE_SSE_STREAM },
    WORKSPACE_SSE_EVENT_REFRESH,
    parseWorkspaceSseRefreshPayload,
    handler
  );
}

/** 解析商城工作台刷新事件负载。 */
function parseWorkspaceSseRefreshPayload(raw: string): WorkspaceSseRefreshPayload | null {
  if (!raw) {
    return null;
  }
  try {
    const payload = JSON.parse(raw) as WorkspaceSseRefreshPayload;
    if (payload.event !== WORKSPACE_SSE_EVENT_REFRESH || !Array.isArray(payload.targets)) {
      return null;
    }
    return payload;
  } catch {
    return null;
  }
}
