<template>
  <el-card class="gorse-overview-dashboard-card" shadow="never">
    <template #header>
      <div class="gorse-overview-dashboard-card__header">
        <div class="gorse-overview-dashboard-card__title">非个性化推荐</div>
        <div class="gorse-overview-dashboard-card__filters">
          <el-select v-model="selectedRecommender" class="gorse-overview-dashboard-card__select" filterable>
            <el-option
              v-for="recommender in recommenderOptions"
              :key="recommender.value"
              :label="recommender.label"
              :value="recommender.value"
            />
          </el-select>
          <el-select
            v-model="selectedCategory"
            class="gorse-overview-dashboard-card__select"
            clearable
            filterable
            placeholder="全部分类"
          >
            <el-option
              v-for="category in categoryOptions"
              :key="category.value"
              :label="category.label"
              :value="category.value"
            />
          </el-select>
        </div>
      </div>
    </template>
    <div v-loading="dashboardItemsLoading" class="gorse-overview-dashboard-table no-card">
      <ProTable row-key="item_id" :data="dashboardItems" :columns="dashboardColumns" :pagination="false" :tool-button="false">
        <template #comment="{ row }">
          <el-link
            v-if="BUTTONS['goods:info:detail'] && resolveDashboardGoodsId(row)"
            type="primary"
            @click.stop="handleOpenGoodsDetail(row)"
          >
            {{ row.comment || "--" }}
          </el-link>
          <span v-else>{{ row.comment || "--" }}</span>
        </template>
        <template #categories="{ row }">
          <el-tooltip
            v-if="formatDashboardCategoryText(row.categories) !== '--'"
            :content="formatDashboardCategoryText(row.categories)"
            placement="top"
          >
            <span class="gorse-overview-dashboard-table__categories">{{ formatDashboardCategoryText(row.categories) }}</span>
          </el-tooltip>
          <span v-else class="gorse-overview-dashboard-table__empty">--</span>
        </template>
        <template #timestamp="{ row }">{{ formatTimestamp(row.timestamp) }}</template>
        <template #desc="{ row }">{{ readDashboardLabelValue(row, "desc") }}</template>
        <template #price="{ row }">{{ readDashboardLabelValue(row, "price") }}</template>
        <template #discount_price="{ row }">{{ readDashboardLabelValue(row, "discount_price") }}</template>
        <template #inventory="{ row }">{{ readDashboardLabelValue(row, "inventory") }}</template>
        <template #status="{ row }">
          <DictLabel
            v-if="readDashboardLabelValue(row, 'status') !== '--'"
            :model-value="readDashboardLabelValue(row, 'status')"
            code="goods_status"
          />
          <span v-else>--</span>
        </template>
        <template #score="{ row }">
          <span class="gorse-overview-dashboard-table__score">{{ formatScore(row.score) }}</span>
        </template>
      </ProTable>
    </div>
  </el-card>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import { useRouter } from "vue-router";
import dayjs from "dayjs";
import { ElMessage } from "element-plus";
import ProTable from "@/components/ProTable/index.vue";
import type { ColumnProps } from "@/components/ProTable/interface";
import DictLabel from "@/components/Dict/DictLabel.vue";
import { defRecommendGorseService } from "@/api/admin/recommend_gorse";
import { useAuthButtons } from "@/hooks/useAuthButtons";
import { useRecommendGorseStore } from "@/stores/modules/recommendGorse";
import { navigateTo } from "@/utils/router";
import type { Item } from "@/rpc/admin/v1/recommend_gorse";

/** 非个性化推荐标签二级表头列。 */
interface DashboardLabelColumn {
  /** 后端返回的标签键。 */
  key: string;
  /** 页面展示列名。 */
  label: string;
  /** 列最小宽度。 */
  minWidth: number;
}

/** Gorse 推荐分类筛选项。 */
interface CategorySelectOption {
  /** 分类原始编号，继续作为Gorse 接口筛选值。 */
  value: string;
  /** 分类中文名称。 */
  label: string;
}

