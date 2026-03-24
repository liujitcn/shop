import service from "@/utils/request";
import {
  type BaseConfigForm,
  type BaseConfigService,
  type PageBaseConfigRequest,
  type PageBaseConfigResponse,
} from "@/rpc/admin/base_config";
import type { Empty } from "@/rpc/google/protobuf/empty";
import type { Int64Value, StringValue } from "@/rpc/google/protobuf/wrappers";
import type { SetStatusRequest } from "@/rpc/common/common";

const BASE_CONFIG_URL = "/admin/base/config";

/** Admin系统配置服务 */
export class BaseConfigServiceImpl implements BaseConfigService {
  /** 刷新缓存 */
  RefreshBaseConfig(request: Empty): Promise<Empty> {
    return service<Empty, Empty>({
      url: `${BASE_CONFIG_URL}/refresh`,
      method: "post",
      data: request,
    });
  }
  /** 查询系统配置分页列表 */
  PageBaseConfig(request: PageBaseConfigRequest): Promise<PageBaseConfigResponse> {
    return service<PageBaseConfigRequest, PageBaseConfigResponse>({
      url: `${BASE_CONFIG_URL}`,
      method: "get",
      params: request,
    });
  }
  /** 查询系统配置 */
  GetBaseConfig(request: Int64Value): Promise<BaseConfigForm> {
    return service<Int64Value, BaseConfigForm>({
      url: `${BASE_CONFIG_URL}/${request.value}`,
      method: "get",
    });
  }
  /** 创建系统配置 */
  CreateBaseConfig(request: BaseConfigForm): Promise<Empty> {
    return service<BaseConfigForm, Empty>({
      url: `${BASE_CONFIG_URL}`,
      method: "post",
      data: request,
    });
  }
  /** 更新系统配置 */
  UpdateBaseConfig(request: BaseConfigForm): Promise<Empty> {
    return service<BaseConfigForm, Empty>({
      url: `${BASE_CONFIG_URL}/${request.id}`,
      method: "put",
      data: request,
    });
  }
  /** 删除系统配置 */
  DeleteBaseConfig(request: StringValue): Promise<Empty> {
    return service<StringValue, Empty>({
      url: `${BASE_CONFIG_URL}/${request.value}`,
      method: "delete",
    });
  }
  /** 设置状态 */
  SetBaseConfigStatus(request: SetStatusRequest): Promise<Empty> {
    return service<SetStatusRequest, Empty>({
      url: `${BASE_CONFIG_URL}/${request.id}/status`,
      method: "put",
      data: request,
    });
  }
}

export const defBaseConfigService = new BaseConfigServiceImpl();
