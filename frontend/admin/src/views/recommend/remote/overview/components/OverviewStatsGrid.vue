<template>
  <section v-loading="loading" class="remote-metric-grid">
    <article v-for="item in stats" :key="item.name" class="remote-metric-card">
      <div class="remote-metric-card__header">
        <span>{{ item.label }}</span>
        <el-tag :type="item.increase ? 'success' : 'danger'" effect="light" round>
          {{ item.increase ? "+" : "" }}{{ formatRemoteNumber(item.diff) }}
        </el-tag>
      </div>
      <strong>{{ formatRemoteNumber(item.current) }}</strong>
      <p>{{ item.description }}</p>
    </article>
  </section>
</template>

<script setup lang="ts">
import { formatRemoteNumber, type RemoteTimeseriesMetric } from "../../utils";

/** 概览统计卡片入参。 */
interface OverviewStatsGridProps {
  /** 是否正在加载统计数据。 */
  loading?: boolean;
  /** 统计卡片数据。 */
  stats: RemoteTimeseriesMetric[];
}

withDefaults(defineProps<OverviewStatsGridProps>(), {
  loading: false
});
</script>

<style scoped lang="scss">
.remote-metric-grid {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 16px;
}

.remote-metric-card {
  padding: 18px;
  border: 1px solid var(--admin-page-card-border);
  border-radius: var(--admin-page-radius);
  background: var(--admin-page-card-bg);
  box-shadow: var(--admin-page-shadow);

  &__header {
    display: flex;
    gap: 8px;
    align-items: center;
    justify-content: space-between;
    color: var(--admin-page-text-secondary);
    font-size: 13px;
  }

  strong {
    display: block;
    margin-top: 12px;
    color: var(--admin-page-text-primary);
    font-size: 28px;
    line-height: 1;
  }

  p {
    margin: 10px 0 0;
    color: var(--admin-page-text-secondary);
    font-size: 13px;
  }
}

@media (max-width: 1200px) {
  .remote-metric-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}

@media (max-width: 900px) {
  .remote-metric-grid {
    grid-template-columns: 1fr;
  }
}
</style>
