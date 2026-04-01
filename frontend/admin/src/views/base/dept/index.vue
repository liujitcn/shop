<template>
  <div class="table-box">
    <ProTable
      ref="proTable"
      row-key="id"
      :indent="20"
      :columns="columns"
      :header-actions="headerActions"
      :request-api="requestBaseDeptTable"
      :pagination="false"
      :default-expand-all="false"
      :tree-props="{ children: 'children', hasChildren: 'hasChildren' }"
    />

    <FormDialog
      v-model="dialog.visible"
      ref="formDialogRef"
      :title="dialog.title"
      width="600px"
      :model="formData"
      :fields="formFields"
      :rules="rules"
      label-width="90px"
      @confirm="handleSubmit"
      @close="handleCloseDialog"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { CirclePlus, Delete, EditPen } from "@element-plus/icons-vue";
import type { ColumnProps, HeaderActionProps, ProTableInstance } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
import FormDialog from "@/components/Dialog/FormDialog.vue";
import type { ProFormField, ProFormOption } from "@/components/ProForm/interface";
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
const formDialogRef = ref<InstanceType<typeof FormDialog>>();

const dialog = reactive({
  title: "",
  visible: false
});

const deptOptions = ref<TreeOptionResponse_Option[]>([]);
const statusOptions: ProFormOption[] = [
  { label: "启用", value: Status.ENABLE },
  { label: "禁用", value: Status.DISABLE }
];

const formData = reactive<BaseDeptForm>({
  /** 部门ID */
  id: 0,
  /** 父节点ID */
  parentId: 0,
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
  status: [{ required: true, message: "状态不能为空", trigger: "change" }]
});

/** 部门表单字段配置。 */
const formFields = computed<ProFormField[]>(() => [
  {
    prop: "parentId",
    label: "上级部门",
    component: "tree-select",
    options: deptOptions.value,
    props: {
      placeholder: "选择上级部门",
      filterable: true,
      checkStrictly: true,
      renderAfterExpand: false,
      style: { width: "100%" }
    }
  },
  { prop: "name", label: "部门名称", component: "input", props: { placeholder: "请输入部门名称" } },
  {
    prop: "sort",
    label: "排序",
    component: "input-number",
    props: { min: 1, precision: 0, step: 1, controlsPosition: "right", style: { width: "100%" } }
  },
  { prop: "remark", label: "备注", component: "textarea", props: { placeholder: "请输入备注" } },
  { prop: "status", label: "状态", component: "radio-group", options: statusOptions }
]);

/** 部门树表格列配置。 */
const columns: ColumnProps[] = [
  { type: "selection", width: 55 },
  { prop: "name", label: "部门名称", minWidth: 140, align: "left", search: { el: "input" } },
  { prop: "remark", label: "备注", minWidth: 160, search: { el: "input" } },
  { prop: "sort", label: "排序", minWidth: 90, align: "right" },
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
      disabled: () => !BUTTONS.value["base:dept:status"],
      beforeChange: scope => handleBeforeSetStatus(scope.row as BaseDept)
    }
  },
  { prop: "createdAt", label: "创建时间", minWidth: 180 },
  { prop: "updatedAt", label: "更新时间", minWidth: 180 },
  {
    prop: "operation",
    label: "操作",
    width: 220,
    fixed: "right",
    align: "left",
    cellType: "actions",
    actions: [
      {
        label: "新增",
        type: "primary",
        link: true,
        icon: CirclePlus,
        hidden: () => !BUTTONS.value["base:dept:create"],
        params: scope => ({ parentId: scope.row.id }),
        onClick: (_, params) => handleOpenDialog((params?.parentId as number | undefined) ?? 0)
      },
      {
        label: "编辑",
        type: "primary",
        link: true,
        icon: EditPen,
        hidden: () => !BUTTONS.value["base:dept:update"],
        params: scope => ({
          parentId: scope.row.parentId,
          deptId: scope.row.id
        }),
        onClick: (_, params) => handleOpenDialog(params?.parentId as number | undefined, params?.deptId as number | undefined)
      },
      {
        label: "删除",
        type: "danger",
        link: true,
        icon: Delete,
        hidden: () => !BUTTONS.value["base:dept:delete"],
        onClick: scope => handleDelete(scope.row as BaseDept)
      }
    ]
  }
];

/** 部门顶部按钮配置。 */
const headerActions: HeaderActionProps[] = [
  {
    label: "新增",
    type: "success",
    icon: CirclePlus,
    hidden: () => !BUTTONS.value["base:dept:create"],
    onClick: () => handleOpenDialog(0)
  },
  {
    label: "删除",
    type: "danger",
    icon: Delete,
    hidden: () => !BUTTONS.value["base:dept:delete"],
    disabled: scope => !scope.selectedList.length,
    onClick: scope => handleDelete(scope.selectedList as BaseDept[])
  }
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
 * 重置部门表单。
 */
function resetForm() {
  formDialogRef.value?.resetFields();
  formDialogRef.value?.clearValidate();
  formData.id = 0;
  formData.parentId = 0;
  formData.name = "";
  formData.sort = 1;
  formData.status = Status.ENABLE;
  formData.remark = "";
}

/**
 * 打开部门弹窗。
 */
async function handleOpenDialog(parentId?: number, deptId?: number) {
  resetForm();
  await loadDeptOptions();
  dialog.title = deptId ? "修改部门" : "新增部门";
  dialog.visible = true;
  if (deptId) {
    defBaseDeptService.GetBaseDept({ value: deptId }).then(data => {
      Object.assign(formData, data);
    });
    return;
  }

  formData.parentId = parentId ?? 0;
}

/**
 * 提交部门表单。
 */
function handleSubmit() {
  formDialogRef.value?.validate()?.then(valid => {
    if (!valid) return;

    const submitData = JSON.parse(JSON.stringify(formData)) as BaseDeptForm;
    const request = submitData.id ? defBaseDeptService.UpdateBaseDept(submitData) : defBaseDeptService.CreateBaseDept(submitData);
    request.then(() => {
      ElMessage.success(submitData.id ? "修改部门成功" : "新增部门成功");
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
    await ElMessageBox.confirm(`是否确定${text}部门？\n部门名称：${deptName}`, "提示", {
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
 * 删除部门，兼容单项删除与批量删除。
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
      ? `是否确定删除部门？\n部门名称：${deptList[0].name || `ID:${deptList[0].id}`}`
      : `确认删除已选中的 ${deptList.length} 个部门吗？`
    : "确认删除已选中的部门吗？";

  ElMessageBox.confirm(confirmMessage, "警告", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning"
  }).then(
    () => {
      defBaseDeptService.DeleteBaseDept({ value: deptIds }).then(() => {
        ElMessage.success("删除部门成功");
        refreshTable();
      });
    },
    () => {
      ElMessage.info("已取消删除部门");
    }
  );
}

/**
 * 关闭部门弹窗。
 */
function handleCloseDialog() {
  dialog.visible = false;
  resetForm();
}
</script>
