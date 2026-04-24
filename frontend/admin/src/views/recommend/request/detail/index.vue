<template>
  <div v-loading="loading" class="app-container recommend-request-detail-page">
    <el-card v-if="detailData.request" class="detail-hero-card" shadow="never">
      <div class="detail-summary-grid">
        <div class="detail-summary-card">
          <span class="detail-summary-card__label">最终推荐器</span>
          <strong class="detail-summary-card__value">{{ finalProviderName }}</strong>
        </div>
        <div class="detail-summary-card">
          <span class="detail-summary-card__label">命中策略</span>
          <div class="detail-summary-card__value detail-summary-card__value--dict">
            <DictLabel :model-value="strategyValue" code="recommend_strategy" />
          </div>
        </div>
        <button type="button" class="detail-summary-card detail-summary-card--action" @click="handleSwitchTab('context')">
          <span class="detail-summary-card__label">推荐链路</span>
          <strong class="detail-summary-card__value">{{ detailTraceList.length }}</strong>
        </button>
        <button type="button" class="detail-summary-card detail-summary-card--action" @click="handleSwitchTab('item')">
          <span class="detail-summary-card__label">推荐商品</span>
          <strong class="detail-summary-card__value">{{ detailData.itemList.length }}</strong>
        </button>
        <div class="detail-summary-card">
          <span class="detail-summary-card__label">关联事件</span>
          <strong class="detail-summary-card__value">{{ totalEventCount }}</strong>
        </div>
      </div>
    </el-card>

    <el-card class="detail-tabs-card" shadow="never">
      <el-tabs v-if="detailData.request" v-model="activeTabName" class="detail-tabs">
        <el-tab-pane label="请求信息" name="request">
          <div class="detail-tab-panel detail-tab-panel--padding">
            <div class="detail-section-panel">
              <div class="detail-section-panel__header">
                <span>请求基础信息</span>
              </div>

              <el-descriptions :column="2" border class="detail-descriptions">
                <el-descriptions-item label="主体类型">
                  <DictLabel :model-value="detailData.request?.actorType" code="recommend_actor_type" />
                </el-descriptions-item>
                <el-descriptions-item label="推荐场景">
                  <DictLabel :model-value="detailData.request?.scene" code="recommend_scene" />
                </el-descriptions-item>
                <el-descriptions-item label="主体名称" :span="2">{{ detailActorInfoText }}</el-descriptions-item>
                <el-descriptions-item label="请求时间">{{ detailData.request?.requestAt || "--" }}</el-descriptions-item>
                <el-descriptions-item label="页码">{{ detailData.request?.pageNum ?? "--" }}</el-descriptions-item>
                <el-descriptions-item label="分页大小">{{ detailData.request?.pageSize ?? "--" }}</el-descriptions-item>
                <el-descriptions-item label="本次返回总数">{{ detailData.request?.total ?? "--" }}</el-descriptions-item>
              </el-descriptions>
            </div>
          </div>
        </el-tab-pane>

        <el-tab-pane label="链路与上下文" name="context">
          <div class="detail-tab-panel detail-tab-panel--padding">
            <div class="detail-grid">
              <div class="detail-section-panel">
                <div class="detail-section-panel__header">
                  <span>推荐链路</span>
                </div>

                <div v-if="detailTraceList.length" class="trace-flow">
                  <template v-for="(item, index) in detailTraceList" :key="`${item.providerName}-${index}`">
                    <div
                      class="trace-flow__item"
                      :class="{
                        'trace-flow__item--hit': item.hit,
                        'trace-flow__item--final': item.isFinal,
                        'trace-flow__item--error': Boolean(item.errorMsg),
                        'trace-flow__item--turn': (index + 1) % 4 === 0 && index < detailTraceList.length - 1,
                        'trace-flow__item--last': index === detailTraceList.length - 1
                      }"
                    >
                      <div class="trace-flow__top">
                        <span class="trace-flow__index">步骤 {{ index + 1 }}</span>
                        <div class="trace-flow__tag">
                          <el-tag v-if="item.isFinal" type="success" effect="light">最终命中</el-tag>
                          <el-tag v-else-if="item.hit" type="primary" effect="light">命中</el-tag>
                          <el-tag v-else type="info" effect="light">未命中</el-tag>
                        </div>
                      </div>
                      <el-tooltip :content="item.providerName || '未命名推荐器'" effect="light" placement="top">
                        <div class="trace-flow__title">
                          <span class="trace-flow__title-label">推荐器</span>
                          <span class="trace-flow__name">{{ item.providerName || "未命名推荐器" }}</span>
                        </div>
                      </el-tooltip>
                      <div class="trace-flow__body">
                        <div class="trace-flow__metric">
                          <span>返回数量</span>
                          <strong>{{ item.resultCount }}</strong>
                        </div>
                        <div class="trace-flow__metric">
                          <span>错误信息</span>
                          <strong>{{ item.errorMsg || "--" }}</strong>
                        </div>
                      </div>
                    </div>
                  </template>
                </div>

                <el-empty v-else description="当前请求没有链路轨迹数据" />
              </div>

              <div class="detail-section-panel">
                <div class="detail-section-panel__header">
                  <span>推荐上下文</span>
                </div>

                <el-descriptions :column="1" border class="detail-descriptions">
                  <el-descriptions-item label="上下文推荐器">{{ detailData.context?.providerName || "--" }}</el-descriptions-item>
                </el-descriptions>

                <div class="detail-json">
                  <div class="detail-json__title">原始上下文 JSON</div>
                  <pre class="detail-json__block">{{ formattedContextJson }}</pre>
                </div>
              </div>
            </div>
          </div>
        </el-tab-pane>

        <el-tab-pane label="推荐商品" name="item">
          <div class="detail-tab-panel detail-tab-panel--padding">
            <div class="detail-section-panel detail-section-panel--table">
              <div class="detail-section-panel__header">
                <span>推荐商品</span>
              </div>

              <ProTable
                row-key="position"
                :data="detailData.itemList"
                :columns="itemColumns"
                :pagination="false"
                :tool-button="false"
                :border="true"
              >
                <template #goodsName="scope">
                  <el-link
                    v-if="scope.row.goodsId"
                    type="primary"
                    :underline="false"
                    @click.stop="handleOpenGoodsDetail(scope.row)"
                  >
                    {{ scope.row.goodsName || "--" }}
                  </el-link>
                  <span v-else>{{ scope.row.goodsName || "--" }}</span>
                </template>

                <template #eventCount="scope">
                  <div class="detail-event-count-cell">
                    <el-link
                      v-if="Number(scope.row.eventCount || 0) > 0"
                      type="primary"
                      :underline="false"
                      @click.stop="handleOpenEventDialog(scope.row)"
                    >
                      {{ scope.row.eventCount ?? 0 }}
                    </el-link>
                    <span v-else>{{ scope.row.eventCount ?? 0 }}</span>
                  </div>
                </template>
              </ProTable>
            </div>
          </div>
        </el-tab-pane>
      </el-tabs>

      <div v-else class="detail-empty-panel">
        <el-empty description="暂无推荐请求详情数据" />
      </div>
    </el-card>

    <RecommendRequestEventDialog ref="eventDialogRef" />
  </div>
