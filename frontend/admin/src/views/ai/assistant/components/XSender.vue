<template>
  <div class="agent-sender-wrap">
    <BaseXSender
      ref="senderRef"
      class="agent-sender"
      variant="updown"
      submit-type="enter"
      placeholder="输入问题、处理建议或指定商品 / 订单"
      :loading="sending"
      :clearable="true"
      :mention-config="mentionConfig"
      :trigger-config="triggerConfig"
      :select-config="selectConfig"
      :tip-config="false"
      @change="handleInputChange"
      @submit="handleSubmit"
      @paste-file="handlePasteFile"
    >
      <template v-if="selectedAttachments.length" #header>
        <Attachments
          class="agent-attachments"
          :items="attachmentItems"
          overflow="wrap"
          :hide-upload="true"
          @delete-card="handleDeleteCard"
        />
      </template>

      <template #prefix>
        <div class="agent-prefix-actions">
          <el-popover placement="top-start" :width="236" trigger="click" popper-class="agent-sender-popover">
            <template #reference>
              <button class="agent-icon-button" type="button" aria-label="上传附件">
                <el-icon><Paperclip /></el-icon>
              </button>
            </template>
            <div class="agent-popover-card">
              <div class="agent-popover-title">上传文件或图片</div>
              <div class="agent-popover-desc">可先补充截图、表格或文档，后续再接统一附件接口。</div>
              <button class="agent-popover-action" type="button" @click="handleSelectAttachment">选择本地文件</button>
            </div>
          </el-popover>

          <el-popover placement="top" :width="220" trigger="click" popper-class="agent-sender-popover">
            <template #reference>
              <button class="agent-icon-button" type="button" aria-label="语音输入">
                <el-icon><Microphone /></el-icon>
              </button>
            </template>
            <div class="agent-popover-card">
              <div class="agent-popover-title">语音输入</div>
              <div class="agent-popover-desc">麦克风入口先预留，首版暂不接浏览器录音与权限申请。</div>
            </div>
          </el-popover>

          <span class="agent-action-text">{{ actionHintText }}</span>
        </div>
      </template>

      <template #action-list>
        <el-tooltip content="发送" placement="top">
          <button
            class="agent-send-button"
            type="button"
            :disabled="sending || isSubmitDisabled"
            aria-label="发送"
            @click="handleSubmit"
          >
            <el-icon v-if="!sending"><Promotion /></el-icon>
            <el-icon v-else class="is-loading"><Loading /></el-icon>
          </button>
        </el-tooltip>
      </template>
    </BaseXSender>

    <input ref="fileInputRef" class="agent-file-input" type="file" multiple @change="handleFileChange" />
  </div>
</template>

<script setup lang="ts" name="XSender">
import { computed, ref } from "vue";
import { Attachments, XSender as BaseXSender } from "vue-element-plus-x";
import type { FilesCardProps, FilesType } from "vue-element-plus-x/types/components/FilesCard/types";
import { Loading, Microphone, Paperclip, Promotion } from "@element-plus/icons-vue";
import { ElMessage } from "element-plus";
import { defFileService } from "@/api/base/file";
import type { AiAssistantAttachment } from "@/rpc/base/v1/ai_assistant";

type SubmitPayload = {
  text: string;
  attachments: AiAssistantAttachment[];
};

type MentionItem = {
  id: string;
  name: string;
  avatar?: string | URL;
  pinyin?: string;
};

type MentionConfig = {
  dialogTitle: string;
  callEvery?: boolean;
  everyText?: string;
  asyncMatch?: (matchStr: string) => Promise<MentionItem[]>;
  emptyText?: string;
  options?: MentionItem[];
};

type TriggerConfig = {
  dialogTitle: string;
  keyMap?: string[];
  key: string;
  options: Array<{ id: string; name: string; pinyin?: string }>;
};

type SelectConfig = {
  dialogTitle: string;
  key: string;
  options: Array<{ id: string; name: string; preview?: string | URL }>;
  multiple?: boolean;
  emptyText?: string;
  showSearch?: boolean;
  placeholder?: string;
  searchEmptyText?: string;
};

defineProps<{
  /** 消息发送加载状态。 */
  sending: boolean;
}>();

const emit = defineEmits<{
  /** 提交输入器内容。 */
  submit: [payload: SubmitPayload];
}>();

