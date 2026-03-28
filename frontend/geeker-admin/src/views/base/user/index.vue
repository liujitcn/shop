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
      <ProTable ref="proTable" row-key="id" :columns="columns" :request-api="requestBaseUserTable" :init-param="initParam">
        <template #tableHeader="{ selectedList, selectedListIds }">
          <el-button v-hasPerm="['base:user:create']" type="success" :icon="CirclePlus" @click="handleOpenDialog()">
            新增
          </el-button>
          <el-button
            v-hasPerm="'base:user:delete'"
            type="danger"
            :icon="Delete"
            :disabled="!selectedListIds.length"
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
            :disabled="!BUTTONS['base:user:status']"
            :before-change="() => handleBeforeSetStatus(scope.row)"
          />
        </template>

        <template #operation="scope">
          <el-button v-hasPerm="'base:user:pwd'" type="primary" link :icon="RefreshLeft" @click="handleResetPassword(scope.row)">
            重置密码
          </el-button>
          <el-button v-hasPerm="'base:user:update'" type="primary" link :icon="EditPen" @click="handleOpenDialog(scope.row.id)">
            编辑
          </el-button>
          <el-button v-hasPerm="'base:user:delete'" type="danger" link :icon="Delete" @click="handleDelete(scope.row)">
            删除
          </el-button>
        </template>
      </ProTable>
    </div>

    <el-dialog v-model="dialog.visible" :title="dialog.title" width="800px" @closed="handleCloseDialog">
      <el-form ref="dataFormRef" :model="formData" :rules="rules" label-width="80px">
        <el-form-item label="用户账号" prop="userName">
          <el-input v-model="formData.userName" :readonly="!!formData.id" placeholder="请输入用户账号" />
        </el-form-item>

        <el-form-item label="用户昵称" prop="nickName">
          <el-input v-model="formData.nickName" placeholder="请输入用户昵称" />
        </el-form-item>

        <el-form-item label="角色" prop="roleId">
          <el-select v-model="formData.roleId" placeholder="请选择">
            <el-option v-for="item in baseRoleOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </el-form-item>

        <el-form-item label="用户部门" prop="deptId">
          <el-tree-select
            v-model="formData.deptId"
            placeholder="请选择用户部门"
            :data="basedDeptOptions"
            filterable
            check-strictly
            :render-after-expand="false"
          />
        </el-form-item>

        <el-form-item label="手机号码" prop="phone">
          <el-input v-model="formData.phone" placeholder="请输入手机号码" />
        </el-form-item>

        <el-form-item v-if="!formData.id" label="密码" prop="pwd">
          <el-input v-model="formData.pwd" placeholder="请输入密码" type="password" show-password />
        </el-form-item>

        <el-form-item label="性别" prop="gender">
          <Dict v-model="formData.gender" code="base_user_gender" />
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

        <el-form-item label="备注" prop="remark">
          <el-input v-model="formData.remark" placeholder="请输入备注" type="textarea" />
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
import { useDebounceFn } from "@vueuse/core";
import { ElMessage, ElMessageBox } from "element-plus";
import { CirclePlus, Delete, EditPen, RefreshLeft } from "@element-plus/icons-vue";
import type { ColumnProps, ProTableInstance } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
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
const dataFormRef = ref();

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
  deptId: [{ required: true, message: "用户部门不能为空", trigger: "blur" }],
  roleId: [{ required: true, message: "用户角色不能为空", trigger: "blur" }],
  phone: [
    {
      pattern: /^1[3|4|5|6|7|8|9][0-9]\d{8}$/,
      message: "请输入正确的手机号码",
      trigger: "blur"
    }
  ],
  pwd: [{ pattern: /^(?=.*[0-9])(.{6,18})$/, message: "请输入6-18位密码", trigger: "blur" }],
  status: [{ required: true, message: "用户状态不能为空", trigger: "blur" }]
});

const basedDeptOptions = ref<TreeOptionResponse_Option[]>([]);
const baseRoleOptions = ref<SelectOptionResponse_Option[]>([]);

/** 用户表格列配置。 */
const columns: ColumnProps[] = [
  { type: "selection", width: 55 },
  { prop: "userName", label: "用户账号", search: { el: "input" } },
  { prop: "nickName", label: "昵称", search: { el: "input" } },
  { prop: "phone", label: "手机号码", align: "center", search: { el: "input" } },
  { prop: "gender", label: "性别", align: "center", dictCode: "base_user_gender", search: { el: "select" } },
  { prop: "status", label: "状态", width: 100, dictCode: "status", search: { el: "select" } },
  { prop: "remark", label: "备注" },
  { prop: "createdAt", label: "创建时间", width: 180 },
  { prop: "updatedAt", label: "更新时间", width: 180 },
  { prop: "operation", label: "操作", width: 220, fixed: "right" }
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
  try {
    await ElMessageBox.confirm(`是否确定${text}用户为：${row.nickName}?`, "提示", {
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
  ElMessageBox.prompt(`请输入用户${row.userName ? `【${row.userName}】` : ""}的新密码`, "重置密码", {
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
  dialog.visible = true;
  const [optionBaseRoleResponse, optionBaseDeptResponse] = await Promise.all([
    defBaseRoleService.OptionBaseRole({}),
    defBaseDeptService.OptionBaseDept({})
  ]);
  baseRoleOptions.value = optionBaseRoleResponse.list || [];
  basedDeptOptions.value = optionBaseDeptResponse.list || [];

  if (id) {
    dialog.title = "修改用户";
    defBaseUserService.GetBaseUser({ value: id }).then(data => {
      Object.assign(formData, data);
    });
    return;
  }

  dialog.title = "新增用户";
  resetForm();
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
  dataFormRef.value?.resetFields();
  dataFormRef.value?.clearValidate();
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
  dataFormRef.value?.validate((valid: boolean) => {
    if (!valid) return;

    const request = formData.id ? defBaseUserService.UpdateBaseUser(formData) : defBaseUserService.CreateBaseUser(formData);
    request.then(() => {
      ElMessage.success(formData.id ? "修改用户成功" : "新增用户成功");
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
      ? `确认删除用户【${userList[0].nickName || "--"}】（账号：${userList[0].userName || "--"}）吗？`
      : `确认删除已选中的 ${userList.length} 个用户吗？`
    : "确认删除已选中的用户吗？";

  ElMessageBox.confirm(confirmMessage, "警告", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning"
  }).then(
    () => {
      defBaseUserService.DeleteBaseUser({ value: userIds }).then(() => {
        ElMessage.success("删除成功");
        refreshTable();
      });
    },
    () => {
      ElMessage.info("已取消删除");
    }
  );
}
</script>