</template>

<script setup lang="ts">
import { computed, onActivated, reactive, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import type { ColumnProps } from "@/components/ProTable/interface";
import DictLabel from "@/components/Dict/DictLabel.vue";
import ProTable from "@/components/ProTable/index.vue";
import RecommendRequestEventDialog from "@/views/recommend/request/detail/components/EventDialog.vue";
import { defRecommendRequestService } from "@/api/admin/recommend_request";
import type { RecommendRequestDetailResponse, RecommendRequestItem, RecommendRequestTrace } from "@/rpc/admin/recommend_request";
import { useTabsStore } from "@/stores/modules/tabs";
import { navigateTo } from "@/utils/router";
import { formatJson, formatPrice, formatSrc } from "@/utils/utils";

defineOptions({
  name: "RecommendRequestDetail",
  inheritAttrs: false
});

/** 推荐请求详情页支持的标签页名称。 */
type RecommendRequestDetailTabName = "request" | "context" | "item";

const route = useRoute();
const router = useRouter();
const tabsStore = useTabsStore();
const eventDialogRef = ref<InstanceType<typeof RecommendRequestEventDialog>>();
const loading = ref(false);
const requestRecordId = ref(0);
const detailRequestId = ref(0);
const activeTabName = ref<RecommendRequestDetailTabName>("request");

/** 推荐请求详情工作区标题固定为“推荐请求详情”，避免依赖列表页查询参数透传。 */
const workspaceTitle = "推荐请求详情";

/** 判断当前是否仍停留在推荐请求详情页，避免离开后继续改写其他页面标题。 */
function isCurrentRecommendRequestDetailRoute() {
  return route.name === "RecommendRequestDetail" || route.path.includes("/request/detail/");
}

/** 创建默认推荐请求详情，避免切换记录时残留上一条数据。 */
function createDefaultDetailData(): RecommendRequestDetailResponse {
  return {
    request: undefined,
    context: undefined,
    itemList: []
  };
}

const detailData = reactive<RecommendRequestDetailResponse>(createDefaultDetailData());

/** 当前请求链路列表，统一回退为空数组。 */
const detailTraceList = computed<RecommendRequestTrace[]>(() => detailData.context?.trace ?? []);

/** 当前请求展示的最终推荐器。 */
const finalProviderName = computed(() => detailData.context?.finalProviderName || detailData.request?.providerName || "--");

/** 当前请求展示的推荐策略，统一优先使用上下文中的最终策略。 */
const strategyValue = computed(() => detailData.context?.strategy ?? detailData.request?.strategy);

/** 当前请求主体信息展示文本。 */
const detailActorInfoText = computed(() => formatRecommendActorName(detailData.request));

/** 当前请求上下文 JSON 的格式化内容。 */
const formattedContextJson = computed(() => formatJson(detailData.context?.rawJson || "{}"));

/** 当前请求商品关联事件总数。 */
const totalEventCount = computed(() => detailData.itemList.reduce((sum, item) => sum + Number(item.eventCount || 0), 0));

/** 推荐商品表格列配置。 */
const itemColumns: ColumnProps[] = [
  {
    prop: "picture",
    label: "商品图片",
    minWidth: 100,
    cellType: "image",
    imageProps: {
      src: scope => formatSrc(String((scope.row as RecommendRequestItem).picture || "")),
      previewSrc: scope => formatSrc(String((scope.row as RecommendRequestItem).picture || "")),
      width: 56,
      height: 56
    }
  },
  { prop: "goodsName", label: "商品名称", minWidth: 220, showOverflowTooltip: false },
  { prop: "position", label: "结果位置", minWidth: 100, align: "right" },
  {
    prop: "price",
    label: "当前价格（元）",
    minWidth: 130,
    align: "right",
    render: scope => formatPrice((scope.row as RecommendRequestItem).price)
  },
  {
    prop: "discountPrice",
    label: "折扣价格（元）",
    minWidth: 130,
    align: "right",
    render: scope => formatPrice((scope.row as RecommendRequestItem).discountPrice)
  },
  { prop: "goodsStatus", label: "商品状态", minWidth: 120, dictCode: "goods_status" },
  { prop: "eventCount", label: "事件条数", minWidth: 160, showOverflowTooltip: false }
];

/**
 * 统一格式化推荐主体名称文案，避免主体名称为空时直接展示空白。
 */
function formatRecommendActorName(request?: RecommendRequestDetailResponse["request"]) {
  if (!request) return "--";
  return request.actorName || "--";
}

// 监听路由参数变化，更新推荐请求详情数据。
watch(
  () => route.params.requestRecordId,
  () => {
    if (!isCurrentRecommendRequestDetailRoute()) return;
    const currentRequestRecordId = syncRequestRecordIdFromRoute();
    syncWorkspaceTitle();
    if (!currentRequestRecordId) {
      resetDetailData();
      return;
    }
    handleQuery(currentRequestRecordId);
  },
  { immediate: true }
);

/** 重置推荐请求详情，避免切换记录时残留上一条数据。 */
function resetDetailData() {
  Object.assign(detailData, createDefaultDetailData());
}

/** 从路由中同步当前推荐请求记录ID。 */
function syncRequestRecordIdFromRoute() {
  requestRecordId.value = Number(route.params.requestRecordId ?? 0);
  return requestRecordId.value;
}

/** 同步当前页签和浏览器标题，确保列表点击进入详情时无需刷新即可生效。 */
function syncWorkspaceTitle() {
  tabsStore.setTabsTitle(workspaceTitle);
  document.title = `${workspaceTitle} - ${import.meta.env.VITE_GLOB_APP_TITLE}`;
}

/** 查询推荐请求详情，并刷新当前页面展示数据。 */
function handleQuery(targetRequestRecordId: number = requestRecordId.value) {
  if (!targetRequestRecordId) return;
  const requestId = ++detailRequestId.value;
  loading.value = true;
  defRecommendRequestService
    .GetRecommendRequest({ value: targetRequestRecordId })
    .then(data => {
      if (requestId !== detailRequestId.value) return;
      resetDetailData();
      Object.assign(detailData, data);
      activeTabName.value = "request";
    })
    .catch(() => {
      if (requestId !== detailRequestId.value) return;
      resetDetailData();
    })
    .finally(() => {
      if (requestId !== detailRequestId.value) return;
      loading.value = false;
    });
}

/** 切换到指定标签页，统一复用顶部摘要快捷入口。 */
function handleSwitchTab(tabName: RecommendRequestDetailTabName) {
  activeTabName.value = tabName;
}

/**
 * 打开商品详情页，便于从推荐结果快速回看商品信息。
 */
function handleOpenGoodsDetail(item: RecommendRequestItem) {
  if (!item.goodsId) {
    ElMessage.warning("当前推荐商品缺少商品ID");
    return;
  }

  void navigateTo(router, `/goods/detail/${item.goodsId}`);
}

/**
 * 打开关联事件弹窗，并默认展示当前推荐商品的事件数据。
 */
function handleOpenEventDialog(item: RecommendRequestItem) {
  if (Number(item.eventCount || 0) <= 0) {
    ElMessage.warning("当前商品没有关联事件");
    return;
  }

  const targetRequestRecordId = detailData.request?.id || requestRecordId.value;
  if (!targetRequestRecordId) {
    ElMessage.warning("当前推荐请求记录不存在");
    return;
  }

  eventDialogRef.value?.openDialog({
    requestRecordId: targetRequestRecordId,
    item
  });
}

onActivated(() => {
  if (!isCurrentRecommendRequestDetailRoute()) return;
  syncWorkspaceTitle();
  const currentRequestRecordId = syncRequestRecordIdFromRoute();
  if (!currentRequestRecordId || loading.value) return;
  handleQuery(currentRequestRecordId);
});
</script>

<style scoped lang="scss">
.detail-hero-card,
.detail-tabs-card {
  border: 1px solid var(--admin-page-card-border);
  border-radius: var(--admin-page-radius);
  background: var(--admin-page-card-bg);
  box-shadow: var(--admin-page-shadow);
}

.detail-hero-card {
  margin-bottom: 18px;
}

:deep(.detail-hero-card .el-card__body) {
  padding: 16px;
}

:deep(.detail-tabs-card .el-card__body) {
  padding: 0;
}

.detail-summary-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 16px;
}

