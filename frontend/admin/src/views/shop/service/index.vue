<!-- 商城服务 -->
<template>
  <div class="app-container">
    <div class="search-bar">
      <el-form ref="queryFormRef" :model="queryParams" :inline="true">
        <el-form-item label="标签" prop="label">
          <el-input v-model="queryParams.label" placeholder="请输入标签" clearable />
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
          v-hasPerm="['shop:service:create']"
          type="success"
          icon="plus"
          @click="handleOpenDialog()"
        >
          新增
        </el-button>
        <el-button
          v-hasPerm="['shop:service:create']"
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
        :data="pageData"
        highlight-current-row
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="55" align="center" />
        <el-table-column label="标签" prop="label" />
        <el-table-column label="值" prop="value" />
        <el-table-column label="排序" prop="sort" align="right" />
        <el-table-column label="状态" align="center" prop="status" width="80">
          <template #default="scope">
            <el-switch
              v-model="scope.row.status"
              inline-prompt
              :active-value="Status.ENABLE"
              :inactive-value="Status.DISABLE"
              active-text="启用"
              inactive-text="禁用"
              :disabled="!useUserStore().hasPerm('shop:service:status')"
              @change="handleSetStatus(scope.row)"
            />
          </template>
        </el-table-column>
        <el-table-column label="创建时间" prop="createdAt" align="center" />
        <el-table-column label="更新时间" prop="updatedAt" align="center" />
        <el-table-column fixed="right" label="操作" width="150">
          <template #default="scope">
            <el-button
              v-hasPerm="['shop:service:update']"
              type="primary"
              size="small"
              link
              icon="edit"
              @click="handleOpenDialog(scope.row.id)"
            >
              编辑
            </el-button>
            <el-button
              v-hasPerm="['shop:service:delete']"
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

    <!-- 商城服务表单弹窗 -->
    <el-dialog
      v-model="dialog.visible"
      :title="dialog.title"
      width="500px"
      @close="handleCloseDialog"
    >
      <el-form
        ref="dataFormRef"
        :model="formData"
        :rules="rules"
        label-suffix=":"
        label-width="100px"
      >
        <el-form-item label="标签" prop="label">
          <el-input v-model="formData.label" placeholder="请输入标签" />
        </el-form-item>
        <el-form-item label="值" prop="value">
          <el-input
            v-model="formData.value"
            placeholder="请输入值"
            type="textarea"
          />
        </el-form-item>
        <el-form-item label="排序" prop="sort">
          <el-input-number
            v-model="formData.sort"
            controls-position="right"
            :min="1"
            :precision="0"
            :step="1"
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
          <el-button type="primary" @click="handleSubmit">确定</el-button>
          <el-button @click="handleCloseDialog">取消</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { useUserStore } from "@/store";
import { defShopServiceService } from "@/api/admin/shop_service";
import { PageShopServiceRequest, ShopService, ShopServiceForm } from "@/rpc/admin/shop_service";
import { Status } from "@/rpc/common/enum";

defineOptions({
  name: "ShopService",
  inheritAttrs: false,
});

const queryFormRef = ref();
const dataFormRef = ref();

const loading = ref(false);
const ids = ref<number[]>([]);
const total = ref(0);

const queryParams = reactive<PageShopServiceRequest>({
  /** 标签 */
  label: "",
  /** 状态 */
  status: undefined,
  /** 当前页码 */
  pageNum: 0,
  /** 每一页的行数 */
  pageSize: 10,
});

// 商城服务表格数据
const pageData = ref<ShopService[]>([]);

const dialog = reactive({
  title: "",
  visible: false,
});

const formData = reactive<ShopServiceForm>({
  /** 商城服务ID */
  id: 0,
  /** 标签 */
  label: "",
  /** 值 */
  value: "",
  /** 排序 */
  sort: 1,
  /** 状态 */
  status: Status.ENABLE,
});

const rules = reactive({
  label: [{ required: true, message: "标签不能为空", trigger: "blur" }],
  value: [{ required: true, message: "值不能为空", trigger: "blur" }],
});

// 查询商城服务
function handleQuery() {
  loading.value = true;
  defShopServiceService
    .PageShopService(queryParams)
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

// 打开商城服务弹窗
function handleOpenDialog(serviceId?: number) {
  dialog.visible = true;
  if (serviceId) {
    dialog.title = "修改商城服务";
    defShopServiceService
      .GetShopService({
        value: serviceId,
      })
      .then((data) => {
        Object.assign(formData, data);
      });
  } else {
    dialog.title = "新增商城服务";
    formData.id = 0;
  }
}

// 商城服务表单提交
function handleSubmit() {
  dataFormRef.value.validate((valid: any) => {
    if (valid) {
      loading.value = true;
      const submitsData = JSON.parse(JSON.stringify(formData));
      const id = submitsData.id;
      if (id) {
        defShopServiceService
          .UpdateShopService(submitsData)
          .then(() => {
            ElMessage.success("修改成功");
            handleCloseDialog();
            handleResetQuery();
          })
          .finally(() => (loading.value = false));
      } else {
        defShopServiceService
          .CreateShopService(submitsData)
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

// 关闭商城服务弹窗
function handleCloseDialog() {
  dialog.visible = false;
  resetForm();
}

// 设置商城服务状态
function handleSetStatus(row: ShopService) {
  let text = row.status === Status.ENABLE ? "启用" : "禁用";
  ElMessageBox.confirm(`是否确定${text}商城服务?`, "提示", {
    confirmButtonText: "确认",
    cancelButtonText: "取消",
    type: "warning",
  })
    .then(() => {
      defShopServiceService.SetShopServiceStatus({ id: row.id, status: row.status }).then(() => {
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
 * 删除商城服务
 *
 * @param serviceId 商城服务ID
 */
function handleDelete(serviceId?: number) {
  const dictIds = [serviceId || ids.value].join(",");
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
      defShopServiceService
        .DeleteShopService({
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

<style scoped>
.search-bar {
  margin-bottom: 20px;
}
.table-wrapper {
  margin-bottom: 20px;
}
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.right {
  display: flex;
  gap: 10px;
}
</style>
