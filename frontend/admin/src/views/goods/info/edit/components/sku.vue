<template>
  <div class="goods-edit-sku">
    <el-card class="goods-edit-sku__hero" shadow="never">
      <div class="goods-edit-sku__hero-content">
        <div>
          <h2 class="goods-edit-sku__title">设置商品库存</h2>
          <p class="goods-edit-sku__desc">先维护规格定义，再为自动生成的 SKU 组合填写编号、价格、库存、规格图片和初始销量。</p>
        </div>

        <div class="goods-edit-sku__summary">
          <div class="goods-edit-sku__summary-item">
            <span>规格数</span>
            <strong>{{ formData.specList?.length ?? 0 }}</strong>
          </div>
          <div class="goods-edit-sku__summary-item">
            <span>SKU 数量</span>
            <strong>{{ formData.skuList?.length ?? 0 }}</strong>
          </div>
          <div class="goods-edit-sku__summary-item">
            <span>库存总量</span>
            <strong>{{ totalInventory }}</strong>
          </div>
          <div class="goods-edit-sku__summary-item">
            <span>价格区间</span>
            <strong>{{ priceRangeText }}</strong>
          </div>
        </div>
      </div>
    </el-card>

    <el-card class="goods-edit-sku__card" shadow="never">
      <template #header>
        <div class="goods-edit-sku__card-header">
          <span>商品规格</span>
          <el-button type="success" icon="plus" @click="handleOpenGoodsSpecDialog()">添加</el-button>
        </div>
      </template>

      <div class="goods-edit-sku__tip">先维护规格名称和规格项，系统会根据所有规格项组合自动生成下方的 SKU 行。</div>

      <ProTable :data="formData.specList" :columns="specColumns" :pagination="false" :tool-button="false">
        <template #operation="scope">
          <el-button type="primary" size="small" link icon="edit" @click="handleOpenGoodsSpecDialog(scope.$index, scope.row)">
            编辑
          </el-button>
          <el-button type="danger" size="small" link icon="delete" @click="handleDeleteGoodsSpec(scope.$index)"> 删除 </el-button>
        </template>
      </ProTable>
    </el-card>

    <el-card class="goods-edit-sku__card" shadow="never">
      <template #header>
        <div class="goods-edit-sku__card-header">
          <span>商品库存</span>
        </div>
      </template>

      <div class="goods-edit-sku__tip">请逐行补齐 SKU 编号、价格、折扣价、库存、规格图片和初始销量，提交前会统一校验。</div>
      <el-form ref="skuFormRef" :model="formData" size="small">
        <ProTable :key="skuTableKey" :data="formData.skuList" :columns="skuColumns" :pagination="false" :tool-button="false">
          <template #skuCode="scope">
            <el-form-item :prop="'skuList[' + scope.$index + '].skuCode'" :rules="rules.skuCode">
              <el-input v-model="scope.row.skuCode" />
            </el-form-item>
          </template>

          <template #price="scope">
            <el-form-item :prop="'skuList[' + scope.$index + '].price'" :rules="rules.price">
              <el-input-number v-model="scope.row.price" :min="0.01" :precision="2" :step="0.01" />
            </el-form-item>
          </template>

          <template #discountPrice="scope">
            <el-form-item :prop="'skuList[' + scope.$index + '].discountPrice'" :rules="rules.discountPrice">
              <el-input-number v-model="scope.row.discountPrice" :min="0.01" :precision="2" :step="0.01" />
            </el-form-item>
          </template>

          <template #inventory="scope">
            <el-form-item :prop="'skuList[' + scope.$index + '].inventory'" :rules="rules.inventory">
              <el-input-number v-model="scope.row.inventory" controls-position="right" :min="1" :precision="0" :step="1" />
            </el-form-item>
          </template>

          <template #picture="scope">
            <el-form-item :prop="'skuList[' + scope.$index + '].picture'">
              <UploadImg v-model:image-url="scope.row.picture" />
            </el-form-item>
          </template>

          <template #initSaleNum="scope">
            <el-form-item :prop="'skuList[' + scope.$index + '].initSaleNum'">
              <el-input-number v-model="scope.row.initSaleNum" controls-position="right" :min="0" :precision="0" :step="1" />
            </el-form-item>
          </template>
        </ProTable>
      </el-form>
      <template #footer>
        <div class="goods-edit-sku__footer">
          <el-button @click="handlePrev">上一步，设置商品属性</el-button>
          <el-button type="primary" @click="submitForm">提交</el-button>
        </div>
      </template>
    </el-card>

    <!-- 规格表单弹窗 -->
    <FormDialog
      v-model="specDialog.visible"
      ref="specFormRef"
      :title="specDialog.title"
      width="500px"
      :model="specDialog.specFormData"
      :fields="specFormFields"
      :rules="specDialog.rules"
      label-width="100px"
      @confirm="handleGoodsSpecSubmit"
      @close="handleCloseGoodsSpecDialog"
    />
  </div>
