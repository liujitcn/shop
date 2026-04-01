<template>
  <div class="table-box">
    <ProTable
      ref="proTable"
      row-key="id"
      :columns="columns"
      :header-actions="headerActions"
      :request-api="requestBaseRoleTable"
    />

    <FormDialog
      v-model="dialog.visible"
      ref="formDialogRef"
      :title="dialog.title"
      width="500px"
      :model="formData"
      :fields="formFields"
      :rules="rules"
      label-width="100px"
      @confirm="handleSubmit"
      @close="handleCloseDialog"
    />

    <el-drawer v-model="assignPermDialogVisible" :title="`【${checkedBaseRole.name}】权限分配`" size="500">
      <div class="perm-toolbar">
        <el-input v-model="permKeywords" clearable class="perm-search" placeholder="菜单权限名称">
          <template #prefix>
            <Search />
          </template>
        </el-input>

        <div class="perm-toolbar__actions">
          <div class="perm-toolbar__group">
            <span class="perm-toolbar__label">树操作</span>
            <el-button type="primary" size="small" plain class="perm-toolbar__button" @click="togglePermTree">
              <template #icon>
                <Switch />
              </template>
              {{ isExpanded ? "收缩节点" : "展开节点" }}
            </el-button>
          </div>
          <div class="perm-toolbar__group perm-toolbar__group--linkage">
            <span class="perm-toolbar__label">勾选模式</span>
            <el-checkbox v-model="parentChildLinked" @change="handelParentChildLinkedChange">父子联动</el-checkbox>
            <el-tooltip placement="bottom">
              <template #content>如果只需勾选菜单权限，不需要勾选子菜单或者按钮权限，请关闭父子联动</template>
              <el-icon class="perm-linkage__icon">
                <QuestionFilled />
              </el-icon>
            </el-tooltip>
          </div>
        </div>
      </div>

      <el-tree
        ref="permTreeRef"
        node-key="value"
        show-checkbox
        :data="menuPermOptions"
        :filter-node-method="handlePermFilter"
        :default-expand-all="true"
        :check-strictly="!parentChildLinked"
        class="mt-5"
      >
        <template #default="{ data }">
          {{ data.label }}
        </template>
      </el-tree>

      <template #footer>
        <div class="dialog-footer">
          <el-button type="primary" @click="handleAssignPermSubmit">确 定</el-button>
          <el-button @click="assignPermDialogVisible = false">取 消</el-button>
        </div>
      </template>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, reactive, ref, watch } from "vue";
import { ElMessage, ElMessageBox, ElTree } from "element-plus";
import type { CheckboxValueType } from "element-plus";
import { CirclePlus, Delete, EditPen, Position, QuestionFilled, Search, Switch } from "@element-plus/icons-vue";
import type { ColumnProps, HeaderActionProps, ProTableInstance } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
import FormDialog from "@/components/Dialog/FormDialog.vue";
import type { ProFormField, ProFormOption } from "@/components/ProForm/interface";
import { useAuthButtons } from "@/hooks/useAuthButtons";
import { defBaseRoleService } from "@/api/admin/base_role";
import type { BaseRole, BaseRoleForm, PageBaseRoleRequest } from "@/rpc/admin/base_role";
import { defBaseMenuService } from "@/api/admin/base_menu";
import type { TreeOptionResponse_Option } from "@/rpc/common/common";
import { Status } from "@/rpc/common/enum";
import { buildPageRequest, normalizeSelectedIds } from "@/utils/proTable";

defineOptions({
  name: "BaseRole",
  inheritAttrs: false
});

interface CheckedBaseRole {
  id?: number;
  name?: string;
}

const { BUTTONS } = useAuthButtons();
const proTable = ref<ProTableInstance>();
const formDialogRef = ref<InstanceType<typeof FormDialog>>();
const permTreeRef = ref<InstanceType<typeof ElTree>>();

const dialog = reactive({
  title: "",
  visible: false
});

const menuPermOptions = ref<TreeOptionResponse_Option[]>([]);
const statusOptions: ProFormOption[] = [
  { label: "启用", value: Status.ENABLE },
  { label: "禁用", value: Status.DISABLE }
];

