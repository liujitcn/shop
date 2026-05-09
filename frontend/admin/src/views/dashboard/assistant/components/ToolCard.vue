<template>
  <div class="agent-tool-card">
    <div class="agent-card-title">
      <el-icon><Operation /></el-icon>
      工具调用
    </div>
    <div v-for="tool in tools" :key="tool.name" class="agent-tool-row">
      <span class="agent-tool-status"></span>
      <span class="agent-tool-name">{{ tool.name }}</span>
      <span class="agent-tool-time">{{ tool.elapsed }}</span>
    </div>
  </div>
</template>

<script setup lang="ts" name="ToolCard">
import { Operation } from "@element-plus/icons-vue";
import type { AiAssistantTool } from "@/rpc/base/v1/ai_assistant";

defineProps<{
  /** 工具调用展示列表。 */
  tools: AiAssistantTool[];
}>();
</script>

<style scoped lang="scss">
.agent-tool-card {
  min-width: 320px;
  padding: 14px 16px;
  background: #f8fafc;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 14px;
}

.agent-card-title {
  display: flex;
  gap: 6px;
  align-items: center;
  margin-bottom: 14px;
  font-size: 14px;
  font-weight: 700;
  color: var(--admin-page-text-primary);
}

.agent-tool-row {
  display: grid;
  grid-template-columns: 14px minmax(0, 1fr) auto;
  gap: 10px;
  align-items: center;
  min-height: 28px;
  font-size: 13px;
  color: var(--admin-page-text-secondary);
}

.agent-tool-status {
  width: 8px;
  height: 8px;
  background: var(--el-color-success);
  border-radius: 50%;
}

.agent-tool-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.agent-tool-time {
  color: var(--admin-page-text-placeholder);
}
</style>
