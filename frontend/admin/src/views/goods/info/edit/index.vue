<!-- 商品编辑 -->
<template>
  <div v-loading="loading" class="app-container">
    <el-card class="goods-edit-head-card" shadow="never">
      <div class="goods-edit-head">
        <div class="goods-edit-head__intro">
          <span class="goods-edit-head__eyebrow">商品信息汇总</span>
          <h1 class="goods-edit-head__title">{{ pageTitle }}</h1>
          <p class="goods-edit-head__subtitle">{{ pageSubtitle }}</p>
        </div>

        <div class="goods-edit-head__metrics">
          <div class="goods-edit-head__metric">
            <span>商品属性</span>
            <strong>{{ formData.propList.length }}</strong>
          </div>
          <div class="goods-edit-head__metric">
            <span>规格项</span>
            <strong>{{ formData.specList.length }}</strong>
          </div>
          <div class="goods-edit-head__metric">
            <span>总库存</span>
            <strong>{{ totalInventory }}</strong>
          </div>
        </div>
      </div>
    </el-card>

    <el-card class="goods-edit-steps-card" shadow="never">
      <el-steps :active="active" process-status="finish" finish-status="success" simple>
        <el-step v-for="step in stepList" :key="step" :title="step" />
      </el-steps>
    </el-card>

    <div class="goods-edit-stage">
      <info v-show="active == 0" v-if="loaded" v-model="formData" @next="next" />
      <prop v-show="active == 1" v-if="loaded" v-model="formData" @prev="prev" @next="next" />
      <spec v-show="active == 2" v-if="loaded" v-model="formData" @prev="prev" @next="next" />
      <sku v-show="active == 3" v-if="loaded" v-model="formData" @prev="prev" @reset-form="resetForm" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, toRefs, watch } from "vue";
import { useRoute } from "vue-router";
import info from "./components/info.vue";
import prop from "./components/prop.vue";
import spec from "./components/spec.vue";
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
    banner,
    /** 商品详情 */
    detail,
    /** 状态 */
    status: GoodsStatus.PUT_ON,
    /** 分类名称 */
    categoryName: "",
    /** 商品属性 */
    propList,
    /** 商品SKU */
    skuList,
    /** 商品规格 */
    specList
  } as GoodsInfoForm
});

const { loaded, active, formData } = toRefs(state);

const stepList = ["商品信息", "商品属性", "规格项", "商品规格"];

/** 顶部标题，兼容新增与编辑场景。 */
const pageTitle = computed(() => (goodsId.value ? "编辑商品" : "新增商品"));

/** 顶部副标题，补充当前商品维护流程说明。 */
const pageSubtitle = computed(() =>
  goodsId.value ? "按步骤调整商品信息、属性、规格项和商品规格。" : "按步骤完成商品信息、属性、规格项和商品规格配置。"
);

/** 顶部库存汇总，便于快速查看当前商品库存总量。 */
const totalInventory = computed(() =>
  (formData.value.skuList ?? []).reduce((total: number, item: Record<string, unknown>) => total + Number(item.inventory ?? 0), 0)
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

// 监听路由参数变化，刷新当前商品编辑数据。
watch(
  () => route.query.goodsId,
  newGoodsId => {
    goodsId.value = newGoodsId as unknown as number;
    handleQuery();
  }
);

/** 返回上一步，并在首步时保持不动。 */
function prev() {
  state.active = Math.max(0, state.active - 1);
}

/** 前进到下一步，并在最后一步保持当前位置。 */
function next() {
  state.active = Math.min(stepList.length - 1, state.active + 1);
}

/** 重置商品编辑表单。 */
function resetForm() {
  state.loaded = false;
  state.active = 0;
  Object.assign(state.formData, createDefaultFormData());
}

/** 查询商品详情，并兼容编辑态 SKU 的金额与规格字段格式。 */
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

          // 将规格项展开为表格字段，便于后续库存步骤直接渲染动态列。
          const specItemObj: Record<string, string> = {};
          item.specItem.forEach((spec, index) => {
            specItemObj[`specItem${index}`] = spec;
          });
          Object.assign(item, specItemObj);
          item.specItem = [];
        });

        Object.assign(state.formData, normalizedData);
        state.loaded = true;
      })
      .finally(() => {
        loading.value = false;
      });
    return;
  }

  state.loaded = true;
  loading.value = false;
}

onMounted(() => {
  handleQuery();
});
</script>

<style scoped lang="scss">
.goods-edit-head-card,
.goods-edit-steps-card {
  border: 1px solid var(--admin-page-card-border);
  border-radius: var(--admin-page-radius);
  background: var(--admin-page-card-bg);
  box-shadow: var(--admin-page-shadow);
}

.goods-edit-head-card {
  margin-bottom: 14px;
}

.goods-edit-steps-card {
  margin-bottom: 14px;
}

:deep(.goods-edit-head-card .el-card__body),
:deep(.goods-edit-steps-card .el-card__body) {
  padding: 16px 18px;
}

.goods-edit-head {
  display: flex;
  gap: 16px;
  align-items: center;
  justify-content: space-between;
}

.goods-edit-head__intro {
  min-width: 0;
}

.goods-edit-head__eyebrow {
  font-size: 12px;
  font-weight: 600;
  color: var(--admin-page-text-secondary);
}

.goods-edit-head__title {
  margin: 6px 0 0;
  font-size: 22px;
  font-weight: 700;
  color: var(--admin-page-text-primary);
}

.goods-edit-head__subtitle {
  margin: 6px 0 0;
  font-size: 13px;
  line-height: 1.6;
  color: var(--admin-page-text-secondary);
}

.goods-edit-head__metrics {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.goods-edit-head__metric {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 88px;
  padding: 10px 12px;
  border: 1px solid var(--admin-page-card-border-soft);
  border-radius: var(--admin-page-radius);
  background: var(--admin-page-card-bg-soft);
}

.goods-edit-head__metric span {
  font-size: 12px;
  color: var(--admin-page-text-secondary);
}

.goods-edit-head__metric strong {
  font-size: 18px;
  color: var(--admin-page-text-primary);
}

.goods-edit-steps-card :deep(.el-steps) {
  background: var(--admin-page-card-bg-soft);
  border: 1px solid var(--admin-page-card-border-soft);
  border-radius: var(--admin-page-radius);
}

.goods-edit-steps-card :deep(.el-step__title) {
  font-size: 14px;
  font-weight: 600;
}

.goods-edit-stage {
  min-height: 320px;
}

@media (width <= 768px) {
  .goods-edit-head,
  .goods-edit-head__metrics {
    flex-direction: column;
    align-items: stretch;
  }
}
</style>
