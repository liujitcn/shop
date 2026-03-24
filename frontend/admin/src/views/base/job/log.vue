<template>
  <div class="app-container">
    <div class="search-bar">
      <el-form ref="queryFormRef" :model="queryParams" :inline="true">
        <el-form-item label="状态" prop="status">
          <Dict v-model="queryParams.status" code="status" />
        </el-form-item>
        <el-form-item prop="requestTime" label="操作时间">
          <el-date-picker
            v-model="queryParams.executeTime"
            :editable="false"
            class="!w-[240px]"
            type="daterange"
            range-separator="~"
            start-placeholder="开始时间"
            end-placeholder="截止时间"
            value-format="YYYY-MM-DD"
          />
        </el-form-item>

        <el-form-item>
          <el-button type="primary" icon="search" @click="handleQuery">搜索</el-button>
          <el-button icon="refresh" @click="handleResetQuery">重置</el-button>
        </el-form-item>
      </el-form>
    </div>

    <el-card shadow="never">
      <el-table v-loading="loading" :data="pageData" highlight-current-row border>
        <el-table-column label="执行时间" prop="executeTime" align="center" />
        <el-table-column label="状态" prop="status">
          <template #default="scope">
            <DictLabel v-model="scope.row.status" code="base_job_log_status" />
          </template>
        </el-table-column>
        <el-table-column label="消耗时间(ms)" prop="processTime" align="right" />
        <el-table-column fixed="right" label="操作" width="100">
          <template #default="scope">
            <el-button
              v-hasPerm="['base:job-log:info']"
              type="primary"
              size="small"
              link
              icon="info"
              @click="handleOpenDialog(scope.row.id)"
            >
              详情
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <pagination
        v-if="total > 0"
        v-model:total="total"
        v-model:page="queryParams.pageNum"
        v-model:limit="queryParams.pageSize"
        @pagination="handleQuery"
      />
    </el-card>

    <el-dialog
      v-model="dialog.visible"
      :title="dialog.title"
      width="1200px"
      @close="handleCloseDialog"
    >
      <div class="detail-container">
        <!-- 基础信息 -->
        <el-descriptions title="基础信息" border :column="2">
          <el-descriptions-item label="状态">
            <DictLabel v-model="detail.status" code="base_job_log_status" />
          </el-descriptions-item>
          <el-descriptions-item label="耗时">{{ detail.processTime }}</el-descriptions-item>
          <el-descriptions-item label="操作时间">{{ detail.executeTime }}</el-descriptions-item>
        </el-descriptions>

        <!-- 请求信息 -->
        <el-descriptions title="请求信息" border :column="1" class="mt-4">
          <el-descriptions-item label="执行参数">
            <pre class="code-block">{{ formatJson(detail.input) }}</pre>
          </el-descriptions-item>
          <el-descriptions-item label="输出结果">
            <pre class="code-block">{{ formatJson(detail.output) }}</pre>
          </el-descriptions-item>
        </el-descriptions>
        <!-- 失败原因（条件显示） -->
        <el-alert
          v-if="detail.status === 2"
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
defineOptions({
  name: "BaseJobLog",
  inheritAttrs: false,
});

import { defBaseJobService } from "@/api/admin/base_job";
import { formatJson } from "@/utils/utils";
import { BaseJobLog, PageBaseJobLogRequest } from "@/rpc/admin/base_job";
const queryFormRef = ref();

const route = useRoute();
const jobId = ref(route.query.jobId as unknown as number);

const loading = ref(false);
const total = ref(0);

const queryParams = reactive<PageBaseJobLogRequest>({
  /** 任务id */
  jobId: jobId.value,
  /** 状态码 */
  status: undefined,
  /** 请求时间 */
  executeTime: ["", ""],
  /** 当前页码 */
  pageNum: 0,
  /** 每一页的行数 */
  pageSize: 10,
});

// 日志表格数据
const pageData = ref<BaseJobLog[]>();

const dialog = reactive({
  title: "",
  visible: false,
});

const detail = reactive<BaseJobLog>({
  /** 日志ID */
  id: 0,
  /** 任务ID */
  jobId: 0,
  /** 执行参数 */
  input: "",
  /** 输出结果 */
  output: "",
  /** 错误信息 */
  error: "",
  /** 状态：1、成功。2、失败。 */
  status: 0,
  /** 消耗时间/毫秒 */
  processTime: "",
  /** 执行时间 */
  executeTime: "",
});

/** 查询 */
function handleQuery() {
  loading.value = true;
  defBaseJobService
    .PageBaseJobLog(queryParams)
    .then((data) => {
      pageData.value = data.list;
      total.value = data.total;
    })
    .finally(() => {
      loading.value = false;
    });
}
/** 重置查询 */
function handleResetQuery() {
  queryFormRef.value.resetFields();
  queryParams.pageNum = 1;
  queryParams.executeTime = ["", ""];
  handleQuery();
}

// 打开系统日志弹窗
function handleOpenDialog(logId?: number) {
  dialog.visible = true;
  if (logId) {
    dialog.title = "定时任务日志详情";
    defBaseJobService
      .GetBaseJobLog({
        value: logId,
      })
      .then((data) => {
        Object.assign(detail, data);
      });
  }
}

// 关闭系统日志弹窗
function handleCloseDialog() {
  dialog.visible = false;
}

onMounted(() => {
  handleQuery();
});
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
