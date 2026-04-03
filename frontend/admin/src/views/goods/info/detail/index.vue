<!-- 商品详情 -->
<template>
  <div v-loading="loading" class="app-container">
    <el-card class="goods-hero-card" shadow="never">
      <div class="goods-hero">
        <div class="goods-hero__media">
          <el-image
            class="goods-cover-image"
            :src="formData.picture"
            :preview-src-list="coverPreviewList"
            fit="cover"
            preview-teleported
          >
            <template #error>
              <div class="image-placeholder">暂无主图</div>
            </template>
          </el-image>
        </div>

        <div class="goods-hero__content">
          <div class="goods-hero__eyebrow">商品资料总览</div>
          <div class="goods-hero__title-row">
            <h1 class="goods-hero__title">{{ formData.name || "-" }}</h1>
            <el-tag :type="statusTagType" effect="light" round>{{ statusText }}</el-tag>
          </div>
          <p class="goods-hero__desc">{{ formData.desc || "暂无商品描述" }}</p>

          <div class="goods-meta-grid">
            <div class="goods-meta-item">
              <span class="goods-meta-item__label">商品分类</span>
              <span class="goods-meta-item__value">{{ formData.categoryName || "-" }}</span>
            </div>
            <button type="button" class="goods-meta-item goods-meta-item--action" @click="handleNavigateSection('banner')">
              <span class="goods-meta-item__label">轮播图</span>
              <span class="goods-meta-item__value">{{ bannerCount }} 张</span>
            </button>
            <button type="button" class="goods-meta-item goods-meta-item--action" @click="handleNavigateSection('detail')">
              <span class="goods-meta-item__label">详情图</span>
              <span class="goods-meta-item__value">{{ detailCount }} 张</span>
            </button>
            <button type="button" class="goods-meta-item goods-meta-item--action" @click="handleNavigateSection('sku')">
              <span class="goods-meta-item__label">SKU 数量</span>
              <span class="goods-meta-item__value">{{ skuCount }} 个</span>
            </button>
          </div>

          <div class="goods-metric-list">
            <button type="button" class="goods-metric-card goods-metric-card--action" @click="handleNavigateSection('prop')">
              <span class="goods-metric-card__label">商品属性</span>
              <strong class="goods-metric-card__value">{{ propCount }}</strong>
            </button>
            <button type="button" class="goods-metric-card goods-metric-card--action" @click="handleNavigateSection('spec')">
              <span class="goods-metric-card__label">商品规格</span>
              <strong class="goods-metric-card__value">{{ specCount }}</strong>
            </button>
            <div class="goods-metric-card">
              <span class="goods-metric-card__label">库存总量</span>
              <strong class="goods-metric-card__value">{{ totalInventory }}</strong>
            </div>
          </div>
        </div>
      </div>
    </el-card>

    <el-tabs v-model="activeTabName" class="goods-detail-tabs">
      <el-tab-pane label="基本信息" name="basic">
        <el-card class="detail-section-card" shadow="never">
          <template #header>
            <div class="detail-section-card__header">
              <span>商品信息</span>
            </div>
          </template>

          <el-descriptions :column="2" border class="goods-descriptions">
            <el-descriptions-item label="商品分类">{{ formData.categoryName || "-" }}</el-descriptions-item>
            <el-descriptions-item label="状态">
              <el-tag :type="statusTagType" effect="light" round>{{ statusText }}</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="商品名称" :span="2">{{ formData.name || "-" }}</el-descriptions-item>
            <el-descriptions-item label="商品描述" :span="2">{{ formData.desc || "-" }}</el-descriptions-item>
            <el-descriptions-item label="轮播图数量">{{ bannerCount }} 张</el-descriptions-item>
            <el-descriptions-item label="详情图数量">{{ detailCount }} 张</el-descriptions-item>
            <el-descriptions-item label="SKU 数量">{{ skuCount }} 个</el-descriptions-item>
            <el-descriptions-item label="库存总量">{{ totalInventory }}</el-descriptions-item>
          </el-descriptions>
        </el-card>

        <el-card ref="bannerSectionRef" class="detail-section-card" shadow="never">
          <template #header>
            <div class="detail-section-card__header">
              <span>轮播图</span>
              <span class="detail-section-card__extra">{{ bannerCount }} 张</span>
            </div>
          </template>

          <div v-if="formData.banner.length" class="image-gallery">
            <el-image
              v-for="(img, index) in formData.banner"
              :key="index"
              class="image-gallery__item"
              :src="img"
              :preview-src-list="formData.banner"
              :initial-index="index"
              fit="cover"
              preview-teleported
            />
          </div>
          <el-empty v-else description="暂无轮播图" :image-size="90" />
        </el-card>

        <el-card ref="detailSectionRef" class="detail-section-card" shadow="never">
          <template #header>
            <div class="detail-section-card__header">
              <span>商品详情图</span>
              <span class="detail-section-card__extra">{{ detailCount }} 张</span>
            </div>
          </template>

          <div v-if="formData.detail.length" class="detail-image-list">
            <el-image
              v-for="(img, index) in formData.detail"
              :key="index"
              class="detail-image-list__item"
              :src="img"
              :preview-src-list="formData.detail"
              :initial-index="index"
              fit="cover"
              preview-teleported
            />
          </div>
          <el-empty v-else description="暂无详情图" :image-size="90" />
        </el-card>
      </el-tab-pane>

      <el-tab-pane label="库存信息" name="sku">
        <el-card ref="skuSectionRef" class="detail-table-card" shadow="never">
          <template #header>
            <div class="detail-section-card__header">
              <span>SKU 库存明细</span>
              <span class="detail-section-card__extra">{{ skuCount }} 个 SKU</span>
            </div>
          </template>

          <ProTable row-key="skuCode" :data="formData.skuList" :columns="skuColumns" :pagination="false" :tool-button="false">
            <template #picture="scope">
              <div class="sku-image-cell">
                <el-popover placement="right" :width="420" trigger="hover">
                  <img :src="scope.row.picture" class="sku-image-cell__preview" alt="规格图片预览" />
                  <template #reference>
                    <img :src="scope.row.picture" class="sku-image-cell__thumb" alt="规格图片缩略图" />
                  </template>
                </el-popover>
              </div>
            </template>
          </ProTable>
        </el-card>
      </el-tab-pane>

      <el-tab-pane label="规格信息" name="spec">
        <el-card ref="specSectionRef" class="detail-table-card" shadow="never">
          <template #header>
            <div class="detail-section-card__header">
              <span>商品规格</span>
              <span class="detail-section-card__extra">{{ specCount }} 项</span>
            </div>
          </template>

          <ProTable row-key="name" :data="formData.specList" :columns="specColumns" :pagination="false" :tool-button="false" />
        </el-card>
      </el-tab-pane>

      <el-tab-pane label="属性信息" name="prop">
        <el-card ref="propSectionRef" class="detail-table-card" shadow="never">
          <template #header>
            <div class="detail-section-card__header">
              <span>商品属性</span>
              <span class="detail-section-card__extra">{{ propCount }} 项</span>
            </div>
          </template>

          <ProTable row-key="label" :data="formData.propList" :columns="propColumns" :pagination="false" :tool-button="false" />
        </el-card>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, reactive, ref, watch } from "vue";
