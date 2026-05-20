<template>
  <ProDialog
    :model-value="modelValue"
    title="AI图片详情"
    width="980px"
    top="4vh"
    :confirm-loading="retrying"
    :confirm-text="canRetry ? '重试生成' : '确定'"
    cancel-text="关闭"
    @update:model-value="emit('update:modelValue', $event)"
    @confirm="handleConfirm"
    @closed="handleClosed"
  >
    <el-skeleton v-if="loading && !image" :rows="8" animated />
    <template v-else-if="image">
      <div class="ai-image-detail">
        <div class="ai-image-detail__summary">
          <div>
            <div class="ai-image-detail__title">图片 #{{ image.id }}</div>
            <div class="ai-image-detail__prompt">{{ image.prompt }}</div>
          </div>
          <el-tag :type="statusMeta.type" effect="plain">{{ statusMeta.label }}</el-tag>
        </div>

        <GeneratingPreview v-if="isGenerating" />
        <ResultGrid v-else-if="isSuccess" :items="images" />
        <el-alert
          v-else-if="canRetry"
          :title="image.error_message || 'AI图片生成失败'"
          type="error"
          show-icon
          :closable="false"
        />

        <el-descriptions class="ai-image-detail__params" :column="2" border>
          <el-descriptions-item label="模型">{{ image.model || "--" }}</el-descriptions-item>
          <el-descriptions-item label="尺寸">{{ image.size || "--" }}</el-descriptions-item>
          <el-descriptions-item label="质量">{{ image.quality || "--" }}</el-descriptions-item>
          <el-descriptions-item label="格式">{{ image.output_format || "--" }}</el-descriptions-item>
          <el-descriptions-item label="背景">{{ image.background || "--" }}</el-descriptions-item>
          <el-descriptions-item label="数量">{{ image.n || 1 }}</el-descriptions-item>
          <el-descriptions-item label="重试次数">{{ image.retry_count || 0 }}</el-descriptions-item>
          <el-descriptions-item label="创建时间">{{ formatTimestamp(image.created_at) || "--" }}</el-descriptions-item>
          <el-descriptions-item label="完成时间">{{ formatTimestamp(image.finished_at) || "--" }}</el-descriptions-item>
        </el-descriptions>
      </div>
    </template>
  </ProDialog>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from "vue";
import { ElMessage } from "element-plus";
import ProDialog from "@/components/Dialog/ProDialog.vue";
import { defAiImageService } from "@/api/base/ai_image";
import { AiImageStatus, type AiImage } from "@/rpc/base/v1/ai_image";
import GeneratingPreview from "./GeneratingPreview.vue";
import ResultGrid from "./ResultGrid.vue";
import { formatTimestamp, isGeneratingStatus, isRetryableStatus, normalizeImages, resolveAiImageStatusMeta } from "./utils";

defineOptions({
  name: "DetailDialog"
});

const props = defineProps<{
  /** 弹窗显示状态。 */
  modelValue: boolean;
  /** 当前图片ID。 */
  imageId: string;
}>();

const emit = defineEmits<{
  "update:modelValue": [value: boolean];
  refreshed: [];
}>();

const image = ref<AiImage>();
const loading = ref(false);
const retrying = ref(false);
let timer: number | undefined;

const images = computed(() => normalizeImages(image.value?.images ?? []));
const isGenerating = computed(() => isGeneratingStatus(image.value?.status));
const canRetry = computed(() => isRetryableStatus(image.value?.status));
const isSuccess = computed(() => image.value?.status === AiImageStatus.SUCCESS);
const statusMeta = computed(() => resolveAiImageStatusMeta(image.value?.status));

watch(
  () => [props.modelValue, props.imageId] as const,
  ([visible, imageId]) => {
    clearPollTimer();
    if (!visible || !imageId) return;
    void loadImage();
  },
  { immediate: true }
);

onBeforeUnmount(() => {
  clearPollTimer();
});

/** 加载图片详情，并在生成中继续轮询。 */
async function loadImage() {
  if (!props.imageId) return;
  loading.value = true;
  try {
    image.value = await defAiImageService.GetAiImage({ id: props.imageId });
    emit("refreshed");
    if (isGenerating.value && props.modelValue) {
      timer = window.setTimeout(() => void loadImage(), 2000);
    }
  } finally {
    loading.value = false;
  }
}

/** 处理弹窗确认按钮。 */
async function handleConfirm() {
  if (!canRetry.value || !image.value) {
    emit("update:modelValue", false);
    return;
  }
  retrying.value = true;
  try {
    await defAiImageService.RetryAiImage({ id: image.value.id });
    ElMessage.success("已重新提交生成");
    clearPollTimer();
    await loadImage();
  } finally {
    retrying.value = false;
  }
}

/** 弹窗关闭后清理轮询。 */
function handleClosed() {
  clearPollTimer();
  image.value = undefined;
}

/** 清理图片详情轮询。 */
function clearPollTimer() {
  if (!timer) return;
  window.clearTimeout(timer);
  timer = undefined;
}
</script>

<style scoped lang="scss">
.ai-image-detail {
  display: grid;
  gap: 18px;
}

.ai-image-detail__summary {
  display: flex;
  gap: 12px;
  align-items: flex-start;
  justify-content: space-between;
}

.ai-image-detail__title {
  font-size: 18px;
  font-weight: 700;
  line-height: 26px;
  color: var(--admin-page-text-primary);
}

.ai-image-detail__prompt {
  margin-top: 4px;
  color: var(--admin-page-text-secondary);
  overflow-wrap: anywhere;
}

.ai-image-detail__params {
  :deep(.el-descriptions__label) {
    width: 112px;
  }
}
</style>
