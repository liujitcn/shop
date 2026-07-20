<template>
  <DataPanelCard title="订单状态分布" description="关注履约状态与取消结构。">
    <ECharts :option="option" />
  </DataPanelCard>
</template>

<script setup lang="ts">
import { computed, reactive, watch } from "vue";
import ECharts from "@/components/ECharts/index.vue";
import type { ECOption } from "@/components/ECharts/config";
import { defOrderAnalyticsService } from "@/api/shop/admin/order_analytics";
import DataPanelCard from "@/components/Card/DataPanelCard.vue";
import type { AnalyticsPieResponse, AnalyticsTimeType } from "@/rpc/common/v1/analytics";
import type { OptionBaseDictResponse_BaseDictItem } from "@/rpc/system/admin/v1/base_dict";
import { useDictStore } from "@/stores/modules/dict";

const props = defineProps<{
  timeType: AnalyticsTimeType;
  /** 默认租户选择的租户ID。 */
  tenantId?: number;
  /** 当前选择的门店ID。 */
  tenantStoreId?: number;
}>();

const dictStore = useDictStore();
const pieData = reactive<AnalyticsPieResponse>({
  items: []
});

const option = computed<ECOption>(() => ({
  color: ["#2d6cdf", "#15a87b", "#f08c2e", "#d9485f", "#7c3aed", "#0ea5e9", "#ef4444", "#84cc16"],
  tooltip: {
    trigger: "item",
    formatter: "{b}<br/>{c} ({d}%)"
  },
  legend: {
    bottom: 0,
    left: "center",
    textStyle: {
      color: "#6d7b8f"
    }
  },
  series: [
    {
      type: "pie",
      radius: ["34%", "74%"],
      center: ["50%", "42%"],
      roseType: "radius",
      itemStyle: {
        borderRadius: 10
      },
      label: {
        color: "#4f5d73"
      },
      data: pieData.items
    }
  ]
}));

/** 根据订单履约状态字典转换图表展示名称。 */
function resolveOrderInfoStatusName(statusValue: string, dictList: OptionBaseDictResponse_BaseDictItem[]) {
  const matchedItem = dictList.find(dictItem => dictItem.value === statusValue);
  return matchedItem?.label || `状态${statusValue}`;
}

/** 加载订单状态分布，并在前端完成状态文案转换。 */
async function loadData(timeType: AnalyticsTimeType, tenantId?: number, tenantStoreId?: number) {
  const statusDictList = await dictStore.ensureDictionary("order_info_status");
  const data = await defOrderAnalyticsService.PieOrderAnalytics({
    time_type: timeType,
    tenant_id: tenantId,
    tenant_store_id: tenantStoreId
  });
  // 兼容后端在空 repeated 字段场景下省略 items，避免空数据图表触发运行时异常。
  const items = Array.isArray(data.items) ? data.items : [];
  pieData.items = items.map(item => ({
    ...item,
    name: resolveOrderInfoStatusName(item.name, statusDictList)
  }));
}

watch(
  () => [props.timeType, props.tenantId, props.tenantStoreId] as const,
  ([timeType, tenantId, tenantStoreId]) => {
    loadData(timeType, tenantId, tenantStoreId);
  },
  { immediate: true }
);
</script>
