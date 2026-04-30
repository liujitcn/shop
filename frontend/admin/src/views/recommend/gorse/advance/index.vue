<template>
  <div class="gorse-page gorse-advance-page">
    <div class="gorse-advance-page__grid">
      <el-card class="gorse-advance-card" shadow="never">
        <template #header>
          <div class="gorse-advance-card__header">
            <div class="gorse-advance-card__title">导出</div>
            <el-tabs v-model="exportDataTypeKey" class="gorse-advance-card__tabs gorse-advance-card__tabs--compact">
              <el-tab-pane v-for="option in dataTypeOptions" :key="option.key" :label="option.label" :name="option.key" />
            </el-tabs>
          </div>
        </template>

        <div class="gorse-advance-card__body gorse-advance-card__body--center">
          <div class="gorse-export-panel">
            <div class="gorse-export-panel__meta">
              <span class="gorse-export-panel__label">当前导出文件</span>
              <strong>{{ currentExportDataTypeMeta.fileName }}</strong>
              <p>{{ currentExportDataTypeMeta.exportDescription }}</p>
            </div>
            <div class="gorse-export-panel__tags">
              <el-tag effect="plain" size="small">JSONL</el-tag>
              <el-tag effect="plain" size="small" type="success">UTF-8</el-tag>
              <el-tag effect="plain" size="small" type="info">可重新导入</el-tag>
            </div>
            <el-button :loading="exportLoading" type="primary" @click="handleExportData">
              {{ currentExportDataTypeMeta.exportButtonText }}
            </el-button>
          </div>
        </div>
      </el-card>

      <el-card class="gorse-advance-card" shadow="never">
        <template #header>
          <div class="gorse-advance-card__header">
            <div class="gorse-advance-card__title">导入</div>
            <el-tabs v-model="importDataTypeKey" class="gorse-advance-card__tabs gorse-advance-card__tabs--compact">
              <el-tab-pane v-for="option in dataTypeOptions" :key="option.key" :label="option.label" :name="option.key" />
            </el-tabs>
          </div>
        </template>

        <div class="gorse-advance-card__body">
          <el-upload
            ref="importUploadRef"
            action="#"
            :auto-upload="false"
            :show-file-list="false"
            accept=".json,.jsonl,application/json"
            :on-change="handleSelectImportFileChange"
          >
            <div class="gorse-upload-panel">
              <el-icon class="gorse-upload-panel__icon"><UploadFilled /></el-icon>
              <div class="gorse-upload-panel__title">
                {{ currentImportDataTypeMeta.importButtonText }}
              </div>
              <div class="gorse-upload-panel__text">{{ currentImportDataTypeMeta.importDescription }}</div>
              <div class="gorse-upload-panel__formats">
                <el-tag effect="plain" size="small">JSON 数组</el-tag>
                <el-tag effect="plain" size="small">JSON 对象</el-tag>
                <el-tag effect="plain" size="small">JSONL</el-tag>
              </div>
            </div>
          </el-upload>

          <div class="gorse-import-hints">
            <div class="gorse-import-hints__item">
              <span>目标类型</span>
              <strong>{{ currentImportDataTypeMeta.label }}</strong>
            </div>
            <div class="gorse-import-hints__item">
              <span>必填字段</span>
              <strong>{{ currentImportRequiredFields }}</strong>
            </div>
            <div class="gorse-import-hints__item">
              <span>文件状态</span>
              <strong>{{ importFileName || "待选择" }}</strong>
            </div>
          </div>

          <div class="gorse-import-actions">
            <el-button :disabled="!importFileContent" @click="handleClearImportFile">清空</el-button>
            <el-button :disabled="!canImport" :loading="importLoading" type="primary" @click="handleImportData">
              确认导入{{ currentImportDataTypeMeta.label }}
            </el-button>
          </div>
        </div>
      </el-card>
    </div>

    <el-card class="gorse-preview-card" shadow="never">
      <template #header>
        <div class="gorse-preview-card__toolbar">
          <div class="gorse-advance-card__title">预览</div>
          <div v-if="previewRecordCount" class="gorse-preview-card__filters">
            <el-input
              v-model="previewKeyword"
              clearable
              class="gorse-preview-card__search"
              :placeholder="previewSearchPlaceholder"
            />
            <el-radio-group v-model="previewStatusFilter" size="small">
              <el-radio-button label="all">全部 {{ previewRecordCount }}</el-radio-button>
              <el-radio-button label="success">通过 {{ previewSuccessCount }}</el-radio-button>
              <el-radio-button label="failed">失败 {{ previewFailedCount }}</el-radio-button>
            </el-radio-group>
          </div>
        </div>
      </template>

      <el-alert v-if="previewError" class="gorse-import-alert" :closable="false" type="error" :title="previewError" />

      <div class="no-card gorse-preview-card__table">
        <ProTable
          :key="previewTableKey"
          row-key="previewKey"
          :data="filteredPreviewRows"
          :columns="previewColumns"
          :pagination="true"
          :tool-button="false"
        />
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { ElMessage } from "element-plus";
import { UploadFilled } from "@element-plus/icons-vue";
import type { ColumnProps } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
import { defRecommendGorseService } from "@/api/admin/recommend_gorse";
import { AdvanceDataType } from "@/rpc/common/v1/enum";
import { type ExportDataRequest, type ImportDataRequest } from "@/rpc/admin/v1/recommend_gorse";
import type { UploadFile, UploadInstance, UploadRawFile } from "element-plus";

