<template>
  <div class="app-container">
    <el-card shadow="never">
      <div class="mb-10px">
        <el-button type="success" icon="plus" @click="handleOpenGoodsSpecDialog()">添加</el-button>
      </div>
      <el-table :data="formData.specList" highlight-current-row border>
        <el-table-column type="index" width="50" />
        <el-table-column label="规格名称" prop="name" />
        <el-table-column label="规格内容" prop="item" />
        <el-table-column label="排序" prop="sort" />
        <el-table-column label="操作" width="150" align="center">
          <template #default="scope">
            <el-button
              type="primary"
              size="small"
              link
              icon="edit"
              @click="handleOpenGoodsSpecDialog(scope.$index, scope.row)"
            >
              编辑
            </el-button>
            <el-button
              type="danger"
              size="small"
              link
              icon="delete"
              @click="handleDeleteGoodsSpec(scope.$index)"
            >
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-card shadow="never">
      <template #header>
        <span>商品库存</span>
      </template>
      <el-form ref="skuFormRef" :model="formData" size="small" :inline="true">
        <el-table :data="formData.skuList" highlight-current-row border>
          <el-table-column type="index" width="50" />
          <el-table-column
            v-for="(item, index) in formData.specList"
            :key="index"
            align="center"
            :prop="'specItem' + index"
            :label="item.name"
          />

          <el-table-column label="规格编号" align="center">
            <template #default="scope">
              <el-form-item :prop="'skuList[' + scope.$index + '].skuCode'" :rules="rules.skuCode">
                <el-input v-model="scope.row.skuCode" />
              </el-form-item>
            </template>
          </el-table-column>

          <el-table-column label="价格（元）" align="center">
            <template #default="scope">
              <el-form-item :prop="'skuList[' + scope.$index + '].price'" :rules="rules.price">
                <el-input-number
                  v-model="scope.row.price"
                  :min="0.01"
                  :precision="2"
                  :step="0.01"
                />
              </el-form-item>
            </template>
          </el-table-column>

          <el-table-column label="折扣价格（元）" align="center">
            <template #default="scope">
              <el-form-item
                :prop="'skuList[' + scope.$index + '].discountPrice'"
                :rules="rules.discountPrice"
              >
                <el-input-number
                  v-model="scope.row.discountPrice"
                  :min="0.01"
                  :precision="2"
                  :step="0.01"
                />
              </el-form-item>
            </template>
          </el-table-column>

          <el-table-column label="库存" align="center">
            <template #default="scope">
              <el-form-item
                :prop="'skuList[' + scope.$index + '].inventory'"
                :rules="rules.inventory"
              >
                <el-input-number
                  v-model="scope.row.inventory"
                  controls-position="right"
                  :min="1"
                  :precision="0"
                  :step="1"
                />
              </el-form-item>
            </template>
          </el-table-column>

          <el-table-column label="规格图片" align="center">
            <template #default="scope">
              <el-form-item :prop="'skuList[' + scope.$index + '].picture'">
                <single-image-upload v-model="scope.row.picture" file-type="goods" />
              </el-form-item>
            </template>
          </el-table-column>

          <el-table-column label="初始销量" align="center">
            <template #default="scope">
              <el-form-item :prop="'skuList[' + scope.$index + '].initSaleNum'">
                <el-input-number
                  v-model="scope.row.initSaleNum"
                  controls-position="right"
                  :min="0"
                  :precision="0"
                  :step="1"
                />
              </el-form-item>
            </template>
          </el-table-column>
        </el-table>
      </el-form>
      <template #footer>
        <el-button @click="handlePrev">上一步，设置商品属性</el-button>
        <el-button type="primary" @click="submitForm">提交</el-button>
      </template>
    </el-card>

    <!-- 规格表单弹窗 -->
    <el-dialog
      v-model="specDialog.visible"
      :title="specDialog.title"
      width="500px"
      @close="handleCloseGoodsSpecDialog"
    >
      <el-form
        ref="specFormRef"
        :model="specDialog.specFormData"
        :rules="specDialog.rules"
        label-suffix=":"
        label-width="100px"
      >
        <el-form-item label="规格名称" prop="name">
          <el-input
            v-model="specDialog.specFormData.name"
            placeholder="请输入规格名称"
          />
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
          <el-input-number
            v-model="specDialog.specFormData.sort"
            controls-position="right"
            :min="1"
            :precision="0"
            :step="1"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button type="primary" @click="handleGoodsSpecSubmit">确定</el-button>
          <el-button @click="addGoodsSpecItem">添加规格</el-button>
          <el-button @click="handleCloseGoodsSpecDialog">取消</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>
<script setup lang="ts">
import { Delete } from "@element-plus/icons-vue";
import { defGoodsService } from "@/api/admin/goods";
import type { GoodsSpec } from "@/rpc/admin/goods_spec";
import { useRouter } from "vue-router";
defineOptions({
  name: "GoodsEditSku",
  inheritAttrs: false,
});
const emit = defineEmits(["prev", "next", "update:modelValue", "resetForm"]);

const router = useRouter();

const skuFormRef = ref(ElForm);
const specFormRef = ref(ElForm);

const props = defineProps({
  modelValue: {
    type: Object,
    default: () => {},
  },
});

const formData: any = computed({
  get: () => props.modelValue,
  set: (value) => {
    emit("update:modelValue", value);
  },
});

