<template>
  <div class="app-container">
    <el-card shadow="never">
      <div class="mb-10px">
        <el-button type="success" icon="plus" @click="handleOpenGoodsSpecDialog()">添加</el-button>
      </div>
      <ProTable :data="formData.specList" :columns="specColumns" :pagination="false" :tool-button="false">
        <template #operation="scope">
          <el-button type="primary" size="small" link icon="edit" @click="handleOpenGoodsSpecDialog(scope.$index, scope.row)">
            编辑
          </el-button>
          <el-button type="danger" size="small" link icon="delete" @click="handleDeleteGoodsSpec(scope.$index)"> 删除 </el-button>
        </template>
      </ProTable>
    </el-card>

    <el-card shadow="never">
      <template #header>
        <span>商品库存</span>
      </template>
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
        <el-button @click="handlePrev">上一步，设置商品属性</el-button>
        <el-button type="primary" @click="submitForm">提交</el-button>
      </template>
    </el-card>

    <!-- 规格表单弹窗 -->
    <ProDialog v-model="specDialog.visible" :title="specDialog.title" width="500px" @close="handleCloseGoodsSpecDialog">
      <el-form ref="specFormRef" :model="specDialog.specFormData" :rules="specDialog.rules" label-suffix=":" label-width="100px">
        <el-form-item label="规格名称" prop="name">
          <el-input v-model="specDialog.specFormData.name" placeholder="请输入规格名称" />
        </el-form-item>
        <!-- 规格内容（动态可增减） -->
        <el-form-item
          v-for="(item, index) in specDialog.specFormData.item"
          :key="index"
          :label="`规格项 ${index + 1}`"
          :prop="`item.${index}`"
          :rules="{ required: true, message: '规格项不能为空', trigger: 'blur' }"
        >
          <div class="flex-items">
            <el-input v-model="specDialog.specFormData.item[index]" placeholder="请输入规格内容" />
            <el-button
              type="danger"
              circle
              :disabled="specDialog.specFormData.item.length === 1"
              class="ml-2"
              @click="removeGoodsSpecItem(index)"
            >
              <el-icon><Delete /></el-icon>
            </el-button>
          </div>
        </el-form-item>
        <el-form-item label="排序" prop="sort">
          <el-input-number v-model="specDialog.specFormData.sort" controls-position="right" :min="1" :precision="0" :step="1" />
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button type="primary" @click="handleGoodsSpecSubmit">确定</el-button>
          <el-button @click="addGoodsSpecItem">添加规格</el-button>
          <el-button @click="handleCloseGoodsSpecDialog">取消</el-button>
        </div>
      </template>
    </ProDialog>
  </div>
</template>
<script setup lang="ts">
import { computed, reactive, ref, toRefs } from "vue";
import { useRoute } from "vue-router";
import { ElMessage, ElNotification } from "element-plus";
import { Delete } from "@element-plus/icons-vue";
import type { ColumnProps } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
import ProDialog from "@/components/Dialog/ProDialog.vue";
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
const specFormRef = ref();

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

const specColumns: ColumnProps[] = [
  { type: "index", width: 50 },
  { prop: "name", label: "规格名称" },
  { prop: "item", label: "规格内容" },
  { prop: "sort", label: "排序", width: 100, align: "right" },
  { prop: "operation", label: "操作", width: 150, align: "center" }
];

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
    { prop: "price", label: "价格（元）", align: "center", width: 150 },
    { prop: "discountPrice", label: "折扣价格（元）", align: "center", width: 150 },
    { prop: "inventory", label: "库存", align: "center", width: 140 },
    { prop: "picture", label: "规格图片", align: "center", width: 180 },
    { prop: "initSaleNum", label: "初始销量", align: "center", width: 140 }
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
    name: [{ required: true, message: "请输入规格名称", trigger: "blur" }]
  }
});

/** 添加规格项输入框。 */
const addGoodsSpecItem = () => {
  specDialog.specFormData.item.push("");
};

/** 删除指定规格项输入框。 */
const removeGoodsSpecItem = (index: number) => {
  specDialog.specFormData.item.splice(index, 1);
};

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
function handleGoodsSpecSubmit() {
  specFormRef.value.validate((valid: any) => {
    if (valid) {
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
    }
  });
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
    console.log("oldSpecValues", oldSpecValues);
    // 查找所有包含旧数据规格值的新数据项
    mergedNewArr.forEach(newItem => {
      const newSpecValues = getSpecItemValues(newItem);
      console.log("newSpecValues", oldSpecValues);
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

  console.log(oldSkuList);
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
  console.log(newSkuList);
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
<style scoped>
/* 新增样式 */
.flex-items {
  display: flex;
  align-items: center;
  gap: 8px; /* 元素间距 */
}

/* 让输入框自动填充剩余空间 */
.flex-items :deep(.el-input) {
  flex: 1;
}

/* 调整表单内容区域布局 */
:deep(.el-form-item__content) {
  display: flex;
  overflow: visible; /* 解决布局溢出问题 */
}
</style>
