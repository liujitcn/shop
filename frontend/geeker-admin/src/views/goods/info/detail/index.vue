<!-- 商品属性 -->
<template>
  <div v-loading="loading" class="app-container">
    <el-tabs class="demo-tabs">
      <el-tab-pane label="基本信息">
        <el-form label-position="left" label-width="120px" class="detail-form">
          <!-- 基础信息 -->
          <el-row :gutter="20">
            <el-col :span="12">
              <el-form-item label="商品分类">
                <span>{{ formData.categoryName || "-" }}</span>
              </el-form-item>

              <el-form-item label="商品名称">
                <span>{{ formData.name }}</span>
              </el-form-item>

              <el-form-item label="商品描述">
                <span>{{ formData.desc || "-" }}</span>
              </el-form-item>
            </el-col>

            <!-- 商品主图 -->
            <el-col :span="12">
              <el-form-item label="商品主图">
                <div class="demo-image__preview">
                  <el-image
                    style="width: 100px; height: 100px"
                    :src="formData.picture"
                    :preview-src-list="[formData.picture]"
                    :zoom-rate="1.2"
                    :max-scale="7"
                    :min-scale="0.2"
                    :initial-index="4"
                    fit="cover"
                  >
                    <template #error>
                      <div class="image-slot">
                        <i class="el-icon-picture-outline" />
                      </div>
                    </template>
                  </el-image>
                </div>
              </el-form-item>
            </el-col>
          </el-row>

          <!-- 轮播图 -->
          <el-form-item label="轮播图">
            <div class="demo-image__preview">
              <el-image
                v-for="(img, index) in formData.banner"
                :key="index"
                style="width: 100px; height: 100px"
                :src="img"
                :preview-src-list="formData.banner"
                :zoom-rate="1.2"
                :max-scale="7"
                :min-scale="0.2"
                :initial-index="4"
                fit="cover"
              />
            </div>
          </el-form-item>

          <!-- 商品详情图 -->
          <el-form-item label="商品详情">
            <div class="demo-image__preview">
              <el-image
                v-for="(img, index) in formData.detail"
                :key="index"
                style="width: 100px; height: 100px"
                :src="img"
                :preview-src-list="formData.detail"
                :zoom-rate="1.2"
                :max-scale="7"
                :min-scale="0.2"
                :initial-index="4"
                fit="cover"
              />
            </div>
          </el-form-item>

          <!-- 状态 -->
          <el-form-item label="状态">
            <el-tag :type="formData.status === 1 ? 'success' : 'info'">
              {{ formData.status === 1 ? "上架" : "已下架" }}
            </el-tag>
          </el-form-item>
        </el-form>
      </el-tab-pane>
      <el-tab-pane label="属性信息">
        <el-card shadow="never">
          <ProTable row-key="label" :data="formData.propList" :columns="propColumns" :pagination="false" :tool-button="false" />
        </el-card>
      </el-tab-pane>
      <el-tab-pane label="库存信息">
        <el-card shadow="never">
          <ProTable row-key="skuCode" :data="formData.skuList" :columns="skuColumns" :pagination="false" :tool-button="false">
            <template #picture="scope">
              <el-popover placement="right" :width="400" trigger="hover">
                <img :src="scope.row.picture" width="400" height="400" />
                <template #reference>
                  <img :src="scope.row.picture" style="max-width: 60px; max-height: 60px" />
                </template>
              </el-popover>
            </template>
          </ProTable>
        </el-card>
      </el-tab-pane>
      <el-tab-pane label="规格信息">
        <el-card shadow="never">
          <ProTable row-key="name" :data="formData.specList" :columns="specColumns" :pagination="false" :tool-button="false" />
        </el-card>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from "vue";
import { useRoute } from "vue-router";
import type { ColumnProps } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
import { type GoodsForm } from "@/rpc/admin/goods";
import { type GoodsProp } from "@/rpc/admin/goods_prop";
import { type GoodsSpec } from "@/rpc/admin/goods_spec";
import { defGoodsService } from "@/api/admin/goods";
import { GoodsStatus } from "@/rpc/common/enum";

defineOptions({
  name: "GoodsDetail",
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

const formData = reactive<GoodsForm>({
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
});

const propColumns: ColumnProps[] = [
  { prop: "label", label: "商品属性标签" },
  { prop: "value", label: "商品属性值" },
  { prop: "sort", label: "排序", align: "right", width: 100 }
];

const specColumns: ColumnProps[] = [
  { prop: "name", label: "规格名称" },
  { prop: "item", label: "规格内容" },
  { prop: "sort", label: "排序", align: "right", width: 100 }
];

const skuColumns = computed<ColumnProps[]>(() => {
  const dynamicSpecColumns = formData.specList.map((item, index) => ({
    prop: `specItem${index}`,
    label: item.name,
    align: "center"
  }));

  return [
    ...dynamicSpecColumns,
    { prop: "picture", label: "规格图片", minWidth: 150, align: "center" },
    { prop: "skuCode", label: "规格编号", minWidth: 140 },
    { prop: "initSaleNum", label: "初始销量", align: "right", width: 100 },
    { prop: "realSaleNum", label: "真实销量", align: "right", width: 100 },
    { prop: "price", label: "价格（元）", align: "right", width: 110, cellType: "money" },
    { prop: "discountPrice", label: "折扣价格（元）", align: "right", width: 120, cellType: "money" },
    { prop: "inventory", label: "库存", align: "right", width: 100 }
  ];
});

// 监听路由参数变化，更新商品属性
watch(
  () => [route.query.goodsId],
  ([newGoodsId]) => {
    goodsId.value = newGoodsId as unknown as number;
    handleQuery();
  }
);

// 查询
function handleQuery() {
  loading.value = true;
  defGoodsService
    .GetGoods({
      value: goodsId.value
    })
    .then(data => {
      data.skuList.map(item => {
        item.specItem.forEach((spec, index) => {
          (item as any)[`specItem${index}`] = spec;
        });
      });
      Object.assign(formData, data);
    })
    .finally(() => {
      loading.value = false;
    });
}

onMounted(() => {
  handleQuery();
});
</script>
