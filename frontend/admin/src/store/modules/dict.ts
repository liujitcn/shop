import { store } from "@/store";
import { defBaseDictService } from "@/api/admin/base_dict";
import {
  type ListBaseDictResponse_BaseDict,
  type ListBaseDictResponse_BaseDictItem,
} from "@/rpc/admin/base_dict";

export const useDictStore = defineStore("dict", () => {
  const dictionary = useStorage<Record<string, ListBaseDictResponse_BaseDictItem[]>>(
    "dictionary",
    {}
  );

  const setDictionary = (dict: ListBaseDictResponse_BaseDict) => {
    dictionary.value[dict.code] = dict.items;
  };

  const loadDictionaries = async () => {
    const dictRes = await defBaseDictService.ListBaseDict({});
    const dictList = dictRes.list;
    dictList.forEach(setDictionary);
  };

  const getDictionary = (dictCode: string): ListBaseDictResponse_BaseDictItem[] => {
    return dictionary.value[dictCode] || [];
  };

  const clearDictionaryCache = () => {
    dictionary.value = {};
  };

  const updateDictionaryCache = async () => {
    clearDictionaryCache(); // 先清除旧缓存
    await loadDictionaries(); // 重新加载最新字典数据
  };

  return {
    dictionary,
    setDictionary,
    loadDictionaries,
    getDictionary,
    clearDictionaryCache,
    updateDictionaryCache,
  };
});

export function useDictStoreHook() {
  return useDictStore(store);
}
