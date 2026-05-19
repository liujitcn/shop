import type { AiImage, CreateAiImageTaskRequest } from "@/rpc/base/v1/ai_image";

/** AI 图片生成状态值。 */
export enum TaskStatus {
  Pending = 1,
  Running = 2,
  Success = 3,
  Failed = 4,
  Timeout = 5
}

/** AI 图片生成表单。 */
export type GenerateFormModel = Pick<
  CreateAiImageTaskRequest,
  "prompt" | "size" | "quality" | "background" | "output_format" | "n" | "save_output" | "polish_prompt"
>;

/** 图片卡片展示项。 */
export type ImageItem = AiImage & {
  /** 前端渲染稳定键。 */
  key: string;
  /** 补齐静态资源域名后的预览地址。 */
  previewUrl: string;
};
