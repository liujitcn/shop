import service from "@/utils/request";
import {
  type GoodsSpecService,
  type ListGoodsSpecRequest,
  type ListGoodsSpecResponse,
} from "@/rpc/admin/goods_spec";

const GOODS_SPEC_URL = "/admin/goods/spec";

/** Admin规格服务 */
export class GoodsSpecServiceImpl implements GoodsSpecService {
  /** 查询商品规格列表 */
  ListGoodsSpec(request: ListGoodsSpecRequest): Promise<ListGoodsSpecResponse> {
    return service<ListGoodsSpecRequest, ListGoodsSpecResponse>({
      url: `${GOODS_SPEC_URL}/list`,
      method: "get",
      params: request,
    });
  }
}

export const defGoodsSpecService = new GoodsSpecServiceImpl();
