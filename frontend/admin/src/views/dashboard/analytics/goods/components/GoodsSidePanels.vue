<template>
  <div class="goods-side-panels">
    <GoodsHotRankChart :time-type="timeType" />
    <GoodsCategoryChart :time-type="timeType" />
  </div>
</template>

<script setup lang="ts">
import type { AnalyticsTimeType } from "@/rpc/common/analytics";
import GoodsCategoryChart from "./GoodsCategoryChart.vue";
import GoodsHotRankChart from "./GoodsHotRankChart.vue";

defineProps<{
  timeType: AnalyticsTimeType;
}>();
</script>

<style scoped lang="scss">
/* 底部两个分析卡片按 5:5 比例横向排列，小屏时再回落为单列。 */
.goods-side-panels {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
}

@media (max-width: 768px) {
  .goods-side-panels {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
