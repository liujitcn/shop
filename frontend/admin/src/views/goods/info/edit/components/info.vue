<template>
  <div class="goods-edit-info">
    <el-card class="goods-edit-info__hero" shadow="never">
      <div class="goods-edit-info__hero-content">
        <div>
          <h2 class="goods-edit-info__title">填写商品信息</h2>
          <p class="goods-edit-info__desc">
            先完成分类、标题、描述、主图、轮播图、详情图和上下架状态，后续属性与库存都会基于这里的数据继续编辑。
          </p>
        </div>

        <div class="goods-edit-info__summary">
          <div class="goods-edit-info__summary-item">
            <span>轮播图</span>
            <strong>{{ formData.banner?.length ?? 0 }} 张</strong>
          </div>
          <div class="goods-edit-info__summary-item">
            <span>详情图</span>
            <strong>{{ formData.detail?.length ?? 0 }} 张</strong>
          </div>
        </div>
      </div>
    </el-card>

    <el-card class="goods-edit-info__card" shadow="never">
      <template #header>
        <div class="goods-edit-info__card-header">
          <span>基础信息</span>
        </div>
      </template>

      <ProForm ref="baseFormRef" :model="formData" :fields="baseFormFields" :rules="rules" label-width="120px" />
    </el-card>

    <el-card class="goods-edit-info__card" shadow="never">
      <template #header>
        <div class="goods-edit-info__card-header">
          <span>图片与详情</span>
        </div>
      </template>

      <ProForm ref="mediaFormRef" :model="formData" :fields="mediaFormFields" :rules="rules" label-width="120px" />

      <template #footer>
        <div class="goods-edit-info__footer">
          <el-button type="primary" @click="handleNext">下一步，设置商品属性</el-button>
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
defineOptions({
  name: "GoodsEditInfo",
  inheritAttrs: false
});
const emit = defineEmits(["next", "update:modelValue"]);
import type { TreeOptionResponse_Option } from "@/rpc/common/common";
import { GoodsStatus } from "@/rpc/common/enum";
const baseFormRef = ref<ProFormInstance>();
const mediaFormRef = ref<ProFormInstance>();

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

/** 基础信息字段配置。 */
const baseFormFields = computed<ProFormField[]>(() => [
  {
    prop: "categoryId",
    label: "商品分类",
    component: "tree-select",
    options: goodsCategoryOptions.value as unknown as Array<{ label: string; value: string | number; children?: any[] }>,
    colSpan: 12,
    props: { placeholder: "请选择商品分类", filterable: true, style: { width: "100%" } }
  },
  { prop: "name", label: "商品名称", component: "input", colSpan: 12, props: { placeholder: "请输入商品名称" } },
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
    prop: "desc",
    label: "商品描述",
    component: "textarea",
    colSpan: 24,
    props: { placeholder: "请输入商品描述", rows: 4 }
  }
]);

/** 图片与详情字段配置。 */
const mediaFormFields = computed<ProFormField[]>(() => [
  { prop: "picture", label: "商品主图", component: "image-upload", colSpan: 24 },
  { prop: "banner", label: "商品轮播图", component: "images-upload", colSpan: 24 },
  { prop: "detail", label: "商品详情", component: "images-upload", colSpan: 24 }
]);

async function handleNext() {
  try {
    const baseValid = await baseFormRef.value?.validate();
    const mediaValid = await mediaFormRef.value?.validate();
    if (!baseValid || !mediaValid) return;
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

<style scoped lang="scss">
.goods-edit-info {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.goods-edit-info__hero,
.goods-edit-info__card {
  border: 1px solid var(--admin-page-card-border);
  border-radius: 16px;
  background: var(--admin-page-card-bg);
  box-shadow: var(--admin-page-shadow);
}

.goods-edit-info__hero-content {
  display: grid;
  grid-template-columns: minmax(0, 1.2fr) minmax(220px, 0.8fr);
  gap: 24px;
  align-items: center;
}

.goods-edit-info__title {
  margin: 0 0 10px;
  font-size: 20px;
  font-weight: 700;
  color: var(--admin-page-text-primary);
}

.goods-edit-info__desc {
  margin: 0;
  font-size: 14px;
  line-height: 1.7;
  color: var(--admin-page-text-secondary);
}

.goods-edit-info__summary {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
}

.goods-edit-info__summary-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 16px;
  border: 1px solid var(--admin-page-card-border-soft);
  border-radius: 12px;
  background: var(--admin-page-card-bg-soft);
}

.goods-edit-info__summary-item span {
  font-size: 13px;
  color: var(--admin-page-text-secondary);
}

.goods-edit-info__summary-item strong {
  font-size: 20px;
  color: var(--admin-page-text-primary);
}

.goods-edit-info__card-header {
  font-size: 16px;
  font-weight: 600;
  color: var(--admin-page-text-primary);
}

.goods-edit-info__footer {
  display: flex;
  justify-content: flex-end;
}

@media (width <= 992px) {
  .goods-edit-info__hero-content {
    grid-template-columns: 1fr;
  }
}

@media (width <= 768px) {
  .goods-edit-info__summary {
    grid-template-columns: 1fr;
  }
}
</style>
