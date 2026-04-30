import { defineStore } from "pinia";
import { defGoodsCategoryService } from "@/api/admin/goods_category";
import { defRecommendGorseService } from "@/api/admin/recommend_gorse";
import { useDictStoreHook } from "@/stores/modules/dict";
import type { OptionBaseDictsResponse_BaseDictItem } from "@/rpc/admin/v1/base_dict";
import type { ConfigResponse } from "@/rpc/admin/v1/recommend_gorse";
import type { TreeOptionResponse_Option } from "@/rpc/common/v1/common";

/** Gorse 推荐下拉选项。 */
interface GorseSelectOption {
  /** 页面显示名称。 */
  label: string;
  /** 接口请求原始值。 */
  value: string;
}

/** 商品分类树节点。 */
export interface GoodsCategoryTreeOption {
  /** 节点值。 */
  value: string;
  /** 节点名称。 */
  label: string;
  /** 是否禁用。 */
  disabled?: boolean;
  /** 子节点。 */
  children?: GoodsCategoryTreeOption[];
}

/** Gorse 推荐配置原始对象，按当前固定返回属性读取。 */
type ConfigRecord = Record<string, unknown>;

/** 配置字段读取路径，数组用于兼容不同 JSON 命名策略。 */
type ConfigFieldKey = string | string[];

/** 推荐反馈类型中文文案，展示时转换编码，接口请求值仍保持原始编码。 */
const feedbackTypeLabelMap: Record<string, string> = {
  EXPOSURE: "曝光",
  CLICK: "点击",
  VIEW: "浏览",
  COLLECT: "收藏",
  ADD_CART: "加购",
  ORDER_CREATE: "下单",
  ORDER_PAY: "支付"
};

/** Gorse 推荐器缺少字典时的中文兜底文案。 */
const gorseRecommenderFallbackLabelMap: Record<string, string> = {
  latest: "最新推荐",
  collaborative: "协同过滤"
};

/** 推荐器字典编码。 */
const recommendProviderDictCode = "recommend_provider";

export const useRecommendGorseStore = defineStore({
  id: "shop-recommend-gorse",
  state: () => ({
    config: {} as ConfigResponse,
    categoryTreeOptions: [] as GoodsCategoryTreeOption[],
    categoryLabelMap: {} as Record<string, string>
  }),
  getters: {
    /** 概览页推荐器下拉数据，value 直接作为 ListDashboardItems 接口的推荐器名称。 */
    dashboardRecommenderOptions(state): GorseSelectOption[] {
      const recommend = toRecord(state.config.recommend);
      const nonPersonalizedRecommenders = readRecordList(recommend, ["non-personalized", "non_personalized"])
        .map(item => `non-personalized/${String(item.name ?? "").trim()}`)
        .filter(item => item !== "non-personalized/");
      // 下拉只展示非个性化推荐配置项，latest 固定补在第一项。
      const recommenders = uniqueStrings(["latest", ...nonPersonalizedRecommenders]);

      const dictStore = useDictStoreHook();
      const dictList = dictStore.getDictionary(recommendProviderDictCode);
      return recommenders.map(item => ({
        label: formatDashboardRecommenderLabel(item, dictList),
        value: item
      }));
    },
    /** 用户推荐页推荐器下拉数据，value 直接作为 GetUserRecommend 接口的 recommender。 */
    userRecommendRecommenderOptions(state): GorseSelectOption[] {
      const recommend = toRecord(state.config.recommend);
      const recommenders = buildUserRecommendRecommenders(recommend);

      const dictStore = useDictStoreHook();
      const dictList = dictStore.getDictionary(recommendProviderDictCode);
      return recommenders.map(item => ({
        label: formatGorseRecommenderLabel(item, dictList),
        value: item
      }));
    },
    /** 用户相似推荐器下拉数据，value 直接作为 GetUserSimilar 接口的 recommender。 */
    userToUserRecommenderOptions(state): GorseSelectOption[] {
      const recommend = toRecord(state.config.recommend);
      const recommenders = readRecordList(recommend, ["user-to-user", "user_to_user"])
        .map(item => String(item.name ?? "").trim())
        .filter(Boolean);

      const dictStore = useDictStoreHook();
      const dictList = dictStore.getDictionary(recommendProviderDictCode);
      return uniqueStrings(recommenders).map(item => ({
        label: formatUserToUserRecommenderLabel(item, dictList),
        value: item
      }));
    },
    /** 商品相似推荐器下拉数据，value 直接作为 GetItemSimilar 接口的 recommender。 */
    itemToItemRecommenderOptions(state): GorseSelectOption[] {
      const recommend = toRecord(state.config.recommend);
      const recommenders = readRecordList(recommend, ["item-to-item", "item_to_item"])
        .map(item => String(item.name ?? "").trim())
        .filter(Boolean);

      const dictStore = useDictStoreHook();
      const dictList = dictStore.getDictionary(recommendProviderDictCode);
      return uniqueStrings(recommenders).map(item => ({
        label: formatItemToItemRecommenderLabel(item, dictList),
        value: item
      }));
    },
    /** 概览页推荐性能指标下拉数据，value 直接作为 GetTimeSeries 接口的指标名。 */
    performanceOptions(state): GorseSelectOption[] {
      const recommend = readRecord(state.config, "recommend");
      const dataSource = readRecord(recommend, "data_source");
      const options: GorseSelectOption[] = [{ label: "正向反馈占比（全部）", value: "positive_feedback_ratio" }];
      readStringList(dataSource, "positive_feedback_types").forEach(type => {
        // 页面下拉展示中文文案，接口请求仍使用 Gorse 服务原始反馈类型编码。
        options.push({ label: `正向反馈占比（${formatFeedbackTypeLabel(type)}）`, value: `positive_feedback_ratio_${type}` });
      });
      options.push(
        { label: "协同过滤 NDCG", value: "cf_ndcg" },
        { label: "协同过滤准确率", value: "cf_precision" },
        { label: "协同过滤召回率", value: "cf_recall" },
        { label: "点击率 AUC", value: "ctr_auc" },
        { label: "点击率准确率", value: "ctr_precision" },
        { label: "点击率召回率", value: "ctr_recall" }
      );
      return options;
    }
  },
  actions: {
    /**
     * 加载 Gorse 推荐配置，默认复用已加载的原始 ConfigResponse。
     */
    async loadConfig(force = false) {
      const dictStore = useDictStoreHook();
      await dictStore.loadDictionaries(force);

      // 配置已加载且未强制刷新时，直接复用 store 中的原始响应。
      if (Object.keys(toRecord(this.config)).length && !force) return this.config;

      this.config = await defRecommendGorseService.GetConfig({});
      return this.config;
    },
    /** 加载商品分类树与分类路径映射。 */
    async loadGoodsCategoryOptions(force = false) {
      if (this.categoryTreeOptions.length && Object.keys(this.categoryLabelMap).length && !force) {
        return {
          treeOptions: this.categoryTreeOptions,
          labelMap: this.categoryLabelMap
        };
      }

      const data = await defGoodsCategoryService.OptionGoodsCategories({});
      const nextLabelMap: Record<string, string> = {};
      this.categoryTreeOptions = buildGoodsCategoryTreeOptions(data.list ?? [], nextLabelMap);
      this.categoryLabelMap = nextLabelMap;
      return {
        treeOptions: this.categoryTreeOptions,
        labelMap: this.categoryLabelMap
      };
    }
  }
});

