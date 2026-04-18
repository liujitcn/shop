<template>
  <div class="goods-edit-prop">
    <el-card class="goods-edit-prop__card" shadow="never">
      <div class="goods-edit-prop__actions">
        <el-button type="success" icon="plus" @click="handleAdd()">添加属性</el-button>
      </div>

      <el-form ref="dataFormRef" :model="formData" :rules="rules" :inline="true">
        <ProTable row-key="sort" :data="formData.propList" :columns="columns" :pagination="false" :tool-button="false">
          <template #label="scope">
            <el-form-item :prop="'propList[' + scope.$index + '].label'" :rules="rules.label">
              <el-input v-model="scope.row.label" placeholder="请输入属性名称" />
            </el-form-item>
          </template>

          <template #value="scope">
            <el-form-item :prop="'propList[' + scope.$index + '].value'" :rules="rules.value">
              <el-input v-model="scope.row.value" type="textarea" :rows="2" placeholder="请输入属性内容" resize="none" />
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
          <el-button @click="handlePrev">上一步</el-button>
          <el-button type="primary" @click="handleNext">下一步</el-button>
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
  name: "GoodsEditProp",
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
    value: [{ required: true, message: "请填写属性内容", trigger: "blur" }],
    sort: [{ required: true, message: "请填写排序", trigger: "blur" }]
  }
});

const { rules } = toRefs(state);

/** 商品属性编辑表格列配置。 */
const columns: ColumnProps[] = [
  { type: "index", width: 50 },
  { prop: "label", label: "名称", minWidth: 160 },
  { prop: "value", label: "内容", minWidth: 220 },
  { prop: "sort", label: "排序", minWidth: 160 },
  { prop: "operation", label: "操作", width: 120, align: "center" }
];

/** 新增一行空属性。 */
function handleAdd() {
  ensurePropList();
  formData.value.propList.push({
    sort: 1
  });
}

/** 删除指定属性。 */
function handleRemove(index: number) {
  ensurePropList();
  formData.value.propList.splice(index, 1);
}

/** 返回上一步。 */
function handlePrev() {
  emit("prev");
}

/** 校验属性后进入规格步骤。 */
function handleNext() {
  ensurePropList();

  const hasInvalidProp = formData.value.propList.some((item: Record<string, unknown>) => {
    const label = String(item.label ?? "").trim();
    const value = String(item.value ?? "").trim();
    const sort = Number(item.sort ?? 0);
    return !label || !value || !Number.isInteger(sort) || sort < 1;
  });

  if (hasInvalidProp) {
    ElMessage.warning("请先完善商品属性");
    return;
  }

  emit("next");
}
</script>

<style scoped lang="scss">
.goods-edit-prop__card {
  border: 1px solid var(--admin-page-card-border);
  border-radius: 16px;
  background: var(--admin-page-card-bg);
  box-shadow: var(--admin-page-shadow);
}

:deep(.goods-edit-prop__card .el-card__body) {
  padding-top: 18px;
}

.goods-edit-prop__actions {
  display: flex;
  justify-content: flex-end;
  margin-bottom: 16px;
}

.goods-edit-prop__footer {
  display: flex;
  justify-content: space-between;
  gap: 12px;
}

@media (width <= 768px) {
  .goods-edit-prop__footer {
    flex-direction: column;
  }
}
</style>
