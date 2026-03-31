<template>
  <div class="app-container">
    <el-card shadow="never">
      <ProForm ref="proFormRef" :model="formData" :fields="formFields" :rules="rules" label-width="120px" />
      <template #footer>
        <el-button type="primary" @click="handleNext">下一步，设置商品属性</el-button>
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
defineOptions({
  name: "GoodsEditInfo",
  inheritAttrs: false
});
const emit = defineEmits(["next", "update:modelValue"]);
import type { TreeOptionResponse_Option } from "@/rpc/common/common";
import { GoodsStatus } from "@/rpc/common/enum";
const proFormRef = ref<ProFormInstance>();

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
    picture: [{ required: true, message: "请上传商品图片", trigger: "change" }],
    banner: [{ required: true, message: "请上传商品轮播图", trigger: "change" }],
    detail: [{ required: true, message: "请上传商品详情", trigger: "change" }]
  }
});

const { goodsCategoryOptions, rules } = toRefs(state);

const formFields = computed<ProFormField[]>(() => [
  {
    prop: "categoryId",
    label: "商品分类",
    component: "tree-select",
    options: goodsCategoryOptions.value as unknown as Array<{ label: string; value: string | number; children?: any[] }>,
    props: { placeholder: "请选择商品分类", filterable: true }
  },
  { prop: "name", label: "商品名称", component: "input", props: { placeholder: "请输入商品名称" } },
  { prop: "desc", label: "商品描述", component: "textarea", props: { placeholder: "请输入商品描述" } },
  { prop: "picture", label: "商品主图", component: "image-upload" },
  { prop: "banner", label: "商品轮播图", component: "images-upload" },
  { prop: "detail", label: "商品详情", component: "images-upload" },
  {
    prop: "status",
    label: "状态",
    component: "switch",
    props: {
      inlinePrompt: true,
      activeText: "上架",
      inactiveText: "下架",
      activeValue: GoodsStatus.PUT_ON,
      inactiveValue: GoodsStatus.PULL_OFF
    }
  }
]);

async function handleNext() {
  try {
    const valid = await proFormRef.value?.validate();
    if (!valid) return;
    emit("next");
  } catch {
    ElMessage.warning("请完善商品信息后再设置商品属性");
  }
}

// 查询
function handleQuery() {
  // 加载分类下拉数据源
  defGoodsCategoryService.OptionGoodsCategory({}).then(res => {
    state.goodsCategoryOptions = res.list;
  });
}

onMounted(() => {
  handleQuery();
});
</script>
