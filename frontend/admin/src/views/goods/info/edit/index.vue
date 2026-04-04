<!-- 商品编辑 -->
<template>
  <div v-loading="loading" class="app-container">
    <el-card class="goods-edit-hero-card" shadow="never">
      <div class="goods-edit-hero">
        <div class="goods-edit-hero__intro">
          <span class="goods-edit-hero__label">商品编辑流程</span>
          <div class="goods-edit-hero__main">
            <h1 class="goods-edit-hero__title">{{ pageTitle }}</h1>
            <p class="goods-edit-hero__desc">{{ pageDescription }}</p>
          </div>
        </div>

        <div class="goods-edit-metrics">
          <div class="goods-edit-metric-card">
            <span class="goods-edit-metric-card__label">商品属性</span>
            <strong class="goods-edit-metric-card__value">{{ formData.propList.length }}</strong>
          </div>
          <div class="goods-edit-metric-card">
            <span class="goods-edit-metric-card__label">商品规格</span>
            <strong class="goods-edit-metric-card__value">{{ formData.specList.length }}</strong>
          </div>
          <div class="goods-edit-metric-card">
            <span class="goods-edit-metric-card__label">SKU 数量</span>
            <strong class="goods-edit-metric-card__value">{{ formData.skuList.length }}</strong>
          </div>
        </div>
      </div>
    </el-card>

    <el-card class="goods-edit-steps-card" shadow="never">
      <el-steps :active="active" process-status="finish" finish-status="success" simple>
        <el-step title="填写商品信息" description="分类、标题、主图、轮播图、详情图与状态" />
        <el-step title="设置商品属性" description="补充商品属性名称、属性值与排序" />
        <el-step title="设置商品库存" description="维护规格、库存、价格、图片和销量" />
      </el-steps>
    </el-card>

    <div class="goods-edit-stage">
      <info v-show="active == 0" v-if="loaded == true" v-model="formData" @prev="prev" @next="next" />
      <prop v-show="active == 1" v-if="loaded == true" v-model="formData" @prev="prev" @next="next" />
      <sku v-show="active == 2" v-if="loaded == true" v-model="formData" @prev="prev" @next="next" @reset-form="resetForm" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, toRefs, watch } from "vue";
import { useRoute } from "vue-router";
import info from "./components/info.vue";
import prop from "./components/prop.vue";
import sku from "./components/sku.vue";
import type { GoodsInfoForm } from "@/rpc/admin/goods_info";
import type { GoodsProp } from "@/rpc/admin/goods_prop";
import type { GoodsSpec } from "@/rpc/admin/goods_spec";
import { defGoodsInfoService } from "@/api/admin/goods_info";
import { GoodsStatus } from "@/rpc/common/enum";

defineOptions({
  name: "GoodsEdit",
  inheritAttrs: false
});

const route = useRoute();

const loading = ref(false);

const goodsId = ref(route.query.goodsId as unknown as number);

const propList = reactive<GoodsProp[]>([]);
const skuList = reactive<any[]>([]);
const specList = reactive<GoodsSpec[]>([]);
const banner = reactive<string[]>([]);
const detail = reactive<string[]>([]);

const state = reactive({
  loaded: false,
  active: 0,
  formData: {
    /** 商品ID */
    id: 0,
    /** 分类ID */
    categoryId: undefined,
    /** 名称 */
    name: "",
    /** 描述 */
    desc: "",
    /** 商品图片 */
    picture: "",
    /** 轮播图 */
    banner: banner,
    /** 商品详情 */
    detail: detail,
    /** 状态 */
    status: GoodsStatus.PUT_ON,
    categoryName: "",
    /** 商品属性 */
    propList: propList,
    /** 商品SKU */
    skuList: skuList,
    /** 商品规格 */
    specList: specList
  } as GoodsInfoForm
});

const { loaded, active, formData } = toRefs(state);

/** 当前页面标题，兼容新增与编辑场景。 */
const pageTitle = computed(() => (goodsId.value ? "编辑商品" : "新增商品"));

/** 当前页面说明文案，帮助运营快速理解三步编辑流程。 */
const pageDescription = computed(() =>
  goodsId.value
    ? "按步骤更新商品信息、属性与库存，所有历史字段都会完整保留。"
    : "按步骤完成商品基础信息、属性和库存维护，提交前不会丢失任何已填写内容。"
);

/** 创建商品表单默认值，确保数组字段始终可用。 */
function createDefaultFormData(): GoodsInfoForm {
  return {
    /** 商品ID */
    id: 0,
    /** 分类ID */
    categoryId: undefined,
    /** 名称 */
    name: "",
    /** 描述 */
    desc: "",
    /** 商品图片 */
    picture: "",
    /** 轮播图 */
    banner: [],
    /** 商品详情 */
    detail: [],
    /** 状态 */
    status: GoodsStatus.PUT_ON,
    /** 分类名称 */
    categoryName: "",
    /** 商品属性 */
    propList: [],
    /** 商品SKU */
    skuList: [],
    /** 商品规格 */
    specList: []
  };
}

