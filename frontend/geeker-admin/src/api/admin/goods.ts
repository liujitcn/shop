import service from "@/utils/request";
import {
  type GoodsForm,
  type GoodsService,
  type PageGoodsRequest,
  type PageGoodsResponse,
  type ListGoodsRequest,
  type ListGoodsResponse,
} from "@/rpc/admin/goods";
import type { Empty } from "@/rpc/google/protobuf/empty";
import type { Int64Value, StringValue } from "@/rpc/google/protobuf/wrappers";
import type { SetStatusRequest } from "@/rpc/common/common";

const GOODS_URL = "/admin/goods/info";

/** Admin商品服务 */
export class GoodsServiceImpl implements GoodsService {
  /** 查询商品列表 */
  ListGoods(request: ListGoodsRequest): Promise<ListGoodsResponse> {
    return service<ListGoodsRequest, ListGoodsResponse>({
      url: `${GOODS_URL}/list`,
      method: "get",
      params: request,
    });
  }
  /** 查询商品分页列表 */
  PageGoods(request: PageGoodsRequest): Promise<PageGoodsResponse> {
    return service<PageGoodsRequest, PageGoodsResponse>({
      url: `${GOODS_URL}`,
      method: "get",
      params: request,
    });
  }
  /** 查询商品 */
  GetGoods(request: Int64Value): Promise<GoodsForm> {
    return service<Int64Value, GoodsForm>({
      url: `${GOODS_URL}/${request.value}`,
      method: "get",
    });
  }
  /** 创建商品 */
  CreateGoods(request: GoodsForm): Promise<Empty> {
    return service<GoodsForm, Empty>({
      url: `${GOODS_URL}`,
      method: "post",
      data: request,
    });
  }
  /** 更新商品 */
  UpdateGoods(request: GoodsForm): Promise<Empty> {
    return service<GoodsForm, Empty>({
      url: `${GOODS_URL}/${request.id}`,
      method: "put",
      data: request,
    });
  }
  /** 删除商品 */
  DeleteGoods(request: StringValue): Promise<Empty> {
    return service<StringValue, Empty>({
      url: `${GOODS_URL}/${request.value}`,
      method: "delete",
    });
  }
  /** 设置状态 */
  SetGoodsStatus(request: SetStatusRequest): Promise<Empty> {
    return service<SetStatusRequest, Empty>({
      url: `${GOODS_URL}/${request.id}/status`,
      method: "put",
      data: request,
    });
  }
}

export const defGoodsService = new GoodsServiceImpl();
