import service from "@/utils/request";
import {
  type ShopHotForm,
  type ShopHotItemForm,
  type ShopHotService,
  type PageShopHotItemRequest,
  type PageShopHotItemResponse,
  type PageShopHotRequest,
  type PageShopHotResponse,
} from "@/rpc/admin/shop_hot";
import type { Empty } from "@/rpc/google/protobuf/empty";
import type { Int64Value, StringValue } from "@/rpc/google/protobuf/wrappers";
import type { SetStatusRequest } from "@/rpc/common/common";

const SHOP_HOT_URL = "/admin/shop/hot";

const SHOP_HOT_ITEM_URL = "/admin/shop/hot-item";

/** 商城热门推荐服务 */
export class ShopHotServiceImpl implements ShopHotService {
  /** 查询商城热门推荐分页列表 */
  PageShopHot(request: PageShopHotRequest): Promise<PageShopHotResponse> {
    return service<PageShopHotRequest, PageShopHotResponse>({
      url: `${SHOP_HOT_URL}`,
      method: "get",
      params: request,
    });
  }
  /** 查询商城热门推荐 */
  GetShopHot(request: Int64Value): Promise<ShopHotForm> {
    return service<Int64Value, ShopHotForm>({
      url: `${SHOP_HOT_URL}/${request.value}`,
      method: "get",
    });
  }
  /** 创建商城热门推荐 */
  CreateShopHot(request: ShopHotForm): Promise<Empty> {
    return service<ShopHotForm, Empty>({
      url: `${SHOP_HOT_URL}`,
      method: "post",
      data: request,
    });
  }
  /** 更新商城热门推荐 */
  UpdateShopHot(request: ShopHotForm): Promise<Empty> {
    return service<ShopHotForm, Empty>({
      url: `${SHOP_HOT_URL}/${request.id}`,
      method: "put",
      data: request,
    });
  }
  /** 删除商城热门推荐 */
  DeleteShopHot(request: StringValue): Promise<Empty> {
    return service<Int64Value, Empty>({
      url: `${SHOP_HOT_URL}/${request.value}`,
      method: "delete",
    });
  }
  /** 设置状态 */
  SetShopHotStatus(request: SetStatusRequest): Promise<Empty> {
    return service<SetStatusRequest, Empty>({
      url: `${SHOP_HOT_URL}/${request.id}/status`,
      method: "put",
      data: request,
    });
  }
  /** 查询商城热门推荐属性分页列表 */
  PageShopHotItem(request: PageShopHotItemRequest): Promise<PageShopHotItemResponse> {
    return service<PageShopHotItemRequest, PageShopHotItemResponse>({
      url: `${SHOP_HOT_ITEM_URL}`,
      method: "get",
      params: request,
    });
  }
  /** 查询商城热门推荐属性 */
  GetShopHotItem(request: Int64Value): Promise<ShopHotItemForm> {
    return service<Int64Value, ShopHotItemForm>({
      url: `${SHOP_HOT_ITEM_URL}/${request.value}`,
      method: "get",
    });
  }
  /** 创建商城热门推荐属性 */
  CreateShopHotItem(request: ShopHotItemForm): Promise<Empty> {
    return service<ShopHotItemForm, Empty>({
      url: `${SHOP_HOT_ITEM_URL}`,
      method: "post",
      data: request,
    });
  }
  /** 更新商城热门推荐属性 */
  UpdateShopHotItem(request: ShopHotItemForm): Promise<Empty> {
    return service<ShopHotItemForm, Empty>({
      url: `${SHOP_HOT_ITEM_URL}/${request.id}`,
      method: "put",
      data: request,
    });
  }
  /** 删除商城热门推荐属性 */
  DeleteShopHotItem(request: StringValue): Promise<Empty> {
    return service<Int64Value, Empty>({
      url: `${SHOP_HOT_ITEM_URL}/${request.value}`,
      method: "delete",
    });
  }
  /** 设置状态 */
  SetShopHotItemStatus(request: SetStatusRequest): Promise<Empty> {
    return service<SetStatusRequest, Empty>({
      url: `${SHOP_HOT_ITEM_URL}/${request.id}/status`,
      method: "put",
      data: request,
    });
  }
}

export const defShopHotService = new ShopHotServiceImpl();
