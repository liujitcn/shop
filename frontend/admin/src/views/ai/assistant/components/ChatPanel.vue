<template>
  <main class="agent-chat-panel" :class="{ 'is-empty': isEmptyState }">
    <template v-if="isEmptyState">
      <div class="agent-chat-empty">
        <div class="agent-chat-empty__title">{{ welcomeTitle }}</div>
        <div class="agent-chat-empty__desc">可直接提问，也可以上传附件一起分析。</div>
        <div class="agent-chat-empty__sender">
          <XSender :key="senderKey" :sending="sending" @submit="handleSubmit" />
        </div>
      </div>
    </template>

    <template v-else>
      <div class="agent-chat-header">
        <div>
          <div class="agent-chat-title">{{ activeSession?.title }}</div>
          <div class="agent-chat-desc">{{ activeSession?.summary }}</div>
        </div>
      </div>

      <div class="agent-chat-content">
        <BubbleList class="agent-message-list" :list="bubbleList" max-height="100%" :auto-scroll="true">
          <template #content="{ item }">
            <div class="agent-message-body">
              <div v-if="item.role !== 'user' && (item.replySourceTag || item.model || item.fallback)" class="agent-message-meta">
                <span
                  v-if="item.replySourceTag"
                  class="agent-message-meta__tag"
                  :class="resolveTagClass(item.replySourceTag.tone)"
                >
                  {{ item.replySourceTag.text }}
                </span>
                <span v-if="item.model" class="agent-message-meta__model">{{ item.model }}</span>
              </div>
              <div
                class="agent-message-content"
                :class="{
                  'is-thinking': item.progressState === 'streaming',
                  'is-failed-assistant': isAssistantFailedMessage(item),
                  'is-user': item.role === 'user'
                }"
              >
                <div v-if="isAssistantFailedMessage(item)" class="agent-message-error">
                  <div class="agent-message-error__title">服务器异常，建议稍后重试</div>
                  <div class="agent-message-error__content">{{ resolveAssistantErrorMessage(item) }}</div>
                  <el-collapse v-if="item.fallback_reason" class="agent-message-error__detail" accordion>
                    <el-collapse-item title="错误详情" :name="String(item.id)">
                      <pre>{{ item.fallback_reason }}</pre>
                    </el-collapse-item>
                  </el-collapse>
                </div>
                <AiMarkdown v-else-if="item.role !== 'user'" :content="item.content" :streaming="item.progressState === 'streaming'" />
                <span v-else>{{ item.content }}</span>
                <span v-if="item.progressState === 'streaming'" class="agent-thinking-dots"> <i></i><i></i><i></i> </span>
              </div>
              <div v-if="item.attachments?.length" class="agent-message-attachments">
                <Attachments :items="buildMessageAttachmentItems(item.attachments)" overflow="wrap" :hide-upload="true" />
              </div>
            </div>
          </template>
          <template #footer="{ item }">
            <div class="agent-message-actions" :class="{ 'is-user': item.role === 'user' }">
              <template v-for="action in resolveMessageActions(item)" :key="action.key">
                <el-popconfirm
                  v-if="shouldConfirmMessageAction(action.key)"
                  :title="resolveActionConfirmTitle(action.key, item)"
                  confirm-button-text="确定"
                  cancel-button-text="取消"
                  width="220"
                  @confirm="handleMessageAction(action.key, item)"
                >
                  <template #reference>
                    <button
                      class="agent-message-action"
                      type="button"
                      :disabled="sending || item.progressState === 'streaming'"
                      :aria-label="action.label"
                    >
                      <el-icon><component :is="action.icon" /></el-icon>
                    </button>
                  </template>
                </el-popconfirm>
                <el-tooltip v-else :content="action.label" placement="top">
                  <button
                    class="agent-message-action"
                    type="button"
                    :disabled="sending || item.progressState === 'streaming'"
                    :aria-label="action.label"
                    @click="handleMessageAction(action.key, item)"
                  >
                    <el-icon><component :is="action.icon" /></el-icon>
                  </button>
                </el-tooltip>
              </template>
            </div>
          </template>
        </BubbleList>
      </div>

      <div class="agent-sender-wrap">
        <XSender :key="senderKey" :sending="sending" @submit="handleSubmit" />
      </div>
    </template>
  </main>
