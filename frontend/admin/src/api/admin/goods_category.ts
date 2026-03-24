import service from "@/utils/request";
import {
  type GoodsCategoryForm,
  type GoodsCategoryService,
  type OptionGoodsCategoryRequest,
  type TreeGoodsCategoryResponse,
} from "@/rpc/admin/goods_category";
import { type Empty } from "@/rpc/google/protobuf/empty";
import { type Int64Value, type StringValue } from "@/rpc/google/protobuf/wrappers";
import { type SetStatusRequest, type TreeOptionResponse } from "@/rpc/common/common";

const GOODS_CATEGORY_URL = "/admin/goods/category";

/** Admin分类服务 */
export class GoodsCategoryServiceImpl implements GoodsCategoryService {
  /** 查询分类树形列表 */
  TreeGoodsCategory(request: Empty): Promise<TreeGoodsCategoryResponse> {
    return service<Empty, TreeGoodsCategoryResponse>({
      url: `${GOODS_CATEGORY_URL}/tree`,
      method: "get",
      params: request,
    });
  }
  /** 查询分类树形选择 */
  OptionGoodsCategory(request: OptionGoodsCategoryRequest): Promise<TreeOptionResponse> {
    return service<Empty, TreeOptionResponse>({
      url: `${GOODS_CATEGORY_URL}/option`,
      method: "get",
      params: request,
    });
  }
  /** 查询分类 */
  GetGoodsCategory(request: Int64Value): Promise<GoodsCategoryForm> {
    return service<Int64Value, GoodsCategoryForm>({
      url: `${GOODS_CATEGORY_URL}/${request.value}`,
      method: "get",
    });
  }
  /** 创建分类 */
  CreateGoodsCategory(request: GoodsCategoryForm): Promise<Empty> {
    return service<GoodsCategoryForm, Empty>({
      url: `${GOODS_CATEGORY_URL}`,
      method: "post",
      data: request,
    });
  }
  /** 更新分类 */
  UpdateGoodsCategory(request: GoodsCategoryForm): Promise<Empty> {
    return service<GoodsCategoryForm, Empty>({
      url: `${GOODS_CATEGORY_URL}/${request.id}`,
      method: "put",
      data: request,
    });
  }
  /** 删除分类 */
  DeleteGoodsCategory(request: StringValue): Promise<Empty> {
    return service<Int64Value, Empty>({
      url: `${GOODS_CATEGORY_URL}/${request.value}`,
      method: "delete",
    });
  }
  /** 设置状态 */
  SetGoodsCategoryStatus(request: SetStatusRequest): Promise<Empty> {
    return service<SetStatusRequest, Empty>({
      url: `${GOODS_CATEGORY_URL}/${request.id}/status`,
      method: "put",
      data: request,
    });
  }
}

export const defGoodsCategoryService = new GoodsCategoryServiceImpl();
