<template>
  <div v-loading="loading" class="gorse-page gorse-config-page">
    <el-card v-if="configSections.length" class="gorse-tabs-card" shadow="never">
      <el-tabs v-model="activeSection" class="gorse-config-tabs">
        <el-tab-pane v-for="section in configSections" :key="section.key" :label="section.label" :name="section.key">
          <div class="gorse-config-groups">
            <div v-for="group in section.groups" :key="group.key" class="gorse-config-group">
              <div v-if="group.label" class="gorse-config-group__title">{{ group.label }}</div>
              <el-empty v-if="group.empty" :description="group.empty" :image-size="56" />
              <div v-else class="gorse-config-list">
                <div v-for="configField in group.fields" :key="`${group.key}-${configField.label}`" class="gorse-config-item">
                  <label>{{ configField.label }}</label>
                  <div class="gorse-config-item__value">
                    <div v-if="configField.tags" class="gorse-config-tags">
                      <el-tag v-for="tag in configField.tags" :key="`${configField.label}-${tag}`" type="info" effect="plain">
                        {{ tag }}
                      </el-tag>
                      <span v-if="!configField.tags.length" class="gorse-config-empty">未配置</span>
                    </div>
                    <el-input
                      v-else-if="configField.multiline"
                      :model-value="configField.text"
                      type="textarea"
                      :rows="resolveTextareaRows(configField.text)"
                      readonly
                    />
                    <el-input v-else :model-value="configField.text" readonly />
                  </div>
                </div>
              </div>
            </div>
          </div>
        </el-tab-pane>
      </el-tabs>
    </el-card>
    <el-empty v-else description="暂无推荐配置" />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import { ElMessage } from "element-plus";
import { useRecommendGorseStore } from "@/stores/modules/recommendGorse";

defineOptions({
  name: "GorseConfig"
});

/** 推荐配置原始对象，页面按当前固定返回结构直接读取。 */
type ConfigRecord = Record<string, unknown>;

/** 配置字段读取路径，数组用于兼容不同 JSON 命名策略。 */
type ConfigFieldKey = string | string[];

/** 配置字段展示元数据，字段名直接对应当前返回值。 */
interface ConfigFieldMeta {
  /** 后端固定返回字段名。 */
  key: ConfigFieldKey;
  /** 页面展示中文字段名。 */
  label: string;
  /** 是否按标签列表展示。 */
  tags?: boolean;
  /** 是否按多行文本展示。 */
  multiline?: boolean;
}

/** 配置字段展示项。 */
interface ConfigDisplayField {
  /** 中文字段名。 */
  label: string;
  /** 字段文本值。 */
  text: string;
  /** 标签文本集合。 */
  tags?: string[];
  /** 是否多行展示。 */
  multiline: boolean;
}

/** 配置展示分组。 */
interface ConfigDisplayGroup {
  /** 分组唯一标识。 */
  key: string;
  /** 分组中文名称。 */
  label: string;
  /** 无数据提示。 */
  empty?: string;
  /** 分组字段。 */
  fields: ConfigDisplayField[];
}

/** 配置展示页签。 */
interface ConfigDisplaySection {
  /** 页签唯一标识。 */
  key: string;
  /** 页签中文名称。 */
  label: string;
  /** 页签下的配置分组。 */
  groups: ConfigDisplayGroup[];
}

const loading = ref(false);
const activeSection = ref("");
const recommendGorseStore = useRecommendGorseStore();
const configSections = computed(() => buildConfigSections(recommendGorseStore.config));

const databaseFields: ConfigFieldMeta[] = [field("数据存储数据库", "data_store"), field("缓存存储数据库", "cache_store")];

const mysqlFields: ConfigFieldMeta[] = [
  field("事务隔离级别", "isolation_level"),
  field("最大打开连接数", "max_open_conns"),
  field("最大空闲连接数", "max_idle_conns"),
  field("连接最长复用时间", "conn_max_lifetime")
];

