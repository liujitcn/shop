import { AiImageStatus, type AiImageResult, type CreateAiImageRequest } from "@/rpc/base/v1/ai_image";
import type { ProFormOption } from "@/components/ProForm/interface";
import type { EnumProps } from "@/components/ProTable/interface";

/** AI 图片状态展示配置。 */
export const aiImageStatusOptions: Array<EnumProps & { type: "primary" | "success" | "warning" | "danger" | "info" }> = [
  { label: "待处理", value: AiImageStatus.PENDING, tagType: "info", type: "info" },
  { label: "生成中", value: AiImageStatus.RUNNING, tagType: "warning", type: "warning" },
  { label: "成功", value: AiImageStatus.SUCCESS, tagType: "success", type: "success" },
  { label: "失败", value: AiImageStatus.FAILED, tagType: "danger", type: "danger" },
  { label: "超时", value: AiImageStatus.TIMEOUT, tagType: "danger", type: "danger" }
];

/** AI 图片尺寸选项。 */
export const imageSizeOptions: ProFormOption[] = [
  { label: "1024 x 1024", value: "1024x1024" },
  { label: "1536 x 1024", value: "1536x1024" },
  { label: "1024 x 1536", value: "1024x1536" },
  { label: "自动", value: "auto" }
];

/** AI 图片质量选项。 */
export const imageQualityOptions: ProFormOption[] = [
  { label: "自动", value: "auto" },
  { label: "高", value: "high" },
  { label: "中", value: "medium" },
  { label: "低", value: "low" },
  { label: "HD", value: "hd" },
  { label: "标准", value: "standard" }
];

/** AI 图片输出格式选项。 */
export const imageFormatOptions: ProFormOption[] = [
  { label: "PNG", value: "png" },
  { label: "JPEG", value: "jpeg" },
  { label: "WEBP", value: "webp" }
];

/** AI 图片背景选项。 */
export const imageBackgroundOptions: ProFormOption[] = [
  { label: "自动", value: "auto" },
  { label: "透明", value: "transparent" },
  { label: "不透明", value: "opaque" }
];

/** AI 图片生成表单。 */
export type GenerateFormModel = Pick<
  CreateAiImageRequest,
  "prompt" | "size" | "quality" | "background" | "output_format" | "n" | "polish_prompt"
>;

/** 图片卡片展示项。 */
export type ImageItem = AiImageResult & {
  /** 前端渲染稳定键。 */
  key: string;
  /** 补齐静态资源域名后的预览地址。 */
  previewUrl: string;
};
