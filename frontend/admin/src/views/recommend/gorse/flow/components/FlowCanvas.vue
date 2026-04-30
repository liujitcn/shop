<template>
  <div class="gorse-flow-canvas-card">
    <div id="container" ref="canvasRef" class="logic-flow-view" @drop="emit('drop', $event)" @dragover.prevent />
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue";

const emit = defineEmits<{
  /** 画布容器挂载完成后通知父页面初始化 LogicFlow。 */
  ready: [element: HTMLDivElement];
  /** 将拖拽节点投放到画布。 */
  drop: [event: DragEvent];
}>();

const canvasRef = ref<HTMLDivElement>();

onMounted(() => {
  // LogicFlow 必须拿到真实 DOM 容器后才能创建实例。
  if (canvasRef.value) emit("ready", canvasRef.value);
});
</script>

<style scoped lang="scss">
.gorse-flow-canvas-card {
  min-width: 0;
  overflow: hidden;
  background: var(--admin-page-card-bg);
  border: 1px solid var(--admin-page-card-border);
  border-radius: var(--admin-page-radius);
  box-shadow: var(--admin-page-shadow);
}

.logic-flow-view {
  width: 100%;
  height: 75vh;
}
</style>
