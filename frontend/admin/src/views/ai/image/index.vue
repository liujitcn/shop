<template>
  <section class="ai-image-page">
    <div class="ai-image-workspace">
      <el-card class="ai-image-panel" shadow="never">
        <template #header>
          <div class="ai-image-panel__header">
            <div>
              <h1 class="ai-image-panel__title">AI图片</h1>
              <p class="ai-image-panel__subtitle">生成商品图、活动素材和内容配图。</p>
            </div>
            <el-tag effect="plain" type="success">{{ currentModel }}</el-tag>
          </div>
        </template>

        <el-form ref="formRef" :model="form" :rules="rules" label-position="top" class="ai-image-form">
          <el-form-item label="提示词" prop="prompt">
            <el-input
              v-model="form.prompt"
              type="textarea"
              :rows="7"
              maxlength="1200"
              show-word-limit
              resize="none"
              placeholder="例如：一张清爽明亮的有机蔬菜礼盒主图，白色背景，自然光，适合电商商品首图"
            />
            <div class="ai-image-prompt-actions">
              <el-button text type="primary" :icon="MagicStick" :loading="polishing" @click="handlePolishPrompt">
                AI 润色
              </el-button>
              <el-checkbox v-model="form.polish_prompt">生成前自动润色</el-checkbox>
            </div>
          </el-form-item>

          <div class="ai-image-form__grid">
            <el-form-item label="尺寸">
              <el-select v-model="form.size">
                <el-option v-for="item in sizeOptions" :key="item.value" :label="item.label" :value="item.value" />
              </el-select>
            </el-form-item>
            <el-form-item label="质量">
              <el-select v-model="form.quality">
                <el-option v-for="item in qualityOptions" :key="item.value" :label="item.label" :value="item.value" />
              </el-select>
            </el-form-item>
            <el-form-item label="格式">
              <el-select v-model="form.output_format">
                <el-option v-for="item in formatOptions" :key="item.value" :label="item.label" :value="item.value" />
              </el-select>
            </el-form-item>
          </div>

          <div class="ai-image-form__grid ai-image-form__grid--compact">
            <el-form-item label="数量">
              <el-input-number v-model="form.n" :min="1" :max="4" controls-position="right" />
            </el-form-item>
            <el-form-item label="背景">
              <el-segmented v-model="form.background" :options="backgroundOptions" />
            </el-form-item>
          </div>

          <div class="ai-image-form__footer">
            <el-checkbox v-model="form.save_output">保存到素材目录</el-checkbox>
            <div class="ai-image-form__actions">
              <el-button :icon="RefreshLeft" @click="handleReset">重置</el-button>
              <el-button type="primary" :icon="MagicStick" :loading="generating" @click="handleGenerate">生成图片</el-button>
            </div>
          </div>
        </el-form>
      </el-card>
    </div>

    <div class="ai-image-results">
      <div class="ai-image-results__toolbar">
        <div>
          <h2>生成结果</h2>
          <p v-if="lastPrompt">{{ lastPrompt }}</p>
        </div>
        <el-button v-if="images.length" :icon="Delete" @click="handleClearResults">清空</el-button>
      </div>

      <div v-if="generating" class="ai-image-loading" aria-live="polite">
        <div class="ai-image-loading__preview">
          <span class="ai-image-loading__grain" />
          <span class="ai-image-loading__horizon" />
          <span class="ai-image-loading__subject ai-image-loading__subject--main" />
          <span class="ai-image-loading__subject ai-image-loading__subject--side" />
          <el-icon class="ai-image-loading__icon"><Picture /></el-icon>
          <span class="ai-image-loading__spark ai-image-loading__spark--one" />
          <span class="ai-image-loading__spark ai-image-loading__spark--two" />
          <span class="ai-image-loading__spark ai-image-loading__spark--three" />
          <span class="ai-image-loading__scan" />
        </div>
        <div class="ai-image-loading__stages" aria-hidden="true">
          <span v-for="stage in generatingStages" :key="stage">{{ stage }}</span>
        </div>
        <div class="ai-image-loading__dots" aria-hidden="true">
          <span />
          <span />
          <span />
        </div>
        <div class="ai-image-loading__text">
          <strong>正在生成图片</strong>
          <small>模型正在构图、补光和渲染细节</small>
        </div>
      </div>

      <el-empty v-else-if="!images.length" description="暂无图片" />

      <div v-else class="ai-image-grid">
        <article v-for="item in images" :key="item.key" class="ai-image-card">
          <el-image
            class="ai-image-card__media"
            :src="item.previewUrl"
            fit="cover"
            :preview-src-list="[item.previewUrl]"
            preview-teleported
          />
          <div class="ai-image-card__meta">
            <div class="ai-image-card__name">{{ item.name || "AI图片" }}</div>
            <div class="ai-image-card__tags">
              <el-tag size="small" effect="plain">{{ item.mime_type || "image/png" }}</el-tag>
              <el-tag v-if="item.saved" size="small" effect="plain" type="success">已保存</el-tag>
            </div>
          </div>
          <div v-if="item.storage_path || item.request_id" class="ai-image-card__trace">
            <span v-if="item.request_id">批次：{{ item.request_id }}</span>
            <span v-if="item.storage_path">目录：{{ item.storage_path }}</span>
          </div>
          <div class="ai-image-card__actions">
            <el-tooltip content="复制地址" placement="top">
              <el-button circle :icon="CopyDocument" @click="handleCopyUrl(item)" />
            </el-tooltip>
            <el-tooltip content="下载图片" placement="top">
              <el-button circle :icon="Download" @click="handleDownload(item)" />
            </el-tooltip>
          </div>
        </article>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { reactive, ref } from "vue";
