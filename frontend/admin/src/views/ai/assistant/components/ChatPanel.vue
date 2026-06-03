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
                <template v-else-if="item.role !== 'user'">
                  <AiMarkdown :content="item.content" :streaming="item.progressState === 'streaming'" />
                  <el-collapse v-if="item.fallback_reason" class="agent-message-error__detail" accordion>
                    <el-collapse-item title="错误详情" :name="String(item.id)">
                      <pre>{{ item.fallback_reason }}</pre>
                    </el-collapse-item>
                  </el-collapse>
                </template>
                <span v-else>{{ item.content }}</span>
                <span v-if="item.progressState === 'streaming'" class="agent-thinking-dots"> <i></i><i></i><i></i> </span>
              </div>
              <div v-if="item.attachments?.length" class="agent-message-attachments">
                <Attachments :items="buildMessageAttachmentItems(item.attachments)" overflow="wrap" :hide-upload="true" />
              </div>
            </div>
          </template>
          <template #footer="{ item }">
            <div v-message-footer-width class="agent-message-footer" :class="{ 'is-user': item.role === 'user' }">
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
              <div v-if="item.role !== 'user' && hasAssistantUsage(item)" class="agent-message-runtime">
                <el-popover popper-class="agent-message-runtime-popover" placement="top-end" trigger="hover" width="260">
                  <template #reference>
                    <button class="agent-message-runtime-trigger" type="button" aria-label="查看运行明细">
                      <el-icon><DataAnalysis /></el-icon>
                      <span v-if="resolveTokenTotal(item) > 0">{{ formatCompactNumber(resolveTokenTotal(item)) }}</span>
                      <span v-if="item.duration_ms > 0">{{ formatDurationMs(item.duration_ms) }}</span>
                      <span v-if="resolveTokenTotal(item) <= 0 && item.duration_ms <= 0 && item.tools?.length">
                        {{ item.tools.length }}
                      </span>
                    </button>
                  </template>
                  <div class="agent-runtime-detail">
                    <div class="agent-runtime-detail__title">运行明细</div>
                    <div v-if="resolveTokenTotal(item) > 0" class="agent-runtime-detail__section">
                      <div class="agent-runtime-detail__section-title">Token</div>
                      <div class="agent-runtime-detail__row">
                        <span>输入</span>
                        <strong>{{ formatNumber(item.token?.input) }}</strong>
                      </div>
                      <div class="agent-runtime-detail__row">
                        <span>输出</span>
                        <strong>{{ formatNumber(item.token?.output) }}</strong>
                      </div>
                      <div v-if="(item.token?.cache ?? 0) > 0" class="agent-runtime-detail__row">
                        <span>缓存读取</span>
                        <strong>{{ formatNumber(item.token?.cache) }}</strong>
                      </div>
                      <div class="agent-runtime-detail__row is-total">
                        <span>总计</span>
                        <strong>{{ formatNumber(item.token?.total) }}</strong>
                      </div>
                    </div>
                    <div v-if="item.first_token_ms > 0 || item.duration_ms > 0" class="agent-runtime-detail__section">
                      <div class="agent-runtime-detail__section-title">耗时</div>
                      <div v-if="item.first_token_ms > 0" class="agent-runtime-detail__row">
                        <span>首 Token</span>
                        <strong>{{ formatDurationMs(item.first_token_ms) }}</strong>
                      </div>
                      <div v-if="item.duration_ms > 0" class="agent-runtime-detail__row is-total">
                        <span>总耗时</span>
                        <strong>{{ formatDurationMs(item.duration_ms) }}</strong>
                      </div>
                    </div>
                    <div v-if="item.tools?.length" class="agent-runtime-detail__section">
                      <div class="agent-runtime-detail__section-title">工具</div>
                      <el-collapse class="agent-runtime-tools">
                        <el-collapse-item
                          v-for="(tool, toolIndex) in item.tools"
                          :key="resolveToolKey(tool, toolIndex)"
                          :name="resolveToolKey(tool, toolIndex)"
                        >
                          <template #title>
                            <span class="agent-runtime-tools__title">
                              <span>{{ resolveToolTitle(tool) }}</span>
                              <span class="agent-runtime-tools__status" :class="resolveToolStatusClass(tool.status)">
                                {{ resolveToolStatusText(tool.status) }}
                              </span>
                            </span>
                          </template>
                          <div class="agent-runtime-payload">
                            <div class="agent-runtime-payload__label">REQUEST</div>
                            <pre>{{ formatToolPayload(tool.input) }}</pre>
                            <div class="agent-runtime-payload__label">RESPONSE</div>
                            <pre>{{ formatToolPayload(tool.output) }}</pre>
                          </div>
                        </el-collapse-item>
                      </el-collapse>
                    </div>
                  </div>
                </el-popover>
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
import { computed, defineAsyncComponent, h, nextTick } from "vue";
import type { Component, ObjectDirective } from "vue";
import { Attachments, BubbleList } from "vue-element-plus-x";
import type { FilesCardProps } from "vue-element-plus-x/types/FilesCard";
import { CopyDocument, DataAnalysis, Delete, Refresh } from "@element-plus/icons-vue";
import {
  type AiAssistantAttachment,
  type AiAssistantTool,
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

type MessageFooterWidthState = {
  /** 监听上方内容卡片宽度变化。 */
  observer: ResizeObserver;
  /** 当前 footer 对应的内容卡片元素。 */
  contentEl: HTMLElement;
};

const messageFooterWidthStateMap = new WeakMap<HTMLElement, MessageFooterWidthState>();

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

/** 查找 footer 所属气泡里的内容卡片元素。 */
function findMessageContentElement(el: HTMLElement) {
  return el.closest(".elx-bubble__content-wrapper")?.querySelector<HTMLElement>(".elx-bubble__content") ?? null;
}

/** 把消息 footer 宽度同步为上方内容卡片宽度，避免右侧耗时信息越出卡片边界。 */
function syncMessageFooterWidth(el: HTMLElement) {
  const contentEl = findMessageContentElement(el);
  if (!contentEl) return;

  const width = contentEl.getBoundingClientRect().width;
  if (width <= 0) return;

  el.style.setProperty("--agent-message-footer-width", `${Math.ceil(width)}px`);
}

/** 绑定内容卡片尺寸监听，覆盖流式内容、Markdown 渲染和窗口变化后的重新对齐。 */
function bindMessageFooterWidth(el: HTMLElement) {
  const contentEl = findMessageContentElement(el);
  const existingState = messageFooterWidthStateMap.get(el);
  if (!contentEl) {
    existingState?.observer.disconnect();
    messageFooterWidthStateMap.delete(el);
    el.style.removeProperty("--agent-message-footer-width");
    return;
  }

  if (existingState?.contentEl === contentEl) {
    syncMessageFooterWidth(el);
    return;
  }

  existingState?.observer.disconnect();
  const observer = new ResizeObserver(() => syncMessageFooterWidth(el));
  observer.observe(contentEl);
  messageFooterWidthStateMap.set(el, { observer, contentEl });
  syncMessageFooterWidth(el);
}

const vMessageFooterWidth: ObjectDirective<HTMLElement> = {
  mounted(el) {
    void nextTick(() => bindMessageFooterWidth(el));
  },
  updated(el) {
    void nextTick(() => bindMessageFooterWidth(el));
  },
  beforeUnmount(el) {
    messageFooterWidthStateMap.get(el)?.observer.disconnect();
    messageFooterWidthStateMap.delete(el);
  }
};

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

/** 判断是否为需要渲染错误卡片的助手失败消息。 */
function isAssistantFailedMessage(item: ChatMessageItem) {
  return item.role !== "user" && item.status === AiAssistantMessageStatus.FAILED_AAMS;
}

/** 返回助手错误摘要，优先展示服务端可读错误。 */
function resolveAssistantErrorMessage(item: ChatMessageItem) {
  const content = String(item.content ?? "").trim();
  return content || "这次回复没有成功返回，你可以直接重试刚才的问题。";
}

/** 判断助手消息是否存在最终用量或工具信息。 */
function hasAssistantUsage(item: ChatMessageItem) {
  return resolveTokenTotal(item) > 0 || item.first_token_ms > 0 || item.duration_ms > 0 || Boolean(item.tools?.length);
}

/** 返回当前气泡总 token 数。 */
function resolveTokenTotal(item: ChatMessageItem) {
  return Number(item.token?.total ?? 0);
}

/** 将毫秒耗时格式化为秒。 */
function formatDurationMs(value?: number) {
  const ms = Number(value ?? 0);
  if (ms <= 0) return "0s";
  return `${(ms / 1000).toFixed(2)}s`;
}

/** 将较大的数字压缩成适合 footer 展示的短格式。 */
function formatCompactNumber(value?: number) {
  const number = Number(value ?? 0);
  if (number >= 1_000_000) return `${(number / 1_000_000).toFixed(number >= 10_000_000 ? 0 : 1)}M`;
  if (number >= 1_000) return `${(number / 1_000).toFixed(number >= 100_000 ? 0 : 1)}K`;
  return formatNumber(number);
}

/** 格式化明细数字，保持 Token 明细易读。 */
function formatNumber(value?: number) {
  return new Intl.NumberFormat("zh-CN").format(Number(value ?? 0));
}

/** 生成工具标签稳定键。 */
function resolveToolKey(tool: AiAssistantTool, index?: number) {
  return `${tool.type || "tool"}:${tool.name || tool.title}:${index ?? 0}`;
}

/** 生成工具展示名称。 */
function resolveToolTitle(tool: AiAssistantTool) {
  return tool.title || tool.name || "工具";
}

/** 生成工具状态样式名。 */
function resolveToolStatusClass(status?: string) {
  return status === "error" ? "is-error" : "is-success";
}

/** 生成工具状态展示文案。 */
function resolveToolStatusText(status?: string) {
  return status === "error" ? "异常" : "完成";
}

/** 格式化工具原始报文，JSON 内容优先缩进展示。 */
function formatToolPayload(payload?: string) {
  const value = String(payload ?? "");
  if (!value) return "{}";
  try {
    return JSON.stringify(JSON.parse(value), null, 2);
  } catch {
    return value;
  }
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
  :deep(.elx-bubble--start .elx-bubble__content-wrapper .elx-bubble__footer) {
    box-sizing: border-box;
    max-width: 100%;
  }
  :deep(.elx-bubble--start .elx-bubble__content-wrapper .elx-bubble__content) {
    min-width: min(360px, 100%);
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
.agent-message-footer {
  box-sizing: border-box;
  display: grid;
  grid-template-columns: max-content minmax(0, 1fr);
  gap: 8px;
  align-items: center;
  width: var(--agent-message-footer-width, fit-content);
  max-width: 100%;
  min-width: 0;
  padding: 0 16px;
  margin-top: 8px;
}
.agent-message-footer.is-user {
  grid-template-columns: auto;
  justify-content: flex-end;
  padding: 0;
}
.agent-message-runtime {
  display: flex;
  justify-self: end;
  justify-content: flex-end;
  min-width: 0;
}
.agent-message-runtime-trigger {
  display: inline-flex;
  align-items: center;
  width: fit-content;
  max-width: 100%;
  height: 28px;
  gap: 6px;
  padding: 0 0 0 8px;
  overflow: hidden;
  font-size: 12px;
  line-height: 1;
  color: var(--admin-page-text-secondary);
  cursor: pointer;
  background: transparent;
  border: 0;
  border-radius: var(--admin-page-radius);
  transition:
    color 0.2s ease,
    background-color 0.2s ease;
  &:hover {
    color: var(--el-color-primary);
    background: var(--el-fill-color-light);
  }
  .el-icon {
    flex: 0 0 auto;
    font-size: 15px;
  }
  span {
    max-width: 70px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}
.agent-message-actions {
  display: inline-flex;
  gap: 8px;
  align-items: center;
  justify-content: flex-start;
  min-width: 0;
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

<style lang="scss">
.agent-message-runtime-popover {
  .agent-runtime-detail {
    min-width: 0;
    font-size: 12px;
    color: var(--admin-page-text-secondary);
  }
  .agent-runtime-detail__title {
    margin-bottom: 10px;
    font-size: 13px;
    font-weight: 700;
    line-height: 20px;
    color: var(--admin-page-text-primary);
  }
  .agent-runtime-detail__section + .agent-runtime-detail__section {
    padding-top: 10px;
    margin-top: 10px;
    border-top: 1px solid var(--el-border-color-lighter);
  }
  .agent-runtime-detail__section-title {
    margin-bottom: 6px;
    font-weight: 600;
    line-height: 18px;
    color: var(--admin-page-text-regular);
  }
  .agent-runtime-detail__row {
    display: flex;
    gap: 12px;
    align-items: center;
    justify-content: space-between;
    min-height: 22px;
    line-height: 22px;
  }
  .agent-runtime-detail__row strong {
    font-weight: 600;
    font-variant-numeric: tabular-nums;
    color: var(--admin-page-text-primary);
  }
  .agent-runtime-detail__row.is-total strong {
    color: var(--el-color-primary);
  }
  .agent-runtime-tools {
    min-width: 0;
    border: 0;
  }
  .agent-runtime-tools .el-collapse-item__header {
    height: 30px;
    min-width: 0;
    font-size: 12px;
    background: transparent;
    border: 0;
  }
  .agent-runtime-tools .el-collapse-item__wrap {
    background: transparent;
    border: 0;
  }
  .agent-runtime-tools .el-collapse-item__content {
    padding-bottom: 8px;
  }
  .agent-runtime-tools__title {
    display: inline-flex;
    gap: 8px;
    align-items: center;
    min-width: 0;
  }
  .agent-runtime-tools__title > span:first-child {
    max-width: 180px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .agent-runtime-tools__status {
    flex: 0 0 auto;
    padding: 1px 6px;
    line-height: 16px;
    color: var(--el-color-success);
    background: var(--el-color-success-light-9);
    border-radius: var(--admin-page-radius);
  }
  .agent-runtime-tools__status.is-error {
    color: var(--el-color-danger);
    background: var(--el-color-danger-light-9);
  }
  .agent-runtime-payload {
    display: flex;
    flex-direction: column;
    gap: 6px;
    min-width: 0;
  }
  .agent-runtime-payload__label {
    font-size: 11px;
    font-weight: 700;
    line-height: 16px;
    color: var(--admin-page-text-secondary);
  }
  .agent-runtime-payload pre {
    max-height: 180px;
    padding: 8px;
    margin: 0;
    overflow: auto;
    font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace;
    font-size: 11px;
    line-height: 17px;
    color: var(--admin-page-text-regular);
    white-space: pre-wrap;
    word-break: break-word;
    background: var(--el-fill-color-light);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: var(--admin-page-radius);
  }
}
</style>
