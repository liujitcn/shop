import { defineStore } from "pinia";
import pinia from "@/stores";
import { defBaseDictService } from "@/api/admin/base_dict";
import type { DictState } from "@/stores/interface";
import piniaPersistConfig from "@/stores/helper/persist";
import type { OptionBaseDictsResponse_BaseDict, OptionBaseDictsResponse_BaseDictItem } from "@/rpc/admin/v1/base_dict";

export const useDictStore = defineStore({
  id: "shop-dict",
  state: (): DictState => ({
    dictionary: {}
  }),
  getters: {},
  actions: {
    /** 设置单个字典缓存 */
    setDictionary(dict: OptionBaseDictsResponse_BaseDict) {
      if (!dict.code) return;
      this.dictionary[dict.code] = dict.items ?? [];
    },
    /** 从服务端加载全部字典缓存 */
    async loadDictionaries(forceRefresh = false) {
      if (!forceRefresh && Object.keys(this.dictionary).length) return this.dictionary;

      const dictRes = await defBaseDictService.OptionBaseDicts({});
      const nextDictionary: Record<string, OptionBaseDictsResponse_BaseDictItem[]> = {};

      const compatDictRes = dictRes as typeof dictRes & {
        baseDicts?: typeof dictRes.base_dicts;
        list?: typeof dictRes.base_dicts;
      };
      // 优先读取 snake_case 字典集合，兼容旧 camelCase/list 响应。
      const baseDicts = compatDictRes.base_dicts ?? compatDictRes.baseDicts ?? compatDictRes.list ?? [];
      baseDicts.forEach(dict => {
        if (!dict.code) return;
        nextDictionary[dict.code] = dict.items ?? [];
      });

      this.dictionary = nextDictionary;
      return this.dictionary;
    },
    /** 获取指定字典缓存 */
    getDictionary(dictCode: string): OptionBaseDictsResponse_BaseDictItem[] {
      return this.dictionary[dictCode] ?? [];
    },
    /** 清空字典缓存 */
    clearDictionaryCache() {
      this.dictionary = {};
    },
    /** 强制刷新字典缓存 */
    async updateDictionaryCache() {
      this.clearDictionaryCache();
      await this.loadDictionaries(true);
    }
  },
  persist: piniaPersistConfig("shop-dict")
});

/**
 * 在非 setup 场景使用 Dict Store。
 */
export function useDictStoreHook() {
  return useDictStore(pinia);
}
