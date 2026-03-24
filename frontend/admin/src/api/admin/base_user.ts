import service from "@/utils/request";
import {
  type BaseUserForm,
  type BaseUserService,
  type PageBaseUserRequest,
  type PageBaseUserResponse,
  type ResetBaseUserPwdRequest,
  type OptionBaseUserRequest,
} from "@/rpc/admin/base_user";
import type { Empty } from "@/rpc/google/protobuf/empty";
import type { Int64Value, StringValue } from "@/rpc/google/protobuf/wrappers";
import type { SelectOptionResponse, SetStatusRequest } from "@/rpc/common/common";

const BASE_USER_URL = "/admin/base/user";

/** Admin用户服务 */
export class BaseUserServiceImpl implements BaseUserService {
  /** 查询用户下拉选择 */
  OptionBaseUser(request: OptionBaseUserRequest): Promise<SelectOptionResponse> {
    return service<OptionBaseUserRequest, SelectOptionResponse>({
      url: `${BASE_USER_URL}/option`,
      method: "get",
      params: request,
    });
  }
  /** 查询用户分页列表 */
  PageBaseUser(request: PageBaseUserRequest): Promise<PageBaseUserResponse> {
    return service<PageBaseUserRequest, PageBaseUserResponse>({
      url: `${BASE_USER_URL}`,
      method: "get",
      params: request,
    });
  }
  /** 查询用户 */
  GetBaseUser(request: Int64Value): Promise<BaseUserForm> {
    return service<Int64Value, BaseUserForm>({
      url: `${BASE_USER_URL}/${request.value}`,
      method: "get",
    });
  }
  /** 创建用户 */
  CreateBaseUser(request: BaseUserForm): Promise<Empty> {
    return service<BaseUserForm, StringValue>({
      url: `${BASE_USER_URL}`,
      method: "post",
      data: request,
    });
  }
  /** 更新用户 */
  UpdateBaseUser(request: BaseUserForm): Promise<Empty> {
    return service<BaseUserForm, Empty>({
      url: `${BASE_USER_URL}/${request.id}`,
      method: "put",
      data: request,
    });
  }
  /** 删除用户 */
  DeleteBaseUser(request: StringValue): Promise<Empty> {
    return service<StringValue, Empty>({
      url: `${BASE_USER_URL}/${request.value}`,
      method: "delete",
    });
  }
  /** 设置状态 */
  SetBaseUserStatus(request: SetStatusRequest): Promise<Empty> {
    return service<SetStatusRequest, Empty>({
      url: `${BASE_USER_URL}/${request.id}/status`,
      method: "put",
      data: request,
    });
  }
  /** 重置密码 */
  ResetBaseUserPwd(request: ResetBaseUserPwdRequest): Promise<Empty> {
    return service<StringValue, StringValue>({
      url: `${BASE_USER_URL}/` + request.id + "/pwd",
      method: "put",
      data: request,
    });
  }
}

export const defBaseUserService = new BaseUserServiceImpl();