const senderRef = ref<InstanceType<typeof BaseXSender>>();
const fileInputRef = ref<HTMLInputElement>();
const inputText = ref("");
const selectedAttachments = ref<AiAssistantAttachment[]>([]);
const uploading = ref(false);

const mentionConfig: MentionConfig = {
  dialogTitle: "智能提示",
  options: []
};

const triggerConfig: TriggerConfig[] = [];

const selectConfig: SelectConfig[] = [];

const actionHintText = computed(() => {
  if (selectedAttachments.value.length) return `已选 ${selectedAttachments.value.length} 个附件`;
  return "可上传附件后继续提问";
});

const attachmentItems = computed<FilesCardProps[]>(() =>
  selectedAttachments.value.map(item => ({
    uid: item.id,
    name: item.name,
    fileSize: item.size,
    url: item.url,
    fileType: resolveFileType(item.name),
    showDelIcon: true
  }))
);

const isSubmitDisabled = computed(() => {
  return !inputText.value.trim() && selectedAttachments.value.length === 0;
});

/** 读取输入内容并发送给父组件。 */
function handleSubmit() {
  if (uploading.value) return;
  const trimmedText = inputText.value.trim();
  if (!trimmedText && selectedAttachments.value.length === 0) return;

  emit("submit", {
    text: trimmedText || "请结合附件内容继续分析",
    attachments: [...selectedAttachments.value]
  });
  inputText.value = "";
  selectedAttachments.value = [];
  senderRef.value?.clear();
  resetFileInput();
}

/** 同步输入器内部文本，保证发送按钮禁用态能实时响应。 */
function handleInputChange() {
  inputText.value = senderRef.value?.getModelValue().text ?? "";
}

/** 打开本地文件选择框。 */
function handleSelectAttachment() {
  fileInputRef.value?.click();
}

/** 将本地文件列表上传后同步到输入器附件区。 */
async function handleFileChange(event: Event) {
  const target = event.target as HTMLInputElement;
  await uploadAttachments(Array.from(target.files ?? []));
}

/** 处理粘贴文件，和点击上传保持同一份附件状态。 */
async function handlePasteFile(firstFile: File, fileList: FileList) {
  await uploadAttachments([firstFile, ...Array.from(fileList).slice(1)]);
}

/** 上传附件并按文件地址去重。 */
async function uploadAttachments(files: File[]) {
  if (!files.length) return;
  uploading.value = true;
  try {
    const response = await defFileService.MultiUploadFile(files, "assistant");
    const fileMap = new Map<string, File[]>();
    files.forEach(file => {
      const group = fileMap.get(file.name) ?? [];
      group.push(file);
      fileMap.set(file.name, group);
    });
    const attachmentMap = new Map(selectedAttachments.value.map(item => [item.url || item.id, item]));
    response?.files?.forEach(item => {
      const matchedFile = fileMap.get(item.name || "")?.shift();
      const attachment = buildAttachmentItem(item.url || "", item.name || "", matchedFile);
      attachmentMap.set(attachment.url || attachment.id, attachment);
    });
    selectedAttachments.value = Array.from(attachmentMap.values());
  } catch {
    ElMessage.error("附件上传失败");
  } finally {
    uploading.value = false;
    resetFileInput();
  }
}

/** 根据文件后缀推断文件卡片类型，优先复用组件自带图标表现。 */
function resolveFileType(fileName: string): FilesType {
  const extension = fileName.split(".").pop()?.toLowerCase() ?? "";
  if (["png", "jpg", "jpeg", "gif", "webp", "svg"].includes(extension)) return "image";
  if (["xls", "xlsx", "csv"].includes(extension)) return "excel";
  if (["doc", "docx"].includes(extension)) return "word";
  if (["ppt", "pptx"].includes(extension)) return "ppt";
  if (extension === "pdf") return "pdf";
  if (["zip", "rar", "7z"].includes(extension)) return "zip";
  if (["mp3", "wav", "m4a"].includes(extension)) return "audio";
  if (["mp4", "mov", "avi"].includes(extension)) return "video";
  if (["md", "markdown"].includes(extension)) return "mark";
  if (["txt", "log"].includes(extension)) return "txt";
  return "file";
}

/** 根据上传结果构建附件展示项。 */
function buildAttachmentItem(url: string, name: string, file?: File): AiAssistantAttachment {
  return {
    id: url || `${name}-${file?.size ?? 0}-${file?.lastModified ?? Date.now()}`,
    name,
    size: file?.size ?? 0,
    url,
    mime_type: file?.type ?? ""
  };
}

