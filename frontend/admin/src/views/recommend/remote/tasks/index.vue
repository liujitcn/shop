<template>
  <div class="remote-page">
    <JsonPanel
      title="任务状态"
      description="展示远程推荐引擎当前任务、调度与执行状态。"
      :json="json"
      :loading="loading"
      @refresh="loadTasks"
    />
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue";
import { ElMessage } from "element-plus";
import { defRecommendRemoteService } from "@/api/admin/recommend_remote";
import JsonPanel from "../components/JsonPanel.vue";

defineOptions({
  name: "RecommendRemoteTasks"
});

const loading = ref(false);
const json = ref("{}");

/** 加载远程推荐任务状态。 */
async function loadTasks() {
  loading.value = true;
  try {
    const data = await defRecommendRemoteService.GetRecommendRemoteTasks({});
    json.value = data.json;
  } catch (error) {
    ElMessage.error("加载任务状态失败");
    throw error;
  } finally {
    loading.value = false;
  }
}

onMounted(() => {
  loadTasks();
});
</script>

<style scoped lang="scss">
.remote-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
</style>
