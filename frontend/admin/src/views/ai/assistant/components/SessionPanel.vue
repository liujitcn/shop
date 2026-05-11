<template>
  <aside class="agent-session-panel">
    <div class="agent-session-brand">
      <div class="agent-session-brand__main">
        <div class="agent-session-brand__copy">
          <div class="agent-session-brand__title">AI助手</div>
          <div class="agent-session-brand__desc">经营问答与辅助处理</div>
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
      label-key="label"
      :show-tooltip="true"
      :label-height="72"
      @change="handleSessionChange"
    >
      <template #label="{ item }">
        <div class="agent-session-item">
          <div class="agent-session-main">
            <div class="agent-session-name">{{ item.label }}</div>
            <div class="agent-session-meta">{{ item.summary }}</div>
          </div>
          <el-dropdown trigger="click" @command="command => handleAction(command as SessionAction, item)">
            <button class="agent-session-more" type="button" aria-label="更多操作" @click.stop>
              <el-icon><MoreFilled /></el-icon>
            </button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="rename">
                  <el-icon><EditPen /></el-icon>
                  重命名
                </el-dropdown-item>
                <el-dropdown-item command="delete">
                  <el-icon><Delete /></el-icon>
                  删除
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </template>
    </Conversations>
  </aside>
</template>

<script setup lang="ts" name="SessionPanel">
import { computed } from "vue";
import { DArrowLeft, Delete, EditPen, MoreFilled, Plus, Search } from "@element-plus/icons-vue";
import { Conversations } from "vue-element-plus-x";
import type { AiAssistantSession } from "@/rpc/base/v1/ai_assistant";

type SessionAction = "rename" | "delete";

type SessionListItem = AiAssistantSession & {
  label: string;
};

const props = defineProps<{
  /** 当前活动会话编号。 */
  active: string;
  /** 会话搜索关键词。 */
  keyword: string;
  /** 过滤后的会话列表。 */
  sessions: SessionListItem[];
}>();

const emit = defineEmits<{
  /** 更新当前活动会话。 */
  "update:active": [value: string];
  /** 更新搜索关键词。 */
  "update:keyword": [value: string];
  /** 通知父组件会话已切换。 */
  change: [item: SessionListItem];
  /** 会话操作菜单。 */
  action: [payload: { action: SessionAction; item: SessionListItem }];
  /** 创建新的会话。 */
  create: [];
  /** 收起会话栏。 */
  toggleCollapse: [];
}>();

const activeID = computed({
  get: () => props.active,
  set: value => emit("update:active", value)
});

/** 同步搜索关键词。 */
function handleKeywordUpdate(value: string) {
  emit("update:keyword", value);
}

/** 同步当前会话，并保留 Conversations 的完整变更对象。 */
function handleSessionChange(item: SessionListItem) {
  emit("change", item);
}

/** 透传会话项操作，后续由父组件接入真实重命名和删除。 */
function handleAction(action: SessionAction, item: SessionListItem) {
  emit("action", { action, item });
}

/** 通知父组件创建新的会话。 */
function handleCreateSession() {
  emit("create");
}
</script>

<style scoped lang="scss">
.agent-session-panel {
  min-width: 0;
  display: flex;
  min-height: 0;
  padding: 20px 16px;
  overflow: hidden;
  flex-direction: column;
  background: var(--admin-page-card-bg);
  border-right: 1px solid var(--admin-page-divider-strong);

  :deep(.el-input__wrapper) {
    padding: 10px 14px;
    background: var(--el-fill-color-light);
    border-radius: calc(var(--admin-page-radius) + 2px);
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
  min-width: 0;
  align-items: center;
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
  width: 32px;
  height: 32px;
  flex: 0 0 auto;
  color: var(--admin-page-text-secondary);
  cursor: pointer;
  align-items: center;
  justify-content: center;
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
  border-radius: 999px;
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
    padding: 0;
    background: transparent;
    border: 0;
    border-radius: calc(var(--admin-page-radius) + 2px);
  }

  :deep(.elx-conversations-item.is-active) {
    background: var(--el-fill-color-light);
  }
}

.agent-session-item {
  display: flex;
  gap: 8px;
  align-items: center;
  min-width: 0;
  min-height: 76px;
  padding: 14px 16px;
  border-radius: calc(var(--admin-page-radius) + 2px);
}

.agent-session-main {
  min-width: 0;
  flex: 1;
}

.agent-session-name {
  overflow: hidden;
  font-size: 14px;
  font-weight: 700;
  line-height: 22px;
  color: var(--admin-page-text-primary);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.agent-session-meta {
  margin-top: 4px;
  overflow: hidden;
  font-size: 12px;
  line-height: 18px;
  color: var(--admin-page-text-secondary);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.agent-session-more {
  display: inline-flex;
  width: 34px;
  height: 34px;
  flex: 0 0 auto;
  color: var(--admin-page-text-secondary);
  visibility: hidden;
  cursor: pointer;
  align-items: center;
  justify-content: center;
  background: #ffffff;
  border: 0;
  border-radius: 10px;
}

.agent-conversations {
  :deep(.elx-conversations-item:hover .agent-session-more),
  :deep(.elx-conversations-item.is-active .agent-session-more) {
    visibility: visible;
  }
}

@media screen and (max-width: 768px) {
  .agent-session-panel {
    display: none;
  }
}
</style>
