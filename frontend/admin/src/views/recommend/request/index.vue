<template>
  <div class="table-box">
    <ProTable ref="proTable" row-key="id" :columns="columns" :request-api="requestRecommendRequestTable" />

    <ProDialog v-model="detailDialog.visible" :title="detailDialog.title" width="1320px" @close="handleCloseDetailDialog">
      <div v-loading="detailDialog.loading" class="recommend-request-detail">
        <div class="recommend-request-detail__summary">
          <div class="recommend-request-detail__metric">
            <span class="recommend-request-detail__metric-label">推荐请求ID</span>
            <strong class="recommend-request-detail__metric-value">{{ detailData.request?.requestId || "--" }}</strong>
          </div>
          <div class="recommend-request-detail__metric">
            <span class="recommend-request-detail__metric-label">最终推荐器</span>
            <strong class="recommend-request-detail__metric-value">{{ finalProviderName }}</strong>
          </div>
          <div class="recommend-request-detail__metric">
            <span class="recommend-request-detail__metric-label">命中策略</span>
            <strong class="recommend-request-detail__metric-value">{{ strategyLabel }}</strong>
          </div>
          <div class="recommend-request-detail__metric">
            <span class="recommend-request-detail__metric-label">当前页商品数</span>
            <strong class="recommend-request-detail__metric-value">{{ detailData.itemList.length }}</strong>
          </div>
          <div class="recommend-request-detail__metric">
            <span class="recommend-request-detail__metric-label">关联事件总数</span>
            <strong class="recommend-request-detail__metric-value">{{ totalEventCount }}</strong>
          </div>
        </div>

        <el-card shadow="never" class="recommend-request-detail__card">
          <template #header>
            <div class="recommend-request-detail__card-header">
              <span>请求基础信息</span>
            </div>
          </template>

          <el-descriptions :column="2" border class="recommend-request-detail__descriptions">
            <el-descriptions-item label="记录ID">{{ detailData.request?.id || "--" }}</el-descriptions-item>
            <el-descriptions-item label="推荐请求ID">{{ detailData.request?.requestId || "--" }}</el-descriptions-item>
            <el-descriptions-item label="主体类型">
              <DictLabel :model-value="detailData.request?.actorType" code="recommend_actor_type" />
            </el-descriptions-item>
            <el-descriptions-item label="推荐场景">
              <DictLabel :model-value="detailData.request?.scene" code="recommend_scene" />
            </el-descriptions-item>
            <el-descriptions-item label="主体信息" :span="2">{{ detailActorInfoText }}</el-descriptions-item>
            <el-descriptions-item label="请求时间">{{ detailData.request?.requestAt || "--" }}</el-descriptions-item>
            <el-descriptions-item label="页码">{{ detailData.request?.pageNum || "--" }}</el-descriptions-item>
            <el-descriptions-item label="分页大小">{{ detailData.request?.pageSize || "--" }}</el-descriptions-item>
            <el-descriptions-item label="本次返回总数">{{ detailData.request?.total || "--" }}</el-descriptions-item>
            <el-descriptions-item label="命中策略">{{ strategyLabel }}</el-descriptions-item>
          </el-descriptions>
        </el-card>

        <el-card shadow="never" class="recommend-request-detail__card">
          <template #header>
            <div class="recommend-request-detail__card-header">
              <span>推荐链路</span>
              <span class="recommend-request-detail__card-extra">按链路执行顺序展示 `GoodsTrace`</span>
            </div>
          </template>

          <div v-if="detailTraceList.length" class="trace-list">
            <div
              v-for="(item, index) in detailTraceList"
              :key="`${item.providerName}-${index}`"
              class="trace-list__item"
              :class="{
                'trace-list__item--hit': item.hit,
                'trace-list__item--final': item.isFinal,
                'trace-list__item--error': Boolean(item.errorMsg)
              }"
            >
              <div class="trace-list__header">
                <span class="trace-list__index">#{{ index + 1 }}</span>
                <span class="trace-list__name">{{ item.providerName || "未命名推荐器" }}</span>
                <el-tag v-if="item.isFinal" type="success" effect="light">最终命中</el-tag>
                <el-tag v-else-if="item.hit" type="primary" effect="light">命中</el-tag>
                <el-tag v-else type="info" effect="light">未命中</el-tag>
              </div>
              <div class="trace-list__body">
                <div class="trace-list__metric">
                  <span>返回数量</span>
                  <strong>{{ item.resultCount }}</strong>
                </div>
                <div class="trace-list__metric">
                  <span>错误信息</span>
                  <strong>{{ item.errorMsg || "--" }}</strong>
                </div>
              </div>
            </div>
          </div>

          <el-empty v-else description="当前请求没有链路轨迹数据" />
        </el-card>

        <el-card shadow="never" class="recommend-request-detail__card">
          <template #header>
            <div class="recommend-request-detail__card-header">
              <span>推荐上下文</span>
            </div>
          </template>

          <el-descriptions :column="2" border class="recommend-request-detail__descriptions">
            <el-descriptions-item label="锚点商品ID">{{ detailData.context?.goodsId || "--" }}</el-descriptions-item>
            <el-descriptions-item label="关联订单ID">{{ detailData.context?.orderId || "--" }}</el-descriptions-item>
            <el-descriptions-item label="上下文商品ID" :span="2">{{ contextGoodsText }}</el-descriptions-item>
            <el-descriptions-item label="上下文推荐器">{{ detailData.context?.providerName || "--" }}</el-descriptions-item>
            <el-descriptions-item label="最终推荐器">{{ finalProviderName }}</el-descriptions-item>
          </el-descriptions>

          <div class="recommend-request-detail__json">
            <div class="recommend-request-detail__json-title">原始上下文 JSON</div>
            <pre class="recommend-request-detail__json-block">{{ formattedContextJson }}</pre>
          </div>
        </el-card>

        <el-card shadow="never" class="recommend-request-detail__card">
          <template #header>
            <div class="recommend-request-detail__card-header">
              <span>推荐商品</span>
              <span class="recommend-request-detail__card-extra">展示当前请求页的商品，以及对应的 `recommend_event` 数量</span>
            </div>
          </template>

          <ProTable
            row-key="position"
            :data="detailData.itemList"
            :columns="itemColumns"
            :pagination="false"
            :tool-button="false"
            :border="true"
          />
        </el-card>
      </div>

      <template #footer>
        <el-button @click="handleCloseDetailDialog">关闭</el-button>
      </template>
    </ProDialog>

    <ProDialog v-model="eventDialog.visible" :title="eventDialog.title" width="1180px" @close="handleCloseEventDialog">
      <div v-loading="eventDialog.loading" class="recommend-event-detail">
        <el-descriptions :column="2" border class="recommend-event-detail__descriptions">
          <el-descriptions-item label="推荐请求ID">{{ detailData.request?.requestId || "--" }}</el-descriptions-item>
          <el-descriptions-item label="商品ID">{{ currentEventItem?.goodsId || "--" }}</el-descriptions-item>
          <el-descriptions-item label="结果位置">{{ currentEventItem?.position || "--" }}</el-descriptions-item>
          <el-descriptions-item label="事件条数">{{ eventData.total }}</el-descriptions-item>
        </el-descriptions>

        <div class="recommend-event-detail__table">
          <ProTable
            row-key="id"
            :data="eventData.list"
            :columns="eventColumns"
            :pagination="false"
            :tool-button="false"
            :border="true"
          />
        </div>
      </div>

      <template #footer>
        <el-button @click="handleCloseEventDialog">关闭</el-button>
      </template>
    </ProDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from "vue";
