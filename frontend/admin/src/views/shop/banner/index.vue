<!-- 轮播图 -->
<template>
  <div class="table-box">
    <ProTable
      ref="proTable"
      row-key="id"
      :columns="columns"
      :header-actions="headerActions"
      :request-api="requestShopBannerTable"
      :init-param="initParam"
    />

    <FormDialog
      v-model="dialog.visible"
      ref="formDialogRef"
      :title="dialog.title"
      width="500px"
      :model="formData"
      :fields="formFields"
      :rules="rules"
      label-width="100px"
      @confirm="handleSubmit"
      @close="handleCloseDialog"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from "vue";
import { useRoute } from "vue-router";
import type { ColumnProps, HeaderActionProps, ProTableInstance } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
import FormDialog from "@/components/Dialog/FormDialog.vue";
import type { ProFormField, ProFormOption } from "@/components/ProForm/interface";
import { useAuthButtons } from "@/hooks/useAuthButtons";
import { defShopBannerService } from "@/api/admin/shop_banner";
import type { PageShopBannerRequest, ShopBanner, ShopBannerForm } from "@/rpc/admin/shop_banner";
import type { ListGoodsInfoResponse_GoodsInfo } from "@/rpc/admin/goods_info";
import { defGoodsInfoService } from "@/api/admin/goods_info";
import { ShopBannerType, Status } from "@/rpc/common/enum";
import type { TreeOptionResponse_Option } from "@/rpc/common/common";
import { defGoodsCategoryService } from "@/api/admin/goods_category";
import { buildPageRequest, normalizeSelectedIds } from "@/utils/proTable";

defineOptions({
  name: "ShopBanner",
  inheritAttrs: false
});

interface CategoryOption {
  /** 选项名 */
  label: string;
  /** 选项值 */
  value: string;
  /** 是否禁用 */
  disabled: boolean;
  /** 子节点树 */
  children: CategoryOption[];
}

const { BUTTONS } = useAuthButtons();
const proTable = ref<ProTableInstance>();
const formDialogRef = ref<InstanceType<typeof FormDialog>>();
const route = useRoute();

const goodsList = ref<ListGoodsInfoResponse_GoodsInfo[]>([]);
const goodsCategoryOptions = ref<CategoryOption[]>([]);

const initParam = computed<PageShopBannerRequest>(() => {
  const status = Number(route.query.status ?? 0);
  return {
    site: undefined,
    type: undefined,
    status: status > 0 ? status : undefined,
    pageNum: 1,
    pageSize: 10
  };
});

watch(
  () => [route.query.status, proTable.value],
  () => {
    if (!proTable.value) return;
    const status = Number(route.query.status ?? 0);
    Object.assign(proTable.value.searchParam, {
      status: status > 0 ? status : undefined
    });
    Object.assign(proTable.value.searchInitParam, {
      status: status > 0 ? status : undefined
    });
    proTable.value.pageable.pageNum = 1;
    proTable.value.search();
  },
  { immediate: true }
);

const dialog = reactive({
  title: "",
  visible: false
});

const formData = reactive<ShopBannerForm>({
  /** 主键id */
  id: 0,
  /** 位置：枚举【ShopBannerSite】 */
  site: undefined,
  /** 图片链接 */
  picture: "",
  /** 跳转地址 */
  href: "",
  /** 跳转类型：枚举【ShopBannerType】 */
  type: undefined,
  /** 排序 */
  sort: 1,
  /** 状态：枚举【Status】 */
  status: Status.ENABLE
});

const rules = reactive({
  site: [{ required: true, message: "请选择位置", trigger: "change" }],
  picture: [{ required: true, message: "请选择上传图片", trigger: "change" }],
  type: [{ required: true, message: "请选择跳转类型", trigger: "change" }],
  href: [{ required: true, message: "跳转地址不能为空", trigger: "blur" }]
});

const statusOptions: ProFormOption[] = [
  { label: "启用", value: Status.ENABLE },
  { label: "禁用", value: Status.DISABLE }
];

