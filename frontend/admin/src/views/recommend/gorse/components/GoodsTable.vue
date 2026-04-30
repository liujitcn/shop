<template>
  <div v-loading="loading || categoryLoading" class="gorse-goods-table no-card">
    <ProTable row-key="item_id" :data="data" :columns="tableColumns" :pagination="false" :tool-button="false">
      <template #comment="{ row }">
        <el-link
          v-if="BUTTONS['goods:info:detail'] && resolveGoodsId(row)"
          type="primary"
          @click.stop="handleOpenGoodsDetail(row)"
        >
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
          <span class="gorse-goods-table__categories">{{ formatCategoryText(row.categories) }}</span>
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
      <template v-if="showOperation" #operation="{ row }">
        <slot name="operation" :row="row" />
      </template>
    </ProTable>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import dayjs from "dayjs";
import { ElMessage } from "element-plus";
import ProTable from "@/components/ProTable/index.vue";
import type { ColumnProps } from "@/components/ProTable/interface";
import DictLabel from "@/components/Dict/DictLabel.vue";
import { useAuthButtons } from "@/hooks/useAuthButtons";
import { useRecommendGorseStore } from "@/stores/modules/recommendGorse";
import { navigateTo } from "@/utils/router";
import type { Item } from "@/rpc/admin/v1/recommend_gorse";

/** Gorse 推荐商品表格属性。 */
interface GoodsTableProps {
  /** 商品列表。 */
  data: Item[];
  /** 外部加载状态。 */
  loading?: boolean;
  /** 是否展示分数列。 */
  showScore?: boolean;
  /** 是否展示操作列。 */
  showOperation?: boolean;
  /** 操作列宽度。 */
  operationWidth?: number;
}

const props = withDefaults(defineProps<GoodsTableProps>(), {
  loading: false,
  showScore: true,
  showOperation: false,
  operationWidth: 140
});

const router = useRouter();
const { BUTTONS } = useAuthButtons();
const recommendGorseStore = useRecommendGorseStore();
const categoryLoading = ref(false);

const tableColumns = computed<ColumnProps[]>(() => {
  const columns: ColumnProps[] = [
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
    }
  ];
  // 推荐结果和相似商品页需要分数，普通商品列表不强制展示分数。
  if (props.showScore) columns.push({ prop: "score", label: "分数", width: 140, align: "right", showOverflowTooltip: true });
  if (props.showOperation) {
    columns.push({ prop: "operation", label: "操作", width: props.operationWidth, fixed: "right" });
  }
  return columns;
});

/** 读取商品标签值，价格字段按分转元展示。 */
function formatGoodsLabelValue(row: Item, key: keyof NonNullable<Item["labels"]>) {
  const value = row.labels?.[key];
  if (value === undefined || value === null) return "--";
  // Gorse 推荐标签里的价格单位为分，页面统一转为元展示。
  if (key === "price" || key === "discount_price") return (Number(value || 0) / 100).toFixed(2);
  return value;
}

/** 将Gorse 分类编号转换成完整中文路径。 */
function formatCategoryLabel(category: string) {
  const value = String(category ?? "").trim();
  if (!value) return "--";
  return recommendGorseStore.categoryLabelMap[value] || value;
}

/** 将Gorse 分类列表转换成商品列表同款分类文本。 */
function formatCategoryText(categories: string[] = []) {
  const labels = categories.map(formatCategoryLabel).filter(label => label && label !== "--");
  return labels.join("、") || "--";
}

/** 读取商品详情跳转编号，标签不再承载商品编号，只使用 Gorse 推荐顶层 item_id。 */
function resolveGoodsId(row: Item) {
  return String(row.item_id || "").trim();
}

/** 打开商品详情页。 */
function handleOpenGoodsDetail(row: Item) {
  const goodsId = resolveGoodsId(row);
  // 推荐结果缺少商品编号时无法拼装商品详情路由，直接提示并中断跳转。
  if (!goodsId) {
    ElMessage.warning("当前商品缺少详情跳转ID");
    return;
  }
  navigateTo(router, `/goods/detail/${goodsId}`);
}

/** 格式化时间。 */
function formatTimestamp(value: string) {
  if (!value || value.startsWith("0001-01-01")) return "--";
  return dayjs(value).format("YYYY-MM-DD HH:mm:ss");
}

/** 格式化推荐分数。 */
function formatScore(value: number) {
  return Number(value || 0).toFixed(5);
}

onMounted(() => {
  categoryLoading.value = true;
  recommendGorseStore
    .loadGoodsCategoryOptions()
    .catch(() => {
      ElMessage.error("加载商品分类失败");
    })
    .finally(() => {
      categoryLoading.value = false;
    });
});
</script>

<style scoped lang="scss">
.gorse-goods-table {
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