import { View } from "@element-plus/icons-vue";
import type { ColumnProps, ProTableInstance } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
import ProDialog from "@/components/Dialog/ProDialog.vue";
import DictLabel from "@/components/Dict/DictLabel.vue";
import { defRecommendRequestService } from "@/api/admin/recommend_request";
import type {
  GetRecommendRequestEventResponse,
  PageRecommendRequestRequest,
  RecommendRequest,
  RecommendRequestDetailResponse,
  RecommendRequestItem,
  RecommendRequestTrace
} from "@/rpc/admin/recommend_request";
import { buildPageRequest } from "@/utils/proTable";
import { formatJson, formatPrice, formatSrc } from "@/utils/utils";

defineOptions({
  name: "RecommendRequest",
  inheritAttrs: false
});

const proTable = ref<ProTableInstance>();

const detailDialog = reactive({
  title: "推荐请求详情",
  visible: false,
  loading: false
});

const eventDialog = reactive({
  title: "推荐事件数据",
  visible: false,
  loading: false
});

/** 创建默认推荐请求详情，避免切换记录时残留上一条数据。 */
function createDefaultDetailData(): RecommendRequestDetailResponse {
  return {
    request: undefined,
    context: undefined,
    itemList: []
  };
}

/** 创建默认推荐事件响应，避免切换商品时残留上一条数据。 */
function createDefaultEventData(): GetRecommendRequestEventResponse {
  return {
    list: [],
    total: 0
  };
}