.detail-summary-card {
  display: flex;
  box-sizing: border-box;
  min-height: 84px;
  min-width: 0;
  flex-direction: column;
  justify-content: center;
  gap: 6px;
  padding: 16px;
  border: 1px solid var(--admin-page-card-border-soft);
  border-radius: var(--admin-page-radius);
  background: var(--admin-page-card-bg-soft);
}

.detail-summary-card__label {
  font-size: 12px;
  line-height: 1.4;
  color: var(--admin-page-text-secondary);
}

.detail-summary-card__value {
  font-size: 18px;
  font-weight: 700;
  line-height: 1.35;
  color: var(--admin-page-text-primary);
  word-break: break-all;
}

.detail-summary-card__value--dict {
  display: inline-flex;
  align-items: center;
}

.detail-summary-card--action {
  appearance: none;
  font: inherit;
  text-align: left;
  cursor: pointer;
  transition: border-color 0.2s ease;
}

.detail-summary-card--action:hover {
  border-color: var(--admin-page-card-border-muted);
}

.detail-tabs :deep(.el-tabs__header) {
  margin: 0;
  padding: 0 16px;
}

.detail-tabs :deep(.el-tabs__nav-wrap::after) {
  height: 1px;
  background-color: var(--admin-page-divider-strong);
}

