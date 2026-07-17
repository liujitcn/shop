<!-- 代码生成表配置 -->
<template>
  <div class="table-box">
    <ProTable
      ref="proTable"
      class="code-gen-table"
      row-key="id"
      :columns="columns"
      :header-actions="headerActions"
      :request-api="requestCodeGenTable"
    />

    <FormDialog
      v-model="dialog.visible"
      ref="formDialogRef"
      :title="dialog.title"
      width="min(920px, calc(100vw - 32px))"
      top="4vh"
      :model="formData"
      :fields="formFields"
      :rules="codeGenTableRules"
      :confirm-loading="saving"
      label-width="116px"
      :gutter="16"
      :col-span="12"
      @confirm="handleSubmit"
      @close="handleCloseDialog"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, reactive, ref } from "vue";
import { useRouter } from "vue-router";
import { Connection, SetUp, View } from "@element-plus/icons-vue";
import type { ColumnProps, HeaderActionProps, ProTableInstance } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
import FormDialog from "@/components/Dialog/FormDialog.vue";
import type { ProFormField, ProFormOption } from "@/components/ProForm/interface";
import { useAuthButtons } from "@/hooks/useAuthButtons";
import { defBaseMenuService } from "@/api/admin/base_menu";
import { defCodeGenColumnService } from "@/api/admin/code_gen_column";
import { defCodeGenTableService } from "@/api/admin/code_gen_table";
import type { CodeGenDatabaseColumn } from "@/rpc/admin/v1/code_gen_column";
import type {
  CodeGenDatabaseTable,
  CodeGenTable,
  CodeGenTableForm,
  PageCodeGenTablesRequest
} from "@/rpc/admin/v1/code_gen_table";
import type { TreeOptionResponse_Option } from "@/rpc/common/v1/common";
import { buildPageRequest, normalizeSelectedIds } from "@/utils/proTable";
import {
  codeGenPageTypeOptions,
  codeGenStatusOptions,
  codeGenTableRules,
  createDefaultCodeGenLeftTreeConfig,
  createDefaultCodeGenTableForm
} from "../config";

defineOptions({
  name: "CodeGen",
  inheritAttrs: false
});

/** 代码生成表配置删除入参。 */
type CodeGenDeleteTarget = number | string | Array<number | string> | CodeGenTable | CodeGenTable[];

const { BUTTONS } = useAuthButtons();
const router = useRouter();
const proTable = ref<ProTableInstance>();
const formDialogRef = ref<InstanceType<typeof FormDialog>>();
const saving = ref(false);
const databaseTables = ref<CodeGenDatabaseTable[]>([]);
const databaseColumns = ref<CodeGenDatabaseColumn[]>([]);
const leftTreeDatabaseColumns = ref<CodeGenDatabaseColumn[]>([]);
const parentMenuOptions = ref<ProFormOption[]>([]);

const dialog = reactive({
  title: "",
  visible: false
});

const formData = reactive<CodeGenTableForm>(createDefaultCodeGenTableForm());

/** 当前业务表选择项。 */
const databaseTableOptions = computed<ProFormOption[]>(() =>
  databaseTables.value.map(item => ({
    label: item.comment ? `${item.name}（${item.comment}）` : item.name,
    value: item.name,
    disabled: item.disabled && item.name !== formData.name
  }))
);

/** 左树来源表选择项。 */
const leftTreeTableOptions = computed<ProFormOption[]>(() =>
  databaseTables.value.map(item => ({
    label: item.comment ? `${item.name}（${item.comment}）` : item.name,
    value: item.name
  }))
);

/** 当前业务表字段选择项。 */
const databaseColumnOptions = computed<ProFormOption[]>(() => createDatabaseColumnOptions(databaseColumns.value));

/** 左树来源表字段选择项。 */
const leftTreeColumnOptions = computed<ProFormOption[]>(() => createDatabaseColumnOptions(leftTreeDatabaseColumns.value));

