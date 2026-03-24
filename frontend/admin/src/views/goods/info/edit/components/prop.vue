<template>
  <div class="app-container">
    <el-card shadow="never">
      <div class="mb-10px">
        <el-button type="success" icon="plus" @click="handleAdd()">添加</el-button>
      </div>
      <el-form ref="dataFormRef" :model="formData" :rules="rules" :inline="true">
        <el-table :data="formData.propList" highlight-current-row border>
          <el-table-column type="index" width="50" />
          <el-table-column property="label" label="属性名称">
            <template #default="scope">
              <el-form-item :prop="'propList[' + scope.$index + '].label'" :rules="rules.label">
                <el-input v-model="scope.row.label" />
              </el-form-item>
            </template>
          </el-table-column>

          <el-table-column property="value" label="属性值">
            <template #default="scope">
              <el-form-item :prop="'propList[' + scope.$index + '].value'" :rules="rules.value">
                <el-input v-model="scope.row.value" type="textarea" />
              </el-form-item>
            </template>
          </el-table-column>

          <el-table-column property="sort" label="排序">
            <template #default="scope">
              <el-form-item :prop="'propList[' + scope.$index + '].sort'" :rules="rules.sort">
                <el-input-number
                  v-model="scope.row.sort"
                  controls-position="right"
                  :min="1"
                  :precision="0"
                  :step="1"
                />
              </el-form-item>
            </template>
          </el-table-column>

          <el-table-column label="操作" width="150" align="center">
            <template #default="scope">
              <el-button
                type="danger"
                size="small"
                link
                icon="delete"
                @click.stop="handleRemove(scope.$index)"
              >
                删除
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-form>
      <template #footer>
        <el-button @click="handlePrev">上一步，填写商品信息</el-button>
        <el-button type="primary" @click="handleNext">下一步，设置商品库存</el-button>
      </template>
    </el-card>
  </div>
</template>
<script setup lang="ts">
defineOptions({
  name: "GoodsEditGoodsProp",
  inheritAttrs: false,
});
const emit = defineEmits(["prev", "next", "update:modelValue"]);
const dataFormRef = ref(ElForm);

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
    label: [{ required: true, message: "请填写属性名称", trigger: "blur" }],
    value: [{ required: true, message: "请填写属性值", trigger: "blur" }],
    sort: [{ required: true, message: "请填写排序", trigger: "blur" }],
  },
});

const { rules } = toRefs(state);

function handleAdd() {
  formData.value.propList.push({
    sort: 1,
  });
}

function handleRemove(index: number) {
  formData.value.propList.splice(index, 1);
}

function handlePrev() {
  emit("prev");
}

function handleNext() {
  dataFormRef.value.validate((valid: any) => {
    if (valid) {
      emit("next");
    }
  });
}
</script>
