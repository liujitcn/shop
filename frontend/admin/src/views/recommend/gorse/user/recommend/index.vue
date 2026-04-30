<template>
  <div class="gorse-detail-page">
    <div class="table-box gorse-user-recommend-table">
      <ProTable
        ref="proTable"
        v-loading="loading"
        row-key="item_id"
        :data="recommendList"
        :columns="columns"
        :pagination="false"
        :tool-button="['refresh', 'setting', 'search']"
        @refresh="handleRefresh"
        @search="handleSearch"
        @reset="handleReset"
      >
        <template #comment="{ row }">
          <el-link v-if="resolveGoodsId(row)" type="primary" @click.stop="handleOpenGoodsDetail(row)">
            {{ row.comment || "--" }}
          </el-link>
          <span v-else>{{ row.comment || "--" }}</span>
        </template>
        <template #categories="{ row }">
          <el-tooltip
            v-if="formatCategoryText(row.categories) !== '--'"
            :content="formatCategoryText(row.categories)"
            placement="top"
          >
            <span class="gorse-user-recommend-table__categories">{{ formatCategoryText(row.categories) }}</span>
          </el-tooltip>
          <span v-else>--</span>
        </template>
        <template #timestamp="{ row }">{{ formatTimestamp(row.timestamp) }}</template>
        <template #desc="{ row }">{{ formatGoodsLabelValue(row, "desc") }}</template>
        <template #price="{ row }">{{ formatGoodsLabelValue(row, "price") }}</template>
        <template #discount_price="{ row }">{{ formatGoodsLabelValue(row, "discount_price") }}</template>
        <template #inventory="{ row }">{{ formatGoodsLabelValue(row, "inventory") }}</template>
        <template #status="{ row }">
          <DictLabel
            v-if="formatGoodsLabelValue(row, 'status') !== '--'"
            :model-value="formatGoodsLabelValue(row, 'status')"
            code="goods_status"
          />
          <span v-else>--</span>
        </template>
        <template #score="{ row }">{{ formatScore(row.score) }}</template>
      </ProTable>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import dayjs from "dayjs";
import { ElMessage } from "element-plus";
import ProTable from "@/components/ProTable/index.vue";
import type { ColumnProps, ProTableInstance } from "@/components/ProTable/interface";
import DictLabel from "@/components/Dict/DictLabel.vue";
import { defRecommendGorseService } from "@/api/admin/recommend_gorse";
import { buildScopedGoodsCategoryTree, useRecommendGorseStore } from "@/stores/modules/recommendGorse";
import { navigateTo } from "@/utils/router";
import type { Item } from "@/rpc/admin/v1/recommend_gorse";

const route = useRoute();
const router = useRouter();
const userId = computed(() => String(route.params.userId ?? ""));
const recommendGorseStore = useRecommendGorseStore();
const proTable = ref<ProTableInstance>();
const loading = ref(false);
const recommendList = ref<Item[]>([]);
const gorseCategoryIds = ref<string[]>([]);
const recommenderOptions = computed(() => recommendGorseStore.userRecommendRecommenderOptions);

/** 用户推荐结果表格列配置。 */
const columns: ColumnProps[] = [
  {
    prop: "recommender",
    label: "推荐器",
    isShow: false,
    isSetting: false,
    enum: recommenderOptions,
    search: {
      el: "select",
      order: 1,
      props: {
        clearable: true,
        filterable: true,
        placeholder: "请选择推荐器"
      }
    }
  },
  {
    prop: "category",
    label: "商品分类",
    isShow: false,
    isSetting: false,
    enum: computed(() => buildScopedGoodsCategoryTree(recommendGorseStore.categoryTreeOptions, gorseCategoryIds.value)),
    search: {
      el: "tree-select",
      order: 2,
      props: {
        clearable: true,
        checkStrictly: true,
        filterable: true,
        placeholder: "请选择商品分类",
        renderAfterExpand: false,
        style: { width: "100%" }
      }
    }
  },
  {
    prop: "limit",
    label: "返回数量",
    isShow: false,
    isSetting: false,
    search: {
      el: "input-number",
      order: 3,
      defaultValue: 100,
      props: {
        min: 1,
        max: 500,
        controlsPosition: "right",
        style: { width: "100%" }
      }
    }
  },
  { prop: "comment", label: "商品名称", minWidth: 260, showOverflowTooltip: false },
  { prop: "categories", label: "分类", minWidth: 220, showOverflowTooltip: false },
  { prop: "timestamp", label: "时间", minWidth: 170 },
  {
    label: "标签",
    align: "center",
    _children: [
      { prop: "desc", label: "描述", minWidth: 180, showOverflowTooltip: true },
      { prop: "price", label: "原价（元）", minWidth: 110, showOverflowTooltip: true },
      { prop: "discount_price", label: "折扣价（元）", minWidth: 120, showOverflowTooltip: true },
      { prop: "inventory", label: "库存", minWidth: 100, showOverflowTooltip: true },
      { prop: "status", label: "状态", minWidth: 100, showOverflowTooltip: true }
    ]
  },
  { prop: "score", label: "分数", width: 140, align: "right", showOverflowTooltip: true }
];

