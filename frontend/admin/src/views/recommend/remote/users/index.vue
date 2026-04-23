<template>
  <div class="remote-page">
    <el-card class="remote-page__toolbar" shadow="never">
      <el-form :inline="true" :model="query" @submit.prevent>
        <el-form-item label="用户编号">
          <el-input v-model.trim="query.id" clearable placeholder="输入编号可查询单个用户" @keyup.enter="loadUsers(true)" />
        </el-form-item>
        <el-form-item label="返回数量">
          <el-input-number v-model="query.n" :min="1" :max="200" :step="10" controls-position="right" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="loading" @click="loadUsers(true)">查询</el-button>
          <el-button :disabled="!nextCursor" :loading="loading" @click="loadNextPage">下一页</el-button>
          <el-button @click="resetQuery">重置</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card shadow="never">
      <el-table v-loading="loading" :data="list" :row-key="getUserId" border>
        <el-table-column prop="UserId" label="用户编号" min-width="180">
          <template #default="{ row }">{{ getUserId(row) || "--" }}</template>
        </el-table-column>
        <el-table-column prop="Labels" label="标签" min-width="220">
          <template #default="{ row }">{{ formatRemoteCell(row.Labels ?? row.labels) }}</template>
        </el-table-column>
        <el-table-column prop="Comment" label="备注" min-width="180">
          <template #default="{ row }">{{ formatRemoteCell(row.Comment ?? row.comment) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openDetail(row)">详情</el-button>
            <el-button link type="danger" @click="deleteUser(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <JsonPanel
      title="原始响应"
      description="远程推荐用户接口原始 JSON。"
      :json="rawJson"
      :loading="loading"
      @refresh="loadUsers(false)"
    />

    <el-drawer v-model="detailVisible" title="用户详情" size="50%">
      <JsonPanel title="详情 JSON" :json="detailJson" :loading="detailLoading" @refresh="reloadDetail" />
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { defRecommendRemoteService } from "@/api/admin/recommend_remote";
import JsonPanel from "../components/JsonPanel.vue";
import { formatRemoteCell, parseRemoteCursorList, parseRemoteJson, resolveRemoteId, type RemoteRecord } from "../utils";

defineOptions({
  name: "RecommendRemoteUsers"
});

const userIdKeys = ["UserId", "userId", "user_id", "Id", "id"];

const query = reactive({
  id: "",
  cursor: "",
  n: 20
});

const loading = ref(false);
const detailLoading = ref(false);
const detailVisible = ref(false);
const nextCursor = ref("");
const currentDetailId = ref("");
const rawJson = ref("{}");
const detailJson = ref("{}");
const list = ref<RemoteRecord[]>([]);

/** 读取远程用户编号。 */
function getUserId(row: RemoteRecord) {
  return resolveRemoteId(row, userIdKeys);
}

/** 加载远程推荐用户列表或单个用户。 */
async function loadUsers(resetCursor = false) {
  if (resetCursor) query.cursor = "";
  loading.value = true;
  try {
    // 输入用户编号时直接查询单个用户，避免远程列表接口不支持模糊筛选。
    if (query.id) {
      const data = await defRecommendRemoteService.GetRecommendRemoteUser({ id: query.id });
      const record = parseRemoteJson(data.json);
      list.value = typeof record === "object" && record !== null && !Array.isArray(record) ? [record as RemoteRecord] : [];
      rawJson.value = data.json;
      nextCursor.value = "";
      return;
    }

    const data = await defRecommendRemoteService.PageRecommendRemoteUsers({
      id: "",
      cursor: query.cursor,
      n: query.n
    });
    const page = parseRemoteCursorList(data.json, ["Users", "users"]);
    list.value = page.list;
    nextCursor.value = page.cursor;
    rawJson.value = data.json;
  } catch (error) {
    ElMessage.error("加载远程推荐用户失败");
    throw error;
  } finally {
    loading.value = false;
  }
}

/** 加载下一页远程用户。 */
async function loadNextPage() {
  // 没有下一页游标时，当前页已经是最后一页。
  if (!nextCursor.value) {
    ElMessage.warning("暂无下一页数据");
    return;
  }
  query.cursor = nextCursor.value;
  await loadUsers(false);
}

/** 重置查询条件并重新加载用户列表。 */
function resetQuery() {
  query.id = "";
  query.cursor = "";
  query.n = 20;
  loadUsers(true);
}

/** 打开远程用户详情。 */
async function openDetail(row: RemoteRecord) {
  const id = getUserId(row);
  // 缺少用户编号时，无法继续查询远程详情。
  if (!id) {
    ElMessage.warning("用户编号为空，无法查看详情");
    return;
  }
  currentDetailId.value = id;
  detailVisible.value = true;
  await reloadDetail();
}

/** 重新加载当前远程用户详情。 */
async function reloadDetail() {
  // 当前没有选中用户时，不触发远程详情请求。
  if (!currentDetailId.value) return;

  detailLoading.value = true;
  try {
    const data = await defRecommendRemoteService.GetRecommendRemoteUser({ id: currentDetailId.value });
    detailJson.value = data.json;
  } catch (error) {
    ElMessage.error("加载用户详情失败");
    throw error;
  } finally {
    detailLoading.value = false;
  }
}

/** 删除远程推荐用户。 */
async function deleteUser(row: RemoteRecord) {
  const id = getUserId(row);
  // 缺少用户编号时，无法定位删除对象。
  if (!id) {
    ElMessage.warning("用户编号为空，无法删除");
    return;
  }
  await ElMessageBox.confirm(`是否确定删除远程推荐用户？\n用户编号：${id}`, "警告", {
    confirmButtonText: "确认",
    cancelButtonText: "取消",
    type: "warning"
  });
  await defRecommendRemoteService.DeleteRecommendRemoteUser({ id });
  ElMessage.success("删除远程推荐用户成功");
  await loadUsers(false);
}

onMounted(() => {
  loadUsers(true);
});
</script>

<style scoped lang="scss">
.remote-page {
  display: flex;
  flex-direction: column;
  gap: 16px;

  &__toolbar {
    border-color: var(--el-border-color-light);
  }

  &__toolbar :deep(.el-form-item) {
    margin-bottom: 0;
  }
}
</style>
