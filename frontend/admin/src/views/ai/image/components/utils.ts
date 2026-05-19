import type { AiImage } from "@/rpc/base/v1/ai_image";
import type { Timestamp } from "@/rpc/google/protobuf/timestamp";
import { formatSrc } from "@/utils/utils";
import type { ImageItem } from "./types";

/** 标准化图片结果，补齐预览地址和稳定键。 */
export function normalizeImages(list: AiImage[] = []): ImageItem[] {
  return list.map((item, index) => {
    const url = String(item.url ?? "");
    return {
      ...item,
      key: `${index}-${url || item.name || Date.now()}`,
      previewUrl: formatSrc(url)
    };
  });
}

/** 格式化 protobuf 时间戳。 */
export function formatTimestamp(value?: Timestamp) {
  if (!value) return "";
  const seconds = Number(value.seconds || 0);
  if (!seconds) return "";
  return new Date(seconds * 1000).toLocaleString();
}

/** 按 MIME 类型推断下载扩展名。 */
export function resolveImageExtension(mimeType?: string) {
  switch (String(mimeType ?? "").toLowerCase()) {
    case "image/jpeg":
      return "jpg";
    case "image/webp":
      return "webp";
    default:
      return "png";
  }
}
