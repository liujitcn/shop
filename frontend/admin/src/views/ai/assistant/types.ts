import type { AiAssistantAttachment, AiAssistantMessage, AiAssistantSession } from "@/rpc/base/v1/ai_assistant_session";

/** 会话菜单动作类型。 */
export type SessionAction = "rename" | "delete";

/** 聊天输入提交内容。 */
export type SubmitPayload = {
  /** 前端生成的消息编号，用于关联本轮流式回复。 */
  clientMessageId?: string;
  /** 用户输入文本。 */
  text: string;
  /** 已上传附件列表。 */
  attachments: AiAssistantAttachment[];
};

/** 消息进度状态，用于控制气泡加载、失败和常规展示。 */
export type MessageProgressState = "idle" | "pending" | "streaming" | "failed";

/** 回复来源标签配置。 */
export type ReplySourceTag = {
  /** 标签文本。 */
  text: string;
  /** 标签视觉语义。 */
  tone: "primary" | "success" | "warning" | "info";
};

/** 聊天气泡展示项，在后端消息基础上补充 UI 状态。 */
export type ChatMessageItem = AiAssistantMessage & {
  /** BubbleList 稳定渲染键。 */
  key: string;
  /** 气泡左右位置。 */
  placement: "start" | "end";
  /** 气泡视觉样式。 */
  variant?: "filled" | "borderless" | "outlined" | "shadow";
  /** 气泡形状。 */
  shape?: "round" | "corner";
  /** 气泡最大宽度。 */
  maxWidth?: string;
  /** 本地消息进度状态。 */
  progressState?: MessageProgressState;
  /** 是否为前端临时消息，收到服务端最终消息后会被替换。 */
  localOnly?: boolean;
  /** 回复来源标签。 */
  replySourceTag?: ReplySourceTag;
  /** 本轮用户消息编号，用于直连流式回复分组。 */
  clientMessageId?: string;
  /** 本地流式消息键，按会话和用户消息拆分。 */
  streamKey?: string;
};

/** AI 助手 direct stream SSE 事件名称。 */
export type AiAssistantStreamEventName = "delta" | "finish" | "error";

/** AI 助手 direct stream 事件负载。 */
export type AiAssistantStreamPayload = {
  /** 会话 ID。 */
  session_id: string;
  /** 前端本地消息 ID，用于关联当前轮次。 */
  client_message_id: string;
  /** 本次新增文本分片。 */
  delta?: string;
  /** 流式完成后的最终消息列表。 */
  messages?: AiAssistantMessage[];
  /** 流式完成后的最新会话。 */
  session?: AiAssistantSession;
  /** 异常提示。 */
  error_message?: string;
};

/** AI 助手 direct stream 标准化事件。 */
export type AiAssistantStreamEvent = {
  /** SSE 事件名称。 */
  event: AiAssistantStreamEventName;
  /** 已解析的 JSON 负载。 */
  payload: AiAssistantStreamPayload;
};
