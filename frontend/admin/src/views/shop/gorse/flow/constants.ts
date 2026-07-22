import type { ProFormOption } from "@/components/ProForm/interface";
import type { PaletteNode } from "./types";

/** 推荐编排组件面板节点。 */
export const paletteNodes: PaletteNode[] = [
  { type: "latest", icon: "new_releases", label: "最新推荐" },
  { type: "collaborative", icon: "group_work", label: "协同过滤" },
  { type: "non-personalized", icon: "public", label: "非个性化" },
  { type: "user-to-user", icon: "people", label: "用户相似" },
  { type: "item-to-item", icon: "apps", label: "商品相似" },
  { type: "external", icon: "cloud_queue", label: "外部脚本" },
  { type: "ranker", icon: "sort", label: "排序器" },
  { type: "fallback", icon: "history", label: "兜底推荐" }
];

/** 节点类型中文展示文案。 */
export const nodeTypeLabelMap: Record<string, string> = {
  "data-source": "数据源",
  recommend: "推荐结果",
  latest: "最新推荐",
  collaborative: "协同过滤",
  "non-personalized": "非个性化",
  "user-to-user": "用户相似",
  "item-to-item": "商品相似",
  external: "外部脚本",
  ranker: "排序器",
  fallback: "兜底推荐"
};

/** 当前推荐器配置名称的中文展示文案，仅影响画布显示，不改写实际保存配置。 */
export const recommenderNameLabelMap: Record<string, string> = {
  goods_relation: "商品关联",
  similar_users: "相似用户",
  hot_7d: "近 7 天热门",
  hot_pay_30d: "近 30 天支付热门",
  hot_30d: "近 30 天热门",
  most_starred_weekly: "本周高收藏",
  neighbors: "相邻推荐"
};

/** 固定核心节点类型。 */
export const fixedNodeTypes = new Set(["data-source", "recommend"]);

/** 画布中只允许存在一个的节点类型。 */
export const singletonNodeTypes = new Set(["latest", "collaborative", "ranker", "fallback"]);

/** 可作为候选推荐器的节点类型。 */
export const recommenderNodeTypes = new Set([
  "latest",
  "collaborative",
  "non-personalized",
  "user-to-user",
  "item-to-item",
  "external"
]);

/** 用户相似推荐器类型选项。 */
export const userToUserTypeOptions: ProFormOption[] = [
  { label: "共同商品", value: "items" },
  { label: "标签", value: "tags" },
  { label: "向量", value: "embedding" },
  { label: "自动", value: "auto" }
];

/** 商品相似推荐器类型选项。 */
export const itemToItemTypeOptions: ProFormOption[] = [
  { label: "共同用户", value: "users" },
  { label: "标签", value: "tags" },
  { label: "向量", value: "embedding" },
  { label: "对话", value: "chat" },
  { label: "自动", value: "auto" }
];

/** 排序器类型选项。 */
export const rankerTypeOptions: ProFormOption[] = [
  { label: "关闭", value: "none" },
  { label: "因子分解机", value: "fm" },
  { label: "大语言模型", value: "llm" }
];
