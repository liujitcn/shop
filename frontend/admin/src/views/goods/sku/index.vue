<!-- 规格 -->
<template>
  <div class="table-box">
    <ProTable ref="proTable" row-key="id" :columns="columns" :request-api="requestGoodsSkuTable" :init-param="initParam">
      <template v-for="(item, index) in specList" :key="item.id" #[`spec_item${index}`]="scope">
        {{ scope.row[`spec_item${index}`] }}
      </template>

      <template #init_sale_num="scope">
        {{ scope.row.init_sale_num || 0 }}
      </template>

      <template #real_sale_num="scope">
        {{ scope.row.real_sale_num || 0 }}
      </template>
    </ProTable>

    <FormDialog
      v-model="dialog.visible"
      ref="formDialogRef"
      :title="dialog.title"
      width="700px"
      :model="formData"
      :fields="formFields"
      :rules="rules"
      label-width="180px"
      @confirm="handleSubmit"
      @close="handleCloseDialog"
    >
      <template #skuCodeText>{{ formData.sku_code || "-" }}</template>
      <template #specItemText>{{ formData.spec_item.join(" / ") || "-" }}</template>
      <template #initSaleNumText>{{ formData.init_sale_num }}</template>
      <template #realSaleNumText>{{ formData.real_sale_num }}</template>
    </FormDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from "vue";
import { useRoute } from "vue-router";
import { ElMessage } from "element-plus";
import { EditPen } from "@element-plus/icons-vue";
import type { ColumnProps, ProTableInstance } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
import FormDialog from "@/components/Dialog/FormDialog.vue";
import type { ProFormField } from "@/components/ProForm/interface";
import { defGoodsSkuService } from "@/api/admin/goods_sku";
import { defGoodsSpecService } from "@/api/admin/goods_spec";
import type { GoodsSku, PageGoodsSkusRequest } from "@/rpc/admin/v1/goods_sku";
import type { GoodsSpec } from "@/rpc/admin/v1/goods_spec";
import { buildPageRequest } from "@/utils/proTable";

defineOptions({
  name: "GoodsSku",
  inheritAttrs: false
});

const route = useRoute();
const proTable = ref<ProTableInstance>();
const formDialogRef = ref<InstanceType<typeof FormDialog>>();

const goodsId = computed(() => Number(route.query.goodsId ?? 0));
const initParam = computed(() => ({
  goods_id: goodsId.value
}));

const specList = ref<GoodsSpec[]>([]);

const dialog = reactive({
  title: "",
  visible: false
});

const formData = reactive<GoodsSku>({
  /** 商品SKUID */
  id: 0,
  /** 商品id */
  goods_id: 0,
  /** 商品图片 */
  picture: "",
  /** SKU SKU组成，需要与 goods_spec 数组顺序对应 */
  spec_item: [],
  /** SKU编码 */
  sku_code: "",
  /** 当前价格(分) */
  price: 0,
  /** 折扣价格（分） */
  discount_price: 0,
  /** 初始销量 */
  init_sale_num: 0,
  /** 真实销售数量 */
  real_sale_num: 0,
  /** 库存数量 */
  inventory: 0
});

const rules = reactive({});

/** SKU 表单字段配置。 */
const formFields = computed<ProFormField[]>(() => [
  { prop: "sku_code", label: "规格编号", component: "slot", slotName: "skuCodeText" },
  { prop: "spec_item", label: "规格内容", component: "slot", slotName: "specItemText" },
  { prop: "init_sale_num", label: "初始销量", component: "slot", slotName: "initSaleNumText" },
  { prop: "real_sale_num", label: "真实销量", component: "slot", slotName: "realSaleNumText" },
  {
    prop: "inventory",
    label: "库存",
    component: "input-number",
    props: {
      min: 1,
      controlsPosition: "right",
      precision: 0,
      step: 1,
      style: { width: "100%" }
    }
  },
  {
    prop: "price",
    label: "价格（元）",
    component: "input-number",
    props: {
      precision: 2,
      step: 0.1,
      style: { width: "100%" }
    }
  },
  {
    prop: "discount_price",
    label: "折扣价格（元）",
    component: "input-number",
    props: {
      precision: 2,
      step: 0.1,
      style: { width: "100%" }
    }
  }
]);

