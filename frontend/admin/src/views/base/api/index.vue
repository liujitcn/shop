<!-- API 管理 -->
<template>
  <div class="table-box">
    <ProTable ref="proTable" row-key="id" :columns="columns" :request-api="requestBaseApiTable" />

    <el-drawer v-model="detailDrawer.visible" title="API 详情" size="60%" @close="handleCloseDetail">
      <el-descriptions v-if="detailData" :column="1" border>
        <el-descriptions-item label="服务名">{{ detailData.service_name }}</el-descriptions-item>
        <el-descriptions-item label="服务描述">{{ detailData.service_desc }}</el-descriptions-item>
        <el-descriptions-item label="描述">{{ detailData.desc }}</el-descriptions-item>
        <el-descriptions-item label="操作方法">{{ detailData.operation }}</el-descriptions-item>
        <el-descriptions-item label="请求方法">{{ detailData.method }}</el-descriptions-item>
        <el-descriptions-item label="请求地址">{{ detailData.path }}</el-descriptions-item>
      </el-descriptions>

      <div class="schema-list">
        <section v-for="item in schemaItems" :key="item.prop" class="schema-item">
          <h4>{{ item.label }}</h4>
          <pre><code>{{ item.content }}</code></pre>
        </section>
      </div>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { View } from "@element-plus/icons-vue";
import type { ColumnProps, ProTableInstance } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
import { useAuthButtons } from "@/hooks/useAuthButtons";
import { defBaseApiService } from "@/api/admin/base_api";
import type { BaseApi, PageBaseApisRequest } from "@/rpc/admin/v1/base_api";
import { buildPageRequest } from "@/utils/proTable";

defineOptions({
  name: "BaseApi",
  inheritAttrs: false
});

/** JSON Schema 展示项。 */
interface SchemaItem {
  /** 字段名 */
  prop: keyof Pick<BaseApi, "input_schema" | "arg_mapping" | "output_schema">;
  /** 展示标题 */
  label: string;
  /** 格式化后的展示内容 */
  content: string;
}

const { BUTTONS } = useAuthButtons();
const proTable = ref<ProTableInstance>();
const detailData = ref<BaseApi>();

const detailDrawer = reactive({
  visible: false
});

const mcpEnabledOptions = [
  { label: "启用", value: true },
  { label: "禁用", value: false }
];

/** API 表格列配置。 */
const columns: ColumnProps[] = [
  { prop: "service_name", label: "服务名", minWidth: 180, search: { el: "input" } },
  { prop: "service_desc", label: "服务描述", minWidth: 180, search: { el: "input" } },
  { prop: "desc", label: "描述", minWidth: 180, search: { el: "input" } },
  { prop: "operation", label: "操作方法", minWidth: 260, search: { el: "input" } },
  { prop: "method", label: "请求方法", width: 110, search: { el: "input" } },
  { prop: "path", label: "请求地址", minWidth: 260, search: { el: "input" } },
  {
    prop: "mcp_enabled",
    label: "MCP工具",
    width: 120,
    enum: mcpEnabledOptions,
    search: { el: "select" },
    cellType: "status",
    statusProps: {
      activeValue: true,
      inactiveValue: false,
      activeText: "启用",
      inactiveText: "禁用",
      disabled: () => !BUTTONS.value["base:api:mcp-enabled"],
      beforeChange: scope => handleBeforeSetMcpEnabled(scope.row as BaseApi)
    }
  },
  {
    prop: "operation",
    label: "操作",
    width: 110,
    fixed: "right",
    cellType: "actions",
    actions: [
      {
        label: "详情",
        type: "primary",
        link: true,
        icon: View,
        hidden: () => !BUTTONS.value["base:api:info"],
        onClick: scope => handleOpenDetail((scope.row as BaseApi).id)
      }
    ]
  }
];

/** 详情抽屉中的 Schema 展示项。 */
const schemaItems = computed<SchemaItem[]>(() => [
  { prop: "input_schema", label: "入参 Schema", content: formatJSON(detailData.value?.input_schema) },
  { prop: "arg_mapping", label: "参数映射", content: formatJSON(detailData.value?.arg_mapping) },
  { prop: "output_schema", label: "出参 Schema", content: formatJSON(detailData.value?.output_schema) }
]);

/**
 * 请求 API 分页列表，并由 ProTable 统一维护分页与搜索参数。
 */
async function requestBaseApiTable(params: PageBaseApisRequest) {
  const data = await defBaseApiService.PageBaseApis(buildPageRequest(params));
  return { data: { ...data, list: data.base_apis ?? [], total: data.total } };
}

/**
 * 刷新 API 表格。
 */
function refreshTable() {
  proTable.value?.getTableList();
}

/**
 * 打开 API 详情抽屉，详情接口返回完整 JSON Schema 字段。
 */
async function handleOpenDetail(apiId: number) {
  detailData.value = await defBaseApiService.GetBaseApi({ id: apiId });
  detailDrawer.visible = true;
}

/**
 * 关闭详情抽屉并清空旧数据，避免下次打开时短暂展示旧详情。
 */
function handleCloseDetail() {
  detailDrawer.visible = false;
  detailData.value = undefined;
}

/**
 * MCP 启用状态切换前进行二次确认，并调用启用状态接口完成持久化。
 */
async function handleBeforeSetMcpEnabled(row: BaseApi) {
  const nextEnabled = !row.mcp_enabled;
  const text = nextEnabled ? "启用" : "禁用";
  const apiName = row.desc || row.operation || `ID:${row.id}`;
  try {
    await ElMessageBox.confirm(`是否确定${text}该 API 的 MCP 工具能力？\nAPI：${apiName}`, "提示", {
      confirmButtonText: "确认",
      cancelButtonText: "取消",
      type: "warning"
    });
    await defBaseApiService.SetBaseApiMcpEnabled({ id: row.id, mcp_enabled: nextEnabled });
    ElMessage.success(`${text}成功`);
    refreshTable();
    return true;
  } catch {
    return false;
  }
}

/**
 * 尽量格式化 JSON 字符串；非 JSON 内容保持原样，空值展示占位。
 */
function formatJSON(value?: string) {
  if (!value) return "--";
  try {
    return JSON.stringify(JSON.parse(value), null, 2);
  } catch {
    // 后端若返回非标准 JSON 字符串，也保留原文方便排查。
    return value;
  }
}
</script>

<style scoped lang="scss">
.schema-list {
  margin-top: 16px;
}

.schema-item {
  margin-bottom: 16px;

  h4 {
    margin: 0 0 8px;
    font-size: 14px;
  }

  pre {
    max-height: 320px;
    padding: 12px;
    overflow: auto;
    border-radius: 4px;
    background: var(--el-fill-color-light);
    color: var(--el-text-color-primary);
  }
}
</style>