const postgresFields: ConfigFieldMeta[] = [
  field("最大打开连接数", "max_open_conns"),
  field("最大空闲连接数", "max_idle_conns"),
  field("连接最长复用时间", "conn_max_lifetime")
];

const redisFields: ConfigFieldMeta[] = [field("最大搜索结果数", "max_search_results")];

const masterFields: ConfigFieldMeta[] = [
  field("gRPC监听地址", "host"),
  field("gRPC监听端口", "port"),
  field("HTTP监听地址", "http_host"),
  field("HTTP监听端口", "http_port"),
  tagField("HTTP CORS允许域名", "http_cors_domains"),
  tagField("HTTP CORS允许方法", "http_cors_methods"),
  field("工作线程数", "n_jobs"),
  field("元数据超时时间", "meta_timeout")
];

const serverFields: ConfigFieldMeta[] = [
  field("默认返回物品数量", "default_n"),
  field("集群时钟误差", "clock_error"),
  field("自动插入新用户", "auto_insert_user"),
  field("自动插入新物品", "auto_insert_item"),
  field("服务端缓存过期时间", "cache_expire")
];

const recommendFields: ConfigFieldMeta[] = [
  field("缓存元素数量", "cache_size"),
  field("推荐缓存过期时间", "cache_expire"),
  field("在线推荐上下文大小", "context_size"),
  field("活跃用户生存时间", "active_user_ttl")
];

const dataSourceFields: ConfigFieldMeta[] = [
  tagField("正向反馈类型", "positive_feedback_types"),
  tagField("已读反馈类型", "read_feedback_types"),
  tagField("负反馈类型", "negative_feedback_types"),
  field("正向反馈生存时间", "positive_feedback_ttl"),
  field("物品生存时间", "item_ttl")
];

const nonPersonalizedFields: ConfigFieldMeta[] = [
  field("名称", "name"),
  textareaField("评分函数", "score"),
  textareaField("筛选函数", "filter")
];

const itemToItemFields: ConfigFieldMeta[] = [
  field("名称", "name"),
  field("相似度类型", "type"),
  field("相似度字段", "column"),
  textareaField("提示词", "prompt")
];

const userToUserFields: ConfigFieldMeta[] = [field("名称", "name"), field("相似度类型", "type"), field("相似度字段", "column")];

const externalFields: ConfigFieldMeta[] = [field("名称", "name"), textareaField("外部推荐脚本", "script")];

const collaborativeFields: ConfigFieldMeta[] = [
  field("协同过滤类型", "type"),
  field("模型训练周期", "fit_period"),
  field("模型训练轮数", "fit_epoch"),
  field("模型搜索周期", "optimize_period"),
  field("模型搜索试验次数", "optimize_trials"),
  field("早停等待轮数", "early_stopping.patience")
];

const replacementFields: ConfigFieldMeta[] = [
  field("启用已读物品替换", "enable_replacement"),
  field("正向反馈替换衰减", "positive_replacement_decay"),
  field("已读反馈替换衰减", "read_replacement_decay")
];

const rankerFields: ConfigFieldMeta[] = [
  field("排序器类型", "type"),
  field("不活跃用户推荐刷新周期", "cache_expire"),
  tagField("排序前候选推荐器", "recommenders"),
  field("模型拟合周期", "fit_period"),
  field("模型拟合轮数", "fit_epoch"),
  field("超参数优化周期", "optimize_period"),
  field("超参数优化次数", "optimize_trials"),
  textareaField("大语言模型重排查询模板", "query_template"),
  textareaField("大语言模型重排文档模板", "document_template"),
  field("早停等待轮数", "early_stopping.patience"),
  field("重排API密钥", "reranker_api.auth_token"),
  field("重排模型", "reranker_api.model"),
  field("重排API地址", "reranker_api.url")
];

const fallbackFields: ConfigFieldMeta[] = [tagField("个性化推荐用尽时的推荐来源", "recommenders")];

