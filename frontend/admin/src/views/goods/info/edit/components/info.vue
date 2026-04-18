<template>
  <div class="goods-edit-info">
    <el-card class="goods-edit-info__card" shadow="never">
      <ProForm
        ref="formRef"
        class="goods-edit-info__form"
        :model="formData"
        :fields="formFields"
        :rules="rules"
        label-width="96px"
        :gutter="16"
      />

      <template #footer>
        <div class="goods-edit-info__footer">
          <el-button type="primary" @click="handleNext">下一步</el-button>
        </div>
      </template>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, toRefs } from "vue";
import { ElMessage } from "element-plus";
import ProForm from "@/components/ProForm/index.vue";
import type { ProFormField, ProFormInstance } from "@/components/ProForm/interface";
import { defGoodsCategoryService } from "@/api/admin/goods_category";
import type { TreeOptionResponse_Option } from "@/rpc/common/common";
import { GoodsStatus } from "@/rpc/common/enum";

defineOptions({
  name: "GoodsEditInfo",
  inheritAttrs: false
});

const emit = defineEmits(["next", "update:modelValue"]);
const formRef = ref<ProFormInstance>();

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

const state = reactive({
  goodsCategoryOptions: [] as Array<TreeOptionResponse_Option>,
  rules: {
    categoryId: [{ required: true, message: "请选择商品分类", trigger: "change" }],
    name: [{ required: true, message: "请输入商品名称", trigger: "blur" }],
    desc: [{ required: true, message: "请输入商品描述", trigger: "blur" }],
    picture: [{ required: true, message: "请上传商品主图", trigger: "change" }],
    banner: [{ required: true, message: "请上传商品轮播图", trigger: "change" }],
    detail: [{ required: true, message: "请上传商品详情图", trigger: "change" }]
  }
});

const { goodsCategoryOptions, rules } = toRefs(state);

/** 商品信息表单字段配置。 */
const formFields = computed<ProFormField[]>(() => [
  {
    prop: "categoryId",
    label: "分类",
    component: "tree-select",
    options: goodsCategoryOptions.value as unknown as Array<{ label: string; value: string | number; children?: any[] }>,
    colSpan: 8,
    props: { placeholder: "请选择分类", filterable: true, style: { width: "100%" } }
  },
  {
    prop: "name",
    label: "标题",
    component: "input",
    colSpan: 16,
    props: { placeholder: "请输入商品标题" }
  },
  {
    prop: "desc",
    label: "描述",
    component: "textarea",
    colSpan: 12,
    props: {
      placeholder: "一句话概括商品卖点",
      rows: 3,
      maxlength: 120,
      showWordLimit: true,
      resize: "none"
    }
  },
  {
    prop: "status",
    label: "状态",
    component: "switch",
    colSpan: 12,
    props: {
      inlinePrompt: true,
      activeText: "上架",
      inactiveText: "下架",
      activeValue: GoodsStatus.PUT_ON,
      inactiveValue: GoodsStatus.PULL_OFF
    }
  },
  {
    prop: "picture",
    label: "主图",
    component: "image-upload",
    colSpan: 24,
    props: { uploadType: "goods" }
  },
  {
    prop: "banner",
    label: "轮播图",
    component: "images-upload",
    colSpan: 24,
    props: { uploadType: "goods" }
  },
  {
    prop: "detail",
    label: "详情图",
    component: "images-upload",
    colSpan: 24,
    props: { uploadType: "goods" }
  }
]);

/** 校验当前步骤表单后进入属性步骤。 */
async function handleNext() {
  try {
    const isValid = await formRef.value?.validate();
    if (!isValid) return;
    emit("next");
  } catch {
    ElMessage.warning("请先完善商品信息");
  }
}

/** 查询分类树数据，作为商品分类下拉选项。 */
function handleQuery() {
  defGoodsCategoryService.OptionGoodsCategory({}).then(res => {
    state.goodsCategoryOptions = res.list;
  });
}

onMounted(() => {
  handleQuery();
});
</script>

<style scoped lang="scss">
.goods-edit-info__card {
  border: 1px solid var(--admin-page-card-border);
  border-radius: var(--admin-page-radius);
  background: var(--admin-page-card-bg);
  box-shadow: var(--admin-page-shadow);
}

:deep(.goods-edit-info__card .el-card__body) {
  padding-top: 18px;
}

.goods-edit-info__form :deep(.el-form-item) {
  margin-bottom: 16px;
}

.goods-edit-info__form :deep(.el-textarea__inner) {
  min-height: 88px;
}

.goods-edit-info__footer {
  display: flex;
  justify-content: flex-end;
}
</style>
