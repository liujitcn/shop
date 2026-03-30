<!-- 热门推荐选项数据 -->
<template>
  <div class="table-box">
    <ProTable
      ref="proTable"
      row-key="id"
      :columns="columns"
      :header-actions="headerActions"
      :request-api="requestShopHotItemTable"
      :init-param="initParam"
    />

    <FormDialog
      v-model="dialog.visible"
      ref="formDialogRef"
      :title="dialog.title"
      width="1200px"
      :model="formData"
      :fields="formFields"
      :rules="rules"
      label-width="150px"
      @confirm="handleSubmit"
      @close="handleCloseDialog"
    >
      <template #goodsTransferItem="slotScope">
        <el-popover effect="light" trigger="hover" placement="top" width="auto">
          <template #default>
            <div>价格：{{ formatPrice(slotScope.option.price) }}</div>
          </template>
          <template #reference>{{ slotScope.option.label }}</template>
        </el-popover>
      </template>
    </FormDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from "vue";
import { useRoute } from "vue-router";
import { ElMessage, ElMessageBox } from "element-plus";
import { CirclePlus, Delete, EditPen } from "@element-plus/icons-vue";
import type { ColumnProps, HeaderActionProps, ProTableInstance } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
import FormDialog from "@/components/Dialog/FormDialog.vue";
import type { ProFormField, ProFormOption } from "@/components/ProForm/interface";
import { useAuthButtons } from "@/hooks/useAuthButtons";
import { defShopHotService } from "@/api/admin/shop_hot";
import { defGoodsService } from "@/api/admin/goods";
import type { ListGoodsResponse_Goods } from "@/rpc/admin/goods";
import type { PageShopHotItemRequest, ShopHotItem, ShopHotItemForm } from "@/rpc/admin/shop_hot";
import { Status } from "@/rpc/common/enum";
import { buildPageRequest, normalizeSelectedIds } from "@/utils/proTable";
import { formatPrice } from "@/utils/utils";

defineOptions({
  name: "ShopHotItem",
  inheritAttrs: false
});

const route = useRoute();
const { BUTTONS } = useAuthButtons();
const proTable = ref<ProTableInstance>();
const formDialogRef = ref<InstanceType<typeof FormDialog>>();

const hotId = ref(Number(route.query.hotId ?? 0));
const initParam = reactive({
  hotId: hotId.value
});

const dialog = reactive({
  title: "",
  visible: false
});

const formData = reactive<ShopHotItemForm>({
  /** 热门推荐选项ID */
  id: 0,
  /** 热门推荐ID */
  hotId: hotId.value,
  /** 标题 */
  title: "",
  /** 排序 */
  sort: 1,
  /** 商品ID */
  goodsIds: [],
  /** 状态 */
  status: Status.ENABLE
});

const goodsList = ref<ListGoodsResponse_Goods[]>([]);

const rules = computed(() => ({
  title: [{ required: true, message: "请输入热门推荐选项标题", trigger: "blur" }],
  status: [{ required: true, message: "状态不能为空", trigger: "blur" }]
}));

const statusOptions: ProFormOption[] = [
  { label: "启用", value: Status.ENABLE },
  { label: "禁用", value: Status.DISABLE }
];

/** 推荐商品穿梭框数据。 */
const transferData = computed(() =>
  goodsList.value.map(item => ({
    ...item,
    value: item.id,
    label: `${item.categoryName}/${item.name}`
  }))
);

/** 热门推荐选项表单字段配置。 */
const formFields = computed<ProFormField[]>(() => [
  {
    prop: "title",
    label: "热门推荐选项标题",
    component: "input",
    props: { placeholder: "请输入热门推荐选项标题" }
  },
  { prop: "status", label: "状态", component: "radio-group", options: statusOptions },
  {
    prop: "sort",
    label: "排序",
    component: "input-number",
    props: { min: 1, precision: 0, step: 1, controlsPosition: "right", style: { width: "100%" } }
  },
  {
    prop: "goodsIds",
    label: "推荐商品",
    component: "transfer",
    slotName: "goodsTransferItem",
    options: transferData.value,
    props: { titles: ["可选商品", "已选商品"], filterable: true },
    colSpan: 24
  }
]);

/** 热门推荐选项表格列配置。 */
const columns: ColumnProps[] = [
  { type: "selection", width: 55 },
  { prop: "title", label: "热门推荐选项标题", search: { el: "input" } },
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
      disabled: () => !BUTTONS.value["shop:hot-item:status"],
      beforeChange: scope => handleBeforeSetStatus(scope.row as ShopHotItem)
    }
  },
  { prop: "createdAt", label: "创建时间", width: 180 },
  { prop: "updatedAt", label: "更新时间", width: 180 },
  {
    prop: "operation",
    label: "操作",
    width: 180,
    fixed: "right",
    cellType: "actions",
    actions: [
      {
        label: "编辑",
        type: "primary",
        link: true,
        icon: EditPen,
        hidden: () => !BUTTONS.value["shop:hot-item:update"],
        params: scope => ({ hotItemId: scope.row.id }),
        onClick: (scope, params) => handleOpenDialog((params?.hotItemId as number | undefined) ?? (scope.row as ShopHotItem).id)
      },
      {
        label: "删除",
        type: "danger",
        link: true,
        icon: Delete,
        hidden: () => !BUTTONS.value["shop:hot-item:delete"],
        onClick: scope => handleDelete(scope.row as ShopHotItem)
      }
    ]
  }
];

