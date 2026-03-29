<template>
  <div class="table-box">
    <ProTable
      ref="proTable"
      row-key="id"
      :columns="columns"
      :request-api="requestGoodsCategoryTable"
      :pagination="false"
      :default-expand-all="false"
      :indent="20"
      :tree-props="{ children: 'children', hasChildren: 'hasChildren' }"
    >
      <template #tableHeader="{ selectedList }">
        <el-button v-hasPerm="['goods:category:create']" type="success" :icon="CirclePlus" @click="handleOpenDialog(0)">
          新增
        </el-button>
        <el-button
          v-hasPerm="['goods:category:delete']"
          type="danger"
          :icon="Delete"
          :disabled="!selectedList.length"
          @click="handleDelete(selectedList)"
        >
          删除
        </el-button>
      </template>

      <template #picture="scope">
        <el-popover placement="right" :width="400" trigger="hover">
          <img :src="scope.row.picture" width="400" height="400" />
          <template #reference>
            <img :src="scope.row.picture" style="max-width: 60px; max-height: 60px" />
          </template>
        </el-popover>
      </template>

      <template #status="scope">
        <el-switch
          v-model="scope.row.status"
          inline-prompt
          :active-value="Status.ENABLE"
          :inactive-value="Status.DISABLE"
          active-text="启用"
          inactive-text="禁用"
          :disabled="!BUTTONS['goods:category:status']"
          :before-change="() => handleBeforeSetStatus(scope.row)"
        />
      </template>

      <template #operation="scope">
        <el-button
          v-if="!scope.row.parentId"
          v-hasPerm="['goods:category:create']"
          type="primary"
          link
          :icon="CirclePlus"
          @click.stop="handleOpenDialog(scope.row.id)"
        >
          新增
        </el-button>
        <el-button
          v-hasPerm="['goods:category:update']"
          type="primary"
          link
          :icon="EditPen"
          @click.stop="handleOpenDialog(scope.row.parentId, scope.row.id)"
        >
          编辑
        </el-button>
        <el-button v-hasPerm="['goods:category:delete']" type="danger" link :icon="Delete" @click.stop="handleDelete(scope.row)">
          删除
        </el-button>
      </template>
    </ProTable>

    <el-dialog v-model="dialog.visible" :title="dialog.title" width="600px" @closed="handleCloseDialog">
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
          <el-input v-model="formData.name" placeholder="请输入分类名称" />
        </el-form-item>
        <el-form-item label="照片" prop="picture">
          <UploadImg v-model:image-url="formData.picture" />
        </el-form-item>
        <el-form-item label="排序" prop="sort">
          <el-input-number v-model="formData.sort" controls-position="right" :min="1" :precision="0" :step="1" />
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
import { reactive, ref } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { CirclePlus, Delete, EditPen } from "@element-plus/icons-vue";
import type { ColumnProps, ProTableInstance } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
import UploadImg from "@/components/Upload/Img.vue";
import { useAuthButtons } from "@/hooks/useAuthButtons";
import { defGoodsCategoryService } from "@/api/admin/goods_category";
import type { GoodsCategory, GoodsCategoryForm } from "@/rpc/admin/goods_category";
import type { TreeOptionResponse_Option } from "@/rpc/common/common";
import { Status } from "@/rpc/common/enum";
import { normalizeSelectedIds } from "@/utils/proTable";

defineOptions({
  name: "GoodsCategory",
  inheritAttrs: false
});

const { BUTTONS } = useAuthButtons();
const proTable = ref<ProTableInstance>();
const dataFormRef = ref();

const dialog = reactive({
  title: "",
  visible: false
});

const categoryOptions = ref<TreeOptionResponse_Option[]>([]);

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
  status: Status.ENABLE
});

const rules = reactive({
  parentId: [{ required: true, message: "上级分类不能为空", trigger: "change" }],
  name: [{ required: true, message: "分类名称不能为空", trigger: "blur" }],
  sort: [{ required: true, message: "排序不能为空", trigger: "blur" }],
  status: [{ required: true, message: "状态不能为空", trigger: "blur" }]
});

/** 分类表格列配置。 */
const columns: ColumnProps[] = [
  { type: "selection", width: 55 },
  { prop: "name", label: "分类名称", align: "left", search: { el: "input" } },
  { prop: "picture", label: "图片", minWidth: 150 },
  { prop: "sort", label: "排序", align: "right" },
  { prop: "status", label: "状态", width: 100, dictCode: "status", search: { el: "select" } },
  { prop: "createdAt", label: "创建时间", width: 180 },
  { prop: "updatedAt", label: "更新时间", width: 180 },
  { prop: "operation", label: "操作", width: 200, fixed: "right", align: "left" }
];

/**
 * 按搜索条件递归过滤分类树，命中父节点或子节点时保留当前节点。
 */
