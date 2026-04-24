<template>
  <div class="table-box">
    <ProTable row-key="id" :columns="columns" :request-api="requestRecommendRequestTable">
      <template #providerName="scope">
        <RecommendProviderLabel :strategy="scope.row.strategy" :provider-name="scope.row.providerName" />
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
import type { PageRecommendRequestRequest, RecommendRequest } from "@/rpc/admin/recommend_request";
import { buildPageRequest } from "@/utils/proTable";
import { navigateTo } from "@/utils/router";

defineOptions({
  name: "RecommendRequest",
  inheritAttrs: false
});

const router = useRouter();

/** 推荐请求表格列配置。 */
const columns: ColumnProps[] = [
  { prop: "requestId", label: "推荐请求ID", minWidth: 180, search: { el: "input" } },
  { prop: "actorType", label: "主体类型", minWidth: 120, dictCode: "recommend_actor_type", search: { el: "select" } },
  {
    prop: "actorName",
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
  { prop: "providerName", label: "最终推荐器", minWidth: 200, showOverflowTooltip: false },
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
        onClick: scope => handleOpenDetailPage(scope.row as RecommendRequest)
      }
    ]
  }
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
 * 统一格式化推荐主体名称文案，避免主体名称为空时直接展示空白。
 */
function formatRecommendActorName(request?: RecommendRequest) {
  if (!request) return "--";
  return request.actorName || "--";
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