/** 热门推荐选项顶部按钮配置。 */
const headerActions: HeaderActionProps[] = [
  {
    label: "新增",
    type: "success",
    icon: CirclePlus,
    hidden: () => !BUTTONS.value["shop:hot-item:create"],
    onClick: () => handleOpenDialog()
  },
  {
    label: "删除",
    type: "danger",
    icon: Delete,
    hidden: () => !BUTTONS.value["shop:hot-item:delete"],
    disabled: scope => !scope.selectedList.length,
    onClick: scope => handleDelete(scope.selectedList as ShopHotItem[])
  }
];

/**
 * 监听路由中的热门推荐 ID，切换后刷新列表并同步表单默认值。
 */
watch(
  () => route.query.hotId,
  newHotId => {
    hotId.value = Number(newHotId ?? 0);
    initParam.hotId = hotId.value;
    formData.hotId = hotId.value;
    formData.id = 0;
    proTable.value?.search();
  }
);

/**
 * 请求热门推荐选项分页数据，并附带当前热门推荐 ID。
 */
async function requestShopHotItemTable(params: PageShopHotItemRequest) {
  const data = await defShopHotService.PageShopHotItem(
    buildPageRequest({
      ...params,
      hotId: hotId.value
    })
  );
  return { data };
}

/**
 * 刷新当前热门推荐选项表格。
 */
function refreshTable() {
  proTable.value?.getTableList();
}

/**
 * 加载推荐商品下拉数据，供穿梭框使用。
 */
async function loadGoodsOptions() {
  const listGoodsResponse = await defGoodsService.ListGoods({ name: "" });
  goodsList.value = listGoodsResponse.list || [];
}

/**
 * 重置热门推荐选项表单，避免新增和编辑之间互相污染。
 */
function resetForm() {
  formDialogRef.value?.resetFields();
  formDialogRef.value?.clearValidate();
  formData.id = 0;
  formData.hotId = hotId.value;
  formData.title = "";
  formData.sort = 1;
  formData.goodsIds = [];
  formData.status = Status.ENABLE;
}

/**
 * 打开热门推荐选项弹窗，并预加载推荐商品数据。
 */
async function handleOpenDialog(hotItemId?: number) {
  resetForm();
  await loadGoodsOptions();
  dialog.title = hotItemId ? "修改热门推荐选项" : "新增热门推荐选项";
  dialog.visible = true;
  if (!hotItemId) return;

  defShopHotService.GetShopHotItem({ value: hotItemId }).then(data => {
    Object.assign(formData, data);
  });
}

/**
 * 关闭热门推荐选项弹窗并恢复默认值。
 */
function handleCloseDialog() {
  dialog.visible = false;
  resetForm();
}

/**
 * 提交热门推荐选项表单。
 */
function handleSubmit() {
  formDialogRef.value?.validate()?.then(isValid => {
    if (!isValid) return;

    formData.hotId = hotId.value;
    const submitData = JSON.parse(JSON.stringify(formData)) as ShopHotItemForm;
    const request = submitData.id
      ? defShopHotService.UpdateShopHotItem(submitData)
      : defShopHotService.CreateShopHotItem(submitData);
    request.then(() => {
      ElMessage.success(submitData.id ? "修改成功" : "新增成功");
      handleCloseDialog();
      refreshTable();
    });
  });
}

/**
 * 在热门推荐选项状态切换前先完成确认与接口调用，避免首屏渲染触发误操作。
 */
async function handleBeforeSetStatus(row: ShopHotItem) {
  const nextStatus = row.status === Status.ENABLE ? Status.DISABLE : Status.ENABLE;
  const text = nextStatus === Status.ENABLE ? "启用" : "禁用";
  const hotItemName = row.title || `ID:${row.id}`;
  try {
    await ElMessageBox.confirm(`是否确定${text}推荐项：${hotItemName}？`, "提示", {
      confirmButtonText: "确认",
      cancelButtonText: "取消",
      type: "warning"
    });
    await defShopHotService.SetShopHotItemStatus({ id: row.id, status: nextStatus });
    ElMessage.success(`${text}成功`);
    refreshTable();
    return true;
  } catch {
    return false;
  }
}

/**
 * 删除热门推荐选项，兼容单项删除与批量删除。
 */
function handleDelete(selected?: number | string | Array<number | string> | ShopHotItem | ShopHotItem[]) {
  const hotItemList = Array.isArray(selected)
    ? (selected.filter(item => typeof item === "object") as ShopHotItem[])
    : selected && typeof selected === "object"
      ? [selected as ShopHotItem]
      : [];
  const hotItemIds = (
    hotItemList.length
      ? hotItemList.map(item => item.id)
      : normalizeSelectedIds(selected as number | string | Array<number | string>)
  ).join(",");
  if (!hotItemIds) {
    ElMessage.warning("请勾选删除项");
    return;
  }

  const confirmMessage = hotItemList.length
    ? hotItemList.length === 1
      ? `是否确定删除推荐项：${hotItemList[0].title || `ID:${hotItemList[0].id}`}？`
      : `确认删除已选中的 ${hotItemList.length} 个热门推荐项吗？`
    : "确认删除已选中的热门推荐项吗？";

  ElMessageBox.confirm(confirmMessage, "警告", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning"
  }).then(
    () => {
      defShopHotService.DeleteShopHotItem({ value: hotItemIds }).then(() => {
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

<style scoped>
:deep(.el-transfer-panel) {
  width: 450px;
}

:deep(.el-transfer-panel__list) {
  width: 100%;
  height: 400px;
}
</style>
