import dayjs from "dayjs";
import { formatJson } from "@/utils/utils";

/** 远程推荐原始记录。 */
export type RemoteRecord = Record<string, unknown>;

/** 远程推荐游标列表。 */
export interface RemoteCursorList<T extends RemoteRecord = RemoteRecord> {
  /** 当前页记录。 */
  list: T[];
  /** 下一页游标。 */
  cursor: string;
  /** 原始响应。 */
  raw: unknown;
}

/** 远程推荐时间序列指标配置。 */
export interface RemoteTimeseriesMetricConfig {
  /** 远程指标名称。 */
  name: string;
  /** 页面展示名称。 */
  label: string;
  /** 指标说明。 */
  description: string;
}

/** 远程推荐时间序列指标数据。 */
export interface RemoteTimeseriesMetric extends RemoteTimeseriesMetricConfig {
  /** 横轴时间。 */
  axis: string[];
  /** 指标值集合。 */
  values: number[];
  /** 当前值。 */
  current: number;
  /** 相邻两期差值。 */
  diff: number;
  /** 是否上升。 */
  increase: boolean;
}

/** 远程推荐分类行。 */
export interface RemoteCategoryRow {
  /** 分类名称。 */
  name: string;
  /** 分类数量。 */
  count: string;
}

/** 远程推荐配置分组。 */
export interface RemoteConfigSection {
  /** 分组名称。 */
  name: string;
  /** 分组字段。 */
  fields: RemoteConfigField[];
}

/** 远程推荐配置字段。 */
export interface RemoteConfigField {
  /** 字段名称。 */
  name: string;
  /** 字段原始值。 */
  value: unknown;
  /** 字段展示值。 */
  text: string;
  /** 是否为复杂结构。 */
  complex: boolean;
}

/** Gorse 管理端概览页使用的时间序列指标。 */
export const remoteOverviewMetrics: RemoteTimeseriesMetricConfig[] = [
  { name: "num_users", label: "用户数", description: "当前推荐引擎用户总量" },
  { name: "num_items", label: "商品数", description: "当前推荐引擎商品总量" },
  { name: "num_feedback", label: "反馈数", description: "累计反馈行为总量" },
  { name: "num_pos_feedbacks", label: "正向反馈", description: "正向反馈行为总量" },
  { name: "num_neg_feedbacks", label: "负向反馈", description: "负向反馈行为总量" }
];

/** 解析远程推荐 JSON 字符串。 */
export function parseRemoteJson(json: string): unknown {
  try {
    return JSON.parse(json || "{}");
  } catch {
    return {};
  }
}

/** 格式化远程推荐 JSON 字符串。 */
export function formatRemoteJson(json: string) {
  return formatJson(json || "{}");
}

/** 将任意远程值格式化为 JSON 字符串。 */
export function stringifyRemoteValue(value: unknown) {
  return JSON.stringify(value ?? {}, null, 2);
}

/** 将远程推荐响应解析为游标列表。 */
export function parseRemoteCursorList(json: string, listKeys: string[]): RemoteCursorList {
  const raw = parseRemoteJson(json);
  const record = isRemoteRecord(raw) ? raw : {};
  const list = resolveList(record, listKeys);
  const cursor = resolveString(resolveRemoteValue(record, ["Cursor", "cursor"]));
  return { list, cursor, raw };
}

/** 将远程推荐响应解析为记录列表。 */
export function parseRemoteRecordList(json: string, listKeys: string[] = []) {
  const raw = parseRemoteJson(json);
  if (Array.isArray(raw)) return raw.filter(isRemoteRecord);
  const record = isRemoteRecord(raw) ? raw : {};
  const list = resolveList(record, listKeys);
  if (list.length > 0) return list;
  return Object.values(record).find(Array.isArray)?.filter(isRemoteRecord) ?? [];
}

/** 将远程推荐响应解析为单条记录。 */
export function parseRemoteRecord(json: string, recordKeys: string[] = []) {
  const raw = parseRemoteJson(json);
  if (!isRemoteRecord(raw)) return {};
  for (const key of recordKeys) {
    const value = raw[key];
    if (isRemoteRecord(value)) return value;
  }
  return raw;
}

/** 读取远程记录编号。 */
export function resolveRemoteId(record: RemoteRecord, keys: string[]) {
  return resolveString(resolveRemoteValue(record, keys));
}

/** 按多个候选字段读取远程记录值。 */
export function resolveRemoteValue(record: RemoteRecord, keys: string[]) {
  for (const key of keys) {
    const value = record[key];
    if (value !== undefined && value !== null && String(value).trim() !== "") return value;
  }
  return undefined;
}

/** 按多个候选字段读取远程数组值。 */
export function resolveRemoteArray(record: RemoteRecord, keys: string[]) {
  const value = resolveRemoteValue(record, keys);
  if (Array.isArray(value)) return value;
  if (value === undefined || value === null || value === "") return [];
  return [value];
}

/** 按多个候选字段读取远程布尔值。 */
export function resolveRemoteBoolean(record: RemoteRecord, keys: string[]) {
  const value = resolveRemoteValue(record, keys);
  if (typeof value === "boolean") return value;
  if (typeof value === "string") return value.toLowerCase() === "true";
  if (typeof value === "number") return value === 1;
  return false;
}

/** 按多个候选字段读取远程数值。 */
export function resolveRemoteNumber(record: RemoteRecord, keys: string[]) {
  const value = resolveRemoteValue(record, keys);
  const numberValue = Number(value ?? 0);
  if (!Number.isFinite(numberValue)) return 0;
  return numberValue;
}