.detail-tabs :deep(.el-tabs__item) {
  height: 36px;
  padding: 0 8px;
  font-size: 14px;
  font-weight: 600;
}

.detail-tabs :deep(.el-tabs__content) {
  padding: 0;
}

.detail-tab-panel {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.detail-tab-panel--padding {
  padding: 16px;
}

.detail-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.85fr) minmax(320px, 0.95fr);
  gap: 16px;
  align-items: start;
}

.detail-section-panel {
  border: 1px solid var(--admin-page-card-border-soft);
  border-radius: var(--admin-page-radius);
  background: var(--admin-page-card-bg-soft);
  padding: 16px;
}

.detail-section-panel--table {
  background: var(--admin-page-card-bg);
}

.detail-section-panel__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 16px;
  color: var(--admin-page-text-primary);
}

.detail-section-panel__header span:first-child {
  font-size: 16px;
  font-weight: 600;
}

.detail-section-panel__extra {
  font-size: 12px;
  color: var(--admin-page-text-secondary);
}

.detail-descriptions {
  :deep(.el-descriptions__label) {
    width: 140px;
  }
}

.detail-json {
  margin-top: 16px;
}

.detail-json__title {
  margin-bottom: 8px;
  color: var(--admin-page-text-secondary);
  font-size: 13px;
}