import { useRoute } from "vue-router";
import type { ColumnProps } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
import { type GoodsForm } from "@/rpc/admin/goods";
import { type GoodsProp } from "@/rpc/admin/goods_prop";
import { type GoodsSku } from "@/rpc/admin/goods_sku";
import { type GoodsSpec } from "@/rpc/admin/goods_spec";
import { defGoodsService } from "@/api/admin/goods";
import { GoodsStatus } from "@/rpc/common/enum";

defineOptions({
  name: "GoodsDetail",
  inheritAttrs: false
});

/** 扩展 SKU 行数据，兼容详情页动态规格列渲染。 */
type GoodsSkuTableRow = GoodsSku & Record<string, string | number | string[] | undefined>;
type GoodsDetailTabName = "basic" | "sku" | "spec" | "prop";
type GoodsDetailSectionKey = "banner" | "detail" | "sku" | "spec" | "prop";

const route = useRoute();
const loading = ref(false);
const goodsId = ref(route.query.goodsId as unknown as number);
const activeTabName = ref<GoodsDetailTabName>("basic");
const bannerSectionRef = ref();
const detailSectionRef = ref();
const skuSectionRef = ref();
const specSectionRef = ref();
const propSectionRef = ref();

const propList = reactive<GoodsProp[]>([]);
const skuList = reactive<GoodsSkuTableRow[]>([]);
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
});

/** 统一计算主图预览列表，避免空图片时打开预览报错。 */
const coverPreviewList = computed(() => (formData.picture ? [formData.picture] : []));

/** 统一计算页面顶部商品状态文案。 */
const statusText = computed(() => (formData.status === GoodsStatus.PUT_ON ? "上架中" : "已下架"));

/** 统一计算页面顶部商品状态标签样式。 */
const statusTagType = computed(() => (formData.status === GoodsStatus.PUT_ON ? "success" : "info"));

/** 统计轮播图数量，集中服务于概览区和分组标题。 */
const bannerCount = computed(() => formData.banner.length);

/** 统计详情图数量，集中服务于概览区和分组标题。 */
const detailCount = computed(() => formData.detail.length);