import { ElMessage, type FormInstance, type FormRules } from "element-plus";
import { CopyDocument, Delete, Download, MagicStick, Picture, RefreshLeft } from "@element-plus/icons-vue";
import { defAiImageService, type GenerateAiImagePayload } from "@/api/base/ai_image";
import type { AiImage, GenerateAiImageRequest } from "@/rpc/base/v1/ai_image";
import { formatSrc } from "@/utils/utils";

defineOptions({
  name: "AiImage"
});

/** AI 图片生成表单。 */
type AiImageForm = Omit<GenerateAiImageRequest, "model" | "style" | "response_format">;

/** 图片卡片展示项。 */
type AiImageItem = AiImage & {
  /** 前端渲染稳定键。 */
  key: string;
  /** 补齐静态资源域名后的预览地址。 */
  previewUrl: string;
};

const formRef = ref<FormInstance>();
const generating = ref(false);
const polishing = ref(false);
const images = ref<AiImageItem[]>([]);
const lastPrompt = ref("");
const currentModel = "gpt-image-2";

const form = reactive<AiImageForm>({
  prompt: "",
  size: "1024x1024",
  quality: "auto",
  background: "auto",
  output_format: "png",
  n: 1,
  save_output: true,
  polish_prompt: false
});

const rules: FormRules<AiImageForm> = {
  prompt: [{ required: true, message: "请输入图片提示词", trigger: "blur" }]
};

const sizeOptions = [
  { label: "1024 × 1024", value: "1024x1024" },
  { label: "1536 × 1024", value: "1536x1024" },
  { label: "1024 × 1536", value: "1024x1536" },
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

/** 图片生成过程阶段文案。 */
const generatingStages = ["构图", "补光", "细节", "出图"];

/** 生成 AI 图片。 */
async function handleGenerate() {
  if (!formRef.value) return;
  const valid = await formRef.value.validate().catch(() => false);
  if (!valid) return;

  generating.value = true;
  try {
    const response = await defAiImageService.GenerateAiImage(buildGeneratePayload());
    const normalizedImages = normalizeImages(response.images ?? []);
    images.value = normalizedImages;
    lastPrompt.value = response.prompt || form.prompt.trim();
    if (response.prompt && response.prompt !== form.prompt.trim()) {
      form.prompt = response.prompt;
    }
    ElMessage.success(`已生成 ${normalizedImages.length} 张图片`);
  } finally {
    generating.value = false;
  }
}

/** 重置图片生成表单。 */
function handleReset() {
  form.prompt = "";
  form.size = "1024x1024";
  form.quality = "auto";
  form.background = "auto";
  form.output_format = "png";
  form.n = 1;
  form.save_output = true;
  form.polish_prompt = false;
  formRef.value?.clearValidate();
}

/** 润色当前图片提示词。 */
async function handlePolishPrompt() {
  const prompt = form.prompt.trim();
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
    form.prompt = response.prompt || prompt;
    ElMessage.success("提示词已润色");
  } finally {
    polishing.value = false;
  }
}