const blobFields: ConfigFieldMeta[] = [field("Blob URI", "uri")];

const tracingFields: ConfigFieldMeta[] = [field("追踪导出器类型", "exporter"), field("追踪采样器类型", "sampler")];

const openaiFields: ConfigFieldMeta[] = [
  field("基础地址", "base_url"),
  field("认证令牌", "auth_token"),
  field("对话模型", "chat_completion_model")
];

watch(
  configSections,
  sections => {
    // 配置加载前不选中页签，加载后默认选中第一个有效配置组。
    if (!sections.length) {
      activeSection.value = "";
      return;
    }
    if (!sections.some(section => section.key === activeSection.value)) activeSection.value = sections[0].key;
  },
  { immediate: true }
);

/** 加载 Gorse 推荐配置。 */
async function loadConfig() {
  loading.value = true;
  try {
    await recommendGorseStore.loadConfig(true);
  } catch (error) {
    ElMessage.error("加载推荐配置失败");
    throw error;
  } finally {
    loading.value = false;
  }
}

/** 根据配置内容长度动态设置只读文本域高度。 */
function resolveTextareaRows(text: string) {
  const rows = text.split("\n").length;
  return Math.min(12, Math.max(3, rows));
}

/** 构建固定字段配置，展示名称统一使用中文。 */
function field(label: string, key: ConfigFieldKey): ConfigFieldMeta {
  return { label, key };
}

/** 构建标签列表字段配置。 */
function tagField(label: string, key: ConfigFieldKey): ConfigFieldMeta {
  return { ...field(label, key), tags: true };
}

/** 构建多行文本字段配置。 */
function textareaField(label: string, key: ConfigFieldKey): ConfigFieldMeta {
  return { ...field(label, key), multiline: true };
}

/** 按当前固定返回结构构建页面页签。 */
function buildConfigSections(config: unknown): ConfigDisplaySection[] {
  const root = toRecord(config);
  const sections: ConfigDisplaySection[] = [];

  const database = readRecord(root, "database");
  appendSection(sections, "database", "数据库", [
    buildGroup("database-basic", "基础配置", database, databaseFields),
    buildGroup("database-mysql", "MySQL配置", readRecord(database, "mysql"), mysqlFields),
    buildGroup("database-postgres", "PostgreSQL配置", readRecord(database, "postgres"), postgresFields),
    buildGroup("database-redis", "Redis配置", readRecord(database, "redis"), redisFields)
  ]);

  const master = readRecord(root, "master");
  appendSection(sections, "master", "主节点", [buildGroup("master-basic", "基础配置", master, masterFields)]);

  const server = readRecord(root, "server");
  appendSection(sections, "server", "服务", [buildGroup("server-basic", "基础配置", server, serverFields)]);

  const recommend = readRecord(root, "recommend");
  appendSection(sections, "recommend", "推荐", buildRecommendGroups(recommend));

  const blob = readRecord(root, "blob");
  appendSection(sections, "blob", "Blob存储", [
    buildGroup("blob-basic", "基础配置", blob, blobFields),
    buildGroup("blob-s3", "S3配置", readRecord(blob, "s3"), []),
    buildGroup("blob-gcs", "GCS配置", readRecord(blob, "gcs"), []),
    buildGroup("blob-azure", "Azure Blob配置", readRecord(blob, "azure"), [])
  ]);

  const tracing = readRecord(root, "tracing");
  appendSection(sections, "tracing", "链路追踪", [buildGroup("tracing-basic", "基础配置", tracing, tracingFields)]);

  const oidc = readRecord(root, "oidc");
  appendSection(sections, "oidc", "OIDC认证", [buildGroup("oidc-basic", "基础配置", oidc, [])]);

  const openai = readRecord(root, "openai");
  appendSection(sections, "openai", "OpenAI", [buildGroup("openai-basic", "基础配置", openai, openaiFields)]);

  return sections;
}

