<template>
  <div class="summary-grid" :style="{ gridTemplateColumns: gridTemplateColumns }">
    <article v-for="metric in items" :key="metric.key" class="summary-card" :style="{ '--card-accent': metric.color }">
      <div class="summary-card__meta">
        <div>
          <span class="summary-card__label">{{ metric.label }}</span>
          <div class="summary-card__value">{{ metric.value }}</div>
        </div>
        <div v-if="metric.icon" class="summary-card__icon">
          <el-icon :size="20">
            <component :is="metric.icon" />
          </el-icon>
        </div>
      </div>
      <div class="summary-card__foot">
        <span>{{ metric.footLabel }}</span>
        <b>{{ metric.footValue }}</b>
      </div>
    </article>
  </div>
</template>

<script setup lang="ts">
import { computed, type Component } from "vue";

export interface AnalyticsMetricCardItem {
  key: string;
  label: string;
  value: string;
  footLabel: string;
  footValue: string;
  color: string;
  icon?: Component;
}

const props = defineProps<{
  items: AnalyticsMetricCardItem[];
}>();

const gridTemplateColumns = computed(() => {
  const count = props.items.length;
  if (count <= 1) return "minmax(0, 1fr)";
  return `repeat(${count}, minmax(0, 1fr))`;
});
</script>

<style scoped lang="scss">
.summary-grid {
  display: grid;
  gap: 12px;
}

.summary-card {
  padding: 14px;
  border: 1px solid #e8edf4;
  border-radius: 12px;
  background: #fff;
}

.summary-card__meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.summary-card__label {
  display: block;
  font-size: 14px;
  color: #64748b;
}

.summary-card__icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  color: #fff;
  border-radius: 10px;
  background: var(--card-accent);
}

.summary-card__value {
  margin-top: 8px;
  font-size: 24px;
  font-weight: 700;
  color: #1f2937;
}

.summary-card__foot {
  display: flex;
  gap: 8px;
  align-items: center;
  justify-content: space-between;
  margin-top: 10px;
  padding-top: 10px;
  border-top: 1px solid #eef2f7;
  font-size: 13px;
  color: #94a3b8;
}

.summary-card__foot b {
  color: var(--card-accent);
}

@media (max-width: 1200px) {
  .summary-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 768px) {
  .summary-grid {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
