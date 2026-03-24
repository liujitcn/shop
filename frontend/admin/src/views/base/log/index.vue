<template>
  <div class="app-container">
    <div class="search-bar">
      <el-form ref="queryFormRef" :model="queryParams" :inline="true">
        <el-form-item prop="operation" label="操作方法">
          <el-input
            v-model="queryParams.operation"
            placeholder="操作方法"
            clearable
            @keyup.enter="handleQuery"
          />
        </el-form-item>
        <el-form-item prop="statusCode" label="状态码">
          <el-input
            v-model="queryParams.statusCode"
            placeholder="状态码"
            clearable
            @keyup.enter="handleQuery"
          />
        </el-form-item>
        <el-form-item prop="requestTime" label="操作时间">
          <el-date-picker
            v-model="queryParams.requestTime"
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
        <el-table-column label="操作时间" prop="requestTime" align="center" />
        <el-table-column label="操作方法" prop="operation" />
        <el-table-column label="状态码" prop="statusCode" align="center" />
        <el-table-column label="操作人" prop="userName" align="center" />
        <el-table-column label="IP 地址" prop="clientIp" align="center" />
        <el-table-column label="地区" prop="location" />
        <el-table-column label="浏览器" prop="browserName" />
        <el-table-column label="终端系统" prop="osName" width="200" show-overflow-tooltip />
        <el-table-column label="执行时间(ms)" prop="costTime" align="right" />
        <el-table-column fixed="right" label="操作" width="100">
          <template #default="scope">
            <el-button
              v-hasPerm="['base:log:info']"
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
          <el-descriptions-item label="操作结果">
            <el-tag :type="detail.success ? 'success' : 'danger'">
              {{ detail.success ? "成功" : "失败" }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="状态码">
            <el-tag :type="statusCodeColor">{{ detail.statusCode }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="耗时">{{ detail.costTime }}</el-descriptions-item>
          <el-descriptions-item label="操作时间">{{ detail.requestTime }}</el-descriptions-item>
        </el-descriptions>

        <!-- 请求信息 -->
        <el-descriptions title="请求信息" border :column="1" class="mt-4">
          <el-descriptions-item label="请求ID">{{ detail.requestId }}</el-descriptions-item>
          <el-descriptions-item label="操作方法">
            <el-tag effect="plain">{{ detail.operation }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="请求方法">
            <el-tag effect="plain">{{ detail.method }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="请求路径">{{ detail.path }}</el-descriptions-item>
          <el-descriptions-item label="请求头">
            <pre class="code-block">{{ formatJson(detail.requestHeader) }}</pre>
          </el-descriptions-item>
          <el-descriptions-item label="请求体">
            <pre class="code-block">{{ formatJson(detail.requestBody) }}</pre>
          </el-descriptions-item>
          <el-descriptions-item label="请求结果">
            <pre class="code-block">{{ formatJson(detail.response) }}</pre>
          </el-descriptions-item>
        </el-descriptions>

        <!-- 用户信息 -->
        <el-descriptions title="用户信息" border :column="2" class="mt-4">
          <el-descriptions-item label="用户ID">{{ detail.userId }}</el-descriptions-item>
          <el-descriptions-item label="用户名">{{ detail.userName }}</el-descriptions-item>
          <el-descriptions-item label="客户端IP">{{ detail.clientIp }}</el-descriptions-item>
          <el-descriptions-item label="地理位置">{{ detail.location }}</el-descriptions-item>
        </el-descriptions>

        <!-- 客户端信息 -->
        <el-descriptions title="客户端信息" border :column="2" class="mt-4">
          <el-descriptions-item label="浏览器">
            {{ detail.browserName }} {{ detail.browserVersion }}
          </el-descriptions-item>
          <el-descriptions-item label="操作系统">
            {{ detail.osName }} {{ detail.osVersion }}
          </el-descriptions-item>
          <el-descriptions-item label="User Agent">{{ detail.userAgent }}</el-descriptions-item>
          <el-descriptions-item label="客户端名称">{{ detail.clientName }}</el-descriptions-item>
        </el-descriptions>

        <!-- 失败原因（条件显示） -->
        <el-alert
          v-if="!detail.success"
          title="失败原因"
          type="error"
          :description="detail.reason"
          class="mt-4"
          show-icon
        />
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
defineOptions({
  name: "BaseLog",
  inheritAttrs: false,
});

import { defBaseLogService } from "@/api/admin/base_log";
import { formatJson } from "@/utils/utils";
import { BaseLog, PageBaseLogRequest } from "@/rpc/admin/base_log";
const queryFormRef = ref();

const loading = ref(false);
const total = ref(0);

const queryParams = reactive<PageBaseLogRequest>({
  /** 操作方法 */
  operation: "",
  /** 状态码 */
  statusCode: undefined,
  /** 请求时间 */
  requestTime: ["", ""],
  /** 当前页码 */
  pageNum: 0,
  /** 每一页的行数 */
  pageSize: 10,
});

// 日志表格数据
const pageData = ref<BaseLog[]>();

const dialog = reactive({
  title: "",
  visible: false,
});

// 状态码颜色计算
const statusCodeColor = computed(() => {
  const code = detail.statusCode;
  if (code >= 200 && code < 300) return "success";
  if (code >= 400 && code < 500) return "warning";
  if (code >= 500) return "danger";
  return "info";
});

const detail = reactive<BaseLog>({
  /** 日志ID */
  id: 0,
  /** 请求ID */
  requestId: "",
  /** 请求ID */
  requestTime: "",
  /** 请求方法 */
  method: "",
  /** 操作方法 */
  operation: "",
  /** 请求路径 */
  path: "",
  /** 请求源 */
  referer: "",
  /** 请求URI */
  requestUri: "",
  /** 请求头 */
  requestHeader: "",
  /** 请求体 */
  requestBody: "",
  /** 响应信息 */
  response: "",
  /** 操作耗时 */
  costTime: "",
  /** 操作是否成功 */
  success: false,
  /** 状态码 */
  statusCode: 0,
  /** 操作失败原因 */
  reason: "",
  /** 操作地理位置 */
  location: "",
  /** 操作者用户ID */
  userId: 0,
  /** 操作者账号名 */
  userName: "",
  /** 操作者IP */
  clientIp: "",
  /** 浏览器的用户代理信息 */
  userAgent: "",
  /** 浏览器名称 */
  browserName: "",
  /** 浏览器版本 */
  browserVersion: "",
  /** 客户端ID */
  clientId: "",
  /** 客户端名称 */
  clientName: "",
  /** 操作系统名称 */
  osName: "",
  /** 操作系统版本 */
  osVersion: "",
});

/** 查询 */
function handleQuery() {
  loading.value = true;
  defBaseLogService
    .PageBaseLog(queryParams)
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
  queryParams.requestTime = ["", ""];
  handleQuery();
}

// 打开系统日志弹窗
function handleOpenDialog(logId?: number) {
  dialog.visible = true;
  if (logId) {
    dialog.title = "系统日志详情";
    defBaseLogService
      .GetBaseLog({
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
