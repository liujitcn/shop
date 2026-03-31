<template>
  <div class="app-container">
    <el-card shadow="never">
      <div class="mb-10px">
        <el-button type="success" icon="plus" @click="handleAdd()">添加</el-button>
      </div>
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
        <el-button @click="handlePrev">上一步，填写商品信息</el-button>
        <el-button type="primary" @click="handleNext">下一步，设置商品库存</el-button>
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
