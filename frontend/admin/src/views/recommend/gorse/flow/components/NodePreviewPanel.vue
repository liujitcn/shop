<template>
  <template v-if="nodeForm.type === 'external'">
    <el-alert
      class="gorse-flow-preview-alert"
      type="info"
      show-icon
      :closable="false"
      title="外部推荐脚本预览会读取外部脚本字段。"
    />
    <div class="gorse-flow-preview-box">
      <el-input v-model="previewUserId" placeholder="请输入预览用户编号" clearable />
      <el-button type="primary" :loading="previewLoading" @click="runExternalPreview">
        <i class="material-icons gorse-flow-button-icon">play_arrow</i>
      </el-button>
    </div>
  </template>

  <template v-if="nodeForm.type === 'ranker' && isLlmRanker">
    <el-alert
      class="gorse-flow-preview-alert"
      type="info"
      show-icon
      :closable="false"
      title="排序提示词预览会读取查询模板与文档模板字段。"
    />
    <div class="gorse-flow-preview-box">
      <el-input v-model="rankerPreviewUserId" placeholder="请输入预览用户编号" clearable />
      <el-button type="primary" :loading="rankerPreviewLoading" @click="runRankerPreview">
        <i class="material-icons gorse-flow-button-icon">play_arrow</i>
      </el-button>
    </div>
  </template>
  <template v-else-if="nodeForm.type === 'ranker'">
    <el-alert
      class="gorse-flow-preview-alert"
      type="info"
      show-icon
      :closable="false"
      title="仅大语言模型排序器支持提示词预览，因子分解机排序器无需预览提示词。"
    />
  </template>

  <pre v-if="previewResult" class="gorse-flow-preview-result">{{ previewResult }}</pre>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { defRecommendGorseService } from "@/api/admin/recommend_gorse";
import type { NodeFormState } from "../types";

interface NodePreviewPanelProps {
  /** 当前节点表单状态，用于读取预览接口所需参数。 */
  nodeForm: NodeFormState;
}

const props = defineProps<NodePreviewPanelProps>();

const previewUserId = ref("");
const rankerPreviewUserId = ref("");
const previewResult = ref("");
const previewLoading = ref(false);
const rankerPreviewLoading = ref(false);
const isLlmRanker = computed(() => props.nodeForm.properties.type === "llm");

/** 执行外部推荐脚本预览。 */
async function runExternalPreview() {
  const script = String(props.nodeForm.properties.script ?? "").trim();
  // 外部推荐脚本为空时，不请求后端代理 Gorse 预览接口，避免Gorse 返回 500。
  if (!script) {
    ElMessage.warning("请先填写外部推荐脚本");
    return;
  }
  previewLoading.value = true;
  previewResult.value = "";
  try {
    const response = await defRecommendGorseService.PreviewExternal({
      user_id: previewUserId.value,
      script
    });
    previewResult.value = formatExternalPreview(response.items ?? []);
  } finally {
    previewLoading.value = false;
  }
}

/** 执行排序提示词预览。 */
async function runRankerPreview() {
  // 非大语言模型排序器没有提示词字段，直接拦截避免后端代理 Gorse 接口返回 500。
  if (!isLlmRanker.value) {
    ElMessage.warning("仅大语言模型排序器支持提示词预览");
    return;
  }
  const queryTemplate = String(props.nodeForm.properties.query_template ?? "").trim();
  const documentTemplate = String(props.nodeForm.properties.document_template ?? "").trim();
  // 查询模板与文档模板缺失时，无法构造有效排序提示词。
  if (!queryTemplate || !documentTemplate) {
    ElMessage.warning("请先填写查询模板和文档模板");
    return;
  }
  rankerPreviewLoading.value = true;
  previewResult.value = "";
  try {
    const response = await defRecommendGorseService.PreviewRankerPrompt({
      user_id: rankerPreviewUserId.value,
      query_template: queryTemplate,
      document_template: documentTemplate
    });
    previewResult.value = formatRankerPreview(response.query ?? "", response.documents ?? []);
  } finally {
    rankerPreviewLoading.value = false;
  }
}

/** 格式化外部推荐脚本预览结果，按商品编号逐行展示。 */
function formatExternalPreview(items: string[]) {
  // Gorse没有返回候选商品时，给出明确空结果说明。
  if (items.length === 0) {
    return "暂无推荐商品编号";
  }
  return ["推荐商品编号：", ...items.map((item, index) => `${index + 1}. ${item}`)].join("\n");
}

/** 格式化排序提示词预览结果，把 JSON 字符串中的换行渲染为真实文本段落。 */
function formatRankerPreview(query: string, documents: string[]) {
  const sections = [`【查询提示词】\n${query || "暂无查询提示词"}`];
  // 文档提示词为空时，提示当前用户或候选商品数据不足。
  if (documents.length === 0) {
    sections.push("【文档提示词】\n暂无文档提示词");
    return sections.join("\n\n");
  }
  documents.forEach((document, index) => {
    sections.push(`【文档提示词 ${index + 1}】\n${document}`);
  });
  return sections.join("\n\n");
}

/** 清理预览状态，避免下次编辑串用旧结果。 */
function resetPreviewState() {
  previewUserId.value = "";
  rankerPreviewUserId.value = "";
  previewResult.value = "";
  previewLoading.value = false;
  rankerPreviewLoading.value = false;
}

defineExpose({
  resetPreviewState
});
</script>

<style scoped lang="scss">
.gorse-flow-preview-alert {
  margin-bottom: 12px;
}

.gorse-flow-preview-box {
  display: flex;
  gap: 10px;
  margin-bottom: 12px;
}

.gorse-flow-preview-result {
  margin-top: 10px;
  max-height: 320px;
  padding: 12px;
  overflow: auto;
  font-family:
    Consolas, Menlo, Monaco, "Lucida Console", "Liberation Mono", "DejaVu Sans Mono", "Bitstream Vera Sans Mono", "Courier New",
    monospace;
  font-size: 12px;
  line-height: 1.6;
  color: var(--el-text-color-primary);
  white-space: pre-wrap;
  word-break: break-word;
  background: var(--el-fill-color-light);
  border: 1px solid var(--el-border-color);
  border-radius: 6px;
}

.gorse-flow-button-icon {
  font-size: 18px;
  line-height: 1;
}
</style>