/** 轮播图表单字段配置。 */
const formFields = computed<ProFormField[]>(() => [
  { prop: "site", label: "位置", component: "dict", props: { code: "shop_banner_site" } },
  { prop: "picture", label: "照片", component: "image-upload", props: { uploadType: "banner" } },
  { prop: "type", label: "跳转类型", component: "dict", props: { code: "shop_banner_type" } },
  {
    prop: "href",
    label: "跳转链接",
    component: "select",
    options: goodsList.value.map(item => ({ label: item.name, value: String(item.id) })),
    props: { placeholder: "请选择" },
    visible: model => model.type == ShopBannerType.GOODS_DETAIL
  },
  {
    prop: "href",
    label: "跳转链接",
    component: "tree-select",
    options: goodsCategoryOptions.value,
    props: { placeholder: "请选择商品分类", filterable: true, style: { width: "100%" } },
    visible: model => model.type == ShopBannerType.CATEGORY_DETAIL
  },
  {
    prop: "href",
    label: "跳转链接",
    component: "input",
    props: { placeholder: "请输入跳转链接" },
    visible: model => model.type == ShopBannerType.WEB_VIEW || model.type == ShopBannerType.MINI
  },
  {
    prop: "sort",
    label: "排序",
    component: "input-number",
    props: { min: 1, precision: 0, step: 1, controlsPosition: "right", style: { width: "100%" } }
  },
  { prop: "status", label: "状态", component: "radio-group", options: statusOptions }
]);

/** 轮播图表格列配置。 */
const columns: ColumnProps[] = [
  { type: "selection", width: 55 },
  { prop: "site", label: "位置", dictCode: "shop_banner_site", search: { el: "select" } },
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
  { prop: "type", label: "跳转类型", dictCode: "shop_banner_type", search: { el: "select" } },
  { prop: "href", label: "跳转链接" },
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
      disabled: () => !BUTTONS.value["shop:banner:status"],
      beforeChange: scope => handleBeforeSetStatus(scope.row as ShopBanner)
    }
  },
  { prop: "createdAt", label: "创建时间", minWidth: 180 },
  { prop: "updatedAt", label: "更新时间", minWidth: 180 },
  {
    prop: "operation",
    label: "操作",
    width: 150,
    fixed: "right",
    cellType: "actions",
    actions: [
      {
        label: "编辑",
        type: "primary",
        link: true,
        icon: EditPen,
        hidden: () => !BUTTONS.value["shop:banner:update"],
        params: scope => ({ bannerId: scope.row.id }),
        onClick: (scope, params) => handleOpenDialog((params?.bannerId as number | undefined) ?? (scope.row as ShopBanner).id)
      },
      {
        label: "删除",
        type: "danger",
        link: true,
        icon: Delete,
        hidden: () => !BUTTONS.value["shop:banner:delete"],
        onClick: scope => handleDelete(scope.row as ShopBanner)
      }
    ]
  }
];

/** 轮播图表格顶部按钮配置。 */
const headerActions: HeaderActionProps[] = [
  {
    label: "新增",
    type: "success",
    icon: CirclePlus,
    hidden: () => !BUTTONS.value["shop:banner:create"],
    onClick: () => handleOpenDialog()
  },
  {
    label: "删除",
    type: "danger",
    icon: Delete,
    hidden: () => !BUTTONS.value["shop:banner:delete"],
    disabled: scope => !scope.selectedList.length,
    onClick: scope => handleDelete(scope.selectedList as ShopBanner[])
  }
];

/**
 * 递归转换分类选项，适配树形选择组件结构。
 */
function categoryOption(oldCategories: TreeOptionResponse_Option[]): CategoryOption[] {
  return oldCategories.map(oldCategory => ({
    value: String(oldCategory.value),
    label: oldCategory.label,
    disabled: oldCategory.disabled,
    children: oldCategory.children ? categoryOption(oldCategory.children) : []
  }));
}

/**
 * 请求轮播图列表，并由 ProTable 统一管理分页和筛选。
 */
async function requestShopBannerTable(params: PageShopBannerRequest) {
  const data = await defShopBannerService.PageShopBanner(buildPageRequest(params));
  return { data };
}

/**
 * 刷新轮播图表格。
 */
