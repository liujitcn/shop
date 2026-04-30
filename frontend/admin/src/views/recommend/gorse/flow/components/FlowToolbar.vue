<template>
  <div class="gorse-flow-toolbar">
    <div class="gorse-flow-palette">
      <div
        v-for="item in paletteNodes"
        :key="item.type"
        class="gorse-flow-palette-node"
        draggable="true"
        @dragstart="emit('paletteDragStart', $event, item.type)"
      >
        <i class="material-icons gorse-flow-palette-icon">{{ item.icon }}</i>
        <span>{{ item.label }}</span>
      </div>
    </div>
    <div class="gorse-flow-actions">
      <el-button @click="emit('clear')">清空</el-button>
      <el-button type="primary" @click="emit('save')">保存</el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { PaletteNode } from "../types";

interface FlowToolbarProps {
  /** 组件面板节点列表。 */
  paletteNodes: PaletteNode[];
}

defineProps<FlowToolbarProps>();

const emit = defineEmits<{
  /** 开始拖拽组件面板节点。 */
  paletteDragStart: [event: DragEvent, type: string];
  /** 清空流程配置。 */
  clear: [];
  /** 保存流程配置。 */
  save: [];
}>();
</script>

<style scoped lang="scss">
.gorse-flow-toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: center;
  justify-content: space-between;
  width: 100%;
}

.gorse-flow-palette {
  display: flex;
  flex: 1;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
  min-width: 0;
}

.gorse-flow-palette-node {
  display: inline-flex;
  gap: 8px;
  align-items: center;
  padding: 8px 10px;
  color: var(--admin-page-text-primary);
  cursor: grab;
  user-select: none;
  background: var(--admin-page-card-bg);
  border: 1px solid var(--admin-page-card-border-soft);
  border-radius: var(--admin-page-radius);
  box-shadow: 0 1px 2px rgb(15 23 42 / 4%);

  &:active {
    cursor: grabbing;
  }
}

.gorse-flow-palette-icon {
  color: var(--el-color-primary);
  font-size: 20px;
}

.gorse-flow-actions {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  margin-left: 16px;
}

@media (max-width: 768px) {
  .gorse-flow-toolbar {
    align-items: flex-start;
  }

  .gorse-flow-actions {
    width: 100%;
    margin-left: 0;
  }
}
</style>
