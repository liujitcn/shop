<template>
  <div class="table-box">
    <ProTable
      ref="proTable"
      row-key="id"
      :indent="20"
      :columns="columns"
      :request-api="requestBaseDeptTable"
      :pagination="false"
      :default-expand-all="false"
      :tree-props="{ children: 'children', hasChildren: 'hasChildren' }"
    >
      <template #tableHeader="{ selectedList }">
        <el-button v-hasPerm="['base:dept:create']" type="success" :icon="CirclePlus" @click="handleOpenDialog(0)">
          新增
        </el-button>
        <el-button
          v-hasPerm="['base:dept:delete']"
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
          :disabled="!BUTTONS['base:dept:status']"
          :before-change="() => handleBeforeSetStatus(scope.row)"
        />
      </template>

      <template #operation="scope">
        <el-button
          v-hasPerm="['base:dept:create']"
          type="primary"
          link
          :icon="CirclePlus"
          @click.stop="handleOpenDialog(scope.row.id)"
        >
          新增
        </el-button>
        <el-button
          v-hasPerm="['base:dept:update']"
          type="primary"
          link
          :icon="EditPen"
          @click.stop="handleOpenDialog(scope.row.parentId, scope.row.id)"
        >
          编辑
        </el-button>
        <el-button v-hasPerm="['base:dept:delete']" type="danger" link :icon="Delete" @click.stop="handleDelete(scope.row)">
          删除
        </el-button>
      </template>
    </ProTable>

    <el-dialog v-model="dialog.visible" :title="dialog.title" width="600px" @closed="handleCloseDialog">
      <el-form ref="dataFormRef" :model="formData" :rules="rules" label-width="80px">
        <el-form-item label="上级部门" prop="parentId">
          <el-tree-select
            v-model="formData.parentId"
            placeholder="选择上级部门"
            :data="deptOptions"
            filterable
            check-strictly
            :render-after-expand="false"
          />
        </el-form-item>
        <el-form-item label="部门名称" prop="name">
          <el-input v-model="formData.name" placeholder="请输入部门名称" />
        </el-form-item>
        <el-form-item label="排序" prop="sort">
          <el-input-number v-model="formData.sort" controls-position="right" :min="1" :precision="0" :step="1" />
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
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { CirclePlus, Delete, EditPen } from "@element-plus/icons-vue";
import type { ColumnProps, ProTableInstance } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
import { useAuthButtons } from "@/hooks/useAuthButtons";
import { defBaseDeptService } from "@/api/admin/base_dept";
import type { BaseDept, BaseDeptForm } from "@/rpc/admin/base_dept";
import type { TreeOptionResponse_Option } from "@/rpc/common/common";
import { Status } from "@/rpc/common/enum";
import { normalizeSelectedIds } from "@/utils/proTable";

defineOptions({
  name: "BaseDept",
  inheritAttrs: false
});

const { BUTTONS } = useAuthButtons();
const proTable = ref<ProTableInstance>();
const dataFormRef = ref();

const dialog = reactive({
  title: "",
  visible: false
});

const deptOptions = ref<TreeOptionResponse_Option[]>([]);

const formData = reactive<BaseDeptForm>({
  /** 部门ID */
  id: 0,
  /** 父节点ID */
  parentId: undefined,
  /** 部门名称 */
  name: "",
  /** 排序 */
  sort: 1,
  /** 菜单状态 */
  status: Status.ENABLE,
  /** 备注 */
  remark: ""
});

const rules = reactive({
  parentId: [{ required: true, message: "上级部门不能为空", trigger: "change" }],
  name: [{ required: true, message: "部门名称不能为空", trigger: "blur" }],
  sort: [{ required: true, message: "排序不能为空", trigger: "blur" }],
  status: [{ required: true, message: "状态不能为空", trigger: "blur" }]
});

/** 部门树表格列配置。 */
const columns: ColumnProps[] = [
  { type: "selection", width: 55 },
  { prop: "name", label: "部门名称", align: "left", search: { el: "input" } },
  { prop: "remark", label: "备注", search: { el: "input" } },
  { prop: "sort", label: "排序", align: "right" },
  { prop: "status", label: "状态", width: 100, dictCode: "status", search: { el: "select" } },
  { prop: "createdAt", label: "创建时间", width: 180 },
  { prop: "updatedAt", label: "更新时间", width: 180 },
  { prop: "operation", label: "操作", width: 200, fixed: "right", align: "left" }
];

/**
 * 按搜索条件递归过滤部门树，命中父节点或子节点时保留当前节点。
 */