function filterCategoryTree(categoryList: GoodsCategory[], keywordMap: { name: string; status: string }) {
  const nameKeyword = keywordMap.name.trim().toLowerCase();
  const statusKeyword = keywordMap.status.trim();

  return categoryList.reduce<GoodsCategory[]>((result, item) => {
    const children = filterCategoryTree(item.children ?? [], keywordMap);
    const name = item.name?.toLowerCase() ?? "";
    const status = String(item.status ?? "");
    const matched = (!nameKeyword || name.includes(nameKeyword)) && (!statusKeyword || status === statusKeyword);

    if (!matched && !children.length) return result;

    result.push({
      ...item,
      children
    });
    return result;
  }, []);
}

/**
 * 请求分类树数据，并按搜索条件过滤树形结构。
 */
async function requestGoodsCategoryTable(params: Record<string, string>) {
  const data = await defGoodsCategoryService.TreeGoodsCategory({});
  const keywordMap = {
    name: params.name ?? "",
    status: String(params.status ?? "")
  };
  return { data: filterCategoryTree(data.list ?? [], keywordMap) };
}

/**
 * 刷新分类树表格。
 */
function refreshTable() {
  proTable.value?.getTableList();
}

/**
 * 加载分类下拉树数据，供弹窗选择上级分类。
 */
async function loadCategoryOptions() {
  const optionGoodsCategoryResponse = await defGoodsCategoryService.OptionGoodsCategory({ parentId: 0 });
  categoryOptions.value = [
    {
      value: 0,
      label: "顶级分类",
      disabled: false,
      children: optionGoodsCategoryResponse.list
    }
  ];
}

/**
 * 打开分类弹窗。
 */
async function handleOpenDialog(parentId?: number, categoryId?: number) {
  await loadCategoryOptions();
  dialog.visible = true;
  if (categoryId) {
    dialog.title = "修改分类";
    defGoodsCategoryService.GetGoodsCategory({ value: categoryId }).then(data => {
      Object.assign(formData, data);
    });
    return;
  }

  dialog.title = "新增分类";
  resetForm();
  formData.parentId = parentId;
}

/**
 * 提交分类表单。
 */
function handleSubmit() {
  dataFormRef.value?.validate((valid: boolean) => {
    if (!valid) return;

    const request = formData.id
      ? defGoodsCategoryService.UpdateGoodsCategory(formData)
      : defGoodsCategoryService.CreateGoodsCategory(formData);
    request.then(() => {
      ElMessage.success(formData.id ? "修改成功" : "新增成功");
      handleCloseDialog();
      refreshTable();
    });
  });
}

/**
 * 在分类状态切换前先完成确认与接口调用，避免首屏渲染触发误操作。
 */
async function handleBeforeSetStatus(row: GoodsCategory) {
  const nextStatus = row.status === Status.ENABLE ? Status.DISABLE : Status.ENABLE;
  const text = nextStatus === Status.ENABLE ? "启用" : "禁用";
  const categoryName = row.name || `ID:${row.id}`;
  try {
    await ElMessageBox.confirm(`是否确定${text}分类：${categoryName}？`, "提示", {
      confirmButtonText: "确认",
      cancelButtonText: "取消",
      type: "warning"
    });
    await defGoodsCategoryService.SetGoodsCategoryStatus({ id: row.id, status: nextStatus });
    ElMessage.success(`${text}成功`);
    refreshTable();
    return true;
  } catch {
    return false;
  }
}

/**
 * 删除分类，兼容单条删除与批量删除。
 */
function handleDelete(selected?: number | string | Array<number | string> | GoodsCategory | GoodsCategory[]) {
  const categoryList = Array.isArray(selected)
    ? (selected.filter(item => typeof item === "object") as GoodsCategory[])
    : selected && typeof selected === "object"
      ? [selected as GoodsCategory]
      : [];
  const categoryIds = (
    categoryList.length
      ? categoryList.map(item => item.id)
      : normalizeSelectedIds(selected as number | string | Array<number | string>)
  ).join(",");
  if (!categoryIds) {
    ElMessage.warning("请勾选删除项");
    return;
  }

  const confirmMessage = categoryList.length
    ? categoryList.length === 1
      ? `是否确定删除分类：${categoryList[0].name || `ID:${categoryList[0].id}`}？`
      : `确认删除已选中的 ${categoryList.length} 个商品分类吗？`
    : "确认删除已选中的商品分类吗？";

  ElMessageBox.confirm(confirmMessage, "警告", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning"
  }).then(
    () => {
      defGoodsCategoryService.DeleteGoodsCategory({ value: categoryIds }).then(() => {
        ElMessage.success("删除成功");
        refreshTable();
      });
    },
    () => {
      ElMessage.info("已取消删除");
    }
  );
}

/**
 * 重置分类表单，避免上次编辑残留到下一次新增。
 */
function resetForm() {
  dataFormRef.value?.resetFields();
  dataFormRef.value?.clearValidate();
  formData.id = 0;
  formData.parentId = undefined;
  formData.name = "";
  formData.picture = "";
  formData.sort = 1;
  formData.status = Status.ENABLE;
}

/**
 * 关闭分类弹窗并恢复表单默认值。
 */
function handleCloseDialog() {
  dialog.visible = false;
  resetForm();
}
</script>
