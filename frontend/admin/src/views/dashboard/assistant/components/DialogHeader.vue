<template>
  <div class="ai-assistant-dialog-header">
    <div class="ai-assistant-dialog-actions">
      <el-tooltip :content="collapsed ? '展开会话栏' : '收起会话栏'" placement="bottom">
        <button class="ai-assistant-dialog-toggle" type="button" @click="$emit('toggleCollapse')">
          <el-icon><DArrowLeft v-if="!collapsed" /><DArrowRight v-else /></el-icon>
        </button>
      </el-tooltip>
    </div>
  </div>
</template>

<script setup lang="ts" name="DialogHeader">
import { DArrowLeft, DArrowRight } from "@element-plus/icons-vue";

defineProps<{
  /** 当前侧栏是否折叠。 */
  collapsed: boolean;
}>();

defineEmits<{
  /** 切换侧栏折叠状态。 */
  toggleCollapse: [];
}>();
</script>

<style scoped lang="scss">
.ai-assistant-dialog-header {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  min-height: 64px;
  padding: 18px 48px 16px;
  background: var(--el-bg-color);
  border-bottom: 1px solid var(--admin-page-divider-strong);
}

.ai-assistant-dialog-actions {
  display: inline-flex;
  align-items: center;
}

.ai-assistant-dialog-toggle {
  display: inline-flex;
  width: 36px;
  height: 36px;
  color: var(--admin-page-text-secondary);
  cursor: pointer;
  align-items: center;
  justify-content: center;
  background: transparent;
  border: 0;
  border-radius: 10px;

  &:hover {
    color: var(--el-color-primary);
    background: var(--el-fill-color-light);
  }
}

@media screen and (max-width: 768px) {
  .ai-assistant-dialog-header {
    min-height: 56px;
    padding: 16px 18px;
  }
}
</style>
