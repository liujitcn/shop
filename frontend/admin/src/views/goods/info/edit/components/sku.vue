<template>
  <div class="goods-edit-stock">
    <el-card class="goods-edit-stock__card" shadow="never">
      <el-form ref="skuFormRef" :model="formData" size="small">
        <ProTable :key="skuTableKey" :data="formData.sku_list" :columns="skuColumns" :pagination="false" :tool-button="false">
          <template #sku_code="scope">
            <el-form-item :prop="'sku_list[' + scope.$index + '].sku_code'" :rules="rules.sku_code">
              <el-input v-model="scope.row.sku_code" placeholder="请输入规格编号" />
            </el-form-item>
          </template>

          <template #price="scope">
            <el-form-item :prop="'sku_list[' + scope.$index + '].price'" :rules="rules.price">
              <el-input-number v-model="scope.row.price" :min="0.01" :precision="2" :step="0.01" />
            </el-form-item>
          </template>

          <template #discount_price="scope">
            <el-form-item :prop="'sku_list[' + scope.$index + '].discount_price'" :rules="rules.discount_price">
              <el-input-number v-model="scope.row.discount_price" :min="0.01" :precision="2" :step="0.01" />
            </el-form-item>
          </template>

          <template #inventory="scope">
            <el-form-item :prop="'sku_list[' + scope.$index + '].inventory'" :rules="rules.inventory">
              <el-input-number v-model="scope.row.inventory" controls-position="right" :min="1" :precision="0" :step="1" />
            </el-form-item>
          </template>

          <template #picture="scope">
            <el-form-item :prop="'sku_list[' + scope.$index + '].picture'">
              <UploadImg v-model:image-url="scope.row.picture" upload-type="goods" />
            </el-form-item>
          </template>

          <template #init_sale_num="scope">
            <el-form-item :prop="'sku_list[' + scope.$index + '].init_sale_num'">
              <el-input-number v-model="scope.row.init_sale_num" controls-position="right" :min="0" :precision="0" :step="1" />
            </el-form-item>
          </template>
        </ProTable>
      </el-form>

      <template #footer>
        <div class="goods-edit-stock__footer">
          <el-button @click="handlePrev">上一步</el-button>
          <el-button type="primary" @click="submitForm">保存商品</el-button>
        </div>
      </template>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, toRefs } from "vue";
import { useRoute } from "vue-router";
import { ElMessage, ElNotification } from "element-plus";
import type { ColumnProps } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
import UploadImg from "@/components/Upload/Img.vue";
import { defGoodsInfoService } from "@/api/admin/goods_info";
import type { GoodsSpec } from "@/rpc/admin/v1/goods_spec";
import { useTabsStore } from "@/stores/modules/tabs";

defineOptions({
  name: "GoodsEditStock",
  inheritAttrs: false
});

type SkuItem = Record<string, any> & {
  id: number;
  sku_code: string;
  price: number;
  discount_price: number;
  init_sale_num: number;
  real_sale_num: number;
  inventory: number;
  picture?: string;
  spec_item?: string[];
};

const emit = defineEmits(["prev", "update:modelValue", "resetForm"]);
const route = useRoute();
const tabsStore = useTabsStore();
const skuFormRef = ref();

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
const isDynamicSpecItemKey = (key: string) => /^spec_item\d+$/.test(key);

/** 确保规格和 SKU 数组始终存在，避免编辑态出现空值报错。 */
function ensureSkuFormArrays() {
  if (!Array.isArray(formData.value.spec_list)) {
    formData.value.spec_list = [];
  }
  if (!Array.isArray(formData.value.sku_list)) {
    formData.value.sku_list = [];
  }
}

const state = reactive({
  rules: {
    sku_code: [{ required: true, message: "请输入规格编号", trigger: "blur" }],
    price: [{ required: true, message: "请输入售价", trigger: "blur" }],
    discount_price: [{ required: true, message: "请输入折后价", trigger: "blur" }],
    inventory: [{ required: true, message: "请输入库存", trigger: "blur" }]
  }
});

