import type { EnumProps } from "@/components/ProTable/interface";
import type { OptionBaseDictsResponse_BaseDictItem } from "@/rpc/admin/v1/base_dict";
import { useDictStoreHook } from "@/stores/modules/dict";

type DictValueType = "number" | "string";
type SelectedId = string | number;

/** 表格分页入参，兼容 ProTable 旧 camelCase 分页字段与新接口 snake_case 字段。 */
type PageRequestParams = Record<string, any> & {
  /** 当前页码，优先使用接口 snake_case 字段。 */
  page_num?: string | number;
  /** 每页条数，优先使用接口 snake_case 字段。 */
  page_size?: string | number;
  /** ProTable 组件内部当前页码字段。 */
  pageNum?: string | number;
  /** ProTable 组件内部每页条数字段。 */
  pageSize?: string | number;
};

/** 归一化后的分页请求，同时保留 camelCase 字段供迁移中的页面解构使用。 */
type NormalizedPageRequest<T extends PageRequestParams> = Omit<T, "pageNum" | "pageSize"> & {
  /** 接口请求当前页码。 */
  page_num: number;
  /** 接口请求每页条数。 */
  page_size: number;
  /** 兼容迁移中的页面当前页码。 */
  pageNum: number;
  /** 兼容迁移中的页面每页条数。 */
  pageSize: number;
};

/**
 * 按配置将字典值转换为表格枚举可识别的类型。
 */
function transformDictValue(dictItem: OptionBaseDictsResponse_BaseDictItem, valueType: DictValueType) {
  if (valueType === "number") return Number(dictItem.value);
  return dictItem.value;
}

/**
 * 将字典缓存转换为 ProTable 搜索枚举数据。
 */
export async function buildDictEnum(code: string, valueType: DictValueType = "number") {
  const dictStore = useDictStoreHook();
  await dictStore.loadDictionaries();

  const data: EnumProps[] = dictStore.getDictionary(code).map(dictItem => ({
    label: dictItem.label,
    value: transformDictValue(dictItem, valueType),
    tagType: dictItem.tag_type
  }));

  return { data };
}

/**
 * 统一补齐分页请求参数，避免组件透传时出现字符串页码，并适配接口 snake_case 字段。
 */
export function buildPageRequest<T extends PageRequestParams>(params: T): NormalizedPageRequest<T> {
  const pageNum = Number(params.page_num ?? params.pageNum ?? 1);
  const pageSize = Number(params.page_size ?? params.pageSize ?? 10);

  return {
    ...params,
    // 新生成类型统一要求 page_num/page_size，旧字段暂时保留给尚未迁移的页面解构。
    page_num: pageNum,
    page_size: pageSize,
    pageNum,
    pageSize
  } as NormalizedPageRequest<T>;
}

/**
 * 统一整理表格多选和单条操作产生的 ID 集合。
 */
export function normalizeSelectedIds(selected?: SelectedId | SelectedId[]) {
  if (Array.isArray(selected)) {
    return selected.filter(item => item !== undefined && item !== null && item !== "");
  }
  if (selected === undefined || selected === null || selected === "") return [];
  return [selected];
}
