<template>
  <div class="gorse-detail-page">
    <div class="table-box">
      <ProTable
        ref="proTable"
        row-key="feedbackKey"
        :data="feedbackList"
        :columns="columns"
        :pagination="false"
        :tool-button="['refresh', 'setting', 'search']"
        @refresh="handleRefresh"
        @search="handleSearch"
        @reset="handleReset"
      >
        <template #feedback_type="{ row }">
          <el-tag effect="light">{{ formatFeedbackType(row.feedback_type) }}</el-tag>
        </template>
        <template #goods_name="{ row }">
          <el-link v-if="resolveGoodsId(row.item)" type="primary" @click.stop="handleOpenGoodsDetail(row.item)">
            {{ row.item?.comment || "--" }}
          </el-link>
          <span v-else>{{ row.item?.comment || "--" }}</span>
        </template>
        <template #categories="{ row }">{{ formatCategoryText(row.item?.categories ?? []) }}</template>
        <template #timestamp="{ row }">{{ formatTimestamp(row.timestamp) }}</template>
        <template #pagination>
          <div class="gorse-cursor-pagination">
            <el-button :disabled="offset <= 0 || loading" @click="handlePrevPage">上一页</el-button>
            <el-button type="primary" :disabled="feedbackList.length < resolvePageSize() || loading" @click="handleNextPage">
              下一页
            </el-button>
          </div>
        </template>
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
import { defRecommendGorseService } from "@/api/admin/recommend_gorse";
import { useRecommendGorseStore } from "@/stores/modules/recommendGorse";
import { navigateTo } from "@/utils/router";
import type { Feedback, Item } from "@/rpc/admin/v1/recommend_gorse";

/** 用户反馈表格行。 */
type FeedbackRow = Feedback & {
  /** 前端表格唯一键。 */
  feedbackKey: string;
};

/** 反馈类型筛选项。 */
interface FeedbackTypeOption {
  /** 接口原始值。 */
  value: string;
  /** 页面显示值。 */
  label: string;
}

/** 翻页数量筛选项。 */
interface PageSizeOption {
  /** 请求页大小。 */
  value: number;
  /** 页面显示文案。 */
  label: string;
}

const route = useRoute();
const router = useRouter();
const userId = computed(() => String(route.params.userId ?? ""));
const recommendGorseStore = useRecommendGorseStore();
const proTable = ref<ProTableInstance>();
const loading = ref(false);
const offset = ref(0);
const feedbackList = ref<FeedbackRow[]>([]);

const feedbackTypeOptions: FeedbackTypeOption[] = [
  { value: "EXPOSURE", label: "曝光" },
  { value: "VIEW", label: "浏览" },
  { value: "CLICK", label: "点击" },
  { value: "COLLECT", label: "收藏" },
  { value: "ADD_CART", label: "加购" },
  { value: "ORDER_CREATE", label: "下单" },
  { value: "ORDER_PAY", label: "支付" }
];
const pageSizeOptions: PageSizeOption[] = [
  { value: 10, label: "10 条" },
  { value: 20, label: "20 条" },
  { value: 50, label: "50 条" },
  { value: 100, label: "100 条" }
];

/** 用户反馈表格列配置。 */
const columns: ColumnProps[] = [
  {
    prop: "feedback_type",
    label: "反馈类型",
    isShow: false,
    isSetting: false,
    enum: feedbackTypeOptions,
    search: {
      el: "select",
      order: 1,
      props: {
        clearable: true,
        filterable: true,
        placeholder: "请选择反馈类型"
      }
    }
  },
  {
    prop: "page_size",
    label: "翻页数量",
    isShow: false,
    isSetting: false,
    enum: pageSizeOptions,
    search: {
      el: "select",
      order: 2,
      defaultValue: 10,
      props: {
        clearable: false,
        filterable: true,
        placeholder: "请选择翻页数量"
      }
    }
  },
  { prop: "feedback_type", label: "反馈类型", minWidth: 120, align: "center" },
  { prop: "goods_name", label: "商品名称", minWidth: 260, showOverflowTooltip: false },
  { prop: "categories", label: "分类", minWidth: 220, showOverflowTooltip: true },
  { prop: "value", label: "反馈值", minWidth: 100, align: "right" },
  { prop: "timestamp", label: "时间", minWidth: 180 },
  { prop: "comment", label: "备注", minWidth: 180, showOverflowTooltip: true }
];

watch(
  userId,
  value => {
    if (!value) {
      feedbackList.value = [];
      return;
    }
    offset.value = 0;
    loadFeedback().catch(() => {
      ElMessage.error("加载用户反馈失败");
    });
  },
  { immediate: true }
);

/** 加载用户反馈列表。 */
async function loadFeedback() {
  if (!userId.value) return;
  loading.value = true;
  try {
    const data = await defRecommendGorseService.GetUserFeedback({
      id: userId.value,
      feedback_type: String(proTable.value?.searchParam?.feedback_type ?? "").trim(),
      offset: offset.value,
      n: resolvePageSize()
    });
    feedbackList.value = (data.feedback ?? []).map((item, index) => ({
      ...item,
      feedbackKey: `${item.feedback_type}-${item.user_id}-${item.item?.item_id ?? index}-${item.timestamp}`
    }));
  } finally {
    loading.value = false;
  }
}

/** 读取当前翻页数量，非法值回退到 10 条。 */
function resolvePageSize() {
  const pageSize = Number(proTable.value?.searchParam?.page_size || 10);
  // 搜索条件被清空或污染时，统一回退到默认翻页数量。
  if (!Number.isFinite(pageSize) || pageSize <= 0) return 10;
  return pageSize;
}

/** 刷新用户反馈列表。 */
function handleRefresh() {
  loadFeedback().catch(() => {
    ElMessage.error("加载用户反馈失败");
  });
}

/** 响应标准搜索事件。 */
function handleSearch() {
  offset.value = 0;
  loadFeedback().catch(() => {
    ElMessage.error("加载用户反馈失败");
  });
}

/** 响应标准重置事件。 */
function handleReset() {
  offset.value = 0;
  loadFeedback().catch(() => {
    ElMessage.error("加载用户反馈失败");
  });
}

/** 跳转下一页反馈。 */
function handleNextPage() {
  offset.value += resolvePageSize();
  loadFeedback().catch(() => {
    ElMessage.error("加载用户反馈失败");
  });
}

/** 返回上一页反馈。 */
function handlePrevPage() {
  offset.value = Math.max(0, offset.value - resolvePageSize());
  loadFeedback().catch(() => {
    ElMessage.error("加载用户反馈失败");
  });
}

/** 格式化反馈类型。 */
function formatFeedbackType(type: string) {
  return feedbackTypeOptions.find(option => option.value === type)?.label || type || "--";
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

/** 格式化反馈时间。 */
function formatTimestamp(value: string) {
  if (!value || value.startsWith("0001-01-01")) return "--";
  return dayjs(value).format("YYYY-MM-DD HH:mm:ss");
}

onMounted(() => {
  recommendGorseStore.loadGoodsCategoryOptions().catch(() => {
    ElMessage.error("加载商品分类失败");
  });
});
</script>

<style scoped lang="scss">
.gorse-detail-page {
  display: flex;
  flex-direction: column;
}

.gorse-cursor-pagination {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding-top: 16px;
}
</style>