/** 规范化商品表单响应，避免属性、规格、SKU 为空时页面报错。 */
function normalizeGoodsInfoForm(data?: Partial<GoodsInfoForm>): GoodsInfoForm {
  return {
    ...createDefaultFormData(),
    ...data,
    banner: Array.isArray(data?.banner) ? data.banner : [],
    detail: Array.isArray(data?.detail) ? data.detail : [],
    propList: Array.isArray(data?.propList) ? data.propList : [],
    skuList: Array.isArray(data?.skuList) ? data.skuList : [],
    specList: Array.isArray(data?.specList) ? data.specList : []
  };
}

// 监听路由参数变化，更新商品属性
watch(
  () => [route.query.goodsId],
  ([newGoodsId]) => {
    goodsId.value = newGoodsId as unknown as number;
    handleQuery();
  }
);

function prev() {
  if (state.active-- <= 0) {
    state.active = 0;
  }
}
function next() {
  if (state.active++ >= 2) {
    state.active = 0;
  }
}

// 重置表单
function resetForm() {
  state.loaded = false;
  state.active = 0;
  Object.assign(state.formData, createDefaultFormData());
}

// 查询
function handleQuery() {
  loading.value = true;
  if (goodsId.value) {
    defGoodsInfoService
      .GetGoodsInfo({
        value: goodsId.value
      })
      .then(data => {
        const normalizedData = normalizeGoodsInfoForm(data);
        normalizedData.skuList.forEach(item => {
          if (!item.initSaleNum) {
            item.initSaleNum = 0;
          }
          if (!item.price) {
            item.price = 0;
          } else {
            item.price = item.price / 100;
          }
          if (!item.discountPrice) {
            item.discountPrice = 0;
          } else {
            item.discountPrice = item.discountPrice / 100;
          }
          if (!item.inventory) {
            item.inventory = 0;
          }
          // 将规格项转换为对象属性
          const specItemObj: Record<string, string> = {};
          item.specItem.forEach((spec, index) => {
            specItemObj[`specItem${index}`] = spec;
          });
          // 使用类型断言合并规格项对象
          Object.assign(item, specItemObj);
          // 将 specItem 设置为空数组而不是删除它
          item.specItem = [];
        });
        Object.assign(state.formData, normalizedData);
        state.loaded = true;
      })
      .finally(() => {
        loading.value = false;
      });
  } else {
    state.loaded = true;
    loading.value = false;
  }
}

onMounted(() => {
  handleQuery();
});
</script>

<style scoped lang="scss">
.goods-edit-hero-card,
.goods-edit-steps-card {
  border: 1px solid var(--admin-page-card-border);
  border-radius: 16px;
  background: var(--admin-page-card-bg);
  box-shadow: var(--admin-page-shadow);
}

.goods-edit-hero-card {
  margin-bottom: 18px;
}

:deep(.goods-edit-hero-card .el-card__body),
:deep(.goods-edit-steps-card .el-card__body) {
  padding: 16px 18px;
}

.goods-edit-hero {
  display: flex;
  gap: 16px;
  align-items: center;
  justify-content: space-between;
  padding: 2px 0;
}

.goods-edit-hero__intro {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
}

.goods-edit-hero__main {
  display: flex;
  flex-wrap: wrap;
  gap: 10px 16px;
  align-items: center;
}

.goods-edit-hero__label {
  display: inline-flex;
  font-size: 12px;
  font-weight: 600;
  color: var(--admin-page-text-secondary);
}

.goods-edit-hero__title {
  margin: 0;
  font-size: 22px;
  font-weight: 700;
  line-height: 1.2;
  color: var(--admin-page-text-primary);
}

.goods-edit-hero__desc {
  margin: 0;
  font-size: 14px;
  line-height: 1.7;
  color: var(--admin-page-text-secondary);
}

.goods-edit-metrics {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.goods-edit-metric-card {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 12px 14px;
  min-width: 140px;
  border: 1px solid var(--admin-page-card-border-soft);
  border-radius: 12px;
  background: var(--admin-page-card-bg-soft);
}

.goods-edit-metric-card__label {
  font-size: 13px;
  color: var(--admin-page-text-secondary);
}

.goods-edit-metric-card__value {
  font-size: 20px;
  font-weight: 700;
  color: var(--admin-page-text-primary);
}

.goods-edit-steps-card {
  margin-bottom: 16px;
}

.goods-edit-steps-card :deep(.el-steps) {
  background: var(--admin-page-card-bg-soft);
  border: 1px solid var(--admin-page-card-border-soft);
  border-radius: 12px;
}

.goods-edit-steps-card :deep(.el-step__title) {
  font-size: 14px;
  font-weight: 600;
}

.goods-edit-steps-card :deep(.el-step__description) {
  font-size: 12px;
  line-height: 1.6;
}

.goods-edit-stage {
  min-height: 320px;
}

@media (width <= 992px) {
  .goods-edit-hero {
    flex-direction: column;
    align-items: stretch;
  }

  .goods-edit-metrics {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (width <= 768px) {
  .goods-edit-hero__title {
    font-size: 22px;
  }

  .goods-edit-metrics {
    grid-template-columns: 1fr;
  }
}
</style>
