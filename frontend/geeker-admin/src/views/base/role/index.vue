<template>
  <div class="table-box">
    <ProTable ref="proTable" row-key="id" :columns="columns" :request-api="requestBaseRoleTable">
      <template #tableHeader="{ selectedList }">
        <el-button v-hasPerm="'base:role:create'" type="success" :icon="CirclePlus" @click="handleOpenDialog()">新增</el-button>
        <el-button
          v-hasPerm="'base:role:delete'"
          type="danger"
          :icon="Delete"
          :disabled="!selectedList.length"
          @click="handleDelete(selectedList)"
        >
          删除
        </el-button>
      </template>

      <template #status="scope">
        <el-switch
          v-model="scope.row.status"
          inline-prompt
          :active-value="Status.ENABLE"
          :inactive-value="Status.DISABLE"
          active-text="启用"
          inactive-text="禁用"
          :disabled="!BUTTONS['base:role:status']"
          :before-change="() => handleBeforeSetStatus(scope.row)"
        />
      </template>

      <template #operation="scope">
        <el-button
          v-hasPerm="'base:role:menus'"
          type="primary"
          link
          :icon="Position"
          @click="handleOpenAssignPermDialog(scope.row)"
        >
          分配权限
        </el-button>
        <el-button v-hasPerm="'base:role:update'" type="primary" link :icon="EditPen" @click="handleOpenDialog(scope.row.id)">
          编辑
        </el-button>
        <el-button v-hasPerm="'base:role:delete'" type="danger" link :icon="Delete" @click="handleDelete(scope.row)">
          删除
        </el-button>
      </template>
    </ProTable>

    <el-dialog v-model="dialog.visible" :title="dialog.title" width="500px" @close="handleCloseDialog">
      <el-form ref="dataFormRef" :model="formData" :rules="rules" label-width="100px">
        <el-form-item label="角色名称" prop="name">
          <el-input v-model="formData.name" placeholder="请输入角色名称" />
        </el-form-item>

        <el-form-item label="角色编码" prop="code">
          <el-input v-model="formData.code" placeholder="请输入角色编码" />
        </el-form-item>

        <el-form-item label="数据权限" prop="dataScope">
          <Dict v-model="formData.dataScope" code="base_role_data_scope" />
        </el-form-item>
        <el-form-item label="菜单权限" prop="menus">
          <el-tree-select
            v-model="formData.menus"
            node-key="value"
            :data="menuPermOptions"
            multiple
            show-checkbox
            @check="handleCheck"
          />
        </el-form-item>

        <el-form-item label="备注" prop="remark">
          <el-input v-model="formData.remark" placeholder="请输入备注" type="textarea" />
        </el-form-item>
        <el-form-item label="状态" prop="status">
          <el-switch
            v-model="formData.status"
            inline-prompt
            active-text="启用"
            inactive-text="禁用"
            :active-value="Status.ENABLE"
            :inactive-value="Status.DISABLE"
          />
        </el-form-item>
      </el-form>

      <template #footer>
        <div class="dialog-footer">
          <el-button type="primary" @click="handleSubmit">确 定</el-button>
          <el-button @click="handleCloseDialog">取 消</el-button>
        </div>
      </template>
    </el-dialog>

    <el-drawer v-model="assignPermDialogVisible" :title="`【${checkedBaseRole.name}】权限分配`" size="500">
      <div class="flex-x-between">
        <el-input v-model="permKeywords" clearable class="w-[150px]" placeholder="菜单权限名称">
          <template #prefix>
            <Search />
          </template>
        </el-input>

        <div class="flex-center ml-5">
          <el-button type="primary" size="small" plain @click="togglePermTree">
            <template #icon>
              <Switch />
            </template>
            {{ isExpanded ? "收缩" : "展开" }}
          </el-button>
          <el-checkbox v-model="parentChildLinked" class="ml-5" @change="handelParentChildLinkedChange">父子联动</el-checkbox>

          <el-tooltip placement="bottom">
            <template #content>如果只需勾选菜单权限，不需要勾选子菜单或者按钮权限，请关闭父子联动</template>
            <el-icon class="ml-1 color-[--el-color-primary] inline-block cursor-pointer">
              <QuestionFilled />
            </el-icon>
          </el-tooltip>
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
import { nextTick, reactive, ref, watch } from "vue";
import { ElMessage, ElMessageBox, ElTree } from "element-plus";
import type { CheckboxValueType } from "element-plus";
import { CirclePlus, Delete, EditPen, Position, QuestionFilled, Search, Switch } from "@element-plus/icons-vue";
import type { ColumnProps, ProTableInstance } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
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
const dataFormRef = ref();
const permTreeRef = ref<InstanceType<typeof ElTree>>();

const dialog = reactive({
  title: "",
  visible: false
});

const menuPermOptions = ref<TreeOptionResponse_Option[]>([]);

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
  dataScope: [{ required: true, message: "请选择数据权限", trigger: "blur" }],
  menus: [{ required: true, message: "请选择菜单权限", trigger: "blur" }],
  status: [{ required: true, message: "请选择状态", trigger: "blur" }]
});

const checkedBaseRole = ref<CheckedBaseRole>({});
const assignPermDialogVisible = ref(false);
const permKeywords = ref("");
const isExpanded = ref(true);
const parentChildLinked = ref(true);

/** 角色表格列配置。 */
const columns: ColumnProps[] = [
  { type: "selection", width: 55 },
  { prop: "name", label: "角色名称", search: { el: "input" } },
  { prop: "code", label: "角色编码", search: { el: "input" } },
  { prop: "dataScope", label: "数据权限", dictCode: "base_role_data_scope", search: { el: "select" } },
  { prop: "remark", label: "备注" },
  { prop: "status", label: "状态", width: 100, dictCode: "status", search: { el: "select" } },
  { prop: "createdAt", label: "创建时间", width: 180 },
  { prop: "updatedAt", label: "更新时间", width: 180 },
  { prop: "operation", label: "操作", width: 220, fixed: "right" }
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
  dialog.visible = true;
  await loadMenuPermOptions();
  if (roleId) {
    dialog.title = "修改角色";
    defBaseRoleService.GetBaseRole({ value: roleId }).then(data => {
      Object.assign(formData, data);
    });
    return;
  }

  dialog.title = "新增角色";
  resetForm();
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
  dataFormRef.value?.validate((valid: boolean) => {
    if (!valid) return;

    const request = formData.id ? defBaseRoleService.UpdateBaseRole(formData) : defBaseRoleService.CreateBaseRole(formData);
    request.then(() => {
      ElMessage.success(formData.id ? "修改成功" : "新增成功");
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
  dataFormRef.value?.resetFields();
  dataFormRef.value?.clearValidate();
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
    await ElMessageBox.confirm(`是否确定${text}角色：${roleName}？`, "提示", {
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
      ? `是否确定删除角色：${roleList[0].name || roleList[0].code || `ID:${roleList[0].id}`}？`
      : `确认删除已选中的 ${roleList.length} 个角色吗？`
    : "确认删除已选中的角色吗？";

  ElMessageBox.confirm(confirmMessage, "警告", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning"
  }).then(
    () => {
      defBaseRoleService.DeleteBaseRole({ value: roleIds }).then(() => {
        ElMessage.success("删除成功");
        refreshTable();
      });
    },
    () => {
      ElMessage.info("已取消删除");
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
