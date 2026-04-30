import service from "@/utils/request";
import {
  type BaseDeptForm,
  type BaseDeptService,
  type CreateBaseDeptRequest,
  type DeleteBaseDeptRequest,
  type GetBaseDeptRequest,
  type OptionBaseDeptsRequest,
  type SetBaseDeptStatusRequest,
  type TreeBaseDeptsRequest,
  type TreeBaseDeptsResponse,
  type UpdateBaseDeptRequest
} from "@/rpc/admin/v1/base_dept";
import { type Empty } from "@/rpc/google/protobuf/empty";
import { type TreeOptionResponse } from "@/rpc/common/v1/common";

const BASE_DEPT_URL = "/v1/admin/base/dept";

/** Admin部门服务 */
export class BaseDeptServiceImpl implements BaseDeptService {
  /** 查询部门树形列表 */
  TreeBaseDepts(request: TreeBaseDeptsRequest): Promise<TreeBaseDeptsResponse> {
    return service<TreeBaseDeptsRequest, TreeBaseDeptsResponse>({
      url: `${BASE_DEPT_URL}/tree`,
      method: "get",
      params: request
    });
  }

  /** 查询部门树形选择 */
  OptionBaseDepts(request: OptionBaseDeptsRequest): Promise<TreeOptionResponse> {
    return service<OptionBaseDeptsRequest, TreeOptionResponse>({
      url: `${BASE_DEPT_URL}/option`,
      method: "get",
      params: request
    });
  }

  /** 查询部门 */
  GetBaseDept(request: GetBaseDeptRequest): Promise<BaseDeptForm> {
    return service<GetBaseDeptRequest, BaseDeptForm>({
      url: `${BASE_DEPT_URL}/${request.id}`,
      method: "get"
    });
  }

  /** 创建部门 */
  CreateBaseDept(request: CreateBaseDeptRequest): Promise<Empty> {
    return service<BaseDeptForm | undefined, Empty>({
      url: `${BASE_DEPT_URL}`,
      method: "post",
      data: request.base_dept
    });
  }

  /** 更新部门 */
  UpdateBaseDept(request: UpdateBaseDeptRequest): Promise<Empty> {
    return service<BaseDeptForm | undefined, Empty>({
      url: `${BASE_DEPT_URL}/${request.base_dept?.id ?? ""}`,
      method: "put",
      data: request.base_dept
    });
  }

  /** 删除部门 */
  DeleteBaseDept(request: DeleteBaseDeptRequest): Promise<Empty> {
    return service<DeleteBaseDeptRequest, Empty>({
      url: `${BASE_DEPT_URL}/${request.id}`,
      method: "delete"
    });
  }

  /** 设置状态 */
  SetBaseDeptStatus(request: SetBaseDeptStatusRequest): Promise<Empty> {
    return service<SetBaseDeptStatusRequest, Empty>({
      url: `${BASE_DEPT_URL}/${request.id}/status`,
      method: "put",
      data: request
    });
  }
}

export const defBaseDeptService = new BaseDeptServiceImpl();
