<template>
  <div class="remote-page">
    <el-card class="remote-page__hero" shadow="never">
      <div>
        <p>远程推荐</p>
        <h2>推荐概览</h2>
        <span>所有数据实时来自远程推荐引擎，管理后台不保存副本。</span>
      </div>
      <el-button type="primary" :loading="loading" @click="loadOverview">刷新概览</el-button>
    </el-card>

    <div class="remote-page__grid">
      <JsonPanel
        title="运行统计"
        description="展示推荐引擎的核心统计信息。"
        :json="overviewJson"
        :loading="loading"
        @refresh="loadOverview"
      />
      <JsonPanel
        title="分类数据"
        description="展示远程推荐商品分类分布。"
        :json="categoriesJson"
        :loading="loading"
        @refresh="loadOverview"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue";
import { ElMessage } from "element-plus";
import { defRecommendRemoteService } from "@/api/admin/recommend_remote";
import JsonPanel from "../components/JsonPanel.vue";

defineOptions({
  name: "RecommendRemoteOverview"
});

const loading = ref(false);
const overviewJson = ref("{}");
const categoriesJson = ref("{}");

/** 加载推荐概览与分类数据。 */
async function loadOverview() {
  loading.value = true;
  try {
    const overview = await defRecommendRemoteService.GetRecommendRemoteOverview({});
    const categories = await defRecommendRemoteService.GetRecommendRemoteCategories({});
    overviewJson.value = overview.json;
    categoriesJson.value = categories.json;
  } catch (error) {
    ElMessage.error("加载推荐概览失败");
    throw error;
  } finally {
    loading.value = false;
  }
}

onMounted(() => {
  loadOverview();
});
</script>

<style scoped lang="scss">
.remote-page {
  display: flex;
  flex-direction: column;
  gap: 16px;

  &__hero {
    border-color: var(--el-border-color-light);
    background: radial-gradient(circle at top right, var(--el-color-primary-light-9), transparent 36%), var(--el-bg-color);
  }

  &__hero :deep(.el-card__body) {
    display: flex;
    gap: 16px;
    align-items: center;
    justify-content: space-between;
  }

  &__hero p {
    margin: 0 0 6px;
    color: var(--el-color-primary);
    font-weight: 600;
  }

  &__hero h2 {
    margin: 0 0 8px;
    color: var(--el-text-color-primary);
    font-size: 26px;
  }

  &__hero span {
    color: var(--el-text-color-secondary);
  }

  &__grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 16px;
  }
}

@media (max-width: 900px) {
  .remote-page__hero :deep(.el-card__body) {
    align-items: flex-start;
    flex-direction: column;
  }

  .remote-page__grid {
    grid-template-columns: 1fr;
  }
}
</style>
