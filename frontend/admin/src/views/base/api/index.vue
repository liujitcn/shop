<!-- API 管理 -->
<template>
  <div class="table-box">
    <ProTable ref="proTable" row-key="id" :columns="columns" :request-api="requestBaseApiTable" />

    <el-drawer v-model="detailDrawer.visible" title="API 详情" size="70%" @close="handleCloseDetail">
      <el-descriptions v-if="detailData" :column="1" border>
        <el-descriptions-item label="服务名">{{ detailData.service_name }}</el-descriptions-item>
        <el-descriptions-item label="服务描述">{{ detailData.service_desc }}</el-descriptions-item>
        <el-descriptions-item label="描述">{{ detailData.desc }}</el-descriptions-item>
        <el-descriptions-item label="操作方法">{{ detailData.operation }}</el-descriptions-item>
        <el-descriptions-item label="请求方法">{{ detailData.method }}</el-descriptions-item>
        <el-descriptions-item label="请求地址">{{ detailData.path }}</el-descriptions-item>
        <el-descriptions-item label="MCP工具名">{{ detailData.mcp_tool_name }}</el-descriptions-item>
        <el-descriptions-item label="MCP工具">{{ detailData.mcp_enabled ? "启用" : "禁用" }}</el-descriptions-item>
      </el-descriptions>

      <div v-if="detailDoc" class="api-doc">
        <section class="api-doc-section">
          <div class="api-doc-title">请求参数</div>
          <el-table
            v-if="detailDoc.parameters.length > 0"
            :data="detailDoc.parameters"
            row-key="path"
            default-expand-all
            :tree-props="{ children: 'children' }"
          >
            <el-table-column prop="path" label="字段" min-width="220" />
            <el-table-column prop="in" label="位置" width="90" />
            <el-table-column label="类型" min-width="180">
              <template #default="{ row }">{{ formatSchemaType(row) }}</template>
            </el-table-column>
            <el-table-column label="必填" width="80">
              <template #default="{ row }">{{ row.required ? "是" : "否" }}</template>
            </el-table-column>
            <el-table-column prop="description" label="说明" min-width="240" show-overflow-tooltip />
          </el-table>
          <el-empty v-else description="无请求参数" :image-size="72" />
        </section>

        <section class="api-doc-section">
          <div class="api-doc-title">请求体</div>
          <el-table
            v-if="requestBodyRows.length > 0"
            :data="requestBodyRows"
            row-key="path"
            default-expand-all
            :tree-props="{ children: 'children' }"
          >
            <el-table-column prop="path" label="字段" min-width="220" />
            <el-table-column label="类型" min-width="180">
              <template #default="{ row }">{{ formatSchemaType(row) }}</template>
            </el-table-column>
            <el-table-column label="必填" width="80">
              <template #default="{ row }">{{ row.required ? "是" : "否" }}</template>
            </el-table-column>
            <el-table-column prop="description" label="说明" min-width="240" show-overflow-tooltip />
          </el-table>
          <el-empty v-else description="无请求体" :image-size="72" />
        </section>

        <section class="api-doc-section">
          <div class="api-doc-title">返回值</div>
          <el-collapse v-if="detailDoc.responses.length > 0">
            <el-collapse-item v-for="response in detailDoc.responses" :key="response.status" :name="response.status">
              <template #title>
                <span class="api-doc-response-title">{{ response.status }} {{ response.description }}</span>
              </template>
              <el-table
                v-if="responseBodyRows(response).length > 0"
                :data="responseBodyRows(response)"
                row-key="path"
                default-expand-all
                :tree-props="{ children: 'children' }"
              >
                <el-table-column prop="path" label="字段" min-width="220" />
                <el-table-column label="类型" min-width="180">
                  <template #default="{ row }">{{ formatSchemaType(row) }}</template>
                </el-table-column>
                <el-table-column prop="description" label="说明" min-width="240" show-overflow-tooltip />
              </el-table>
              <el-empty v-else description="无响应体" :image-size="72" />
            </el-collapse-item>
          </el-collapse>
          <el-empty v-else description="无返回值" :image-size="72" />
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
import type { BaseApi, BaseApiDoc, BaseApiDocResponse, BaseApiDocSchema, PageBaseApisRequest } from "@/rpc/admin/v1/base_api";
import { buildPageRequest } from "@/utils/proTable";

defineOptions({
  name: "BaseApi",
  inheritAttrs: false
});

const { BUTTONS } = useAuthButtons();
const proTable = ref<ProTableInstance>();
const detailData = ref<BaseApi>();
const detailDoc = ref<BaseApiDoc>();

const detailDrawer = reactive({
  visible: false
});

const requestBodyRows = computed(() => schemaRows(detailDoc.value?.request_body));

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
  { prop: "mcp_tool_name", label: "MCP工具名", minWidth: 260, search: { el: "input" } },
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
 * 打开 API 详情抽屉。
 */
async function handleOpenDetail(apiId: number) {
  const [baseApi, baseApiDoc] = await Promise.all([
    defBaseApiService.GetBaseApi({ id: apiId }),
    defBaseApiService.GetBaseApiDoc({ id: apiId })
  ]);
  detailData.value = baseApi;
  detailDoc.value = baseApiDoc;
  detailDrawer.visible = true;
}

/**
 * 关闭详情抽屉并清空旧数据，避免下次打开时短暂展示旧详情。
 */
function handleCloseDetail() {
  detailDrawer.visible = false;
  detailData.value = undefined;
  detailDoc.value = undefined;
}

/**
 * 将可选 Schema 转成表格行。
 */
function schemaRows(schema?: BaseApiDocSchema) {
  return schema ? [schema] : [];
}

/**
 * 获取响应体表格行。
 */
function responseBodyRows(response: BaseApiDocResponse) {
  return schemaRows(response.body);
}

/**
 * 格式化 Schema 类型，补充格式、引用类型与枚举值。
 */
function formatSchemaType(schema: BaseApiDocSchema) {
  const values = [schema.type];
  if (schema.format) values.push(`<${schema.format}>`);
  if (schema.ref) values.push(schema.ref);
  if (schema.enum.length > 0) values.push(schema.enum.join(" | "));
  return values.filter(Boolean).join(" ");
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
</script>

<style scoped lang="scss">
.api-doc {
  margin-top: 18px;
}
.api-doc-section {
  margin-top: 20px;
}
.api-doc-title {
  margin-bottom: 10px;
  font-size: 15px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.api-doc-response-title {
  font-weight: 500;
  color: var(--el-text-color-primary);
}
</style>