/** 代码生成表配置表单字段。 */
const formFields = computed<ProFormField[]>(() => [
  {
    prop: "name",
    label: "业务表名",
    component: "select",
    options: databaseTableOptions.value,
    colSpan: 24,
    labelTooltip: "从当前数据库表中选择，已经配置的表不可重复选择。",
    props: {
      placeholder: "请选择数据库表",
      clearable: true,
      filterable: true,
      style: { width: "100%" },
      onChange: handleTableNameChange
    }
  },
  {
    prop: "comment",
    label: "业务表描述",
    component: "input",
    colSpan: 24,
    props: { placeholder: "选择业务表后自动带出，可修改" }
  },
  { prop: "business_name", label: "业务名", component: "input", props: { placeholder: "如 base_dept" } },
  { prop: "entity_name", label: "实体名", component: "input", props: { placeholder: "如 BaseDept" } },
  { prop: "module_path", label: "模块路径", component: "input", props: { placeholder: "如 base" } },
  { prop: "permission_prefix", label: "权限前缀", component: "input", props: { placeholder: "如 base:dept" } },
  {
    prop: "parent_menu_id",
    label: "父级菜单",
    component: "tree-select",
    options: parentMenuOptions.value,
    props: {
      placeholder: "请选择生成页面挂载菜单",
      clearable: true,
      filterable: true,
      emptyValues: [0, undefined, null, ""],
      valueOnClear: 0,
      checkStrictly: true,
      renderAfterExpand: false,
      style: { width: "100%" }
    }
  },
  {
    prop: "page_type",
    label: "页面类型",
    component: "segmented",
    options: codeGenPageTypeOptions,
    props: { onChange: handlePageTypeChange }
  },
  {
    prop: "parent_column",
    label: "父节点字段",
    component: "select",
    options: databaseColumnOptions.value,
    props: { placeholder: "请选择父节点字段", clearable: true, filterable: true, style: { width: "100%" } },
    visible: model => model.page_type === "tree"
  },
  {
    prop: "tree_label_column",
    label: "树显示字段",
    component: "select",
    options: databaseColumnOptions.value,
    props: { placeholder: "请选择树显示字段", clearable: true, filterable: true, style: { width: "100%" } },
    visible: model => model.page_type === "tree"
  },
  {
    prop: "left_tree_config.table_name",
    label: "左树数据表",
    component: "select",
    options: leftTreeTableOptions.value,
    props: {
      placeholder: "请选择左树数据表",
      clearable: true,
      filterable: true,
      style: { width: "100%" },
      onChange: handleLeftTreeTableNameChange
    },
    visible: model => model.page_type === "left_tree"
  },
  {
    prop: "left_tree_config.comment",
    label: "左树描述",
    component: "input",
    props: { placeholder: "选择左树数据表后自动带出，可修改" },
    visible: model => model.page_type === "left_tree"
  },
  {
    prop: "left_tree_config.filter_column",
    label: "筛选字段",
    labelTooltip: "选择当前业务表中用于关联左树节点值的字段。点击左树节点后，右侧列表将按该字段筛选。",
    component: "select",
    options: databaseColumnOptions.value,
    props: { placeholder: "请选择当前表筛选字段", clearable: true, filterable: true, style: { width: "100%" } },
    visible: model => model.page_type === "left_tree"
  },
  {
    prop: "left_tree_config.parent_column",
    label: "左树父字段",
    component: "select",
    options: leftTreeColumnOptions.value,
    props: { placeholder: "请选择左树父字段", clearable: true, filterable: true, style: { width: "100%" } },
    visible: model => model.page_type === "left_tree"
  },
  {
    prop: "left_tree_config.label_column",
    label: "左树显示字段",
    component: "select",
    options: leftTreeColumnOptions.value,
    props: { placeholder: "请选择左树显示字段", clearable: true, filterable: true, style: { width: "100%" } },
    visible: model => model.page_type === "left_tree"
  },
  {
    prop: "left_tree_config.value_column",
    label: "左树值字段",
    component: "select",
    options: leftTreeColumnOptions.value,
    props: { placeholder: "请选择左树值字段", clearable: true, filterable: true, style: { width: "100%" } },
    visible: model => model.page_type === "left_tree"
  },
  {
    prop: "gen_backend",
    label: "生成后端",
    component: "switch",
    // 三个生成开关始终从新行开始并排展示。
    rowBreakBefore: true,
    colSpan: 8,
    props: { activeText: "生成", inactiveText: "跳过" }
  },
  {
    prop: "gen_frontend",
    label: "生成前端",
    component: "switch",
    colSpan: 8,
    props: { activeText: "生成", inactiveText: "跳过" }
  },
  { prop: "gen_sql", label: "生成SQL", component: "switch", colSpan: 8, props: { activeText: "生成", inactiveText: "跳过" } },
  { prop: "status", label: "状态", component: "segmented", options: codeGenStatusOptions, colSpan: 24 },
  { prop: "remark", label: "备注", component: "textarea", colSpan: 24, props: { placeholder: "请输入备注", rows: 3 } }
]);

