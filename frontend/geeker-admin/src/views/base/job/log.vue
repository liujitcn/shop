<template>
  <div class="table-box">
    <ProTable ref="proTable" row-key="id" :columns="columns" :request-api="requestBaseJobLogTable" />

    <el-dialog v-model="dialog.visible" :title="dialog.title" width="1200px" @close="handleCloseDialog">
      <div class="detail-container">
        <el-descriptions title="基础信息" border :column="2">
          <el-descriptions-item label="状态">
            <DictLabel v-model="detail.status" code="base_job_log_status" />
          </el-descriptions-item>
          <el-descriptions-item label="耗时">{{ detail.processTime }}</el-descriptions-item>
          <el-descriptions-item label="操作时间">{{ detail.executeTime }}</el-descriptions-item>
        </el-descriptions>

        <el-descriptions title="请求信息" border :column="1" class="mt-4">
          <el-descriptions-item label="执行参数">
            <pre class="code-block">{{ formatJson(detail.input) }}</pre>
          </el-descriptions-item>
          <el-descriptions-item label="输出结果">
            <pre class="code-block">{{ formatJson(detail.output) }}</pre>
          </el-descriptions-item>
        </el-descriptions>

        <el-alert
          v-if="detail.status === BaseJobLogStatus.FAIL"
          title="失败原因"
          type="error"
          :description="detail.error"
          class="mt-4"
          show-icon
        />
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, watch } from "vue";
import { useRoute } from "vue-router";
import { InfoFilled } from "@element-plus/icons-vue";
import type { ColumnProps, ProTableInstance } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
import { defBaseJobService } from "@/api/admin/base_job";
import { formatJson } from "@/utils/utils";
import type { BaseJobLog, PageBaseJobLogRequest } from "@/rpc/admin/base_job";
import { BaseJobLogStatus } from "@/rpc/common/enum";
import { buildPageRequest } from "@/utils/proTable";

defineOptions({
  name: "BaseJobLog",
  inheritAttrs: false
});

const route = useRoute();
const proTable = ref<ProTableInstance>();
const jobId = ref(Number(route.query.jobId ?? 0));

const dialog = reactive({
  title: "",
  visible: false
});

const detail = reactive<BaseJobLog>({
  /** 任务日志ID */
  id: 0,
  /** 任务ID */
  jobId: 0,
  /** 执行参数 */
  input: "",
  /** 输出结果 */
  output: "",
  /** 错误信息 */
  error: "",
  /** 状态 */
  status: BaseJobLogStatus.UNKNOWN_BJLS,
  /** 消耗时间 */
  processTime: "",
  /** 执行时间 */
  executeTime: ""
});

/** 定时任务日志表格列配置。 */
const columns: ColumnProps[] = [
  {
    prop: "status",
    label: "状态",
    width: 120,
    dictCode: "base_job_log_status",
    search: { el: "select" }
  },
  {
    prop: "executeTime",
    label: "执行时间",
    width: 180,
    search: {
      el: "date-picker",
      props: {
        type: "daterange",
        editable: false,
        class: "!w-[240px]",
        rangeSeparator: "~",
        startPlaceholder: "开始时间",
        endPlaceholder: "截止时间",
        valueFormat: "YYYY-MM-DD"
      }
    }
  },
  { prop: "processTime", label: "消耗时间(ms)", align: "right" },
  {
    prop: "detailAction",
    label: "操作",
    width: 100,
    fixed: "right",
    cellType: "actions",
    actions: [
      {
        label: "详情",
        type: "primary",
        link: true,
        icon: InfoFilled,
        onClick: scope => handleOpenDialog((scope.row as BaseJobLog).id)
      }
    ]
  }
];

watch(
  () => route.query.jobId,
  value => {
    jobId.value = Number(value ?? 0);
    proTable.value?.search();
  }
);

/**
 * 请求定时任务日志列表，并补充当前任务 ID。
 */
async function requestBaseJobLogTable(params: PageBaseJobLogRequest) {
  const data = await defBaseJobService.PageBaseJobLog(
    buildPageRequest({
      ...params,
      jobId: jobId.value,
      executeTime: params.executeTime ?? ["", ""]
    })
  );
  return { data };
}

/**
 * 打开定时任务日志详情弹窗。
 */
function handleOpenDialog(logId?: number) {
  dialog.visible = true;
  if (!logId) return;

  dialog.title = "定时任务日志详情";
  defBaseJobService.GetBaseJobLog({ value: logId }).then(data => {
    Object.assign(detail, data);
  });
}

/**
 * 关闭定时任务日志详情弹窗。
 */
function handleCloseDialog() {
  dialog.visible = false;
}
</script>

<style scoped>
.detail-container {
  padding: 20px;
  background: #fff;
  border-radius: 4px;
  max-height: 70vh;
  overflow-y: auto;
}

.mt-4 {
  margin-top: 16px;
}

.code-block {
  background: #f5f7fa;
  padding: 12px;
  border-radius: 4px;
  max-height: 200px;
  overflow: auto;
  margin: 0;
}
</style>
