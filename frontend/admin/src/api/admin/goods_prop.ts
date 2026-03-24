import service from "@/utils/request";
import {
  type GoodsProp,
  type GoodsPropService,
  type PageGoodsPropRequest,
  type PageGoodsPropResponse,
} from "@/rpc/admin/goods_prop";
import type { Empty } from "@/rpc/google/protobuf/empty";
import type { Int64Value, StringValue } from "@/rpc/google/protobuf/wrappers";

const GOODS_PROP_URL = "/admin/goods/prop";

/** Admin属性服务 */
export class GoodsPropServiceImpl implements GoodsPropService {
  /** 查询属性分页列表 */
  PageGoodsProp(request: PageGoodsPropRequest): Promise<PageGoodsPropResponse> {
    return service<PageGoodsPropRequest, PageGoodsPropResponse>({
      url: `${GOODS_PROP_URL}`,
      method: "get",
      params: request,
    });
  }
  /** 查询属性 */
  GetGoodsProp(request: Int64Value): Promise<GoodsProp> {
    return service<Int64Value, GoodsProp>({
      url: `${GOODS_PROP_URL}/${request.value}`,
      method: "get",
    });
  }
  /** 创建属性 */
  CreateGoodsProp(request: GoodsProp): Promise<Empty> {
    return service<GoodsProp, Empty>({
      url: `${GOODS_PROP_URL}`,
      method: "post",
      data: request,
    });
  }
  /** 更新属性 */
  UpdateGoodsProp(request: GoodsProp): Promise<Empty> {
    return service<GoodsProp, Empty>({
      url: `${GOODS_PROP_URL}/${request.id}`,
      method: "put",
      data: request,
    });
  }
  /** 删除属性 */
  DeleteGoodsProp(request: StringValue): Promise<Empty> {
    return service<StringValue, Empty>({
      url: `${GOODS_PROP_URL}/${request.value}`,
      method: "delete",
    });
  }
}

export const defGoodsPropService = new GoodsPropServiceImpl();
