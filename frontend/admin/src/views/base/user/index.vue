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
    >
      <template #passwordStrength>
        <PasswordStrength :password="formData.pwd" />
      </template>
    </FormDialog>

    <FormDialog
      v-model="resetPwdDialog.visible"
      ref="resetPwdFormDialogRef"
      :title="resetPwdDialog.title"
      width="520px"
      :model="resetPwdForm"
      :fields="resetPwdFields"
      :rules="resetPwdRules"
      label-width="90px"
      @confirm="handleConfirmResetPassword"
      @close="handleCloseResetPasswordDialog"
    >
      <template #resetPwdStrength>
        <PasswordStrength :password="resetPwdForm.pwd" />
      </template>
    </FormDialog>
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
import PasswordStrength from "@/components/PasswordStrength/index.vue";
import type { ProFormField, ProFormOption } from "@/components/ProForm/interface";
import TreeFilter from "@/components/TreeFilter/index.vue";
import { useAuthButtons } from "@/hooks/useAuthButtons";
import { defBaseUserService } from "@/api/admin/base_user";
import type { BaseUser, BaseUserForm, PageBaseUsersRequest, ResetBaseUserPasswordRequest } from "@/rpc/admin/v1/base_user";
import { defBaseDeptService } from "@/api/admin/base_dept";
import { defBaseRoleService } from "@/api/admin/base_role";
import type { SelectOptionResponse_Option, TreeOptionResponse_Option } from "@/rpc/common/v1/common";
import { Status } from "@/rpc/common/v1/enum";
import { normalizeSelectedIds } from "@/utils/proTable";
import { PASSWORD_STRENGTH_ERROR_MESSAGE, validatePasswordStrengthValue } from "@/utils/passwordStrength";
import { PASSWORD_CRYPTO_SCENE, encryptPassword } from "@/utils/passwordCrypto";

interface BaseUserFormState extends Omit<BaseUserForm, "pwd"> {
  /** 密码明文只保留在前端表单中，提交前转换为密码密文。 */
  pwd: string;
}

interface ResetBaseUserPasswordFormState extends Omit<ResetBaseUserPasswordRequest, "pwd"> {
  /** 密码明文只保留在前端表单中，提交前转换为密码密文。 */
  pwd: string;
}

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
const resetPwdFormDialogRef = ref<InstanceType<typeof FormDialog>>();

const initParam = reactive({
  dept_id: undefined as number | undefined
});
const deptFilterValue = ref("");

const dialog = reactive({
  visible: false,
  title: "新增用户"
});
const resetPwdDialog = reactive({
  visible: false,
  title: "重置密码"
});

