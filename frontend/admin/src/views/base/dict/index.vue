<!-- 字典 -->
<template>
  <div class="app-container">
    <div class="search-bar">
      <el-form ref="queryFormRef" :model="queryParams" :inline="true">
        <el-form-item label="字典名称" prop="name">
          <el-input
            v-model="queryParams.name"
            placeholder="字典名称"
            clearable
            @keyup.enter="handleQuery"
          />
        </el-form-item>
        <el-form-item label="字典编码" prop="code">
          <el-input
            v-model="queryParams.code"
            placeholder="字典编码"
            clearable
            @keyup.enter="handleQuery"
          />
        </el-form-item>

        <el-form-item label="状态" prop="status">
          <Dict v-model="queryParams.status" code="status" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" icon="search" @click="handleQuery()">搜索</el-button>
          <el-button icon="refresh" @click="handleResetQuery()">重置</el-button>
        </el-form-item>
      </el-form>
    </div>

    <el-card shadow="never">
      <div class="mb-[10px]">
        <el-button
          v-hasPerm="['base:dict:create']"
          type="success"
          icon="plus"
          @click="handleOpenDialog()"
        >
          新增
        </el-button>
        <el-button
          v-hasPerm="['base:dict:delete']"
          type="danger"
          :disabled="ids.length === 0"
          icon="delete"
          @click="handleDelete()"
        >
          删除
        </el-button>
      </div>

      <el-table
        v-loading="loading"
        highlight-current-row
        :data="tableData"
        border
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="55" align="center" />
        <el-table-column label="字典名称" prop="name" />
        <el-table-column label="字典编码" prop="code" />
        <el-table-column label="状态" align="center" prop="status" width="80">
          <template #default="scope">
            <el-switch
              v-model="scope.row.status"
              inline-prompt
              :active-value="Status.ENABLE"
              :inactive-value="Status.DISABLE"
              active-text="启用"
              inactive-text="禁用"
              :disabled="!useUserStore().hasPerm('base:dict:status')"
              @change="handleSetStatus(scope.row)"
            />
          </template>
        </el-table-column>
        <el-table-column label="创建时间" prop="createdAt" align="center" />
        <el-table-column label="更新时间" prop="updatedAt" align="center" />
        <el-table-column fixed="right" label="操作" align="center" width="220">
          <template #default="scope">
            <el-button
              v-hasPerm="['base:dict:items']"
              type="primary"
              link
              size="small"
              :icon="List"
              @click.stop="handleOpenBaseDictItem(scope.row)"
            >
              字典数据
            </el-button>

            <el-button
              v-hasPerm="['base:dict:update']"
              type="primary"
              link
              size="small"
              icon="edit"
              @click.stop="handleOpenDialog(scope.row.id)"
            >
              编辑
            </el-button>
            <el-button
              v-hasPerm="['base:dict:delete']"
              type="danger"
              link
              size="small"
              icon="delete"
              @click.stop="handleDelete(scope.row.id)"
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

    <!--字典弹窗-->
    <el-dialog
      v-model="dialog.visible"
      :title="dialog.title"
      width="500px"
      @close="handleCloseDialog"
    >
      <el-form ref="dataFormRef" :model="formData" :rules="rules" label-width="100px">
        <el-card shadow="never">
          <el-form-item label="字典名称" prop="name">
            <el-input v-model="formData.name" placeholder="请输入字典名称" />
          </el-form-item>

          <el-form-item label="字典编码" prop="code">
            <el-input v-model="formData.code" placeholder="请输入字典编码" />
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
        </el-card>
      </el-form>

      <template #footer>
        <div class="dialog-footer">
          <el-button type="primary" @click="handleSubmitClick">确 定</el-button>
          <el-button @click="handleCloseDialog">取 消</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { useUserStore } from "@/store";
import { List } from "@element-plus/icons-vue";
defineOptions({
  name: "BaseDict",
  inheritAttrs: false,
});

import { defBaseDictService } from "@/api/admin/base_dict";
import type { BaseDict, BaseDictForm, PageBaseDictRequest } from "@/rpc/admin/base_dict";

import router from "@/router";
import { Status } from "@/rpc/common/enum";