/** 构建推荐配置页签内的所有分组。 */
function buildRecommendGroups(recommend: ConfigRecord | undefined): ConfigDisplayGroup[] {
  if (!recommend) return [];
  return [
    buildGroup("recommend-basic", "推荐基础配置", recommend, recommendFields),
    buildGroup("recommend-data-source", "数据源配置", readRecord(recommend, "data_source"), dataSourceFields),
    ...buildListGroups(
      "recommend-non-personalized",
      "非个性化推荐器",
      readRecordList(recommend, ["non-personalized", "non_personalized"]),
      nonPersonalizedFields
    ),
    ...buildListGroups(
      "recommend-item-to-item",
      "物品相似推荐器",
      readRecordList(recommend, ["item-to-item", "item_to_item"]),
      itemToItemFields
    ),
    ...buildListGroups(
      "recommend-user-to-user",
      "用户相似推荐器",
      readRecordList(recommend, ["user-to-user", "user_to_user"]),
      userToUserFields
    ),
    ...buildListGroups("recommend-external", "外部推荐器", readRecordList(recommend, "external"), externalFields),
    buildGroup("recommend-collaborative", "协同过滤配置", readRecord(recommend, "collaborative"), collaborativeFields),
    buildGroup("recommend-replacement", "替换配置", readRecord(recommend, "replacement"), replacementFields),
    buildGroup("recommend-ranker", "排序器配置", readRecord(recommend, "ranker"), rankerFields),
    buildGroup("recommend-fallback", "回退推荐配置", readRecord(recommend, "fallback"), fallbackFields)
  ];
}

/** 追加存在配置内容的页签，避免展示空白内容。 */
function appendSection(sections: ConfigDisplaySection[], key: string, label: string, groups: ConfigDisplayGroup[]) {
  const availableGroups = groups.filter(group => group.empty || group.fields.length > 0);
  if (!availableGroups.length) return;
  sections.push({ key, label, groups: availableGroups });
}

/** 构建普通对象配置分组。 */
function buildGroup(key: string, label: string, record: ConfigRecord | undefined, metas: ConfigFieldMeta[]): ConfigDisplayGroup {
  if (!record) return { key, label, fields: [] };
  // 当前返回中的空对象属于固定节点，页面保留分组并给出明确空态。
  if (!Object.keys(record).length || !metas.length) return { key, label, empty: "暂无配置", fields: [] };
  return {
    key,
    label,
    fields: metas.map(meta => buildDisplayField(record, meta))
  };
}

/** 构建数组对象配置分组。 */
function buildListGroups(key: string, label: string, list: ConfigRecord[], metas: ConfigFieldMeta[]): ConfigDisplayGroup[] {
  if (!list.length) return [{ key: `${key}-empty`, label, empty: "暂无配置", fields: [] }];
  return list.map((item, index) => {
    const itemName = formatValue(readValue(item, "name"));
    const suffix = itemName === "未配置" ? `第 ${index + 1} 项` : itemName;
    return {
      key: `${key}-${index}`,
      label: `${label}：${suffix}`,
      fields: metas.map(meta => buildDisplayField(item, meta))
    };
  });
}

/** 将字段元数据转换成页面字段。 */
function buildDisplayField(record: ConfigRecord, meta: ConfigFieldMeta): ConfigDisplayField {
  const value = readValue(record, meta.key);
  const tags = meta.tags ? formatTags(value) : undefined;
  const text = formatValue(value);
  return {
    label: meta.label,
    text,
    tags,
    multiline: Boolean(meta.multiline) || text.length > 120 || text.includes("\n")
  };
}

/** 从对象中按固定字段名读取值。 */
function readValue(record: ConfigRecord | undefined, key: ConfigFieldKey) {
  if (!record) return undefined;
  const keys = Array.isArray(key) ? key : [key];
  for (const currentKey of keys) {
    let value: unknown = record;
    let exists = true;
    for (const keyPart of currentKey.split(".")) {
      const currentRecord = toOptionalRecord(value);
      // 当前命名策略下字段不存在时，继续尝试下一个候选字段名。
      if (!currentRecord || !(keyPart in currentRecord)) {
        exists = false;
        break;
      }
      value = currentRecord[keyPart];
    }
    if (exists && value !== null && value !== undefined) return value;
  }
  return undefined;
}