/** 根据 Gorse 推荐返回的分类ID，裁剪出带父子层级的分类树。 */
export function buildScopedGoodsCategoryTree(tree: GoodsCategoryTreeOption[], categoryIDs: string[]) {
  const categoryIDSet = new Set(categoryIDs.map(item => String(item).trim()).filter(Boolean));
  return filterGoodsCategoryTree(tree, categoryIDSet);
}

/** 从对象中读取固定字段值。 */
function readValue(source: unknown, key: ConfigFieldKey) {
  const record = toRecord(source);
  const keys = Array.isArray(key) ? key : [key];
  for (const currentKey of keys) {
    const value = record[currentKey];
    // 当前命名策略下字段不存在时，继续尝试下一个候选字段名。
    if (value !== null && value !== undefined) return value;
  }
  return undefined;
}

/** 格式化推荐反馈类型中文文案。 */
function formatFeedbackTypeLabel(type: string) {
  return feedbackTypeLabelMap[type.trim().toUpperCase()] ?? type;
}

/** 格式化推荐器中文文案。 */
function formatDashboardRecommenderLabel(recommender: string, dictList: OptionBaseDictsResponse_BaseDictItem[]) {
  return formatGorseRecommenderLabel(recommender, dictList);
}

/** 格式化通用推荐器中文文案。 */
function formatGorseRecommenderLabel(recommender: string, dictList: OptionBaseDictsResponse_BaseDictItem[]) {
  const value = recommender.trim();
  const dictValue = buildGorseRecommenderDictValue(value);
  const matchedItem = dictList.find(item => item.value === dictValue);
  if (matchedItem?.label) return matchedItem.label;

  // 字典缺少基础推荐器时，回退到页面内置中文文案，避免直接展示英文标识。
  return gorseRecommenderFallbackLabelMap[value] ?? value;
}

/** 格式化用户相似推荐器中文文案。 */
function formatUserToUserRecommenderLabel(recommender: string, dictList: OptionBaseDictsResponse_BaseDictItem[]) {
  const value = recommender.trim();
  const dictValue = buildGorseRecommenderDictValue(value, "user_to_user");
  const matchedItem = dictList.find(item => item.value === dictValue);
  if (matchedItem?.label) return matchedItem.label;

  return value;
}

/** 格式化商品相似推荐器中文文案。 */
function formatItemToItemRecommenderLabel(recommender: string, dictList: OptionBaseDictsResponse_BaseDictItem[]) {
  const value = recommender.trim();
  const dictValue = buildGorseRecommenderDictValue(value, "item_to_item");
  const matchedItem = dictList.find(item => item.value === dictValue);
  if (matchedItem?.label) return matchedItem.label;

  return value;
}

