<template>
  <div class="remote-overview-page">
    <OverviewStatsGrid :loading="statsLoading" :stats="statCards" />
    <OverviewPerformancePanel ref="performancePanelRef" :positive-feedback-types="positiveFeedbackTypes" />
    <OverviewRecommendationTable
      ref="recommendationTableRef"
      :recommenders="recommenders"
      :categories="categories"
      :cache-size="cacheSize"
    />
  </div>
</template>

<script setup lang="ts">
import { nextTick, onMounted, ref } from "vue";
import { ElMessage } from "element-plus";
import OverviewStatsGrid from "./components/OverviewStatsGrid.vue";
import OverviewPerformancePanel from "./components/OverviewPerformancePanel.vue";
import OverviewRecommendationTable from "./components/OverviewRecommendationTable.vue";
import { defRecommendRemoteService } from "@/api/admin/recommend_remote";
import {
  buildRemoteTimeseriesMetric,
  parseRemoteCategories,
  parseRemoteJson,
  remoteOverviewMetrics,
  resolveRemoteArray,
  resolveRemoteNumber,
  resolveRemoteValue,
  type RemoteRecord,
  type RemoteTimeseriesMetric
} from "../utils";

defineOptions({
  name: "RecommendRemoteOverview"
});

const statsLoading = ref(false);
const statCards = ref<RemoteTimeseriesMetric[]>(remoteOverviewMetrics.map(item => buildRemoteTimeseriesMetric(item, "[]")));
const positiveFeedbackTypes = ref<string[]>([]);
const recommenders = ref<string[]>(["latest"]);
const categories = ref<string[]>([]);
const cacheSize = ref(100);
const performancePanelRef = ref<InstanceType<typeof OverviewPerformancePanel>>();
const recommendationTableRef = ref<InstanceType<typeof OverviewRecommendationTable>>();

/** 页面初始化，加载远程推荐统计、性能趋势和非个性化推荐。 */
async function loadOverview() {
  // 配置、分类和统计相互独立，任一接口异常都不阻塞其他区块渲染。
  await Promise.allSettled([loadConfig(), loadCategories(), loadStats()]);
  await nextTick();
  await Promise.allSettled([performancePanelRef.value?.refresh(), recommendationTableRef.value?.refresh()]);
}

/** 加载远程推荐配置，用于反馈类型、缓存数量和非个性化推荐器。 */
async function loadConfig() {
  try {
    const data = await defRecommendRemoteService.GetRecommendRemoteConfig({});
    const config = parseRemoteJson(data.json) as RemoteRecord;
    const recommend = resolveRemoteValue(config, ["recommend"]);
    const database = resolveRemoteValue(config, ["database"]);
    if (typeof database === "object" && database !== null) {
      cacheSize.value = resolveRemoteNumber(database as RemoteRecord, ["cache_size", "cacheSize"]) || 100;
    }
    if (typeof recommend === "object" && recommend !== null) {
      const recommendRecord = recommend as RemoteRecord;
      const dataSource = resolveRemoteValue(recommendRecord, ["data_source", "dataSource"]);
      if (typeof dataSource === "object" && dataSource !== null) {
        positiveFeedbackTypes.value = resolveRemoteArray(dataSource as RemoteRecord, [
          "positive_feedback_types",
          "positiveFeedbackTypes"
        ]).map(String);
      }
      const nonPersonalized = resolveRemoteArray(recommendRecord, ["non-personalized", "nonPersonalized"]);
      recommenders.value = ["latest"].concat(
        nonPersonalized
          .filter(item => typeof item === "object" && item !== null)
          .map(item => `non-personalized/${String(resolveRemoteValue(item as RemoteRecord, ["name", "Name"]) ?? "")}`)
          .filter(item => item !== "non-personalized/")
      );
    }
  } catch (error) {
    ElMessage.error("加载推荐配置失败");
    throw error;
  }
}

/** 加载远程推荐顶部五个统计卡。 */
async function loadStats() {
  statsLoading.value = true;
  try {
    const responses = await Promise.all(
      remoteOverviewMetrics.map(item =>
        defRecommendRemoteService.GetRecommendRemoteTimeseries({ name: item.name, begin: "", end: "" })
      )
    );
    statCards.value = remoteOverviewMetrics.map((item, index) =>
      buildRemoteTimeseriesMetric(item, responses[index]?.json ?? "[]")
    );
  } catch (error) {
    ElMessage.error("加载概览统计失败");
    throw error;
  } finally {
    statsLoading.value = false;
  }
}

/** 加载远程推荐分类下拉。 */
async function loadCategories() {
  try {
    const data = await defRecommendRemoteService.GetRecommendRemoteCategories({});
    categories.value = parseRemoteCategories(data.json).map(item => item.name);
  } catch (error) {
    ElMessage.error("加载推荐分类失败");
    throw error;
  }
}

onMounted(() => {
  loadOverview();
});
</script>

<style scoped lang="scss">
.remote-overview-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
</style>
