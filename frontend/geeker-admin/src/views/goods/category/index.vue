<template>
  <div class="table-box">
    <ProTable
      ref="proTable"
      row-key="id"
      :columns="columns"
      :header-actions="headerActions"
      :request-api="requestGoodsCategoryTable"
      :pagination="false"
      :default-expand-all="false"
      :indent="20"
      :tree-props="{ children: 'children', hasChildren: 'hasChildren' }"
    />

    <FormDialog
      v-model="dialog.visible"
      ref="formDialogRef"
      :title="dialog.title"
      width="600px"
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
import { ElMessage, ElMessageBox } from "element-plus";
import { CirclePlus, Delete, EditPen } from "@element-plus/icons-vue";
import type { ColumnProps, HeaderActionProps, ProTableInstance } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
import FormDialog from "@/components/Dialog/FormDialog.vue";
import type { ProFormField, ProFormOption } from "@/components/ProForm/interface";
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
const formDialogRef = ref<InstanceType<typeof FormDialog>>();

const dialog = reactive({
  title: "",
  visible: false
});

const categoryOptions = ref<TreeOptionResponse_Option[]>([]);
const statusOptions: ProFormOption[] = [
  { label: "启用", value: Status.ENABLE },
  { label: "禁用", value: Status.DISABLE }
];

const formData = reactive<GoodsCategoryForm>({
  /** 分类ID */
  id: 0,
  /** 父节点ID */
  parentId: 0,
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
  status: [{ required: true, message: "状态不能为空", trigger: "change" }]
});

/** 分类表单字段配置。 */
const formFields = computed<ProFormField[]>(() => [
  {
    prop: "parentId",
    label: "上级分类",
    component: "tree-select",
    options: categoryOptions.value,
    props: {
      placeholder: "选择上级分类",
      filterable: true,
      checkStrictly: true,
      renderAfterExpand: false,
      style: { width: "100%" }
    }
  },
  { prop: "name", label: "分类名称", component: "input", props: { placeholder: "请输入分类名称" } },
  { prop: "picture", label: "照片", component: "image-upload" },
  {
    prop: "sort",
    label: "排序",
    component: "input-number",
    props: { min: 1, precision: 0, step: 1, controlsPosition: "right", style: { width: "100%" } }
  },
  { prop: "status", label: "状态", component: "radio-group", options: statusOptions }
]);

/** 分类表格列配置。 */
const columns: ColumnProps[] = [
  { type: "selection", width: 55 },
  { prop: "name", label: "分类名称", align: "left", search: { el: "input" } },
  {
    prop: "picture",
    label: "图片",
    minWidth: 150,
    cellType: "image",
    imageProps: {
      previewWidth: 400,
      previewHeight: 400,
      width: 60,
      height: 60
    }
  },
  { prop: "sort", label: "排序", align: "right" },
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
      disabled: () => !BUTTONS.value["goods:category:status"],
      beforeChange: scope => handleBeforeSetStatus(scope.row as GoodsCategory)
    }
  },
  { prop: "createdAt", label: "创建时间", width: 180 },
  { prop: "updatedAt", label: "更新时间", width: 180 },
  {
    prop: "operation",
    label: "操作",
    width: 200,
    fixed: "right",
    align: "left",
    cellType: "actions",
    actions: [
      {
        label: "新增",
        type: "primary",
        link: true,
        icon: CirclePlus,
        hidden: scope => !BUTTONS.value["goods:category:create"] || Boolean((scope.row as GoodsCategory).parentId),
        params: scope => ({ parentId: scope.row.id }),
        onClick: (_, params) => handleOpenDialog((params?.parentId as number | undefined) ?? 0)
      },
      {
        label: "编辑",
        type: "primary",
        link: true,
        icon: EditPen,
        hidden: () => !BUTTONS.value["goods:category:update"],
        params: scope => ({
          parentId: scope.row.parentId,
          categoryId: scope.row.id
        }),
        onClick: (_, params) => handleOpenDialog(params?.parentId as number | undefined, params?.categoryId as number | undefined)
      },
      {
        label: "删除",
        type: "danger",
        link: true,
        icon: Delete,
        hidden: () => !BUTTONS.value["goods:category:delete"],
        onClick: scope => handleDelete(scope.row as GoodsCategory)
      }
    ]
  }
];

/** 分类顶部按钮配置。 */
const headerActions: HeaderActionProps[] = [
  {
    label: "新增",
    type: "success",
    icon: CirclePlus,
    hidden: () => !BUTTONS.value["goods:category:create"],
    onClick: () => handleOpenDialog(0)
  },
  {
    label: "删除",
    type: "danger",
    icon: Delete,
    hidden: () => !BUTTONS.value["goods:category:delete"],
    disabled: scope => !scope.selectedList.length,
    onClick: scope => handleDelete(scope.selectedList as GoodsCategory[])
  }
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
 * 重置分类表单，避免上次编辑残留到下一次新增。
 */
function resetForm() {
  formDialogRef.value?.resetFields();
  formDialogRef.value?.clearValidate();
  formData.id = 0;
  formData.parentId = 0;
  formData.name = "";
  formData.picture = "";
  formData.sort = 1;
  formData.status = Status.ENABLE;
}

/**
 * 打开分类弹窗。
 */
async function handleOpenDialog(parentId?: number, categoryId?: number) {
  resetForm();
  await loadCategoryOptions();
  dialog.title = categoryId ? "修改分类" : "新增分类";
  dialog.visible = true;

  if (categoryId) {
    defGoodsCategoryService.GetGoodsCategory({ value: categoryId }).then(data => {
      Object.assign(formData, data);
    });
    return;
  }

  formData.parentId = parentId ?? 0;
}

/**
 * 提交分类表单。
 */
function handleSubmit() {
  formDialogRef.value?.validate()?.then(valid => {
    if (!valid) return;

    const submitData = JSON.parse(JSON.stringify(formData)) as GoodsCategoryForm;
    const request = submitData.id
      ? defGoodsCategoryService.UpdateGoodsCategory(submitData)
      : defGoodsCategoryService.CreateGoodsCategory(submitData);
    request.then(() => {
      ElMessage.success(submitData.id ? "修改成功" : "新增成功");
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
 * 关闭分类弹窗并恢复表单默认值。
 */
function handleCloseDialog() {
  dialog.visible = false;
  resetForm();
}
</script>
