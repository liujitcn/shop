<template>
  <aside class="agent-session-panel">
    <div class="agent-session-brand">
      <div class="agent-session-brand__main">
        <div class="agent-session-brand__copy">
          <div class="agent-session-brand__title">AI 助手</div>
          <div class="agent-session-brand__desc">通用问答与内容处理</div>
        </div>
      </div>
      <el-tooltip content="收起会话栏" placement="top">
        <button class="agent-session-toggle" type="button" aria-label="收起会话栏" @click="$emit('toggleCollapse')">
          <el-icon><DArrowLeft /></el-icon>
        </button>
      </el-tooltip>
    </div>
    <div class="agent-session-brand-divider"></div>
    <el-input
      :model-value="keyword"
      placeholder="搜索对话"
      clearable
      :prefix-icon="Search"
      @update:model-value="handleKeywordUpdate"
    />
    <div class="agent-divider"></div>
    <div class="agent-panel-header">
      <div class="agent-panel-title">最近对话</div>
      <button class="agent-panel-create" type="button" aria-label="新建会话" @click="handleCreateSession">
        <el-icon><Plus /></el-icon>
        <span>新建</span>
      </button>
    </div>
    <Conversations
      v-model:active="activeID"
      class="agent-conversations"
      :items="sessions"
      row-key="id"
      label-key="title"
      :menu="sessionMenus"
      :show-tooltip="true"
      :show-built-in-menu="true"
      show-built-in-menu-type="hover"
      :label-height="72"
      @change="handleSessionChange"
      @menu-command="handleMenuCommand"
    >
      <template #label="{ item }">
        <div class="agent-session-item">
          <div class="agent-session-main">
            <div class="agent-session-name">{{ item.title }}</div>
            <div class="agent-session-meta">{{ item.summary }}</div>
          </div>
        </div>
      </template>
    </Conversations>
  </aside>
</template>

<script setup lang="ts" name="SessionPanel">
import { computed } from "vue";
import { Conversations } from "vue-element-plus-x";
import type { ConversationMenu, ConversationMenuCommand } from "vue-element-plus-x/types/Conversations";
import { DArrowLeft, Delete, EditPen, Plus, Search } from "@element-plus/icons-vue";
import type { AiAssistantSession } from "@/rpc/base/v1/ai_assistant_session";

type SessionAction = "rename" | "delete";

const props = defineProps<{
  /** 当前活动会话编号。 */
  active: string;
  /** 会话搜索关键词。 */
  keyword: string;
  /** 过滤后的会话列表。 */
  sessions: AiAssistantSession[];
}>();

const emit = defineEmits<{
  /** 更新当前活动会话。 */
  "update:active": [value: string];
  /** 更新搜索关键词。 */
  "update:keyword": [value: string];
  /** 通知父组件会话已切换。 */
  change: [item: AiAssistantSession];
  /** 会话操作菜单。 */
  action: [payload: { action: SessionAction; item: AiAssistantSession }];
  /** 创建新的会话。 */
  create: [];
  /** 收起会话栏。 */
  toggleCollapse: [];
}>();

const activeID = computed({
  get: () => props.active,
  set: value => emit("update:active", value)
});

const sessionMenus: ConversationMenu[] = [
  {
    label: "重命名",
    key: "rename",
    icon: EditPen,
    command: "rename"
  },
  {
    label: "删除",
    key: "delete",
    icon: Delete,
    divided: true,
    command: "delete"
  }
];

/** 同步搜索关键词。 */
function handleKeywordUpdate(value: string) {
  emit("update:keyword", value);
}

/** 同步当前会话，并保留 Conversations 的完整变更对象。 */
function handleSessionChange(item: AiAssistantSession) {
  emit("change", item);
}

/** 透传 Conversations 内置菜单操作，后续由父组件接入真实重命名和删除。 */
function handleMenuCommand(command: ConversationMenuCommand, item: AiAssistantSession) {
  const action = String(command ?? "") as SessionAction;
  if (action !== "rename" && action !== "delete") return;
  emit("action", { action, item });
}

/** 通知父组件创建新的会话。 */
function handleCreateSession() {
  emit("create");
}
</script>