</template>
<script setup lang="ts">
import { computed, reactive, ref, toRefs } from "vue";
import { useRoute } from "vue-router";
import { ElMessage, ElNotification } from "element-plus";
import type { ColumnProps } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
import FormDialog from "@/components/Dialog/FormDialog.vue";
import type { ProFormField } from "@/components/ProForm/interface";
import UploadImg from "@/components/Upload/Img.vue";
import { defGoodsService } from "@/api/admin/goods";
import type { GoodsSpec } from "@/rpc/admin/goods_spec";
import { useTabsStore } from "@/stores/modules/tabs";
defineOptions({
  name: "GoodsEditSku",
  inheritAttrs: false
});
const emit = defineEmits(["prev", "next", "update:modelValue", "resetForm"]);
const route = useRoute();
const tabsStore = useTabsStore();

type SkuItem = Record<string, any> & {
  id: number;
  skuCode: string;
  price: number;
  discountPrice: number;
  initSaleNum: number;
  realSaleNum: number;
  inventory: number;
  picture?: string;
  specItem?: string[];
};

/** 判断当前字段是否为动态生成的规格项字段。 */
const isDynamicSpecItemKey = (key: string) => /^specItem\d+$/.test(key);

const skuFormRef = ref();
const specFormRef = ref<InstanceType<typeof FormDialog>>();

const props = defineProps({
  modelValue: {
    type: Object,
    default: () => ({})
  }
});

const formData: any = computed({
  get: () => props.modelValue,
  set: value => {
    emit("update:modelValue", value);
  }
});

/** 确保商品规格和 SKU 数组始终存在，避免编辑时空值报错。 */
function ensureSkuFormArrays() {
  if (!Array.isArray(formData.value.specList)) {
    formData.value.specList = [];
  }
  if (!Array.isArray(formData.value.skuList)) {
    formData.value.skuList = [];
  }
}

const state = reactive({
  rules: {
    skuCode: [{ required: true, message: "请输入商品编号", trigger: "blur" }],
    price: [{ required: true, message: "请输入商品价格", trigger: "blur" }],
    discountPrice: [{ required: true, message: "请输入折扣价格", trigger: "blur" }],
    inventory: [{ required: true, message: "请输入商品库存", trigger: "blur" }]
  }
});

const { rules } = toRefs(state);

/** 汇总当前全部 SKU 的库存数量。 */
const totalInventory = computed(() => {
  return (formData.value.skuList ?? []).reduce((total: number, item: SkuItem) => total + Number(item.inventory ?? 0), 0);
});

/** 汇总当前 SKU 价格区间，便于运营快速核对价格梯度。 */
const priceRangeText = computed(() => {
  const priceList = (formData.value.skuList ?? []).map((item: SkuItem) => Number(item.price ?? 0)).filter(price => price > 0);
  if (!priceList.length) return "-";
  const minPrice = Math.min(...priceList);
  const maxPrice = Math.max(...priceList);
  return minPrice === maxPrice ? `${minPrice.toFixed(2)} 元` : `${minPrice.toFixed(2)} ~ ${maxPrice.toFixed(2)} 元`;
});

/** 商品规格编辑表格列配置。 */
const specColumns: ColumnProps[] = [
  { type: "index", width: 50 },
  { prop: "name", label: "规格名称", minWidth: 140 },
  { prop: "item", label: "规格内容", minWidth: 160 },
  { prop: "sort", label: "排序", minWidth: 100, align: "right" },
  { prop: "operation", label: "操作", width: 150, align: "center" }
];