const formData = reactive<BaseRoleForm>({
  /** 角色ID */
  id: 0,
  /** 角色名称 */
  name: "",
  /** 角色值 */
  code: "",
  /** 数据权限：0全部数据1部门及子部门数据2本部门数据3本人数据 */
  dataScope: 1,
  /** 分配的菜单列表 */
  menus: [],
  /** 状态 */
  status: Status.ENABLE,
  /** 备注 */
  remark: ""
});

const rules = reactive({
  name: [{ required: true, message: "请输入角色名称", trigger: "blur" }],
  code: [{ required: true, message: "请输入角色编码", trigger: "blur" }],
  dataScope: [{ required: true, message: "请选择数据权限", trigger: "change" }],
  menus: [{ required: true, message: "请选择菜单权限", trigger: "change" }],
  status: [{ required: true, message: "请选择状态", trigger: "change" }]
});

const checkedBaseRole = ref<CheckedBaseRole>({});
const assignPermDialogVisible = ref(false);
const permKeywords = ref("");
const isExpanded = ref(true);
const parentChildLinked = ref(true);

/** 角色表单字段配置。 */
const formFields = computed<ProFormField[]>(() => [
  { prop: "name", label: "角色名称", component: "input", props: { placeholder: "请输入角色名称" } },
  { prop: "code", label: "角色编码", component: "input", props: { placeholder: "请输入角色编码" } },
  { prop: "dataScope", label: "数据权限", component: "dict", props: { code: "base_role_data_scope" } },
  {
    prop: "menus",
    label: "菜单权限",
    component: "tree-select",
    options: menuPermOptions.value as unknown as ProFormOption[],
    props: {
      nodeKey: "value",
      props: { label: "label", children: "children" },
      multiple: true,
      showCheckbox: true,
      checkStrictly: true,
      style: { width: "100%" },
      onCheck: handleCheck
    }
  },
  { prop: "remark", label: "备注", component: "textarea", props: { placeholder: "请输入备注" } },
  { prop: "status", label: "状态", component: "radio-group", options: statusOptions }
]);

/** 角色表格列配置。 */
const columns: ColumnProps[] = [
  { type: "selection", width: 55 },
  { prop: "name", label: "角色名称", minWidth: 140, search: { el: "input" } },
  { prop: "code", label: "角色编码", minWidth: 160, search: { el: "input" } },
  { prop: "dataScope", label: "数据权限", minWidth: 120, dictCode: "base_role_data_scope", search: { el: "select" } },
  { prop: "remark", label: "备注", minWidth: 160 },
  {
    prop: "status",
    label: "状态",
    minWidth: 100,
    search: { el: "select" },
    cellType: "status",
    statusProps: {
      activeValue: Status.ENABLE,
      inactiveValue: Status.DISABLE,
      activeText: "启用",
      inactiveText: "禁用",
      disabled: () => !BUTTONS.value["base:role:status"],
      beforeChange: scope => handleBeforeSetStatus(scope.row as BaseRole)
    }
  },
  { prop: "createdAt", label: "创建时间", minWidth: 180 },
  { prop: "updatedAt", label: "更新时间", minWidth: 180 },
  {
    prop: "operation",
    label: "操作",
    width: 280,
    fixed: "right",
    cellType: "actions",
    actions: [
      {
        label: "分配权限",
        type: "primary",
        link: true,
        icon: Position,
        hidden: () => !BUTTONS.value["base:role:menus"],
        onClick: scope => handleOpenAssignPermDialog(scope.row as BaseRole)
      },
      {
        label: "编辑",
        type: "primary",
        link: true,
        icon: EditPen,
        hidden: () => !BUTTONS.value["base:role:update"],
        params: scope => ({ roleId: scope.row.id }),
        onClick: (scope, params) => handleOpenDialog((params?.roleId as number | undefined) ?? (scope.row as BaseRole).id)
      },
      {
        label: "删除",
        type: "danger",
        link: true,
        icon: Delete,
        hidden: () => !BUTTONS.value["base:role:delete"],
        onClick: scope => handleDelete(scope.row as BaseRole)
      }
    ]
  }
];

