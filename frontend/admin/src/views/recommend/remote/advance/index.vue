<template>
  <div class="remote-page remote-advance-page">
    <el-card class="remote-hero-card" shadow="never">
      <div class="remote-hero-card__content">
        <p>Gorse Dashboard</p>
        <h2>高级调试</h2>
        <span>参照 Gorse Advance 页面组织远程数据导出与导入，管理后台不落库。</span>
      </div>
    </el-card>

    <section class="remote-advance-page__grid">
      <el-card class="remote-section-card" shadow="never">
        <template #header>
          <div class="remote-section-card__header">
            <strong>Data Export</strong>
            <span>导出远程推荐当前页数据</span>
          </div>
        </template>

        <el-form :model="form" label-width="110px">
          <el-form-item label="数据类型">
            <el-select v-model="form.type" style="width: 240px" @change="handleTypeChange">
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
            <el-button :disabled="!exportNextCursor" :loading="exportLoading" @click="exportNextPage">导出下一页</el-button>
          </el-form-item>
        </el-form>
      </el-card>

      <el-card class="remote-section-card" shadow="never">
        <template #header>
          <div class="remote-section-card__header">
            <strong>Data Import</strong>
            <span>写入远程推荐引擎</span>
          </div>
        </template>

        <el-form label-width="110px">
          <el-form-item label="导入类型">
            <el-tag effect="light">{{ selectedDataTypeLabel }}</el-tag>
          </el-form-item>
          <el-form-item label="导入 JSON">
            <el-input v-model="importJson" type="textarea" :rows="11" placeholder="粘贴远程推荐用户或商品 JSON 数组" />
          </el-form-item>
          <el-form-item>
            <el-button type="success" :loading="importLoading" @click="importData">导入到远程推荐</el-button>
          </el-form-item>
        </el-form>
      </el-card>
    </section>

    <el-card class="remote-section-card" shadow="never">
      <template #header>
        <div class="remote-section-card__header">
          <strong>导出预览</strong>
          <span>当前页 {{ exportRows.length }} 条，下一页游标：{{ exportNextCursor || "无" }}</span>
        </div>
      </template>

      <el-table v-loading="exportLoading" :data="exportRows" border>
        <el-table-column label="编号" min-width="180">
          <template #default="{ row }">{{ getExportRowId(row) || "--" }}</template>
        </el-table-column>
        <el-table-column v-if="form.type === 'items'" label="分类" min-width="220">
          <template #default="{ row }">{{ formatRemoteCell(resolveRemoteValue(row, ["Categories", "categories"])) }}</template>
        </el-table-column>
        <el-table-column v-if="form.type === 'items'" label="隐藏" min-width="100" align="center">
          <template #default="{ row }">
            <el-tag
              :type="resolveRemoteBoolean(row, ['IsHidden', 'isHidden', 'is_hidden']) ? 'warning' : 'success'"
              effect="light"
            >
              {{ resolveRemoteBoolean(row, ["IsHidden", "isHidden", "is_hidden"]) ? "隐藏" : "展示" }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="标签" min-width="220">
          <template #default="{ row }">{{ formatRemoteCell(resolveRemoteValue(row, ["Labels", "labels"])) }}</template>
        </el-table-column>
        <el-table-column label="描述" min-width="220">
          <template #default="{ row }">{{
            formatRemoteCell(resolveRemoteValue(row, ["Comment", "comment", "Description", "description"]))
          }}</template>
        </el-table-column>
        <el-table-column label="更新时间" min-width="180">
          <template #default="{ row }">
            {{
              formatRemoteDateTime(
                resolveRemoteValue(row, ["LastUpdateTime", "lastUpdateTime", "last_update_time", "Timestamp", "timestamp"])
              )
            }}
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-card class="remote-section-card remote-danger-card" shadow="never">
      <template #header>
        <div class="remote-section-card__header">
          <strong>Danger Zone</strong>
          <span>清空远程推荐数据存储与缓存</span>
        </div>
      </template>
      <div class="remote-danger-action">
        <el-button type="danger" plain :loading="purgeLoading" @click="openPurgeDialog">Purge Database</el-button>
        <span>Purge all data in data storage and cache storage.</span>
      </div>
    </el-card>

    <el-dialog v-model="purgeDialogVisible" title="Are you absolutely sure?" width="560px" append-to-body>
      <div class="remote-purge-dialog">
        <p>This action <strong>cannot</strong> be undone. This will permanently:</p>
        <el-checkbox-group v-model="purgeCheckList" class="remote-purge-dialog__checks">
          <el-checkbox label="delete_users">Delete all users.</el-checkbox>
          <el-checkbox label="delete_items">Delete all items.</el-checkbox>
          <el-checkbox label="delete_feedback">Delete all feedbacks.</el-checkbox>
          <el-checkbox label="delete_cache">Delete all caches.</el-checkbox>
        </el-checkbox-group>
      </div>
      <template #footer>
        <el-button @click="purgeDialogVisible = false">取消</el-button>
        <el-button type="danger" plain :disabled="!canPurgeData" :loading="purgeLoading" @click="purgeData">
          I understand the consequences, purge all data
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { defRecommendRemoteService } from "@/api/admin/recommend_remote";
import {
  formatRemoteCell,
  formatRemoteDateTime,
  resolveRemoteBoolean,
  resolveRemoteId,
  resolveRemoteValue,
  type RemoteRecord
} from "../utils";

