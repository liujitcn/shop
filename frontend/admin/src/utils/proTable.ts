import type { EnumProps } from "@/components/ProTable/interface";
import type { OptionBaseDictResponse_BaseDictItem } from "@/rpc/admin/base_dict";
import { useDictStoreHook } from "@/stores/modules/dict";

type DictValueType = "number" | "string";
type SelectedId = string | number;

/**
 * 按配置将字典值转换为表格枚举可识别的类型。
 */
function transformDictValue(dictItem: OptionBaseDictResponse_BaseDictItem, valueType: DictValueType) {
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
    tagType: dictItem.tagType
  }));

  return { data };
}

/**
 * 统一补齐分页请求参数，避免组件透传时出现字符串页码。
 */
export function buildPageRequest<T extends Record<string, any>>(params: T) {
  return {
    ...params,
    pageNum: Number(params.pageNum ?? 1),
    pageSize: Number(params.pageSize ?? 10)
  } as T;
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
