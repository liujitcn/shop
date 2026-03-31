<!-- 用户管理 -->
<template>
  <div class="main-box">
    <TreeFilter
      label="name"
      title="部门列表"
      :request-api="requestDeptTreeFilter"
      :show-all="false"
      :default-value="deptFilterValue"
      @change="changeTreeFilter"
    />

    <div class="table-box">
      <ProTable
        ref="proTable"
        row-key="id"
        :columns="columns"
        :header-actions="headerActions"
        :request-api="requestBaseUserTable"
        :init-param="initParam"
      />
    </div>

    <FormDialog
      v-model="dialog.visible"
      ref="formDialogRef"
      :title="dialog.title"
      width="800px"
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
import { useDebounceFn } from "@vueuse/core";
import { ElMessage, ElMessageBox } from "element-plus";
import { CirclePlus, Delete, EditPen, RefreshLeft } from "@element-plus/icons-vue";
import type { ColumnProps, HeaderActionProps, ProTableInstance } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
import FormDialog from "@/components/Dialog/FormDialog.vue";
import type { ProFormField, ProFormOption } from "@/components/ProForm/interface";
import TreeFilter from "@/components/TreeFilter/index.vue";
import { useAuthButtons } from "@/hooks/useAuthButtons";
import { defBaseUserService } from "@/api/admin/base_user";
import type { BaseUser, BaseUserForm, PageBaseUserRequest } from "@/rpc/admin/base_user";
import { defBaseDeptService } from "@/api/admin/base_dept";
import { defBaseRoleService } from "@/api/admin/base_role";
import type { SelectOptionResponse_Option, TreeOptionResponse_Option } from "@/rpc/common/common";
import { Status } from "@/rpc/common/enum";
import { buildPageRequest, normalizeSelectedIds } from "@/utils/proTable";

defineOptions({
  name: "BaseUser",
  inheritAttrs: false
});

type DeptFilterNode = {
  id: string;
  name: string;
  children?: DeptFilterNode[];
};

const { BUTTONS } = useAuthButtons();
const proTable = ref<ProTableInstance>();
const formDialogRef = ref<InstanceType<typeof FormDialog>>();

const initParam = reactive({
  deptId: undefined as number | undefined
});
const deptFilterValue = ref("");

const dialog = reactive({
  visible: false,
  title: "新增用户"
});

const formData = reactive<BaseUserForm>({
  /** 用户ID */
  id: 0,
  /** 用户账号 */
  userName: "",
  /** 用户昵称 */
  nickName: "",
  /** 角色ID */
  roleId: undefined,
  /** 部门ID */
  deptId: undefined,
  /** 手机号 */
  phone: "",
  /** 密码 */
  pwd: "",
  /** 性别 */
  gender: 3,
  /** 头像 */
  avatar: "",
  /** 用户状态 */
  status: Status.ENABLE,
  /** 备注名 */
  remark: ""
});

const rules = reactive({
  userName: [{ required: true, message: "用户账号不能为空", trigger: "blur" }],
  nickName: [{ required: true, message: "用户昵称不能为空", trigger: "blur" }],
  deptId: [{ required: true, message: "用户部门不能为空", trigger: "change" }],
  roleId: [{ required: true, message: "用户角色不能为空", trigger: "change" }],
  phone: [
    {
      pattern: /^1[3|4|5|6|7|8|9][0-9]\d{8}$/,
      message: "请输入正确的手机号码",
      trigger: "blur"
    }
  ],
  pwd: [{ pattern: /^(?=.*[0-9])(.{6,18})$/, message: "请输入6-18位密码", trigger: "blur" }],
  status: [{ required: true, message: "用户状态不能为空", trigger: "change" }]
});

const basedDeptOptions = ref<TreeOptionResponse_Option[]>([]);
const baseRoleOptions = ref<SelectOptionResponse_Option[]>([]);
const statusOptions: ProFormOption[] = [
  { label: "启用", value: Status.ENABLE },
  { label: "禁用", value: Status.DISABLE }
];

/** 用户表单字段配置。 */
const formFields = computed<ProFormField[]>(() => [
  {
    prop: "userName",
    label: "用户账号",
    component: "input",
    props: { placeholder: "请输入用户账号", readonly: !!formData.id }
  },
  { prop: "nickName", label: "用户昵称", component: "input", props: { placeholder: "请输入用户昵称" } },
  {
    prop: "roleId",
    label: "角色",
    component: "select",
    options: baseRoleOptions.value.map(item => ({ label: item.label, value: item.value })),
    props: { placeholder: "请选择" }
  },
  {
    prop: "deptId",
    label: "用户部门",
    component: "tree-select",
    options: basedDeptOptions.value as unknown as ProFormOption[],
    props: {
      placeholder: "请选择用户部门",
      filterable: true,
      checkStrictly: true,
      renderAfterExpand: false,
      style: { width: "100%" }
    }
  },
  { prop: "phone", label: "手机号码", component: "input", props: { placeholder: "请输入手机号码" } },
  {
    prop: "pwd",
    label: "密码",
    component: "password",
    props: { placeholder: "请输入密码", showPassword: true },
    visible: model => !model.id
  },
  { prop: "gender", label: "性别", component: "dict", props: { code: "base_user_gender" } },
  { prop: "status", label: "状态", component: "radio-group", options: statusOptions },
  { prop: "remark", label: "备注", component: "textarea", props: { placeholder: "请输入备注" } }
]);

