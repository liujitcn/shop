import service from "@/utils/request";
import {
  type CreateShopBannerRequest,
  type DeleteShopBannerRequest,
  type GetShopBannerRequest,
  type PageShopBannersRequest,
  type PageShopBannersResponse,
  type SetShopBannerStatusRequest,
  type ShopBannerForm,
  type ShopBannerService,
  type UpdateShopBannerRequest
} from "@/rpc/admin/v1/shop_banner";
import type { Empty } from "@/rpc/google/protobuf/empty";

const SHOP_BANNER_URL = "/v1/admin/shop/banner";

/** 轮播图服务 */
export class ShopBannerServiceImpl implements ShopBannerService {
  /** 查询轮播图列表 */
  PageShopBanners(request: PageShopBannersRequest): Promise<PageShopBannersResponse> {
    return service<PageShopBannersRequest, PageShopBannersResponse>({
      url: `${SHOP_BANNER_URL}`,
      method: "get",
      params: request
    });
  }
  /** 查询轮播图 */
  GetShopBanner(request: GetShopBannerRequest): Promise<ShopBannerForm> {
    return service<GetShopBannerRequest, ShopBannerForm>({
      url: `${SHOP_BANNER_URL}/${request.id}`,
      method: "get"
    });
  }
  /** 创建轮播图 */
  CreateShopBanner(request: CreateShopBannerRequest): Promise<Empty> {
    return service<ShopBannerForm | undefined, Empty>({
      url: `${SHOP_BANNER_URL}`,
      method: "post",
      data: request.shop_banner
    });
  }
  /** 更新轮播图 */
  UpdateShopBanner(request: UpdateShopBannerRequest): Promise<Empty> {
    return service<ShopBannerForm | undefined, Empty>({
      url: `${SHOP_BANNER_URL}/${request.id}`,
      method: "put",
      data: request.shop_banner
    });
  }
  /** 删除轮播图 */
  DeleteShopBanner(request: DeleteShopBannerRequest): Promise<Empty> {
    return service<DeleteShopBannerRequest, Empty>({
      url: `${SHOP_BANNER_URL}/${request.ids}`,
      method: "delete"
    });
  }
  /** 设置状态 */
  SetShopBannerStatus(request: SetShopBannerStatusRequest): Promise<Empty> {
    return service<SetShopBannerStatusRequest, Empty>({
      url: `${SHOP_BANNER_URL}/${request.id}/status`,
      method: "put",
      data: request
    });
  }
}

export const defShopBannerService = new ShopBannerServiceImpl();
