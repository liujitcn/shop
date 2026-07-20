<template>
  <PageLayout
    title="商品分析"
    description="按时间维度查看商品供给、行为转化与分类成交结构的汇总和趋势变化。"
    :period-label="activeTimeLabel"
    content-ratio="minmax(0, 1fr) minmax(0, 1fr)"
  >
    <template #toolbar>
      <div class="analytics-toolbar">
        <el-tree-select
          v-if="isDefaultTenant"
          v-model="tenantStoreTreeValue"
          :data="tenantStoreTreeOptions"
          clearable
          filterable
          check-strictly
          :render-after-expand="false"
          placeholder="全部租户/门店"
          class="analytics-toolbar__scope"
        />
        <el-select v-else v-model="tenantStoreId" clearable filterable placeholder="全部门店" class="analytics-toolbar__scope">
          <el-option v-for="item in tenantStoreOptions" :key="item.value" :label="item.label" :value="item.value" />
        </el-select>
        <TimeTabs v-model="activeTime" />
      </div>
    </template>

    <template #metrics>
      <SummaryCards
        :time-type="activeTime"
        :tenant-id="tenantStoreScope.tenant_id"
        :tenant-store-id="tenantStoreScope.tenant_store_id"
      />
    </template>

    <TrendChart
      class="goods-analytics__trend"
      :time-type="activeTime"
      :tenant-id="tenantStoreScope.tenant_id"
      :tenant-store-id="tenantStoreScope.tenant_store_id"
    />
    <SidePanels
      class="goods-analytics__panels"
      :time-type="activeTime"
      :tenant-id="tenantStoreScope.tenant_id"
      :tenant-store-id="tenantStoreScope.tenant_store_id"
    />
  </PageLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useRoute } from "vue-router";
import type { EnumProps } from "@/components/ProTable/interface";
import { defTenantStoreService } from "@/api/shop/admin/tenant_store";
import { AnalyticsTimeType } from "@/rpc/common/v1/analytics";
import type { OptionTenantStoreResponse_Option } from "@/rpc/shop/admin/v1/tenant_store";
import { useUserStore } from "@/stores/modules/user";
import { DEFAULT_TENANT_CODE, parseTenantStoreTreeValue, transformTenantStoreTreeOptions } from "@/views/shop/admin/utils/tenant";
import PageLayout from "../components/PageLayout.vue";
import TimeTabs from "../components/TimeTabs.vue";
import SidePanels from "./components/SidePanels.vue";
import SummaryCards from "./components/SummaryCards.vue";
import TrendChart from "./components/TrendChart.vue";

defineOptions({
  name: "GoodsAnalytics"
});

const route = useRoute();
const userStore = useUserStore();

// 默认以周维度展示，枚举值与 proto v1 生成成员保持一致。
const activeTime = ref<AnalyticsTimeType>(AnalyticsTimeType.ANALYTICS_TIME_TYPE_WEEK);
const routeTenantID = Number(route.query.tenantId ?? 0);
const routeTenantStoreID = Number(route.query.tenantStoreId ?? 0);
const tenantStoreTreeValue = ref(
  routeTenantStoreID > 0 ? `store:${routeTenantStoreID}` : routeTenantID > 0 ? `tenant:${routeTenantID}` : undefined
);
const tenantStoreId = ref<number | undefined>(routeTenantStoreID > 0 ? routeTenantStoreID : undefined);
const tenantStoreTreeOptions = ref<EnumProps[]>([]);
const tenantStoreOptions = ref<OptionTenantStoreResponse_Option[]>([]);

/** 当前登录账号是否默认租户。 */
const isDefaultTenant = computed(() => userStore.userInfo.tenant_code === DEFAULT_TENANT_CODE);

/** 当前商品分析的租户与门店查询范围。 */
const tenantStoreScope = computed(() => {
  if (isDefaultTenant.value) {
    return parseTenantStoreTreeValue(tenantStoreTreeValue.value);
  }
  return { tenant_store_id: tenantStoreId.value };
});

const activeTimeLabel = computed(() => {
  switch (activeTime.value) {
    case AnalyticsTimeType.ANALYTICS_TIME_TYPE_MONTH:
      return "本月";
    case AnalyticsTimeType.ANALYTICS_TIME_TYPE_YEAR:
      return "本年";
    default:
      return "本周";
  }
});

/** 加载当前账号可选择的租户门店范围。 */
async function loadTenantStoreOptions() {
  if (isDefaultTenant.value) {
    const response = await defTenantStoreService.TreeTenantStore({ keyword: "" });
    tenantStoreTreeOptions.value = transformTenantStoreTreeOptions(response.list ?? []);
    return;
  }
  const response = await defTenantStoreService.OptionTenantStore({ keyword: "" });
  tenantStoreOptions.value = response.list ?? [];
}

onMounted(() => {
  void loadTenantStoreOptions();
});
</script>

<style scoped lang="scss">
.analytics-toolbar {
  display: flex;
  gap: 12px;
  align-items: center;
}
.analytics-toolbar__scope {
  width: 240px;
}

/* 商品行为趋势独占首行，底部两个图表保持同排均分。 */
.goods-analytics__trend,
.goods-analytics__panels {
  grid-column: 1 / -1;
}

@media (width <= 768px) {
  .analytics-toolbar {
    width: 100%;
    align-items: stretch;
    flex-direction: column;
  }
  .analytics-toolbar__scope {
    width: 100%;
  }
}
</style>
