<template>
  <div>
    <el-card class="analytics-card analytics-card--summary" shadow="never">
      <div class="analytics-toolbar">
        <div>
          <h2 class="analytics-title">{{ title }}</h2>
          <p class="analytics-desc">{{ description }}</p>
        </div>
        <span class="analytics-period">{{ periodLabel }}</span>
        <div class="analytics-toolbar__tabs">
          <slot name="toolbar" />
        </div>
      </div>
      <slot name="metrics" />
    </el-card>

    <section class="chart-grid" :style="{ gridTemplateColumns: chartColumns }">
      <slot />
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";

const props = withDefaults(
  defineProps<{
    title: string;
    description: string;
    periodLabel: string;
    contentRatio?: string;
  }>(),
  {
    contentRatio: "minmax(0, 1.25fr) minmax(320px, 0.9fr)"
  }
);

const chartColumns = computed(() => props.contentRatio);
</script>

<style scoped lang="scss">
.analytics-card {
  border: 1px solid #e5eaf1;
  border-radius: 16px;
  box-shadow: 0 8px 24px rgb(15 23 42 / 4%);
}

:deep(.analytics-card .el-card__body) {
  padding: 18px;
}

.analytics-card--summary {
  margin-bottom: 16px;
}

.analytics-toolbar {
  display: flex;
  gap: 24px;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}

.analytics-title {
  margin: 0;
  font-size: 22px;
  font-weight: 700;
  color: #1f2937;
}

.analytics-desc {
  max-width: 560px;
  margin: 8px 0 0;
  color: #64748b;
  line-height: 1.7;
}

.analytics-period {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 72px;
  height: 32px;
  padding: 0 12px;
  font-size: 12px;
  font-weight: 600;
  color: #2563eb;
  background: #eff6ff;
  border: 1px solid #dbeafe;
  border-radius: 999px;
}

.analytics-toolbar__tabs {
  min-width: 280px;
}

.chart-grid {
  display: grid;
  gap: 16px;
  margin-top: 16px;
}

@media (max-width: 1200px) {
  .chart-grid {
    grid-template-columns: minmax(0, 1fr) !important;
  }
}

@media (max-width: 768px) {
  .analytics-toolbar {
    flex-direction: column;
    align-items: stretch;
  }

  .analytics-toolbar__tabs {
    width: 100%;
    min-width: 0;
  }

  .chart-grid {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
