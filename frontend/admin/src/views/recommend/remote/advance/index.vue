<template>
  <div class="remote-page">
    <el-alert title="高级功能直接调用远程推荐原生数据接口，管理后台不落库。" type="warning" :closable="false" show-icon />

    <el-card shadow="never">
      <template #header>
        <strong>数据导入导出</strong>
      </template>
      <el-form :model="form" label-width="110px">
        <el-form-item label="数据类型">
          <el-select v-model="form.type" style="width: 240px">
            <el-option v-for="item in dataTypes" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="返回数量">
          <el-input-number v-model="form.n" :min="1" :max="500" :step="50" controls-position="right" />
        </el-form-item>
        <el-form-item label="游标">
          <el-input v-model.trim="form.cursor" clearable placeholder="继续导出下一页时填写上次返回的游标" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="exportLoading" @click="exportData">导出当前页</el-button>
        </el-form-item>
        <el-form-item label="导入 JSON">
          <el-input v-model="importJson" type="textarea" :rows="10" placeholder="粘贴远程推荐原生 JSON 数组" />
        </el-form-item>
        <el-form-item>
          <el-button type="success" :loading="importLoading" @click="importData">导入到远程推荐</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <JsonPanel
      title="导出结果"
      description="当前导出页的原始 JSON 响应。"
      :json="exportJson"
      :loading="exportLoading"
      @refresh="exportData"
    />
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { defRecommendRemoteService } from "@/api/admin/recommend_remote";
import JsonPanel from "../components/JsonPanel.vue";

defineOptions({
  name: "RecommendRemoteAdvance"
});

/** 高级数据类型选项。 */
interface DataTypeOption {
  /** 选项名称。 */
  label: string;
  /** 远程数据类型。 */
  value: string;
}

const dataTypes: DataTypeOption[] = [
  { label: "用户数据", value: "users" },
  { label: "商品数据", value: "items" }
];

const form = reactive({
  type: "users",
  cursor: "",
  n: 100
});

const exportLoading = ref(false);
const importLoading = ref(false);
const exportJson = ref("{}");
const importJson = ref("[]");

/** 导出远程推荐数据当前页。 */
async function exportData() {
  exportLoading.value = true;
  try {
    const data = await defRecommendRemoteService.ExportRecommendRemoteData({
      type: form.type,
      cursor: form.cursor,
      n: form.n
    });
    exportJson.value = data.json;
  } catch (error) {
    ElMessage.error("导出远程推荐数据失败");
    throw error;
  } finally {
    exportLoading.value = false;
  }
}

/** 导入远程推荐数据。 */
async function importData() {
  const body = importJson.value.trim();
  // 导入内容为空时，不允许提交到远程推荐引擎。
  if (!body) {
    ElMessage.warning("请先填写导入 JSON");
    return;
  }
  try {
    JSON.parse(body);
  } catch {
    ElMessage.error("导入 JSON 格式不正确");
    return;
  }

  await ElMessageBox.confirm("是否确定导入数据到远程推荐？该操作会直接写入远程推荐引擎。", "警告", {
    confirmButtonText: "确认",
    cancelButtonText: "取消",
    type: "warning"
  });

  importLoading.value = true;
  try {
    await defRecommendRemoteService.ImportRecommendRemoteData({
      type: form.type,
      json: body
    });
    ElMessage.success("导入远程推荐数据成功");
  } catch (error) {
    ElMessage.error("导入远程推荐数据失败");
    throw error;
  } finally {
    importLoading.value = false;
  }
}
</script>

<style scoped lang="scss">
.remote-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
</style>
