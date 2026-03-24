<!-- 商品属性 -->
<template>
  <div v-loading="loading" class="app-container">
    <el-steps :active="active" process-status="finish" finish-status="success" simple>
      <el-step title="填写商品信息" />
      <el-step title="设置商品属性" />
      <el-step title="设置商品库存" />
    </el-steps>

    <info v-show="active == 0" v-if="loaded == true" v-model="formData" @prev="prev" @next="next" />
    <prop v-show="active == 1" v-if="loaded == true" v-model="formData" @prev="prev" @next="next" />
    <sku
      v-show="active == 2"
      v-if="loaded == true"
      v-model="formData"
      @prev="prev"
      @next="next"
      @reset-form="resetForm"
    />
  </div>
</template>

<script setup lang="ts">
import info from "./components/info.vue";
import prop from "./components/prop.vue";
import sku from "./components/sku.vue";
import { GoodsForm } from "@/rpc/admin/goods";
import type { GoodsProp } from "@/rpc/admin/goods_prop";
import type { GoodsSpec } from "@/rpc/admin/goods_spec";
import { defGoodsService } from "@/api/admin/goods";
import { GoodsStatus } from "@/rpc/common/enum";

defineOptions({
  name: "GoodsEdit",
  inheritAttrs: false,
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
    specList: specList,
  } as GoodsForm,
});

const { loaded, active, formData } = toRefs(state);

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
  state.formData = {
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
    categoryName: "",
    /** 商品属性 */
    propList: [],
    /** 商品SKU */
    skuList: [],
    /** 商品规格 */
    specList: [],
  };
}

// 查询
function handleQuery() {
  loading.value = true;
  if (goodsId.value) {
    defGoodsService
      .GetGoods({
        value: goodsId.value,
      })
      .then((data) => {
        data.skuList.map((item) => {
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
        state.formData = data;
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
