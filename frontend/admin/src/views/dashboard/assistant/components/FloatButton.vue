<template>
  <button
    ref="buttonRef"
    class="agent-float-button"
    :style="buttonStyle"
    type="button"
    aria-label="打开 AI 助手"
    @pointerdown="$emit('pointerdown', $event)"
    @click="$emit('click')"
  >
    <Icon />
  </button>
</template>

<script setup lang="ts" name="FloatButton">
import { ref } from "vue";
import Icon from "./Icon.vue";

defineProps<{
  /** 悬浮按钮定位样式。 */
  buttonStyle: Record<string, string>;
}>();

defineEmits<{
  /** 记录拖动起点。 */
  pointerdown: [event: PointerEvent];
  /** 打开 AI 助手弹窗。 */
  click: [];
}>();

const buttonRef = ref<HTMLButtonElement>();

/** 对外暴露按钮实例，供父组件设置指针捕获。 */
defineExpose({
  buttonRef
});
</script>

<style scoped lang="scss">
.agent-float-button {
  position: fixed;
  z-index: 3001;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 64px;
  height: 64px;
  color: #ffffff;
  cursor: grab;
  pointer-events: auto;
  touch-action: none;
  background:
    radial-gradient(circle at 72% 24%, rgb(255 255 255 / 42%) 0 9%, transparent 10%),
    linear-gradient(145deg, var(--el-color-primary-light-3), var(--el-color-primary) 54%, var(--el-color-primary-dark-2));
  border: 1px solid color-mix(in srgb, var(--el-color-primary-light-5) 72%, #ffffff 28%);
  border-radius: 50%;
  box-shadow:
    0 16px 32px color-mix(in srgb, var(--el-color-primary) 32%, transparent),
    inset 0 1px 0 rgb(255 255 255 / 40%);
  transition:
    box-shadow 0.2s ease,
    transform 0.2s ease;

  &::after {
    position: absolute;
    inset: -5px;
    pointer-events: none;
    content: "";
    border: 1px solid color-mix(in srgb, var(--el-color-primary) 24%, transparent);
    border-radius: 50%;
  }

  &:hover {
    box-shadow:
      0 20px 38px color-mix(in srgb, var(--el-color-primary) 42%, transparent),
      inset 0 1px 0 rgb(255 255 255 / 46%);
    transform: translateY(-1px);
  }

  &:active {
    cursor: grabbing;
    transform: translateY(0);
  }
}

html.dark {
  .agent-float-button {
    box-shadow:
      0 16px 32px color-mix(in srgb, var(--el-color-primary) 24%, transparent),
      inset 0 1px 0 rgb(255 255 255 / 24%);
  }
}

@media screen and (max-width: 768px) {
  .agent-float-button {
    width: 56px;
    height: 56px;
  }
}
</style>