/** 角色顶部按钮配置。 */
const headerActions: HeaderActionProps[] = [
  {
    label: "新增",
    type: "success",
    icon: CirclePlus,
    hidden: () => !BUTTONS.value["base:role:create"],
    onClick: () => handleOpenDialog()
  },
  {
    label: "删除",
    type: "danger",
    icon: Delete,
    hidden: () => !BUTTONS.value["base:role:delete"],
    disabled: scope => !scope.selectedList.length,
    onClick: scope => handleDelete(scope.selectedList as BaseRole[])
  }
];

/**
 * 请求角色列表，并由 ProTable 统一维护分页与搜索参数。
 */
async function requestBaseRoleTable(params: PageBaseRoleRequest) {
  const data = await defBaseRoleService.PageBaseRole(buildPageRequest(params));
  return { data };
}

/**
 * 刷新角色表格。
 */
function refreshTable() {
  proTable.value?.getTableList();
}

/**
 * 加载菜单权限树数据。
 */
async function loadMenuPermOptions() {
  const optionBaseMenuRes = await defBaseMenuService.OptionBaseMenu({});
  menuPermOptions.value = optionBaseMenuRes.list ?? [];
}

/**
 * 打开角色弹窗。
 */
async function handleOpenDialog(roleId?: number) {
  resetForm();
  await loadMenuPermOptions();
  dialog.title = roleId ? "修改角色" : "新增角色";
  dialog.visible = true;
  if (!roleId) return;

  defBaseRoleService.GetBaseRole({ value: roleId }).then(data => {
    Object.assign(formData, data);
  });
}

/**
 * 同步树选择组件已勾选菜单到表单值。
 */
function handleCheck(currentNode: unknown, { checkedNodes }: { checkedNodes: Array<{ value: number }> }) {
  formData.menus = checkedNodes.map(node => node.value);
}

/**
 * 提交角色表单。
 */
function handleSubmit() {
  formDialogRef.value?.validate()?.then(valid => {
    if (!valid) return;

    const submitData = JSON.parse(JSON.stringify(formData)) as BaseRoleForm;
    const request = submitData.id ? defBaseRoleService.UpdateBaseRole(submitData) : defBaseRoleService.CreateBaseRole(submitData);
    request.then(() => {
      ElMessage.success(submitData.id ? "修改角色成功" : "新增角色成功");
      handleCloseDialog();
      refreshTable();
    });
  });
}

/**
 * 关闭角色弹窗并恢复默认表单值。
 */
function handleCloseDialog() {
  dialog.visible = false;
  resetForm();
}

/**
 * 重置角色表单，避免新增与编辑之间互相污染。
 */
function resetForm() {
  formDialogRef.value?.resetFields();
  formDialogRef.value?.clearValidate();
  formData.id = 0;
  formData.name = "";
  formData.code = "";
  formData.dataScope = 1;
  formData.menus = [];
  formData.status = Status.ENABLE;
  formData.remark = "";
}

/**
 * 在角色状态切换前先完成确认与接口调用，避免首屏渲染触发误操作。
 */
async function handleBeforeSetStatus(row: BaseRole) {
  const nextStatus = row.status === Status.ENABLE ? Status.DISABLE : Status.ENABLE;
  const text = nextStatus === Status.ENABLE ? "启用" : "禁用";
  const roleName = row.name || row.code || `ID:${row.id}`;
  try {
    await ElMessageBox.confirm(`是否确定${text}角色？\n角色名称：${roleName}`, "提示", {
      confirmButtonText: "确认",
      cancelButtonText: "取消",
      type: "warning"
    });
    await defBaseRoleService.SetBaseRoleStatus({ id: row.id, status: nextStatus });
    ElMessage.success(`${text}成功`);
    refreshTable();
    return true;
  } catch {
    return false;
  }
}

/**
 * 删除角色，兼容单条删除与批量删除。
 */