/** 清空当前生成结果。 */
function handleClearResults() {
  images.value = [];
  lastPrompt.value = "";
}

/** 复制图片地址。 */
async function handleCopyUrl(item: AiImageItem) {
  await navigator.clipboard.writeText(item.previewUrl);
  ElMessage.success("图片地址已复制");
}

/** 下载图片结果。 */
function handleDownload(item: AiImageItem) {
  const link = document.createElement("a");
  link.href = item.previewUrl;
  link.download = item.name || `ai-image.${resolveImageExtension(item.mime_type)}`;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
}

/** 标准化图片结果，补齐预览地址和稳定键。 */
function normalizeImages(list: AiImage[]): AiImageItem[] {
  return list.map((item, index) => {
    const url = String(item.url ?? "");
    return {
      ...item,
      key: `${index}-${url || item.name || Date.now()}`,
      previewUrl: formatSrc(url)
    };
  });
}

/** 构造图片生成提交参数，只传当前模型实际会消费的字段。 */
function buildGeneratePayload(): GenerateAiImagePayload {
  return {
    prompt: form.prompt.trim(),
    model: currentModel,
    size: form.size,
    quality: form.quality,
    background: form.background,
    output_format: form.output_format,
    n: Number(form.n || 1),
    save_output: form.save_output,
    polish_prompt: form.polish_prompt
  };
}

/** 按 MIME 类型推断下载扩展名。 */
function resolveImageExtension(mimeType?: string) {
  switch (String(mimeType ?? "").toLowerCase()) {
    case "image/jpeg":
      return "jpg";
    case "image/webp":
      return "webp";
    default:
      return "png";
  }
}
</script>

<style scoped lang="scss">
.ai-image-page {
  display: grid;
  min-height: 100%;
  gap: 18px;
  grid-template-columns: minmax(320px, 420px) minmax(0, 1fr);
}

.ai-image-workspace {
  display: flex;
  flex-direction: column;
  gap: 14px;
  min-width: 0;
}

.ai-image-panel {
  background: var(--admin-page-card-bg);
  border: 1px solid var(--admin-page-card-border);
  border-radius: var(--admin-page-radius);
  box-shadow: var(--admin-page-shadow);

  :deep(.el-card__header) {
    padding: 18px 20px;
    border-color: var(--admin-page-divider);
  }

  :deep(.el-card__body) {
    padding: 20px;
  }
}

.ai-image-panel__header {
  display: flex;
  gap: 12px;
  align-items: flex-start;
  justify-content: space-between;
}

.ai-image-panel__title,
.ai-image-results__toolbar h2 {
  margin: 0;
  font-size: 20px;
  line-height: 28px;
  color: var(--admin-page-text-primary);
}

.ai-image-panel__subtitle,
.ai-image-results__toolbar p {
  margin: 6px 0 0;
  font-size: 13px;
  line-height: 20px;
  color: var(--admin-page-text-secondary);
}

