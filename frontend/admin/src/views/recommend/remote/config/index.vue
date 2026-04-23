<template>
  <div class="remote-page">
    <el-alert title="当前页面只读取远程推荐配置，不在管理后台保存配置副本。" type="info" :closable="false" show-icon />
    <JsonPanel title="推荐配置" description="展示远程推荐当前生效配置。" :json="json" :loading="loading" @refresh="loadConfig" />
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue";
import { ElMessage } from "element-plus";
import { defRecommendRemoteService } from "@/api/admin/recommend_remote";
import JsonPanel from "../components/JsonPanel.vue";

defineOptions({
  name: "RecommendRemoteConfig"
});

const loading = ref(false);
const json = ref("{}");

/** 加载远程推荐配置。 */
async function loadConfig() {
  loading.value = true;
  try {
    const data = await defRecommendRemoteService.GetRecommendRemoteConfig({});
    json.value = data.json;
  } catch (error) {
    ElMessage.error("加载推荐配置失败");
    throw error;
  } finally {
    loading.value = false;
  }
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
</style>