watch(
  userId,
  value => {
    if (!value) {
      recommendList.value = [];
      return;
    }
    loadRecommend().catch(() => {
      ElMessage.error("加载用户推荐结果失败");
    });
  },
  { immediate: true }
);

/** 加载 Gorse 推荐可用分类ID。 */
async function loadGorseCategories() {
  const gorseData = await defRecommendGorseService.OptionCategories({});
  gorseCategoryIds.value = (gorseData.categories ?? []).map(item => String(item).trim()).filter(Boolean);
}

/** 加载用户推荐结果。 */
async function loadRecommend() {
  if (!userId.value) return;
  loading.value = true;
  try {
    const data = await defRecommendGorseService.GetUserRecommend({
      id: userId.value,
      recommender: resolveSelectedRecommender(),
      category: String(proTable.value?.searchParam?.category ?? "").trim(),
      n: resolveSelectedLimit()
    });
    recommendList.value = data.items ?? [];
  } finally {
    loading.value = false;
  }
}

/** 读取当前选中的推荐器，空值表示走Gorse默认个性化推荐。 */
function resolveSelectedRecommender() {
  return String(proTable.value?.searchParam?.recommender ?? "").trim();
}

/** 读取当前选中的返回数量，异常值回退到Gorse 接口默认 100。 */
function resolveSelectedLimit() {
  const limit = Number(proTable.value?.searchParam?.limit || 100);
  // 搜索条件为空或非法时，统一按Gorse 推荐默认返回条数请求。
  if (!Number.isFinite(limit) || limit <= 0) return 100;
  return limit;
}

/** 刷新用户推荐结果。 */
function handleRefresh() {
  loadRecommend().catch(() => {
    ElMessage.error("加载用户推荐结果失败");
  });
}

/** 响应标准搜索事件。 */
function handleSearch() {
  loadRecommend().catch(() => {
    ElMessage.error("加载用户推荐结果失败");
  });
}

/** 响应标准重置事件。 */
function handleReset() {
  loadRecommend().catch(() => {
    ElMessage.error("加载用户推荐结果失败");
  });
}

/** 读取商品标签值，价格字段按分转元展示。 */
function formatGoodsLabelValue(row: Item, key: keyof NonNullable<Item["labels"]>) {
  const value = row.labels?.[key];
  if (value === undefined || value === null) return "--";
  if (key === "price" || key === "discount_price") return (Number(value || 0) / 100).toFixed(2);
  return value;
}

/** 将Gorse 分类列表转换成完整中文路径。 */
function formatCategoryText(categories: string[] = []) {
  const labels = categories
    .map(category => recommendGorseStore.categoryLabelMap[String(category)] || String(category))
    .filter(Boolean);
  return labels.join("、") || "--";
}

/** 读取商品详情跳转编号。 */
function resolveGoodsId(item?: Item) {
  return String(item?.item_id || "").trim();
}

/** 打开商品详情页。 */
function handleOpenGoodsDetail(item?: Item) {
  const goodsId = resolveGoodsId(item);
  if (!goodsId) {
    ElMessage.warning("当前商品缺少详情跳转ID");
    return;
  }
  navigateTo(router, `/goods/detail/${goodsId}`);
}

/** 格式化Gorse 推荐用户时间。 */
function formatTimestamp(value: string) {
  if (!value || value.startsWith("0001-01-01")) return "--";
  return dayjs(value).format("YYYY-MM-DD HH:mm:ss");
}

/** 格式化推荐分数。 */
function formatScore(value: number) {
  return Number(value || 0).toFixed(5);
}

onMounted(() => {
  Promise.all([recommendGorseStore.loadConfig(), recommendGorseStore.loadGoodsCategoryOptions(), loadGorseCategories()]).catch(
    () => {
      ElMessage.error("加载推荐分类失败");
    }
  );
});
</script>

<style scoped lang="scss">
.gorse-detail-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.gorse-user-recommend-table {
  &__categories {
    display: inline-block;
    max-width: 100%;
    overflow: hidden;
    text-overflow: ellipsis;
    vertical-align: middle;
    white-space: nowrap;
  }
}
</style>
