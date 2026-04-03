<template>
  <article class="chart-card" :class="cardClass">
    <div class="chart-card__header">
      <div>
        <h3 class="chart-card__title">{{ title }}</h3>
        <p v-if="description" class="chart-card__desc">{{ description }}</p>
      </div>
      <slot name="extra" />
    </div>
    <div class="chart-card__body">
      <slot />
    </div>
  </article>
</template>

<script setup lang="ts">
import { computed } from "vue";

const props = withDefaults(
  defineProps<{
    title: string;
    description?: string;
    primary?: boolean;
  }>(),
  {
    description: "",
    primary: false
  }
);

const cardClass = computed(() => ({
  "chart-card--primary": props.primary,
  "chart-card--secondary": !props.primary
}));
</script>

<style scoped lang="scss">
.chart-card {
  padding: 18px;
  border: 1px solid #e5eaf1;
  border-radius: 16px;
  background: #fff;
  box-shadow: 0 8px 24px rgb(15 23 42 / 4%);
}

.chart-card__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 12px;
}

.chart-card__title {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: #1f2937;
}

.chart-card__desc {
  margin: 6px 0 0;
  font-size: 13px;
  line-height: 1.7;
  color: #64748b;
}

.chart-card__body {
  height: 360px;
}
</style>