/** 统计属性项数量。 */
const propCount = computed(() => formData.propList.length);

/** 统计规格项数量。 */
const specCount = computed(() => formData.specList.length);

/** 统计 SKU 数量。 */
const skuCount = computed(() => formData.skuList.length);

/** 汇总全部 SKU 的库存，便于首屏快速查看商品备货情况。 */
const totalInventory = computed(() => formData.skuList.reduce((total, item) => total + Number(item.inventory ?? 0), 0));

/** 页面分区和标签页映射，统一控制概览卡跳转行为。 */
const detailSectionMap: Record<GoodsDetailSectionKey, { tab: GoodsDetailTabName; targetRef: typeof bannerSectionRef }> = {
  banner: { tab: "basic", targetRef: bannerSectionRef },
  detail: { tab: "basic", targetRef: detailSectionRef },
  sku: { tab: "sku", targetRef: skuSectionRef },
  spec: { tab: "spec", targetRef: specSectionRef },
  prop: { tab: "prop", targetRef: propSectionRef }
};

/** 商品属性明细表格列配置。 */
const propColumns: ColumnProps[] = [
  { prop: "label", label: "商品属性标签", minWidth: 180 },
  { prop: "value", label: "商品属性值", minWidth: 220 },
  { prop: "sort", label: "排序", align: "right", minWidth: 100 }
];

/** 商品规格明细表格列配置。 */
const specColumns: ColumnProps[] = [
  { prop: "name", label: "规格名称", minWidth: 160 },
  { prop: "item", label: "规格内容", minWidth: 220 },
  { prop: "sort", label: "排序", align: "right", minWidth: 100 }
];

/** 商品 SKU 明细表格列配置。 */
const skuColumns = computed<ColumnProps[]>(() => {
  /** 根据规格定义动态拼接表头，保证不同商品规格也能完整展示。 */
  const dynamicSpecColumns = formData.specList.map((item, index) => ({
    prop: `specItem${index}`,
    label: item.name,
    align: "center",
    minWidth: 120
  }));

  return [
    ...dynamicSpecColumns,
    {
      prop: "specItem",
      label: "规格描述",
      minWidth: 180,
      render: scope => ((scope.row.specItem as string[] | undefined) ?? []).join(" / ")
    },
    { prop: "picture", label: "规格图片", minWidth: 110, align: "center" },
    { prop: "skuCode", label: "规格编号", minWidth: 160 },
    { prop: "initSaleNum", label: "初始销量", align: "right", minWidth: 100 },
    { prop: "realSaleNum", label: "真实销量", align: "right", minWidth: 100 },
    { prop: "price", label: "价格（元）", align: "right", minWidth: 110, cellType: "money" },
    { prop: "discountPrice", label: "折扣价格（元）", align: "right", minWidth: 130, cellType: "money" },
    { prop: "inventory", label: "库存", align: "right", minWidth: 100 }
  ];
});

// 监听路由参数变化，更新商品详情数据。
watch(
  () => route.query.goodsId,
  newGoodsId => {
    goodsId.value = newGoodsId as unknown as number;
    if (goodsId.value) {
      handleQuery();
    }
  }
);

/**
 * 查询商品详情，并将 SKU 规格项展开到平铺字段，便于表格动态列直接渲染。
 */
function handleQuery() {
  if (!goodsId.value) return;
  loading.value = true;
  defGoodsService
    .GetGoods({
      value: goodsId.value
    })
    .then(data => {
      data.skuList.forEach(item => {
        // 将规格数组转成扁平字段，避免在表格单元格里重复写解析逻辑。
        item.specItem.forEach((spec, index) => {
          (item as GoodsSkuTableRow)[`specItem${index}`] = spec;
        });
      });
      Object.assign(formData, data);
    })
    .finally(() => {
      loading.value = false;
    });
}

/**
 * 从顶部统计卡跳转到对应分区，先切换标签页，再滚动到目标区域。
 */
async function handleNavigateSection(sectionKey: GoodsDetailSectionKey) {
  const sectionConfig = detailSectionMap[sectionKey];
  activeTabName.value = sectionConfig.tab;
  await nextTick();
  sectionConfig.targetRef.value?.$el?.scrollIntoView({ behavior: "smooth", block: "start" });
}

onMounted(() => {
  handleQuery();
});
</script>

<style scoped lang="scss">
.goods-hero-card,
.detail-section-card,
.detail-table-card {
  border: 1px solid #e5eaf1;
  border-radius: 16px;
  box-shadow: 0 8px 24px rgb(15 23 42 / 4%);
}

.goods-hero-card {
  margin-bottom: 20px;
}

:deep(.goods-hero-card .el-card__body),
:deep(.detail-section-card .el-card__body),
:deep(.detail-table-card .el-card__body) {
  padding: 16px;
}

