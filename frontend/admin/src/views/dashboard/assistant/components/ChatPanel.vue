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
            <ToolCard v-if="item.kind === 'tool'" :tools="item.tools || []" />
            <ConfirmCard
              v-else-if="item.kind === 'confirm'"
              :title="item.confirmTitle || '待确认操作'"
              :lines="item.confirmLines || []"
            />
            <div v-else class="agent-message-body">
              <div class="agent-message-content">{{ item.content }}</div>
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
import type { AiAssistantAttachment, AiAssistantMessage, AiAssistantSession } from "@/rpc/base/v1/ai_assistant";
import ConfirmCard from "./ConfirmCard.vue";
import ToolCard from "./ToolCard.vue";
import XSender from "./XSender.vue";

type ChatMessageItem = AiAssistantMessage & {
  placement: "start" | "end";
  variant?: "filled" | "borderless" | "outlined" | "shadow";
  shape?: "round" | "corner";
  maxWidth?: string;
  confirmTitle?: string;
  confirmLines?: string[];
};

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
</script>

<style scoped lang="scss">
.agent-chat-panel {
  display: flex;
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
}

.agent-chat-empty {
  display: flex;
  width: min(1100px, calc(100% - 96px));
  margin: 0 auto;
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
  padding: 8px 0 0;
}

.agent-message-content {
  line-height: 24px;
  white-space: pre-wrap;
}

.agent-message-body {
  display: flex;
  gap: 10px;
  flex-direction: column;
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
  width: min(760px, calc(100% - 72px));
  margin: 0 auto;
  padding: 18px 0 0;
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
</style>