const detailData = reactive<RecommendRequestDetailResponse>(createDefaultDetailData());
const eventData = reactive<GetRecommendRequestEventResponse>(createDefaultEventData());
const currentEventItem = ref<RecommendRequestItem>();

/** 当前请求链路列表，统一回退为空数组。 */
const detailTraceList = computed<RecommendRequestTrace[]>(() => detailData.context?.trace ?? []);

/** 当前请求展示的最终推荐器。 */
const finalProviderName = computed(() => detailData.context?.finalProviderName || detailData.request?.providerName || "--");

/** 当前请求展示的推荐策略。 */
const strategyLabel = computed(() => formatStrategyLabel(detailData.context?.strategy || detailData.request?.strategy || ""));

/** 当前请求主体信息展示文本。 */
const detailActorInfoText = computed(() => formatRecommendActorInfo(detailData.request));

/** 当前请求的上下文商品ID展示文本。 */
const contextGoodsText = computed(() => {
  const contextGoodsIds = detailData.context?.contextGoodsIds ?? [];
  if (!contextGoodsIds.length) return "--";
  return contextGoodsIds.join(", ");
});

/** 当前请求上下文 JSON 的格式化内容。 */
const formattedContextJson = computed(() => formatJson(detailData.context?.rawJson || "{}"));

/** 当前请求商品关联事件总数。 */
const totalEventCount = computed(() => detailData.itemList.reduce((sum, item) => sum + Number(item.eventCount || 0), 0));

/** 推荐请求表格列配置。 */
const columns: ColumnProps[] = [
  { prop: "requestId", label: "推荐请求ID", minWidth: 180, search: { el: "input" } },
  { prop: "actorType", label: "主体类型", minWidth: 120, dictCode: "recommend_actor_type", search: { el: "select" } },
  {
    prop: "actorInfo",
    label: "主体信息",
    minWidth: 150,
    render: scope => formatRecommendActorInfo(scope.row as RecommendRequest)
  },
  { prop: "scene", label: "推荐场景", minWidth: 120, dictCode: "recommend_scene", search: { el: "select" } },
  {
    prop: "strategy",
    label: "命中策略",
    minWidth: 120,
    render: scope => formatStrategyLabel((scope.row as RecommendRequest).strategy)
  },
  { prop: "providerName", label: "最终推荐器", minWidth: 180 },
  { prop: "pageNum", label: "页码", minWidth: 90, align: "right" },
  { prop: "pageSize", label: "分页大小", minWidth: 100, align: "right" },
  { prop: "total", label: "本次返回总数", minWidth: 130, align: "right" },
  {
    prop: "requestAt",
    label: "请求时间",
    minWidth: 180,
    search: {
      el: "date-picker",
      props: {
        type: "daterange",
        editable: false,
        class: "!w-[240px]",
        rangeSeparator: "~",
        startPlaceholder: "开始时间",
        endPlaceholder: "截止时间",
        valueFormat: "YYYY-MM-DD"
      }
    }
  },
  {
    prop: "detailAction",
    label: "操作",
    width: 110,
    fixed: "right",
    cellType: "actions",
    actions: [
      {
        label: "详情",
        type: "primary",
        link: true,
        icon: View,
        onClick: scope => handleOpenDetailDialog((scope.row as RecommendRequest).id)
      }
    ]
  }
];