/** 用户表格列配置。 */
const columns: ColumnProps[] = [
  { type: "selection", width: 55 },
  { prop: "userName", label: "用户账号", search: { el: "input" } },
  { prop: "nickName", label: "昵称", search: { el: "input" } },
  { prop: "phone", label: "手机号码", align: "center", search: { el: "input" } },
  { prop: "gender", label: "性别", align: "center", dictCode: "base_user_gender", search: { el: "select" } },
  {
    prop: "status",
    label: "状态",
    width: 100,
    search: { el: "select" },
    cellType: "status",
    statusProps: {
      activeValue: Status.ENABLE,
      inactiveValue: Status.DISABLE,
      activeText: "启用",
      inactiveText: "禁用",
      disabled: () => !BUTTONS.value["base:user:status"],
      beforeChange: scope => handleBeforeSetStatus(scope.row as BaseUser)
    }
  },
  { prop: "remark", label: "备注" },
  { prop: "createdAt", label: "创建时间", width: 180 },
  { prop: "updatedAt", label: "更新时间", width: 180 },
  {
    prop: "operation",
    label: "操作",
    width: 260,
    fixed: "right",
    cellType: "actions",
    actions: [
      {
        label: "重置密码",
        type: "primary",
        link: true,
        icon: RefreshLeft,
        hidden: () => !BUTTONS.value["base:user:pwd"],
        onClick: scope => handleResetPassword(scope.row as BaseUser)
      },
      {
        label: "编辑",
        type: "primary",
        link: true,
        icon: EditPen,
        hidden: () => !BUTTONS.value["base:user:update"],
        params: scope => ({ userId: scope.row.id }),
        onClick: (scope, params) => handleOpenDialog((params?.userId as number | undefined) ?? (scope.row as BaseUser).id)
      },
      {
        label: "删除",
        type: "danger",
        link: true,
        icon: Delete,
        hidden: () => !BUTTONS.value["base:user:delete"],
        onClick: scope => handleDelete(scope.row as BaseUser)
      }
    ]
  }
];

/** 用户顶部按钮配置。 */
const headerActions: HeaderActionProps[] = [
  {
    label: "新增",
    type: "success",
    icon: CirclePlus,
    hidden: () => !BUTTONS.value["base:user:create"],
    onClick: () => handleOpenDialog()
  },
  {
    label: "删除",
    type: "danger",
    icon: Delete,
    hidden: () => !BUTTONS.value["base:user:delete"],
    disabled: scope => !scope.selectedList.length,
    onClick: scope => handleDelete(scope.selectedList as BaseUser[])
  }
];

/**
 * 递归转换部门树筛选数据，适配 TreeFilter 组件的 id/name 字段。
 */
function transformDeptFilterNodes(options: TreeOptionResponse_Option[] = []): DeptFilterNode[] {
  return options.map(option => ({
    id: String(option.value),
    name: option.label,
    children: transformDeptFilterNodes(option.children ?? [])
  }));
}

/**
 * 请求部门树筛选数据。
 */
async function requestDeptTreeFilter() {
  const response = await defBaseDeptService.OptionBaseDept({});
  return {
    data: transformDeptFilterNodes(response.list ?? [])
  };
}

/**
 * 切换部门树筛选时同步更新表格初始化参数。
 */
function changeTreeFilter(value: string) {
  deptFilterValue.value = value ?? "";
  initParam.deptId = value ? Number(value) : undefined;
  if (proTable.value) {
    proTable.value.pageable.pageNum = 1;
  }
}

/**
 * 请求用户分页列表，并统一处理分页参数。
 */
async function requestBaseUserTable(params: PageBaseUserRequest) {
  const data = await defBaseUserService.PageBaseUser(
    buildPageRequest({
      ...params,
      deptId: initParam.deptId
    })
  );
  return { data };
}

/**
 * 刷新用户表格数据。
 */
function refreshTable() {
  proTable.value?.getTableList();
}

/**
 * 在用户状态切换前先完成确认与接口调用，避免首屏渲染触发误操作。
 */
