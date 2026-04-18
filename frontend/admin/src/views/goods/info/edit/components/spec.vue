<template>
  <div class="goods-edit-spec">
    <el-card class="goods-edit-spec__card" shadow="never">
      <div class="goods-edit-spec__actions">
        <el-button type="success" icon="plus" @click="handleOpenGoodsSpecDialog()">添加规格</el-button>
      </div>

      <ProTable :data="formData.specList" :columns="specColumns" :pagination="false" :tool-button="false">
        <template #operation="scope">
          <el-button type="primary" size="small" link icon="edit" @click="handleOpenGoodsSpecDialog(scope.$index, scope.row)">
            编辑
          </el-button>
          <el-button type="danger" size="small" link icon="delete" @click="handleDeleteGoodsSpec(scope.$index)"> 删除 </el-button>
        </template>
      </ProTable>

      <template #footer>
        <div class="goods-edit-spec__footer">
          <el-button @click="handlePrev">上一步</el-button>
          <el-button type="primary" @click="handleNext">下一步</el-button>
        </div>
      </template>
    </el-card>

    <FormDialog
      v-model="specDialog.visible"
      ref="specFormRef"
      :title="specDialog.title"
      width="500px"
      :model="specDialog.specFormData"
      :fields="specFormFields"
      :rules="specDialog.rules"
      label-width="96px"
      @confirm="handleGoodsSpecSubmit"
      @close="handleCloseGoodsSpecDialog"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from "vue";
import { ElMessage } from "element-plus";
import type { ColumnProps } from "@/components/ProTable/interface";
import type { ProFormField } from "@/components/ProForm/interface";
import ProTable from "@/components/ProTable/index.vue";
import FormDialog from "@/components/Dialog/FormDialog.vue";
import type { GoodsSpec } from "@/rpc/admin/goods_spec";

defineOptions({
  name: "GoodsEditSpec",
  inheritAttrs: false
});

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

const emit = defineEmits(["prev", "next", "update:modelValue"]);
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

/** 判断当前字段是否为动态生成的规格项字段。 */
const isDynamicSpecItemKey = (key: string) => /^specItem\d+$/.test(key);

/** 确保规格和 SKU 数组始终存在，避免编辑态出现空值报错。 */
function ensureSpecFormArrays() {
  if (!Array.isArray(formData.value.specList)) {
    formData.value.specList = [];
  }
  if (!Array.isArray(formData.value.skuList)) {
    formData.value.skuList = [];
  }
}

/** 商品规格表格列配置。 */
const specColumns: ColumnProps[] = [
  { type: "index", width: 50 },
  { prop: "name", label: "名称", minWidth: 160 },
  {
    prop: "item",
    label: "规格值",
    minWidth: 220,
    render: scope => (Array.isArray(scope.row.item) ? scope.row.item.join(" / ") : "--")
  },
  { prop: "sort", label: "排序", minWidth: 100, align: "right" },
  { prop: "operation", label: "操作", width: 150, align: "center" }
];

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
          // 规格内容至少保留一项，并且每一项都必须有值。
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
    label: "规格值",
    component: "dynamic-list",
    props: {
      inputProps: { placeholder: "请输入规格值" }
    }
  },
  {
    prop: "sort",
    label: "排序",
    component: "input-number",
    props: { min: 1, precision: 0, step: 1, controlsPosition: "right", style: { width: "100%" } }
  }
]);

/** 打开规格弹窗，并在编辑态回填已有规格。 */
function handleOpenGoodsSpecDialog(index?: number, row?: GoodsSpec) {
  ensureSpecFormArrays();
  resetGoodsSpecForm();
  specDialog.visible = true;

  if (!row) {
    specDialog.title = "新增规格";
    specDialog.index = -1;
    return;
  }

  specDialog.title = "编辑规格";
  specDialog.index = index ?? -1;
  specDialog.specFormData = JSON.parse(JSON.stringify(row));
}

/** 提交规格弹窗，并在保存后重建 SKU 组合。 */
async function handleGoodsSpecSubmit() {
  try {
    const isValid = await specFormRef.value?.validate();
    if (!isValid) return;

    ensureSpecFormArrays();
    const specFormData = JSON.parse(JSON.stringify(specDialog.specFormData));
    if (specDialog.index >= 0) {
      formData.value.specList[specDialog.index] = specFormData;
    } else {
      formData.value.specList.push(specFormData);
    }

    formData.value.specList.sort((item1: GoodsSpec, item2: GoodsSpec) => item1.sort - item2.sort);
    generateSkuList();
    ElMessage.success("保存规格成功");
    handleCloseGoodsSpecDialog();
  } catch {
    // 表单校验失败时保留弹窗，交由表单项展示错误。
  }
}

