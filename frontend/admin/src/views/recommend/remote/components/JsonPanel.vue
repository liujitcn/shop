<template>
  <el-card class="remote-json-panel" shadow="never">
    <template #header>
      <div class="remote-json-panel__header">
        <div>
          <strong>{{ title }}</strong>
          <p v-if="description">{{ description }}</p>
        </div>
        <div class="remote-json-panel__actions">
          <slot name="actions" />
          <el-button size="small" :loading="loading" @click="handleRefresh">刷新</el-button>
        </div>
      </div>
    </template>

    <el-skeleton v-if="loading" :rows="8" animated />
    <pre v-else class="remote-json-panel__code">{{ formattedJson }}</pre>
  </el-card>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { formatRemoteJson } from "../utils";

defineOptions({
  name: "RecommendRemoteJsonPanel"
});

/** JSON 面板入参。 */
interface JsonPanelProps {
  /** 面板标题。 */
  title: string;
  /** 面板说明。 */
  description?: string;
  /** JSON 字符串。 */
  json: string;
  /** 加载状态。 */
  loading?: boolean;
}

const props = withDefaults(defineProps<JsonPanelProps>(), {
  description: "",
  loading: false
});

const emit = defineEmits<{
  /** 触发重新加载。 */
  refresh: [];
}>();

const formattedJson = computed(() => formatRemoteJson(props.json));

/** 触发父级刷新远程数据。 */
function handleRefresh() {
  emit("refresh");
}
</script>

<style scoped lang="scss">
.remote-json-panel {
  border-color: var(--el-border-color-light);
  background: var(--el-bg-color);

  &__header {
    display: flex;
    gap: 16px;
    align-items: flex-start;
    justify-content: space-between;
  }

  &__header strong {
    color: var(--el-text-color-primary);
    font-size: 16px;
  }

  &__header p {
    margin: 6px 0 0;
    color: var(--el-text-color-secondary);
    font-size: 13px;
  }

  &__actions {
    display: flex;
    flex-shrink: 0;
    gap: 8px;
    align-items: center;
  }

  &__code {
    min-height: 260px;
    max-height: 560px;
    padding: 16px;
    overflow: auto;
    color: var(--el-text-color-primary);
    font-size: 13px;
    line-height: 1.7;
    white-space: pre-wrap;
    word-break: break-word;
    background: var(--el-fill-color-lighter);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 10px;
  }
}
</style>