/** 商品 SKU 编辑表格列配置。 */
const skuColumns = computed<ColumnProps[]>(() => {
  const dynamicSpecColumns = (formData.value.specList ?? []).map((item: GoodsSpec, index: number) => ({
    prop: `specItem${index}`,
    label: item.name,
    align: "center"
  }));

  return [
    { type: "index", width: 50 },
    ...dynamicSpecColumns,
    { prop: "skuCode", label: "规格编号", align: "center", minWidth: 160 },
    { prop: "price", label: "价格（元）", align: "center", minWidth: 150 },
    { prop: "discountPrice", label: "折扣价格（元）", align: "center", minWidth: 160 },
    { prop: "inventory", label: "库存", align: "center", minWidth: 140 },
    { prop: "picture", label: "规格图片", align: "center", minWidth: 180 },
    { prop: "initSaleNum", label: "初始销量", align: "center", minWidth: 140 }
  ];
});

const skuTableKey = computed(() =>
  (formData.value.specList ?? []).map((item: GoodsSpec) => `${item.name}:${(item.item ?? []).join(",")}`).join("|")
);

// 规格弹窗状态
const specDialog = reactive({
  visible: false,
  title: "",
  index: -1 as number,
  specFormData: {
    /** 规格ID */
    id: 0,
    /** 规格名称 */
    name: "",
    /** 规格内容 */
    item: [""],
    /** 排序 */
    sort: 1
  } as GoodsSpec,
  rules: {
    name: [{ required: true, message: "请输入规格名称", trigger: "blur" }],
    item: [
      {
        validator: (_rule: unknown, value: string[], callback: (error?: Error) => void) => {
          // 动态规格项统一在这里校验，确保至少存在一项且每项都有内容。
          if (!Array.isArray(value) || !value.length || value.some(item => !String(item ?? "").trim())) {
            callback(new Error("请完善规格内容"));
            return;
          }
          callback();
        },
        trigger: "blur"
      }
    ],
    sort: [{ required: true, message: "请输入排序", trigger: "blur" }]
  }
});

/** 规格弹窗字段配置。 */
const specFormFields = computed<ProFormField[]>(() => [
  {
    prop: "name",
    label: "规格名称",
    component: "input",
    props: { placeholder: "请输入规格名称" }
  },
  {
    prop: "item",
    label: "规格内容",
    component: "dynamic-list",
    props: {
      inputProps: { placeholder: "请输入规格内容" }
    }
  },
  {
    prop: "sort",
    label: "排序",
    component: "input-number",
    props: { min: 1, precision: 0, step: 1, controlsPosition: "right", style: { width: "100%" } }
  }
]);

/** 打开规格弹窗，并在编辑态回填规格数据。 */
function handleOpenGoodsSpecDialog(index?: number, row?: GoodsSpec) {
  ensureSkuFormArrays();
  resetGoodsSpecForm();
  specDialog.visible = true;
  if (row) {
    const specFormData = JSON.parse(JSON.stringify(row));
    specDialog.title = "修改规格";
    specDialog.index = index ? index : 0;
    specDialog.specFormData = specFormData;
  } else {
    specDialog.title = "新增规格";
    specDialog.index = -1;
    specDialog.specFormData.id = 0;
  }
}

/** 提交规格表单并重建 SKU 组合。 */
async function handleGoodsSpecSubmit() {
  try {
    const isValid = await specFormRef.value?.validate();
    if (!isValid) return;

    ensureSkuFormArrays();
    const specFormData = JSON.parse(JSON.stringify(specDialog.specFormData));
    if (specDialog.index >= 0) {
      formData.value.specList[specDialog.index] = specFormData;
    } else {
      formData.value.specList.push(specFormData);
    }
    formData.value.specList.sort((item1: GoodsSpec, item2: GoodsSpec) => item1.sort - item2.sort);
    ElMessage.success("保存规格成功");
    generateSkuList();
    handleCloseGoodsSpecDialog();
  } catch {
    // FormDialog 校验失败时保持弹窗开启，交由表单展示错误提示。
  }
}

/** 重置规格弹窗表单，避免新增与编辑切换时残留旧值。 */
function resetGoodsSpecForm() {
  specFormRef.value?.resetFields();
  specFormRef.value?.clearValidate();
  specDialog.specFormData.id = 0;
  specDialog.specFormData.name = "";
  specDialog.specFormData.sort = 1;
  specDialog.specFormData.item = [""]; // 需要手动清空动态字段
}

/** 关闭规格弹窗并恢复默认表单状态。 */
function handleCloseGoodsSpecDialog() {
  specDialog.visible = false;
  resetGoodsSpecForm();
}