/** 代码生成表配置列表列。 */
const columns: ColumnProps[] = [
  { type: "selection", width: 55 },
  { prop: "name", label: "业务表名", minWidth: 160, search: { el: "input" } },
  { prop: "comment", label: "业务表描述", minWidth: 160, showOverflowTooltip: true },
  { prop: "business_name", label: "业务名", minWidth: 140, search: { el: "input" } },
  { prop: "entity_name", label: "实体名", minWidth: 140 },
  { prop: "module_path", label: "模块路径", minWidth: 140, search: { el: "input" } },
  { prop: "page_type", label: "页面类型", minWidth: 120, enum: codeGenPageTypeOptions, search: { el: "select" } },
  { prop: "status", label: "状态", width: 100, enum: codeGenStatusOptions, search: { el: "select" }, tag: true },
  { prop: "remark", label: "备注", minWidth: 180, showOverflowTooltip: true },
  { prop: "created_at", label: "创建时间", minWidth: 180 },
  {
    prop: "operation",
    label: "操作",
    width: 440,
    fixed: "right",
    cellType: "actions",
    actions: [
      {
        label: "字段配置",
        type: "success",
        link: true,
        icon: SetUp,
        hidden: () => !BUTTONS.value["tool:code-gen-table:column"],
        onClick: scope => handleOpenColumnConfig((scope.row as CodeGenTable).id)
      },
      {
        label: "Proto配置",
        type: "warning",
        link: true,
        icon: Connection,
        hidden: () => !BUTTONS.value["tool:code-gen-table:proto"],
        onClick: scope => handleOpenProtoConfig((scope.row as CodeGenTable).id)
      },
      {
        label: "预览",
        type: "primary",
        link: true,
        icon: View,
        hidden: () => !BUTTONS.value["tool:code-gen-table:preview"],
        onClick: scope => handleOpenPreview((scope.row as CodeGenTable).id)
      },
      {
        label: "编辑",
        type: "primary",
        link: true,
        icon: EditPen,
        hidden: () => !BUTTONS.value["tool:code-gen-table:update"],
        onClick: scope => handleOpenDialog((scope.row as CodeGenTable).id)
      },
      {
        label: "删除",
        type: "danger",
        link: true,
        icon: Delete,
        hidden: () => !BUTTONS.value["tool:code-gen-table:delete"],
        onClick: scope => handleDelete(scope.row as CodeGenTable)
      }
    ]
  }
];

/** 打开已经保存的代码生成页面预览。 */
async function handleOpenPreview(tableId: number) {
  await router.push(`/tool/code-gen/preview/${tableId}`);
}

/** 打开字段配置页面。 */
async function handleOpenColumnConfig(tableId: number) {
  await router.push(`/tool/code-gen/columns/${tableId}`);
}

/** 打开Proto接口配置页面。 */
async function handleOpenProtoConfig(tableId: number) {
  await router.push(`/tool/code-gen/proto/${tableId}`);
}

/** 代码生成表配置列表顶部操作。 */
const headerActions: HeaderActionProps[] = [
  {
    label: "新增",
    type: "success",
    icon: CirclePlus,
    hidden: () => !BUTTONS.value["tool:code-gen-table:create"],
    onClick: () => handleOpenDialog()
  },
  {
    label: "删除",
    type: "danger",
    icon: Delete,
    hidden: () => !BUTTONS.value["tool:code-gen-table:delete"],
    disabled: scope => !scope.selectedList.length,
    onClick: scope => handleDelete(scope.selectedList as CodeGenTable[])
  }
];

/** 请求代码生成表配置列表。 */
async function requestCodeGenTable(params: PageCodeGenTablesRequest) {
  const data = await defCodeGenTableService.PageCodeGenTables(buildPageRequest(params));
  return { data: { ...data, list: data.code_gen_tables ?? [] } };
}

