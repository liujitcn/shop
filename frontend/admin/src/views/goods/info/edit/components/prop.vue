<template>
  <div class="goods-edit-prop">
    <el-card class="goods-edit-prop__hero" shadow="never">
      <div class="goods-edit-prop__hero-content">
        <div>
          <div class="goods-edit-prop__eyebrow">第二步</div>
          <h2 class="goods-edit-prop__title">设置商品属性</h2>
          <p class="goods-edit-prop__desc">为商品补充可展示的属性信息，例如材质、产地、卖点说明等，排序越小越靠前。</p>
        </div>
        <div class="goods-edit-prop__summary">
          <div class="goods-edit-prop__summary-item">
            <span>当前属性数</span>
            <strong>{{ formData.propList?.length ?? 0 }}</strong>
          </div>
          <div class="goods-edit-prop__summary-item">
            <span>已完善属性</span>
            <strong>{{ completedPropCount }}</strong>
          </div>
        </div>
      </div>
    </el-card>

    <el-card class="goods-edit-prop__card" shadow="never">
      <template #header>
        <div class="goods-edit-prop__card-header">
          <span>属性编辑器</span>
          <el-button type="success" icon="plus" @click="handleAdd()">添加</el-button>
        </div>
      </template>

      <div class="goods-edit-prop__tip">建议将属性名称控制在简短可读的范围内，属性值可填写多行说明，排序越小越靠前。</div>

      <el-form ref="dataFormRef" :model="formData" :rules="rules" :inline="true">
        <ProTable row-key="sort" :data="formData.propList" :columns="columns" :pagination="false" :tool-button="false">
          <template #label="scope">
            <el-form-item :prop="'propList[' + scope.$index + '].label'" :rules="rules.label">
              <el-input v-model="scope.row.label" />
            </el-form-item>
          </template>

          <template #value="scope">
            <el-form-item :prop="'propList[' + scope.$index + '].value'" :rules="rules.value">
              <el-input v-model="scope.row.value" type="textarea" />
            </el-form-item>
          </template>

          <template #sort="scope">
            <el-form-item :prop="'propList[' + scope.$index + '].sort'" :rules="rules.sort">
              <el-input-number v-model="scope.row.sort" controls-position="right" :min="1" :precision="0" :step="1" />
            </el-form-item>
          </template>

          <template #operation="scope">
            <el-button type="danger" size="small" link icon="delete" @click.stop="handleRemove(scope.$index)"> 删除 </el-button>
          </template>
        </ProTable>
      </el-form>

      <template #footer>
        <div class="goods-edit-prop__footer">
          <el-button @click="handlePrev">上一步，填写商品信息</el-button>
          <el-button type="primary" @click="handleNext">下一步，设置商品库存</el-button>
        </div>
      </template>
    </el-card>
  </div>
</template>
<script setup lang="ts">
import { computed, reactive, ref, toRefs } from "vue";
import { ElMessage } from "element-plus";
import type { ColumnProps } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
defineOptions({
  name: "GoodsEditGoodsProp",
  inheritAttrs: false
});
const emit = defineEmits(["prev", "next", "update:modelValue"]);
const dataFormRef = ref();

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

/** 确保商品属性数组始终存在，避免编辑时空值报错。 */
function ensurePropList() {
  if (!Array.isArray(formData.value.propList)) {
    formData.value.propList = [];
  }
}

const state = reactive({
  rules: {
    label: [{ required: true, message: "请填写属性名称", trigger: "blur" }],
    value: [{ required: true, message: "请填写属性值", trigger: "blur" }],
    sort: [{ required: true, message: "请填写排序", trigger: "blur" }]
  }
});

const { rules } = toRefs(state);

/** 统计已填写完整的属性数量，帮助运营快速判断完成度。 */
const completedPropCount = computed(() => {
  return (formData.value.propList ?? []).filter((item: Record<string, unknown>) => {
    const label = String(item.label ?? "").trim();
    const value = String(item.value ?? "").trim();
    const sort = Number(item.sort ?? 0);
    return !!label && !!value && Number.isInteger(sort) && sort > 0;
  }).length;
});

/** 商品属性编辑表格列配置。 */
const columns: ColumnProps[] = [
  { type: "index", width: 50 },
  { prop: "label", label: "属性名称", minWidth: 140 },
  { prop: "value", label: "属性值", minWidth: 160 },
  { prop: "sort", label: "排序", minWidth: 220 },
  { prop: "operation", label: "操作", width: 150, align: "center" }
];

function handleAdd() {
  ensurePropList();
  formData.value.propList.push({
    sort: 1
  });
}

function handleRemove(index: number) {
  ensurePropList();
  formData.value.propList.splice(index, 1);
}

function handlePrev() {
  emit("prev");
}

async function handleNext() {
  ensurePropList();

  const hasInvalidProp = formData.value.propList.some((item: Record<string, unknown>) => {
    const label = String(item.label ?? "").trim();
    const value = String(item.value ?? "").trim();
    const sort = Number(item.sort ?? 0);
    return !label || !value || !Number.isInteger(sort) || sort < 1;
  });

  if (hasInvalidProp) {
    ElMessage.warning("请完善商品属性后再设置商品库存");
    return;
  }

  emit("next");
}
</script>

<style scoped lang="scss">
.goods-edit-prop {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.goods-edit-prop__hero,
.goods-edit-prop__card {
  border: 1px solid #e6ebf2;
  border-radius: 24px;
  box-shadow: 0 18px 40px rgb(15 23 42 / 6%);
}

.goods-edit-prop__hero {
  overflow: hidden;
  background: linear-gradient(135deg, rgb(255 255 255 / 98%) 0%, rgb(245 250 255 / 95%) 100%);
}

.goods-edit-prop__hero-content {
  display: grid;
  grid-template-columns: minmax(0, 1.2fr) minmax(220px, 0.8fr);
  gap: 24px;
  align-items: center;
}

.goods-edit-prop__eyebrow {
  font-size: 13px;
  font-weight: 600;
  letter-spacing: 0.08em;
  color: #5b6b83;
  text-transform: uppercase;
}

.goods-edit-prop__title {
  margin: 10px 0 12px;
  font-size: 28px;
  font-weight: 700;
  color: #1f2a37;
}

.goods-edit-prop__desc {
  margin: 0;
  font-size: 15px;
  line-height: 1.8;
  color: #526071;
}

.goods-edit-prop__summary-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 18px;
  border: 1px solid #e6ebf2;
  border-radius: 18px;
  background: rgb(255 255 255 / 82%);
}

.goods-edit-prop__summary-item span {
  font-size: 13px;
  color: #6b7a90;
}

.goods-edit-prop__summary-item strong {
  font-size: 22px;
  color: #1f2a37;
}

.goods-edit-prop__card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  font-size: 16px;
  font-weight: 600;
  color: #1f2a37;
}

.goods-edit-prop__tip {
  padding: 14px 16px;
  margin-bottom: 16px;
  font-size: 14px;
  line-height: 1.7;
  color: #526071;
  background: #f8fbff;
  border: 1px solid #e2edf8;
  border-radius: 16px;
}

.goods-edit-prop__footer {
  display: flex;
  justify-content: space-between;
  gap: 12px;
}

@media (width <= 992px) {
  .goods-edit-prop__hero-content {
    grid-template-columns: 1fr;
  }
}

@media (width <= 768px) {
  .goods-edit-prop__card-header,
  .goods-edit-prop__footer {
    flex-direction: column;
    align-items: stretch;
  }
}
</style>
