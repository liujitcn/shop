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
.goods-side-panels {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
</style>
