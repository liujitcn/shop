<!-- 系统配置 -->
<template>
  <div class="app-container">
    <div class="search-bar">
      <el-form ref="queryFormRef" :model="queryParams" :inline="true">
        <el-form-item label="配置位置" prop="site">
          <Dict v-model="queryParams.site" code="base_config_site" />
        </el-form-item>
        <el-form-item label="配置名称" prop="name">
          <el-input
            v-model="queryParams.name"
            placeholder="请输入配置名称"
            clearable
            @keyup.enter="handleQuery"
          />
        </el-form-item>
        <el-form-item label="配置类型" prop="type">
          <Dict v-model="queryParams.type" code="base_config_type" />
        </el-form-item>
        <el-form-item label="配置键" prop="key">
          <el-input
            v-model="queryParams.key"
            placeholder="请输入配置键"
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
          v-hasPerm="['base:config:create']"
          type="success"
          icon="plus"
          @click="handleOpenDialog()"
        >
          新增
        </el-button>
        <el-button
          v-hasPerm="['base:config:create']"
          type="danger"
          :disabled="ids.length === 0"
          icon="delete"
          @click="handleDelete()"
        >
          删除
        </el-button>
        <el-button
          v-hasPerm="['base:config:refresh']"
          color="#626aef"
          icon="RefreshLeft"
          @click="handleRefreshCache"
        >
          刷新缓存
        </el-button>
      </div>

      <el-table
        ref="dataTableRef"
        v-loading="loading"
        :data="pageData"
        highlight-current-row
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="55" align="center" />
        <el-table-column label="配置位置">
          <template #default="scope">
            <DictLabel v-model="scope.row.site" code="base_config_site" />
          </template>
        </el-table-column>
        <el-table-column label="配置名称" prop="name" />
        <el-table-column label="配置类型">
          <template #default="scope">
            <DictLabel v-model="scope.row.type" code="base_config_type" />
          </template>
        </el-table-column>
        <el-table-column label="配置键" prop="key" />
        <el-table-column label="状态" align="center" prop="status" width="80">
          <template #default="scope">
            <el-switch
              v-model="scope.row.status"
              inline-prompt
              :active-value="Status.ENABLE"
              :inactive-value="Status.DISABLE"
              active-text="启用"
              inactive-text="禁用"
              :disabled="!useUserStore().hasPerm('base:config:status')"
              @change="handleSetStatus(scope.row)"
            />
          </template>
        </el-table-column>
        <el-table-column label="创建时间" prop="createdAt" align="center" />
        <el-table-column label="更新时间" prop="updatedAt" align="center" />
        <el-table-column fixed="right" label="操作" width="150">
          <template #default="scope">
            <el-button
              v-hasPerm="['base:config:update']"
              type="primary"
              size="small"
              link
              icon="edit"
              @click="handleOpenDialog(scope.row.id)"
            >
              编辑
            </el-button>
            <el-button
              v-hasPerm="['base:config:delete']"
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

    <!-- 系统配置表单弹窗 -->
    <el-dialog
      v-model="dialog.visible"
      :title="dialog.title"
      width="1200px"
      @close="handleCloseDialog"
    >
      <el-form
        ref="dataFormRef"
        :model="formData"
        :rules="rules"
        label-suffix=":"
        label-width="100px"
      >
        <el-form-item label="配置名称" prop="name">
          <el-input
            v-model="formData.name"
            placeholder="请输入配置名称"
            :disabled="formData.id > 0"
          />
        </el-form-item>
        <el-form-item label="配置位置" prop="site">
          <Dict v-model="formData.site" code="base_config_site" :disabled="formData.id > 0" />
        </el-form-item>
        <el-form-item label="配置键" prop="key">
          <el-input
            v-model="formData.key"
            placeholder="请输入配置键"
            :disabled="formData.id > 0"
          />
        </el-form-item>
        <el-form-item label="配置类型" prop="type">
          <Dict v-model="formData.type" code="base_config_type" :disabled="formData.id > 0" />
        </el-form-item>
        <el-form-item v-if="formData.type == BaseConfigType.TEXT" label="配置值" prop="value">
          <el-input v-model="formData.value" placeholder="请输入配置值" />
        </el-form-item>
        <el-form-item v-if="formData.type == BaseConfigType.IMAGE" label="配置值" prop="value">
          <single-image-upload v-model="formData.value" file-type="config" />
        </el-form-item>
        <el-form-item v-if="formData.type == BaseConfigType.RICH_TEXT" label="配置值" prop="value">
          <wang-editor v-model="formData.value" file-type="config" />
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
          <el-button type="primary" @click="handleSubmit">确定</el-button>
          <el-button @click="handleCloseDialog">取消</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { useUserStore } from "@/store";