/** 删除规格后同步刷新 SKU 组合。 */
function handleDeleteGoodsSpec(index: number) {
  ensureSpecFormArrays();
  formData.value.specList.splice(index, 1);
  generateSkuList();
}

/** 关闭规格弹窗并恢复默认表单状态。 */
function handleCloseGoodsSpecDialog() {
  specDialog.visible = false;
  resetGoodsSpecForm();
}

/** 重置规格弹窗表单，避免新增和编辑切换时残留旧值。 */
function resetGoodsSpecForm() {
  specFormRef.value?.resetFields();
  specFormRef.value?.clearValidate();
  specDialog.specFormData.id = 0;
  specDialog.specFormData.name = "";
  specDialog.specFormData.sort = 1;
  specDialog.specFormData.item = [""];
}

/**
 * 合并新旧 SKU 列表。
 * 新规格组合生成后，旧组合里已经填写过的价格、库存和图片会尽量迁移过来。
 */
const mergeArraysByGoodsSpecItem = (
  newArr: SkuItem[],
  oldArr: SkuItem[],
  mergeFn?: (aVal: any, bVal: any, key: string) => any
): SkuItem[] => {
  const mergedNewArr = newArr.map(item => ({ ...item }));

  oldArr.forEach(oldItem => {
    const oldSpecValues = getSpecItemValues(oldItem);
    mergedNewArr.forEach(newItem => {
      const newSpecValues = getSpecItemValues(newItem);
      if (isSubset(oldSpecValues, newSpecValues)) {
        mergeNonSpecProperties(newItem, oldItem, mergeFn);
      }
    });
  });

  return mergedNewArr;
};

/** 提取当前 SKU 行里所有规格值，便于后续比对旧组合和新组合。 */
const getSpecItemValues = (item: SkuItem): Set<string> => {
  const values: string[] = [];
  Object.keys(item).forEach(key => {
    if (isDynamicSpecItemKey(key) && item[key] != null) {
      values.push(String(item[key]));
    }
  });
  return new Set(values);
};

/** 判断旧规格集合是否仍然包含在新规格组合中。 */
const isSubset = (subset: Set<string>, superset: Set<string>): boolean => {
  for (const elem of subset) {
    if (!superset.has(elem)) return false;
  }
  return true;
};

/** 合并旧 SKU 行里的非规格字段，保留运营已经填写过的业务数据。 */
const mergeNonSpecProperties = (newItem: SkuItem, oldItem: SkuItem, mergeFn?: (aVal: any, bVal: any, key: string) => any) => {
  Object.keys(oldItem).forEach(key => {
    if (!isDynamicSpecItemKey(key) && key !== "specItem") {
      const oldVal = oldItem[key];
      if (key in newItem) {
        newItem[key] = mergeFn ? mergeFn(newItem[key], oldVal, key) : oldVal;
      } else {
        newItem[key] = oldVal;
      }
    }
  });
};

/** 使用迭代方式生成规格值的笛卡尔积，避免递归实现过深。 */
function cartesianIterative<T>(arrays: T[][]): T[][] {
  return arrays
    .reduce<T[][]>((results, currentArray) => results.flatMap(prevComb => currentArray.map(item => [...prevComb, item])), [[]])
    .filter(comb => comb.length > 0);
}

/** 按当前规格定义重新生成 SKU 列表。 */
function generateSkuList() {
  ensureSpecFormArrays();
  if (!formData.value.specList.length) {
    formData.value.skuList = [];
    return;
  }

  const oldSkuList = formData.value.skuList;
  const specItem = formData.value.specList.map((spec: GoodsSpec) => spec.item);
  const combinations = cartesianIterative(specItem);
  const newSkuList: SkuItem[] = [];

  combinations.forEach(comb => {
    const sku = {
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

  formData.value.skuList = mergeArraysByGoodsSpecItem(newSkuList, oldSkuList);
}

/** 返回上一步。 */
function handlePrev() {
  emit("prev");
}

/** 校验规格配置后进入库存步骤。 */
function handleNext() {
  ensureSpecFormArrays();

  if (!formData.value.specList.length) {
    ElMessage.warning("请先添加商品规格");
    return;
  }

  emit("next");
}
</script>

<style scoped lang="scss">
.goods-edit-spec__card {
  border: 1px solid var(--admin-page-card-border);
  border-radius: 16px;
  background: var(--admin-page-card-bg);
  box-shadow: var(--admin-page-shadow);
}

:deep(.goods-edit-spec__card .el-card__body) {
  padding-top: 18px;
}

.goods-edit-spec__actions {
  display: flex;
  justify-content: flex-end;
  margin-bottom: 16px;
}

.goods-edit-spec__footer {
  display: flex;
  justify-content: space-between;
  gap: 12px;
}

@media (width <= 768px) {
  .goods-edit-spec__footer {
    flex-direction: column;
  }
}
</style>