</template>

<script setup lang="ts" name="ChatPanel">
import { computed, defineAsyncComponent, h } from "vue";
import type { Component } from "vue";
import { Attachments, BubbleList } from "vue-element-plus-x";
import type { FilesCardProps } from "vue-element-plus-x/types/FilesCard";
import { CopyDocument, Delete, Refresh } from "@element-plus/icons-vue";
import {
  type AiAssistantAttachment,
  type AiAssistantSession
} from "@/rpc/base/v1/ai_assistant_session";
import { AiAssistantMessageStatus } from "@/rpc/common/v1/enum";
import XSender from "./XSender.vue";

// AI Markdown 渲染器依赖较重，仅在真正出现助手消息时再加载。
const AiMarkdown = defineAsyncComponent(() => import("./AiMarkdown.vue"));
import { buildAssistantAttachmentFileCard } from "../attachment";
import type { ChatMessageAction, ChatMessageItem, ReplySourceTag, SubmitPayload } from "../types";

/** 消息操作按钮配置。 */
type MessageActionOption = {
  /** 操作类型。 */
  key: ChatMessageAction;
  /** 悬浮提示和无障碍文案。 */
  label: string;
  /** 操作图标组件。 */
  icon: Component;
};

/** 朗读图标，按产品示意图绘制为喇叭声波。 */
const SpeakActionIcon = defineAsyncComponent(() =>
  Promise.resolve({
    name: "SpeakActionIcon",
    render() {
      return h(
        "svg",
        {
          class: "agent-message-action__custom-icon",
          xmlns: "http://www.w3.org/2000/svg",
          viewBox: "80 160 864 704",
          fill: "none",
          "aria-hidden": "true"
        },
        [
          h("path", {
            d: "M128 400h144L512 224v576L272 624H128z",
            stroke: "currentColor",
            "stroke-width": "64",
            "stroke-linejoin": "round"
          }),
          h("path", {
            d: "M640 352a224 224 0 0 1 0 320",
            stroke: "currentColor",
            "stroke-width": "64",
            "stroke-linecap": "round"
          }),
          h("path", {
            d: "M768 224a384 384 0 0 1 0 576",
            stroke: "currentColor",
            "stroke-width": "64",
            "stroke-linecap": "round"
          })
        ]
      );
    }
  })
);

/** 分支图标，按产品示意图绘制为向上分叉箭头。 */
const BranchActionIcon = defineAsyncComponent(() =>
  Promise.resolve({
    name: "BranchActionIcon",
    render() {
      return h(
        "svg",
        {
          class: "agent-message-action__custom-icon",
          xmlns: "http://www.w3.org/2000/svg",
          viewBox: "96 160 832 704",
          fill: "none",
          "aria-hidden": "true"
        },
        [
          h("path", {
            d: "M256 832V192",
            stroke: "currentColor",
            "stroke-width": "64",
            "stroke-linecap": "round"
          }),
          h("path", {
            d: "M256 192 128 320M256 192l128 128",
            stroke: "currentColor",
            "stroke-width": "64",
            "stroke-linecap": "round",
            "stroke-linejoin": "round"
          }),
          h("path", {
            d: "M384 640c256 0 384-176 384-448",
            stroke: "currentColor",
            "stroke-width": "64",
            "stroke-linecap": "round"
          }),
          h("path", {
            d: "M768 192 640 320M768 192l128 128",
            stroke: "currentColor",
            "stroke-width": "64",
            "stroke-linecap": "round",
            "stroke-linejoin": "round"
          })
        ]
      );
    }
  })
);

const props = defineProps<{
  /** 当前活动会话。 */
  activeSession?: AiAssistantSession;
  /** 当前会话消息列表。 */
  messages: ChatMessageItem[];
  /** 消息发送加载状态。 */
  sending: boolean;
}>();

