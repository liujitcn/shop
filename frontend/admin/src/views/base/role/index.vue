<template>
  <div class="app-container">
    <div class="search-bar">
      <el-form ref="queryFormRef" :model="queryParams" :inline="true">
        <el-form-item prop="name" label="角色名称">
          <el-input
            v-model="queryParams.name"
            placeholder="角色名称"
            clearable
            @keyup.enter="handleQuery"
          />
        </el-form-item>
        <el-form-item prop="name" label="角色编号">
          <el-input
            v-model="queryParams.code"
            placeholder="角色编号"
            clearable
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
      <div class="mb-10px">
        <el-button
          v-hasPerm="'base:role:create'"
          type="success"
          icon="plus"
          @click="handleOpenDialog()"
        >
          新增
        </el-button>
        <el-button
          v-hasPerm="'base:role:delete'"
          type="danger"
          :disabled="ids.length === 0"
          icon="delete"
          @click="handleDelete()"
        >
          删除
        </el-button>
      </div>

      <el-table
        ref="dataTableRef"
        v-loading="loading"
        :data="roleList"
        highlight-current-row
        border
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="55" align="center" />
        <el-table-column label="角色名称" prop="name" />
        <el-table-column label="角色编码" prop="code" />
        <el-table-column label="数据权限">
          <template #default="scope">
            <DictLabel v-model="scope.row.dataScope" code="base_role_data_scope" />
          </template>
        </el-table-column>
        <el-table-column prop="remark" label="备注" />
        <el-table-column label="状态" align="center" prop="status" width="80">
          <template #default="scope">
            <el-switch
              v-model="scope.row.status"
              inline-prompt
              :active-value="Status.ENABLE"
              :inactive-value="Status.DISABLE"
              active-text="启用"
              inactive-text="禁用"
              :disabled="!useUserStore().hasPerm('base:role:status')"
              @change="handleSetStatus(scope.row)"
            />
          </template>
        </el-table-column>
        <el-table-column label="创建时间" prop="createdAt" align="center" />
        <el-table-column label="更新时间" prop="updatedAt" align="center" />
        <el-table-column fixed="right" label="操作" width="220">
          <template #default="scope">
            <el-button
              v-hasPerm="'base:role:menus'"
              type="primary"
              size="small"
              link
              icon="position"
              @click="handleOpenAssignPermDialog(scope.row)"
            >
              分配权限
            </el-button>
            <el-button
              v-hasPerm="'base:role:update'"
              type="primary"
              size="small"
              link
              icon="edit"
              @click="handleOpenDialog(scope.row.id)"
            >
              编辑
            </el-button>
            <el-button
              v-hasPerm="'base:role:delete'"
              type="danger"
              size="small"
              link
              icon="delete"
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

    <!-- 角色表单弹窗 -->
    <el-dialog
      v-model="dialog.visible"
      :title="dialog.title"
      width="500px"
      @close="handleCloseDialog"
    >
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
          <el-input
            v-model="formData.remark"
            placeholder="请输入备注"
            type="textarea"
          />
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

    <!-- 分配权限弹窗 -->
    <el-drawer
      v-model="assignPermDialogVisible"
      :title="'【' + checkedBaseRole.name + '】权限分配'"
      size="500"
    >
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
          <el-checkbox
            v-model="parentChildLinked"
            class="ml-5"
            @change="handelParentChildLinkedChange"
          >
            父子联动
          </el-checkbox>

          <el-tooltip placement="bottom">
            <template #content>
              如果只需勾选菜单权限，不需要勾选子菜单或者按钮权限，请关闭父子联动
            </template>
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
import { TreeOptionResponse_Option } from "@/rpc/common/common";

defineOptions({
  name: "BaseRole",
  inheritAttrs: false,
});

import { defBaseRoleService } from "@/api/admin/base_role";
import type { BaseRole, BaseRoleForm, PageBaseRoleRequest } from "@/rpc/admin/base_role";
import { defBaseMenuService } from "@/api/admin/base_menu";
import { useUserStore } from "@/store";
import { Status } from "@/rpc/common/enum";

const queryFormRef = ref(ElForm);
const dataFormRef = ref(ElForm);
const permTreeRef = ref<InstanceType<typeof ElTree>>();

const loading = ref(false);
const ids = ref<number[]>([]);
const total = ref(0);

const queryParams = reactive<PageBaseRoleRequest>({
  /** 角色名称 */
  name: "",
  /** 角色编号 */
  code: "",
  /** 状态 */
  status: undefined,
  /** 当前页码 */
  pageNum: 0,
  /** 每一页的行数 */
  pageSize: 10,
});

// 角色表格数据
const roleList = ref<BaseRole[]>();
// 菜单权限下拉
const menuPermOptions = ref<TreeOptionResponse_Option[]>([]);

// 弹窗
const dialog = reactive({
  title: "",
  visible: false,
});
// 角色表单
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
  remark: "",
});

const rules = reactive({
  name: [{ required: true, message: "请输入角色名称", trigger: "blur" }],
  code: [{ required: true, message: "请输入角色编码", trigger: "blur" }],
  dataScope: [{ required: true, message: "请选择数据权限", trigger: "blur" }],
  menus: [{ required: true, message: "请选择菜单权限", trigger: "blur" }],
  status: [{ required: true, message: "请选择状态", trigger: "blur" }],
});