/** 将 Gorse 推荐器名称转换为 recommend_provider 字典值。 */
function buildGorseRecommenderDictValue(recommender: string, scope?: "user_to_user" | "item_to_item") {
  // 非个性化推荐器在接口中使用路径格式，字典中使用 gorse:non_personalized.xxx 格式。
  if (recommender.startsWith("non-personalized/")) {
    return `gorse:${recommender.replace("non-personalized/", "non_personalized.")}`;
  }
  // 用户推荐页会透传 item-to-item 命名空间，字典仍统一使用下划线作用域。
  if (recommender.startsWith("item-to-item/")) {
    return `gorse:${recommender.replace("item-to-item/", "item_to_item.")}`;
  }
  // 用户推荐页会透传 user-to-user 命名空间，字典仍统一使用下划线作用域。
  if (recommender.startsWith("user-to-user/")) {
    return `gorse:${recommender.replace("user-to-user/", "user_to_user.")}`;
  }
  if (scope === "user_to_user") {
    return `gorse:user_to_user.${recommender}`;
  }
  if (scope === "item_to_item") {
    return `gorse:item_to_item.${recommender}`;
  }
  return `gorse:${recommender}`;
}

/** 组装用户推荐页推荐器列表，保持与Gorse dashboard 原始下拉顺序一致。 */
function buildUserRecommendRecommenders(recommend: ConfigRecord) {
  const nonPersonalizedRecommenders = readRecordList(recommend, ["non-personalized", "non_personalized"])
    .map(item => `non-personalized/${String(item.name ?? "").trim()}`)
    .filter(item => item !== "non-personalized/");
  const itemToItemRecommenders = readRecordList(recommend, ["item-to-item", "item_to_item"])
    .map(item => `item-to-item/${String(item.name ?? "").trim()}`)
    .filter(item => item !== "item-to-item/");
  const userToUserRecommenders = readRecordList(recommend, ["user-to-user", "user_to_user"])
    .map(item => `user-to-user/${String(item.name ?? "").trim()}`)
    .filter(item => item !== "user-to-user/");
  const recommenders = ["latest"];

  // 协同过滤配置存在时，补充 dashboard 同款 collaborative 入口。
  if (readRecord(recommend, "collaborative")) {
    recommenders.push("collaborative");
  }
  recommenders.push(...nonPersonalizedRecommenders, ...itemToItemRecommenders, ...userToUserRecommenders);
  return uniqueStrings(recommenders);
}

/** 从对象中读取固定子对象。 */
function readRecord(source: unknown, key: ConfigFieldKey) {
  const value = key ? readValue(source, key) : source;
  if (typeof value !== "object" || value === null || Array.isArray(value)) return undefined;
  return value as ConfigRecord;
}

/** 从对象中读取固定对象数组。 */
function readRecordList(source: unknown, key: ConfigFieldKey) {
  const value = readValue(source, key);
  if (!Array.isArray(value)) return [];
  return value.map(toRecord).filter(item => Object.keys(item).length > 0);
}

/** 从对象中读取固定字符串数组。 */
function readStringList(source: unknown, key: ConfigFieldKey) {
  const value = readValue(source, key);
  if (Array.isArray(value)) return uniqueStrings(value.map(item => String(item).trim()).filter(Boolean));
  if (value === undefined || value === null || value === "") return [];
  return [String(value).trim()].filter(Boolean);
}

/** 将未知值转成普通对象，方便读取当前固定配置属性。 */
function toRecord(value: unknown): ConfigRecord {
  if (typeof value !== "object" || value === null || Array.isArray(value)) return {};
  return value as ConfigRecord;
}

/** 按原顺序去重字符串列表。 */
function uniqueStrings(values: string[]) {
  return Array.from(new Set(values.map(item => item.trim()).filter(Boolean)));
}

/** 构建商品分类树选项，并同步生成完整路径名称映射。 */
function buildGoodsCategoryTreeOptions(
  list: TreeOptionResponse_Option[],
  labelMap: Record<string, string>,
  pathLabels: string[] = []
) {
  return list.map(item => {
    const label = String(item.label ?? "").trim();
    const nextPathLabels = label ? [...pathLabels, label] : pathLabels;
    const value = String(item.value ?? "").trim();
    if (value) {
      labelMap[value] = nextPathLabels.join("/") || label;
    }
    return {
      value,
      label,
      disabled: item.disabled,
      children: buildGoodsCategoryTreeOptions(item.children ?? [], labelMap, nextPathLabels)
    } as GoodsCategoryTreeOption;
  });
}

/** 按分类ID集合裁剪分类树，并保留命中的父节点链路。 */
function filterGoodsCategoryTree(tree: GoodsCategoryTreeOption[], categoryIDSet: Set<string>): GoodsCategoryTreeOption[] {
  return tree
    .map(item => {
      const children = filterGoodsCategoryTree(item.children ?? [], categoryIDSet);
      const isMatched = categoryIDSet.has(String(item.value ?? "").trim());
      if (!isMatched && !children.length) return null;
      return {
        ...item,
        children
      } as GoodsCategoryTreeOption;
    })
    .filter((item): item is GoodsCategoryTreeOption => item !== null);
}