/** 商品库存表格列配置，前缀规格列随商品规格变化动态拼装。 */
const columns = computed<ColumnProps[]>(() => {
  const specColumns = specList.value.map((item, index) => ({
    prop: `spec_item${index}`,
    label: item.name,
    align: "center"
  }));

  return [
    ...specColumns,
    { prop: "sku_code", label: "规格编号", minWidth: 140, search: { el: "input" } },
    { prop: "init_sale_num", label: "初始销量", minWidth: 100, align: "right" },
    { prop: "real_sale_num", label: "真实销量", minWidth: 100, align: "right" },
    { prop: "price", label: "价格（元）", minWidth: 110, align: "right", cellType: "money" },
    { prop: "discount_price", label: "折扣价格（元）", minWidth: 130, align: "right", cellType: "money" },
    { prop: "inventory", label: "库存", minWidth: 90, align: "right" },
    {
      prop: "operation",
      label: "操作",
      width: 100,
      fixed: "right",
      cellType: "actions",
      actions: [
        {
          label: "编辑",
          type: "primary",
          link: true,
          icon: EditPen,
          params: scope => ({ skuId: scope.row.id }),
          onClick: (scope, params) => handleOpenDialog((params?.skuId as number | undefined) ?? (scope.row as GoodsSku).id)
        }
      ]
    }
  ];
});

/**
 * 加载当前商品的规格定义，用于拼装动态表头。
 */
async function loadSpecList() {
  if (!goodsId.value) {
    specList.value = [];
    return;
  }

  const res = await defGoodsSpecService.ListGoodsSpecs({ goods_id: goodsId.value });
  specList.value = res.goods_specs ?? [];
}

/**
 * 请求 SKU 列表，并将规格内容展开为动态列字段。
 */
async function requestGoodsSkuTable(params: PageGoodsSkusRequest) {
  const data = await defGoodsSkuService.PageGoodsSkus(buildPageRequest({ ...params, goods_id: goodsId.value }));
  const list = (data.goods_skus ?? []).map(item => {
    item.spec_item.forEach((spec, index) => {
      (item as Record<string, any>)[`spec_item${index}`] = spec;
    });
    return item;
  });

  return {
    data: {
      ...data,
      list
    }
  };
}

/**
 * 刷新 SKU 表格。
 */
function refreshTable() {
  proTable.value?.getTableList();
}

/**
 * 打开规格编辑弹窗。
 */
function handleOpenDialog(skuId: number) {
  resetForm();
  dialog.visible = true;
  dialog.title = "修改规格";
  defGoodsSkuService.GetGoodsSku({ id: skuId }).then(data => {
    data.price = data.price / 100;
    data.discount_price = data.discount_price / 100;
    Object.assign(formData, data);
  });
}

/**
 * 提交 SKU 表单，仅允许更新现有规格。
 */
function handleSubmit() {
  formDialogRef.value?.validate()?.then(valid => {
    if (!valid) return;

    const submitData = JSON.parse(JSON.stringify(formData)) as GoodsSku;
    submitData.price = submitData.price * 100;
    submitData.discount_price = submitData.discount_price * 100;
    defGoodsSkuService.UpdateGoodsSku({ id: submitData.id, goods_sku: submitData }).then(() => {
      ElMessage.success("修改 SKU 成功");
      handleCloseDialog();
      refreshTable();
    });
  });
}

/**
 * 重置 SKU 表单，避免弹窗之间数据串用。
 */
function resetForm() {
  formDialogRef.value?.resetFields();
  formDialogRef.value?.clearValidate();
  formData.id = 0;
  formData.goods_id = 0;
  formData.picture = "";
  formData.spec_item = [];
  formData.sku_code = "";
  formData.price = 0;
  formData.discount_price = 0;
  formData.init_sale_num = 0;
  formData.real_sale_num = 0;
  formData.inventory = 0;
}

/**
 * 关闭规格弹窗并恢复默认值。
 */
function handleCloseDialog() {
  dialog.visible = false;
  resetForm();
}

watch(
  goodsId,
  async () => {
    await loadSpecList();
    refreshTable();
  },
  { immediate: true }
);
</script>
