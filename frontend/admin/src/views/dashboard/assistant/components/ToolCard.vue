<template>
  <div class="agent-tool-card">
    <div class="agent-card-title">
      <el-icon><Operation /></el-icon>
      工具调用
    </div>
    <div v-if="!normalizedTools.length" class="agent-tool-empty">当前没有可展示的工具结果</div>
    <div v-for="tool in normalizedTools" :key="tool.key" class="agent-tool-row">
      <span class="agent-tool-status" :class="`is-${tool.status}`"></span>
      <div class="agent-tool-main">
        <div class="agent-tool-head">
          <span class="agent-tool-name">{{ tool.name }}</span>
          <span class="agent-tool-state" :class="`is-${tool.status}`">{{ tool.statusText }}</span>
        </div>
        <span v-if="tool.summaryText" class="agent-tool-summary">{{ tool.summaryText }}</span>
      </div>
      <span class="agent-tool-time">{{ tool.elapsed }}</span>
    </div>
  </div>
</template>

<script setup lang="ts" name="ToolCard">
import { computed } from "vue";
import { Operation } from "@element-plus/icons-vue";
import type { AiAssistantTool } from "@/rpc/base/v1/ai_assistant";

type ToolStatus = "success" | "failed";

type ToolViewItem = AiAssistantTool & {
  key: string;
  status: ToolStatus;
  statusText: string;
  summaryText: string;
};

const props = defineProps<{
  /** 工具调用展示列表。 */
  tools: AiAssistantTool[];
}>();

/** 归一化工具展示字段，兼容当前 proto 里较轻量的结构。 */
const normalizedTools = computed<ToolViewItem[]>(() =>
  (props.tools ?? []).map((tool, index) => {
    const summaryText = String(tool.summary ?? "").trim();
    const isFailed = /失败|异常|error|failed/i.test(summaryText);
    return {
      ...tool,
      key: `${tool.name || "tool"}-${index}`,
      status: isFailed ? "failed" : "success",
      statusText: isFailed ? "执行失败" : "执行完成",
      summaryText: summaryText || (isFailed ? "工具执行失败，请查看上下文后重试。" : "工具已执行完成，当前未返回摘要信息。")
    };
  })
);
</script>

<style scoped lang="scss">
.agent-tool-card {
  min-width: 320px;
  padding: 14px 16px;
  background: var(--admin-page-card-bg);
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

.agent-tool-empty {
  font-size: 13px;
  line-height: 22px;
  color: var(--admin-page-text-secondary);
}

.agent-tool-row {
  display: grid;
  grid-template-columns: 14px minmax(0, 1fr) auto;
  gap: 10px;
  align-items: center;
  min-height: 40px;
  padding: 10px 0;
  font-size: 13px;
  color: var(--admin-page-text-secondary);

  & + & {
    border-top: 1px solid var(--el-border-color-lighter);
  }
}

.agent-tool-main {
  display: flex;
  min-width: 0;
  flex-direction: column;
}

.agent-tool-head {
  display: flex;
  gap: 8px;
  align-items: center;
  min-width: 0;
}

.agent-tool-status {
  width: 8px;
  height: 8px;
  background: var(--el-color-success);
  border-radius: 50%;
}

.agent-tool-status.is-failed {
  background: var(--el-color-danger);
}

.agent-tool-name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--admin-page-text-primary);
}

.agent-tool-state {
  padding: 2px 8px;
  font-size: 11px;
  line-height: 16px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
}

.agent-tool-state.is-success {
  color: var(--el-color-success);
}

.agent-tool-state.is-failed {
  color: var(--el-color-danger);
}

.agent-tool-summary {
  margin-top: 2px;
  font-size: 12px;
  line-height: 20px;
  color: var(--admin-page-text-placeholder);
  white-space: pre-wrap;
  word-break: break-word;
}

.agent-tool-time {
  align-self: start;
  white-space: nowrap;
  color: var(--admin-page-text-placeholder);
}
</style>
