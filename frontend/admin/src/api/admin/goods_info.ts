import service from "@/utils/request";
import {
  type GoodsInfoForm,
  type GoodsInfoService,
  type PageGoodsInfoRequest,
  type PageGoodsInfoResponse,
  type OptionGoodsInfoRequest,
  type OptionGoodsInfoResponse
} from "@/rpc/admin/goods_info";
import type { Empty } from "@/rpc/google/protobuf/empty";
import type { Int64Value, StringValue } from "@/rpc/google/protobuf/wrappers";
import type { SetStatusRequest } from "@/rpc/common/common";

const GOODS_URL = "/admin/goods/info";

/** Admin商品服务 */
export class GoodsInfoServiceImpl implements GoodsInfoService {
  /** 查询商品下拉选择 */
  OptionGoodsInfo(request: OptionGoodsInfoRequest): Promise<OptionGoodsInfoResponse> {
    return service<OptionGoodsInfoRequest, OptionGoodsInfoResponse>({
      url: `${GOODS_URL}/option`,
      method: "get",
      params: request
    });
  }
  /** 查询商品分页列表 */
  PageGoodsInfo(request: PageGoodsInfoRequest): Promise<PageGoodsInfoResponse> {
    return service<PageGoodsInfoRequest, PageGoodsInfoResponse>({
      url: `${GOODS_URL}`,
      method: "get",
      params: request
    });
  }
  /** 查询商品 */
  GetGoodsInfo(request: Int64Value): Promise<GoodsInfoForm> {
    return service<Int64Value, GoodsInfoForm>({
      url: `${GOODS_URL}/${request.value}`,
      method: "get"
    });
  }
  /** 创建商品 */
  CreateGoodsInfo(request: GoodsInfoForm): Promise<Empty> {
    return service<GoodsInfoForm, Empty>({
      url: `${GOODS_URL}`,
      method: "post",
      data: request
    });
  }
  /** 更新商品 */
  UpdateGoodsInfo(request: GoodsInfoForm): Promise<Empty> {
    return service<GoodsInfoForm, Empty>({
      url: `${GOODS_URL}/${request.id}`,
      method: "put",
      data: request
    });
  }
  /** 删除商品 */
  DeleteGoodsInfo(request: StringValue): Promise<Empty> {
    return service<StringValue, Empty>({
      url: `${GOODS_URL}/${request.value}`,
      method: "delete"
    });
  }
  /** 设置状态 */
  SetGoodsInfoStatus(request: SetStatusRequest): Promise<Empty> {
    return service<SetStatusRequest, Empty>({
      url: `${GOODS_URL}/${request.id}/status`,
      method: "put",
      data: request
    });
  }
}

export const defGoodsInfoService = new GoodsInfoServiceImpl();