function filterDeptTree(deptList: BaseDept[], keywordMap: { name: string; remark: string; status: string }) {
  const nameKeyword = keywordMap.name.trim().toLowerCase();
  const remarkKeyword = keywordMap.remark.trim().toLowerCase();
  const statusKeyword = keywordMap.status.trim();

  return deptList.reduce<BaseDept[]>((result, item) => {
    const children = filterDeptTree(item.children ?? [], keywordMap);
    const name = item.name?.toLowerCase() ?? "";
    const remark = item.remark?.toLowerCase() ?? "";
    const status = String(item.status ?? "");
    const matched =
      (!nameKeyword || name.includes(nameKeyword)) &&
      (!remarkKeyword || remark.includes(remarkKeyword)) &&
      (!statusKeyword || status === statusKeyword);

    if (!matched && !children.length) return result;

    result.push({
      ...item,
      children
    });
    return result;
  }, []);
}

/**
 * 请求部门树数据，并按搜索条件过滤树形结构。
 */
async function requestBaseDeptTable(params: Record<string, string>) {
  const data = await defBaseDeptService.TreeBaseDept({});
  const keywordMap = {
    name: params.name ?? "",
    remark: params.remark ?? "",
    status: String(params.status ?? "")
  };
  return { data: filterDeptTree(data.list ?? [], keywordMap) };
}

/**
 * 刷新部门树表格。
 */
function refreshTable() {
  proTable.value?.getTableList();
}

/**
 * 加载部门下拉树数据，供弹窗选择上级部门。
 */
async function loadDeptOptions() {
  const optionBaseDeptResponse = await defBaseDeptService.OptionBaseDept({});
  deptOptions.value = [
    {
      value: 0,
      label: "顶级部门",
      disabled: false,
      children: optionBaseDeptResponse.list
    }
  ];
}

/**
 * 打开部门弹窗。
 */
async function handleOpenDialog(parentId?: number, deptId?: number) {
  await loadDeptOptions();
  dialog.visible = true;
  if (deptId) {
    dialog.title = "修改部门";
    defBaseDeptService.GetBaseDept({ value: deptId }).then(data => {
      Object.assign(formData, data);
    });
    return;
  }

  dialog.title = "新增部门";
  resetForm();
  formData.parentId = parentId;
}

/**
 * 提交部门表单。
 */
function handleSubmit() {
  dataFormRef.value?.validate((valid: boolean) => {
    if (!valid) return;

    const request = formData.id ? defBaseDeptService.UpdateBaseDept(formData) : defBaseDeptService.CreateBaseDept(formData);
    request.then(() => {
      ElMessage.success(formData.id ? "修改成功" : "新增成功");
      handleCloseDialog();
      refreshTable();
    });
  });
}

/**
 * 在部门状态切换前先完成确认与接口调用，避免首屏渲染触发误操作。
 */
async function handleBeforeSetStatus(row: BaseDept) {
  const nextStatus = row.status === Status.ENABLE ? Status.DISABLE : Status.ENABLE;
  const text = nextStatus === Status.ENABLE ? "启用" : "禁用";
  const deptName = row.name || `ID:${row.id}`;
  try {
    await ElMessageBox.confirm(`是否确定${text}部门：${deptName}？`, "提示", {
      confirmButtonText: "确认",
      cancelButtonText: "取消",
      type: "warning"
    });
    await defBaseDeptService.SetBaseDeptStatus({ id: row.id, status: nextStatus });
    ElMessage.success(`${text}成功`);
    refreshTable();
    return true;
  } catch {
    return false;
  }
}

/**
 * 删除部门，兼容单条删除与批量删除。
 */
function handleDelete(selected?: number | string | Array<number | string> | BaseDept | BaseDept[]) {
  const deptList = Array.isArray(selected)
    ? (selected.filter(item => typeof item === "object") as BaseDept[])
    : selected && typeof selected === "object"
      ? [selected as BaseDept]
      : [];
  const deptIds = (
    deptList.length ? deptList.map(item => item.id) : normalizeSelectedIds(selected as number | string | Array<number | string>)
  ).join(",");
  if (!deptIds) {
    ElMessage.warning("请勾选删除项");
    return;
  }

  const confirmMessage = deptList.length
    ? deptList.length === 1
      ? `是否确定删除部门：${deptList[0].name || `ID:${deptList[0].id}`}？`
      : `确认删除已选中的 ${deptList.length} 个部门吗？`
    : "确认删除已选中的部门吗？";

  ElMessageBox.confirm(confirmMessage, "警告", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning"
  }).then(
    () => {
      defBaseDeptService.DeleteBaseDept({ value: deptIds }).then(() => {
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
 * 重置部门表单，避免新增与编辑之间互相污染。
 */
function resetForm() {
  dataFormRef.value?.resetFields();
  dataFormRef.value?.clearValidate();
  formData.id = 0;
  formData.parentId = undefined;
  formData.name = "";
  formData.sort = 1;
  formData.status = Status.ENABLE;
  formData.remark = "";
}

/**
 * 关闭部门弹窗并恢复默认值。
 */
function handleCloseDialog() {
  dialog.visible = false;
  resetForm();
}
</script>
