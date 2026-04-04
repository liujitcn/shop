<template>
  <el-tabs :model-value="modelValue" class="analytics-tabs" @update:model-value="handleUpdate">
    <el-tab-pane v-for="item in options" :key="item.value" :label="item.label" :name="item.value" />
  </el-tabs>
</template>

<script setup lang="ts">
import { AnalyticsTimeType } from "@/rpc/common/analytics";

defineProps<{
  modelValue: AnalyticsTimeType;
}>();

const emit = defineEmits<{
  "update:modelValue": [value: AnalyticsTimeType];
}>();

const options = [
  { label: "本周", value: AnalyticsTimeType.WEEK },
  { label: "本月", value: AnalyticsTimeType.MONTH },
  { label: "本年", value: AnalyticsTimeType.YEAR }
];

function handleUpdate(value: string | number) {
  emit("update:modelValue", value as AnalyticsTimeType);
}
</script>

<style scoped lang="scss">
.analytics-tabs {
  :deep(.el-tabs__header) {
    margin: 0;
  }

  :deep(.el-tabs__nav) {
    display: flex;
    gap: 2px;
    padding: 4px 6px;
    background: var(--admin-page-card-bg-soft);
    border: 1px solid var(--admin-page-card-border-soft);
    border-radius: 10px;
  }

  :deep(.el-tabs__nav-wrap::after) {
    display: none;
  }

  :deep(.el-tabs__item) {
    height: 32px;
    padding: 0 14px;
    border: 1px solid transparent;
    border-radius: 8px;
    box-sizing: border-box;
    font-weight: 600;
    color: var(--admin-page-text-secondary);
    transition:
      color 0.2s ease,
      background-color 0.2s ease,
      border-color 0.2s ease;
  }

  :deep(.el-tabs__active-bar) {
    display: none;
  }

  :deep(.el-tabs__item.is-active) {
    color: var(--el-color-primary);
    background: color-mix(in srgb, var(--el-color-primary) 12%, #ffffff);
    border-color: color-mix(in srgb, var(--el-color-primary) 28%, transparent);
  }

  :deep(.el-tabs__item:hover) {
    color: var(--el-color-primary);
  }
}
</style>
