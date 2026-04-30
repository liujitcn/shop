import service from "@/utils/request";
import {
  type CreateGoodsPropRequest,
  type DeleteGoodsPropRequest,
  type GetGoodsPropRequest,
  type GoodsProp,
  type GoodsPropService,
  type PageGoodsPropsRequest,
  type PageGoodsPropsResponse,
  type UpdateGoodsPropRequest
} from "@/rpc/admin/v1/goods_prop";
import type { Empty } from "@/rpc/google/protobuf/empty";

const GOODS_PROP_URL = "/v1/admin/goods/prop";

/** Admin属性服务 */
export class GoodsPropServiceImpl implements GoodsPropService {
  /** 查询属性列表 */
  PageGoodsProps(request: PageGoodsPropsRequest): Promise<PageGoodsPropsResponse> {
    return service<PageGoodsPropsRequest, PageGoodsPropsResponse>({
      url: `${GOODS_PROP_URL}`,
      method: "get",
      params: request
    });
  }
  /** 查询属性 */
  GetGoodsProp(request: GetGoodsPropRequest): Promise<GoodsProp> {
    return service<GetGoodsPropRequest, GoodsProp>({
      url: `${GOODS_PROP_URL}/${request.id}`,
      method: "get"
    });
  }
  /** 创建属性 */
  CreateGoodsProp(request: CreateGoodsPropRequest): Promise<Empty> {
    return service<GoodsProp | undefined, Empty>({
      url: `${GOODS_PROP_URL}`,
      method: "post",
      data: request.goods_prop
    });
  }
  /** 更新属性 */
  UpdateGoodsProp(request: UpdateGoodsPropRequest): Promise<Empty> {
    return service<GoodsProp | undefined, Empty>({
      url: `${GOODS_PROP_URL}/${request.id}`,
      method: "put",
      data: request.goods_prop
    });
  }
  /** 删除属性 */
  DeleteGoodsProp(request: DeleteGoodsPropRequest): Promise<Empty> {
    return service<DeleteGoodsPropRequest, Empty>({
      url: `${GOODS_PROP_URL}/${request.ids}`,
      method: "delete"
    });
  }
}

export const defGoodsPropService = new GoodsPropServiceImpl();
