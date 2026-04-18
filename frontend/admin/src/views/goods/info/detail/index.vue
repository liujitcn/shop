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
          <div class="goods-hero__head">
            <div class="goods-hero__title-wrap">
              <span class="goods-hero__eyebrow">商品概览</span>
              <h1 class="goods-hero__title">{{ formData.name || "-" }}</h1>
            </div>
            <el-tag :type="statusTagType" effect="light" round>{{ statusText }}</el-tag>
          </div>

          <p class="goods-hero__desc">{{ formData.desc || "暂无描述" }}</p>

          <div class="goods-summary-grid">
            <div class="goods-summary-card">
              <span class="goods-summary-card__label">分类</span>
              <strong class="goods-summary-card__value">{{ formData.categoryName || "-" }}</strong>
            </div>
            <div class="goods-summary-card">
              <span class="goods-summary-card__label">SKU</span>
              <strong class="goods-summary-card__value">{{ skuCount }} 个</strong>
            </div>
            <div class="goods-summary-card">
              <span class="goods-summary-card__label">规格</span>
              <strong class="goods-summary-card__value">{{ specCount }} 项</strong>
            </div>
            <div class="goods-summary-card">
              <span class="goods-summary-card__label">库存</span>
              <strong class="goods-summary-card__value">{{ totalInventory }}</strong>
            </div>
          </div>

          <div class="goods-quick-links">
            <button type="button" class="goods-quick-link" @click="handleNavigateSection('banner')">
              <span>轮播图</span>
              <strong>{{ bannerCount }} 张</strong>
            </button>
            <button type="button" class="goods-quick-link" @click="handleNavigateSection('detail')">
              <span>详情图</span>
              <strong>{{ detailCount }} 张</strong>
            </button>
            <button type="button" class="goods-quick-link" @click="handleNavigateSection('prop')">
              <span>属性</span>
              <strong>{{ propCount }} 项</strong>
            </button>
            <button type="button" class="goods-quick-link" @click="handleNavigateSection('spec')">
              <span>规格</span>
              <strong>{{ specCount }} 项</strong>
            </button>
            <button type="button" class="goods-quick-link" @click="handleNavigateSection('sku')">
              <span>库存</span>
              <strong>{{ skuCount }} 行</strong>
            </button>
          </div>
        </div>
      </div>
    </el-card>

    <el-tabs v-model="activeTabName" class="goods-detail-tabs">
      <el-tab-pane label="信息" name="basic">
        <el-card class="detail-section-card" shadow="never">
          <template #header>
            <div class="detail-section-card__header">
              <span>基础信息</span>
            </div>
          </template>

          <el-descriptions :column="2" border class="goods-descriptions">
            <el-descriptions-item label="分类">{{ formData.categoryName || "-" }}</el-descriptions-item>
            <el-descriptions-item label="状态">
              <el-tag :type="statusTagType" effect="light" round>{{ statusText }}</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="标题" :span="2">{{ formData.name || "-" }}</el-descriptions-item>
            <el-descriptions-item label="描述" :span="2">{{ formData.desc || "-" }}</el-descriptions-item>
            <el-descriptions-item label="轮播图">{{ bannerCount }} 张</el-descriptions-item>
            <el-descriptions-item label="详情图">{{ detailCount }} 张</el-descriptions-item>
            <el-descriptions-item label="SKU">{{ skuCount }} 个</el-descriptions-item>
            <el-descriptions-item label="库存">{{ totalInventory }}</el-descriptions-item>
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
              <span>详情图</span>
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

      <el-tab-pane label="属性" name="prop">
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

      <el-tab-pane label="规格" name="spec">
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

      <el-tab-pane label="库存" name="sku">
        <el-card ref="skuSectionRef" class="detail-table-card" shadow="never">
          <template #header>
            <div class="detail-section-card__header">
              <span>SKU 库存</span>
              <span class="detail-section-card__extra">{{ skuCount }} 行</span>
            </div>
          </template>

          <ProTable row-key="skuCode" :data="formData.skuList" :columns="skuColumns" :pagination="false" :tool-button="false">
            <template #picture="scope">
              <div class="sku-image-cell">
                <el-image
                  v-if="getSkuPictureSrc(scope.row.picture)"
                  class="sku-image-cell__thumb"
                  :src="getSkuPictureSrc(scope.row.picture)"
                  :preview-src-list="getSkuPicturePreviewList(scope.row.picture)"
                  fit="cover"
                  preview-teleported
                >
                  <template #error>
                    <div class="sku-image-cell__empty">暂无规格图</div>
                  </template>
                </el-image>
                <div v-else class="sku-image-cell__empty">暂无规格图</div>
              </div>
            </template>
          </ProTable>
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
import { type GoodsInfoForm } from "@/rpc/admin/goods_info";
import { type GoodsProp } from "@/rpc/admin/goods_prop";
import { type GoodsSku } from "@/rpc/admin/goods_sku";
import { type GoodsSpec } from "@/rpc/admin/goods_spec";
import { defGoodsInfoService } from "@/api/admin/goods_info";
import { GoodsStatus } from "@/rpc/common/enum";
import { formatSrc } from "@/utils/utils";

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

