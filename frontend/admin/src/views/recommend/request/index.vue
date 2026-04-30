<template>
  <div class="table-box">
    <ProTable row-key="id" :columns="columns" :request-api="requestRecommendRequestTable">
      <template #provider_name="scope">
        <RecommendProviderLabel :strategy="scope.row.strategy" :provider-name="scope.row.provider_name" />
      </template>
    </ProTable>
  </div>
</template>

<script setup lang="ts">
import { View } from "@element-plus/icons-vue";
import { useRouter } from "vue-router";
import type { ColumnProps } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
import RecommendProviderLabel from "@/views/recommend/request/components/ProviderLabel.vue";
import { defRecommendRequestService } from "@/api/admin/recommend_request";
import type { PageRecommendRequestsRequest, RecommendRequest } from "@/rpc/admin/v1/recommend_request";
import { buildPageRequest } from "@/utils/proTable";
import { navigateTo } from "@/utils/router";

defineOptions({
  name: "RecommendRequest",
  inheritAttrs: false
});

const router = useRouter();

/** 推荐请求表格列配置。 */
const columns: ColumnProps[] = [
  { prop: "request_id", label: "推荐请求ID", minWidth: 180, search: { el: "input" } },
  { prop: "actor_type", label: "主体类型", minWidth: 120, dictCode: "recommend_actor_type", search: { el: "select" } },
  {
    prop: "actor_name",
    label: "主体名称",
    minWidth: 150,
    render: scope => formatRecommendActorName(scope.row as RecommendRequest)
  },
  { prop: "scene", label: "推荐场景", minWidth: 120, dictCode: "recommend_scene", search: { el: "select" } },
  {
    prop: "strategy",
    label: "命中策略",
    minWidth: 120,
    dictCode: "recommend_strategy",
    search: { el: "select" }
  },
  { prop: "provider_name", label: "最终推荐器", minWidth: 200, showOverflowTooltip: false },
  { prop: "page_num", label: "页码", minWidth: 90, align: "right" },
  { prop: "page_size", label: "分页大小", minWidth: 100, align: "right" },
  { prop: "total", label: "本次返回总数", minWidth: 130, align: "right" },
  {
    prop: "request_at",
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
        onClick: scope => handleOpenDetailPage(scope.row as RecommendRequest)
      }
    ]
  }
];

/**
 * 请求推荐请求列表，并统一补齐分页与时间范围字段。
 */
async function requestRecommendRequestTable(params: Record<string, any>) {
  const { pageNum, pageSize, ...requestParams } = buildPageRequest({
    ...params,
    request_at: params.request_at ?? []
  } as Record<string, any>);
  const data = await defRecommendRequestService.PageRecommendRequests({
    ...requestParams,
    page_num: Number(pageNum),
    page_size: Number(pageSize)
  } as PageRecommendRequestsRequest);
  const compatData = data as typeof data & {
    recommendRequests?: typeof data.recommend_requests;
    list?: typeof data.recommend_requests;
  };
  // ProTable 固定消费 list，优先使用新 snake_case 字段并兼容历史响应。
  const list = compatData.recommend_requests ?? compatData.recommendRequests ?? compatData.list ?? [];
  return { data: { ...data, list } };
}

/**
 * 统一格式化推荐主体名称文案，避免主体名称为空时直接展示空白。
 */
function formatRecommendActorName(request?: RecommendRequest) {
  if (!request) return "--";
  return request.actor_name || "--";
}

/**
 * 打开推荐请求详情页，统一改为页面跳转，避免列表页继续维护复杂弹窗状态。
 */
function handleOpenDetailPage(row: RecommendRequest) {
  if (!row.id) {
    ElMessage.warning("推荐请求记录不存在，无法查看详情");
    return;
  }

  navigateTo(router, `/recommend/request/detail/${row.id}`);
}
</script>