<style scoped lang="scss">
.agent-session-panel {
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
  padding: 20px 16px;
  overflow: hidden;
  background: var(--admin-page-card-bg);
  border-right: 1px solid var(--admin-page-divider-strong);
  :deep(.el-input__wrapper) {
    padding: 10px 14px;
    background: var(--el-fill-color-light);
    border-radius: var(--admin-page-radius);
    box-shadow: none;
  }
}
.agent-session-brand {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 64px;
  padding: 0 6px;
}
.agent-session-brand__main {
  display: flex;
  align-items: center;
  min-width: 0;
}
.agent-session-brand__copy {
  min-width: 0;
}
.agent-session-brand__title {
  font-size: 16px;
  font-weight: 700;
  line-height: 24px;
  color: var(--admin-page-text-primary);
}
.agent-session-brand__desc {
  margin-top: 2px;
  font-size: 13px;
  font-weight: 600;
  line-height: 20px;
  color: var(--admin-page-text-secondary);
}
.agent-session-toggle {
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  color: var(--admin-page-text-secondary);
  cursor: pointer;
  background: var(--el-fill-color-light);
  border: 0;
  border-radius: var(--admin-page-radius);
  transition:
    color 0.2s ease,
    background-color 0.2s ease;
  &:hover {
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
  }
}
.agent-session-brand-divider {
  height: 1px;
  margin: 16px 0 20px;
  background: var(--el-border-color-lighter);
}
.agent-panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin: 18px 0 14px;
}
.agent-panel-title {
  font-size: 14px;
  font-weight: 700;
  color: var(--admin-page-text-primary);
}
.agent-panel-create {
  display: inline-flex;
  gap: 4px;
  align-items: center;
  height: 28px;
  padding: 0 10px;
  font-size: 12px;
  font-weight: 600;
  color: var(--el-color-primary);
  cursor: pointer;
  background: var(--el-color-primary-light-9);
  border: 0;
  border-radius: var(--admin-page-radius);
  transition:
    color 0.2s ease,
    background-color 0.2s ease;
  &:hover {
    color: var(--el-color-primary-dark-2);
    background: var(--el-color-primary-light-8);
  }
}
.agent-divider {
  height: 1px;
  margin: 20px 0;
  background: var(--el-border-color-lighter);
}
.agent-conversations {
  flex: 1;
  margin-top: 0;
  :deep(.elx-conversations-list) {
    gap: 10px;
  }
  :deep(.elx-conversations-item) {
    position: relative;
    padding: 0;
    background: transparent;
    border: 1px solid transparent;
    border-radius: var(--admin-page-radius);
    transition:
      background-color 0.2s ease,
      border-color 0.2s ease,
      box-shadow 0.2s ease;
  }
  :deep(.elx-conversations-item:not(:last-child)::after) {
    position: absolute;
    right: 16px;
    bottom: -6px;
    left: 16px;
    height: 1px;
    content: "";
    background: var(--el-border-color-lighter);
  }
  :deep(.elx-conversations-item--active) {
    background: var(--el-color-primary-light-9);
    border-color: var(--el-color-primary-light-5);
    box-shadow: inset 3px 0 0 var(--el-color-primary);
  }
  :deep(.elx-conversations-item--active .agent-session-name) {
    color: var(--el-color-primary);
  }
  :deep(.elx-conversations-item--active .agent-session-meta) {
    color: var(--admin-page-text-primary);
  }
}
.agent-session-item {
  display: flex;
  gap: 8px;
  align-items: center;
  min-width: 0;
  min-height: 76px;
  padding: 14px 16px;
  border-radius: var(--admin-page-radius);
}
.agent-session-main {
  flex: 1;
  min-width: 0;
}
.agent-session-name {
  overflow: hidden;
  text-overflow: ellipsis;
  font-size: 14px;
  font-weight: 700;
  line-height: 22px;
  color: var(--admin-page-text-primary);
  white-space: nowrap;
}
.agent-session-meta {
  margin-top: 4px;
  overflow: hidden;
  text-overflow: ellipsis;
  font-size: 12px;
  line-height: 18px;
  color: var(--admin-page-text-secondary);
  white-space: nowrap;
}

@media screen and (width <= 768px) {
  .agent-session-panel {
    display: none;
  }
}
</style>