defineOptions({
  name: "RecommendGorseAdvance"
});

/** 高级调试数据类型选项。 */
interface AdvanceDataTypeOption {
  /** 标签页唯一键。 */
  key: string;
  /** 数据类型值。 */
  value: AdvanceDataType;
  /** 页面展示名称。 */
  label: string;
  /** 默认文件名。 */
  fileName: string;
  /** 导出按钮文案。 */
  exportButtonText: string;
  /** 导出提示文案。 */
  exportDescription: string;
  /** 导入选择区文案。 */
  importButtonText: string;
  /** 导入提示文案。 */
  importDescription: string;
}

/** 预览校验状态。 */
type PreviewValidationStatus = "success" | "failed";

/** 预览表格筛选状态。 */
type PreviewStatusFilter = "all" | PreviewValidationStatus;

/** 导入记录解析结果。 */
interface ParsedImportRecord {
  /** 源内容位置，用于定位 JSONL 行号或 JSON 数组序号。 */
  sourceLabel: string;
  /** 成功解析出的记录对象。 */
  record: Record<string, unknown>;
  /** JSON 解析或对象结构错误，存在时该行会标记为失败。 */
  parseError: string;
}

/** 当前导入类型对应的导出字段规则。 */
interface ImportSchemaConfig {
  /** 数据类型值。 */
  dataType: AdvanceDataType;
  /** 页面展示名称。 */
  label: string;
  /** 必填字段，必须与导出 JSON 字段名完全一致。 */
  requiredFields: string[];
  /** 可选字段，必须与导出 JSON 字段名完全一致。 */
  optionalFields: string[];
}

/** 统一预览行结构。 */
interface AdvancePreviewRow {
  /** 预览唯一键。 */
  previewKey: string;
  /** 预览序号。 */
  sourceLabel: string;
  /** 校验状态。 */
  status: PreviewValidationStatus;
  /** 失败原因，成功时为空。 */
  failureReason: string;
  /** 用户ID。 */
  userId: string;
  /** 商品ID。 */
  itemId: string;
  /** 是否隐藏。 */
  isHidden: string;
  /** 分类。 */
  categories: string;
  /** 标签。 */
  labels: string;
  /** 值。 */
  value: string;
  /** 备注。 */
  comment: string;
  /** 时间文本。 */
  timestamp: string;
  /** 反馈类型。 */
  feedbackType: string;
}

