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
    <el-skeleton v-if="loading && !task" :rows="8" animated />
    <template v-else-if="task">
      <div class="ai-image-detail">
        <div class="ai-image-detail__summary">
          <div>
            <div class="ai-image-detail__title">图片 #{{ task.id }}</div>
            <div class="ai-image-detail__prompt">{{ task.prompt }}</div>
          </div>
          <el-tag :type="statusMeta.type" effect="plain">{{ statusMeta.label }}</el-tag>
        </div>

        <GeneratingPreview v-if="isGenerating" />
        <ResultGrid v-else-if="task.status === TaskStatus.Success" :items="images" />
        <el-alert
          v-else-if="task.status === TaskStatus.Failed || task.status === TaskStatus.Timeout"
          :title="task.error_message || 'AI图片生成失败'"
          type="error"
          show-icon
          :closable="false"
        />

        <el-descriptions class="ai-image-detail__params" :column="2" border>
          <el-descriptions-item label="模型">{{ task.model || "--" }}</el-descriptions-item>
          <el-descriptions-item label="批次">{{ task.request_id || "--" }}</el-descriptions-item>
          <el-descriptions-item label="尺寸">{{ task.size || "--" }}</el-descriptions-item>
          <el-descriptions-item label="质量">{{ task.quality || "--" }}</el-descriptions-item>
          <el-descriptions-item label="格式">{{ task.output_format || "--" }}</el-descriptions-item>
          <el-descriptions-item label="背景">{{ task.background || "--" }}</el-descriptions-item>
          <el-descriptions-item label="数量">{{ task.n || 1 }}</el-descriptions-item>
          <el-descriptions-item label="重试次数">{{ task.retry_count || 0 }}</el-descriptions-item>
          <el-descriptions-item label="创建时间">{{ formatTimestamp(task.created_at) || "--" }}</el-descriptions-item>
          <el-descriptions-item label="完成时间">{{ formatTimestamp(task.finished_at) || "--" }}</el-descriptions-item>
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
import type { AiImageTask } from "@/rpc/base/v1/ai_image";
import GeneratingPreview from "./GeneratingPreview.vue";
import ResultGrid from "./ResultGrid.vue";
import { TaskStatus } from "./types";
import { formatTimestamp, normalizeImages } from "./utils";

defineOptions({
  name: "DetailDialog"
});

const props = defineProps<{
  /** 弹窗显示状态。 */
  modelValue: boolean;
  /** 当前图片ID。 */
  taskId: string;
}>();

const emit = defineEmits<{
  "update:modelValue": [value: boolean];
  refreshed: [];
}>();

const task = ref<AiImageTask>();
const loading = ref(false);
const retrying = ref(false);
let timer: number | undefined;

const images = computed(() => normalizeImages(task.value?.images ?? []));
const isGenerating = computed(() => task.value?.status === TaskStatus.Pending || task.value?.status === TaskStatus.Running);
const canRetry = computed(() => task.value?.status === TaskStatus.Failed || task.value?.status === TaskStatus.Timeout);
const statusMeta = computed(() => statusMap[task.value?.status ?? TaskStatus.Pending] ?? statusMap[TaskStatus.Pending]);

const statusMap: Record<number, { label: string; type: "primary" | "success" | "warning" | "danger" | "info" }> = {
  [TaskStatus.Pending]: { label: "待处理", type: "info" },
  [TaskStatus.Running]: { label: "生成中", type: "warning" },
  [TaskStatus.Success]: { label: "成功", type: "success" },
  [TaskStatus.Failed]: { label: "失败", type: "danger" },
  [TaskStatus.Timeout]: { label: "超时", type: "danger" }
};

watch(
  () => [props.modelValue, props.taskId] as const,
  ([visible, taskId]) => {
    clearPollTimer();
    if (!visible || !taskId) return;
    void loadTask();
  },
  { immediate: true }
);

onBeforeUnmount(() => {
  clearPollTimer();
});

/** 加载图片详情，并在生成中继续轮询。 */
async function loadTask() {
  if (!props.taskId) return;
  loading.value = true;
  try {
    task.value = await defAiImageService.GetAiImageTask({ id: props.taskId });
    emit("refreshed");
    if (isGenerating.value && props.modelValue) {
      timer = window.setTimeout(() => void loadTask(), 2000);
    }
  } finally {
    loading.value = false;
  }
}

/** 处理弹窗确认按钮。 */
async function handleConfirm() {
  if (!canRetry.value || !task.value) {
    emit("update:modelValue", false);
    return;
  }
  retrying.value = true;
  try {
    task.value = await defAiImageService.RetryAiImageTask({ id: task.value.id });
    ElMessage.success("已重新提交生成");
    clearPollTimer();
    timer = window.setTimeout(() => void loadTask(), 1000);
  } finally {
    retrying.value = false;
  }
}

/** 弹窗关闭后清理轮询。 */
function handleClosed() {
  clearPollTimer();
  task.value = undefined;
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