const dashboardLabelColumns: DashboardLabelColumn[] = [
  { key: "desc", label: "描述", minWidth: 180 },
  { key: "price", label: "原价（元）", minWidth: 110 },
  { key: "discount_price", label: "折扣价（元）", minWidth: 120 },
  { key: "inventory", label: "库存", minWidth: 100 },
  { key: "status", label: "状态", minWidth: 100 }
];

const dashboardColumns: ColumnProps[] = [
  { prop: "comment", label: "商品名称", minWidth: 260, showOverflowTooltip: false },
  { prop: "categories", label: "分类", minWidth: 180 },
  { prop: "timestamp", label: "时间", minWidth: 160 },
  {
    label: "标签",
    align: "center",
    _children: dashboardLabelColumns.map(column => ({
      prop: column.key,
      label: column.label,
      minWidth: column.minWidth,
      showOverflowTooltip: true
    }))
  },
  { prop: "score", label: "分数", width: 170, align: "right", showOverflowTooltip: true }
];

const dashboardItemsLoading = ref(false);
const selectedRecommender = ref("latest");
const selectedCategory = ref("");
const categories = ref<string[]>([]);
const dashboardItems = ref<Item[]>([]);
const dashboardInitialized = ref(false);
const recommendGorseStore = useRecommendGorseStore();
const dashboardItemsLimit = 100;
const router = useRouter();
const { BUTTONS } = useAuthButtons();

const recommenderOptions = computed(() => recommendGorseStore.dashboardRecommenderOptions);
const recommenders = computed(() => recommenderOptions.value.map(item => item.value));
const categoryOptions = computed<CategorySelectOption[]>(() =>
  categories.value.map(category => ({ value: category, label: formatCategoryLabel(category) }))
);

watch(
  recommenders,
  options => {
    // 推荐器下拉来自Gorse 配置 store，当前值失效时回退到第一项。
    if (options.length && !options.includes(selectedRecommender.value)) selectedRecommender.value = options[0];
  },
  { immediate: true }
);

watch([selectedRecommender, selectedCategory], () => {
  // 首次加载由当前组件初始化流程统一触发，避免初始化阶段重复请求列表。
  if (!dashboardInitialized.value) return;
  loadDashboardItems().catch(() => {
    ElMessage.error("加载非个性化推荐失败");
  });
});

/** 初始化推荐商品组件所需配置、分类和列表数据。 */
async function loadDashboardData() {
  try {
    await Promise.all([recommendGorseStore.loadConfig(), recommendGorseStore.loadGoodsCategoryOptions()]);
    normalizeSelectedRecommender();
    await Promise.all([loadCategories(), loadDashboardItems()]);
  } finally {
    dashboardInitialized.value = true;
  }
}

/** 加载 Gorse 推荐仪表盘分类下拉。 */
async function loadCategories() {
  const gorseData = await defRecommendGorseService.OptionCategories({});
  categories.value = (gorseData.categories ?? []).map(item => item.trim()).filter(Boolean);
  // 分类列表刷新后，如果当前分类不再存在，则清空筛选条件。
  if (selectedCategory.value && !categories.value.includes(selectedCategory.value)) selectedCategory.value = "";
}

/** 根据推荐器和分类加载 Gorse 推荐仪表盘商品。 */
async function loadDashboardItems() {
  dashboardItemsLoading.value = true;
  try {
    const data = await defRecommendGorseService.ListDashboardItems({
      recommender: selectedRecommender.value || "latest",
      category: selectedCategory.value,
      end: dashboardItemsLimit
    });
    dashboardItems.value = data.items ?? [];
  } finally {
    dashboardItemsLoading.value = false;
  }
}

/** 读取固定标签列对应的展示值。 */
function readDashboardLabelValue(row: Item, key: keyof NonNullable<Item["labels"]>) {
  const value = row.labels?.[key];
  if (value === undefined || value === null) return "--";
  if (key === "price" || key === "discount_price") return (Number(value || 0) / 100).toFixed(2);
  return value;
}