/** 从对象中按固定字段名读取子对象。 */
function readRecord(record: ConfigRecord | undefined, key: ConfigFieldKey) {
  return toOptionalRecord(readValue(record, key));
}

/** 从对象中按固定字段名读取对象数组。 */
function readRecordList(record: ConfigRecord | undefined, key: ConfigFieldKey) {
  const value = readValue(record, key);
  const singleRecord = toOptionalRecord(value);
  // Gorse 服务若返回单个对象而非对象数组，也按一项配置展示，避免有数据时落到空态。
  if (singleRecord) return [singleRecord];
  if (!Array.isArray(value)) return [];
  return value.map(toOptionalRecord).filter((item): item is ConfigRecord => Boolean(item));
}

/** 将配置响应转换成普通对象。 */
function toRecord(value: unknown): ConfigRecord {
  return toOptionalRecord(value) ?? {};
}

/** 将未知值转换成可读取的普通对象。 */
function toOptionalRecord(value: unknown): ConfigRecord | undefined {
  if (typeof value !== "object" || value === null || Array.isArray(value)) return undefined;
  return value as ConfigRecord;
}

/** 格式化标签列表。 */
function formatTags(value: unknown) {
  if (!Array.isArray(value)) return value === undefined || value === null || value === "" ? [] : [formatValue(value)];
  return value.map(formatValue).filter(item => item !== "未配置");
}

/** 格式化基础配置值，避免整块 JSON 展示。 */
function formatValue(value: unknown) {
  if (value === undefined || value === null || value === "") return "未配置";
  if (Array.isArray(value))
    return (
      value
        .map(formatValue)
        .filter(item => item !== "未配置")
        .join("、") || "未配置"
    );
  if (typeof value === "boolean") return value ? "是" : "否";
  if (typeof value === "object") return "未配置";
  return String(value);
}

onMounted(() => {
  loadConfig();
});
</script>

<style scoped lang="scss">
.gorse-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.gorse-tabs-card {
  border-color: var(--admin-page-card-border);
  background: var(--admin-page-card-bg);
  box-shadow: var(--admin-page-shadow);
}

.gorse-config-tabs {
  :deep(.el-tabs__header) {
    margin-bottom: 18px;
  }

  :deep(.el-tabs__item) {
    color: var(--admin-page-text-secondary);
    font-weight: 600;
  }

  :deep(.el-tabs__item.is-active) {
    color: var(--el-color-primary);
  }
}

.gorse-config-groups {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.gorse-config-group {
  display: flex;
  flex-direction: column;
  gap: 12px;

  &__title {
    padding-left: 10px;
    border-left: 3px solid var(--el-color-primary);
    color: var(--admin-page-text-primary);
    font-weight: 600;
    line-height: 20px;
  }
}

.gorse-config-list {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.gorse-config-item {
  display: grid;
  grid-template-columns: 180px minmax(0, 1fr);
  gap: 12px;
  align-items: start;
  padding: 12px;
  border: 1px solid var(--admin-page-card-border-soft);
  border-radius: 10px;
  background: var(--el-fill-color-lighter);

  label {
    color: var(--admin-page-text-primary);
    font-weight: 600;
    line-height: 32px;
    word-break: break-word;
  }

  &__value {
    min-width: 0;
  }
}

.gorse-config-tags {
  display: flex;
  min-height: 32px;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.gorse-config-empty {
  color: var(--admin-page-text-secondary);
  line-height: 32px;
}

@media (max-width: 1200px) {
  .gorse-config-list {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 700px) {
  .gorse-config-item {
    grid-template-columns: 1fr;
  }
}
</style>
