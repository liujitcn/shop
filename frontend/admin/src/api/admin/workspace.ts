import service from "@/utils/request";
import type {
  ListWorkspacePendingCommentsRequest,
  ListWorkspacePendingCommentsResponse,
  SummaryWorkspaceMetricsRequest,
  SummaryWorkspaceMetricsResponse,
  SummaryWorkspaceRiskRequest,
  SummaryWorkspaceReputationRequest,
  SummaryWorkspaceReputationResponse,
  SummaryWorkspaceRiskResponse,
  SummaryWorkspaceTodoRequest,
  SummaryWorkspaceTodoResponse,
  WorkspaceService
} from "@/rpc/admin/v1/workspace";

const ADMIN_WORKSPACE = "/v1/admin/workspace";

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
  ListWorkspacePendingComments(request: ListWorkspacePendingCommentsRequest): Promise<ListWorkspacePendingCommentsResponse> {
    return service<ListWorkspacePendingCommentsRequest, ListWorkspacePendingCommentsResponse>({
      url: `${ADMIN_WORKSPACE}/comment/pending`,
      method: "get",
      params: request
    });
  }
}

export const defWorkspaceService = new WorkspaceServiceImpl();