function refreshTable() {
  proTable.value?.getTableList();
}

/**
 * 预加载商品与分类选项，确保弹窗打开时下拉可用。
 */
async function loadBannerOptions() {
  const [listGoodsInfoResponse, optionGoodsCategoryResponse] = await Promise.all([
    defGoodsInfoService.ListGoodsInfo({ name: "" }),
    defGoodsCategoryService.OptionGoodsCategory({})
  ]);

  goodsList.value = listGoodsInfoResponse.list || [];
  goodsCategoryOptions.value = categoryOption(optionGoodsCategoryResponse.list || []);
}

/**
 * 重置轮播图表单，避免切换操作时残留旧值。
 */
function resetForm() {
  formDialogRef.value?.resetFields();
  formDialogRef.value?.clearValidate();
  formData.id = 0;
  formData.site = undefined;
  formData.picture = "";
  formData.href = "";
  formData.type = undefined;
  formData.sort = 1;
  formData.status = Status.ENABLE;
}

/**
 * 打开轮播图弹窗，并预加载跳转目标数据。
 */
async function handleOpenDialog(bannerId?: number) {
  await loadBannerOptions();
  dialog.visible = true;
  if (bannerId) {
    dialog.title = "修改轮播图";
    defShopBannerService.GetShopBanner({ value: bannerId }).then(data => {
      Object.assign(formData, data);
    });
    return;
  }

  dialog.title = "新增轮播图";
  resetForm();
}

/**
 * 提交轮播图表单。
 */
function handleSubmit() {
  formDialogRef.value?.validate()?.then(valid => {
    if (!valid) return;

    const request = formData.id
      ? defShopBannerService.UpdateShopBanner(formData)
      : defShopBannerService.CreateShopBanner(formData);
    request.then(() => {
      ElMessage.success(formData.id ? "修改轮播图成功" : "新增轮播图成功");
      handleCloseDialog();
      refreshTable();
    });
  });
}

/**
 * 关闭轮播图弹窗并恢复默认表单值。
 */
function handleCloseDialog() {
  dialog.visible = false;
  resetForm();
}

/**
 * 在轮播图状态切换前先完成确认与接口调用，避免首屏渲染触发误操作。
 */
async function handleBeforeSetStatus(row: ShopBanner) {
  const nextStatus = row.status === Status.ENABLE ? Status.DISABLE : Status.ENABLE;
  const text = nextStatus === Status.ENABLE ? "启用" : "禁用";
  try {
    await ElMessageBox.confirm(`是否确定${text}轮播图？`, "提示", {
      confirmButtonText: "确认",
      cancelButtonText: "取消",
      type: "warning"
    });
    await defShopBannerService.SetShopBannerStatus({ id: row.id, status: nextStatus });
    ElMessage.success(`${text}成功`);
    refreshTable();
    return true;
  } catch {
    return false;
  }
}

/**
 * 删除轮播图，兼容单项删除与批量删除。
 */
function handleDelete(selected?: number | string | Array<number | string> | ShopBanner | ShopBanner[]) {
  const bannerList = Array.isArray(selected)
    ? (selected.filter(item => typeof item === "object") as ShopBanner[])
    : selected && typeof selected === "object"
      ? [selected as ShopBanner]
      : [];
  const bannerIds = (
    bannerList.length
      ? bannerList.map(item => item.id)
      : normalizeSelectedIds(selected as number | string | Array<number | string>)
  ).join(",");
  if (!bannerIds) {
    ElMessage.warning("请勾选删除项");
    return;
  }

  const confirmMessage = bannerList.length
    ? bannerList.length === 1
      ? "是否确定删除轮播图？"
      : `确认删除已选中的 ${bannerList.length} 张轮播图吗？`
    : "确认删除已选中的轮播图吗？";

  ElMessageBox.confirm(confirmMessage, "警告", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning"
  }).then(
    () => {
      defShopBannerService.DeleteShopBanner({ value: bannerIds }).then(() => {
        ElMessage.success("删除轮播图成功");
        refreshTable();
      });
    },
    () => {
      ElMessage.info("已取消删除轮播图");
    }
  );
}
</script>
