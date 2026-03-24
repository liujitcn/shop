<template>
  <div class="app-container">
    <div class="search-bar">
      <el-form ref="queryFormRef" :model="queryParams" :inline="true">
        <el-form-item>
          <el-button class="filter-item" type="primary" icon="search" @click="handleQuery">
            搜索
          </el-button>
          <el-button icon="refresh" @click="handleResetQuery">重置</el-button>
        </el-form-item>
      </el-form>
    </div>

    <el-card shadow="never">
      <div class="mb-10px">
        <el-button
          v-hasPerm="['goods:category:create']"
          type="success"
          icon="plus"
          @click="handleOpenDialog(0)"
        >
          新增
        </el-button>
        <el-button
          v-hasPerm="['goods:category:delete']"
          type="danger"
          :disabled="selectIds.length === 0"
          icon="delete"
          @click="handleDelete()"
        >
          删除
        </el-button>
      </div>

      <el-table
        v-loading="loading"
        :data="categoryList"
        row-key="id"
        default-expand-all
        :tree-props="{ children: 'children', hasChildren: 'hasChildren' }"
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="55" align="center" />
        <el-table-column prop="name" label="分类名称" />
        <el-table-column label="图片" min-width="150" align="center">
          <template #default="scope">
            <el-popover placement="right" :width="400" trigger="hover">
              <img :src="scope.row.picture" width="400" height="400" />
              <template #reference>
                <img :src="scope.row.picture" style="max-width: 60px; max-height: 60px" />
              </template>
            </el-popover>
          </template>
        </el-table-column>
        <el-table-column prop="sort" label="排序" align="right" />
        <el-table-column label="状态" align="center" prop="status" width="80">
          <template #default="scope">
            <el-switch
              v-model="scope.row.status"
              inline-prompt
              :active-value="Status.ENABLE"
              :inactive-value="Status.DISABLE"
              active-text="启用"
              inactive-text="禁用"
              :disabled="!useUserStore().hasPerm('goods:category:status')"
              @change="handleSetStatus(scope.row)"
            />
          </template>
        </el-table-column>
        <el-table-column label="创建时间" prop="createdAt" align="center" />
        <el-table-column label="更新时间" prop="updatedAt" align="center" />
        <el-table-column label="操作" fixed="right" align="left" width="200">
          <template #default="scope">
            <el-button
              v-if="!scope.row.parentId"
              v-hasPerm="['goods:category:create']"
              type="primary"
              link
              size="small"
              icon="plus"
              @click.stop="handleOpenDialog(scope.row.id, undefined)"
            >
              新增
            </el-button>
            <el-button
              v-hasPerm="['goods:category:update']"
              type="primary"
              link
              size="small"
              icon="edit"
              @click.stop="handleOpenDialog(scope.row.parentId, scope.row.id)"
            >
              编辑
            </el-button>
            <el-button
              v-hasPerm="['goods:category:delete']"
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
    </el-card>

    <el-dialog
      v-model="dialog.visible"
      :title="dialog.title"
      width="600px"
      @closed="handleCloseDialog"
    >
      <el-form ref="dataFormRef" :model="formData" :rules="rules" label-width="80px">
        <el-form-item label="上级分类" prop="parentId">
          <el-tree-select
            v-model="formData.parentId"
            placeholder="选择上级分类"
            :data="categoryOptions"
            filterable
            check-strictly
            :render-after-expand="false"
          />
        </el-form-item>
        <el-form-item label="分类名称" prop="name">
          <el-input
            v-model="formData.name"
            placeholder="请输入分类名称"
          />
        </el-form-item>
        <el-form-item label="照片" prop="picture">
          <single-image-upload v-model="formData.picture" file-type="category" />
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
          <el-button type="primary" @click="handleSubmit">确 定</el-button>
          <el-button @click="handleCloseDialog">取 消</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { TreeOptionResponse_Option } from "@/rpc/common/common";

defineOptions({
  name: "GoodsCategory",
  inheritAttrs: false,
});

