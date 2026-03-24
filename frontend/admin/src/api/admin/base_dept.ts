import service from "@/utils/request";
import {
  type BaseDeptForm,
  type BaseDeptService,
  type TreeBaseDeptResponse,
} from "@/rpc/admin/base_dept";
import { type Empty } from "@/rpc/google/protobuf/empty";
import { type Int64Value, type StringValue } from "@/rpc/google/protobuf/wrappers";
import { type SetStatusRequest, type TreeOptionResponse } from "@/rpc/common/common";

const BASE_DEPT_URL = "/admin/base/dept";

/** Admin部门服务 */
export class BaseDeptServiceImpl implements BaseDeptService {
  /** 查询部门树形列表 */
  TreeBaseDept(request: Empty): Promise<TreeBaseDeptResponse> {
    return service<Empty, TreeBaseDeptResponse>({
      url: `${BASE_DEPT_URL}/tree`,
      method: "get",
      params: request,
    });
  }
  /** 查询部门树形选择 */
  OptionBaseDept(request: Empty): Promise<TreeOptionResponse> {
    return service<Empty, TreeOptionResponse>({
      url: `${BASE_DEPT_URL}/option`,
      method: "get",
      params: request,
    });
  }
  /** 查询部门 */
  GetBaseDept(request: Int64Value): Promise<BaseDeptForm> {
    return service<Int64Value, BaseDeptForm>({
      url: `${BASE_DEPT_URL}/${request.value}`,
      method: "get",
    });
  }
  /** 创建部门 */
  CreateBaseDept(request: BaseDeptForm): Promise<Empty> {
    return service<BaseDeptForm, Empty>({
      url: `${BASE_DEPT_URL}`,
      method: "post",
      data: request,
    });
  }
  /** 更新部门 */
  UpdateBaseDept(request: BaseDeptForm): Promise<Empty> {
    return service<BaseDeptForm, Empty>({
      url: `${BASE_DEPT_URL}/${request.id}`,
      method: "put",
      data: request,
    });
  }
  /** 删除部门 */
  DeleteBaseDept(request: StringValue): Promise<Empty> {
    return service<Int64Value, Empty>({
      url: `${BASE_DEPT_URL}/${request.value}`,
      method: "delete",
    });
  }
  /** 设置状态 */
  SetBaseDeptStatus(request: SetStatusRequest): Promise<Empty> {
    return service<SetStatusRequest, Empty>({
      url: `${BASE_DEPT_URL}/${request.id}/status`,
      method: "put",
      data: request,
    });
  }
}

export const defBaseDeptService = new BaseDeptServiceImpl();