const dataTypeOptions: AdvanceDataTypeOption[] = [
  {
    key: "user",
    value: AdvanceDataType.USER_RRADT,
    label: "用户",
    fileName: "users.jsonl",
    exportButtonText: "导出用户数据",
    exportDescription: "导出用户资料为 JSONL 文件，便于备份、迁移或线下排查。",
    importButtonText: "导入用户数据",
    importDescription: "请选择用户 JSON 或 JSONL 文件，系统会逐行校验导出字段 user_id 后生成预览。"
  },
  {
    key: "item",
    value: AdvanceDataType.ITEM_RRADT,
    label: "商品",
    fileName: "items.jsonl",
    exportButtonText: "导出商品数据",
    exportDescription: "导出商品资料为 JSONL 文件，便于备份、迁移或离线分析。",
    importButtonText: "导入商品数据",
    importDescription: "请选择商品 JSON 或 JSONL 文件，系统会逐行校验导出字段 item_id 后生成预览。"
  },
  {
    key: "feedback",
    value: AdvanceDataType.FEEDBACK_RRADT,
    label: "反馈",
    fileName: "feedback.jsonl",
    exportButtonText: "导出反馈数据",
    exportDescription: "导出用户反馈行为为 JSONL 文件，便于分析、备份和数据迁移。",
    importButtonText: "导入反馈数据",
    importDescription: "请选择反馈 JSON 或 JSONL 文件，系统会逐行校验反馈类型、用户和商品后生成预览。"
  }
];

/** 预览校验状态枚举，用于表格标签展示。 */
const previewStatusOptions = [
  { label: "通过", value: "success", tagType: "success" },
  { label: "失败", value: "failed", tagType: "danger" }
];

/** 用户导入字段必须与导出的用户 JSON 字段一致。 */
const userImportSchema: ImportSchemaConfig = {
  dataType: AdvanceDataType.USER_RRADT,
  label: "用户",
  requiredFields: ["user_id"],
  optionalFields: ["labels", "comment"]
};

/** 商品导入字段必须与导出的商品 JSON 字段一致。 */
const itemImportSchema: ImportSchemaConfig = {
  dataType: AdvanceDataType.ITEM_RRADT,
  label: "商品",
  requiredFields: ["item_id"],
  optionalFields: ["is_hidden", "labels", "categories", "timestamp", "comment"]
};

/** 反馈导入字段必须与导出的反馈 JSON 字段一致。 */
const feedbackImportSchema: ImportSchemaConfig = {
  dataType: AdvanceDataType.FEEDBACK_RRADT,
  label: "反馈",
  requiredFields: ["feedback_type", "user_id", "item_id"],
  optionalFields: ["value", "timestamp"]
};

/** 高级导入按数据类型匹配对应的字段规则。 */
const importSchemaList: ImportSchemaConfig[] = [userImportSchema, itemImportSchema, feedbackImportSchema];

const exportDataTypeKey = ref("user");
const importDataTypeKey = ref("user");
const exportLoading = ref(false);
const importLoading = ref(false);
const importFileName = ref("");
const importFileContent = ref("");
const previewError = ref("");
const previewRecordCount = ref(0);
const previewRows = ref<AdvancePreviewRow[]>([]);
const previewKeyword = ref("");
const previewStatusFilter = ref<PreviewStatusFilter>("all");
const previewTableVersion = ref(0);
const importUploadRef = ref<UploadInstance>();

/** 预览序号表格列。 */
const sequencePreviewColumns: ColumnProps[] = [{ prop: "sourceLabel", label: "序号", width: 80 }];

/** 通用校验结果表格列。 */
const validationPreviewColumns: ColumnProps[] = [
  { prop: "status", label: "校验结果", minWidth: 110, tag: true, enum: previewStatusOptions },
  { prop: "failureReason", label: "失败原因", minWidth: 300, showOverflowTooltip: true }
];

/** 用户预览表格列。 */
const userPreviewColumns: ColumnProps[] = [
  { prop: "userId", label: "用户ID", minWidth: 180 },
  { prop: "labels", label: "标签", minWidth: 320, showOverflowTooltip: true },
  { prop: "comment", label: "备注", minWidth: 220, showOverflowTooltip: true }
];