import { defGoodsCategoryService } from "@/api/admin/goods_category";
import { GoodsCategory, GoodsCategoryForm } from "@/rpc/admin/goods_category";
import { Empty } from "@/rpc/google/protobuf/empty";
import { useUserStore } from "@/store";
import { Status } from "@/rpc/common/enum";

const queryFormRef = ref(ElForm);
const dataFormRef = ref(ElForm);

const loading = ref(false);
const selectIds = ref<number[]>([]);
const queryParams = reactive<Empty>({});

const dialog = reactive({
  title: "",
  visible: false,
});

const categoryList = ref<GoodsCategory[]>();
const categoryOptions = ref<TreeOptionResponse_Option[]>();
const formData = reactive<GoodsCategoryForm>({
  /** 分类ID */
  id: 0,
  /** 父节点ID */
  parentId: undefined,
  /** 分类名称 */
  name: "",
  /** 分类图片 */
  picture: "",
  /** 排序 */
  sort: 1,
  /** 菜单状态 */
  status: Status.ENABLE,
});

const rules = reactive({
  parentId: [{ required: true, message: "上级分类不能为空", trigger: "change" }],
  name: [{ required: true, message: "分类名称不能为空", trigger: "blur" }],
  sort: [{ required: true, message: "排序不能为空", trigger: "blur" }],
  status: [{ required: true, message: "状态不能为空", trigger: "blur" }],
});

// 查询分类
function handleQuery() {
  loading.value = true;
  defGoodsCategoryService.TreeGoodsCategory(queryParams).then((data) => {
    categoryList.value = data.list;
    loading.value = false;
  });
}

// 重置查询
function handleResetQuery() {
  queryFormRef.value.resetFields();
  handleQuery();
}

// 处理选中项变化
function handleSelectionChange(selection: any) {
  selectIds.value = selection.map((item: any) => item.id);
}

/**
 * 打开分类弹窗
 *
 * @param parentId 父分类ID
 * @param categoryId 分类ID
 */
async function handleOpenDialog(parentId?: number, categoryId?: number) {
  // 加载分类下拉数据
  const optionGoodsCategoryResponse = await defGoodsCategoryService.OptionGoodsCategory({
    parentId: 0,
  });
  categoryOptions.value = [
    {
      value: 0,
      label: "顶级分类",
      disabled: false,
      children: optionGoodsCategoryResponse.list,
    },
  ];

  dialog.visible = true;
  if (categoryId) {
    dialog.title = "修改分类";
    defGoodsCategoryService
      .GetGoodsCategory({
        value: categoryId,
      })
      .then((data) => {
        Object.assign(formData, data);
      });
  } else {
    dialog.title = "新增分类";
    formData.parentId = parentId;
  }
}

// 提交分类表单
function handleSubmit() {
  dataFormRef.value.validate((valid: any) => {
    if (valid) {
      loading.value = true;
      const categoryId = formData.id;
      if (categoryId) {
        defGoodsCategoryService
          .UpdateGoodsCategory(formData)
          .then(() => {
            ElMessage.success("修改成功");
            handleCloseDialog();
            handleQuery();
          })
          .finally(() => (loading.value = false));
      } else {
        defGoodsCategoryService
          .CreateGoodsCategory(formData)
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
function handleSetStatus(row: GoodsCategory) {
  let text = row.status === Status.ENABLE ? "启用" : "禁用";
  ElMessageBox.confirm(`是否确定${text}分类为：${row.name}?`, "提示", {
    confirmButtonText: "确认",
    cancelButtonText: "取消",
    type: "warning",
  })
    .then(() => {
      defGoodsCategoryService
        .SetGoodsCategoryStatus({ id: row.id, status: row.status })
        .then(() => {
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

// 删除分类
function handleDelete(categoryId?: number) {
  const categoryIds = [categoryId || selectIds.value].join(",");

  if (!categoryIds) {
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
      defGoodsCategoryService
        .DeleteGoodsCategory({
          value: categoryIds,
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

// 重置表单
function resetForm() {
  dataFormRef.value.resetFields();
  dataFormRef.value.clearValidate();

  formData.id = 0;
}

// 关闭弹窗
function handleCloseDialog() {
  dialog.visible = false;
  resetForm();
}

onMounted(() => {
  handleQuery();
});
</script>