function handleDelete(selected?: number | string | Array<number | string> | BaseRole | BaseRole[]) {
  const roleList = Array.isArray(selected)
    ? (selected.filter(item => typeof item === "object") as BaseRole[])
    : selected && typeof selected === "object"
      ? [selected as BaseRole]
      : [];
  const roleIds = (
    roleList.length ? roleList.map(item => item.id) : normalizeSelectedIds(selected as number | string | Array<number | string>)
  ).join(",");
  if (!roleIds) {
    ElMessage.warning("请勾选删除项");
    return;
  }

  const confirmMessage = roleList.length
    ? roleList.length === 1
      ? `是否确定删除角色？\n角色名称：${roleList[0].name || roleList[0].code || `ID:${roleList[0].id}`}`
      : `确认删除已选中的 ${roleList.length} 个角色吗？`
    : "确认删除已选中的角色吗？";

  ElMessageBox.confirm(confirmMessage, "警告", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning"
  }).then(
    () => {
      defBaseRoleService.DeleteBaseRole({ value: roleIds }).then(() => {
        ElMessage.success("删除角色成功");
        refreshTable();
      });
    },
    () => {
      ElMessage.info("已取消删除角色");
    }
  );
}

/**
 * 打开分配菜单权限抽屉，并回显当前角色已拥有的菜单。
 */
async function handleOpenAssignPermDialog(row: BaseRole) {
  if (!row.id) return;

  assignPermDialogVisible.value = true;
  checkedBaseRole.value = { id: row.id, name: row.name };
  await loadMenuPermOptions();
  nextTick(() => {
    permTreeRef.value?.setCheckedKeys(row.menus, false);
  });
}

/**
 * 提交角色菜单权限分配。
 */
function handleAssignPermSubmit() {
  const roleId = checkedBaseRole.value.id;
  if (!roleId) return;

  const checkedNodes = (permTreeRef.value?.getCheckedNodes(false, true) as Array<{ value: number }> | undefined) ?? [];
  const checkedMenuIds = checkedNodes.map(node => Number(node.value));
  defBaseRoleService.SetBaseRoleMenus({ id: roleId, menus: checkedMenuIds }).then(() => {
    ElMessage.success("分配权限成功");
    assignPermDialogVisible.value = false;
    refreshTable();
  });
}

/**
 * 展开或收缩权限树。
 */
function togglePermTree() {
  isExpanded.value = !isExpanded.value;
  if (!permTreeRef.value) return;

  Object.values(permTreeRef.value.store.nodesMap).forEach((node: any) => {
    if (isExpanded.value) node.expand();
    else node.collapse();
  });
}

watch(permKeywords, val => {
  permTreeRef.value?.filter(val);
});

/**
 * 按关键字过滤菜单权限树节点。
 */
function handlePermFilter(value: string, data: Record<string, any>) {
  if (!value) return true;
  return data.label.includes(value);
}

/**
 * 切换父子联动配置。
 */
function handelParentChildLinkedChange(val: CheckboxValueType) {
  parentChildLinked.value = Boolean(val);
}
</script>

<style scoped>
.perm-toolbar {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 14px;
  background: linear-gradient(180deg, #f8fafc 0%, #f3f6fb 100%);
  border: 1px solid #e4eaf3;
  border-radius: 12px;
}

.perm-search {
  width: 100%;
}

.perm-toolbar__actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}

.perm-toolbar__group {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  min-height: 38px;
  padding: 6px 10px;
  background: rgba(255, 255, 255, 0.94);
  border: 1px solid #e4eaf3;
  border-radius: 10px;
}

.perm-toolbar__group--linkage {
  margin-left: auto;
}

.perm-toolbar__label {
  color: #6b7280;
  font-size: 12px;
  font-weight: 600;
  letter-spacing: 0.02em;
  white-space: nowrap;
}

.perm-toolbar__button {
  min-width: 98px;
  border-color: var(--el-color-primary-light-5);
  background: #fff;
}

.perm-toolbar__button:hover,
.perm-toolbar__button:focus-visible {
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
  border-color: var(--el-color-primary-light-5);
}

.perm-linkage__icon {
  color: var(--el-color-primary);
  cursor: pointer;
  font-size: 14px;
}

@media (max-width: 768px) {
  .perm-toolbar__actions {
    align-items: stretch;
    flex-direction: column;
  }

  .perm-toolbar__group,
  .perm-toolbar__group--linkage {
    justify-content: space-between;
    margin-left: 0;
    width: 100%;
  }
}
</style>