function handleDeleteGoodsSpec(index: number) {
  ensureSkuFormArrays();
  formData.value.specList.splice(index, 1);
  generateSkuList();
}

/**
 * 合并新旧SKU列表，旧数据的非specItem属性合并到包含其所有specItem值的新数据中
 * @param newArr 新SKU数组（主数组）
 * @param oldArr 旧SKU数组
 * @param mergeFn 自定义合并逻辑（可选）
 * @returns 合并后的新数组
 */
const mergeArraysByGoodsSpecItem = (
  newArr: SkuItem[],
  oldArr: SkuItem[],
  mergeFn?: (aVal: any, bVal: any, key: string) => any
): SkuItem[] => {
  // 创建新数组的拷贝以避免修改原数组
  const mergedNewArr = newArr.map(item => ({ ...item }));

  // 处理每个旧数据项
  oldArr.forEach(oldItem => {
    const oldSpecValues = getSpecItemValues(oldItem);
    // 查找所有包含旧数据规格值的新数据项
    mergedNewArr.forEach(newItem => {
      const newSpecValues = getSpecItemValues(newItem);
      if (isSubset(oldSpecValues, newSpecValues)) {
        mergeNonSpecProperties(newItem, oldItem, mergeFn);
      }
    });
  });

  return mergedNewArr;
};

/**
 * 提取对象中所有以specItem开头的属性值
 */
const getSpecItemValues = (item: SkuItem): Set<string> => {
  const values: string[] = [];
  Object.keys(item).forEach(key => {
    if (isDynamicSpecItemKey(key) && item[key] != null) {
      values.push(String(item[key]));
    }
  });
  return new Set(values);
};

/**
 * 判断subset是否为superset的子集
 */
const isSubset = (subset: Set<string>, superset: Set<string>): boolean => {
  for (const elem of subset) {
    if (!superset.has(elem)) return false;
  }
  return true;
};

/**
 * 将旧数据的非specItem属性合并到新数据中
 */
const mergeNonSpecProperties = (newItem: SkuItem, oldItem: SkuItem, mergeFn?: (aVal: any, bVal: any, key: string) => any) => {
  Object.keys(oldItem).forEach(key => {
    if (!isDynamicSpecItemKey(key) && key !== "specItem") {
      const oldVal = oldItem[key];
      if (key in newItem) {
        newItem[key] = mergeFn ? mergeFn(newItem[key], oldVal, key) : oldVal; // 默认用旧数据覆盖
      } else {
        newItem[key] = oldVal; // 新数据无该属性则添加
      }
    }
  });
};

// 迭代实现方案
function cartesianIterative<T>(arrays: T[][]): T[][] {
  return arrays
    .reduce<T[][]>((results, currentArray) => results.flatMap(prevComb => currentArray.map(item => [...prevComb, item])), [[]])
    .filter(comb => comb.length > 0); // 过滤初始空值
}

function generateSkuList() {
  ensureSkuFormArrays();
  // 如果规格为空，生成SKU列表为空
  if (formData.value.specList.length == 0) {
    formData.value.skuList = [];
    return;
  }
  const oldSkuList = formData.value.skuList;
  // 提取所有规格项组成二维数组
  const specItem = formData.value.specList.map((spec: GoodsSpec) => spec.item);
  const combinations = cartesianIterative(specItem);
  let newSkuList = [] as SkuItem[];
  combinations.map(comb => {
    let sku = {
      /** 商品SKUID */
      id: 0,
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
    } as SkuItem;
    comb.forEach((value, index) => {
      sku[`specItem${index}`] = value;
    });
    newSkuList.push(sku);
  });
  // 默认合并（覆盖重复属性）
  formData.value.skuList = mergeArraysByGoodsSpecItem(newSkuList, oldSkuList);
}

function handlePrev() {
  emit("prev");
}

/**
 * 商品表单提交
 */
