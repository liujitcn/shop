import service from "@/utils/request";
import { type BaseApiService, type ListBaseApiResponse } from "@/rpc/admin/base_api";
import type { Empty } from "@/rpc/google/protobuf/empty";

const BASE_API_URL = "/admin/base/api";

/** AdminAPI服务 */
export class BaseApiServiceImpl implements BaseApiService {
  /** 查询API列表 */
  ListBaseApi(request: Empty): Promise<ListBaseApiResponse> {
    return service<Empty, ListBaseApiResponse>({
      url: `${BASE_API_URL}/list`,
      method: "get",
      params: request,
    });
  }
}

export const defBaseApiService = new BaseApiServiceImpl();
