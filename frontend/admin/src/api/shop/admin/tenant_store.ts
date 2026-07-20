import service from "@/utils/request";
import {
  type AuditTenantStoreRequest,
  type CreateTenantStoreRequest,
  type DeleteTenantStoreRequest,
  type GetTenantStoreRequest,
  type OptionTenantStoreRequest,
  type OptionTenantStoreResponse,
  type PageTenantStoreRequest,
  type PageTenantStoreResponse,
  type TenantStoreForm,
  type TenantStoreService,
  type TreeTenantStoreRequest,
  type TreeTenantStoreResponse,
  type UpdateTenantStoreRequest
} from "@/rpc/shop/admin/v1/tenant_store";
import type { Empty } from "@/rpc/google/protobuf/empty";

const TENANT_STORE_URL = "/v1/admin/tenant/store";

/** 租户门店服务 */
export class TenantStoreServiceImpl implements TenantStoreService {
  /** 查询租户门店下拉选项 */
  OptionTenantStore(request: OptionTenantStoreRequest): Promise<OptionTenantStoreResponse> {
    return service<OptionTenantStoreRequest, OptionTenantStoreResponse>({
      url: `${TENANT_STORE_URL}/option`,
      method: "get",
      params: request
    });
  }

  /** 查询租户门店树形选项 */
  TreeTenantStore(request: TreeTenantStoreRequest): Promise<TreeTenantStoreResponse> {
    return service<TreeTenantStoreRequest, TreeTenantStoreResponse>({
      url: `${TENANT_STORE_URL}/tree`,
      method: "get",
      params: request
    });
  }

  /** 查询租户门店列表 */
  PageTenantStore(request: PageTenantStoreRequest): Promise<PageTenantStoreResponse> {
    return service<PageTenantStoreRequest, PageTenantStoreResponse>({
      url: `${TENANT_STORE_URL}`,
      method: "get",
      params: request
    });
  }

  /** 查询租户门店 */
  GetTenantStore(request: GetTenantStoreRequest): Promise<TenantStoreForm> {
    return service<GetTenantStoreRequest, TenantStoreForm>({
      url: `${TENANT_STORE_URL}/${request.id}`,
      method: "get"
    });
  }

  /** 创建租户门店 */
  CreateTenantStore(request: CreateTenantStoreRequest): Promise<Empty> {
    return service<TenantStoreForm | undefined, Empty>({
      url: `${TENANT_STORE_URL}`,
      method: "post",
      data: request.tenant_store
    });
  }

  /** 更新租户门店 */
  UpdateTenantStore(request: UpdateTenantStoreRequest): Promise<Empty> {
    return service<TenantStoreForm | undefined, Empty>({
      url: `${TENANT_STORE_URL}/${request.id}`,
      method: "put",
      data: request.tenant_store
    });
  }

  /** 删除租户门店 */
  DeleteTenantStore(request: DeleteTenantStoreRequest): Promise<Empty> {
    return service<DeleteTenantStoreRequest, Empty>({
      url: `${TENANT_STORE_URL}/${request.ids}`,
      method: "delete"
    });
  }

  /** 审核租户门店 */
  AuditTenantStore(request: AuditTenantStoreRequest): Promise<Empty> {
    return service<AuditTenantStoreRequest, Empty>({
      url: `${TENANT_STORE_URL}/${request.id}/audit`,
      method: "put",
      data: request
    });
  }
}

export const defTenantStoreService = new TenantStoreServiceImpl();
