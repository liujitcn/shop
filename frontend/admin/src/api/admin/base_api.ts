import service from "@/utils/request";
import {
  type BaseApi,
  type BaseApiService,
  type GetBaseApiRequest,
  type ListBaseApisRequest,
  type ListBaseApisResponse,
  type PageBaseApisRequest,
  type PageBaseApisResponse,
  type SetBaseApiMcpEnabledRequest
} from "@/rpc/admin/v1/base_api";
import type { Empty } from "@/rpc/google/protobuf/empty";

const BASE_API_URL = "/v1/admin/base/api";

/** AdminAPI服务 */
export class BaseApiServiceImpl implements BaseApiService {
  /** 查询API选项列表 */
  ListBaseApis(request: ListBaseApisRequest): Promise<ListBaseApisResponse> {
    return service<ListBaseApisRequest, ListBaseApisResponse>({
      url: `${BASE_API_URL}/option`,
      method: "get",
      params: request
    });
  }

  /** 查询API分页列表 */
  PageBaseApis(request: PageBaseApisRequest): Promise<PageBaseApisResponse> {
    return service<PageBaseApisRequest, PageBaseApisResponse>({
      url: `${BASE_API_URL}`,
      method: "get",
      params: request
    });
  }

  /** 查询API详情 */
  GetBaseApi(request: GetBaseApiRequest): Promise<BaseApi> {
    return service<GetBaseApiRequest, BaseApi>({
      url: `${BASE_API_URL}/${request.id}`,
      method: "get"
    });
  }

  /** 设置API是否暴露为MCP工具 */
  SetBaseApiMcpEnabled(request: SetBaseApiMcpEnabledRequest): Promise<Empty> {
    return service<SetBaseApiMcpEnabledRequest, Empty>({
      url: `${BASE_API_URL}/${request.id}/mcp-enabled`,
      method: "put",
      data: request
    });
  }
}

export const defBaseApiService = new BaseApiServiceImpl();
