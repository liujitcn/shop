<template>
  <div v-loading="loading" class="remote-page remote-tasks-page">
    <el-card class="remote-hero-card" shadow="never">
      <div class="remote-hero-card__content">
        <p>Gorse Dashboard</p>
        <h2>任务状态</h2>
        <span>按 Gorse 管理端 All Tasks 表格展示推荐引擎任务、开始/结束时间与执行进度。</span>
      </div>
      <div class="remote-hero-card__actions">
        <el-button :loading="loading" @click="loadTasks">刷新</el-button>
      </div>
    </el-card>

    <section class="remote-status-grid">
      <article v-for="item in taskSummary" :key="item.status" class="remote-status-card">
        <span>{{ item.label }}</span>
        <strong>{{ item.count }}</strong>
      </article>
    </section>

    <el-card class="remote-section-card" shadow="never">
      <template #header>
        <div class="remote-section-card__header">
          <strong>All Tasks</strong>
          <span>自动按 3 秒间隔刷新，离开页面后停止</span>
        </div>
      </template>

      <el-table :data="taskRows" border>
        <el-table-column label="任务名称" min-width="180">
          <template #default="{ row }">{{ getTaskName(row) }}</template>
        </el-table-column>
        <el-table-column label="状态" min-width="120">
          <template #default="{ row }">
            <el-tag :type="getTaskStatusType(row)" effect="light">{{ getTaskStatusLabel(row) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="开始时间" min-width="180">
          <template #default="{ row }">{{
            formatRemoteDateTime(resolveRemoteValue(row, ["StartTime", "startTime", "start_time"]))
          }}</template>
        </el-table-column>
        <el-table-column label="结束时间" min-width="180">
          <template #default="{ row }">{{
            formatRemoteDateTime(resolveRemoteValue(row, ["FinishTime", "finishTime", "finish_time"]))
          }}</template>
        </el-table-column>
        <el-table-column label="进度" min-width="240">
          <template #default="{ row }">
            <div v-if="getTaskStatus(row) === 'Failed'" class="remote-task-error">
              {{ formatRemoteCell(resolveRemoteValue(row, ["Error", "error"])) }}
            </div>
            <el-progress v-else :percentage="getTaskProgress(row)" :status="getTaskProgressStatus(row)" />
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import { ElMessage } from "element-plus";
import { defRecommendRemoteService } from "@/api/admin/recommend_remote";
import {
  formatRemoteCell,
  formatRemoteDateTime,
  parseRemoteRecordList,
  resolveRemoteNumber,
  resolveRemoteValue,
  type RemoteRecord
} from "../utils";

/** 任务统计卡片。 */
interface TaskSummaryItem {
  /** 状态值。 */
  status: string;
  /** 状态文案。 */
  label: string;
  /** 任务数量。 */
  count: number;
}

defineOptions({
  name: "RecommendRemoteTasks"
});

const loading = ref(false);
const taskRows = ref<RemoteRecord[]>([]);
let timer: ReturnType<typeof setInterval> | undefined;

/** 任务状态统计。 */
const taskSummary = computed<TaskSummaryItem[]>(() => {
  const statuses = [
    { status: "Running", label: "运行中" },
    { status: "Pending", label: "等待中" },
    { status: "Suspended", label: "已暂停" },
    { status: "Complete", label: "已完成" },
    { status: "Failed", label: "失败" }
  ];
  return statuses.map(item => ({
    ...item,
    count: taskRows.value.filter(row => getTaskStatus(row) === item.status).length
  }));
});

/** 加载远程推荐任务状态。 */
async function loadTasks() {
  loading.value = true;
  try {
    const data = await defRecommendRemoteService.GetRecommendRemoteTasks({});
    taskRows.value = parseRemoteRecordList(data.json, ["Tasks", "tasks", "Nodes", "nodes"]);
  } catch (error) {
    ElMessage.error("加载任务状态失败");
    throw error;
  } finally {
    loading.value = false;
  }
}

/** 读取任务名称。 */
function getTaskName(row: RemoteRecord) {
  return formatRemoteCell(resolveRemoteValue(row, ["Name", "name", "Task", "task"]));
}

/** 读取任务状态。 */
function getTaskStatus(row: RemoteRecord) {
  return String(resolveRemoteValue(row, ["Status", "status"]) ?? "Unknown");
}

/** 读取任务状态中文文案。 */
function getTaskStatusLabel(row: RemoteRecord) {
  const labelMap: Record<string, string> = {
    Pending: "等待中",
    Running: "运行中",
    Suspended: "已暂停",
    Complete: "已完成",
    Failed: "失败"
  };
  return labelMap[getTaskStatus(row)] ?? getTaskStatus(row);
}

/** 读取任务状态标签类型。 */
function getTaskStatusType(row: RemoteRecord) {
  const typeMap: Record<string, "primary" | "success" | "warning" | "danger" | "info"> = {
    Pending: "info",
    Running: "primary",
    Suspended: "warning",
    Complete: "success",
    Failed: "danger"
  };
  return typeMap[getTaskStatus(row)] ?? "info";
}

/** 计算任务进度百分比。 */
function getTaskProgress(row: RemoteRecord) {
  const total = resolveRemoteNumber(row, ["Total", "total"]);
  const count = resolveRemoteNumber(row, ["Count", "count"]);
  if (getTaskStatus(row) === "Complete") return 100;
  if (total <= 0) return 0;
  return Math.min(100, Math.round((count / total) * 100));
}

/** 读取任务进度条状态。 */
function getTaskProgressStatus(row: RemoteRecord) {
  if (getTaskStatus(row) === "Complete") return "success";
  if (getTaskStatus(row) === "Suspended") return "warning";
  return undefined;
}

/** 启动任务自动刷新。 */
function startTaskTimer() {
  timer = setInterval(() => {
    loadTasks();
  }, 3000);
}

/** 停止任务自动刷新。 */
function stopTaskTimer() {
  if (!timer) return;
  clearInterval(timer);
  timer = undefined;
}

onMounted(() => {
  loadTasks();
  startTaskTimer();
});

onBeforeUnmount(() => {
  stopTaskTimer();
});
</script>

<style scoped lang="scss">
.remote-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.remote-hero-card,
.remote-section-card,
.remote-status-card {
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

.remote-status-grid {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 16px;
}

.remote-status-card {
  padding: 16px;
  border: 1px solid var(--admin-page-card-border);
  border-radius: 14px;

  span {
    color: var(--admin-page-text-secondary);
    font-size: 13px;
  }

  strong {
    display: block;
    margin-top: 8px;
    color: var(--admin-page-text-primary);
    font-size: 26px;
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

.remote-task-error {
  color: var(--el-color-danger);
  word-break: break-word;
}

@media (max-width: 900px) {
  .remote-hero-card :deep(.el-card__body) {
    align-items: flex-start;
    flex-direction: column;
  }

  .remote-status-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
