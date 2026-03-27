import service from "@/utils/request";
import {
  type ShopBannerForm,
  type ShopBannerService,
  type PageShopBannerRequest,
  type PageShopBannerResponse,
} from "@/rpc/admin/shop_banner";
import type { Empty } from "@/rpc/google/protobuf/empty";
import type { Int64Value, StringValue } from "@/rpc/google/protobuf/wrappers";
import type { SetStatusRequest } from "@/rpc/common/common";

const SHOP_BANNER_URL = "/admin/shop/banner";

/** 轮播图服务 */
export class ShopBannerServiceImpl implements ShopBannerService {
  /** 查询轮播图分页列表 */
  PageShopBanner(request: PageShopBannerRequest): Promise<PageShopBannerResponse> {
    return service<PageShopBannerRequest, PageShopBannerResponse>({
      url: `${SHOP_BANNER_URL}`,
      method: "get",
      params: request,
    });
  }
  /** 查询轮播图 */
  GetShopBanner(request: Int64Value): Promise<ShopBannerForm> {
    return service<Int64Value, ShopBannerForm>({
      url: `${SHOP_BANNER_URL}/${request.value}`,
      method: "get",
    });
  }
  /** 创建轮播图 */
  CreateShopBanner(request: ShopBannerForm): Promise<Empty> {
    return service<ShopBannerForm, Empty>({
      url: `${SHOP_BANNER_URL}`,
      method: "post",
      data: request,
    });
  }
  /** 更新轮播图 */
  UpdateShopBanner(request: ShopBannerForm): Promise<Empty> {
    return service<ShopBannerForm, Empty>({
      url: `${SHOP_BANNER_URL}/${request.id}`,
      method: "put",
      data: request,
    });
  }
  /** 删除轮播图 */
  DeleteShopBanner(request: StringValue): Promise<Empty> {
    return service<StringValue, Empty>({
      url: `${SHOP_BANNER_URL}/${request.value}`,
      method: "delete",
    });
  }
  /** 设置状态 */
  SetShopBannerStatus(request: SetStatusRequest): Promise<Empty> {
    return service<SetStatusRequest, Empty>({
      url: `${SHOP_BANNER_URL}/${request.id}/status`,
      method: "put",
      data: request,
    });
  }
}

export const defShopBannerService = new ShopBannerServiceImpl();
