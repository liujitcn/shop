import service from "@/utils/request";
import {
  type BaseRoleForm,
  type BaseRoleService,
  type PageBaseRoleRequest,
  type PageBaseRoleResponse,
  type SetMenusRequest,
} from "@/rpc/admin/base_role";
import type { Empty } from "@/rpc/google/protobuf/empty";
import type { Int64Value, StringValue } from "@/rpc/google/protobuf/wrappers";
import type { SelectOptionResponse, SetStatusRequest } from "@/rpc/common/common";

const BASE_ROLE_URL = "/admin/base/role";

/** Admin角色服务 */
export class BaseRoleServiceImpl implements BaseRoleService {
  /** 查询角色下拉选择 */
  OptionBaseRole(request: Empty): Promise<SelectOptionResponse> {
    return service<Empty, SelectOptionResponse>({
      url: `${BASE_ROLE_URL}/option`,
      method: "get",
      params: request,
    });
  }
  /** 查询角色分页列表 */
  PageBaseRole(request: PageBaseRoleRequest): Promise<PageBaseRoleResponse> {
    return service<PageBaseRoleRequest, PageBaseRoleResponse>({
      url: `${BASE_ROLE_URL}`,
      method: "get",
      params: request,
    });
  }
  /** 查询角色 */
  GetBaseRole(request: Int64Value): Promise<BaseRoleForm> {
    return service<Int64Value, BaseRoleForm>({
      url: `${BASE_ROLE_URL}/${request.value}`,
      method: "get",
    });
  }
  /** 创建角色 */
  CreateBaseRole(request: BaseRoleForm): Promise<Empty> {
    return service<BaseRoleForm, Empty>({
      url: `${BASE_ROLE_URL}`,
      method: "post",
      data: request,
    });
  }
  /** 更新角色 */
  UpdateBaseRole(request: BaseRoleForm): Promise<Empty> {
    return service<BaseRoleForm, Empty>({
      url: `${BASE_ROLE_URL}/${request.id}`,
      method: "put",
      data: request,
    });
  }
  /** 删除角色 */
  DeleteBaseRole(request: StringValue): Promise<Empty> {
    return service<StringValue, Empty>({
      url: `${BASE_ROLE_URL}/${request.value}`,
      method: "delete",
    });
  }
  /** 设置状态 */
  SetBaseRoleStatus(request: SetStatusRequest): Promise<Empty> {
    return service<SetStatusRequest, Empty>({
      url: `${BASE_ROLE_URL}/${request.id}/status`,
      method: "put",
      data: request,
    });
  }
  /** 设置角色菜单权限 */
  SetBaseRoleMenus(request: SetMenusRequest): Promise<Empty> {
    return service<SetStatusRequest, Empty>({
      url: `${BASE_ROLE_URL}/` + request.id + "/menus",
      method: "put",
      data: request,
    });
  }
}

export const defBaseRoleService = new BaseRoleServiceImpl();