// 选中的角色
interface CheckedBaseRole {
  id?: number;
  name?: string;
}
const checkedBaseRole = ref<CheckedBaseRole>({});
const assignPermDialogVisible = ref(false);

const permKeywords = ref("");
const isExpanded = ref(true);

const parentChildLinked = ref(true);

// 查询
function handleQuery() {
  loading.value = true;
  defBaseRoleService
    .PageBaseRole(queryParams)
    .then((data) => {
      roleList.value = data.list;
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
  handleQuery();
}

// 行复选框选中
function handleSelectionChange(selection: any) {
  ids.value = selection.map((item: any) => item.id);
}

// 打开角色弹窗
async function handleOpenDialog(roleId?: number) {
  dialog.visible = true;
  // 获取所有的菜单
  const optionBaseMenuRes = await defBaseMenuService.OptionBaseMenu({});
  menuPermOptions.value = optionBaseMenuRes.list;
  if (roleId) {
    dialog.title = "修改角色";
    defBaseRoleService
      .GetBaseRole({
        value: roleId,
      })
      .then((data) => {
        Object.assign(formData, data);
      });
  } else {
    dialog.title = "新增角色";
  }
}
function handleCheck(currentNode: any, { checkedNodes }: { checkedNodes: any[] }) {
  // 提取所有选中节点的ID（包括父节点）
  formData.menus = checkedNodes.map((node) => node.value);
}

// 提交角色表单
function handleSubmit() {
  dataFormRef.value.validate((valid: any) => {
    if (valid) {
      loading.value = true;
      const roleId = formData.id;
      if (roleId) {
        defBaseRoleService
          .UpdateBaseRole(formData)
          .then(() => {
            ElMessage.success("修改成功");
            handleCloseDialog();
            handleResetQuery();
          })
          .finally(() => (loading.value = false));
      } else {
        defBaseRoleService
          .CreateBaseRole(formData)
          .then(() => {
            ElMessage.success("新增成功");
            handleCloseDialog();
            handleResetQuery();
          })
          .finally(() => (loading.value = false));
      }
    }
  });
}

// 关闭弹窗
function handleCloseDialog() {
  dialog.visible = false;

  dataFormRef.value.resetFields();
  dataFormRef.value.clearValidate();

  formData.id = 0;
}

// 设置角色状态
function handleSetStatus(row: BaseRole) {
  let text = row.status === Status.ENABLE ? "启用" : "禁用";
  ElMessageBox.confirm(`是否确定${text}角色为：${row.name}?`, "提示", {
    confirmButtonText: "确认",
    cancelButtonText: "取消",
    type: "warning",
  })
    .then(() => {
      defBaseRoleService.SetBaseRoleStatus({ id: row.id, status: row.status }).then(() => {
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

// 删除角色
function handleDelete(roleId?: number) {
  const roleIds = [roleId || ids.value].join(",");
  if (!roleIds) {
    ElMessage.warning("请勾选删除项");
    return;
  }

  ElMessageBox.confirm("确认删除已选中的数据项?", "警告", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning",
  }).then(
    () => {
      loading.value = true;
      defBaseRoleService
        .DeleteBaseRole({
          value: roleIds,
        })
        .then(() => {
          ElMessage.success("删除成功");
          handleResetQuery();
        })
        .finally(() => (loading.value = false));
    },
    () => {
      ElMessage.info("已取消删除");
    }
  );
}

// 打开分配菜单权限弹窗
async function handleOpenAssignPermDialog(row: BaseRole) {
  const roleId = row.id;
  if (roleId) {
    assignPermDialogVisible.value = true;

    checkedBaseRole.value.id = roleId;
    checkedBaseRole.value.name = row.name;

    // 获取所有的菜单
    const optionMenuRes = await defBaseMenuService.OptionBaseMenu({});
    menuPermOptions.value = optionMenuRes.list;

    // 回显角色已拥有的菜单
    permTreeRef.value!.setCheckedKeys(row.menus, false);
  }
}

// 分配菜单权限提交
function handleAssignPermSubmit() {
  const roleId = checkedBaseRole.value.id;
  if (roleId) {
    const checkedMenuIds: number[] = permTreeRef
      .value!.getCheckedNodes(false, true)
      .map((node: any) => node.value);

    loading.value = true;
    defBaseRoleService
      .SetBaseRoleMenus({
        id: roleId,
        menus: checkedMenuIds,
      })
      .then(() => {
        ElMessage.success("分配权限成功");
        assignPermDialogVisible.value = false;
        handleResetQuery();
      })
      .finally(() => {
        loading.value = false;
      });
  }
}

// 展开/收缩 菜单权限树
function togglePermTree() {
  isExpanded.value = !isExpanded.value;
  if (permTreeRef.value) {
    Object.values(permTreeRef.value.store.nodesMap).forEach((node: any) => {
      if (isExpanded.value) {
        node.expand();
      } else {
        node.collapse();
      }
    });
  }
}

// 权限筛选
watch(permKeywords, (val) => {
  permTreeRef.value!.filter(val);
});

function handlePermFilter(
  value: string,
  data: {
    [key: string]: any;
  }
) {
  if (!value) return true;
  return data.label.includes(value);
}

// 父子菜单节点是否联动
function handelParentChildLinkedChange(val: any) {
  parentChildLinked.value = val;
}

onMounted(() => {
  handleQuery();
});
</script>
