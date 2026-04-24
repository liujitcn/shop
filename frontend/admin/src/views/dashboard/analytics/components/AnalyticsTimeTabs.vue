<template>
  <div class="analytics-tabs" role="tablist" aria-label="时间范围选择">
    <button
      v-for="item in options"
      :key="item.value"
      type="button"
      class="analytics-tabs__item"
      :class="{ 'is-active': modelValue === item.value }"
      role="tab"
      :aria-selected="modelValue === item.value"
      @click="handleUpdate(item.value)"
    >
      {{ item.label }}
    </button>
  </div>
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
  display: inline-flex;
  gap: 2px;
  padding: 4px;
  background: var(--admin-page-card-bg-soft);
  border: 1px solid var(--admin-page-card-border-soft);
  border-radius: var(--admin-page-radius);
}

.analytics-tabs__item {
  min-width: 64px;
  height: 32px;
  padding: 0 14px;
  border: 1px solid transparent;
  border-radius: var(--admin-page-radius);
  background: transparent;
  box-sizing: border-box;
  font-size: 14px;
  font-weight: 600;
  color: var(--admin-page-text-secondary);
  cursor: pointer;
  transition:
    color 0.2s ease,
    background-color 0.2s ease,
    border-color 0.2s ease,
    box-shadow 0.2s ease;
}

.analytics-tabs__item:hover {
  color: var(--el-color-primary);
}

.analytics-tabs__item.is-active {
  color: var(--el-color-primary);
  background: color-mix(in srgb, var(--el-color-primary) 12%, #ffffff);
  border-color: color-mix(in srgb, var(--el-color-primary) 28%, transparent);
  box-shadow: 0 6px 14px -10px color-mix(in srgb, var(--el-color-primary) 60%, transparent);
}
</style>
