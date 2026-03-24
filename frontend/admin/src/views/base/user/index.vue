<!-- 用户管理 -->
<template>
  <div class="app-container">
    <el-row :gutter="20">
      <!-- 部门树 -->
      <el-col :lg="4" :xs="24" class="mb-[12px]">
        <DeptTree v-model="queryParams.deptId" @node-click="handleQuery" />
      </el-col>

      <!-- 用户列表 -->
      <el-col :lg="20" :xs="24">
        <div class="search-bar">
          <el-form ref="queryFormRef" :model="queryParams" :inline="true">
            <el-form-item label="用户账号" prop="userName">
              <el-input
                v-model="queryParams.userName"
                placeholder="用户账号"
                clearable
                style="width: 200px"
                @keyup.enter="handleQuery"
              />
            </el-form-item>
            <el-form-item label="用户昵称" prop="nickName">
              <el-input
                v-model="queryParams.nickName"
                placeholder="用户昵称"
                clearable
                style="width: 200px"
                @keyup.enter="handleQuery"
              />
            </el-form-item>
            <el-form-item label="手机号" prop="phone">
              <el-input
                v-model="queryParams.phone"
                placeholder="手机号"
                clearable
                style="width: 200px"
                @keyup.enter="handleQuery"
              />
            </el-form-item>

            <el-form-item label="状态" prop="status">
              <Dict v-model="queryParams.status" code="status" />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" icon="search" @click="handleQuery">搜索</el-button>
              <el-button icon="refresh" @click="handleResetQuery">重置</el-button>
            </el-form-item>
          </el-form>
        </div>

        <el-card shadow="never">
          <div class="flex-x-between mb-10px">
            <div>
              <el-button
                v-hasPerm="['base:user:create']"
                type="success"
                icon="plus"
                @click="handleOpenDialog()"
              >
                新增
              </el-button>
              <el-button
                v-hasPerm="'base:user:delete'"
                type="danger"
                icon="delete"
                :disabled="ids.length === 0"
                @click="handleDelete()"
              >
                删除
              </el-button>
            </div>
          </div>

          <el-table v-loading="loading" :data="pageData" @selection-change="handleSelectionChange">
            <el-table-column type="selection" width="55" align="center" />
            <el-table-column label="用户账号" prop="userName" />
            <el-table-column label="昵称" prop="nickName" />
            <el-table-column label="手机号码" align="center" prop="phone" />
            <el-table-column label="性别" align="center">
              <template #default="scope">
                <!-- 性别字典翻译 -->
                <DictLabel v-model="scope.row.gender" code="base_user_gender" />
              </template>
            </el-table-column>
            <el-table-column label="状态" align="center" prop="status" width="80">
              <template #default="scope">
                <el-switch
                  v-model="scope.row.status"
                  inline-prompt
                  :active-value="Status.ENABLE"
                  :inactive-value="Status.DISABLE"
                  active-text="启用"
                  inactive-text="禁用"
                  :disabled="!useUserStore().hasPerm('base:user:status')"
                  @change="handleSetStatus(scope.row)"
                />
              </template>
            </el-table-column>
            <el-table-column prop="remark" label="备注" />
            <el-table-column label="创建时间" prop="createdAt" align="center" />
            <el-table-column label="更新时间" prop="updatedAt" align="center" />
            <el-table-column label="操作" fixed="right" width="220">
              <template #default="scope">
                <el-button
                  v-hasPerm="'base:user:pwd'"
                  type="primary"
                  icon="RefreshLeft"
                  size="small"
                  link
                  @click="handleResetPassword(scope.row)"
                >
                  重置密码
                </el-button>
                <el-button
                  v-hasPerm="'base:user:update'"
                  type="primary"
                  icon="edit"
                  link
                  size="small"
                  @click="handleOpenDialog(scope.row.id)"
                >
                  编辑
                </el-button>
                <el-button
                  v-hasPerm="'base:user:delete'"
                  type="danger"
                  icon="delete"
                  link
                  size="small"
                  @click="handleDelete(scope.row.id)"
                >
                  删除
                </el-button>
              </template>
            </el-table-column>
          </el-table>

          <pagination
            v-if="total > 0"
            v-model:total="total"
            v-model:page="queryParams.pageNum"
            v-model:limit="queryParams.pageSize"
            @pagination="handleQuery"
          />
        </el-card>
      </el-col>
    </el-row>

    <!-- 用户表单 -->
    <el-dialog
      v-model="dialog.visible"
      :title="dialog.title"
      width="800px"
      @closed="handleCloseDialog"
    >
      <el-form ref="dataFormRef" :model="formData" :rules="rules" label-width="80px">
        <el-form-item label="用户账号" prop="userName">
          <el-input
            v-model="formData.userName"
            :readonly="!!formData.id"
            placeholder="请输入用户账号"
          />
        </el-form-item>

        <el-form-item label="用户昵称" prop="nickName">
          <el-input v-model="formData.nickName" placeholder="请输入用户昵称" />
        </el-form-item>

        <el-form-item label="角色" prop="roleId">
          <el-select v-model="formData.roleId" placeholder="请选择">
            <el-option
              v-for="item in baseRoleOptions"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
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
          <el-input
            v-model="formData.remark"
            placeholder="请输入备注"
            type="textarea"
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
import { defBaseUserService } from "@/api/admin/base_user";
import type { BaseUser, BaseUserForm, PageBaseUserRequest } from "@/rpc/admin/base_user";

import { defBaseDeptService } from "@/api/admin/base_dept";
import { defBaseRoleService } from "@/api/admin/base_role";
import { useUserStore } from "@/store";

