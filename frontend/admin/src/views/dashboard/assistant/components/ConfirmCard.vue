<template>
  <div class="agent-confirm-card">
    <div class="agent-confirm-title">
      <el-icon><Warning /></el-icon>
      <span>{{ title }}</span>
    </div>
    <div class="agent-confirm-lines">
      <div v-for="line in lines" :key="line">{{ line }}</div>
    </div>
    <div class="agent-confirm-actions">
      <el-button size="small" plain>拒绝</el-button>
      <el-button size="small" type="warning">确认</el-button>
    </div>
  </div>
</template>

<script setup lang="ts" name="ConfirmCard">
import { Warning } from "@element-plus/icons-vue";

defineProps<{
  /** 确认卡标题。 */
  title: string;
  /** 确认卡内容行。 */
  lines: string[];
}>();
</script>

<style scoped lang="scss">
.agent-confirm-card {
  min-width: 320px;
  padding: 16px;
  background: #fff9eb;
  border: 1px solid #f3dfad;
  border-radius: 14px;
}

.agent-confirm-title {
  display: flex;
  gap: 6px;
  align-items: center;
  font-size: 14px;
  font-weight: 700;
  color: var(--el-color-warning-dark-2);
}

.agent-confirm-lines {
  margin-top: 12px;
  font-size: 13px;
  line-height: 24px;
  color: var(--admin-page-text-secondary);
}

.agent-confirm-actions {
  display: flex;
  gap: 8px;
  margin-top: 14px;
}
</style>