/** 商品预览表格列。 */
const itemPreviewColumns: ColumnProps[] = [
  { prop: "itemId", label: "商品ID", minWidth: 180 },
  { prop: "isHidden", label: "是否隐藏", minWidth: 120 },
  { prop: "categories", label: "分类", minWidth: 220, showOverflowTooltip: true },
  { prop: "timestamp", label: "时间", minWidth: 180 },
  { prop: "labels", label: "标签", minWidth: 320, showOverflowTooltip: true },
  { prop: "comment", label: "备注", minWidth: 220, showOverflowTooltip: true }
];

/** 反馈预览表格列。 */
const feedbackPreviewColumns: ColumnProps[] = [
  { prop: "feedbackType", label: "反馈类型", minWidth: 160 },
  { prop: "userId", label: "用户ID", minWidth: 180 },
  { prop: "itemId", label: "商品ID", minWidth: 180 },
  { prop: "value", label: "反馈值", minWidth: 120 },
  { prop: "timestamp", label: "时间", minWidth: 180 }
];

/** 当前选中的高级调试数据类型元信息。 */
const currentExportDataTypeMeta = computed(() => {
  return dataTypeOptions.find(option => option.key === exportDataTypeKey.value) ?? dataTypeOptions[0];
});

/** 当前导入数据类型元信息。 */
const currentImportDataTypeMeta = computed(() => {
  return dataTypeOptions.find(option => option.key === importDataTypeKey.value) ?? dataTypeOptions[0];
});

/** 当前导入类型对应的字段规则。 */
const currentImportSchema = computed(() => {
  return importSchemaList.find(schema => schema.dataType === currentImportDataTypeMeta.value.value) ?? userImportSchema;
});

/** 当前导入类型需要校验的关键字段。 */
const currentImportRequiredFields = computed(() => {
  return currentImportSchema.value.requiredFields.join(" + ");
});

/** 当前预览表头会随导入类型切换。 */
const previewColumns = computed<ColumnProps[]>(() => {
  switch (currentImportDataTypeMeta.value.value) {
    case AdvanceDataType.USER_RRADT:
      return [...sequencePreviewColumns, ...userPreviewColumns, ...validationPreviewColumns];
    case AdvanceDataType.ITEM_RRADT:
      return [...sequencePreviewColumns, ...itemPreviewColumns, ...validationPreviewColumns];
    case AdvanceDataType.FEEDBACK_RRADT:
      return [...sequencePreviewColumns, ...feedbackPreviewColumns, ...validationPreviewColumns];
    default:
      return [...sequencePreviewColumns, ...userPreviewColumns, ...validationPreviewColumns];
  }
});

/** 预览中校验通过的记录数量。 */
const previewSuccessCount = computed(() => {
  return previewRows.value.filter(row => row.status === "success").length;
});

/** 预览中校验失败的记录数量。 */
const previewFailedCount = computed(() => {
  return previewRows.value.filter(row => row.status === "failed").length;
});

/** 当前预览搜索框提示文案。 */
const previewSearchPlaceholder = computed(() => {
  switch (currentImportDataTypeMeta.value.value) {
    case AdvanceDataType.USER_RRADT:
      return "搜索用户ID / 标签 / 备注 / 失败原因";
    case AdvanceDataType.ITEM_RRADT:
      return "搜索商品ID / 分类 / 标签 / 失败原因";
    case AdvanceDataType.FEEDBACK_RRADT:
      return "搜索反馈类型 / 用户ID / 商品ID / 失败原因";
    default:
      return "搜索关键字段 / 失败原因";
  }
});

/** 根据校验结果和关键词过滤后的预览记录。 */
const filteredPreviewRows = computed(() => {
  const keyword = previewKeyword.value.trim().toLowerCase();
  return previewRows.value.filter(row => {
    // 先按校验状态过滤，避免用户误把失败记录隐藏后继续导入。
    if (previewStatusFilter.value !== "all" && row.status !== previewStatusFilter.value) return false;
    if (!keyword) return true;
    return Object.values(row).some(value => normalizeRecordValue(value).toLowerCase().includes(keyword));
  });
});

