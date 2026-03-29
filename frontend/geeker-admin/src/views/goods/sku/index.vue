<!-- 规格 -->
<template>
  <div class="table-box">
    <ProTable ref="proTable" row-key="id" :columns="columns" :request-api="requestGoodsSkuTable" :init-param="initParam">
      <template v-for="(item, index) in specList" :key="item.id" #[`specItem${index}`]="scope">
        {{ scope.row[`specItem${index}`] }}
      </template>

      <template #initSaleNum="scope">
        {{ scope.row.initSaleNum || 0 }}
      </template>

      <template #realSaleNum="scope">
        {{ scope.row.realSaleNum || 0 }}
      </template>

      <template #price="scope">
        {{ formatPrice(scope.row.price) }}
      </template>

      <template #discountPrice="scope">
        {{ formatPrice(scope.row.discountPrice) }}
      </template>

      <template #operation="scope">
        <el-button v-hasPerm="['goods:sku:update']" type="primary" link :icon="EditPen" @click="handleOpenDialog(scope.row.id)">
          编辑
        </el-button>
      </template>
    </ProTable>

    <el-dialog v-model="dialog.visible" :title="dialog.title" width="700px" @close="handleCloseDialog">
      <ProForm ref="proFormRef" :model="formData" :fields="formFields" :rules="rules" label-width="180px">
        <template #skuCodeText>{{ formData.skuCode || "-" }}</template>
        <template #specItemText>{{ formData.specItem.join(" / ") || "-" }}</template>
        <template #initSaleNumText>{{ formData.initSaleNum }}</template>
        <template #realSaleNumText>{{ formData.realSaleNum }}</template>
      </ProForm>
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
import { computed, reactive, ref, watch } from "vue";
import { useRoute } from "vue-router";
import { ElMessage } from "element-plus";
import { EditPen } from "@element-plus/icons-vue";
import type { ColumnProps, ProTableInstance } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
import ProForm from "@/components/ProForm/index.vue";
import type { ProFormField, ProFormInstance } from "@/components/ProForm/interface";
import { defGoodsSkuService } from "@/api/admin/goods_sku";
import { defGoodsSpecService } from "@/api/admin/goods_spec";
import type { GoodsSku, PageGoodsSkuRequest } from "@/rpc/admin/goods_sku";
import type { GoodsSpec } from "@/rpc/admin/goods_spec";
import { buildPageRequest } from "@/utils/proTable";
import { formatPrice } from "@/utils/utils";

defineOptions({
  name: "GoodsSku",
  inheritAttrs: false
});

const route = useRoute();
const proTable = ref<ProTableInstance>();
const proFormRef = ref<ProFormInstance>();

const goodsId = computed(() => Number(route.query.goodsId ?? 0));
const initParam = computed(() => ({
  goodsId: goodsId.value
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
  goodsId: 0,
  /** 商品图片 */
  picture: "",
  /** SKU SKU组成，需要与 goods_spec 数组顺序对应 */
  specItem: [],
  /** SKU编码 */
  skuCode: "",
  /** 当前价格(分) */
  price: 0,
  /** 折扣价格（分） */
  discountPrice: 0,
  /** 初始销量 */
  initSaleNum: 0,
  /** 真实销售数量 */
  realSaleNum: 0,
  /** 库存数量 */
  inventory: 0
});

const rules = reactive({});

/** SKU 表单字段配置。 */
const formFields = computed<ProFormField[]>(() => [
  { prop: "skuCode", label: "规格编号", component: "slot", slotName: "skuCodeText" },
  { prop: "specItem", label: "规格内容", component: "slot", slotName: "specItemText" },
  { prop: "initSaleNum", label: "初始销量", component: "slot", slotName: "initSaleNumText" },
  { prop: "realSaleNum", label: "真实销量", component: "slot", slotName: "realSaleNumText" },
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
    prop: "discountPrice",
    label: "折扣价格（元）",
    component: "input-number",
    props: {
      precision: 2,
      step: 0.1,
      style: { width: "100%" }
    }
  }
]);

/** 动态拼装 SKU 表格列，前缀规格列随商品规格变化。 */
const columns = computed<ColumnProps[]>(() => {
  const specColumns = specList.value.map((item, index) => ({
    prop: `specItem${index}`,
    label: item.name,
    align: "center"
  }));

  return [
    ...specColumns,
    { prop: "skuCode", label: "规格编号", search: { el: "input" } },
    { prop: "initSaleNum", label: "初始销量", align: "right" },
    { prop: "realSaleNum", label: "真实销量", align: "right" },
    { prop: "price", label: "价格（元）", align: "right" },
    { prop: "discountPrice", label: "折扣价格（元）", align: "right" },
    { prop: "inventory", label: "库存", align: "right" },
    { prop: "operation", label: "操作", width: 100, fixed: "right" }
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

  const res = await defGoodsSpecService.ListGoodsSpec({ goodsId: goodsId.value });
  specList.value = res.list ?? [];
}

/**
 * 请求 SKU 列表，并将规格内容展开为动态列字段。
 */
async function requestGoodsSkuTable(params: PageGoodsSkuRequest) {
  const data = await defGoodsSkuService.PageGoodsSku(buildPageRequest({ ...params, goodsId: goodsId.value }));
  const list = (data.list ?? []).map(item => {
    item.specItem.forEach((spec, index) => {
      (item as Record<string, any>)[`specItem${index}`] = spec;
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
  dialog.visible = true;
  dialog.title = "修改规格";
  defGoodsSkuService.GetGoodsSku({ value: skuId }).then(data => {
    data.price = data.price / 100;
    data.discountPrice = data.discountPrice / 100;
    Object.assign(formData, data);
  });
}

/**
 * 提交 SKU 表单，仅允许更新现有规格。
 */
function handleSubmit() {
  proFormRef.value?.validate()?.then(valid => {
    if (!valid) return;

    const submitData = JSON.parse(JSON.stringify(formData)) as GoodsSku;
    submitData.price = submitData.price * 100;
    submitData.discountPrice = submitData.discountPrice * 100;
    defGoodsSkuService.UpdateGoodsSku(submitData).then(() => {
      ElMessage.success("修改成功");
      handleCloseDialog();
      refreshTable();
    });
  });
}

/**
 * 重置 SKU 表单，避免弹窗之间数据串用。
 */
function resetForm() {
  proFormRef.value?.resetFields();
  proFormRef.value?.clearValidate();
  formData.id = 0;
  formData.goodsId = 0;
  formData.picture = "";
  formData.specItem = [];
  formData.skuCode = "";
  formData.price = 0;
  formData.discountPrice = 0;
  formData.initSaleNum = 0;
  formData.realSaleNum = 0;
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
