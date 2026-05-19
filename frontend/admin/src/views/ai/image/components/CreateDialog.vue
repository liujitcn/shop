<template>
  <ProDialog
    :model-value="modelValue"
    title="新增AI图片"
    width="760px"
    top="5vh"
    confirm-text="提交生成"
    :confirm-loading="submitting"
    @update:model-value="emit('update:modelValue', $event)"
    @confirm="handleSubmit"
    @closed="handleClosed"
  >
    <GenerateForm ref="formRef" v-model="form" />
  </ProDialog>
</template>

<script setup lang="ts">
import { reactive, ref } from "vue";
import { ElMessage } from "element-plus";
import ProDialog from "@/components/Dialog/ProDialog.vue";
import { defAiImageService } from "@/api/base/ai_image";
import { Terminal } from "@/rpc/common/v1/enum";
import GenerateForm from "./GenerateForm.vue";
import type { GenerateFormModel } from "./types";

defineOptions({
  name: "CreateDialog"
});

defineProps<{
  /** 弹窗显示状态。 */
  modelValue: boolean;
}>();

const emit = defineEmits<{
  "update:modelValue": [value: boolean];
  created: [taskId: string];
}>();

const currentModel = "gpt-image-2";
const formRef = ref<InstanceType<typeof GenerateForm>>();
const submitting = ref(false);
const form = reactive<GenerateFormModel>(createDefaultForm());

/** 创建默认生成表单。 */
function createDefaultForm(): GenerateFormModel {
  return {
    prompt: "",
    size: "1024x1024",
    quality: "auto",
    background: "auto",
    output_format: "png",
    n: 1,
    save_output: true,
    polish_prompt: false
  };
}

/** 提交图片生成。 */
async function handleSubmit() {
  const valid = await formRef.value?.validate().catch(() => false);
  if (!valid) return;

  submitting.value = true;
  try {
    const task = await defAiImageService.CreateAiImageTask({
      prompt: form.prompt.trim(),
      model: currentModel,
      size: form.size,
      quality: form.quality,
      style: "",
      background: form.background,
      output_format: form.output_format,
      response_format: "",
      n: Number(form.n || 1),
      save_output: form.save_output,
      polish_prompt: form.polish_prompt,
      terminal: Terminal.TERMINAL_ADMIN
    });
    ElMessage.success("已提交AI图片生成");
    emit("update:modelValue", false);
    emit("created", task.id);
  } finally {
    submitting.value = false;
  }
}

/** 弹窗完全关闭后重置表单。 */
function handleClosed() {
  Object.assign(form, createDefaultForm());
  formRef.value?.clearValidate();
}
</script>