/** 预览表格 key 会随筛选条件变化，确保 ProTable 静态数据分页总数同步刷新。 */
const previewTableKey = computed(() => {
  return `preview-${previewTableVersion.value}-${importDataTypeKey.value}-${previewStatusFilter.value}-${previewKeyword.value}-${filteredPreviewRows.value.length}`;
});

/** 当前预览是否允许继续导入。 */
const canImport = computed(() => {
  return !!importFileContent.value && !previewError.value && previewRecordCount.value > 0 && previewFailedCount.value === 0;
});

watch(importDataTypeKey, () => {
  // 切换导入类型后必须重新选择文件，避免旧文件在新类型下继续展示或误导入。
  if (importFileContent.value) {
    handleClearImportFile();
  }
});

/** 导出当前数据类型的 JSONL 文件。 */
async function handleExportData() {
  exportLoading.value = true;
  try {
    const request: ExportDataRequest = {
      data_type: currentExportDataTypeMeta.value.value
    };
    const data = await defRecommendGorseService.ExportData(request);
    downloadTextFile(data.file_name || currentExportDataTypeMeta.value.fileName, data.content || "");
    ElMessage.success("导出 Gorse 推荐数据成功");
  } finally {
    exportLoading.value = false;
  }
}

/** 监听导入文件选择事件，auto-upload=false 时需要通过 on-change 读取本地文件。 */
function handleSelectImportFileChange(file: UploadFile) {
  if (!file.raw) return;
  handleSelectImportFile(file.raw).catch(() => {
    ElMessage.error("读取导入文件失败");
  });
}

/** 选择导入文件后，先在前端读取并生成预览。 */
async function handleSelectImportFile(file: UploadRawFile) {
  try {
    const content = await readTextFile(file);
    importFileName.value = file.name;
    importFileContent.value = content;
    resetPreviewFilter();
    buildPreview(content);
  } catch (error) {
    importFileName.value = "";
    importFileContent.value = "";
    previewRows.value = [];
    previewRecordCount.value = 0;
    previewError.value = "读取导入文件失败";
    resetPreviewFilter();
    throw error;
  }
  return false;
}

/** 清空当前导入文件与预览结果。 */
function handleClearImportFile() {
  importUploadRef.value?.clearFiles();
  importFileName.value = "";
  importFileContent.value = "";
  previewRows.value = [];
  previewRecordCount.value = 0;
  previewError.value = "";
  previewTableVersion.value += 1;
  resetPreviewFilter();
}

/** 重置预览表格筛选条件。 */
function resetPreviewFilter() {
  previewKeyword.value = "";
  previewStatusFilter.value = "all";
}

/** 确认导入当前预览文件。 */
async function handleImportData() {
  if (!canImport.value) return;

  importLoading.value = true;
  try {
    const request: ImportDataRequest = {
      data_type: currentImportDataTypeMeta.value.value,
      file_name: importFileName.value || currentImportDataTypeMeta.value.fileName,
      content: importFileContent.value
    };
    const data = await defRecommendGorseService.ImportData(request);
    ElMessage.success(`导入成功，共处理 ${data.success_count} 条记录`);
    // 导入成功后清空文件与预览数据，避免用户重复提交同一批导入内容。
    handleClearImportFile();
  } finally {
    importLoading.value = false;
  }
}

/** 根据当前文件内容构建导入预览。 */
function buildPreview(content: string) {
  previewError.value = "";
  try {
    const recordList = parseImportRecords(content);
    const rowList = recordList.map((record, index) => buildPreviewRow(record, index));
    previewRecordCount.value = rowList.length;
    previewRows.value = rowList;
    previewTableVersion.value += 1;
  } catch (error) {
    previewRecordCount.value = 0;
    previewRows.value = [];
    previewError.value = error instanceof Error ? error.message : "导入文件格式不正确";
    previewTableVersion.value += 1;
  }
}