const queryFormRef = ref(ElForm);
const dataFormRef = ref(ElForm);

const loading = ref(false);
const ids = ref<number[]>([]);
const total = ref(0);

const queryParams = reactive<PageBaseDictRequest>({
  /** 字典名称 */
  name: "",
  /** 字典编号 */
  code: "",
  /** 状态 */
  status: undefined,
  /** 当前页码 */
  pageNum: 0,
  /** 每一页的行数 */
  pageSize: 10,
});

const tableData = ref<BaseDict[]>();

const dialog = reactive({
  title: "",
  visible: false,
});

const formData = reactive<BaseDictForm>({
  /** 字典ID */
  id: 0,
  /** 字典编号 */
  code: "",
  /** 字典名称 */
  name: "",
  /** 状态 */
  status: Status.ENABLE,
});

const rules = computed(() => {
  const rules: Partial<Record<string, any>> = {
    name: [{ required: true, message: "请输入字典名称", trigger: "blur" }],
    code: [{ required: true, message: "请输入字典编码", trigger: "blur" }],
    status: [{ required: true, message: "状态不能为空", trigger: "blur" }],
  };
  return rules;
});

// 查询
function handleQuery() {
  loading.value = true;
  defBaseDictService
    .PageBaseDict(queryParams)
    .then((data) => {
      tableData.value = data.list;
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

// 行选择
function handleSelectionChange(selection: any) {
  ids.value = selection.map((item: any) => item.id);
}

/**
 * 打开弹窗
 *
 * @param dictId
 */
function handleOpenDialog(dictId?: number) {
  dialog.visible = true;
  if (dictId) {
    dialog.title = "修改字典";
    defBaseDictService
      .GetBaseDict({
        value: dictId,
      })
      .then((data) => {
        Object.assign(formData, { ...data });
      });
  } else {
    dialog.title = "新增字典";
  }
}

// 关闭弹窗
function handleCloseDialog() {
  dialog.visible = false;
  dataFormRef.value.resetFields();
  dataFormRef.value.clearValidate();

  formData.id = 0;
}

// 提交字典表单
function handleSubmitClick() {
  dataFormRef.value.validate((isValid: boolean) => {
    if (isValid) {
      const dictId = formData.id;
      loading.value = true;
      if (dictId) {
        defBaseDictService
          .UpdateBaseDict(formData)
          .then(() => {
            ElMessage.success("修改成功");
            handleCloseDialog();
            handleQuery();
          })
          .finally(() => (loading.value = false));
      } else {
        defBaseDictService
          .CreateBaseDict(formData)
          .then(() => {
            ElMessage.success("新增成功");
            handleCloseDialog();
            handleQuery();
          })
          .finally(() => (loading.value = false));
      }
    }
  });
}

// 设置字典状态
function handleSetStatus(row: BaseDict) {
  let text = row.status === Status.ENABLE ? "启用" : "禁用";
  ElMessageBox.confirm(`是否确定${text}字典为：${row.name}?`, "提示", {
    confirmButtonText: "确认",
    cancelButtonText: "取消",
    type: "warning",
  })
    .then(() => {
      defBaseDictService.SetBaseDictStatus({ id: row.id, status: row.status }).then(() => {
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

/**
 * 删除字典
 *
 * @param dictId 字典ID
 */
function handleDelete(dictId?: number) {
  const dictIds = [dictId || ids.value].join(",");
  if (!dictIds) {
    ElMessage.warning("请勾选删除项");
    return;
  }
  ElMessageBox.confirm("确认删除已选中的数据项?", "警告", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning",
  }).then(
    () => {
      defBaseDictService
        .DeleteBaseDict({
          value: dictIds,
        })
        .then(() => {
          ElMessage.success("删除成功");
          handleResetQuery();
        });
    },
    () => {
      ElMessage.info("已取消删除");
    }
  );
}

// 打开字典数据
function handleOpenBaseDictItem(row: BaseDict) {
  router.push({
    path: "/base/dict-item",
    query: { dictId: row.id, title: "【" + row.name + "】字典数据" },
  });
}

onMounted(() => {
  handleQuery();
});
</script>
