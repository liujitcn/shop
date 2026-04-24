<template>
  <div v-loading="loading" class="remote-page remote-flow-page">
    <el-card class="remote-hero-card" shadow="never">
      <div class="remote-hero-card__content">
        <p>Gorse Dashboard</p>
        <h2>推荐编排</h2>
        <span>参照 Gorse RecFlow 页面展示推荐节点与配置分组，保存后直接写入远程推荐引擎。</span>
      </div>
      <div class="remote-hero-card__actions">
        <el-button :loading="loading" @click="loadFlow">刷新</el-button>
        <el-button type="primary" :loading="saving" @click="saveFlow">保存并生效</el-button>
        <el-button type="danger" plain :loading="resetting" @click="resetFlow">重置远程配置</el-button>
      </div>
    </el-card>

    <el-card class="remote-section-card" shadow="never">
      <template #header>
        <div class="remote-section-card__header">
          <strong>节点工具箱</strong>
          <span>对应 Gorse RecFlow 顶部可拖拽推荐节点</span>
        </div>
      </template>
      <div class="remote-node-list">
        <article v-for="node in flowNodes" :key="node.type" class="remote-node-card">
          <el-icon><component :is="node.icon" /></el-icon>
          <div>
            <strong>{{ node.label }}</strong>
            <span>{{ node.description }}</span>
          </div>
        </article>
      </div>
    </el-card>

    <section class="remote-flow-page__grid">
      <el-card class="remote-section-card" shadow="never">
        <template #header>
          <div class="remote-section-card__header">
            <strong>当前配置分组</strong>
            <span>{{ configSections.length }} 个分组</span>
          </div>
        </template>
        <div v-if="configSections.length" class="remote-config-section-list">
          <section v-for="section in configSections" :key="section.name" class="remote-config-section">
            <h3>{{ section.name }}</h3>
            <div class="remote-config-fields">
              <div v-for="field in section.fields" :key="`${section.name}-${field.name}`" class="remote-config-field">
                <span>{{ field.name }}</span>
                <pre v-if="field.complex">{{ field.text }}</pre>
                <strong v-else>{{ field.text }}</strong>
              </div>
            </div>
          </section>
        </div>
        <el-empty v-else description="暂无推荐编排配置" />
      </el-card>

      <el-card class="remote-section-card" shadow="never">
        <template #header>
          <div class="remote-section-card__header">
            <strong>配置结构</strong>
            <span>{{ schemaSections.length }} 个结构分组</span>
          </div>
        </template>
        <div v-if="schemaSections.length" class="remote-config-section-list">
          <section v-for="section in schemaSections" :key="section.name" class="remote-config-section">
            <h3>{{ section.name }}</h3>
            <div class="remote-config-fields">
              <div v-for="field in section.fields" :key="`${section.name}-${field.name}`" class="remote-config-field">
                <span>{{ field.name }}</span>
                <pre v-if="field.complex">{{ field.text }}</pre>
                <strong v-else>{{ field.text }}</strong>
              </div>
            </div>
          </section>
        </div>
        <el-empty v-else description="暂无配置结构" />
      </el-card>
    </section>

    <el-card class="remote-section-card" shadow="never">
      <template #header>
        <div class="remote-section-card__header">
          <strong>配置编辑</strong>
          <span>高级编辑入口，提交前会校验 JSON 格式</span>
        </div>
      </template>
      <el-input v-model="configJson" type="textarea" :rows="22" placeholder="远程推荐编排 JSON 配置" />
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { Cloudy, Connection, DataAnalysis, Finished, Grid, Histogram, Refresh, Share } from "@element-plus/icons-vue";
import { defRecommendRemoteService } from "@/api/admin/recommend_remote";
import { formatRemoteJson, parseRemoteConfigSections, type RemoteConfigSection } from "../utils";

defineOptions({
  name: "RecommendRemoteFlow"
});

/** 推荐编排节点说明。 */
interface FlowNodeInfo {
  /** 节点类型。 */
  type: string;
  /** 节点名称。 */
  label: string;
  /** 节点说明。 */
  description: string;
  /** 节点图标。 */
  icon: unknown;
}

const flowNodes: FlowNodeInfo[] = [
  { type: "latest", label: "Latest", description: "最新商品兜底推荐", icon: Finished },
  { type: "collaborative", label: "Collaborative", description: "协同过滤推荐", icon: Connection },
  { type: "non-personalized", label: "Non-Personalized", description: "非个性化推荐器", icon: DataAnalysis },
  { type: "user-to-user", label: "User to User", description: "相似用户推荐", icon: Share },
  { type: "item-to-item", label: "Item to Item", description: "相似商品推荐", icon: Grid },
  { type: "external", label: "External", description: "外部脚本推荐器", icon: Cloudy },
  { type: "ranker", label: "Ranker", description: "排序模型节点", icon: Histogram },
  { type: "fallback", label: "Fallback", description: "兜底推荐链路", icon: Refresh }
];

const loading = ref(false);
const saving = ref(false);
const resetting = ref(false);
const configJson = ref("{}");
const configSections = ref<RemoteConfigSection[]>([]);
const schemaSections = ref<RemoteConfigSection[]>([]);