const emit = defineEmits<{
  /** 提交输入框内容。 */
  submit: [payload: SubmitPayload];
  /** 触发消息级操作。 */
  messageAction: [payload: { action: ChatMessageAction; item: ChatMessageItem }];
}>();

const isEmptyState = computed(() => props.messages.length === 0);

const senderKey = computed(() => props.activeSession?.id || "empty-session");

const bubbleList = computed<ChatMessageItem[]>(() =>
  (props.messages ?? []).map(item => ({
    ...item,
    key: String(item.key ?? item.id ?? `${item.role}-${item.created_at?.seconds ?? 0}`),
    content: String(item.content ?? ""),
    placement: item.placement ?? "start"
  }))
);

const welcomeTitle = computed(() => {
  const hour = new Date().getHours();
  if (hour < 12) return "上午好，我是通用 AI 助手";
  if (hour < 18) return "下午好，我是通用 AI 助手";
  return "晚上好，我是通用 AI 助手";
});

/** 读取输入框内容并提交给父组件。 */
function handleSubmit(payload: SubmitPayload) {
  emit("submit", payload);
}

/** 根据消息角色返回可用操作。 */
function resolveMessageActions(item: ChatMessageItem) {
  if (item.progressState === "streaming" || item.status === AiAssistantMessageStatus.GENERATING_AAMS) return [];

  const copyAction: MessageActionOption = { key: "copy" as const, label: "复制", icon: CopyDocument };
  const deleteAction: MessageActionOption = { key: "delete" as const, label: "删除", icon: Delete };
  if (item.role === "user") {
    if (item.status === AiAssistantMessageStatus.FAILED_AAMS) {
      return [{ key: "retry" as const, label: "重新发送", icon: Refresh }, copyAction, deleteAction];
    }
    return [copyAction, deleteAction];
  }

  const actions: MessageActionOption[] = [{ key: "retry" as const, label: "重新生成", icon: Refresh }];
  if (item.role !== "user") {
    if (item.status === AiAssistantMessageStatus.SUCCESS_AAMS) {
      actions.push({ key: "branch" as const, label: "从此处创建分支会话", icon: BranchActionIcon });
      actions.push({ key: "speak" as const, label: item.speaking ? "停止朗读" : "朗读", icon: SpeakActionIcon });
    }
    return [
      ...actions,
      copyAction,
      deleteAction
    ];
  }
  return [copyAction, deleteAction];
}

/** 判断是否为需要渲染错误卡片的助手失败消息。 */
function isAssistantFailedMessage(item: ChatMessageItem) {
  return item.role !== "user" && item.status === AiAssistantMessageStatus.FAILED_AAMS;
}

/** 返回助手错误摘要，优先展示服务端可读错误。 */
function resolveAssistantErrorMessage(item: ChatMessageItem) {
  const content = String(item.content ?? "").trim();
  return content || "Service temporarily unavailable";
}

/** 向父组件透传消息操作，由页面层决定是否重发、复制或移除。 */
function handleMessageAction(action: ChatMessageAction, item: ChatMessageItem) {
  emit("messageAction", { action, item });
}

/** 判断消息操作是否需要二次确认。 */
function shouldConfirmMessageAction(action: ChatMessageAction) {
  return action === "retry" || action === "delete";
}

/** 返回消息操作确认文案，避免误删或误覆盖当前气泡。 */
function resolveActionConfirmTitle(action: ChatMessageAction, item: ChatMessageItem) {
  if (action === "delete") return "确认删除当前消息？";
  if (item.role === "user") return "确认重新发送当前消息？";
  return "重新生成会覆盖当前消息";
}

/** 统一回复来源标签配色。 */
function resolveTagClass(tone?: ReplySourceTag["tone"]) {
  return tone ? `is-${tone}` : "";
}

/** 构建消息附件卡片，统一交给 Attachments / FilesCard 处理图片预览。 */
function buildMessageAttachmentItems(attachments: AiAssistantAttachment[]): FilesCardProps[] {
  return attachments.map(attachment =>
    buildAssistantAttachmentFileCard(attachment, {
      maxWidth: "240px"
    })
  );
}

