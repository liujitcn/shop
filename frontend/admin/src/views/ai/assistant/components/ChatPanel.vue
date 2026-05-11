<template>
  <main class="agent-chat-panel" :class="{ 'is-empty': isEmptyState }">
    <template v-if="isEmptyState">
      <div class="agent-chat-empty">
        <div class="agent-chat-empty__title">{{ welcomeTitle }}</div>
        <div class="agent-chat-empty__desc">可直接提问，也可以上传附件后继续分析当前系统内容。</div>
        <div class="agent-chat-empty__sender">
          <XSender :sending="sending" @submit="handleSubmit" />
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
              <div class="agent-message-content" :class="{ 'is-thinking': item.progressState === 'streaming' }">
                <span>{{ item.content }}</span>
                <span v-if="item.progressState === 'streaming'" class="agent-thinking-dots"> <i></i><i></i><i></i> </span>
              </div>
              <div v-if="item.attachments?.length" class="agent-message-attachments">
                <div v-for="attachment in item.attachments" :key="attachment.id" class="agent-message-attachment">
                  <el-icon><Paperclip /></el-icon>
                  <span>{{ attachment.name }}</span>
                </div>
              </div>
            </div>
          </template>
        </BubbleList>
      </div>

      <div class="agent-sender-wrap">
        <XSender :sending="sending" @submit="handleSubmit" />
      </div>
    </template>
  </main>
</template>

<script setup lang="ts" name="ChatPanel">
import { computed } from "vue";
import { BubbleList } from "vue-element-plus-x";
import { Paperclip } from "@element-plus/icons-vue";
import type { AiAssistantAttachment, AiAssistantSession } from "@/rpc/base/v1/ai_assistant";
import XSender from "./XSender.vue";
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
  if (hour < 12) return "上午好，我是 AI 助手";
  if (hour < 18) return "下午好，我是 AI 助手";
  return "晚上好，我是 AI 助手";
});

/** 读取输入框内容并提交给父组件。 */
function handleSubmit(payload: SubmitPayload) {
  emit("submit", payload);
}

/** 统一回复来源标签配色。 */
function resolveTagClass(tone?: ReplySourceTag["tone"]) {
  return tone ? `is-${tone}` : "";
}
</script>

<style scoped lang="scss">
.agent-chat-panel {
  box-sizing: border-box;
  display: flex;
  position: relative;
  height: 100%;
  min-width: 0;
  min-height: 0;
  padding: 20px 0 24px;
  overflow: hidden;
  flex-direction: column;
  background: var(--admin-page-card-bg);
}

.agent-chat-panel.is-empty {
  justify-content: center;
  padding: 0;
}

.agent-chat-header {
  flex: 0 0 auto;
  display: flex;
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
  min-height: 0;
  width: min(960px, calc(100% - 72px));
  margin: 0 auto;
  overflow: hidden;
  padding-bottom: 168px;
}

.agent-chat-empty {
  display: flex;
  flex: 1;
  width: min(1100px, calc(100% - 96px));
  margin: 0 auto;
  min-height: 0;
  align-items: center;
  justify-content: center;
  flex-direction: column;
}

.agent-chat-empty__title {
  font-size: clamp(28px, 4vw, 44px);
  font-weight: 700;
  line-height: 1.25;
  text-align: center;
  color: var(--admin-page-text-primary);
}

.agent-chat-empty__desc {
  margin-top: 14px;
  font-size: 14px;
  line-height: 24px;
  text-align: center;
  color: var(--admin-page-text-secondary);
}

.agent-chat-empty__sender {
  width: min(980px, 100%);
  margin-top: 28px;
}

.agent-message-list {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding: 8px 0 24px;
}

.agent-message-content {
  line-height: 24px;
  white-space: pre-wrap;
}

.agent-message-content.is-thinking {
  display: inline-flex;
  gap: 8px;
  align-items: center;
}

.agent-message-body {
  display: flex;
  gap: 10px;
  flex-direction: column;
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
  border-radius: 999px;
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
    width: 6px;
    height: 6px;
    display: inline-block;
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
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.agent-message-attachment {
  display: inline-flex;
  gap: 6px;
  align-items: center;
  padding: 7px 12px;
  font-size: 12px;
  color: var(--admin-page-text-secondary);
  background: rgb(255 255 255 / 78%);
  border: 1px solid var(--el-border-color-light);
  border-radius: 999px;
}

.agent-sender-wrap {
  position: absolute;
  right: 0;
  bottom: 24px;
  left: 0;
  width: min(760px, calc(100% - 72px));
  margin: 0 auto;
  padding: 0;
  z-index: 1;
  background: var(--admin-page-card-bg);
}

@media screen and (max-width: 768px) {
  .agent-chat-panel {
    padding: 0 0 22px;
  }

  .agent-chat-panel.is-empty {
    padding: 24px 0;
    justify-content: center;
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
