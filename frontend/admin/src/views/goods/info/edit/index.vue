<!-- 商品编辑 -->
<template>
  <div v-loading="loading" class="app-container">
    <el-card class="goods-edit-head-card" shadow="never">
      <div class="goods-edit-head">
        <div class="goods-edit-head__intro">
          <h1 class="goods-edit-head__title">{{ pageTitle }}</h1>
          <p class="goods-edit-head__subtitle">{{ pageSubtitle }}</p>
        </div>

        <div class="goods-edit-head__metrics">
          <div class="goods-edit-head__metric">
            <span>商品属性</span>
            <strong>{{ formData.prop_list.length }}</strong>
          </div>
          <div class="goods-edit-head__metric">
            <span>规格项</span>
            <strong>{{ formData.spec_list.length }}</strong>
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
import type { GoodsInfoForm } from "@/rpc/admin/v1/goods_info";
import type { GoodsProp } from "@/rpc/admin/v1/goods_prop";
import type { GoodsSpec } from "@/rpc/admin/v1/goods_spec";
import { defGoodsInfoService } from "@/api/admin/goods_info";
import { GoodsStatus } from "@/rpc/common/v1/enum";
import { useTabsStore } from "@/stores/modules/tabs";

defineOptions({
  name: "GoodsEdit",
  inheritAttrs: false
});

const route = useRoute();
const tabsStore = useTabsStore();
const loading = ref(false);
const goodsId = ref(route.query.goodsId as unknown as number);

const prop_list = reactive<GoodsProp[]>([]);
const sku_list = reactive<any[]>([]);
const spec_list = reactive<GoodsSpec[]>([]);
const banner = reactive<string[]>([]);
const detail = reactive<string[]>([]);

const state = reactive({
  loaded: false,
  active: 0,
  formData: {
    /** 商品ID */
    id: 0,
    /** 分类ID列表 */
    category_id: [],
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
    category_name: "",
    /** 商品属性 */
    prop_list,
    /** 商品SKU */
    sku_list,
    /** 商品规格 */
    spec_list
  } as GoodsInfoForm
});

const { loaded, active, formData } = toRefs(state);

const stepList = ["商品信息", "商品属性", "规格项", "商品规格"];

/** 顶部标题，编辑态固定展示“商品编辑”，避免商品名过长影响阅读。 */
const pageTitle = computed(() => (goodsId.value ? "商品编辑" : "新增商品"));

/** 当前工作区标题与页面主标题保持一致，避免页签和浏览器标题带上商品名。 */
const workspaceTitle = computed(() => pageTitle.value);

/** 顶部副标题，补充当前商品维护流程说明。 */
const pageSubtitle = computed(() =>
  goodsId.value ? "按步骤调整商品信息、属性、规格项和商品规格。" : "按步骤完成商品信息、属性、规格项和商品规格配置。"
);

/** 顶部库存汇总，便于快速查看当前商品库存总量。 */
const totalInventory = computed(() =>
  (formData.value.sku_list ?? []).reduce((total: number, item: Record<string, unknown>) => total + Number(item.inventory ?? 0), 0)
);

/** 创建商品表单默认值，确保数组字段始终可用。 */
function createDefaultFormData(): GoodsInfoForm {
  return {
    /** 商品ID */
    id: 0,
    /** 分类ID列表 */
    category_id: [],
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
    category_name: "",
    /** 商品属性 */
    prop_list: [],
    /** 商品SKU */
    sku_list: [],
    /** 商品规格 */
    spec_list: []
  };
}

/** 规范化商品表单响应，避免属性、规格、SKU 为空时页面报错。 */
function normalizeGoodsInfoForm(data?: Partial<GoodsInfoForm>): GoodsInfoForm {
  return {
    ...createDefaultFormData(),
    ...data,
    category_id: Array.isArray(data?.category_id) ? data.category_id : [],
    banner: Array.isArray(data?.banner) ? data.banner : [],
    detail: Array.isArray(data?.detail) ? data.detail : [],
    prop_list: Array.isArray(data?.prop_list) ? data.prop_list : [],
    sku_list: Array.isArray(data?.sku_list) ? data.sku_list : [],
    spec_list: Array.isArray(data?.spec_list) ? data.spec_list : []
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

// 监听页面标题来源变化，确保新增与编辑切换时页签标题及时同步。
watch(
  workspaceTitle,
  () => {
    syncWorkspaceTitle();
  },
  { immediate: true }
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

/** 同步当前页签和浏览器标题，避免编辑态仍显示默认“新增商品”。 */
function syncWorkspaceTitle() {
  const currentTitle = workspaceTitle.value;
  tabsStore.setTabsTitle(currentTitle);
  document.title = currentTitle
    ? `${currentTitle} - ${import.meta.env.VITE_GLOB_APP_TITLE}`
    : import.meta.env.VITE_GLOB_APP_TITLE;
}

/** 查询商品详情，并兼容编辑态 SKU 的金额与规格字段格式。 */
function handleQuery() {
  loading.value = true;
  if (goodsId.value) {
    defGoodsInfoService
      .GetGoodsInfo({
        id: goodsId.value
      })
      .then(data => {
        const normalizedData = normalizeGoodsInfoForm(data);
        normalizedData.sku_list.forEach(item => {
          if (!item.init_sale_num) {
            item.init_sale_num = 0;
          }
          if (!item.price) {
            item.price = 0;
          } else {
            item.price = item.price / 100;
          }
          if (!item.discount_price) {
            item.discount_price = 0;
          } else {
            item.discount_price = item.discount_price / 100;
          }
          if (!item.inventory) {
            item.inventory = 0;
          }

          // 将规格项展开为表格字段，便于后续库存步骤直接渲染动态列。
          const specItemObj: Record<string, string> = {};
          item.spec_item.forEach((spec, index) => {
            specItemObj[`spec_item${index}`] = spec;
          });
          Object.assign(item, specItemObj);
          item.spec_item = [];
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

.goods-edit-head__title {
  margin: 0;
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
