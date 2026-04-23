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

/** 将远程推荐响应解析为游标列表。 */
export function parseRemoteCursorList(json: string, listKeys: string[]): RemoteCursorList {
  const raw = parseRemoteJson(json);
  const record = isRemoteRecord(raw) ? raw : {};
  const list = resolveList(record, listKeys);
  const cursor = resolveString(record.Cursor ?? record.cursor);
  return { list, cursor, raw };
}

/** 读取远程记录编号。 */
export function resolveRemoteId(record: RemoteRecord, keys: string[]) {
  for (const key of keys) {
    const value = record[key];
    if (value !== undefined && value !== null && String(value).trim() !== "") return String(value);
  }
  return "";
}

/** 将单元格内容格式化为可读文本。 */
export function formatRemoteCell(value: unknown) {
  if (value === undefined || value === null || value === "") return "--";
  if (typeof value === "object") return JSON.stringify(value);
  return String(value);
}

/** 判断值是否为远程推荐记录。 */
function isRemoteRecord(value: unknown): value is RemoteRecord {
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
