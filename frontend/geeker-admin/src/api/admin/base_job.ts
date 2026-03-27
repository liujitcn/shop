import service from "@/utils/request";
import {
  type BaseJobForm,
  type BaseJobLog,
  type BaseJobService,
  type ExecBaseJobRequest,
  type PageBaseJobLogRequest,
  type PageBaseJobLogResponse,
  type PageBaseJobRequest,
  type PageBaseJobResponse,
  type StartBaseJobRequest,
  type StopBaseJobRequest,
} from "@/rpc/admin/base_job";
import type { Empty } from "@/rpc/google/protobuf/empty";
import type { Int64Value, StringValue } from "@/rpc/google/protobuf/wrappers";
import type { SetStatusRequest } from "@/rpc/common/common";

const BASE_JOB_URL = "/admin/base/job";
const BASE_JOB_LOG_URL = "/admin/base/job-log";

/** Admin定时任务服务 */
export class BaseJobServiceImpl implements BaseJobService {
  /** 查询定时任务分页列表 */
  PageBaseJob(request: PageBaseJobRequest): Promise<PageBaseJobResponse> {
    return service<PageBaseJobRequest, PageBaseJobResponse>({
      url: `${BASE_JOB_URL}`,
      method: "get",
      params: request,
    });
  }
  /** 查询定时任务 */
  GetBaseJob(request: Int64Value): Promise<BaseJobForm> {
    return service<Int64Value, BaseJobForm>({
      url: `${BASE_JOB_URL}/${request.value}`,
      method: "get",
    });
  }
  /** 创建定时任务 */
  CreateBaseJob(request: BaseJobForm): Promise<Empty> {
    return service<BaseJobForm, Empty>({
      url: `${BASE_JOB_URL}`,
      method: "post",
      data: request,
    });
  }
  /** 更新定时任务 */
  UpdateBaseJob(request: BaseJobForm): Promise<Empty> {
    return service<BaseJobForm, Empty>({
      url: `${BASE_JOB_URL}/${request.id}`,
      method: "put",
      data: request,
    });
  }
  /** 删除定时任务 */
  DeleteBaseJob(request: StringValue): Promise<Empty> {
    return service<StringValue, Empty>({
      url: `${BASE_JOB_URL}/${request.value}`,
      method: "delete",
    });
  }
  /** 设置状态 */
  SetBaseJobStatus(request: SetStatusRequest): Promise<Empty> {
    return service<SetStatusRequest, Empty>({
      url: `${BASE_JOB_URL}/${request.id}/status`,
      method: "put",
      data: request,
    });
  }
  /** 启动任务 */
  StartBaseJob(request: StartBaseJobRequest): Promise<Empty> {
    return service<SetStatusRequest, Empty>({
      url: `${BASE_JOB_URL}/${request.id}/start`,
      method: "put",
      data: request,
    });
  }
  /** 停止任务 */
  StopBaseJob(request: StopBaseJobRequest): Promise<Empty> {
    return service<SetStatusRequest, Empty>({
      url: `${BASE_JOB_URL}/${request.id}/stop`,
      method: "put",
      data: request,
    });
  }
  /** 执行任务 */
  ExecBaseJob(request: ExecBaseJobRequest): Promise<Empty> {
    return service<SetStatusRequest, Empty>({
      url: `${BASE_JOB_URL}/${request.id}/exec`,
      method: "put",
      data: request,
    });
  }
  /** 查询定时任务日志分页列表 */
  PageBaseJobLog(request: PageBaseJobLogRequest): Promise<PageBaseJobLogResponse> {
    return service<PageBaseJobLogRequest, PageBaseJobLogResponse>({
      url: `${BASE_JOB_LOG_URL}`,
      method: "get",
      params: request,
    });
  }
  /** 查询定时任务日志 */
  GetBaseJobLog(request: Int64Value): Promise<BaseJobLog> {
    return service<Int64Value, BaseJobLog>({
      url: `${BASE_JOB_LOG_URL}/${request.value}`,
      method: "get",
      params: request,
    });
  }
}

export const defBaseJobService = new BaseJobServiceImpl();