const { rules } = toRefs(state);

/** 商品 SKU 编辑表格列配置。 */
const skuColumns = computed<ColumnProps[]>(() => {
  const dynamicSpecColumns = (formData.value.spec_list ?? []).map((item: GoodsSpec, index: number) => ({
    prop: `spec_item${index}`,
    label: item.name,
    align: "center",
    minWidth: 120
  }));

  return [
    { type: "index", width: 50 },
    ...dynamicSpecColumns,
    { prop: "sku_code", label: "规格编号", align: "center", minWidth: 160 },
    { prop: "price", label: "售价", align: "center", minWidth: 140 },
    { prop: "discount_price", label: "折后价", align: "center", minWidth: 140 },
    { prop: "inventory", label: "库存", align: "center", minWidth: 120 },
    { prop: "picture", label: "规格图", align: "center", minWidth: 180 },
    { prop: "init_sale_num", label: "初始销量", align: "center", minWidth: 140 }
  ];
});

const skuTableKey = computed(() =>
  (formData.value.spec_list ?? []).map((item: GoodsSpec) => `${item.name}:${(item.item ?? []).join(",")}`).join("|")
);

/** 返回上一步。 */
function handlePrev() {
  emit("prev");
}

/** 提交商品信息。 */
async function submitForm() {
  ensureSkuFormArrays();

  if (!formData.value.spec_list.length || !formData.value.sku_list.length) {
    ElMessage.warning("请先完成规格项配置");
    return;
  }

  const hasInvalidSku = formData.value.sku_list.some((item: SkuItem) => {
    const sku_code = String(item.sku_code ?? "").trim();
    const price = Number(item.price ?? 0);
    const discount_price = Number(item.discount_price ?? 0);
    const inventory = Number(item.inventory ?? 0);
    const init_sale_num = Number(item.init_sale_num ?? 0);
    return (
      !sku_code ||
      !Number.isFinite(price) ||
      price < 0.01 ||
      !Number.isFinite(discount_price) ||
      discount_price < 0.01 ||
      !Number.isInteger(inventory) ||
      inventory < 1 ||
      !Number.isInteger(init_sale_num) ||
      init_sale_num < 0
    );
  });
  if (hasInvalidSku) {
    ElMessage.warning("请先完善库存信息");
    return;
  }

  const submitsData = JSON.parse(JSON.stringify(formData.value));
  const sku_list = submitsData.sku_list;

  // 提交前把动态规格列重新收敛成接口需要的 spec_item 数组。
  sku_list.forEach((obj: SkuItem) => {
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
        const spec_item = specItemMap[key];
        specItemList.push(String(spec_item));
      });
    obj.spec_item = specItemList;
  });

  try {
    const goodsId = submitsData.id;
    if (goodsId) {
      await defGoodsInfoService.UpdateGoodsInfo({ id: submitsData.id, goods_info: submitsData });
      ElNotification({
        title: "提示",
        message: "编辑商品成功",
        type: "success"
      });
      tabsStore.removeTabs(route.fullPath);
      return;
    }

    await defGoodsInfoService.CreateGoodsInfo({ goods_info: submitsData });
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
.goods-edit-stock__card {
  border: 1px solid var(--admin-page-card-border);
  border-radius: var(--admin-page-radius);
  background: var(--admin-page-card-bg);
  box-shadow: var(--admin-page-shadow);
}

:deep(.goods-edit-stock__card .el-card__body) {
  padding-top: 18px;
}

.goods-edit-stock__footer {
  display: flex;
  justify-content: space-between;
  gap: 12px;
}

:deep(.el-form-item__content) {
  display: flex;
  overflow: visible;
}

@media (width <= 768px) {
  .goods-edit-stock__footer {
    flex-direction: column;
  }
}
</style>