/** 加载推荐编排配置和结构。 */
async function loadFlow() {
  loading.value = true;
  try {
    const config = await defRecommendRemoteService.GetRecommendRemoteFlowConfig({});
    const schema = await defRecommendRemoteService.GetRecommendRemoteFlowSchema({});
    configJson.value = formatRemoteJson(config.json);
    configSections.value = parseRemoteConfigSections(config.json);
    schemaSections.value = parseRemoteConfigSections(schema.json);
  } catch (error) {
    ElMessage.error("加载推荐编排失败");
    throw error;
  } finally {
    loading.value = false;
  }
}

/** 保存推荐编排配置。 */
async function saveFlow() {
  const body = configJson.value.trim();
  if (!body) {
    ElMessage.warning("请先填写编排配置");
    return;
  }
  try {
    JSON.parse(body);
  } catch {
    ElMessage.error("编排配置 JSON 格式不正确");
    return;
  }

  saving.value = true;
  try {
    await defRecommendRemoteService.SaveRecommendRemoteFlowConfig({ json: body });
    ElMessage.success("推荐编排保存成功");
    await loadFlow();
  } catch (error) {
    ElMessage.error("保存推荐编排失败");
    throw error;
  } finally {
    saving.value = false;
  }
}

/** 重置推荐编排配置。 */
async function resetFlow() {
  await ElMessageBox.confirm("是否确定重置远程推荐编排？重置后将恢复远程默认配置。", "警告", {
    confirmButtonText: "确认",
    cancelButtonText: "取消",
    type: "warning"
  });

  resetting.value = true;
  try {
    await defRecommendRemoteService.ResetRecommendRemoteFlowConfig({});
    ElMessage.success("推荐编排已重置");
    await loadFlow();
  } catch (error) {
    ElMessage.error("重置推荐编排失败");
    throw error;
  } finally {
    resetting.value = false;
  }
}

onMounted(() => {
  loadFlow();
});
</script>

<style scoped lang="scss">
.remote-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.remote-hero-card,
.remote-section-card {
  border-color: var(--admin-page-card-border);
  background: var(--admin-page-card-bg);
  box-shadow: var(--admin-page-shadow);
}

.remote-hero-card {
  background: radial-gradient(circle at top right, var(--el-color-primary-light-9), transparent 38%), var(--admin-page-card-bg);

  :deep(.el-card__body) {
    display: flex;
    gap: 16px;
    align-items: center;
    justify-content: space-between;
  }

  &__content p {
    margin: 0 0 6px;
    color: var(--el-color-primary);
    font-weight: 600;
  }

  &__content h2 {
    margin: 0 0 8px;
    color: var(--admin-page-text-primary);
    font-size: 26px;
  }

  &__content span {
    color: var(--admin-page-text-secondary);
  }

  &__actions {
    display: flex;
    flex-shrink: 0;
    flex-wrap: wrap;
    gap: 8px;
    justify-content: flex-end;
  }
}

.remote-section-card__header {
  display: flex;
  gap: 8px;
  align-items: baseline;
  justify-content: space-between;

  strong {
    color: var(--admin-page-text-primary);
  }

  span {
    color: var(--admin-page-text-secondary);
    font-size: 13px;
  }
}

.remote-node-list {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}

.remote-node-card {
  display: flex;
  gap: 12px;
  align-items: center;
  padding: 14px;
  border: 1px solid var(--admin-page-card-border-soft);
  border-radius: 12px;
  background: var(--el-fill-color-lighter);

  .el-icon {
    flex-shrink: 0;
    color: var(--el-color-primary);
    font-size: 22px;
  }

  strong {
    display: block;
    color: var(--admin-page-text-primary);
  }

  span {
    display: block;
    margin-top: 4px;
    color: var(--admin-page-text-secondary);
    font-size: 12px;
  }
}

.remote-flow-page__grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
}

.remote-config-section-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
  max-height: 560px;
  overflow: auto;
}

.remote-config-section {
  padding: 14px;
  border: 1px solid var(--admin-page-card-border-soft);
  border-radius: 12px;
  background: var(--el-fill-color-lighter);

  h3 {
    margin: 0 0 12px;
    color: var(--admin-page-text-primary);
    font-size: 15px;
  }
}

.remote-config-fields {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.remote-config-field {
  display: grid;
  grid-template-columns: 180px minmax(0, 1fr);
  gap: 12px;
  align-items: start;

  span {
    color: var(--admin-page-text-secondary);
    line-height: 1.7;
    word-break: break-word;
  }

  strong,
  pre {
    margin: 0;
    color: var(--admin-page-text-primary);
    font-weight: 500;
    white-space: pre-wrap;
    word-break: break-word;
  }

  pre {
    max-height: 220px;
    padding: 10px;
    overflow: auto;
    background: var(--admin-page-card-bg);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
  }
}

@media (max-width: 1100px) {
  .remote-node-list {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .remote-flow-page__grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 700px) {
  .remote-hero-card :deep(.el-card__body) {
    align-items: flex-start;
    flex-direction: column;
  }

  .remote-node-list,
  .remote-config-field {
    grid-template-columns: 1fr;
  }
}
</style>
