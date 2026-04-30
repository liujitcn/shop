<template>
  <div class="table-box gorse-task-page">
    <ProTable
      row-key="name"
      :columns="columns"
      :request-api="requestTaskTable"
      :request-error="handleRequestError"
      :pagination="false"
    >
      <template #name="{ row }">{{ resolveTaskName(row.name) }}</template>
      <template #status="{ row }">
        <el-tag :type="resolveStatusTag(row.status)" effect="light">{{ resolveStatusText(row.status) }}</el-tag>
      </template>
      <template #progress="{ row }">
        <el-progress :percentage="resolveProgress(row)" :stroke-width="8" />
      </template>
      <template #start_time="{ row }">{{ formatTimestamp(row.start_time) }}</template>
      <template #finish_time="{ row }">{{ formatTimestamp(row.finish_time) }}</template>
    </ProTable>
  </div>
</template>

<script setup lang="ts">
import dayjs from "dayjs";
import type { TagProps } from "element-plus";
import { ElMessage } from "element-plus";
import ProTable from "@/components/ProTable/index.vue";
import type { ColumnProps } from "@/components/ProTable/interface";
import { defRecommendGorseService } from "@/api/admin/recommend_gorse";
import type { Task } from "@/rpc/admin/v1/recommend_gorse";

/** 任务状态查询参数。 */
interface TaskTableParams {
  /** 任务名称关键字。 */
  name?: string;
  /** 任务状态。 */
  status?: string;
}

/** 任务状态展示配置。 */
interface TaskStatusMeta {
  /** 中文文案。 */
  label: string;
  /** Element Plus 标签类型。 */
  type: TagProps["type"];
}

const taskStatusMap: Record<string, TaskStatusMeta> = {
  Complete: { label: "已完成", type: "success" },
  Running: { label: "运行中", type: "primary" },
  Pending: { label: "等待中", type: "info" },
  Failed: { label: "失败", type: "danger" },
  Error: { label: "异常", type: "danger" }
};

const taskNameMap: Record<string, string> = {
  "Load Dataset": "加载数据集",
  "Generate user-to-user recommendation": "生成用户相似推荐",
  "Generate item-to-item recommendation": "生成商品相似推荐",
  "Train Collaborative Filtering Model": "训练协同过滤模型",
  "Train Click-Through Rate Prediction Model": "训练点击率预测模型",
  "Generate recommendation": "生成个性化推荐",
  "Collect Garbage in Cache": "清理推荐缓存",
  "Optimize Collaborative Filtering Model": "优化协同过滤模型",
  "Optimize Click-Through Rate Prediction Model": "优化点击率预测模型"
};

/** 任务状态表格列配置。 */
const columns: ColumnProps[] = [
  { prop: "name", label: "任务名称", minWidth: 260, showOverflowTooltip: true, search: { el: "input" } },
  { prop: "tracer", label: "执行节点", minWidth: 120 },
  {
    prop: "status",
    label: "状态",
    minWidth: 120,
    align: "center",
    search: { el: "select" },
    enum: Object.entries(taskStatusMap).map(([value, meta]) => ({ label: meta.label, value }))
  },
  { prop: "progress", label: "进度", minWidth: 220 },
  { prop: "count", label: "当前进度", minWidth: 110, align: "right" },
  { prop: "total", label: "总进度", minWidth: 110, align: "right" },
  { prop: "start_time", label: "开始时间", minWidth: 180 },
  { prop: "finish_time", label: "结束时间", minWidth: 180 },
  { prop: "error", label: "错误信息", minWidth: 220, showOverflowTooltip: true }
];

/** 请求 Gorse 推荐任务表格；任务页不做定时刷新，只响应用户手动进入、搜索或刷新。 */
async function requestTaskTable(params: TaskTableParams = {}) {
  const data = await defRecommendGorseService.ListTasks({});
  const response = data as unknown as Record<string, unknown>;
  const rawTasks = Array.isArray(response.Tasks)
    ? response.Tasks
    : Array.isArray(response.tasks)
      ? response.tasks
      : Array.isArray(response.list)
        ? response.list
        : [];
  const tasks = rawTasks
    .map(item => {
      const record = typeof item === "object" && item !== null && !Array.isArray(item) ? (item as Record<string, unknown>) : {};
      // 兼容当前 json_name 原始字段和旧版小写字段，避免后端未重启或缓存旧接口时页面空白。
      return {
        tracer: String(record.Tracer ?? record.tracer ?? ""),
        name: String(record.Name ?? record.name ?? ""),
        status: String(record.Status ?? record.status ?? ""),
        error: String(record.Error ?? record.error ?? ""),
        count: Number(record.Count ?? record.count ?? 0),
        total: Number(record.Total ?? record.total ?? 0),
        start_time: String(record.StartTime ?? record.start_time ?? record.startTime ?? ""),
        finish_time: String(record.FinishTime ?? record.finish_time ?? record.finishTime ?? "")
      };
    })
    .filter(item => item.name || item.tracer);

  const taskNameKeyword = String(params.name ?? "")
    .trim()
    .toLowerCase();
  const taskStatus = String(params.status ?? "").trim();
  const filteredTasks = tasks.filter(item => {
    // 表格搜索为前端本地过滤，避免 Gorse 原始任务接口不支持查询参数时刷新后失效。
    if (
      taskNameKeyword &&
      !resolveTaskName(item.name).toLowerCase().includes(taskNameKeyword) &&
      !item.name.toLowerCase().includes(taskNameKeyword)
    ) {
      return false;
    }
    if (taskStatus && item.status !== taskStatus) return false;
    return true;
  });

  return { data: filteredTasks };
}

/** 处理任务表格请求失败。 */
function handleRequestError() {
  ElMessage.error("加载 Gorse 推荐任务状态失败");
}

/** 解析任务进度百分比。 */
function resolveProgress(row: Task) {
  if (!row.total) return 0;
  return Math.min(100, Math.round((Number(row.count || 0) / Number(row.total)) * 100));
}

/** 解析任务状态中文文案。 */
function resolveStatusText(status: string) {
  return taskStatusMap[status]?.label || status || "--";
}

/** 解析任务名称中文文案，未覆盖的新任务保留原始名称方便排查。 */
function resolveTaskName(name: string) {
  return taskNameMap[name] || name || "--";
}

/** 解析任务状态标签类型。 */
function resolveStatusTag(status: string) {
  return taskStatusMap[status]?.type || "info";
}

/** 格式化Gorse 任务时间。 */
function formatTimestamp(value: string) {
  if (!value || value.startsWith("0001-01-01")) return "--";
  return dayjs(value).format("YYYY-MM-DD HH:mm:ss");
}
</script>
