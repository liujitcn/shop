import service from "@/utils/request";
import {
  type BaseDictForm,
  type BaseDictItemForm,
  type BaseDictService,
  type ListBaseDictResponse,
  type PageBaseDictItemRequest,
  type PageBaseDictItemResponse,
  type PageBaseDictRequest,
  type PageBaseDictResponse,
} from "@/rpc/admin/base_dict";
import type { Empty } from "@/rpc/google/protobuf/empty";
import type { Int64Value, StringValue } from "@/rpc/google/protobuf/wrappers";
import type { SetStatusRequest } from "@/rpc/common/common";

const BASE_DICT_URL = "/admin/base/dict";

const BASE_DICT_ITEM_URL = "/admin/base/dict-item";

/** Admin字典服务 */
export class BaseDictServiceImpl implements BaseDictService {
  /** 查询字典列表 */
  ListBaseDict(request: Empty): Promise<ListBaseDictResponse> {
    return service<Empty, ListBaseDictResponse>({
      url: `${BASE_DICT_URL}/list`,
      method: "get",
      params: request,
    });
  }
  /** 查询字典分页列表 */
  PageBaseDict(request: PageBaseDictRequest): Promise<PageBaseDictResponse> {
    return service<PageBaseDictRequest, PageBaseDictResponse>({
      url: `${BASE_DICT_URL}`,
      method: "get",
      params: request,
    });
  }
  /** 查询字典 */
  GetBaseDict(request: Int64Value): Promise<BaseDictForm> {
    return service<Int64Value, BaseDictForm>({
      url: `${BASE_DICT_URL}/${request.value}`,
      method: "get",
    });
  }
  /** 创建字典 */
  CreateBaseDict(request: BaseDictForm): Promise<Empty> {
    return service<BaseDictForm, Empty>({
      url: `${BASE_DICT_URL}`,
      method: "post",
      data: request,
    });
  }
  /** 更新字典 */
  UpdateBaseDict(request: BaseDictForm): Promise<Empty> {
    return service<BaseDictForm, Empty>({
      url: `${BASE_DICT_URL}/${request.id}`,
      method: "put",
      data: request,
    });
  }
  /** 删除字典 */
  DeleteBaseDict(request: StringValue): Promise<Empty> {
    return service<Int64Value, Empty>({
      url: `${BASE_DICT_URL}/${request.value}`,
      method: "delete",
    });
  }
  /** 设置状态 */
  SetBaseDictStatus(request: SetStatusRequest): Promise<Empty> {
    return service<SetStatusRequest, Empty>({
      url: `${BASE_DICT_URL}/${request.id}/status`,
      method: "put",
      data: request,
    });
  }
  /** 查询字典属性分页列表 */
  PageBaseDictItem(request: PageBaseDictItemRequest): Promise<PageBaseDictItemResponse> {
    return service<PageBaseDictItemRequest, PageBaseDictItemResponse>({
      url: `${BASE_DICT_ITEM_URL}`,
      method: "get",
      params: request,
    });
  }
  /** 查询字典属性 */
  GetBaseDictItem(request: Int64Value): Promise<BaseDictItemForm> {
    return service<Int64Value, BaseDictItemForm>({
      url: `${BASE_DICT_ITEM_URL}/${request.value}`,
      method: "get",
    });
  }
  /** 创建字典属性 */
  CreateBaseDictItem(request: BaseDictItemForm): Promise<Empty> {
    return service<BaseDictItemForm, Empty>({
      url: `${BASE_DICT_ITEM_URL}`,
      method: "post",
      data: request,
    });
  }
  /** 更新字典属性 */
  UpdateBaseDictItem(request: BaseDictItemForm): Promise<Empty> {
    return service<BaseDictItemForm, Empty>({
      url: `${BASE_DICT_ITEM_URL}/${request.id}`,
      method: "put",
      data: request,
    });
  }
  /** 删除字典属性 */
  DeleteBaseDictItem(request: StringValue): Promise<Empty> {
    return service<Int64Value, Empty>({
      url: `${BASE_DICT_ITEM_URL}/${request.value}`,
      method: "delete",
    });
  }
  /** 设置状态 */
  SetBaseDictItemStatus(request: SetStatusRequest): Promise<Empty> {
    return service<SetStatusRequest, Empty>({
      url: `${BASE_DICT_ITEM_URL}/${request.id}/status`,
      method: "put",
      data: request,
    });
  }
}

export const defBaseDictService = new BaseDictServiceImpl();
