import type { AiAssistantAttachment, AiAssistantMessage, AiAssistantSession } from "@/rpc/base/v1/ai_assistant";

export type SessionAction = "rename" | "delete";

export type SessionListItem = AiAssistantSession & {
  label: string;
};

export type SubmitPayload = {
  clientMessageId?: string;
  text: string;
  attachments: AiAssistantAttachment[];
};

export type MessageProgressState = "idle" | "pending" | "streaming" | "failed";

export type ReplySourceTag = {
  text: string;
  tone: "primary" | "success" | "warning" | "info";
};

export type ChatMessageItem = AiAssistantMessage & {
  key: string;
  placement: "start" | "end";
  variant?: "filled" | "borderless" | "outlined" | "shadow";
  shape?: "round" | "corner";
  maxWidth?: string;
  progressState?: MessageProgressState;
  localOnly?: boolean;
  replySourceTag?: ReplySourceTag;
};