const state = reactive({
  rules: {
    skuCode: [{ required: true, message: "请输入商品编号", trigger: "blur" }],
    price: [{ required: true, message: "请输入商品价格", trigger: "blur" }],
    discountPrice: [{ required: true, message: "请输入折扣价格", trigger: "blur" }],
    inventory: [{ required: true, message: "请输入商品库存", trigger: "blur" }],
  },
});

const { rules } = toRefs(state);

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
    sort: 1,
  } as GoodsSpec,
  rules: {
    name: [{ required: true, message: "请输入规格名称", trigger: "blur" }],
  },
});

// 添加规格项
const addGoodsSpecItem = () => {
  specDialog.specFormData.item.push("");
};

// 删除规格项
const removeGoodsSpecItem = (index: number) => {
  specDialog.specFormData.item.splice(index, 1);
};

// 打开规格弹窗
function handleOpenGoodsSpecDialog(index?: number, row?: GoodsSpec) {
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

// 规格表单提交
function handleGoodsSpecSubmit() {
  specFormRef.value.validate((valid: any) => {
    if (valid) {
      const specFormData = JSON.parse(JSON.stringify(specDialog.specFormData));
      if (specDialog.index >= 0) {
        formData.value.specList[specDialog.index] = specFormData;
      } else {
        formData.value.specList.push(specFormData);
      }
      formData.value.specList.sort((item1: GoodsSpec, item2: GoodsSpec) => item1.sort - item2.sort);
      ElMessage.success("保存成功");
      generateSkuList();
      handleCloseGoodsSpecDialog();
    }
  });
}

// 重置表单
function resetGoodsSpecForm() {
  specFormRef.value.resetFields();
  specFormRef.value.clearValidate();
  specDialog.specFormData.id = 0;
  specDialog.specFormData.sort = 1;
  specDialog.specFormData.item = [""]; // 需要手动清空动态字段
}

// 关闭规格弹窗
function handleCloseGoodsSpecDialog() {
  specDialog.visible = false;
  resetGoodsSpecForm();
}

function handleDeleteGoodsSpec(index: number) {
  formData.value.specList.splice(index, 1);
  generateSkuList();
}

// 假设对象包含以 specItem 开头的动态键和其他属性
type SkuItem = Record<string, any>;
/**
 * 根据以 specItem 开头的键值生成唯一标识
 * @param item 对象
 * @returns 格式如 "specItem1=red&specItem2=large" 的字符串
 */
const getGoodsSpecItemKey = (item: SkuItem): string => {
  const specEntries = Object.entries(item)
    .filter(([key]) => key.startsWith("specItem")) // 提取以 specItem 开头的键
    .sort(([a], [b]) => a.localeCompare(b)); // 按键名排序，避免顺序不同导致 Key 不一致

  return specEntries.map(([k, v]) => `${k}=${v}`).join("&");
};

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
  const mergedNewArr = newArr.map((item) => ({ ...item }));

  // 处理每个旧数据项
  oldArr.forEach((oldItem) => {
    const oldSpecValues = getSpecItemValues(oldItem);
    console.log("oldSpecValues", oldSpecValues);
    // 查找所有包含旧数据规格值的新数据项
    mergedNewArr.forEach((newItem) => {
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
  Object.keys(item).forEach((key) => {
    if (key.startsWith("specItem") && item[key] != null) {
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
const mergeNonSpecProperties = (
  newItem: SkuItem,
  oldItem: SkuItem,
  mergeFn?: (aVal: any, bVal: any, key: string) => any
) => {
  Object.keys(oldItem).forEach((key) => {
    if (!key.startsWith("specItem")) {
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
    .reduce<
      T[][]
    >((results, currentArray) => results.flatMap((prevComb) => currentArray.map((item) => [...prevComb, item])), [[]])
    .filter((comb) => comb.length > 0); // 过滤初始空值
}

function generateSkuList() {
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
  combinations.map((comb) => {
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
      inventory: 0,
    } as SkuItem;
    let valueList = [];
    comb.forEach((value, index) => {
      sku[`specItem${index}`] = value;
      valueList.push(value);
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
function submitForm() {
  skuFormRef.value.validate((skuValid: any) => {
    if (skuValid) {
      // 重组商品的规格和SKU列表
      let submitsData = JSON.parse(JSON.stringify(formData.value));

      // 编辑sku
      const skuList = submitsData.skuList;
      skuList.map((obj: SkuItem) => {
        let specItemMap: Record<string, any> = {};
        Object.entries(obj).forEach(([key, value]) => {
          if (key.startsWith("specItem")) {
            specItemMap[key] = value;
          }
          if (key.endsWith("rice")) {
            obj[key] = value * 100;
          }
        });
        let specItemList: string[] = [];
        Object.keys(specItemMap)
          .sort()
          .forEach((key) => {
            const specItem = specItemMap[key];
            specItemList.push(specItem);
          });
        obj["specItem"] = specItemList;
      });

      const goodsId = submitsData.id;
      if (goodsId) {
        // 编辑商品提交
        defGoodsService.UpdateGoods(submitsData).then(() => {
          emit("resetForm");
          router.go(-1);
          ElNotification({
            title: "提示",
            message: "编辑商品成功",
            type: "success",
          });
        });
      } else {
        // 新增商品提交
        defGoodsService.CreateGoods(submitsData).then(() => {
          emit("resetForm");
          router.go(-1);
          ElNotification({
            title: "提示",
            message: "新增商品成功",
            type: "success",
          });
        });
      }
    }
  });
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