/** 将 Gorse 推荐分类编号转换成商品分类中文名，缺少映射时保留原值方便排查。 */
function formatCategoryLabel(category: string) {
  const value = String(category ?? "").trim();
  if (!value) return "--";
  return recommendGorseStore.categoryLabelMap[value] || value;
}

/** 将推荐返回的分类列表转换为商品列表同款分类文本。 */
function formatDashboardCategoryText(categoriesValue: string[] = []) {
  const labels = categoriesValue.map(formatCategoryLabel).filter(label => label && label !== "--");
  return labels.join("、") || "--";
}

/** 读取推荐商品详情跳转ID，优先使用 Gorse 推荐返回的商品编号。 */
function resolveDashboardGoodsId(row: Item) {
  return String(row.item_id || "").trim();
}

/** 打开推荐商品详情页。 */
function handleOpenGoodsDetail(row: Item) {
  const goodsId = resolveDashboardGoodsId(row);
  // 推荐结果缺少商品编号时无法拼装商品详情路由，直接提示并中断跳转。
  if (!goodsId) {
    ElMessage.warning("当前商品缺少详情跳转ID");
    return;
  }
  navigateTo(router, `/goods/detail/${goodsId}`);
}

/** 校正当前推荐器，确保选中值来自 store 下拉。 */
function normalizeSelectedRecommender() {
  if (recommenders.value.length && !recommenders.value.includes(selectedRecommender.value))
    selectedRecommender.value = recommenders.value[0];
}

/** 格式化推荐商品时间。 */
function formatTimestamp(value: string) {
  if (!value) return "--";
  return dayjs(value).format("YYYY-MM-DD HH:mm:ss");
}

/** 格式化推荐商品分数。 */
function formatScore(value: number) {
  return Number(value || 0).toFixed(5);
}

onMounted(() => {
  loadDashboardData().catch(() => {
    ElMessage.error("加载非个性化推荐失败");
  });
});
</script>

<style scoped lang="scss">
.gorse-overview-dashboard-card {
  border-color: var(--admin-page-card-border);
  background: var(--admin-page-card-bg);
  box-shadow: var(--admin-page-shadow);

  &__header {
    display: flex;
    gap: 16px;
    align-items: center;
    justify-content: space-between;
  }

  &__title {
    color: var(--admin-page-text-primary);
    font-size: 16px;
    font-weight: 700;
    line-height: 24px;
  }

  &__filters {
    display: flex;
    flex-wrap: wrap;
    gap: 12px;
    justify-content: flex-end;
  }

  &__select {
    width: 260px;
    max-width: 100%;
  }
}

.gorse-overview-dashboard-table {
  width: 100%;

  /* ProTable 在卡片内部使用时不继承通用列表页固定高度，避免页面底部出现多余空白。 */
  :deep(.table-main) {
    height: auto;
    min-height: auto;
  }

  :deep(.el-table) {
    flex: initial;
  }

  :deep(.el-table__inner-wrapper),
  :deep(.el-table__body-wrapper),
  :deep(.el-scrollbar),
  :deep(.el-scrollbar__wrap),
  :deep(.el-scrollbar__view) {
    height: auto;
  }

  &__categories {
    display: inline-block;
    width: 100%;
    overflow: hidden;
    vertical-align: middle;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  &__empty {
    color: var(--admin-page-text-secondary);
  }

  &__score {
    white-space: nowrap;
  }
}

@media (max-width: 900px) {
  .gorse-overview-dashboard-card__header {
    align-items: flex-start;
    flex-direction: column;
  }

  .gorse-overview-dashboard-card__filters {
    justify-content: flex-start;
    width: 100%;
  }
}

@media (max-width: 560px) {
  .gorse-overview-dashboard-card__select {
    width: 100%;
  }
}
</style>