/** 删除单个已选附件。 */
function handleRemoveAttachment(attachmentID: string) {
  selectedAttachments.value = selectedAttachments.value.filter(item => item.id !== attachmentID);
  if (!selectedAttachments.value.length) resetFileInput();
}

/** 复用 Attachments 删除事件，保持附件状态与组件展示同步。 */
function handleDeleteCard(item: { uid?: string | number }) {
  if (!item.uid) return;
  handleRemoveAttachment(String(item.uid));
}

/** 清空文件输入框，保证重复选择同名文件时仍能触发 change。 */
function resetFileInput() {
  if (fileInputRef.value) fileInputRef.value.value = "";
}
</script>

<style scoped lang="scss">
.agent-sender-wrap {
  width: 100%;
}

.agent-sender {
  :deep(.elx-x-sender) {
    border-radius: calc(var(--admin-page-radius) + 2px);
    box-shadow: none;
  }

  :deep(.elx-x-sender__content) {
    padding: 8px 10px 10px;
    background: var(--admin-page-card-bg);
    border: 1px solid var(--admin-page-card-border);
    border-radius: calc(var(--admin-page-radius) + 2px);
  }

  :deep(.elx-x-sender__content--variant-updown) {
    gap: 10px;
  }

  :deep(.chat-rich-text) {
    min-height: 64px;
    max-height: 120px;
    padding: 8px 10px;
    font-size: 14px;
    line-height: 22px;
  }

  :deep(.chat-placeholder-wrap) {
    padding: 8px 10px;
    font-size: 14px;
    font-weight: 400;
  }

  :deep(.elx-x-sender__updown-action-list) {
    align-items: center;
    justify-content: space-between;
  }

  :deep(.elx-x-sender__action-list) {
    height: auto;
  }
}

.agent-attachments {
  padding: 12px 12px 0;

  :deep(.elx-attachments-list) {
    gap: 8px;
  }

  :deep(.elx-files-card) {
    max-width: 220px;
  }
}

.agent-prefix-actions {
  display: flex;
  gap: 10px;
  align-items: center;
  min-height: 36px;
}

.agent-icon-button,
.agent-send-button {
  display: inline-flex;
  width: 34px;
  height: 34px;
  color: var(--admin-page-text-secondary);
  cursor: pointer;
  align-items: center;
  justify-content: center;
  background: #ffffff;
  border: 1px solid var(--el-border-color);
  border-radius: 50%;
  transition:
    color 0.2s ease,
    border-color 0.2s ease,
    background-color 0.2s ease,
    opacity 0.2s ease;

  &:hover {
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
    border-color: var(--el-color-primary-light-5);
  }
}

.agent-send-button {
  color: var(--el-color-primary);
  background: #effaf7;
  border-color: #b7ece1;

  &:disabled {
    cursor: not-allowed;
    opacity: 0.5;
  }
}

.agent-action-text {
  font-size: 12px;
  line-height: 18px;
  color: var(--admin-page-text-secondary);
}

.is-loading {
  animation: rotating 1s linear infinite;
}

.agent-file-input {
  display: none;
}

:global(.agent-sender-popover) {
  padding: 0 !important;
  border-radius: 16px !important;
}

.agent-popover-card {
  padding: 14px;
}

.agent-popover-title {
  font-size: 14px;
  font-weight: 700;
  color: var(--admin-page-text-primary);
}

.agent-popover-desc {
  margin-top: 6px;
  font-size: 12px;
  line-height: 20px;
  color: var(--admin-page-text-secondary);
}

.agent-popover-action {
  width: 100%;
  height: 36px;
  margin-top: 12px;
  font-size: 13px;
  font-weight: 600;
  color: var(--el-color-primary);
  cursor: pointer;
  background: var(--el-color-primary-light-9);
  border: 0;
  border-radius: 10px;
}

:global(.agent-sender-popover.el-popover) {
  box-shadow: 0 14px 36px rgb(15 23 42 / 10%);
}

@media screen and (max-width: 768px) {
  .agent-prefix-actions {
    gap: 8px;
  }

  .agent-action-text {
    display: none;
  }
}

@keyframes rotating {
  from {
    transform: rotate(0deg);
  }

  to {
    transform: rotate(360deg);
  }
}
</style>
