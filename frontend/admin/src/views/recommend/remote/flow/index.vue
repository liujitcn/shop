<template>
  <div class="remote-page">
    <el-alert
      title="推荐编排保存后会直接写入远程推荐引擎，并由远程推荐引擎接管生效流程。"
      type="warning"
      :closable="false"
      show-icon
    />

    <el-card shadow="never">
      <template #header>
        <div class="remote-page__header">
          <strong>编排配置</strong>
          <div>
            <el-button :loading="loading" @click="loadFlow">刷新</el-button>
            <el-button type="primary" :loading="saving" @click="saveFlow">保存并生效</el-button>
            <el-button type="danger" plain :loading="resetting" @click="resetFlow">重置远程配置</el-button>
          </div>
        </div>
      </template>
      <el-input v-model="configJson" type="textarea" :rows="22" placeholder="远程推荐编排 JSON 配置" />
    </el-card>

    <JsonPanel
      title="配置结构"
      description="远程推荐引擎返回的编排配置结构说明。"
      :json="schemaJson"
      :loading="loading"
      @refresh="loadFlow"
    />
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { defRecommendRemoteService } from "@/api/admin/recommend_remote";
import { formatRemoteJson } from "../utils";
import JsonPanel from "../components/JsonPanel.vue";

defineOptions({
  name: "RecommendRemoteFlow"
});

const loading = ref(false);
const saving = ref(false);
const resetting = ref(false);
const configJson = ref("{}");
const schemaJson = ref("{}");

/** 加载推荐编排配置和结构。 */
async function loadFlow() {
  loading.value = true;
  try {
    const config = await defRecommendRemoteService.GetRecommendRemoteFlowConfig({});
    const schema = await defRecommendRemoteService.GetRecommendRemoteFlowSchema({});
    configJson.value = formatRemoteJson(config.json);
    schemaJson.value = schema.json;
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
  // 编排配置为空时，不允许提交到远程推荐引擎。
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

  &__header {
    display: flex;
    gap: 16px;
    align-items: center;
    justify-content: space-between;
  }
}

@media (max-width: 900px) {
  .remote-page__header {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
