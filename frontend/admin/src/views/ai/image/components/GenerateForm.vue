<template>
  <ProForm ref="formRef" class="ai-image-form" :model="model" :fields="formFields" label-position="top" label-width="0">
    <template #promptActions>
      <div class="ai-image-prompt-actions">
        <el-button text type="primary" :icon="MagicStick" :loading="polishing" @click="handlePolishPrompt">AI 润色</el-button>
        <el-checkbox v-model="model.polish_prompt">生成前自动润色</el-checkbox>
      </div>
    </template>
  </ProForm>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { ElMessage } from "element-plus";
import { MagicStick } from "@element-plus/icons-vue";
import ProForm from "@/components/ProForm/index.vue";
import type { ProFormField, ProFormInstance } from "@/components/ProForm/interface";
import { defAiImageService } from "@/api/base/ai_image";
import {
  imageBackgroundOptions,
  imageFormatOptions,
  imageQualityOptions,
  imageSizeOptions,
  type GenerateFormModel
} from "./types";

defineOptions({
  name: "GenerateForm"
});

const model = defineModel<GenerateFormModel>({ required: true });
const formRef = ref<ProFormInstance>();
const polishing = ref(false);

const formFields: ProFormField[] = [
  {
    prop: "prompt",
    label: "提示词",
    component: "textarea",
    colSpan: 24,
    rules: [{ required: true, message: "请输入图片提示词", trigger: "blur" }],
    props: {
      rows: 7,
      maxlength: 1200,
      showWordLimit: true,
      resize: "none",
      placeholder: "例如：一张清爽明亮的有机蔬菜礼盒主图，白色背景，自然光，适合电商商品首图"
    }
  },
  { prop: "prompt_actions", label: "", component: "slot", slotName: "promptActions", colSpan: 24 },
  { prop: "size", label: "尺寸", component: "select", options: imageSizeOptions, colSpan: 8 },
  { prop: "quality", label: "质量", component: "select", options: imageQualityOptions, colSpan: 8 },
  { prop: "output_format", label: "格式", component: "select", options: imageFormatOptions, colSpan: 8 },
  {
    prop: "n",
    label: "数量",
    component: "input-number",
    colSpan: 8,
    props: { min: 1, max: 4, controlsPosition: "right" }
  },
  { prop: "background", label: "背景", component: "segmented", options: imageBackgroundOptions, colSpan: 16 }
];

/** 润色当前图片提示词。 */
async function handlePolishPrompt() {
  const prompt = model.value.prompt.trim();
  if (!prompt) {
    ElMessage.warning("请输入需要润色的提示词");
    return;
  }

  polishing.value = true;
  try {
    const response = await defAiImageService.PolishAiImagePrompt({
      prompt,
      scene: "商城后台图片生成"
    });
    model.value.prompt = response.prompt || prompt;
    ElMessage.success("提示词已润色");
  } finally {
    polishing.value = false;
  }
}

/** 校验生成表单。 */
async function validate() {
  return formRef.value?.validate();
}

/** 清理生成表单校验状态。 */
function clearValidate() {
  formRef.value?.clearValidate();
}

defineExpose({
  validate,
  clearValidate
});
</script>

<style scoped lang="scss">
.ai-image-prompt-actions {
  display: flex;
  gap: 12px;
  align-items: center;
  justify-content: space-between;
  margin-top: -12px;
}

@media (max-width: 720px) {
  .ai-image-prompt-actions {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