/** 推荐商品表格列配置。 */
const itemColumns: ColumnProps[] = [
  { prop: "goodsId", label: "商品ID", minWidth: 120 },
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
  { prop: "goodsName", label: "商品名称", minWidth: 220 },
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
  { prop: "eventCount", label: "事件条数", minWidth: 110, align: "right" },
  {
    prop: "eventAction",
    label: "操作",
    width: 120,
    fixed: "right",
    cellType: "actions",
    actions: [
      {
        label: "查看事件",
        type: "primary",
        link: true,
        icon: View,
        disabled: scope => Number((scope.row as RecommendRequestItem).eventCount || 0) <= 0,
        onClick: scope => handleOpenEventDialog(scope.row as RecommendRequestItem)
      }
    ]
  }
];

/** 推荐事件表格列配置。 */
const eventColumns: ColumnProps[] = [
  { prop: "id", label: "事件ID", minWidth: 120 },
  { prop: "actorType", label: "主体类型", minWidth: 120, dictCode: "recommend_actor_type" },
  { prop: "scene", label: "推荐场景", minWidth: 120, dictCode: "recommend_scene" },
  { prop: "eventType", label: "事件类型", minWidth: 120, dictCode: "recommend_event_type" },
  { prop: "goodsNum", label: "商品数量", minWidth: 100, align: "right" },
  { prop: "position", label: "结果位置", minWidth: 100, align: "right" },
  { prop: "eventAt", label: "事件时间", minWidth: 180 }
];

/**
 * 请求推荐请求列表，并统一补齐分页与时间范围字段。
 */
async function requestRecommendRequestTable(params: PageRecommendRequestRequest) {
  const data = await defRecommendRequestService.PageRecommendRequest(
    buildPageRequest({
      ...params,
      requestAt: params.requestAt ?? []
    })
  );
  return { data };
}

/**
 * 打开推荐请求详情弹窗并加载完整链路数据。
 */
async function handleOpenDetailDialog(requestRecordId?: number) {
  resetDetailData();
  detailDialog.title = "推荐请求详情";
  detailDialog.visible = true;
  if (!requestRecordId) return;

  detailDialog.loading = true;
  try {
    const data = await defRecommendRequestService.GetRecommendRequest({ value: requestRecordId });
    Object.assign(detailData, data);
    detailDialog.title = `推荐请求详情 ${data.request?.requestId ? `- ${data.request.requestId}` : ""}`;
  } finally {
    detailDialog.loading = false;
  }
}

/**
 * 关闭推荐请求详情弹窗，并重置详情与事件状态。
 */
function handleCloseDetailDialog() {
  detailDialog.visible = false;
  resetDetailData();
  handleCloseEventDialog();
}

/**
 * 重置推荐请求详情，避免切换记录时残留上一条数据。
 */
function resetDetailData() {
  Object.assign(detailData, createDefaultDetailData());
}

/**
 * 打开推荐商品关联事件弹窗，并加载对应的 recommend_event 数据。
 */
async function handleOpenEventDialog(item: RecommendRequestItem) {
  const requestRecordId = detailData.request?.id || 0;
  if (!requestRecordId || !item.goodsId) {
    ElMessage.warning("推荐请求记录不完整，无法查看事件");
    return;
  }

  resetEventData();
  currentEventItem.value = { ...item };
  eventDialog.title = `推荐事件数据 - 商品 ${item.goodsName || item.goodsId}`;
  eventDialog.visible = true;
  eventDialog.loading = true;
  try {
    const data = await defRecommendRequestService.GetRecommendRequestEvent({
      requestRecordId,
      goodsId: item.goodsId,
      position: item.position
    });
    Object.assign(eventData, data);
  } finally {
    eventDialog.loading = false;
  }
}