/** 将记录对象转换为当前页面可展示的统一预览行。 */
function buildPreviewRow(parsedRecord: ParsedImportRecord, index: number): AdvancePreviewRow {
  const record = parsedRecord.record;
  const failureReasonList = parsedRecord.parseError ? [parsedRecord.parseError] : validatePreviewRecord(record);
  const status: PreviewValidationStatus = failureReasonList.length ? "failed" : "success";
  const categories = record.Categories ?? record.categories;
  const labels = record.Labels ?? record.labels;
  return {
    previewKey: `${importDataTypeKey.value}-${index}`,
    sourceLabel: String(index + 1),
    status,
    failureReason: failureReasonList.join("；"),
    userId: normalizeRecordValue(record.UserId ?? record.user_id),
    itemId: normalizeRecordValue(record.ItemId ?? record.item_id),
    isHidden: normalizeRecordValue(record.IsHidden ?? record.is_hidden),
    categories: formatPreviewValue(categories),
    labels: formatPreviewValue(labels),
    value: normalizeRecordValue(record.Value ?? record.value),
    comment: normalizeRecordValue(record.Comment ?? record.comment),
    timestamp: normalizeRecordValue(record.Timestamp ?? record.timestamp),
    feedbackType: normalizeRecordValue(record.FeedbackType ?? record.feedback_type)
  };
}

/** 校验当前数据类型的字段名和关键字段是否与导出结构一致。 */
function validatePreviewRecord(record: Record<string, unknown>) {
  const schema = currentImportSchema.value;
  const allowedFieldSet = new Set([...schema.requiredFields, ...schema.optionalFields]);
  const recordFieldList = Object.keys(record);
  const failureReasonList: string[] = [];
  // 空对象虽然是合法 JSON，但无法与任一导出结构匹配。
  if (!recordFieldList.length) {
    failureReasonList.push("记录不能为空对象");
  }

  const unexpectedFieldList = recordFieldList.filter(field => !allowedFieldSet.has(field));
  // 用户选错反馈或商品文件时，会因为字段集合不匹配被逐行拦截。
  if (unexpectedFieldList.length) {
    failureReasonList.push(`${schema.label}导出结构不包含字段：${unexpectedFieldList.join("、")}`);
  }

  schema.requiredFields.forEach(field => {
    if (!hasNonEmptyRecordValue(record[field])) {
      failureReasonList.push(`${schema.label}数据缺少导出字段 ${field}`);
    }
  });
  return failureReasonList;
}

/** 解析 JSON 数组、单对象或 JSONL 文本。 */
function parseImportRecords(content: string): ParsedImportRecord[] {
  const normalizedContent = content.replace(/^\uFEFF/, "").trim();
  // 文件内容为空时，不存在任何可供预览的记录对象。
  if (!normalizedContent) {
    throw new Error("导入文件内容不能为空");
  }

  // 文件整体是 JSON 数组时，优先按数组解析。
  if (normalizedContent.startsWith("[")) {
    const parsedValue = JSON.parse(normalizedContent);
    if (!Array.isArray(parsedValue)) {
      throw new Error("导入文件不是合法的 JSON 数组");
    }
    return parsedValue.map((value, index) => buildParsedImportRecord(value, `第 ${index + 1} 条`));
  }

  // 文件整体是单个 JSON 对象时，直接包装成单条预览记录。
  if (normalizedContent.startsWith("{") && !normalizedContent.includes("\n")) {
    return [buildParsedImportRecord(JSON.parse(normalizedContent), "第 1 条")];
  }

  const parsedRecordList: ParsedImportRecord[] = [];
  normalizedContent.split("\n").forEach((line, index) => {
    const trimmedLine = line.trim();
    // 空白行对 JSONL 导入没有业务意义，统一跳过。
    if (!trimmedLine) return;

    const sourceLabel = `第 ${index + 1} 行`;
    // JSONL 每一行都必须是单个 JSON 对象，解析失败也要留在表格里提示。
    if (!trimmedLine.startsWith("{")) {
      parsedRecordList.push({
        sourceLabel,
        record: {},
        parseError: "不是合法的 JSONL 对象行"
      });
      return;
    }

    try {
      parsedRecordList.push(buildParsedImportRecord(JSON.parse(trimmedLine), sourceLabel));
    } catch (error) {
      parsedRecordList.push({
        sourceLabel,
        record: {},
        parseError: `JSON 解析失败：${getErrorMessage(error)}`
      });
    }
  });
  // 过滤空白行后没有任何记录时，提示用户重新选择文件。
  if (!parsedRecordList.length) {
    throw new Error("导入文件缺少有效数据");
  }
  return parsedRecordList;
}