const formData = reactive<BaseUserFormState>({
  /** 用户ID */
  id: 0,
  /** 用户账号 */
  user_name: "",
  /** 用户昵称 */
  nick_name: "",
  /** 角色ID */
  role_id: undefined,
  /** 部门ID */
  dept_id: undefined,
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
const resetPwdForm = reactive<ResetBaseUserPasswordFormState>({
  id: 0,
  pwd: ""
});
const resetPwdTargetName = ref("");

const rules = reactive({
  user_name: [{ required: true, message: "用户账号不能为空", trigger: "blur" }],
  nick_name: [{ required: true, message: "用户昵称不能为空", trigger: "blur" }],
  dept_id: [{ required: true, message: "用户部门不能为空", trigger: "change" }],
  role_id: [{ required: true, message: "用户角色不能为空", trigger: "change" }],
  phone: [
    {
      pattern: /^1[3|4|5|6|7|8|9][0-9]\d{8}$/,
      message: "请输入正确的手机号码",
      trigger: "blur"
    }
  ],
  pwd: [
    { required: true, message: "请输入密码", trigger: "blur" },
    { validator: validatePasswordField, trigger: "blur" }
  ],
  status: [{ required: true, message: "用户状态不能为空", trigger: "change" }]
});
const resetPwdRules = reactive({
  pwd: [
    { required: true, message: "请输入新密码", trigger: "blur" },
    { validator: validatePasswordField, trigger: "blur" }
  ]
});

const basedDeptOptions = ref<TreeOptionResponse_Option[]>([]);
const baseRoleOptions = ref<SelectOptionResponse_Option[]>([]);
const statusOptions: ProFormOption[] = [
  { label: "启用", value: Status.ENABLE },
  { label: "禁用", value: Status.DISABLE }
];
const resetPwdFields: ProFormField[] = [
  {
    prop: "pwd",
    label: "新密码",
    component: "password",
    props: { placeholder: "请输入新密码", showPassword: true }
  },
  {
    prop: "resetPwdStrength",
    label: "强度提示",
    component: "slot",
    slotName: "resetPwdStrength"
  }
];

/** 用户表单字段配置。 */
const formFields = computed<ProFormField[]>(() => [
  {
    prop: "user_name",
    label: "用户账号",
    component: "input",
    props: { placeholder: "请输入用户账号", readonly: !!formData.id }
  },
  { prop: "nick_name", label: "用户昵称", component: "input", props: { placeholder: "请输入用户昵称" } },
  {
    prop: "role_id",
    label: "角色",
    component: "select",
    options: baseRoleOptions.value.map(item => ({ label: item.label, value: item.value })),
    props: { placeholder: "请选择" }
  },
  {
    prop: "dept_id",
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
  {
    prop: "passwordStrength",
    label: "强度提示",
    component: "slot",
    slotName: "passwordStrength",
    visible: model => !model.id
  },
  { prop: "gender", label: "性别", component: "dict", props: { code: "base_user_gender" } },
  { prop: "status", label: "状态", component: "radio-group", options: statusOptions },
  { prop: "remark", label: "备注", component: "textarea", props: { placeholder: "请输入备注" } }
]);

/** 用户表格列配置。 */
const columns: ColumnProps[] = [
  { type: "selection", width: 55 },
  { prop: "user_name", label: "用户账号", minWidth: 140, search: { el: "input" } },
  { prop: "nick_name", label: "昵称", minWidth: 100, search: { el: "input" } },
  { prop: "phone", label: "手机号码", minWidth: 130, align: "center", search: { el: "input" } },
  { prop: "gender", label: "性别", minWidth: 90, align: "center", dictCode: "base_user_gender", search: { el: "select" } },
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
      disabled: () => !BUTTONS.value["base:user:status"],
      beforeChange: scope => handleBeforeSetStatus(scope.row as BaseUser)
    }
  },
  { prop: "remark", label: "备注", minWidth: 160 },
  { prop: "created_at", label: "创建时间", minWidth: 180 },
  { prop: "updated_at", label: "更新时间", minWidth: 180 },
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
  const response = await defBaseDeptService.OptionBaseDepts({});
  return {
    data: transformDeptFilterNodes(response.list ?? [])
  };
}

/**
 * 切换部门树筛选时同步更新表格初始化参数。
 */
function changeTreeFilter(value: string) {
  deptFilterValue.value = value ?? "";
  initParam.dept_id = value ? Number(value) : undefined;
  if (proTable.value) {
    proTable.value.pageable.pageNum = 1;
  }
}

/**
 * 请求用户分页列表，并统一处理分页参数。
 */
async function requestBaseUserTable(params: Partial<PageBaseUsersRequest> & { pageNum?: number; pageSize?: number }) {
  const data = await defBaseUserService.PageBaseUsers({
    user_name: params.user_name ?? "",
    nick_name: params.nick_name ?? "",
    dept_id: initParam.dept_id,
    phone: params.phone ?? "",
    status: params.status,
    page_num: Number(params.page_num ?? params.pageNum ?? 1),
    page_size: Number(params.page_size ?? params.pageSize ?? 10)
  });
  return { data: { list: data.base_users, total: data.total } };
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
  const user_name = row.nick_name || row.user_name || `ID:${row.id}`;
  try {
    await ElMessageBox.confirm(`是否确定${text}用户？\n用户名称：${user_name}`, "提示", {
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
 * 打开重置密码弹窗，并回填当前操作用户。
 */
function handleResetPassword(row: BaseUser) {
  resetPwdFormDialogRef.value?.resetFields();
  resetPwdFormDialogRef.value?.clearValidate();
  resetPwdForm.id = row.id;
  resetPwdForm.pwd = "";
  resetPwdTargetName.value = row.nick_name || row.user_name || `ID:${row.id}`;
  resetPwdDialog.title = `重置密码：${resetPwdTargetName.value}`;
  resetPwdDialog.visible = true;
}

/**
 * 打开用户弹窗，并加载角色和部门下拉选项。
 */
async function handleOpenDialog(id?: number) {
  resetForm();
  const [optionBaseRoleResponse, optionBaseDeptResponse] = await Promise.all([
    defBaseRoleService.OptionBaseRoles({}),
    defBaseDeptService.OptionBaseDepts({})
  ]);
  baseRoleOptions.value = optionBaseRoleResponse.list || [];
  basedDeptOptions.value = optionBaseDeptResponse.list || [];

  dialog.title = id ? "修改用户" : "新增用户";
  dialog.visible = true;
  if (!id) return;

  defBaseUserService.GetBaseUser({ id }).then(data => {
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
 * 关闭重置密码弹窗并恢复默认表单值。
 */
function handleCloseResetPasswordDialog() {
  resetPwdDialog.visible = false;
  resetPwdFormDialogRef.value?.resetFields();
  resetPwdFormDialogRef.value?.clearValidate();
  resetPwdForm.id = 0;
  resetPwdForm.pwd = "";
  resetPwdTargetName.value = "";
}

/**
 * 重置用户表单，避免新增与编辑之间互相污染。
 */
function resetForm() {
  formDialogRef.value?.resetFields();
  formDialogRef.value?.clearValidate();
  formData.id = 0;
  formData.user_name = "";
  formData.nick_name = "";
  formData.role_id = undefined;
  formData.dept_id = undefined;
  formData.phone = "";
  formData.pwd = "";
  formData.gender = 3;
  formData.avatar = "";
  formData.status = Status.ENABLE;
  formData.remark = "";
}

/**
 * 确认重置用户密码，并复用统一密码强度校验。
 */
async function handleConfirmResetPassword() {
  resetPwdFormDialogRef.value?.validate()?.then(async valid => {
    if (!valid) return;

    const pwd = await encryptPassword(resetPwdForm.pwd, PASSWORD_CRYPTO_SCENE.RESET_BASE_USER_PASSWORD);
    defBaseUserService.ResetBaseUserPassword({ id: resetPwdForm.id, pwd }).then(() => {
      ElMessage.success(`重置密码成功\n用户名称：${resetPwdTargetName.value}`);
      handleCloseResetPasswordDialog();
    });
  });
}

/**
 * 提交用户表单，使用防抖避免重复提交。
 */
const handleSubmit = useDebounceFn(() => {
  formDialogRef.value?.validate()?.then(async valid => {
    if (!valid) return;

    const submitData = JSON.parse(JSON.stringify(formData)) as BaseUserFormState;
    const baseUser = {
      ...submitData,
      pwd: submitData.id ? undefined : await encryptPassword(submitData.pwd, PASSWORD_CRYPTO_SCENE.CREATE_BASE_USER)
    } as BaseUserForm;
    const request = submitData.id
      ? defBaseUserService.UpdateBaseUser({ base_user: baseUser })
      : defBaseUserService.CreateBaseUser({ base_user: baseUser });
    request.then(() => {
      ElMessage.success(submitData.id ? "修改用户成功" : "新增用户成功");
      handleCloseDialog();
      refreshTable();
    });
  });
}, 1000);

/**
 * 校验密码强度，新增用户和重置密码统一要求达到最高强度。
 *
 * @param _rule 表单规则对象
 * @param value 当前密码值
 * @param callback 校验回调
 */
function validatePasswordField(_rule: unknown, value: string, callback: (error?: Error) => void) {
  if (!value) {
    callback();
    return;
  }
  const result = validatePasswordStrengthValue(value);
  if (!result.valid) {
    callback(new Error(result.message || PASSWORD_STRENGTH_ERROR_MESSAGE));
    return;
  }
  callback();
}

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
      ? `是否确定删除用户？\n用户名称：${userList[0].nick_name || userList[0].user_name || `ID:${userList[0].id}`}`
      : `确认删除已选中的 ${userList.length} 个用户吗？`
    : "确认删除已选中的用户吗？";

  ElMessageBox.confirm(confirmMessage, "警告", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning"
  }).then(
    () => {
      defBaseUserService.DeleteBaseUser({ id: userIds }).then(() => {
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
