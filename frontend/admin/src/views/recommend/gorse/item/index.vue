<template>
  <div class="table-box gorse-item-page">
    <ProTable
      v-loading="loading || categoryLoading"
      ref="proTable"
      row-key="item_id"
      :data="filteredItemList"
      :columns="columns"
      :pagination="false"
      :tool-button="['refresh', 'setting', 'search']"
      @refresh="handleRefresh"
      @search="handleSearch"
      @reset="handleReset"
    >
      <template #item_id="{ row }">{{ row.item_id || "--" }}</template>
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
          <span class="gorse-item-table__categories">{{ formatCategoryText(row.categories) }}</span>
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
    </ProTable>

    <div class="gorse-cursor-pagination">
      <el-button :disabled="!cursorStack.length || loading" @click="handlePrevPage">上一页</el-button>
      <el-button type="primary" :disabled="!nextCursor || loading" @click="handleNextPage">下一页</el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Delete, View } from "@element-plus/icons-vue";
import { computed, onMounted, ref, watch } from "vue";
import { useRouter } from "vue-router";
import dayjs from "dayjs";
import { ElMessage, ElMessageBox } from "element-plus";
import ProTable from "@/components/ProTable/index.vue";
import type { ColumnProps, ProTableInstance } from "@/components/ProTable/interface";
import DictLabel from "@/components/Dict/DictLabel.vue";
import { defRecommendGorseService } from "@/api/admin/recommend_gorse";
import { useAuthButtons } from "@/hooks/useAuthButtons";
import { useRecommendGorseStore } from "@/stores/modules/recommendGorse";
import { navigateTo } from "@/utils/router";
import type { Item } from "@/rpc/admin/v1/recommend_gorse";

const router = useRouter();
const { BUTTONS } = useAuthButtons();
const recommendGorseStore = useRecommendGorseStore();
const loading = ref(false);
const categoryLoading = ref(false);
const proTable = ref<ProTableInstance>();
const pageSize = ref(10);
const currentCursor = ref("");
const nextCursor = ref("");
const cursorStack = ref<string[]>([]);
const itemList = ref<Item[]>([]);
const filteredItemList = ref<Item[]>([]);

/** 推荐商品表格列配置。 */
const columns = computed<ColumnProps[]>(() => [
  {
    prop: "item_id",
    label: "商品ID",
    minWidth: 160,
    showOverflowTooltip: true,
    search: { el: "input", order: 1, props: { placeholder: "请输入商品ID" } }
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
  {
    prop: "operation",
    label: "操作",
    width: 150,
    fixed: "right",
    cellType: "actions",
    actions: [
      {
        label: "相似",
        type: "primary",
        link: true,
        icon: View,
        onClick: scope => handleOpenSimilar(scope.row as Item)
      },
      {
        label: "删除",
        type: "danger",
        link: true,
        icon: Delete,
        onClick: scope => handleDelete(scope.row as Item)
      }
    ]
  }
]);

watch(
  itemList,
  () => {
    applyItemFilter();
  },
  { deep: true, immediate: true }
);

/** 加载推荐商品游标分页数据。 */
async function loadItemPage(cursor = currentCursor.value) {
  loading.value = true;
  try {
    const data = await defRecommendGorseService.PageItems({ cursor, n: pageSize.value });
    currentCursor.value = cursor;
    nextCursor.value = data.cursor || "";
    itemList.value = data.items ?? [];
  } finally {
    loading.value = false;
  }
}

/** 刷新当前游标页商品数据。 */
function handleRefresh() {
  loadItemPage().catch(() => {
    ElMessage.error("刷新推荐商品失败");
  });
}

/** 按当前搜索条件过滤推荐商品列表。 */
function applyItemFilter() {
  const keyword = String(proTable.value?.searchParam?.item_id ?? "").trim();
  if (!keyword) {
    filteredItemList.value = [...itemList.value];
    return;
  }
  filteredItemList.value = itemList.value.filter(item => String(item.item_id || "").includes(keyword));
}

/** 响应公共搜索事件，按商品ID过滤当前页数据。 */
function handleSearch() {
  applyItemFilter();
}

/** 响应公共重置事件，清空商品ID筛选结果。 */
function handleReset() {
  applyItemFilter();
}

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

/** 读取商品详情跳转编号，只使用 Gorse 推荐顶层 item_id。 */
function resolveGoodsId(row: Item) {
  return String(row.item_id || "").trim();
}

/** 打开商品详情页。 */
function handleOpenGoodsDetail(row: Item) {
  const goodsId = resolveGoodsId(row);
  if (!goodsId) {
    ElMessage.warning("当前商品缺少详情跳转ID");
    return;
  }
  navigateTo(router, `/goods/detail/${goodsId}`);
}

/** 打开相似商品页面。 */
function handleOpenSimilar(row: Item) {
  if (!row.item_id) {
    ElMessage.warning("当前商品缺少商品ID");
    return;
  }
  navigateTo(router, `/recommend/gorse/item/similar/${row.item_id}`);
}

/** 删除 Gorse 推荐商品。 */
function handleDelete(row: Item) {
  const goodsText = row.comment || row.item_id;
  ElMessageBox.confirm(`是否确定删除 Gorse 推荐商品？\n商品：${goodsText}`, "警告", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning"
  }).then(
    () => {
      defRecommendGorseService.DeleteItem({ id: row.item_id }).then(() => {
        ElMessage.success("删除 Gorse 推荐商品成功");
        loadItemPage().catch(() => {
          ElMessage.error("刷新推荐商品失败");
        });
      });
    },
    () => {
      ElMessage.info("已取消删除 Gorse 推荐商品");
    }
  );
}

/** 跳转下一页游标数据。 */
function handleNextPage() {
  if (!nextCursor.value) return;
  cursorStack.value.push(currentCursor.value);
  loadItemPage(nextCursor.value).catch(() => {
    ElMessage.error("加载推荐商品失败");
  });
}

/** 返回上一页游标数据。 */
function handlePrevPage() {
  const previousCursor = cursorStack.value.pop();
  if (previousCursor === undefined) return;
  loadItemPage(previousCursor).catch(() => {
    ElMessage.error("加载推荐商品失败");
  });
}

/** 格式化时间。 */
function formatTimestamp(value: string) {
  if (!value || value.startsWith("0001-01-01")) return "--";
  return dayjs(value).format("YYYY-MM-DD HH:mm:ss");
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
  loadItemPage("").catch(() => {
    ElMessage.error("加载推荐商品失败");
  });
});
</script>

<style scoped lang="scss">
.gorse-item-table {
  &__categories {
    display: inline-block;
    max-width: 100%;
    overflow: hidden;
    text-overflow: ellipsis;
    vertical-align: middle;
    white-space: nowrap;
  }
}

.gorse-cursor-pagination {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 16px 0 0;
}
</style>