/** 打开新增或编辑弹窗，并加载当前表单所需选项。 */
async function handleOpenDialog(tableId?: number) {
  resetForm();
  const [tableData, menuData] = await Promise.all([
    defCodeGenTableService.ListCodeGenDatabaseTables({}),
    defBaseMenuService.OptionBaseMenus({})
  ]);
  databaseTables.value = tableData.tables ?? [];
  parentMenuOptions.value = convertMenuOptions(menuData.list ?? []);
  if (tableId) {
    const detail = await defCodeGenTableService.GetCodeGenTable({ id: tableId });
    Object.assign(formData, detail);
    formData.left_tree_config ??= createDefaultCodeGenLeftTreeConfig();
    await Promise.all([loadDatabaseColumns(databaseColumns, formData.name), loadLeftTreeDatabaseColumns()]);
    dialog.title = "编辑代码生成表配置";
  } else {
    dialog.title = "新增代码生成表配置";
  }
  dialog.visible = true;
}

/** 选择业务表后同步数据库注释、默认命名、字段选项和树字段默认值。 */
async function handleTableNameChange(tableName: string) {
  const table = databaseTables.value.find(item => item.name === tableName);
  formData.comment = table?.comment ?? "";
  formData.business_name = table?.business_name ?? "";
  formData.entity_name = table?.entity_name ?? "";
  formData.module_path = table?.module_path ?? "";
  formData.permission_prefix = table?.permission_prefix ?? "";
  await loadDatabaseColumns(databaseColumns, tableName);
  resetUnavailableTableColumns();
  formData.parent_column = resolveDefaultColumn(databaseColumns.value, "parent_id");
  formData.tree_label_column = resolveDefaultColumn(databaseColumns.value, "name");
}

/** 页面类型变化时清理不再生效的页面字段。 */
function handlePageTypeChange(pageType: string) {
  if (pageType !== "tree") {
    formData.parent_column = "";
    formData.tree_label_column = "";
  }
  if (pageType !== "left_tree") {
    resetLeftTreeConfig();
  }
}

/** 左树来源表变化时覆盖描述、加载字段选项并设置约定默认字段。 */
async function handleLeftTreeTableNameChange(tableName: string) {
  const config = ensureLeftTreeConfig();
  const table = databaseTables.value.find(item => item.name === tableName);
  config.comment = table?.comment ?? "";
  await loadLeftTreeDatabaseColumns();
  resetUnavailableLeftTreeColumns();
  config.parent_column = resolveDefaultColumn(leftTreeDatabaseColumns.value, "parent_id");
  config.label_column = resolveDefaultColumn(leftTreeDatabaseColumns.value, "name");
  config.value_column = resolveDefaultColumn(leftTreeDatabaseColumns.value, "id");
}

/** 提交代码生成表配置。 */
async function handleSubmit() {
  const valid = await formDialogRef.value?.validate();
  if (!valid) return;
  saving.value = true;
  try {
    if (formData.id) {
      await defCodeGenTableService.UpdateCodeGenTable({ id: formData.id, code_gen_table: { ...formData } });
      ElMessage.success("编辑代码生成表配置成功");
    } else {
      await defCodeGenTableService.CreateCodeGenTable({ code_gen_table: { ...formData } });
      ElMessage.success("新增代码生成表配置成功");
    }
    handleCloseDialog();
    refreshTable();
  } finally {
    saving.value = false;
  }
}

/** 删除单项或批量代码生成表配置。 */
async function handleDelete(selected?: CodeGenDeleteTarget) {
  const tableList = Array.isArray(selected)
    ? (selected.filter(item => typeof item === "object") as CodeGenTable[])
    : selected && typeof selected === "object"
      ? [selected as CodeGenTable]
      : [];
  const tableIds = (
    tableList.length ? tableList.map(item => item.id) : normalizeSelectedIds(selected as number | string | Array<number | string>)
  ).join(",");
  if (!tableIds) {
    ElMessage.warning("请勾选删除项");
    return;
  }
  const confirmMessage =
    tableList.length === 1 ? `确认删除业务表：${tableList[0].name}？` : `确认删除已选中的 ${tableList.length} 条配置？`;
  try {
    await ElMessageBox.confirm(confirmMessage, "警告", {
      confirmButtonText: "确定",
      cancelButtonText: "取消",
      type: "warning"
    });
  } catch {
    ElMessage.info("已取消删除代码生成表配置");
    return;
  }
  await defCodeGenTableService.DeleteCodeGenTable({ ids: tableIds });
  ElMessage.success("删除代码生成表配置成功");
  refreshTable();
}