async function submitForm() {
  ensureSkuFormArrays();

  if (!formData.value.skuList.length) {
    ElMessage.warning("请先添加商品规格并生成库存组合");
    return;
  }

  const hasInvalidSku = formData.value.skuList.some((item: SkuItem) => {
    const skuCode = String(item.skuCode ?? "").trim();
    const price = Number(item.price ?? 0);
    const discountPrice = Number(item.discountPrice ?? 0);
    const inventory = Number(item.inventory ?? 0);
    const initSaleNum = Number(item.initSaleNum ?? 0);
    return (
      !skuCode ||
      !Number.isFinite(price) ||
      price < 0.01 ||
      !Number.isFinite(discountPrice) ||
      discountPrice < 0.01 ||
      !Number.isInteger(inventory) ||
      inventory < 1 ||
      !Number.isInteger(initSaleNum) ||
      initSaleNum < 0
    );
  });
  if (hasInvalidSku) {
    ElMessage.warning("请完善商品库存信息后再提交");
    return;
  }

  // 重组商品的规格和SKU列表
  const submitsData = JSON.parse(JSON.stringify(formData.value));

  const skuList = submitsData.skuList;
  skuList.map((obj: SkuItem) => {
    const specItemMap: Record<string, any> = {};
    Object.entries(obj).forEach(([key, value]) => {
      if (isDynamicSpecItemKey(key)) {
        specItemMap[key] = value;
      }
      if (key.endsWith("rice")) {
        obj[key] = value * 100;
      }
    });
    const specItemList: string[] = [];
    Object.keys(specItemMap)
      .sort()
      .forEach(key => {
        const specItem = specItemMap[key];
        specItemList.push(String(specItem));
      });
    obj.specItem = specItemList;
  });

  try {
    const goodsId = submitsData.id;
    if (goodsId) {
      await defGoodsService.UpdateGoods(submitsData);
      ElNotification({
        title: "提示",
        message: "编辑商品成功",
        type: "success"
      });
      tabsStore.removeTabs(route.fullPath);
      return;
    }

    await defGoodsService.CreateGoods(submitsData);
    emit("resetForm");
    ElNotification({
      title: "提示",
      message: "新增商品成功",
      type: "success"
    });
    tabsStore.removeTabs(route.fullPath);
  } catch (error: any) {
    ElMessage.error(error?.message || "提交商品失败");
  }
}
</script>
<style scoped lang="scss">
.goods-edit-sku {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.goods-edit-sku__hero,
.goods-edit-sku__card {
  border: 1px solid var(--admin-page-card-border);
  border-radius: 16px;
  background: var(--admin-page-card-bg);
  box-shadow: var(--admin-page-shadow);
}

.goods-edit-sku__hero-content {
  display: grid;
  grid-template-columns: minmax(0, 1.2fr) minmax(220px, 0.8fr);
  gap: 24px;
  align-items: center;
}

.goods-edit-sku__title {
  margin: 0 0 10px;
  font-size: 20px;
  font-weight: 700;
  color: var(--admin-page-text-primary);
}

.goods-edit-sku__desc {
  margin: 0;
  font-size: 14px;
  line-height: 1.7;
  color: var(--admin-page-text-secondary);
}

.goods-edit-sku__summary {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
}

.goods-edit-sku__summary-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 16px;
  border: 1px solid var(--admin-page-card-border-soft);
  border-radius: 12px;
  background: var(--admin-page-card-bg-soft);
}

.goods-edit-sku__summary-item span {
  font-size: 13px;
  color: var(--admin-page-text-secondary);
}

.goods-edit-sku__summary-item strong {
  font-size: 20px;
  color: var(--admin-page-text-primary);
}

.goods-edit-sku__card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  font-size: 16px;
  font-weight: 600;
  color: var(--admin-page-text-primary);
}

.goods-edit-sku__tip {
  padding: 14px 16px;
  margin-bottom: 16px;
  font-size: 14px;
  line-height: 1.7;
  color: var(--admin-page-text-secondary);
  background: var(--admin-page-card-bg-soft);
  border: 1px solid var(--admin-page-card-border-soft);
  border-radius: 12px;
}

.goods-edit-sku__footer {
  display: flex;
  justify-content: space-between;
  gap: 12px;
}

.flex-items {
  display: flex;
  align-items: center;
  gap: 8px;
}

.flex-items :deep(.el-input) {
  flex: 1;
}

:deep(.el-form-item__content) {
  display: flex;
  overflow: visible;
}

@media (width <= 992px) {
  .goods-edit-sku__hero-content {
    grid-template-columns: 1fr;
  }
}

@media (width <= 768px) {
  .goods-edit-sku__summary {
    grid-template-columns: 1fr;
  }

  .goods-edit-sku__card-header,
  .goods-edit-sku__footer {
    flex-direction: column;
    align-items: stretch;
  }
}
</style>