.goods-hero {
  display: grid;
  grid-template-columns: 240px minmax(0, 1fr);
  gap: 16px;
  align-items: stretch;
}

.goods-hero__media {
  position: relative;
}

.goods-cover-image {
  width: 100%;
  height: 100%;
  min-height: 240px;
  overflow: hidden;
  border-radius: 12px;
  background: #f3f4f6;
}

.goods-hero__content {
  display: flex;
  flex-direction: column;
  gap: 14px;
  justify-content: center;
  min-width: 0;
}

.goods-hero__eyebrow {
  font-size: 12px;
  font-weight: 600;
  color: #64748b;
}

.goods-hero__title-row {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  align-items: center;
}

.goods-hero__title {
  margin: 0;
  font-size: 22px;
  font-weight: 700;
  line-height: 1.2;
  color: #1f2937;
}

.goods-hero__desc {
  margin: 0;
  font-size: 14px;
  line-height: 1.7;
  color: #64748b;
}

.goods-meta-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}

.goods-meta-item,
.goods-metric-card {
  padding: 12px 14px;
  border: 1px solid #e8edf4;
  border-radius: 12px;
  background: #f8fafc;
}

.goods-meta-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.goods-meta-item--action,
.goods-metric-card--action {
  cursor: pointer;
  transition:
    transform 0.2s ease,
    box-shadow 0.2s ease,
    border-color 0.2s ease;
}

.goods-meta-item--action {
  width: 100%;
  text-align: left;
}

.goods-metric-card--action {
  width: 100%;
  text-align: left;
}

.goods-meta-item--action:hover,
.goods-metric-card--action:hover {
  border-color: #cdd7e5;
}

.goods-meta-item__label,
.goods-metric-card__label {
  font-size: 13px;
  color: #64748b;
}

.goods-meta-item__value,
.goods-metric-card__value {
  font-size: 16px;
  font-weight: 700;
  color: #1f2937;
}

.goods-metric-list {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}

.goods-detail-tabs :deep(.el-tabs__header) {
  margin-bottom: 14px;
}

.goods-detail-tabs :deep(.el-tabs__nav-wrap::after) {
  height: 1px;
  background-color: #e6ebf2;
}

.goods-detail-tabs :deep(.el-tabs__nav) {
  gap: 8px;
}

.goods-detail-tabs :deep(.el-tabs__item) {
  height: 36px;
  padding: 0 6px;
  font-size: 14px;
  font-weight: 600;
}

.goods-detail-tabs :deep(.el-tabs__item.is-active) {
  color: #1f2937;
}

.detail-section-card {
  margin-bottom: 16px;
}

.detail-table-card {
  overflow: hidden;
}

.detail-section-card__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  font-size: 16px;
  font-weight: 600;
  color: #1f2937;
}

.detail-section-card__extra {
  font-size: 13px;
  font-weight: 500;
  color: #94a3b8;
}

.goods-descriptions :deep(.el-descriptions__label) {
  width: 110px;
  font-weight: 600;
}

.goods-descriptions :deep(.el-descriptions__cell) {
  padding: 10px 14px;
}

.image-gallery {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
  gap: 12px;
}

.image-gallery__item {
  width: 100%;
  height: 160px;
  overflow: hidden;
  border-radius: 12px;
  background: #f3f4f6;
}

.detail-image-list {
  display: grid;
  gap: 12px;
}

.detail-image-list__item {
  width: 100%;
  min-height: 260px;
  overflow: hidden;
  border-radius: 12px;
  background: #f3f4f6;
}

.sku-image-cell {
  display: flex;
  justify-content: center;
}

.sku-image-cell__thumb {
  width: 60px;
  height: 60px;
  object-fit: cover;
  border: 1px solid #dbe4f0;
  border-radius: 10px;
}

.sku-image-cell__preview {
  width: 400px;
  height: 400px;
  object-fit: cover;
  border-radius: 12px;
}

.image-placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
  min-height: 180px;
  font-size: 14px;
  color: #94a3b8;
  background: #f3f4f6;
}

.image-placeholder--large {
  min-height: 320px;
}

@media (width <= 1200px) {
  .goods-hero {
    grid-template-columns: 280px minmax(0, 1fr);
  }

  .goods-meta-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (width <= 992px) {
  .goods-hero {
    grid-template-columns: 1fr;
  }

  .goods-cover-image {
    min-height: 280px;
  }
}

@media (width <= 768px) {
  .goods-hero__title {
    font-size: 24px;
  }

  .goods-meta-grid,
  .goods-metric-list {
    grid-template-columns: 1fr;
  }

  .image-gallery {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .detail-image-list__item {
    min-height: 180px;
  }
}
</style>
