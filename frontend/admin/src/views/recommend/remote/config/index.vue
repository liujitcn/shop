<template>
  <div v-loading="loading" class="remote-page remote-config-page">
    <el-card class="remote-hero-card" shadow="never">
      <div class="remote-hero-card__content">
        <p>Gorse Dashboard</p>
        <h2>推荐配置</h2>
        <span>参照 Gorse Settings 页面按配置分组展示字段，当前页面只读取远程推荐配置，不保存副本。</span>
      </div>
      <div class="remote-hero-card__actions">
        <el-button type="primary" :loading="loading" @click="loadConfig">刷新配置</el-button>
      </div>
    </el-card>

    <template v-if="configSections.length">
      <el-card v-for="section in configSections" :key="section.name" class="remote-section-card" shadow="never">
        <template #header>
          <div class="remote-section-card__header">
            <strong>{{ section.name }}</strong>
            <span>{{ section.fields.length }} 个配置项</span>
          </div>
        </template>

        <div class="remote-config-list">
          <div v-for="field in section.fields" :key="`${section.name}-${field.name}`" class="remote-config-item">
            <label>{{ field.name }}</label>
            <el-input v-if="!field.complex" :model-value="field.text" readonly />
            <el-input v-else :model-value="field.text" type="textarea" :rows="resolveTextareaRows(field.text)" readonly />
          </div>
        </div>
      </el-card>
    </template>
    <el-empty v-else description="暂无推荐配置" />
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue";
import { ElMessage } from "element-plus";
import { defRecommendRemoteService } from "@/api/admin/recommend_remote";
import { buildRemoteConfigSections, type RemoteConfigSection } from "../utils";

defineOptions({
  name: "RemoteConfig"
});

const loading = ref(false);
const configSections = ref<RemoteConfigSection[]>([]);

/** 加载远程推荐配置。 */
async function loadConfig() {
  loading.value = true;
  try {
    const data = await defRecommendRemoteService.GetConfig({});
    configSections.value = buildRemoteConfigSections(data.config);
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

onMounted(() => {
  loadConfig();
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

.remote-config-list {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.remote-config-item {
  display: grid;
  grid-template-columns: 220px minmax(0, 1fr);
  gap: 16px;
  align-items: start;
  padding: 14px;
  border: 1px solid var(--admin-page-card-border-soft);
  border-radius: 12px;
  background: var(--el-fill-color-lighter);

  label {
    color: var(--admin-page-text-primary);
    font-weight: 600;
    line-height: 32px;
    word-break: break-word;
  }
}

@media (max-width: 900px) {
  .remote-hero-card :deep(.el-card__body) {
    align-items: flex-start;
    flex-direction: column;
  }

  .remote-config-item {
    grid-template-columns: 1fr;
  }
}
</style>
