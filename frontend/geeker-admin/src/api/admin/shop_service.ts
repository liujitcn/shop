import service from "@/utils/request";
import {
  type ShopServiceForm,
  type ShopServiceService,
  type PageShopServiceRequest,
  type PageShopServiceResponse,
} from "@/rpc/admin/shop_service";
import type { Empty } from "@/rpc/google/protobuf/empty";
import type { Int64Value, StringValue } from "@/rpc/google/protobuf/wrappers";
import type { SetStatusRequest } from "@/rpc/common/common";

const SHOP_SERVICE_URL = "/admin/shop/service";

/** 商城服务 */
export class ShopServiceServiceImpl implements ShopServiceService {
  /** 查询商城服务分页列表 */
  PageShopService(request: PageShopServiceRequest): Promise<PageShopServiceResponse> {
    return service<PageShopServiceRequest, PageShopServiceResponse>({
      url: `${SHOP_SERVICE_URL}`,
      method: "get",
      params: request,
    });
  }
  /** 查询商城服务 */
  GetShopService(request: Int64Value): Promise<ShopServiceForm> {
    return service<Int64Value, ShopServiceForm>({
      url: `${SHOP_SERVICE_URL}/${request.value}`,
      method: "get",
    });
  }
  /** 创建商城服务 */
  CreateShopService(request: ShopServiceForm): Promise<Empty> {
    return service<ShopServiceForm, Empty>({
      url: `${SHOP_SERVICE_URL}`,
      method: "post",
      data: request,
    });
  }
  /** 更新商城服务 */
  UpdateShopService(request: ShopServiceForm): Promise<Empty> {
    return service<ShopServiceForm, Empty>({
      url: `${SHOP_SERVICE_URL}/${request.id}`,
      method: "put",
      data: request,
    });
  }
  /** 删除商城服务 */
  DeleteShopService(request: StringValue): Promise<Empty> {
    return service<Int64Value, Empty>({
      url: `${SHOP_SERVICE_URL}/${request.value}`,
      method: "delete",
    });
  }
  /** 设置状态 */
  SetShopServiceStatus(request: SetStatusRequest): Promise<Empty> {
    return service<SetStatusRequest, Empty>({
      url: `${SHOP_SERVICE_URL}/${request.id}/status`,
      method: "put",
      data: request,
    });
  }
}

export const defShopServiceService = new ShopServiceServiceImpl(); 