</script>

<style scoped lang="scss">
.agent-chat-panel {
  position: relative;
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  min-width: 0;
  height: 100%;
  min-height: 0;
  padding: 20px 0 24px;
  overflow: hidden;
  background: var(--admin-page-card-bg);
}
.agent-chat-panel.is-empty {
  justify-content: center;
  padding: 0;
}
.agent-chat-header {
  display: flex;
  flex: 0 0 auto;
  align-items: flex-start;
  justify-content: space-between;
  width: min(960px, calc(100% - 72px));
  margin: 0 auto;
}
.agent-chat-title {
  font-size: 16px;
  font-weight: 700;
  line-height: 24px;
  color: var(--admin-page-text-primary);
}
.agent-chat-desc {
  margin-top: 4px;
  font-size: 12px;
  color: var(--admin-page-text-secondary);
}
.agent-chat-content {
  display: flex;
  flex: 1;
  width: min(960px, calc(100% - 72px));
  min-height: 0;
  padding-bottom: 168px;
  margin: 0 auto;
  overflow: hidden;
}
.agent-chat-empty {
  display: flex;
  flex: 1;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  width: min(1100px, calc(100% - 96px));
  min-height: 0;
  margin: 0 auto;
}
.agent-chat-empty__title {
  font-size: clamp(28px, 4vw, 44px);
  font-weight: 700;
  line-height: 1.25;
  color: var(--admin-page-text-primary);
  text-align: center;
}
.agent-chat-empty__desc {
  margin-top: 14px;
  font-size: 14px;
  line-height: 24px;
  color: var(--admin-page-text-secondary);
  text-align: center;
}
.agent-chat-empty__sender {
  width: min(980px, 100%);
  margin-top: 28px;
}
.agent-message-list {
  flex: 1;
  min-height: 0;
  padding: 8px 0 24px;
  overflow: auto;
  :deep(.elx-bubble__content-wrapper),
  :deep(.elx-bubble__content) {
    min-width: 0;
  }
  :deep(.elx-bubble__content) {
    border-radius: var(--admin-page-radius);
  }
  :deep(.elx-bubble--start .elx-bubble__content-wrapper .elx-bubble__content--corner),
  :deep(.elx-bubble--end .elx-bubble__content-wrapper .elx-bubble__content--corner) {
    border-start-start-radius: var(--admin-page-radius);
    border-start-end-radius: var(--admin-page-radius);
  }
  :deep(.elx-bubble-list__boundary-content),
  :deep(.elx-bubble-list__embedded-item) {
    border-radius: var(--admin-page-radius);
  }
}
.agent-message-content {
  min-width: 0;
  line-height: 24px;
}
.agent-message-content.is-user {
  white-space: pre-wrap;
}
.agent-message-content.is-thinking {
  display: inline-flex;
  gap: 8px;
  align-items: center;
}
.agent-message-content.is-failed-assistant {
  width: min(460px, 100%);
  max-width: 100%;
}
.agent-message-body {
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-width: 0;
  max-width: 100%;
}
.agent-message-meta {
  display: inline-flex;
  gap: 8px;
  align-items: center;
  font-size: 12px;
  color: var(--admin-page-text-secondary);
}
.agent-message-meta__tag {
  padding: 2px 8px;
  background: var(--el-fill-color-light);
  border-radius: var(--admin-page-radius);
}
.agent-message-meta__tag.is-primary {
  color: var(--el-color-primary);
}
.agent-message-meta__tag.is-success {
  color: var(--el-color-success);
}
.agent-message-meta__tag.is-warning {
  color: var(--el-color-warning);
}
.agent-message-meta__tag.is-info {
  color: var(--admin-page-text-secondary);
}
.agent-message-meta__model {
  opacity: 0.85;
}
.agent-message-error {
  box-sizing: border-box;
  width: 100%;
  max-width: 100%;
  padding: 12px 14px;
  color: var(--admin-page-text-primary);
  background: var(--el-color-danger-light-9);
  border: 1px solid var(--el-color-danger-light-7);
  border-radius: var(--admin-page-radius);
}
.agent-message-error__title {
  font-size: 14px;
  font-weight: 700;
  line-height: 22px;
  color: var(--el-color-danger);
}
.agent-message-error__content {
  margin-top: 6px;
  font-size: 13px;
  line-height: 22px;
  color: var(--admin-page-text-regular);
  word-break: break-word;
}
.agent-message-error__detail {
  margin-top: 8px;
  border: 0;
  :deep(.el-collapse-item__header) {
    height: 28px;
    font-size: 12px;
    color: var(--admin-page-text-secondary);
    background: transparent;
    border: 0;
  }
  :deep(.el-collapse-item__wrap) {
    background: transparent;
    border: 0;
  }
  :deep(.el-collapse-item__content) {
    padding-bottom: 0;
  }
  pre {
    max-height: 180px;
    padding: 8px;
    margin: 0;
    overflow: auto;
    font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace;
    font-size: 12px;
    line-height: 18px;
    color: var(--admin-page-text-secondary);
    white-space: pre-wrap;
    word-break: break-word;
    background: var(--el-fill-color-blank);
    border: 1px solid var(--el-border-color-light);
    border-radius: var(--admin-page-radius);
  }
}
.agent-thinking-dots {
  display: inline-flex;
  gap: 4px;
  align-items: center;
  i {
    display: inline-block;
    width: 6px;
    height: 6px;
    background: currentcolor;
    border-radius: 50%;
    animation: thinking-bounce 1.2s infinite ease-in-out;
  }
  i:nth-child(2) {
    animation-delay: 0.15s;
  }
  i:nth-child(3) {
    animation-delay: 0.3s;
  }
}
.agent-message-attachments {
  :deep(.elx-files-card) {
    border-radius: var(--admin-page-radius);
  }
  :deep(.elx-files-card-img),
  :deep(.elx-files-card__image-preview),
  :deep(.elx-files-card-delete-icon) {
    border-radius: var(--admin-page-radius);
  }
}
.agent-message-actions {
  display: inline-flex;
  gap: 8px;
  align-items: center;
  width: 100%;
  margin-top: 8px;
}
.agent-message-actions.is-user {
  justify-content: flex-end;
}
.agent-message-action {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  color: var(--admin-page-text-secondary);
  cursor: pointer;
  background: transparent;
  border: 0;
  border-radius: var(--admin-page-radius);
  transition:
    color 0.2s ease,
    background-color 0.2s ease,
    opacity 0.2s ease;
  &:hover {
    color: var(--el-color-primary);
    background: var(--el-fill-color-light);
  }
  &:disabled {
    cursor: not-allowed;
    opacity: 0.45;
  }
}
.agent-message-action :deep(.el-icon) {
  width: 1em;
  height: 1em;
  font-size: 16px;
  line-height: 1;
}
.agent-message-action :deep(svg) {
  display: block;
  width: 1em;
  height: 1em;
}
.agent-message-action :deep(.agent-message-action__custom-icon) {
  width: 16px;
  height: 16px;
}
.agent-sender-wrap {
  position: absolute;
  right: 0;
  bottom: 24px;
  left: 0;
  z-index: 1;
  width: min(760px, calc(100% - 72px));
  padding: 0;
  margin: 0 auto;
  background: var(--admin-page-card-bg);
}

@media screen and (width <= 768px) {
  .agent-chat-panel {
    padding: 0 0 22px;
  }
  .agent-chat-panel.is-empty {
    justify-content: center;
    padding: 24px 0;
  }
  .agent-chat-header,
  .agent-chat-content,
  .agent-sender-wrap {
    width: calc(100% - 36px);
  }
  .agent-chat-empty {
    width: calc(100% - 36px);
  }
  .agent-chat-empty__title {
    font-size: 30px;
  }
  .agent-chat-empty__sender {
    width: 100%;
  }
}

@keyframes thinking-bounce {
  0%,
  80%,
  100% {
    opacity: 0.35;
    transform: translateY(0);
  }
  40% {
    opacity: 1;
    transform: translateY(-2px);
  }
}
</style>
