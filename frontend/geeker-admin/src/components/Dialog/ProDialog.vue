<template>
  <el-dialog
    :model-value="modelValue"
    :title="title"
    :width="width"
    :top="top"
    :destroy-on-close="destroyOnClose"
    :close-on-click-modal="closeOnClickModal"
    :close-on-press-escape="closeOnPressEscape"
    v-bind="$attrs"
    @update:model-value="handleVisibleChange"
    @close="handleClose"
    @closed="handleClosed"
  >
    <slot />

    <template v-if="$slots.footer" #footer>
      <slot name="footer" />
    </template>
    <template v-else #footer>
      <div class="dialog-footer">
        <el-button @click="handleCancel"> {{ cancelText }} </el-button>
        <el-button type="primary" :loading="confirmLoading" @click="handleConfirm"> {{ confirmText }} </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts" name="ProDialog">
interface ProDialogProps {
  modelValue: boolean;
  title?: string;
  width?: string | number;
  top?: string;
  confirmText?: string;
  cancelText?: string;
  confirmLoading?: boolean;
  destroyOnClose?: boolean;
  closeOnClickModal?: boolean;
  closeOnPressEscape?: boolean;
}

withDefaults(defineProps<ProDialogProps>(), {
  title: "",
  width: "500px",
  top: "8vh",
  confirmText: "确定",
  cancelText: "取消",
  confirmLoading: false,
  destroyOnClose: false,
  closeOnClickModal: true,
  closeOnPressEscape: true
});

const emit = defineEmits<{
  "update:modelValue": [value: boolean];
  confirm: [];
  cancel: [];
  close: [];
  closed: [];
}>();

/** 同步弹窗显示状态到外部。 */
function handleVisibleChange(value: boolean) {
  emit("update:modelValue", value);
}

/** 处理点击确定按钮后的回调。 */
function handleConfirm() {
  emit("confirm");
}

/** 处理点击取消按钮后的回调，并主动关闭弹窗。 */
function handleCancel() {
  emit("update:modelValue", false);
  emit("cancel");
}

/** 处理弹窗关闭时的回调。 */
function handleClose() {
  emit("close");
}

/** 处理弹窗完全关闭后的回调。 */
function handleClosed() {
  emit("closed");
}
</script>