.detail-json__block {
  margin: 0;
  padding: 14px 16px;
  max-height: 260px;
  overflow: auto;
  background: var(--admin-page-card-bg);
  border: 1px solid var(--admin-page-card-border-soft);
  border-radius: var(--admin-page-radius);
  color: var(--admin-page-text-primary);
  white-space: pre-wrap;
  word-break: break-all;
}

.trace-flow {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 40px 32px;
  align-items: stretch;
}

.trace-flow__item {
  position: relative;
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 12px;
  padding: 16px;
  background: var(--admin-page-card-bg);
  border: 1px solid var(--admin-page-card-border-soft);
  border-radius: var(--admin-page-radius);
  box-sizing: border-box;
}

.trace-flow__item--hit {
  border-color: var(--el-color-primary-light-5);
}

.trace-flow__item--final {
  background: var(--admin-page-accent-soft-bg);
  border-color: var(--admin-page-accent-soft-border);
}

.trace-flow__item--error {
  box-shadow: inset 0 0 0 1px rgb(245 108 108 / 18%);
}

.trace-flow__item::after {
  content: "→";
  position: absolute;
  top: 50%;
  right: -26px;
  transform: translateY(-50%);
  color: var(--el-color-primary);
  font-size: 24px;
  font-weight: 700;
  line-height: 1;
  pointer-events: none;
}

.trace-flow__item--turn::after {
  content: "↓";
  top: auto;
  right: auto;
  bottom: -30px;
  left: 50%;
  transform: translateX(-50%);
}

.trace-flow__item--last::after {
  display: none;
}

.trace-flow__top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.trace-flow__index {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 60px;
  height: 24px;
  padding: 0 8px;
  border-radius: 999px;
  background: var(--admin-page-card-bg-soft);
  color: var(--admin-page-text-secondary);
  font-size: 12px;
  font-weight: 600;
  white-space: nowrap;
}

.trace-flow__tag {
  display: flex;
  align-items: center;
  flex-shrink: 0;
}

.trace-flow__title {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 4px;
  padding: 12px;
  border-radius: 12px;
  background: var(--admin-page-card-bg-soft);
}

.trace-flow__title-label {
  color: var(--admin-page-text-secondary);
  font-size: 12px;
  line-height: 1;
}

.trace-flow__name {
  color: var(--admin-page-text-primary);
  font-size: 15px;
  font-weight: 700;
  line-height: 1.4;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.trace-flow__body {
  display: flex;
  flex-direction: column;
  gap: 10px;
  flex: 1;
}

.trace-flow__metric {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  color: var(--admin-page-text-secondary);
}

.trace-flow__metric strong {
  color: var(--admin-page-text-primary);
  word-break: break-all;
}

.detail-event-count-cell {
  display: inline-flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
  width: 100%;
}

.detail-empty-panel {
  padding: 32px 16px;
}

@media screen and (max-width: 992px) {
  .detail-grid {
    grid-template-columns: minmax(0, 1fr);
  }

  .trace-flow {
    grid-template-columns: minmax(0, 1fr);
  }

  .trace-flow__item::after {
    content: "↓";
    top: auto;
    right: auto;
    bottom: -30px;
    left: 50%;
    transform: translateX(-50%);
  }

  .trace-flow__item--last::after {
    display: none;
  }
}
</style>
