import service from "@/utils/request";
import { type BaseApiService, type ListBaseApisRequest, type ListBaseApisResponse } from "@/rpc/admin/v1/base_api";

const BASE_API_URL = "/v1/admin/base/api";

/** AdminAPI服务 */
export class BaseApiServiceImpl implements BaseApiService {
  /** 查询API列表 */
  ListBaseApis(request: ListBaseApisRequest): Promise<ListBaseApisResponse> {
    return service<ListBaseApisRequest, ListBaseApisResponse>({
      url: `${BASE_API_URL}`,
      method: "get",
      params: request
    });
  }
}

export const defBaseApiService = new BaseApiServiceImpl();