/** 将单元格内容格式化为可读文本。 */
export function formatRemoteCell(value: unknown) {
  if (value === undefined || value === null || value === "") return "--";
  if (Array.isArray(value)) return value.length ? value.join("、") : "--";
  if (typeof value === "object") return JSON.stringify(value);
  return String(value);
}

/** 将远程时间字段格式化为后台统一时间文案。 */
export function formatRemoteDateTime(value: unknown) {
  const text = resolveString(value);
  if (!text) return "--";
  const date = dayjs(text);
  if (!date.isValid()) return text;
  return date.format("YYYY-MM-DD HH:mm:ss");
}

/** 将远程数值格式化为千分位文本。 */
export function formatRemoteNumber(value: unknown) {
  const numberValue = Number(value ?? 0);
  if (!Number.isFinite(numberValue)) return "0";
  return numberValue.toLocaleString();
}

/** 解析远程时间序列响应。 */
export function parseRemoteTimeseries(json: string) {
  const raw = parseRemoteJson(json);
  const list = Array.isArray(raw)
    ? raw
    : parseRemoteRecordList(json, ["Timeseries", "timeseries", "Items", "items", "Values", "values"]);
  const axis: string[] = [];
  const values: number[] = [];

  list.forEach((item, index) => {
    if (isRemoteRecord(item)) {
      const value = resolveRemoteValue(item, ["Value", "value", "Count", "count"]);
      const timestamp = resolveRemoteValue(item, ["Timestamp", "timestamp", "Time", "time", "Date", "date"]);
      values.push(resolveRemoteNumber({ value }, ["value"]));
      axis.push(formatTimeseriesAxis(timestamp, index));
      return;
    }
    values.push(resolveRemoteNumber({ value: item }, ["value"]));
    axis.push(String(index + 1));
  });

  return { axis, values };
}

/** 构建时间序列指标卡片数据。 */
export function buildRemoteTimeseriesMetric(config: RemoteTimeseriesMetricConfig, json: string): RemoteTimeseriesMetric {
  const timeseries = parseRemoteTimeseries(json);
  const current = timeseries.values.at(-1) ?? 0;
  const previous = timeseries.values.length > 1 ? (timeseries.values.at(-2) ?? 0) : current;
  const diff = current - previous;
  return {
    ...config,
    axis: timeseries.axis,
    values: timeseries.values,
    current,
    diff,
    increase: diff >= 0
  };
}

/** 解析远程推荐分类响应。 */
export function parseRemoteCategories(json: string): RemoteCategoryRow[] {
  const raw = parseRemoteJson(json);
  if (Array.isArray(raw)) {
    return raw.map((item, index) => normalizeCategoryRow(item, index)).filter(item => item.name);
  }
  if (!isRemoteRecord(raw)) return [];
  const list = resolveList(raw, ["Categories", "categories", "Items", "items", "List", "list"]);
  if (list.length > 0) return list.map((item, index) => normalizeCategoryRow(item, index)).filter(item => item.name);
  return Object.entries(raw).map(([name, count]) => ({ name, count: formatRemoteCell(count) }));
}

/** 解析远程配置分组，按 Gorse 设置页结构展示。 */
export function parseRemoteConfigSections(json: string): RemoteConfigSection[] {
  const raw = parseRemoteJson(json);
  if (!isRemoteRecord(raw)) return [];
  return Object.entries(raw).map(([name, value]) => ({
    name,
    fields: buildConfigFields(value)
  }));
}

/** 判断值是否为远程推荐记录。 */
export function isRemoteRecord(value: unknown): value is RemoteRecord {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

/** 从响应中读取列表字段。 */
function resolveList(record: RemoteRecord, listKeys: string[]) {
  for (const key of listKeys) {
    const value = record[key];
    if (Array.isArray(value)) return value.filter(isRemoteRecord);
  }
  return [];
}

/** 将远程字段转成字符串。 */
function resolveString(value: unknown) {
  if (value === undefined || value === null) return "";
  return String(value);
}

/** 格式化时间序列横轴。 */
function formatTimeseriesAxis(value: unknown, index: number) {
  const text = resolveString(value);
  if (!text) return String(index + 1);
  const date = dayjs(text);
  if (!date.isValid()) return text;
  return date.format("MM-DD HH:mm");
}

/** 将远程分类项规范成页面行。 */
function normalizeCategoryRow(value: unknown, index: number): RemoteCategoryRow {
  if (!isRemoteRecord(value)) {
    return {
      name: resolveString(value) || `分类${index + 1}`,
      count: "--"
    };
  }
  const name = resolveString(resolveRemoteValue(value, ["Name", "name", "Category", "category", "Label", "label", "Key", "key"]));
  const count = formatRemoteCell(resolveRemoteValue(value, ["Count", "count", "Value", "value", "Total", "total"]));
  return {
    name: name || `分类${index + 1}`,
    count
  };
}

/** 将远程配置结构转换为字段列表。 */
function buildConfigFields(value: unknown): RemoteConfigField[] {
  if (isRemoteRecord(value)) {
    return Object.entries(value).map(([name, fieldValue]) => ({
      name,
      value: fieldValue,
      text: formatConfigValue(fieldValue),
      complex: isComplexConfigValue(fieldValue)
    }));
  }
  return [
    {
      name: "value",
      value,
      text: formatConfigValue(value),
      complex: isComplexConfigValue(value)
    }
  ];
}

/** 判断配置值是否需要多行展示。 */
function isComplexConfigValue(value: unknown) {
  return Array.isArray(value) || isRemoteRecord(value);
}

/** 格式化配置字段值。 */
function formatConfigValue(value: unknown) {
  if (value === undefined || value === null) return "未配置";
  if (isComplexConfigValue(value)) return stringifyRemoteValue(value);
  return String(value);
}
