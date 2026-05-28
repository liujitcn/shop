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
                  'is-user': item.role === 'user'
                }"
              >
                <AiMarkdown v-if="item.role !== 'user'" :content="item.content" :streaming="item.progressState === 'streaming'" />
                <span v-else>{{ item.content }}</span>
                <span v-if="item.progressState === 'streaming'" class="agent-thinking-dots"> <i></i><i></i><i></i> </span>
              </div>
              <div v-if="item.attachments?.length" class="agent-message-attachments">
                <Attachments :items="buildMessageAttachmentItems(item.attachments)" overflow="wrap" :hide-upload="true" />
              </div>
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
import { computed, defineAsyncComponent } from "vue";
import { Attachments, BubbleList } from "vue-element-plus-x";
import type { FilesCardProps } from "vue-element-plus-x/types/FilesCard";
import type { AiAssistantAttachment, AiAssistantSession } from "@/rpc/base/v1/ai_assistant_session";
import XSender from "./XSender.vue";

// AI Markdown 渲染器依赖较重，仅在真正出现助手消息时再加载。
const AiMarkdown = defineAsyncComponent(() => import("./AiMarkdown.vue"));
import { buildAssistantAttachmentFileCard } from "../attachment";
import type { ChatMessageItem, ReplySourceTag } from "../types";

type SubmitPayload = {
  text: string;
  attachments: AiAssistantAttachment[];
};

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
.agent-message-body {
  display: flex;
  flex-direction: column;
  gap: 10px;
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
