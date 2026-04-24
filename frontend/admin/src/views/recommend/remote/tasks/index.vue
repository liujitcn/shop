<template>
  <div>
    <ProTable
      ref="proTable"
      row-key="__rowKey"
      :columns="columns"
      :request-api="requestTaskTable"
      :pagination="false"
      :tool-button="false"
    >
      <template #status="{ row }">
        <el-tag :type="getTaskStatusType(row)" effect="light">{{ getTaskStatusLabel(row) }}</el-tag>
      </template>
      <template #progress="{ row }">
        <el-text v-if="getTaskStatus(row) === 'Failed'" type="danger">
          {{ formatRemoteCell(resolveRemoteValue(row, ["Error", "error"])) }}
        </el-text>
        <el-progress v-else :percentage="getTaskProgress(row)" :status="getTaskProgressStatus(row)" />
      </template>
    </ProTable>
  </div>
</template>

<script setup lang="ts">
import type { ColumnProps, ProTableInstance } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
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

defineOptions({
  name: "RecommendRemoteTasks"
});

/** 任务表格行。 */
interface TaskRow extends RemoteRecord {
  /** 表格稳定行键。 */
  __rowKey: string;
  /** 中文任务名称。 */
  taskName: string;
  /** 中文任务状态。 */
  statusText: string;
  /** 格式化开始时间。 */
  startTimeText: string;
  /** 格式化结束时间。 */
  finishTimeText: string;
}

const proTable = ref<ProTableInstance>();

/** 任务表格列配置。 */
const columns: ColumnProps[] = [
  { prop: "taskName", label: "任务名称", minWidth: 220, align: "left" },
  { prop: "status", label: "状态", minWidth: 120 },
  { prop: "startTimeText", label: "开始时间", minWidth: 180 },
  { prop: "finishTimeText", label: "结束时间", minWidth: 180 },
  { prop: "progress", label: "进度", minWidth: 260 }
];

/** 查询远程推荐任务表格。 */
async function requestTaskTable() {
  try {
    const data = await defRecommendRemoteService.GetRecommendRemoteTasks({});
    return {
      data: parseRemoteRecordList(data.json, ["Tasks", "tasks", "Nodes", "nodes"]).map(normalizeTaskRow)
    };
  } catch (error) {
    ElMessage.error("加载任务状态失败");
    throw error;
  }
}

/** 将远程任务记录转换为 ProTable 行。 */
function normalizeTaskRow(row: RemoteRecord, index: number): TaskRow {
  return {
    ...row,
    __rowKey: `${getTaskName(row)}-${index}`,
    taskName: getTaskNameLabel(row),
    statusText: getTaskStatusLabel(row),
    startTimeText: formatRemoteDateTime(resolveRemoteValue(row, ["StartTime", "startTime", "start_time"])),
    finishTimeText: formatRemoteDateTime(resolveRemoteValue(row, ["FinishTime", "finishTime", "finish_time"]))
  };
}

/** 读取任务英文名称。 */
function getTaskName(row: RemoteRecord) {
  return String(resolveRemoteValue(row, ["Name", "name", "Task", "task"]) ?? "");
}

/** 读取任务中文名称。 */
function getTaskNameLabel(row: RemoteRecord) {
  const name = getTaskName(row);
  const labelMap: Record<string, string> = {
    "Load Dataset": "加载数据集",
    "Generate user-to-user recommendation": "生成相似用户推荐",
    "Generate item-to-item recommendation": "生成相似商品推荐",
    "Train Collaborative Filtering Model": "训练协同过滤模型",
    "Train Click-Through Rate Prediction Model": "训练点击率预测模型",
    "Generate recommendation": "生成推荐缓存",
    "Collect Garbage in Cache": "清理推荐缓存垃圾",
    "Optimize Collaborative Filtering Model": "优化协同过滤模型",
    "Optimize Click-Through Rate Prediction Model": "优化点击率预测模型"
  };
  return labelMap[name] ?? formatRemoteCell(name);
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
</script>
