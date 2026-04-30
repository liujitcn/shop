import service from "@/utils/request";
import {
  type GetGoodsSkuRequest,
  type GoodsSku,
  type GoodsSkuService,
  type PageGoodsSkusRequest,
  type PageGoodsSkusResponse,
  type UpdateGoodsSkuRequest
} from "@/rpc/admin/v1/goods_sku";
import type { Empty } from "@/rpc/google/protobuf/empty";

const GOODS_SKU_URL = "/v1/admin/goods/sku";

/** Admin商品SKU服务 */
export class GoodsSkuServiceImpl implements GoodsSkuService {
  /** 查询sku列表 */
  PageGoodsSkus(request: PageGoodsSkusRequest): Promise<PageGoodsSkusResponse> {
    return service<PageGoodsSkusRequest, PageGoodsSkusResponse>({
      url: `${GOODS_SKU_URL}`,
      method: "get",
      params: request
    });
  }
  /** 查询sku */
  GetGoodsSku(request: GetGoodsSkuRequest): Promise<GoodsSku> {
    return service<GetGoodsSkuRequest, GoodsSku>({
      url: `${GOODS_SKU_URL}/${request.id}`,
      method: "get"
    });
  }
  /** 更新sku */
  UpdateGoodsSku(request: UpdateGoodsSkuRequest): Promise<Empty> {
    return service<GoodsSku | undefined, Empty>({
      url: `${GOODS_SKU_URL}/${request.id}`,
      method: "put",
      data: request.goods_sku
    });
  }
}

export const defGoodsSkuService = new GoodsSkuServiceImpl();
