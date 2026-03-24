import service from "@/utils/request";
import {
  type UserStore,
  type UserStoreService,
  type PageUserStoreRequest,
  type PageUserStoreResponse,
  type AuditUserStoreForm,
} from "@/rpc/admin/user_store";
import type { Int64Value } from "@/rpc/google/protobuf/wrappers";
import type { Empty } from "@/rpc/google/protobuf/empty";

const USER_STORE_URL = "/admin/user/store";

/** Admin用户门店服务 */
export class UserStoreServiceImpl implements UserStoreService {
  /** 查询用户门店分页列表 */
  PageUserStore(request: PageUserStoreRequest): Promise<PageUserStoreResponse> {
    return service<PageUserStoreRequest, PageUserStoreResponse>({
      url: `${USER_STORE_URL}`,
      method: "get",
      params: request,
    });
  }
  /** 查询用户门店 */
  GetUserStore(request: Int64Value): Promise<UserStore> {
    return service<Int64Value, UserStore>({
      url: `${USER_STORE_URL}/${request.value}`,
      method: "get",
    });
  }
  /** 门店认证 */
  AuditUserStore(request: AuditUserStoreForm): Promise<Empty> {
    return service<AuditUserStoreForm, Empty>({
      url: `${USER_STORE_URL}/${request.id}/audit`,
      method: "put",
      data: request,
    });
  }
}

export const defUserStoreService = new UserStoreServiceImpl();