async function handleBeforeSetStatus(row: BaseUser) {
  const nextStatus = row.status === Status.ENABLE ? Status.DISABLE : Status.ENABLE;
  const text = nextStatus === Status.ENABLE ? "启用" : "禁用";
  const userName = row.nickName || row.userName || `ID:${row.id}`;
  try {
    await ElMessageBox.confirm(`是否确定${text}用户？\n用户名称：${userName}`, "提示", {
      confirmButtonText: "确认",
      cancelButtonText: "取消",
      type: "warning"
    });
    await defBaseUserService.SetBaseUserStatus({ id: row.id, status: nextStatus });
    ElMessage.success(`${text}成功`);
    refreshTable();
    return true;
  } catch {
    return false;
  }
}

/**
 * 重置用户密码。
 */
function handleResetPassword(row: BaseUser) {
  const userName = row.nickName || row.userName || `ID:${row.id}`;
  ElMessageBox.prompt(`请输入新密码\n用户名称：${userName}`, "重置密码", {
    confirmButtonText: "确定",
    cancelButtonText: "取消"
  }).then(
    ({ value }) => {
      if (!/^(?=.*[0-9])(.{6,18})$/.test(value)) {
        ElMessage.warning("请输入6-18位密码");
        return false;
      }

      defBaseUserService.ResetBaseUserPwd({ id: row.id, pwd: value }).then(() => {
        ElMessage.success(`密码重置成功，新密码是：${value}`);
      });
    },
    () => {
      ElMessage.info("已取消重置密码");
    }
  );
}

/**
 * 打开用户弹窗，并加载角色和部门下拉选项。
 */
async function handleOpenDialog(id?: number) {
  resetForm();
  const [optionBaseRoleResponse, optionBaseDeptResponse] = await Promise.all([
    defBaseRoleService.OptionBaseRole({}),
    defBaseDeptService.OptionBaseDept({})
  ]);
  baseRoleOptions.value = optionBaseRoleResponse.list || [];
  basedDeptOptions.value = optionBaseDeptResponse.list || [];

  dialog.title = id ? "修改用户" : "新增用户";
  dialog.visible = true;
  if (!id) return;

  defBaseUserService.GetBaseUser({ value: id }).then(data => {
    Object.assign(formData, data);
  });
}

/**
 * 关闭用户弹窗并恢复默认表单值。
 */
function handleCloseDialog() {
  dialog.visible = false;
  resetForm();
}

/**
 * 重置用户表单，避免新增与编辑之间互相污染。
 */
function resetForm() {
  formDialogRef.value?.resetFields();
  formDialogRef.value?.clearValidate();
  formData.id = 0;
  formData.userName = "";
  formData.nickName = "";
  formData.roleId = undefined;
  formData.deptId = undefined;
  formData.phone = "";
  formData.pwd = "";
  formData.gender = 3;
  formData.avatar = "";
  formData.status = Status.ENABLE;
  formData.remark = "";
}

/**
 * 提交用户表单，使用防抖避免重复提交。
 */
const handleSubmit = useDebounceFn(() => {
  formDialogRef.value?.validate()?.then(valid => {
    if (!valid) return;

    const submitData = JSON.parse(JSON.stringify(formData)) as BaseUserForm;
    const request = submitData.id ? defBaseUserService.UpdateBaseUser(submitData) : defBaseUserService.CreateBaseUser(submitData);
    request.then(() => {
      ElMessage.success(submitData.id ? "修改用户成功" : "新增用户成功");
      handleCloseDialog();
      refreshTable();
    });
  });
}, 1000);

/**
 * 删除用户，兼容单条删除与批量删除。
 */
function handleDelete(selected?: number | string | Array<number | string> | BaseUser | BaseUser[]) {
  const userList = Array.isArray(selected)
    ? selected.filter(item => typeof item === "object")
    : selected && typeof selected === "object"
      ? [selected]
      : [];
  const userIds = normalizeSelectedIds(
    userList.length ? userList.map(item => item.id) : (selected as number | string | Array<number | string> | undefined)
  ).join(",");
  if (!userIds) {
    ElMessage.warning("请勾选删除项");
    return;
  }

  const confirmMessage = userList.length
    ? userList.length === 1
      ? `是否确定删除用户？\n用户名称：${userList[0].nickName || userList[0].userName || `ID:${userList[0].id}`}`
      : `确认删除已选中的 ${userList.length} 个用户吗？`
    : "确认删除已选中的用户吗？";

  ElMessageBox.confirm(confirmMessage, "警告", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning"
  }).then(
    () => {
      defBaseUserService.DeleteBaseUser({ value: userIds }).then(() => {
        ElMessage.success("删除用户成功");
        refreshTable();
      });
    },
    () => {
      ElMessage.info("已取消删除用户");
    }
  );
}
</script>
