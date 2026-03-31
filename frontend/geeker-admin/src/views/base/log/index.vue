<template>
  <div class="table-box">
    <ProTable ref="proTable" row-key="id" :columns="columns" :request-api="requestBaseLogTable" />

    <ProDialog v-model="dialog.visible" :title="dialog.title" width="1500px" @close="handleCloseDialog">
      <div class="detail-container">
        <el-descriptions title="基础信息" border :column="2">
          <el-descriptions-item label="操作结果">
            <el-tag :type="detail.success ? 'success' : 'danger'" effect="light">
              {{ detail.success ? "成功" : "失败" }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="状态码">
            <el-tag :type="statusCodeColor" effect="light">{{ detail.statusCode || "--" }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="耗时">{{ detail.costTime || "--" }}</el-descriptions-item>
          <el-descriptions-item label="操作时间">{{ detail.requestTime || "--" }}</el-descriptions-item>
        </el-descriptions>

        <el-descriptions title="请求信息" border :column="2" direction="vertical" class="mt-4 compact-descriptions">
          <el-descriptions-item label="请求ID">{{ detail.requestId || "--" }}</el-descriptions-item>
          <el-descriptions-item label="操作方法">
            <el-tag effect="plain">{{ detail.operation || "--" }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="请求方法">
            <el-tag effect="plain">{{ detail.method || "--" }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="请求路径">{{ detail.path || "--" }}</el-descriptions-item>
          <el-descriptions-item v-if="detail.requestUri" label="请求 URI" :span="2">{{ detail.requestUri }}</el-descriptions-item>
          <el-descriptions-item v-if="detail.referer" label="来源页" :span="2">{{ detail.referer }}</el-descriptions-item>
          <el-descriptions-item label="请求头" :span="2">
            <pre class="code-block">{{ formatPayload(detail.requestHeader) }}</pre>
          </el-descriptions-item>
          <el-descriptions-item label="请求体" :span="2">
            <pre class="code-block">{{ formatPayload(detail.requestBody) }}</pre>
          </el-descriptions-item>
          <el-descriptions-item label="请求结果" :span="2">
            <pre class="code-block">{{ formatPayload(detail.response) }}</pre>
          </el-descriptions-item>
        </el-descriptions>

        <el-descriptions title="用户信息" border :column="2" class="mt-4">
          <el-descriptions-item label="用户ID">{{ detail.userId || "--" }}</el-descriptions-item>
          <el-descriptions-item label="用户名">{{ detail.userName || "--" }}</el-descriptions-item>
          <el-descriptions-item label="客户端IP">{{ detail.clientIp || "--" }}</el-descriptions-item>
          <el-descriptions-item label="地理位置">{{ detail.location || "--" }}</el-descriptions-item>
        </el-descriptions>

        <el-descriptions title="客户端信息" border :column="2" direction="vertical" class="mt-4 compact-descriptions">
          <el-descriptions-item label="浏览器">
            {{ [detail.browserName, detail.browserVersion].filter(Boolean).join(" ") || "--" }}
          </el-descriptions-item>
          <el-descriptions-item label="操作系统">
            {{ [detail.osName, detail.osVersion].filter(Boolean).join(" ") || "--" }}
          </el-descriptions-item>
          <el-descriptions-item label="客户端名称">{{ detail.clientName || "--" }}</el-descriptions-item>
          <el-descriptions-item label="客户端ID">{{ detail.clientId || "--" }}</el-descriptions-item>
          <el-descriptions-item label="User Agent" :span="2">{{ detail.userAgent || "--" }}</el-descriptions-item>
        </el-descriptions>

        <el-alert v-if="!detail.success" title="失败原因" type="error" :description="detail.reason" class="mt-4" show-icon />
      </div>

      <template #footer>
        <el-button @click="handleCloseDialog">关闭</el-button>
      </template>
    </ProDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from "vue";
import { InfoFilled } from "@element-plus/icons-vue";
import type { ColumnProps, ProTableInstance } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
import ProDialog from "@/components/Dialog/ProDialog.vue";
import { defBaseLogService } from "@/api/admin/base_log";
import { formatJson } from "@/utils/utils";
import type { BaseLog, PageBaseLogRequest } from "@/rpc/admin/base_log";
import { buildPageRequest } from "@/utils/proTable";

defineOptions({
  name: "BaseLog",
  inheritAttrs: false
});

const proTable = ref<ProTableInstance>();

const dialog = reactive({
  title: "",
  visible: false
});

/** 状态码颜色计算。 */
const statusCodeColor = computed(() => {
  const code = detail.statusCode;
  if (code >= 200 && code < 300) return "success";
  if (code >= 400 && code < 500) return "warning";
  if (code >= 500) return "danger";
  return "info";
});

/** 创建默认日志详情，避免多次查看时残留上一条数据。 */
function createDefaultDetail(): BaseLog {
  return {
    /** 日志ID */
    id: 0,
    /** 请求ID */
    requestId: "",
    /** 请求时间 */
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
    osVersion: ""
  };
}

const detail = reactive<BaseLog>(createDefaultDetail());

function normalizeBoolean(value: unknown): boolean {
  if (typeof value === "boolean") return value;
  if (typeof value === "number") return value !== 0;
  if (typeof value === "string") {
    const normalized = value.trim().toLowerCase();
    return normalized === "true" || normalized === "1" || normalized === "success";
  }
  return false;
}

function normalizeNumber(value: unknown): number {
  if (typeof value === "number") return Number.isFinite(value) ? value : 0;
  if (typeof value === "string") {
    const parsed = Number(value);
    return Number.isFinite(parsed) ? parsed : 0;
  }
  return 0;
}

function normalizeDetail(data: BaseLog): BaseLog {
  return {
    ...data,
    success: normalizeBoolean((data as BaseLog & { success?: unknown }).success),
    statusCode: normalizeNumber((data as BaseLog & { statusCode?: unknown }).statusCode)
  };
}

function formatPayload(value: string): string {
  return value ? formatJson(value) : "--";
}

/** 日志表格列配置。 */
const columns: ColumnProps[] = [
  {
    prop: "operation",
    label: "操作方法",
    search: { el: "input" }
  },
  {
    prop: "statusCode",
    label: "状态码",
    align: "center",
    search: { el: "input-number", props: { min: 0, controlsPosition: "right" } }
  },
  {
    prop: "requestTime",
    label: "操作时间",
    align: "center",
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
  { prop: "userName", label: "操作人", align: "center" },
  { prop: "clientIp", label: "IP 地址", align: "center" },
  { prop: "location", label: "地区" },
  { prop: "browserName", label: "浏览器" },
  { prop: "osName", label: "终端系统", width: 200 },
  { prop: "costTime", label: "执行时间(ms)", align: "right" },
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
        onClick: scope => handleOpenDialog((scope.row as BaseLog).id)
      }
    ]
  }
];

/**
 * 请求系统日志列表，并由 ProTable 统一处理分页与搜索参数。
 */
async function requestBaseLogTable(params: PageBaseLogRequest) {
  const requestParams = buildPageRequest({
    ...params,
    requestTime: params.requestTime ?? ["", ""]
  });
  const data = await defBaseLogService.PageBaseLog(requestParams);
  return { data };
}

/**
 * 打开系统日志详情弹窗。
 */
function handleOpenDialog(logId?: number) {
  resetDetail();
  dialog.title = "系统日志详情";
  dialog.visible = true;
  if (!logId) return;

  defBaseLogService.GetBaseLog({ value: logId }).then(data => {
    Object.assign(detail, normalizeDetail(data));
  });
}

/**
 * 重置日志详情，避免关闭后保留旧数据。
 */
function resetDetail() {
  Object.assign(detail, createDefaultDetail());
}

/**
 * 关闭系统日志弹窗。
 */
function handleCloseDialog() {
  dialog.visible = false;
  resetDetail();
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
  max-height: 240px;
  overflow: auto;
  margin: 0;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-all;
}

:deep(.compact-descriptions .el-descriptions__label) {
  font-weight: 600;
}

:deep(.compact-descriptions .el-descriptions__content) {
  vertical-align: top;
}
</style>
