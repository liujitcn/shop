import service from "@/utils/request";
import {
  type BaseMenuForm,
  type BaseMenuService,
  type TreeBaseMenuResponse,
} from "@/rpc/admin/base_menu";
import { type Empty } from "@/rpc/google/protobuf/empty";
import { type Int64Value, type StringValue } from "@/rpc/google/protobuf/wrappers";
import { type SetStatusRequest, type TreeOptionResponse } from "@/rpc/common/common";

const BASE_MENU_URL = "/admin/base/menu";

/** Admin菜单管理服务 */
export class BaseMenuServiceImpl implements BaseMenuService {
  /** 查询部门树形列表 */
  TreeBaseMenu(request: Empty): Promise<TreeBaseMenuResponse> {
    return service<Empty, TreeBaseMenuResponse>({
      url: `${BASE_MENU_URL}/tree`,
      method: "get",
      params: request,
    });
  }
  /** 查询部门树形选择 */
  OptionBaseMenu(request: Empty): Promise<TreeOptionResponse> {
    return service<Empty, TreeOptionResponse>({
      url: `${BASE_MENU_URL}/option`,
      method: "get",
      params: request,
    });
  }
  /** 查询部门 */
  GetBaseMenu(request: Int64Value): Promise<BaseMenuForm> {
    return service<Int64Value, BaseMenuForm>({
      url: `${BASE_MENU_URL}/${request.value}`,
      method: "get",
    });
  }
  /** 创建部门 */
  CreateBaseMenu(request: BaseMenuForm): Promise<Empty> {
    return service<BaseMenuForm, Empty>({
      url: `${BASE_MENU_URL}`,
      method: "post",
      data: request,
    });
  }
  /** 更新部门 */
  UpdateBaseMenu(request: BaseMenuForm): Promise<Empty> {
    return service<BaseMenuForm, Empty>({
      url: `${BASE_MENU_URL}/${request.id}`,
      method: "put",
      data: request,
    });
  }
  /** 删除部门 */
  DeleteBaseMenu(request: StringValue): Promise<Empty> {
    return service<Int64Value, Empty>({
      url: `${BASE_MENU_URL}/${request.value}`,
      method: "delete",
    });
  }
  /** 设置状态 */
  SetBaseMenuStatus(request: SetStatusRequest): Promise<Empty> {
    return service<SetStatusRequest, Empty>({
      url: `${BASE_MENU_URL}/${request.id}/status`,
      method: "put",
      data: request,
    });
  }
}

export const defBaseMenuService = new BaseMenuServiceImpl();