/** 查询数据库表字段选项。 */
async function loadDatabaseColumns(target: { value: CodeGenDatabaseColumn[] }, tableName: string) {
  if (!tableName) {
    target.value = [];
    return;
  }
  const data = await defCodeGenColumnService.ListCodeGenDatabaseColumns({ table_name: tableName });
  target.value = data.columns ?? [];
}

/** 查询左树来源表字段选项。 */
async function loadLeftTreeDatabaseColumns() {
  const config = ensureLeftTreeConfig();
  if (formData.page_type !== "left_tree") {
    leftTreeDatabaseColumns.value = [];
    return;
  }
  await loadDatabaseColumns(leftTreeDatabaseColumns, config.table_name);
}

/** 转换数据库字段为 ProForm 选择项。 */
function createDatabaseColumnOptions(columns: CodeGenDatabaseColumn[]): ProFormOption[] {
  return columns.map(item => ({
    label: item.column_comment
      ? `${item.column_name}（${item.column_comment} / ${item.column_type || item.db_type}）`
      : `${item.column_name}（${item.column_type || item.db_type}）`,
    value: item.column_name
  }));
}

/** 从字段列表中解析存在的约定默认字段。 */
function resolveDefaultColumn(columns: CodeGenDatabaseColumn[], columnName: string) {
  return columns.some(item => item.column_name === columnName) ? columnName : "";
}

/** 转换菜单树为 ProForm 树形选择项。 */
function convertMenuOptions(options: TreeOptionResponse_Option[]): ProFormOption[] {
  return options.map(item => ({
    label: item.label,
    value: item.value,
    disabled: item.disabled,
    children: convertMenuOptions(item.children ?? [])
  }));
}

/** 清理当前业务表已不存在的字段配置。 */
function resetUnavailableTableColumns() {
  const columnNames = new Set(databaseColumns.value.map(item => item.column_name));
  if (formData.parent_column && !columnNames.has(formData.parent_column)) formData.parent_column = "";
  if (formData.tree_label_column && !columnNames.has(formData.tree_label_column)) formData.tree_label_column = "";
  const config = ensureLeftTreeConfig();
  if (config.filter_column && !columnNames.has(config.filter_column)) {
    config.filter_column = "";
  }
}

/** 清理左树来源表已不存在的字段配置。 */
function resetUnavailableLeftTreeColumns() {
  const columnNames = new Set(leftTreeDatabaseColumns.value.map(item => item.column_name));
  const config = ensureLeftTreeConfig();
  if (config.parent_column && !columnNames.has(config.parent_column)) {
    config.parent_column = "";
  }
  if (config.label_column && !columnNames.has(config.label_column)) {
    config.label_column = "";
  }
  if (config.value_column && !columnNames.has(config.value_column)) {
    config.value_column = "";
  }
}

/** 清空左树右表专属配置。 */
function resetLeftTreeConfig() {
  formData.left_tree_config = createDefaultCodeGenLeftTreeConfig();
  leftTreeDatabaseColumns.value = [];
}

/** 确保左树配置对象存在并返回当前配置。 */
function ensureLeftTreeConfig() {
  formData.left_tree_config ??= createDefaultCodeGenLeftTreeConfig();
  return formData.left_tree_config;
}

/** 重置弹窗表单和字段选项。 */
function resetForm() {
  Object.assign(formData, createDefaultCodeGenTableForm());
  databaseColumns.value = [];
  leftTreeDatabaseColumns.value = [];
  void nextTick(() => {
    formDialogRef.value?.resetFields();
    formDialogRef.value?.clearValidate();
  });
}

/** 关闭弹窗并清理表单状态。 */
function handleCloseDialog() {
  dialog.visible = false;
  resetForm();
}

/** 刷新代码生成表配置列表。 */
function refreshTable() {
  proTable.value?.getTableList();
}

// 页面从缓存重新激活时刷新列表数据。
</script>

<style scoped lang="scss">
/* 固定操作列表头与普通表头使用同一主题背景，并保持行内操作单行展示。 */
:deep(.code-gen-table) {
  --el-table-header-bg-color: var(--el-fill-color-light);

  td.el-table-fixed-column--right .cell {
    white-space: nowrap;
  }
}
</style>
