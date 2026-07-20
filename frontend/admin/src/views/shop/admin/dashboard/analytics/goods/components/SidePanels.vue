<template>
  <div class="goods-side-panels">
    <HotRankChart :time-type="timeType" :tenant-id="tenantId" :tenant-store-id="tenantStoreId" />
    <CategoryChart :time-type="timeType" :tenant-id="tenantId" :tenant-store-id="tenantStoreId" />
  </div>
</template>

<script setup lang="ts">
import type { AnalyticsTimeType } from "@/rpc/common/v1/analytics";
import CategoryChart from "./CategoryChart.vue";
import HotRankChart from "./HotRankChart.vue";

defineProps<{
  timeType: AnalyticsTimeType;
  /** 默认租户选择的租户ID。 */
  tenantId?: number;
  /** 当前选择的门店ID。 */
  tenantStoreId?: number;
}>();
</script>

<style scoped lang="scss">
/* 底部两个分析卡片按 5:5 比例横向排列，小屏时再回落为单列。 */
.goods-side-panels {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
}

@media (width <= 768px) {
  .goods-side-panels {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