defineOptions({
  name: "RemoteAdvance"
});

/** 高级数据类型选项。 */
interface DataTypeOption {
  /** 选项名称。 */
  label: string;
  /** 远程数据类型。 */
  value: "users" | "items";
}

const dataTypes: DataTypeOption[] = [
  { label: "用户数据", value: "users" },
  { label: "商品数据", value: "items" }
];

const form = reactive({
  type: "users" as DataTypeOption["value"],
  cursor: "",
  n: 100
});

const exportLoading = ref(false);
const importLoading = ref(false);
const exportRows = ref<RemoteRecord[]>([]);
const exportNextCursor = ref("");
const importJson = ref("[]");
const purgeLoading = ref(false);
const purgeDialogVisible = ref(false);
const purgeCheckList = ref<string[]>([]);

/** 当前选择的数据类型文案。 */
const selectedDataTypeLabel = computed(() => dataTypes.find(item => item.value === form.type)?.label ?? "用户数据");
const canPurgeData = computed(() => purgeCheckList.value.length === purgeConfirmItems.length);
const purgeConfirmItems = ["delete_users", "delete_items", "delete_feedback", "delete_cache"];

/** 切换数据类型时清空导出预览与游标。 */
function handleTypeChange() {
  form.cursor = "";
  exportRows.value = [];
  exportNextCursor.value = "";
}

/** 导出远程推荐数据当前页。 */
async function exportData() {
  exportLoading.value = true;
  try {
    const data = await defRecommendRemoteService.ExportData({
      type: form.type,
      cursor: form.cursor,
      n: form.n
    });
    exportRows.value = data.list.map(item => (item.raw ?? item) as RemoteRecord);
    exportNextCursor.value = data.cursor;
  } catch (error) {
    ElMessage.error("导出远程推荐数据失败");
    throw error;
  } finally {
    exportLoading.value = false;
  }
}

/** 根据导出结果游标继续导出下一页。 */
async function exportNextPage() {
  if (!exportNextCursor.value) {
    ElMessage.warning("暂无下一页数据");
    return;
  }
  form.cursor = exportNextCursor.value;
  await exportData();
}

/** 导入远程推荐数据。 */
async function importData() {
  const body = importJson.value.trim();
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

  await ElMessageBox.confirm(`是否确定导入${selectedDataTypeLabel.value}到远程推荐？该操作会直接写入远程推荐引擎。`, "警告", {
    confirmButtonText: "确认",
    cancelButtonText: "取消",
    type: "warning"
  });

  importLoading.value = true;
  try {
    await defRecommendRemoteService.ImportData({
      type: form.type,
      json: body
    });
    ElMessage.success("导入远程推荐数据成功");
    await exportData();
  } catch (error) {
    ElMessage.error("导入远程推荐数据失败");
    throw error;
  } finally {
    importLoading.value = false;
  }
}

/** 读取导出预览行编号。 */
function getExportRowId(row: RemoteRecord) {
  return form.type === "users"
    ? resolveRemoteId(row, ["UserId", "userId", "user_id", "Id", "id"])
    : resolveRemoteId(row, ["ItemId", "itemId", "item_id", "Id", "id"]);
}

/** 打开远程推荐清空确认弹窗。 */
function openPurgeDialog() {
  purgeCheckList.value = [];
  purgeDialogVisible.value = true;
}

/** 清空远程推荐用户、商品、反馈和缓存数据。 */
async function purgeData() {
  if (!canPurgeData.value) {
    ElMessage.warning("请先勾选全部清空确认项");
    return;
  }

  await ElMessageBox.confirm("是否确定清空远程推荐全部用户、商品、反馈和缓存？该操作不可恢复。", "危险操作", {
    confirmButtonText: "确认清空",
    cancelButtonText: "取消",
    type: "error"
  });

  purgeLoading.value = true;
  try {
    await defRecommendRemoteService.PurgeData({ checkList: purgeConfirmItems });
    ElMessage.success("远程推荐数据已清空");
    purgeDialogVisible.value = false;
    handleTypeChange();
  } catch (error) {
    ElMessage.error("清空远程推荐数据失败");
    throw error;
  } finally {
    purgeLoading.value = false;
  }
}
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

.remote-advance-page__grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
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

.remote-danger-card {
  :deep(.el-card__header) {
    border-bottom-color: var(--el-color-danger-light-7);
  }
}

.remote-danger-action {
  display: grid;
  grid-template-columns: 220px minmax(0, 1fr);
  gap: 18px;
  align-items: center;

  span {
    color: var(--admin-page-text-secondary);
  }
}

.remote-purge-dialog {
  p {
    margin: 0 0 16px;
    color: var(--admin-page-text-secondary);
    font-size: 16px;
    line-height: 1.7;

    strong {
      color: var(--el-color-danger);
    }
  }

  &__checks {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }
}

@media (max-width: 900px) {
  .remote-hero-card :deep(.el-card__body) {
    align-items: flex-start;
    flex-direction: column;
  }

  .remote-advance-page__grid {
    grid-template-columns: 1fr;
  }

  .remote-danger-action {
    grid-template-columns: 1fr;
  }
}
</style>
