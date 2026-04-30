import service from "@/utils/request";
import {
  type CreateShopServiceRequest,
  type DeleteShopServiceRequest,
  type GetShopServiceRequest,
  type PageShopServicesRequest,
  type PageShopServicesResponse,
  type SetShopServiceStatusRequest,
  type ShopServiceForm,
  type ShopServiceService,
  type UpdateShopServiceRequest
} from "@/rpc/admin/v1/shop_service";
import type { Empty } from "@/rpc/google/protobuf/empty";

const SHOP_SERVICE_URL = "/v1/admin/shop/service";

/** 商城服务 */
export class ShopServiceServiceImpl implements ShopServiceService {
  /** 查询商城服务列表 */
  PageShopServices(request: PageShopServicesRequest): Promise<PageShopServicesResponse> {
    return service<PageShopServicesRequest, PageShopServicesResponse>({
      url: `${SHOP_SERVICE_URL}`,
      method: "get",
      params: request
    });
  }
  /** 查询商城服务 */
  GetShopService(request: GetShopServiceRequest): Promise<ShopServiceForm> {
    return service<GetShopServiceRequest, ShopServiceForm>({
      url: `${SHOP_SERVICE_URL}/${request.id}`,
      method: "get"
    });
  }
  /** 创建商城服务 */
  CreateShopService(request: CreateShopServiceRequest): Promise<Empty> {
    return service<ShopServiceForm | undefined, Empty>({
      url: `${SHOP_SERVICE_URL}`,
      method: "post",
      data: request.shop_service
    });
  }
  /** 更新商城服务 */
  UpdateShopService(request: UpdateShopServiceRequest): Promise<Empty> {
    return service<ShopServiceForm | undefined, Empty>({
      url: `${SHOP_SERVICE_URL}/${request.id}`,
      method: "put",
      data: request.shop_service
    });
  }
  /** 删除商城服务 */
  DeleteShopService(request: DeleteShopServiceRequest): Promise<Empty> {
    return service<DeleteShopServiceRequest, Empty>({
      url: `${SHOP_SERVICE_URL}/${request.ids}`,
      method: "delete"
    });
  }
  /** 设置状态 */
  SetShopServiceStatus(request: SetShopServiceStatusRequest): Promise<Empty> {
    return service<SetShopServiceStatusRequest, Empty>({
      url: `${SHOP_SERVICE_URL}/${request.id}/status`,
      method: "put",
      data: request
    });
  }
}

export const defShopServiceService = new ShopServiceServiceImpl();
