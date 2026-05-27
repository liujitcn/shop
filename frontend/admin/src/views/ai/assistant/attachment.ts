import type { AiAssistantAttachment } from "@/rpc/base/v1/ai_assistant_session";
import { formatSrc } from "@/utils/utils";
import type { FilesCardProps, FilesType } from "vue-element-plus-x/types/FilesCard";

/** AI 助手附件图片判断所需的最小字段。 */
export type AssistantAttachmentPreviewMeta = Pick<AiAssistantAttachment, "name" | "mime_type">;

/** AI 助手附件卡片生成选项，用于复用 FilesCard 内置预览、删除和状态能力。 */
export type AssistantAttachmentFileCardOptions = Partial<
  Pick<FilesCardProps, "imgFile" | "imgVariant" | "maxWidth" | "percent" | "showDelIcon" | "status">
>;

/** 补齐 AI 助手附件地址，用于图片预览和浏览器新窗口打开。 */
export function resolveAssistantAttachmentUrl(url?: string) {
  return formatSrc(String(url ?? ""));
}

/** 根据文件后缀推断附件卡片类型，优先复用组件自带图标表现。 */
export function resolveAssistantAttachmentFileType(fileName?: string): FilesType {
  const extension = String(fileName ?? "")
    .split(".")
    .pop()
    ?.toLowerCase();
  if (["png", "jpg", "jpeg", "gif", "webp", "svg"].includes(extension ?? "")) return "image";
  if (["xls", "xlsx", "csv"].includes(extension ?? "")) return "excel";
  if (["doc", "docx"].includes(extension ?? "")) return "word";
  if (["ppt", "pptx"].includes(extension ?? "")) return "ppt";
  if (extension === "pdf") return "pdf";
  if (["zip", "rar", "7z"].includes(extension ?? "")) return "zip";
  if (["mp3", "wav", "m4a"].includes(extension ?? "")) return "audio";
  if (["mp4", "mov", "avi"].includes(extension ?? "")) return "video";
  if (["md", "markdown"].includes(extension ?? "")) return "mark";
  if (["txt", "log"].includes(extension ?? "")) return "txt";
  if (["json", "xml"].includes(extension ?? "")) return "code";
  return "file";
}

/** 判断附件是否为图片，优先相信 MIME 类型，缺失时回退到后缀。 */
export function isAssistantImageAttachment(attachment: AssistantAttachmentPreviewMeta) {
  const mimeType = String(attachment.mime_type ?? "").toLowerCase();
  if (mimeType.startsWith("image/")) return true;
  return resolveAssistantAttachmentFileType(attachment.name) === "image";
}

/** 将后端附件转换成 Element Plus X FilesCard 数据，避免页面重复手写附件预览逻辑。 */
export function buildAssistantAttachmentFileCard(
  attachment: AiAssistantAttachment,
  options: AssistantAttachmentFileCardOptions = {}
): FilesCardProps {
  const isImage = isAssistantImageAttachment(attachment);
  const previewUrl = resolveAssistantAttachmentUrl(attachment.url);
  return {
    uid: attachment.id || attachment.url || attachment.name,
    name: attachment.name,
    fileSize: Number(attachment.size ?? 0),
    url: previewUrl,
    thumbUrl: isImage ? previewUrl : undefined,
    fileType: resolveAssistantAttachmentFileType(attachment.name),
    imgPreview: isImage,
    imgPreviewMask: isImage,
    showDelIcon: Boolean(options.showDelIcon),
    imgFile: options.imgFile,
    imgVariant: options.imgVariant,
    maxWidth: options.maxWidth,
    status: options.status,
    percent: options.percent
  };
}
