import service from "@/utils/request";
import {
  type CreateShopHotItemRequest,
  type CreateShopHotRequest,
  type DeleteShopHotItemRequest,
  type DeleteShopHotRequest,
  type GetShopHotItemRequest,
  type GetShopHotRequest,
  type PageShopHotItemRequest,
  type PageShopHotItemResponse,
  type PageShopHotRequest,
  type PageShopHotResponse,
  type SetShopHotItemStatusRequest,
  type SetShopHotStatusRequest,
  type ShopHotForm,
  type ShopHotItemForm,
  type ShopHotService,
  type UpdateShopHotItemRequest,
  type UpdateShopHotRequest
} from "@/rpc/shop/admin/v1/shop_hot";
import type { Empty } from "@/rpc/google/protobuf/empty";

const SHOP_HOT_URL = "/v1/admin/shop/hot";

const SHOP_HOT_ITEM_URL = "/v1/admin/shop/hot-item";

/** 商城热门推荐服务 */
export class ShopHotServiceImpl implements ShopHotService {
  /** 查询商城热门推荐列表 */
  PageShopHot(request: PageShopHotRequest): Promise<PageShopHotResponse> {
    return service<PageShopHotRequest, PageShopHotResponse>({
      url: `${SHOP_HOT_URL}`,
      method: "get",
      params: request
    });
  }
  /** 查询商城热门推荐 */
  GetShopHot(request: GetShopHotRequest): Promise<ShopHotForm> {
    return service<GetShopHotRequest, ShopHotForm>({
      url: `${SHOP_HOT_URL}/${request.id}`,
      method: "get"
    });
  }
  /** 创建商城热门推荐 */
  CreateShopHot(request: CreateShopHotRequest): Promise<Empty> {
    return service<ShopHotForm | undefined, Empty>({
      url: `${SHOP_HOT_URL}`,
      method: "post",
      data: request.shop_hot
    });
  }
  /** 更新商城热门推荐 */
  UpdateShopHot(request: UpdateShopHotRequest): Promise<Empty> {
    return service<ShopHotForm | undefined, Empty>({
      url: `${SHOP_HOT_URL}/${request.id}`,
      method: "put",
      data: request.shop_hot
    });
  }
  /** 删除商城热门推荐 */
  DeleteShopHot(request: DeleteShopHotRequest): Promise<Empty> {
    return service<DeleteShopHotRequest, Empty>({
      url: `${SHOP_HOT_URL}/${request.ids}`,
      method: "delete"
    });
  }
  /** 设置状态 */
  SetShopHotStatus(request: SetShopHotStatusRequest): Promise<Empty> {
    return service<SetShopHotStatusRequest, Empty>({
      url: `${SHOP_HOT_URL}/${request.id}/status`,
      method: "put",
      data: request
    });
  }
  /** 查询商城热门推荐属性列表 */
  PageShopHotItem(request: PageShopHotItemRequest): Promise<PageShopHotItemResponse> {
    return service<PageShopHotItemRequest, PageShopHotItemResponse>({
      url: `${SHOP_HOT_ITEM_URL}`,
      method: "get",
      params: request
    });
  }
  /** 查询商城热门推荐属性 */
  GetShopHotItem(request: GetShopHotItemRequest): Promise<ShopHotItemForm> {
    return service<GetShopHotItemRequest, ShopHotItemForm>({
      url: `${SHOP_HOT_ITEM_URL}/${request.id}`,
      method: "get"
    });
  }
  /** 创建商城热门推荐属性 */
  CreateShopHotItem(request: CreateShopHotItemRequest): Promise<Empty> {
    return service<ShopHotItemForm | undefined, Empty>({
      url: `${SHOP_HOT_ITEM_URL}`,
      method: "post",
      data: request.shop_hot_item
    });
  }
  /** 更新商城热门推荐属性 */
  UpdateShopHotItem(request: UpdateShopHotItemRequest): Promise<Empty> {
    return service<ShopHotItemForm | undefined, Empty>({
      url: `${SHOP_HOT_ITEM_URL}/${request.id}`,
      method: "put",
      data: request.shop_hot_item
    });
  }
  /** 删除商城热门推荐属性 */
  DeleteShopHotItem(request: DeleteShopHotItemRequest): Promise<Empty> {
    return service<DeleteShopHotItemRequest, Empty>({
      url: `${SHOP_HOT_ITEM_URL}/${request.ids}`,
      method: "delete"
    });
  }
  /** 设置状态 */
  SetShopHotItemStatus(request: SetShopHotItemStatusRequest): Promise<Empty> {
    return service<SetShopHotItemStatusRequest, Empty>({
      url: `${SHOP_HOT_ITEM_URL}/${request.id}/status`,
      method: "put",
      data: request
    });
  }
}

export const defShopHotService = new ShopHotServiceImpl();