import { defBaseConfigService } from "@/api/admin/base_config";
import { BaseConfig, BaseConfigForm, PageBaseConfigRequest } from "@/rpc/admin/base_config";
import { BaseConfigType, Status } from "@/rpc/common/enum";

defineOptions({
  name: "BaseConfig",
  inheritAttrs: false,
});

const queryFormRef = ref();
const dataFormRef = ref();

const loading = ref(false);
const ids = ref<number[]>([]);
const total = ref(0);

const queryParams = reactive<PageBaseConfigRequest>({
  /** 位置：枚举【BaseConfigSite】 */
  site: undefined,
  /** 配置名称 */
  name: undefined,
  /** 配置类型：枚举【BaseConfigType】 */
  type: undefined,
  /** 系统配置key */
  key: undefined,
  /** 状态 */
  status: undefined,
  /** 当前页码 */
  pageNum: 0,
  /** 每一页的行数 */
  pageSize: 10,
});

// 系统配置表格数据
const pageData = ref<BaseConfig[]>([]);

const dialog = reactive({
  title: "",
  visible: false,
});

const formData = reactive<BaseConfigForm>({
  /** 配置ID */
  id: 0,
  /** 位置：枚举【BaseConfigSite】 */
  site: undefined,
  /** 配置名称 */
  name: "",
  /** 配置类型：枚举【BaseConfigType】 */
  type: undefined,
  /** 配置key */
  key: "",
  /** 配置value */
  value: "",
  /** 状态 */
  status: Status.ENABLE,
});

const rules = reactive({
  site: [{ required: true, message: "请选择系统配置位置", trigger: "blur" }],
  name: [{ required: true, message: "请输入系统配置名称", trigger: "blur" }],
  type: [{ required: true, message: "请选择系统配置类型", trigger: "blur" }],
  key: [{ required: true, message: "请输入系统配置编码", trigger: "blur" }],
  value: [{ required: true, message: "配置值不能为空", trigger: "blur" }],
  status: [{ required: true, message: "请选择状态", trigger: "blur" }],
});

// 查询系统配置
function handleQuery() {
  loading.value = true;
  defBaseConfigService
    .PageBaseConfig(queryParams)
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
  handleQuery();
}

// 行复选框选中项变化
function handleSelectionChange(selection: any) {
  ids.value = selection.map((item: any) => item.id);
}

// 打开系统配置弹窗
function handleOpenDialog(configId?: number) {
  dialog.visible = true;
  if (configId) {
    dialog.title = "修改系统配置";
    defBaseConfigService
      .GetBaseConfig({
        value: configId,
      })
      .then((data) => {
        Object.assign(formData, data);
      });
  } else {
    dialog.title = "新增系统配置";
    formData.id = 0;
  }
}

// 刷新缓存(防抖)
const handleRefreshCache = useDebounceFn(() => {
  defBaseConfigService.RefreshBaseConfig({}).then(() => {
    ElMessage.success("刷新成功");
  });
}, 1000);

// 系统配置表单提交
function handleSubmit() {
  dataFormRef.value.validate((valid: any) => {
    if (valid) {
      loading.value = true;
      const id = formData.id;
      if (id) {
        defBaseConfigService
          .UpdateBaseConfig(formData)
          .then(() => {
            ElMessage.success("修改成功");
            handleCloseDialog();
            handleResetQuery();
          })
          .finally(() => (loading.value = false));
      } else {
        defBaseConfigService
          .CreateBaseConfig(formData)
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

// 重置表单
function resetForm() {
  dataFormRef.value.resetFields();
  dataFormRef.value.clearValidate();
  formData.id = 0;
  formData.value = "";
}

// 关闭系统配置弹窗
function handleCloseDialog() {
  dialog.visible = false;
  resetForm();
}

// 设置配置状态
function handleSetStatus(row: BaseConfig) {
  let text = row.status === Status.ENABLE ? "启用" : "禁用";
  ElMessageBox.confirm(`是否确定${text}配置为：${row.name}?`, "提示", {
    confirmButtonText: "确认",
    cancelButtonText: "取消",
    type: "warning",
  })
    .then(() => {
      defBaseConfigService.SetBaseConfigStatus({ id: row.id, status: row.status }).then(() => {
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
 * 删除系统配置
 *
 * @param configId 配置ID
 */
function handleDelete(configId?: number) {
  const dictIds = [configId || ids.value].join(",");
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
      defBaseConfigService
        .DeleteBaseConfig({
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

onMounted(() => {
  handleQuery();
});
</script>
