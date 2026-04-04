import service from "@/utils/request";
import type {
  WorkspaceMetricsRequest,
  WorkspaceMetricsResponse,
  WorkspaceRiskListRequest,
  WorkspaceRiskListResponse,
  WorkspaceService,
  WorkspaceTodoListRequest,
  WorkspaceTodoListResponse
} from "@/rpc/admin/workspace";

const ADMIN_WORKSPACE = "/admin/workspace";

/** Admin 工作台服务 */
export class WorkspaceServiceImpl implements WorkspaceService {
  /** 查询工作台顶部指标 */
  GetWorkspaceMetrics(request: WorkspaceMetricsRequest): Promise<WorkspaceMetricsResponse> {
    return service<WorkspaceMetricsRequest, WorkspaceMetricsResponse>({
      url: `${ADMIN_WORKSPACE}/metric`,
      method: "get",
      params: request
    });
  }

  /** 查询工作台待处理事项 */
  GetWorkspaceTodoList(request: WorkspaceTodoListRequest): Promise<WorkspaceTodoListResponse> {
    return service<WorkspaceTodoListRequest, WorkspaceTodoListResponse>({
      url: `${ADMIN_WORKSPACE}/todo`,
      method: "get",
      params: request
    });
  }

  /** 查询工作台风险提醒 */
  GetWorkspaceRiskList(request: WorkspaceRiskListRequest): Promise<WorkspaceRiskListResponse> {
    return service<WorkspaceRiskListRequest, WorkspaceRiskListResponse>({
      url: `${ADMIN_WORKSPACE}/risk`,
      method: "get",
      params: request
    });
  }
}

export const defWorkspaceService = new WorkspaceServiceImpl();