/** 构建带源位置的解析记录，非对象内容会作为失败行展示。 */
function buildParsedImportRecord(value: unknown, sourceLabel: string): ParsedImportRecord {
  // 只有普通对象才能作为高级调试记录参与预览与导入。
  if (typeof value !== "object" || value === null || Array.isArray(value)) {
    return {
      sourceLabel,
      record: {},
      parseError: "记录必须是 JSON 对象"
    };
  }
  return {
    sourceLabel,
    record: value as Record<string, unknown>,
    parseError: ""
  };
}

/** 判断必填字段是否存在有效内容。 */
function hasNonEmptyRecordValue(value: unknown) {
  return normalizeRecordValue(value).trim() !== "";
}

/** 获取异常文本，避免 JSON 解析失败时丢失具体原因。 */
function getErrorMessage(error: unknown) {
  if (error instanceof Error) return error.message;
  return "未知错误";
}

/** 将本地文件读取为 UTF-8 文本。 */
function readTextFile(file: File) {
  return new Promise<string>((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(typeof reader.result === "string" ? reader.result : "");
    reader.onerror = () => reject(reader.error ?? new Error("文件读取失败"));
    reader.readAsText(file, "utf-8");
  });
}

/** 下载文本文件到浏览器本地。 */
function downloadTextFile(fileName: string, content: string) {
  const blob = new Blob([content], { type: "application/x-ndjson;charset=utf-8" });
  const blobUrl = window.URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = blobUrl;
  link.download = fileName;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  window.URL.revokeObjectURL(blobUrl);
}

/** 将未知值转换成适合预览展示的文本。 */
function normalizeRecordValue(value: unknown) {
  if (value === null || value === undefined) return "";
  if (typeof value === "string") return value;
  if (typeof value === "number" || typeof value === "boolean") return String(value);
  return JSON.stringify(value);
}

/** 将对象、数组或原始值格式化为预览文本。 */
function formatPreviewValue(value: unknown) {
  if (value === null || value === undefined) return "";
  if (Array.isArray(value))
    return value
      .map(item => normalizeRecordValue(item))
      .filter(Boolean)
      .join(" | ");
  if (typeof value === "object") return JSON.stringify(value);
  return normalizeRecordValue(value);
}
</script>

<style scoped lang="scss">
.gorse-advance-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.gorse-advance-page__grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
}

.gorse-advance-card,
.gorse-preview-card {
  border: 1px solid var(--admin-page-card-border);
  border-radius: var(--admin-page-radius);
  background: var(--admin-page-card-bg);
  box-shadow: var(--admin-page-shadow);
  overflow: hidden;
}

.gorse-advance-card__header {
  display: flex;
  gap: 16px;
  align-items: center;
  justify-content: space-between;
}

.gorse-advance-card__title {
  color: var(--admin-page-text-primary);
  font-size: 18px;
  font-weight: 600;
}

