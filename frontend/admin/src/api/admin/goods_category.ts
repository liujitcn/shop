import service from "@/utils/request";
import {
  type CreateGoodsCategoryRequest,
  type DeleteGoodsCategoryRequest,
  type GetGoodsCategoryRequest,
  type GoodsCategoryForm,
  type GoodsCategoryService,
  type OptionGoodsCategoryRequest,
  type SetGoodsCategoryStatusRequest,
  type TreeGoodsCategoryRequest,
  type TreeGoodsCategoryResponse,
  type UpdateGoodsCategoryRequest
} from "@/rpc/admin/v1/goods_category";
import { type Empty } from "@/rpc/google/protobuf/empty";
import { type TreeOptionResponse } from "@/rpc/common/v1/common";

const GOODS_CATEGORY_URL = "/v1/admin/goods/category";

/** Admin分类服务 */
export class GoodsCategoryServiceImpl implements GoodsCategoryService {
  /** 查询分类树形列表 */
  TreeGoodsCategory(request: TreeGoodsCategoryRequest): Promise<TreeGoodsCategoryResponse> {
    return service<TreeGoodsCategoryRequest, TreeGoodsCategoryResponse>({
      url: `${GOODS_CATEGORY_URL}/tree`,
      method: "get",
      params: request
    });
  }
  /** 查询分类树形选择 */
  OptionGoodsCategory(request: OptionGoodsCategoryRequest): Promise<TreeOptionResponse> {
    return service<OptionGoodsCategoryRequest, TreeOptionResponse>({
      url: `${GOODS_CATEGORY_URL}/option`,
      method: "get",
      params: request
    });
  }
  /** 查询分类 */
  GetGoodsCategory(request: GetGoodsCategoryRequest): Promise<GoodsCategoryForm> {
    return service<GetGoodsCategoryRequest, GoodsCategoryForm>({
      url: `${GOODS_CATEGORY_URL}/${request.id}`,
      method: "get"
    });
  }
  /** 创建分类 */
  CreateGoodsCategory(request: CreateGoodsCategoryRequest): Promise<Empty> {
    return service<GoodsCategoryForm | undefined, Empty>({
      url: `${GOODS_CATEGORY_URL}`,
      method: "post",
      data: request.goods_category
    });
  }
  /** 更新分类 */
  UpdateGoodsCategory(request: UpdateGoodsCategoryRequest): Promise<Empty> {
    return service<GoodsCategoryForm | undefined, Empty>({
      url: `${GOODS_CATEGORY_URL}/${request.id}`,
      method: "put",
      data: request.goods_category
    });
  }
  /** 删除分类 */
  DeleteGoodsCategory(request: DeleteGoodsCategoryRequest): Promise<Empty> {
    return service<DeleteGoodsCategoryRequest, Empty>({
      url: `${GOODS_CATEGORY_URL}/${request.ids}`,
      method: "delete"
    });
  }
  /** 设置状态 */
  SetGoodsCategoryStatus(request: SetGoodsCategoryStatusRequest): Promise<Empty> {
    return service<SetGoodsCategoryStatusRequest, Empty>({
      url: `${GOODS_CATEGORY_URL}/${request.id}/status`,
      method: "put",
      data: request
    });
  }
}

export const defGoodsCategoryService = new GoodsCategoryServiceImpl();