.ai-image-form {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.ai-image-form__grid {
  display: grid;
  gap: 12px;
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.ai-image-prompt-actions {
  display: flex;
  gap: 12px;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  margin-top: 8px;
}

.ai-image-form__footer {
  display: flex;
  gap: 12px;
  align-items: center;
  justify-content: space-between;
  padding-top: 6px;
}

.ai-image-form__actions {
  display: flex;
  gap: 10px;
  align-items: center;
}

.ai-image-results {
  min-width: 0;
  padding: 20px;
  background: var(--admin-page-card-bg);
  border: 1px solid var(--admin-page-card-border);
  border-radius: var(--admin-page-radius);
  box-shadow: var(--admin-page-shadow);
}

.ai-image-results__toolbar {
  display: flex;
  gap: 16px;
  align-items: flex-start;
  justify-content: space-between;
  margin-bottom: 18px;
}

.ai-image-results__toolbar p {
  max-width: 760px;
}

.ai-image-loading {
  display: flex;
  flex-direction: column;
  gap: 14px;
  align-items: center;
  justify-content: center;
  min-height: 360px;
  padding: 28px;
  color: var(--admin-page-text-secondary);
}

.ai-image-loading__preview {
  position: relative;
  display: grid;
  place-items: center;
  width: min(100%, 400px);
  aspect-ratio: 4 / 3;
  overflow: hidden;
  background:
    linear-gradient(110deg, transparent 0%, rgb(255 255 255 / 42%) 46%, transparent 64%), var(--admin-page-card-bg-muted);
  background-size:
    220% 100%,
    100% 100%;
  border: 1px solid var(--admin-page-card-border-soft);
  border-radius: var(--admin-page-radius);
  animation: ai-image-loading-shimmer 1.8s ease-in-out infinite;
}

.ai-image-loading__grain,
.ai-image-loading__horizon,
.ai-image-loading__subject,
.ai-image-loading__spark,
.ai-image-loading__scan {
  position: absolute;
  content: "";
}

.ai-image-loading__grain {
  inset: 0;
  background-image:
    linear-gradient(90deg, rgb(255 255 255 / 10%) 1px, transparent 1px),
    linear-gradient(0deg, rgb(255 255 255 / 10%) 1px, transparent 1px);
  background-size: 28px 28px;
  opacity: 0.46;
  animation: ai-image-loading-grid 5.4s linear infinite;
}

.ai-image-loading__horizon {
  right: 9%;
  bottom: 22%;
  left: 9%;
  height: 2px;
  background: var(--admin-page-card-border-soft);
  opacity: 0.82;
}

.ai-image-loading__subject {
  border-radius: 999px;
  animation: ai-image-loading-breathe 2.4s ease-in-out infinite;
}

.ai-image-loading__subject--main {
  bottom: 18%;
  left: 16%;
  width: 38%;
  height: 22%;
  background: var(--admin-page-accent-soft-bg);
}

.ai-image-loading__subject--side {
  top: 18%;
  right: 18%;
  width: 52px;
  height: 52px;
  background: var(--admin-page-card-bg-soft);
  border: 1px solid var(--admin-page-card-border-soft);
  animation-delay: 0.3s;
}

.ai-image-loading__icon {
  z-index: 1;
  font-size: 68px;
  color: var(--admin-page-text-secondary);
  opacity: 0.58;
}

.ai-image-loading__spark {
  width: 8px;
  height: 8px;
  background: var(--admin-page-accent-soft-border);
  border-radius: 999px;
  opacity: 0;
  animation: ai-image-loading-spark 2.6s ease-in-out infinite;
}

.ai-image-loading__spark--one {
  top: 25%;
  left: 24%;
}

.ai-image-loading__spark--two {
  top: 36%;
  right: 30%;
  animation-delay: 0.45s;
}

.ai-image-loading__spark--three {
  right: 22%;
  bottom: 30%;
  animation-delay: 0.9s;
}

.ai-image-loading__scan {
  inset: 0;
  background: linear-gradient(180deg, transparent 0%, var(--admin-page-accent-soft-bg) 50%, transparent 100%);
  opacity: 0.72;
  transform: translateY(-100%);
  animation: ai-image-loading-scan 2.2s ease-in-out infinite;
}

.ai-image-loading__stages {
  display: grid;
  width: min(100%, 400px);
  gap: 8px;
  grid-template-columns: repeat(4, minmax(0, 1fr));
}

.ai-image-loading__stages span {
  min-width: 0;
  padding: 7px 8px;
  overflow: hidden;
  font-size: 12px;
  line-height: 18px;
  color: var(--admin-page-text-secondary);
  text-align: center;
  text-overflow: ellipsis;
  white-space: nowrap;
  background: var(--admin-page-card-bg-soft);
  border: 1px solid var(--admin-page-card-border-soft);
  border-radius: var(--admin-page-radius);
  animation: ai-image-loading-stage 2.8s ease-in-out infinite;
}

.ai-image-loading__stages span:nth-child(2) {
  animation-delay: 0.35s;
}

.ai-image-loading__stages span:nth-child(3) {
  animation-delay: 0.7s;
}

.ai-image-loading__stages span:nth-child(4) {
  animation-delay: 1.05s;
}

.ai-image-loading__dots {
  display: flex;
  gap: 10px;
  align-items: center;
  justify-content: center;
}

.ai-image-loading__dots span {
  width: 10px;
  height: 10px;
  background: var(--admin-page-text-secondary);
  border-radius: 999px;
  opacity: 0.38;
  animation: ai-image-loading-dot 1.2s ease-in-out infinite;
}

.ai-image-loading__dots span:nth-child(2) {
  animation-delay: 0.16s;
}

.ai-image-loading__dots span:nth-child(3) {
  animation-delay: 0.32s;
}

.ai-image-loading__text {
  display: flex;
  flex-direction: column;
  gap: 4px;
  align-items: center;
  text-align: center;
}

.ai-image-loading__text strong {
  font-size: 15px;
  line-height: 22px;
  color: var(--admin-page-text-primary);
}

.ai-image-loading__text small {
  font-size: 12px;
  line-height: 18px;
  color: var(--admin-page-text-secondary);
}

.ai-image-grid {
  display: grid;
  gap: 16px;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
}

.ai-image-card {
  position: relative;
  min-width: 0;
  overflow: hidden;
  background: var(--admin-page-card-bg-soft);
  border: 1px solid var(--admin-page-card-border-soft);
  border-radius: var(--admin-page-radius);
}

.ai-image-card__media {
  display: block;
  width: 100%;
  aspect-ratio: 1 / 1;
  background: var(--admin-page-card-bg-muted);
}

.ai-image-card__meta {
  display: flex;
  gap: 10px;
  align-items: center;
  justify-content: space-between;
  padding: 12px;
}

.ai-image-card__name {
  min-width: 0;
  overflow: hidden;
  font-size: 13px;
  font-weight: 700;
  color: var(--admin-page-text-primary);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ai-image-card__tags {
  display: flex;
  flex-shrink: 0;
  gap: 6px;
}

.ai-image-card__trace {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 0 12px 12px;
  overflow-wrap: anywhere;
  font-size: 12px;
  line-height: 18px;
  color: var(--admin-page-text-secondary);
}

.ai-image-card__actions {
  position: absolute;
  top: 10px;
  right: 10px;
  display: flex;
  gap: 8px;
}

@media (width <= 1180px) {
  .ai-image-page {
    grid-template-columns: 1fr;
  }
}

@keyframes ai-image-loading-shimmer {
  0% {
    background-position:
      140% 0,
      0 0;
  }

  100% {
    background-position:
      -80% 0,
      0 0;
  }
}

@keyframes ai-image-loading-scan {
  0%,
  20% {
    transform: translateY(-100%);
  }

  80%,
  100% {
    transform: translateY(100%);
  }
}

@keyframes ai-image-loading-grid {
  0% {
    background-position:
      0 0,
      0 0;
  }

  100% {
    background-position:
      56px 28px,
      28px 56px;
  }
}

@keyframes ai-image-loading-breathe {
  0%,
  100% {
    opacity: 0.62;
    transform: scale(1);
  }

  50% {
    opacity: 0.9;
    transform: scale(1.04);
  }
}

@keyframes ai-image-loading-spark {
  0%,
  100% {
    opacity: 0;
    transform: scale(0.7);
  }

  45% {
    opacity: 0.88;
    transform: scale(1.4);
  }
}

@keyframes ai-image-loading-stage {
  0%,
  100% {
    border-color: var(--admin-page-card-border-soft);
    color: var(--admin-page-text-secondary);
  }

  45% {
    border-color: var(--admin-page-accent-soft-border);
    color: var(--admin-page-text-primary);
  }
}

@keyframes ai-image-loading-dot {
  0%,
  100% {
    opacity: 0.32;
    transform: translateY(0);
  }

  50% {
    opacity: 0.86;
    transform: translateY(-4px);
  }
}

@media (width <= 640px) {
  .ai-image-form__grid,
  .ai-image-form__grid--compact {
    grid-template-columns: 1fr;
  }

  .ai-image-form__footer,
  .ai-image-results__toolbar {
    align-items: stretch;
    flex-direction: column;
  }

  .ai-image-form__actions {
    justify-content: flex-end;
  }

  .ai-image-prompt-actions {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