.gorse-advance-card__body {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.gorse-advance-card__tabs {
  width: 100%;

  :deep(.el-tabs__header) {
    margin-bottom: 0;
  }

  :deep(.el-tabs__nav-wrap::after) {
    background: var(--admin-page-divider);
  }
}

.gorse-advance-card__tabs--compact {
  width: auto;

  :deep(.el-tabs__header) {
    margin-bottom: 0;
  }

  :deep(.el-tabs__nav-wrap::after) {
    display: none;
  }
}

.gorse-advance-card__body--center {
  align-items: flex-start;
  justify-content: center;
  min-height: 132px;
}

.gorse-export-panel {
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  gap: 14px;
  width: 100%;
  max-width: 100%;
  padding: 18px;
  border: 1px solid var(--admin-page-card-border-soft);
  border-radius: var(--admin-page-radius);
  background: linear-gradient(135deg, rgb(20 184 166 / 9%), rgb(20 184 166 / 2%)), var(--admin-page-card-bg-soft);
}

.gorse-export-panel__meta {
  display: flex;
  flex-direction: column;
  gap: 6px;

  strong {
    color: var(--admin-page-text-primary);
    font-size: 18px;
    line-height: 24px;
  }

  p {
    margin: 0;
    color: var(--admin-page-text-secondary);
    font-size: 13px;
    line-height: 20px;
  }
}

.gorse-export-panel__label {
  color: var(--admin-page-text-secondary);
  font-size: 12px;
}

.gorse-export-panel__tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.gorse-upload-panel {
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  gap: 10px;
  align-items: center;
  justify-content: center;
  width: 100%;
  max-width: 100%;
  padding: 28px 20px;
  background: linear-gradient(180deg, rgb(14 165 233 / 7%), rgb(14 165 233 / 2%)), var(--admin-page-card-bg-soft);
  border: 1px dashed var(--admin-page-card-border-muted);
  border-radius: var(--admin-page-radius);
  cursor: pointer;
}

.gorse-upload-panel__icon {
  color: var(--el-color-primary);
  font-size: 28px;
}

.gorse-upload-panel__title {
  max-width: 100%;
  overflow: hidden;
  color: var(--admin-page-text-primary);
  font-size: 15px;
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.gorse-upload-panel__text {
  color: var(--admin-page-text-secondary);
  font-size: 13px;
  text-align: center;
}

.gorse-upload-panel__formats {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  justify-content: center;
}

.gorse-import-hints {
  box-sizing: border-box;
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
  max-width: 100%;
}

.gorse-import-hints__item {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 4px;
  padding: 10px 12px;
  border: 1px solid var(--admin-page-card-border-soft);
  border-radius: var(--admin-page-radius);
  background: var(--admin-page-card-bg-soft);

  span {
    color: var(--admin-page-text-secondary);
    font-size: 12px;
    line-height: 18px;
  }

  strong {
    overflow: hidden;
    color: var(--admin-page-text-primary);
    font-size: 13px;
    line-height: 20px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

.gorse-import-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  align-items: center;
  justify-content: flex-end;
}

.gorse-import-alert {
  margin-bottom: 16px;
}

.gorse-preview-card__filters {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  align-items: center;
  justify-content: flex-end;
}

.gorse-preview-card__search {
  width: 260px;
  max-width: 100%;
}

.gorse-preview-card__toolbar {
  display: flex;
  gap: 12px;
  align-items: center;
  justify-content: space-between;
}

.gorse-preview-card__table {
  width: 100%;

  :deep(.table-main) {
    height: auto;
    min-height: auto;
  }

  :deep(.el-table) {
    flex: initial;
  }

  :deep(.el-table__inner-wrapper),
  :deep(.el-table__body-wrapper),
  :deep(.el-scrollbar),
  :deep(.el-scrollbar__wrap),
  :deep(.el-scrollbar__view) {
    height: auto;
  }
}

@media (width <= 960px) {
  .gorse-advance-page__grid,
  .gorse-import-hints {
    grid-template-columns: 1fr;
  }

  .gorse-advance-card__header,
  .gorse-import-actions,
  .gorse-preview-card__filters,
  .gorse-preview-card__toolbar {
    flex-direction: column;
    align-items: stretch;
  }
}
</style>
