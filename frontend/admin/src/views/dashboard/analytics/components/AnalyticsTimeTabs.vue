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
    padding: 4px;
    background: #f8fafc;
    border: 1px solid #e8edf4;
    border-radius: 10px;
  }

  :deep(.el-tabs__nav-wrap::after) {
    display: none;
  }

  :deep(.el-tabs__item) {
    height: 32px;
    padding: 0 14px;
    border-radius: 8px;
    color: #64748b;
  }

  :deep(.el-tabs__active-bar) {
    display: none;
  }

  :deep(.el-tabs__item.is-active) {
    color: #1f2937;
    background: #fff;
  }
}
</style>