/**
 * 关闭推荐事件弹窗，并清理当前商品上下文。
 */
function handleCloseEventDialog() {
  eventDialog.visible = false;
  resetEventData();
  currentEventItem.value = undefined;
}

/**
 * 重置推荐事件数据，避免切换商品时残留上一条数据。
 */
function resetEventData() {
  Object.assign(eventData, createDefaultEventData());
}

/**
 * 统一格式化推荐策略文案，避免直接暴露内部策略标识。
 */
function formatStrategyLabel(strategy: string) {
  if (strategy === "remote") return "远程推荐";
  if (strategy === "local") return "本地推荐";
  return strategy || "--";
}

/**
 * 统一格式化推荐主体信息文案。
 */
function formatRecommendActorInfo(request?: RecommendRequest) {
  if (!request) return "--";
  return request.actorInfo || "--";
}
</script>

<style scoped lang="scss">
.recommend-request-detail,
.recommend-event-detail {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.recommend-request-detail__summary {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 12px;
}

.recommend-request-detail__metric {
  padding: 16px;
  background: var(--admin-page-card-bg-soft);
  border: 1px solid var(--admin-page-card-border-soft);
  border-radius: var(--admin-page-radius);
}

.recommend-request-detail__metric-label {
  display: block;
  margin-bottom: 8px;
  font-size: 13px;
  color: var(--admin-page-text-secondary);
}

.recommend-request-detail__metric-value {
  font-size: 18px;
  line-height: 1.4;
  color: var(--admin-page-text-primary);
  word-break: break-all;
}

.recommend-request-detail__card {
  border-color: var(--admin-page-card-border-soft);
}

.recommend-request-detail__card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  color: var(--admin-page-text-primary);
}

.recommend-request-detail__card-extra {
  font-size: 12px;
  color: var(--admin-page-text-secondary);
}

.recommend-request-detail__descriptions,
.recommend-event-detail__descriptions {
  :deep(.el-descriptions__label) {
    width: 140px;
  }
}

.trace-list {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  gap: 12px;
}

.trace-list__item {
  padding: 16px;
  background: var(--admin-page-card-bg-soft);
  border: 1px solid var(--admin-page-card-border-soft);
  border-radius: var(--admin-page-radius);
}

.trace-list__item--hit {
  border-color: var(--el-color-primary-light-5);
}

.trace-list__item--final {
  background: var(--admin-page-accent-soft-bg);
  border-color: var(--admin-page-accent-soft-border);
}

.trace-list__item--error {
  box-shadow: inset 0 0 0 1px rgb(245 108 108 / 18%);
}

.trace-list__header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
}

.trace-list__index {
  color: var(--admin-page-text-secondary);
  font-size: 13px;
}

.trace-list__name {
  flex: 1;
  color: var(--admin-page-text-primary);
  font-weight: 600;
  word-break: break-all;
}

.trace-list__body {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.trace-list__metric {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  color: var(--admin-page-text-secondary);
}

.trace-list__metric strong {
  color: var(--admin-page-text-primary);
  word-break: break-all;
}

.recommend-request-detail__json {
  margin-top: 16px;
}

.recommend-request-detail__json-title {
  margin-bottom: 8px;
  color: var(--admin-page-text-secondary);
  font-size: 13px;
}

.recommend-request-detail__json-block {
  margin: 0;
  padding: 14px 16px;
  max-height: 260px;
  overflow: auto;
  background: var(--admin-page-card-bg-soft);
  border: 1px solid var(--admin-page-card-border-soft);
  border-radius: var(--admin-page-radius);
  color: var(--admin-page-text-primary);
  white-space: pre-wrap;
  word-break: break-all;
}

.recommend-event-detail__table {
  margin-top: 16px;
}
</style>
