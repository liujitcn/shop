<!-- 轮播图 -->
<template>
  <div class="table-box">
    <ProTable ref="proTable" row-key="id" :columns="columns" :request-api="requestShopBannerTable">
      <template #tableHeader="{ selectedList }">
        <el-button v-hasPerm="['shop:banner:create']" type="success" :icon="CirclePlus" @click="handleOpenDialog()">
          新增
        </el-button>
        <el-button
          v-hasPerm="['shop:banner:delete']"
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
          :disabled="!BUTTONS['shop:banner:status']"
          :before-change="() => handleBeforeSetStatus(scope.row)"
        />
      </template>

      <template #operation="scope">
        <el-button v-hasPerm="['shop:banner:update']" type="primary" link :icon="EditPen" @click="handleOpenDialog(scope.row.id)">
          编辑
        </el-button>
        <el-button v-hasPerm="['shop:banner:delete']" type="danger" link :icon="Delete" @click="handleDelete(scope.row)">
          删除
        </el-button>
      </template>
    </ProTable>

    <el-dialog v-model="dialog.visible" :title="dialog.title" width="500px" @close="handleCloseDialog">
      <ProForm ref="proFormRef" :model="formData" :fields="formFields" :rules="rules" label-width="100px" />
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
import { computed, reactive, ref } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { CirclePlus, Delete, EditPen } from "@element-plus/icons-vue";
import type { ColumnProps, ProTableInstance } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
import ProForm from "@/components/ProForm/index.vue";
import type { ProFormField, ProFormInstance, ProFormOption } from "@/components/ProForm/interface";
import { useAuthButtons } from "@/hooks/useAuthButtons";
import { defShopBannerService } from "@/api/admin/shop_banner";
import type { PageShopBannerRequest, ShopBanner, ShopBannerForm } from "@/rpc/admin/shop_banner";
import type { ListGoodsResponse_Goods } from "@/rpc/admin/goods";
import { defGoodsService } from "@/api/admin/goods";
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
const proFormRef = ref<ProFormInstance>();

const goodsList = ref<ListGoodsResponse_Goods[]>([]);
const goodsCategoryOptions = ref<CategoryOption[]>([]);

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
  { prop: "picture", label: "照片", component: "image-upload" },
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
  { prop: "picture", label: "图片", minWidth: 150 },
  { prop: "type", label: "跳转类型", dictCode: "shop_banner_type", search: { el: "select" } },
  { prop: "href", label: "跳转链接" },
  { prop: "sort", label: "排序", align: "right" },
  { prop: "status", label: "状态", width: 100, dictCode: "status", search: { el: "select" } },
  { prop: "createdAt", label: "创建时间", width: 180 },
  { prop: "updatedAt", label: "更新时间", width: 180 },
  { prop: "operation", label: "操作", width: 150, fixed: "right" }
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
  const [listGoodsResponse, optionGoodsCategoryResponse] = await Promise.all([
    defGoodsService.ListGoods({ name: "" }),
    defGoodsCategoryService.OptionGoodsCategory({})
  ]);

  goodsList.value = listGoodsResponse.list || [];
  goodsCategoryOptions.value = categoryOption(optionGoodsCategoryResponse.list || []);
}

/**
 * 重置轮播图表单，避免切换操作时残留旧值。
 */
function resetForm() {
  proFormRef.value?.resetFields();
  proFormRef.value?.clearValidate();
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
  proFormRef.value?.validate()?.then(valid => {
    if (!valid) return;

    const request = formData.id
      ? defShopBannerService.UpdateShopBanner(formData)
      : defShopBannerService.CreateShopBanner(formData);
    request.then(() => {
      ElMessage.success(formData.id ? "修改成功" : "新增成功");
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
  const bannerName = row.href || row.picture || `ID:${row.id}`;
  try {
    await ElMessageBox.confirm(`是否确定${text}轮播图：${bannerName}？`, "提示", {
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
      ? `是否确定删除轮播图：${bannerList[0].href || bannerList[0].picture || `ID:${bannerList[0].id}`}？`
      : `确认删除已选中的 ${bannerList.length} 张轮播图吗？`
    : "确认删除已选中的轮播图吗？";

  ElMessageBox.confirm(confirmMessage, "警告", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning"
  }).then(
    () => {
      defShopBannerService.DeleteShopBanner({ value: bannerIds }).then(() => {
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
