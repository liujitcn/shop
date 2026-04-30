import service from "@/utils/request";
import {
  type UserStore,
  type UserStoreService,
  type PageUserStoresRequest,
  type PageUserStoresResponse,
  type GetUserStoreRequest,
  type AuditUserStoreRequest
} from "@/rpc/admin/v1/user_store";
import type { Empty } from "@/rpc/google/protobuf/empty";

const USER_STORE_URL = "/v1/admin/user/store";

/** Admin用户门店服务 */
export class UserStoreServiceImpl implements UserStoreService {
  /** 查询用户门店分页列表 */
  PageUserStores(request: PageUserStoresRequest): Promise<PageUserStoresResponse> {
    return service<PageUserStoresRequest, PageUserStoresResponse>({
      url: `${USER_STORE_URL}`,
      method: "get",
      params: request
    });
  }
  /** 查询用户门店 */
  GetUserStore(request: GetUserStoreRequest): Promise<UserStore> {
    return service<GetUserStoreRequest, UserStore>({
      url: `${USER_STORE_URL}/${request.id}`,
      method: "get"
    });
  }
  /** 门店认证 */
  AuditUserStore(request: AuditUserStoreRequest): Promise<Empty> {
    return service<AuditUserStoreRequest, Empty>({
      url: `${USER_STORE_URL}/${request.id}/audit`,
      method: "put",
      data: request
    });
  }
}

export const defUserStoreService = new UserStoreServiceImpl();
