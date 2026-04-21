<template>
  <div>
    <el-card class="analytics-card analytics-card--summary" shadow="never">
      <div class="analytics-toolbar">
        <div v-if="hasHeaderText">
          <h2 v-if="title" class="analytics-title">{{ title }}</h2>
          <p v-if="description" class="analytics-desc" :class="{ 'analytics-desc--compact': !title }">
            {{ description }}
          </p>
        </div>
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
    title?: string;
    description?: string;
    periodLabel?: string;
    contentRatio?: string;
  }>(),
  {
    title: "",
    description: "",
    periodLabel: "",
    contentRatio: "minmax(0, 1.25fr) minmax(320px, 0.9fr)"
  }
);

/** 统一控制页头文案显隐，避免空标题或空描述继续占位。 */
const hasHeaderText = computed(() => Boolean(props.title || props.description));

const chartColumns = computed(() => props.contentRatio);
</script>

<style scoped lang="scss">
.analytics-card {
  border: 1px solid var(--admin-page-card-border);
  border-radius: 16px;
  background: var(--admin-page-card-bg);
  box-shadow: var(--admin-page-shadow);
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
  color: var(--admin-page-text-primary);
}

.analytics-desc {
  max-width: 560px;
  margin: 8px 0 0;
  color: var(--admin-page-text-secondary);
  line-height: 1.7;
}

.analytics-desc--compact {
  margin-top: 0;
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
