<template>
  <div class="gorse-overview-page">
    <MetricCards />
    <PerformanceChart />
    <DashboardRecommendCard />
  </div>
</template>

<script setup lang="ts">
import MetricCards from "./components/MetricCards.vue";
import PerformanceChart from "./components/PerformanceChart.vue";
import DashboardRecommendCard from "./components/DashboardRecommendCard.vue";
</script>

<style scoped lang="scss">
.gorse-overview-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
</style>