const formData = reactive<GoodsInfoForm>({
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
const statusText = computed(() => (formData.status === GoodsStatus.PUT_ON ? "上架" : "下架"));

/** 统一计算页面顶部商品状态标签样式。 */
const statusTagType = computed(() => (formData.status === GoodsStatus.PUT_ON ? "success" : "info"));

/** 统计轮播图数量。 */
const bannerCount = computed(() => formData.banner.length);

/** 统计详情图数量。 */
const detailCount = computed(() => formData.detail.length);

/** 统计属性项数量。 */
const propCount = computed(() => formData.propList.length);

/** 统计规格项数量。 */
const specCount = computed(() => formData.specList.length);

/** 统计 SKU 数量。 */
const skuCount = computed(() => formData.skuList.length);

/** 详情页按 SKU 汇总库存，避免额外维护聚合字段。 */
const totalInventory = computed(() => formData.skuList.reduce((total, item) => total + Number(item.inventory ?? 0), 0));

/** 页面分区和标签页映射，统一控制概览区跳转行为。 */
const detailSectionMap: Record<GoodsDetailSectionKey, { tab: GoodsDetailTabName; targetRef: typeof bannerSectionRef }> = {
  banner: { tab: "basic", targetRef: bannerSectionRef },
  detail: { tab: "basic", targetRef: detailSectionRef },
  sku: { tab: "sku", targetRef: skuSectionRef },
  spec: { tab: "spec", targetRef: specSectionRef },
  prop: { tab: "prop", targetRef: propSectionRef }
};

/** 商品属性明细表格列配置。 */
const propColumns: ColumnProps[] = [
  { prop: "label", label: "名称", minWidth: 180 },
  { prop: "value", label: "内容", minWidth: 220 },
  { prop: "sort", label: "排序", align: "right", minWidth: 100 }
];

/** 商品规格明细表格列配置。 */
const specColumns: ColumnProps[] = [
  { prop: "name", label: "名称", minWidth: 160 },
  {
    prop: "item",
    label: "规格值",
    minWidth: 220,
    render: scope => (Array.isArray(scope.row.item) ? scope.row.item.join(" / ") : "--")
  },
  { prop: "sort", label: "排序", align: "right", minWidth: 100 }
];

/** 商品 SKU 明细表格列配置。 */
const skuColumns = computed<ColumnProps[]>(() => {
  // 根据规格定义动态拼接表头，保证不同商品规格也能完整展示。
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
      label: "规格组合",
      minWidth: 180,
      render: scope => ((scope.row.specItem as string[] | undefined) ?? []).join(" / ")
    },
    { prop: "picture", label: "规格图", minWidth: 110, align: "center" },
    { prop: "skuCode", label: "SKU", minWidth: 160 },
    { prop: "initSaleNum", label: "初始销量", align: "right", minWidth: 100 },
    { prop: "realSaleNum", label: "销量", align: "right", minWidth: 100 },
    { prop: "price", label: "售价", align: "right", minWidth: 110, cellType: "money" },
    { prop: "discountPrice", label: "折后价", align: "right", minWidth: 130, cellType: "money" },
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
  defGoodsInfoService
    .GetGoodsInfo({
      value: goodsId.value
    })
    .then(data => {
      data.skuList.forEach(item => {
        // 将规格数组转成扁平字段，避免在表格单元格里重复解析。
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

/** 统一补齐 SKU 规格图地址，兼容后端返回相对路径的场景。 */
function getSkuPictureSrc(picture?: string) {
  const value = String(picture ?? "").trim();
  if (!value) return "";
  return formatSrc(value);
}

/** 统一生成规格图预览列表，避免图片为空时传入无效预览数据。 */
function getSkuPicturePreviewList(picture?: string) {
  const pictureSrc = getSkuPictureSrc(picture);
  return pictureSrc ? [pictureSrc] : [];
}

/**
 * 从顶部快捷入口跳转到对应分区，先切换标签页，再滚动到目标区域。
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
  border: 1px solid var(--admin-page-card-border);
  border-radius: 16px;
  background: var(--admin-page-card-bg);
  box-shadow: var(--admin-page-shadow);
}

.goods-hero-card {
  margin-bottom: 18px;
}

:deep(.goods-hero-card .el-card__body),
:deep(.detail-section-card .el-card__body),
:deep(.detail-table-card .el-card__body) {
  padding: 16px;
}

.goods-hero {
  display: grid;
  grid-template-columns: 220px minmax(0, 1fr);
  gap: 18px;
  align-items: stretch;
}

.goods-cover-image {
  width: 100%;
  height: 100%;
  min-height: 220px;
  overflow: hidden;
  border-radius: 14px;
  background: var(--admin-page-card-bg-muted);
}

.goods-hero__content {
  display: flex;
  flex-direction: column;
  gap: 14px;
  min-width: 0;
}

.goods-hero__head {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  align-items: flex-start;
  justify-content: space-between;
}

.goods-hero__title-wrap {
  min-width: 0;
}

.goods-hero__eyebrow {
  font-size: 12px;
  font-weight: 600;
  color: var(--admin-page-text-secondary);
}

.goods-hero__title {
  margin: 6px 0 0;
  font-size: 22px;
  font-weight: 700;
  line-height: 1.25;
  color: var(--admin-page-text-primary);
}

.goods-hero__desc {
  margin: 0;
  font-size: 14px;
  line-height: 1.7;
  color: var(--admin-page-text-secondary);
}

.goods-summary-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}

.goods-summary-card,
.goods-quick-link {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 12px 14px;
  border: 1px solid var(--admin-page-card-border-soft);
  border-radius: 12px;
  background: var(--admin-page-card-bg-soft);
}

.goods-summary-card__label,
.goods-quick-link span {
  font-size: 12px;
  color: var(--admin-page-text-secondary);
}

.goods-summary-card__value,
.goods-quick-link strong {
  font-size: 16px;
  font-weight: 700;
  color: var(--admin-page-text-primary);
}

.goods-quick-links {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 10px;
}

.goods-quick-link {
  width: 100%;
  text-align: left;
  cursor: pointer;
  transition: border-color 0.2s ease;
}

.goods-quick-link:hover {
  border-color: var(--admin-page-card-border-muted);
}

.goods-detail-tabs :deep(.el-tabs__header) {
  margin-bottom: 14px;
}

.goods-detail-tabs :deep(.el-tabs__nav-wrap::after) {
  height: 1px;
  background-color: var(--admin-page-divider-strong);
}

.goods-detail-tabs :deep(.el-tabs__item) {
  height: 36px;
  padding: 0 8px;
  font-size: 14px;
  font-weight: 600;
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
  color: var(--admin-page-text-primary);
}

.detail-section-card__extra {
  font-size: 13px;
  font-weight: 500;
  color: var(--admin-page-text-placeholder);
}

.goods-descriptions :deep(.el-descriptions__label) {
  width: 96px;
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
  background: var(--admin-page-card-bg-muted);
}

.detail-image-list {
  display: grid;
  gap: 12px;
}

.detail-image-list__item {
  width: 100%;
  min-height: 240px;
  overflow: hidden;
  border-radius: 12px;
  background: var(--admin-page-card-bg-muted);
}

.sku-image-cell {
  display: flex;
  justify-content: center;
}

.sku-image-cell__thumb {
  width: 60px;
  height: 60px;
  overflow: hidden;
  border: 1px solid var(--admin-page-card-border-muted);
  border-radius: 10px;
  background: var(--admin-page-card-bg-muted);
}

.sku-image-cell__empty {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 60px;
  height: 60px;
  padding: 6px;
  font-size: 12px;
  line-height: 1.4;
  text-align: center;
  color: var(--admin-page-text-placeholder);
  border: 1px solid var(--admin-page-card-border-muted);
  border-radius: 10px;
  background: var(--admin-page-card-bg-muted);
}

.image-placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
  min-height: 180px;
  font-size: 14px;
  color: var(--admin-page-text-placeholder);
  background: var(--admin-page-card-bg-muted);
}

@media (width <= 1200px) {
  .goods-summary-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .goods-quick-links {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}

@media (width <= 992px) {
  .goods-hero {
    grid-template-columns: 1fr;
  }

  .goods-cover-image {
    min-height: 260px;
  }
}

@media (width <= 768px) {
  .goods-summary-grid,
  .goods-quick-links {
    grid-template-columns: 1fr;
  }

  .detail-image-list__item {
    min-height: 180px;
  }
}
</style>
