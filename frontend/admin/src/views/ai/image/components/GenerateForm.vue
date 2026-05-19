<template>
  <el-form ref="formRef" :model="model" :rules="rules" label-position="top" class="ai-image-form">
    <el-form-item label="提示词" prop="prompt">
      <el-input
        v-model="model.prompt"
        type="textarea"
        :rows="7"
        maxlength="1200"
        show-word-limit
        resize="none"
        placeholder="例如：一张清爽明亮的有机蔬菜礼盒主图，白色背景，自然光，适合电商商品首图"
      />
      <div class="ai-image-prompt-actions">
        <el-button text type="primary" :icon="MagicStick" :loading="polishing" @click="handlePolishPrompt">AI 润色</el-button>
        <el-checkbox v-model="model.polish_prompt">生成前自动润色</el-checkbox>
      </div>
    </el-form-item>

    <div class="ai-image-form__grid">
      <el-form-item label="尺寸">
        <el-select v-model="model.size">
          <el-option v-for="item in sizeOptions" :key="item.value" :label="item.label" :value="item.value" />
        </el-select>
      </el-form-item>
      <el-form-item label="质量">
        <el-select v-model="model.quality">
          <el-option v-for="item in qualityOptions" :key="item.value" :label="item.label" :value="item.value" />
        </el-select>
      </el-form-item>
      <el-form-item label="格式">
        <el-select v-model="model.output_format">
          <el-option v-for="item in formatOptions" :key="item.value" :label="item.label" :value="item.value" />
        </el-select>
      </el-form-item>
    </div>

    <div class="ai-image-form__grid ai-image-form__grid--compact">
      <el-form-item label="数量">
        <el-input-number v-model="model.n" :min="1" :max="4" controls-position="right" />
      </el-form-item>
      <el-form-item label="背景">
        <el-segmented v-model="model.background" :options="backgroundOptions" />
      </el-form-item>
    </div>

    <el-form-item>
      <el-checkbox v-model="model.save_output">保存到素材目录</el-checkbox>
    </el-form-item>
  </el-form>
</template>

<script setup lang="ts">
import { reactive, ref } from "vue";
import { ElMessage, type FormInstance, type FormRules } from "element-plus";
import { MagicStick } from "@element-plus/icons-vue";
import { defAiImageService } from "@/api/base/ai_image";
import type { GenerateFormModel } from "./types";

defineOptions({
  name: "GenerateForm"
});

const model = defineModel<GenerateFormModel>({ required: true });
const formRef = ref<FormInstance>();
const polishing = ref(false);

const rules = reactive<FormRules<GenerateFormModel>>({
  prompt: [{ required: true, message: "请输入图片提示词", trigger: "blur" }]
});

const sizeOptions = [
  { label: "1024 x 1024", value: "1024x1024" },
  { label: "1536 x 1024", value: "1536x1024" },
  { label: "1024 x 1536", value: "1024x1536" },
  { label: "自动", value: "auto" }
];

const qualityOptions = [
  { label: "自动", value: "auto" },
  { label: "高", value: "high" },
  { label: "中", value: "medium" },
  { label: "低", value: "low" },
  { label: "HD", value: "hd" },
  { label: "标准", value: "standard" }
];

const formatOptions = [
  { label: "PNG", value: "png" },
  { label: "JPEG", value: "jpeg" },
  { label: "WEBP", value: "webp" }
];

const backgroundOptions = [
  { label: "自动", value: "auto" },
  { label: "透明", value: "transparent" },
  { label: "不透明", value: "opaque" }
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
.ai-image-form {
  display: grid;
  gap: 4px;
}

.ai-image-form__grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}

.ai-image-form__grid--compact {
  grid-template-columns: minmax(120px, 180px) minmax(0, 1fr);
}

.ai-image-prompt-actions {
  display: flex;
  gap: 12px;
  align-items: center;
  justify-content: space-between;
  margin-top: 8px;
}

@media (max-width: 720px) {
  .ai-image-form__grid,
  .ai-image-form__grid--compact {
    grid-template-columns: 1fr;
  }
}
</style>
