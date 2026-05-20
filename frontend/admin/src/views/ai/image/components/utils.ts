import { AiImageStatus, type AiImageResult } from "@/rpc/base/v1/ai_image";
import type { Timestamp } from "@/rpc/google/protobuf/timestamp";
import { formatSrc } from "@/utils/utils";
import { aiImageStatusOptions, type GenerateFormModel, type ImageItem } from "./types";

/** 标准化图片结果，补齐预览地址和稳定键。 */
export function normalizeImages(list: AiImageResult[] = []): ImageItem[] {
  return list.map((item, index) => {
    const url = String(item.url ?? "");
    return {
      ...item,
      key: `${index}-${url || item.name || Date.now()}`,
      previewUrl: formatSrc(url)
    };
  });
}

/** 创建默认 AI 图片生成表单。 */
export function createDefaultGenerateForm(): GenerateFormModel {
  return {
    prompt: "",
    size: "1024x1024",
    quality: "auto",
    background: "auto",
    output_format: "png",
    n: 1,
    polish_prompt: false
  };
}

/** 格式化 protobuf 时间戳。 */
export function formatTimestamp(value?: Timestamp) {
  if (!value) return "";
  const seconds = Number(value.seconds || 0);
  if (!seconds) return "";
  const date = new Date(seconds * 1000);
  const pad = (num: number) => String(num).padStart(2, "0");
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(
    date.getMinutes()
  )}:${pad(date.getSeconds())}`;
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

/** 获取 AI 图片状态展示信息。 */
export function resolveAiImageStatusMeta(status?: AiImageStatus) {
  return aiImageStatusOptions.find(item => item.value === status) ?? aiImageStatusOptions[0];
}

/** 判断 AI 图片是否仍在生成中。 */
export function isGeneratingStatus(status?: AiImageStatus) {
  return status === AiImageStatus.PENDING || status === AiImageStatus.RUNNING;
}

/** 判断 AI 图片是否支持重试。 */
export function isRetryableStatus(status?: AiImageStatus) {
  return status === AiImageStatus.FAILED || status === AiImageStatus.TIMEOUT;
}