import DeptTree from "./components/dept-tree.vue";
import { SelectOptionResponse_Option, TreeOptionResponse_Option } from "@/rpc/common/common";
import { Status } from "@/rpc/common/enum";

defineOptions({
  name: "BaseUser",
  inheritAttrs: false,
});

const queryFormRef = ref(ElForm);
const dataFormRef = ref(ElForm);

const queryParams = reactive<PageBaseUserRequest>({
  /** 用户账号 */
  userName: "",
  /** 用户昵称 */
  nickName: "",
  /** 手机号 */
  phone: "",
  /** 部门ID */
  deptId: undefined,
  /** 状态 */
  status: undefined,
  /** 当前页码 */
  pageNum: 0,
  /** 每一页的行数 */
  pageSize: 10,
});

const pageData = ref<BaseUser[]>();
const total = ref(0);
const loading = ref(false);

const dialog = reactive({
  visible: false,
  title: "新增用户",
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
  remark: "",
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
      trigger: "blur",
    },
  ],
  pwd: [{ pattern: /^(?=.*[0-9])(.{6,18})$/, message: "请输入6-18位密码", trigger: "blur" }],
  status: [{ required: true, message: "用户状态不能为空", trigger: "blur" }],
});

// 选中的用户ID
const ids = ref<number[]>([]);
// 部门下拉数据源
const basedDeptOptions = ref<TreeOptionResponse_Option[]>();
// 角色下拉数据源
const baseRoleOptions = ref<SelectOptionResponse_Option[]>();

// 查询
function handleQuery() {
  loading.value = true;
  defBaseUserService
    .PageBaseUser(queryParams)
    .then((data) => {
      pageData.value = data.list;
      total.value = data.total;
    })
    .finally(() => {
      loading.value = false;
    });
}

// 重置查询
function handleResetQuery() {
  queryFormRef.value.resetFields();
  queryParams.pageNum = 1;
  queryParams.deptId = undefined;
  handleQuery();
}

// 选中项发生变化
function handleSelectionChange(selection: any[]) {
  ids.value = selection.map((item) => item.id);
}

// 设置用户状态
function handleSetStatus(row: BaseUser) {
  let text = row.status === Status.ENABLE ? "启用" : "禁用";
  ElMessageBox.confirm(`是否确定${text}用户为：${row.nickName}?`, "提示", {
    confirmButtonText: "确认",
    cancelButtonText: "取消",
    type: "warning",
  })
    .then(() => {
      defBaseUserService.SetBaseUserStatus({ id: row.id, status: row.status }).then(() => {
        handleQuery();
        ElMessage.success(`${text}成功`);
      });
    })
    .catch(() => {
      if (row.status == 0) {
        row.status = 1;
      } else {
        row.status = 0;
      }
    });
}

// 重置密码
function handleResetPassword(row: BaseUser) {
  ElMessageBox.prompt(
    `请输入用户${row.userName ? "【" + row.userName + "】" : ""}的新密码`,
    "重置密码",
    {
      confirmButtonText: "确定",
      cancelButtonText: "取消",
    }
  ).then(
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
 * 打开弹窗
 *
 * @param id 用户ID
 */
async function handleOpenDialog(id?: number) {
  dialog.visible = true;
  // 加载角色下拉数据源
  const optionBaseRoleResponse = await defBaseRoleService.OptionBaseRole({});
  baseRoleOptions.value = optionBaseRoleResponse.list || [];
  // 加载部门下拉数据源
  const optionBaseDeptResponse = await defBaseDeptService.OptionBaseDept({});
  basedDeptOptions.value = optionBaseDeptResponse.list || [];

  if (id) {
    dialog.title = "修改用户";
    defBaseUserService
      .GetBaseUser({
        value: id,
      })
      .then((data) => {
        Object.assign(formData, { ...data });
      });
  } else {
    dialog.title = "新增用户";
  }
}

// 关闭弹窗
function handleCloseDialog() {
  dialog.visible = false;
  dataFormRef.value.resetFields();
  dataFormRef.value.clearValidate();

  formData.id = 0;
}

// 提交用户表单（防抖）
const handleSubmit = useDebounceFn(() => {
  dataFormRef.value.validate((valid: boolean) => {
    if (valid) {
      const userId = formData.id;
      loading.value = true;
      if (userId) {
        defBaseUserService
          .UpdateBaseUser(formData)
          .then(() => {
            ElMessage.success("修改用户成功");
            handleCloseDialog();
            handleResetQuery();
          })
          .finally(() => (loading.value = false));
      } else {
        defBaseUserService
          .CreateBaseUser(formData)
          .then(() => {
            ElMessage.success("新增用户成功");
            handleCloseDialog();
            handleResetQuery();
          })
          .finally(() => (loading.value = false));
      }
    }
  });
}, 1000);

/**
 * 删除用户
 *
 * @param userId  用户ID
 */
function handleDelete(userId?: number) {
  const userIds = [userId || ids.value].join(",");
  if (!userIds) {
    ElMessage.warning("请勾选删除项");
    return;
  }

  ElMessageBox.confirm("确认删除用户?", "警告", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning",
  }).then(
    function () {
      loading.value = true;
      defBaseUserService
        .DeleteBaseUser({
          value: userIds,
        })
        .then(() => {
          ElMessage.success("删除成功");
          handleResetQuery();
        })
        .finally(() => (loading.value = false));
    },
    function () {
      ElMessage.info("已取消删除");
    }
  );
}

onMounted(() => {
  handleQuery();
});
</script>
