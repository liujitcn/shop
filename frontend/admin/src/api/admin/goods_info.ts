import service from "@/utils/request";
import {
  type CreateGoodsInfoRequest,
  type DeleteGoodsInfoRequest,
  type GetGoodsInfoRequest,
  type GoodsInfoForm,
  type GoodsInfoService,
  type PageGoodsInfosRequest,
  type PageGoodsInfosResponse,
  type OptionGoodsInfosRequest,
  type OptionGoodsInfosResponse,
  type SetGoodsInfoStatusRequest,
  type UpdateGoodsInfoRequest
} from "@/rpc/admin/v1/goods_info";
import type { Empty } from "@/rpc/google/protobuf/empty";

const GOODS_URL = "/v1/admin/goods/info";

/** Admin商品服务 */
export class GoodsInfoServiceImpl implements GoodsInfoService {
  /** 查询商品下拉选择 */
  OptionGoodsInfos(request: OptionGoodsInfosRequest): Promise<OptionGoodsInfosResponse> {
    return service<OptionGoodsInfosRequest, OptionGoodsInfosResponse>({
      url: `${GOODS_URL}/option`,
      method: "get",
      params: request
    });
  }
  /** 查询商品列表 */
  PageGoodsInfos(request: PageGoodsInfosRequest): Promise<PageGoodsInfosResponse> {
    return service<PageGoodsInfosRequest, PageGoodsInfosResponse>({
      url: `${GOODS_URL}`,
      method: "get",
      params: request
    });
  }
  /** 查询商品 */
  GetGoodsInfo(request: GetGoodsInfoRequest): Promise<GoodsInfoForm> {
    return service<GetGoodsInfoRequest, GoodsInfoForm>({
      url: `${GOODS_URL}/${request.id}`,
      method: "get"
    });
  }
  /** 创建商品 */
  CreateGoodsInfo(request: CreateGoodsInfoRequest): Promise<Empty> {
    return service<GoodsInfoForm | undefined, Empty>({
      url: `${GOODS_URL}`,
      method: "post",
      data: request.goods_info
    });
  }
  /** 更新商品 */
  UpdateGoodsInfo(request: UpdateGoodsInfoRequest): Promise<Empty> {
    return service<GoodsInfoForm | undefined, Empty>({
      url: `${GOODS_URL}/${request.id}`,
      method: "put",
      data: request.goods_info
    });
  }
  /** 删除商品 */
  DeleteGoodsInfo(request: DeleteGoodsInfoRequest): Promise<Empty> {
    return service<DeleteGoodsInfoRequest, Empty>({
      url: `${GOODS_URL}/${request.ids}`,
      method: "delete"
    });
  }
  /** 设置状态 */
  SetGoodsInfoStatus(request: SetGoodsInfoStatusRequest): Promise<Empty> {
    return service<SetGoodsInfoStatusRequest, Empty>({
      url: `${GOODS_URL}/${request.id}/status`,
      method: "put",
      data: request
    });
  }
}

export const defGoodsInfoService = new GoodsInfoServiceImpl();
