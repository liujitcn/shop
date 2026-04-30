<template>
  <div v-loading="overviewLoading" class="gorse-overview-metrics">
    <el-card
      v-for="card in metricCards"
      :key="card.key"
      class="gorse-overview-metric"
      shadow="never"
      :style="{ '--metric-color': card.color }"
    >
      <div class="gorse-overview-metric__label">{{ card.label }}</div>
      <div class="gorse-overview-metric__value">{{ card.value }}</div>
      <div :class="['gorse-overview-metric__delta', card.increase ? 'is-up' : 'is-down']">
        <span>{{ card.increase ? "▲" : "▼" }}</span>
        <span>{{ card.delta }}</span>
      </div>
      <el-tooltip placement="top" :disabled="!card.tooltipRows.length">
        <template #content>
          <div v-for="row in card.tooltipRows" :key="row" class="gorse-overview-metric__tooltip-row">{{ row }}</div>
        </template>
        <svg
          class="gorse-overview-metric__sparkline"
          :viewBox="`0 0 ${sparklineViewBoxWidth} ${sparklineViewBoxHeight}`"
          preserveAspectRatio="none"
          aria-hidden="true"
        >
          <path class="gorse-overview-metric__sparkline-fill" :d="card.sparklineFill" />
          <path class="gorse-overview-metric__sparkline-line" :d="card.sparkline" />
        </svg>
      </el-tooltip>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import dayjs from "dayjs";
import { ElMessage } from "element-plus";
import { defRecommendGorseService } from "@/api/admin/recommend_gorse";
import type { TimeSeriesPoint } from "@/rpc/admin/v1/recommend_gorse";

/** 概览指标元数据。 */
interface MetricDefinition {
  /** 指标接口名称。 */
  key: string;
  /** 页面展示名称。 */
  label: string;
  /** 指标强调色。 */
  color: string;
}

/** 概览指标卡片。 */
interface MetricCard {
  /** 指标接口名称。 */
  key: string;
  /** 页面展示名称。 */
  label: string;
  /** 指标强调色。 */
  color: string;
  /** 当前值文本。 */
  value: string;
  /** 环比变化文本。 */
  delta: string;
  /** 是否增长。 */
  increase: boolean;
  /** 迷你趋势图平滑折线路径。 */
  sparkline: string;
  /** 迷你趋势图填充区域路径。 */
  sparklineFill: string;
  /** 迷你趋势图悬浮提示行。 */
  tooltipRows: string[];
}

/** 迷你趋势图坐标点。 */
interface SparklineCoord {
  /** 横轴坐标。 */
  x: number;
  /** 纵轴坐标。 */
  y: number;
}

/** 迷你趋势图横向视图宽度。 */
const sparklineViewBoxWidth = 120;
/** 迷你趋势图纵向视图高度。 */
const sparklineViewBoxHeight = 42;
/** 迷你趋势图顶部留白，避免曲线贴顶。 */
const sparklineTopY = 8;
/** 迷你趋势图底部位置，最低点贴住卡片底部。 */
const sparklineBottomY = sparklineViewBoxHeight;

const metricDefinitions: MetricDefinition[] = [
  { key: "num_users", label: "用户数", color: "#00b8d8" },
  { key: "num_items", label: "商品数", color: "#17c671" },
  { key: "num_feedback", label: "反馈数", color: "#ffb400" },
  { key: "num_pos_feedbacks", label: "正向反馈", color: "#ff4169" },
  { key: "num_neg_feedbacks", label: "负向反馈", color: "#2d8cff" }
];

const overviewLoading = ref(false);
const overviewMetricMap = ref<Record<string, TimeSeriesPoint[]>>({});

const metricCards = computed<MetricCard[]>(() =>
  metricDefinitions.map(definition => {
    const points = overviewMetricMap.value[definition.key] ?? [];
    const value = readLatestValue(points);
    const delta = value - readPreviousValue(points);
    return {
      key: definition.key,
      label: definition.label,
      color: definition.color,
      value: formatCount(value),
      delta: formatCount(Math.abs(delta)),
      increase: delta >= 0,
      sparkline: buildSparkline(points),
      sparklineFill: buildSparklineFill(points),
      tooltipRows: buildSparklineTooltipRows(points)
    };
  })
);

/** 加载顶部五个概览指标数据。 */
async function loadOverview() {
  overviewLoading.value = true;
  try {
    const metricMap: Record<string, TimeSeriesPoint[]> = {};
    const metricList = await Promise.all(
      metricDefinitions.map(async metric => {
        const data = await defRecommendGorseService.GetTimeSeries({
          name: metric.key,
          begin: "",
          end: ""
        });
        return { name: metric.key, points: normalizeTimeSeriesPoints(data) };
      })
    );
    metricList.forEach(metric => {
      metricMap[metric.name] = metric.points;
    });
    overviewMetricMap.value = metricMap;
  } finally {
    overviewLoading.value = false;
  }
}

/** 读取最后一个指标点的数值。 */
function readLatestValue(points: TimeSeriesPoint[]) {
  if (!points.length) return 0;
  return Number(points[points.length - 1].value || 0);
}

/** 读取倒数第二个指标点的数值。 */
function readPreviousValue(points: TimeSeriesPoint[]) {
  if (points.length < 2) return 0;
  return Number(points[points.length - 2].value || 0);
}

/** 格式化统计卡片数值。 */
function formatCount(value: number) {
  return Number.isInteger(value) ? value.toLocaleString() : value.toFixed(5);
}

/** 格式化趋势图横轴时间。 */
function formatAxisTime(value: string) {
  if (!value) return "--";
  return dayjs(value).format("MM-DD HH:mm");
}

/** 兼容后台代理 points 包装和 Gorse 原生数组两种时间序列返回结构。 */
function normalizeTimeSeriesPoints(response: unknown): TimeSeriesPoint[] {
  const responseRecord =
    typeof response === "object" && response !== null && !Array.isArray(response) ? (response as Record<string, unknown>) : {};
  const rawPoints = (
    Array.isArray(response)
      ? response
      : Array.isArray(responseRecord.Points ?? responseRecord.points)
        ? (responseRecord.Points ?? responseRecord.points)
        : []
  ) as unknown[];
  return rawPoints
    .map(point => {
      const record =
        typeof point === "object" && point !== null && !Array.isArray(point) ? (point as Record<string, unknown>) : {};
      const timestamp = String(record.timestamp ?? record.Timestamp ?? "");
      const value = Number(record.value ?? record.Value ?? 0);
      return { name: String(record.name ?? record.Name ?? ""), timestamp, value };
    })
    .filter(point => point.timestamp);
}

/** 构建统计卡片迷你折线图坐标。 */
function buildSparkline(points: TimeSeriesPoint[]) {
  return buildSmoothSparklinePath(buildSparklineCoords(points));
}

/** 构建统计卡片迷你折线图填充区域路径。 */
function buildSparklineFill(points: TimeSeriesPoint[]) {
  const coords = buildSparklineCoords(points);
  if (!coords.length) return "";

  const firstCoord = coords[0];
  const lastCoord = coords[coords.length - 1];
  const linePath = buildSmoothSparklinePath(coords);
  // 填充区域沿平滑曲线闭合到底部，后续预留槽位保持空白。
  return `${linePath} L ${formatSparklineCoord(lastCoord.x)},${formatSparklineCoord(
    sparklineViewBoxHeight
  )} L ${formatSparklineCoord(firstCoord.x)},${formatSparklineCoord(sparklineViewBoxHeight)} Z`;
}

/** 构建统计卡片迷你折线图悬浮提示。 */
function buildSparklineTooltipRows(points: TimeSeriesPoint[]) {
  return points.map(point => `${formatAxisTime(point.timestamp)}：${formatCount(Number(point.value || 0))}`);
}

/** 构建统计卡片迷你折线图基础坐标。 */
function buildSparklineCoords(points: TimeSeriesPoint[]): SparklineCoord[] {
  const values = points.map(point => Number(point.value || 0));
  if (!values.length) return [];

  const minValue = Math.min(...values);
  const maxValue = Math.max(...values);
  const valueRange = maxValue - minValue;
  const lastIndex = Math.max(values.length - 1, 4);
  return values.map((value, index) => {
    // 原始卡片固定预留 5 个横轴槽位，数据不足时不拉伸到最右侧。
    const x = (index / lastIndex) * sparklineViewBoxWidth;
    // 最低点贴住底部，最高点停在顶部留白线，不顶到卡片上沿。
    const y =
      valueRange > 0
        ? sparklineBottomY - ((value - minValue) / valueRange) * (sparklineBottomY - sparklineTopY)
        : sparklineBottomY;
    return { x, y };
  });
}

/** 构建穿过所有坐标点的平滑 SVG 路径。 */
function buildSmoothSparklinePath(coords: SparklineCoord[]) {
  if (!coords.length) return "";

  const firstCoord = coords[0];
  if (coords.length === 1) {
    // 单点数据补一条极短线段，避免 SVG path 只有移动命令时不可见。
    return `M ${formatSparklineCoord(firstCoord.x)},${formatSparklineCoord(firstCoord.y)} L ${formatSparklineCoord(
      firstCoord.x + 0.01
    )},${formatSparklineCoord(firstCoord.y)}`;
  }

  const commands = [`M ${formatSparklineCoord(firstCoord.x)},${formatSparklineCoord(firstCoord.y)}`];
  for (let index = 0; index < coords.length - 1; index += 1) {
    const previousCoord = coords[Math.max(0, index - 1)];
    const currentCoord = coords[index];
    const nextCoord = coords[index + 1];
    const afterNextCoord = coords[Math.min(coords.length - 1, index + 2)];
    // 使用 Catmull-Rom 到三次贝塞尔转换，让曲线平滑并经过每个数据点。
    const controlPoint1 = {
      x: currentCoord.x + (nextCoord.x - previousCoord.x) / 6,
      y: clampSparklineY(currentCoord.y + (nextCoord.y - previousCoord.y) / 6)
    };
    const controlPoint2 = {
      x: nextCoord.x - (afterNextCoord.x - currentCoord.x) / 6,
      y: clampSparklineY(nextCoord.y - (afterNextCoord.y - currentCoord.y) / 6)
    };
    commands.push(
      `C ${formatSparklineCoord(controlPoint1.x)},${formatSparklineCoord(controlPoint1.y)} ${formatSparklineCoord(
        controlPoint2.x
      )},${formatSparklineCoord(controlPoint2.y)} ${formatSparklineCoord(nextCoord.x)},${formatSparklineCoord(nextCoord.y)}`
    );
  }

  return commands.join(" ");
}

/** 格式化 SVG 坐标，避免路径字符串包含过长小数。 */
function formatSparklineCoord(value: number) {
  return value.toFixed(2);
}

/** 限制平滑曲线控制点纵坐标，避免贝塞尔曲线越过预留边界。 */
function clampSparklineY(value: number) {
  return Math.min(sparklineBottomY, Math.max(sparklineTopY, value));
}

onMounted(() => {
  loadOverview().catch(() => {
    ElMessage.error("加载 Gorse 推荐概览指标失败");
  });
});
</script>

<style scoped lang="scss">
.gorse-overview-metrics {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 16px;
}

.gorse-overview-metric {
  position: relative;
  min-height: 140px;
  overflow: hidden;
  border: 1px solid var(--admin-page-card-border);
  background: var(--admin-page-card-bg);
  box-shadow: var(--admin-page-shadow);

  :deep(.el-card__body) {
    position: static;
    display: flex;
    height: 100%;
    min-height: 140px;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 18px 16px 22px;
  }

  &__label {
    position: relative;
    z-index: 1;
    color: var(--admin-page-text-secondary);
    font-size: 12px;
    font-weight: 700;
    letter-spacing: 1.5px;
    line-height: 18px;
    text-align: center;
    text-transform: uppercase;
  }

  &__value {
    position: relative;
    z-index: 1;
    margin-top: 8px;
    color: var(--admin-page-text-primary);
    font-size: 34px;
    font-weight: 700;
    line-height: 40px;
  }

  &__delta {
    position: relative;
    z-index: 1;
    display: inline-flex;
    gap: 4px;
    align-items: center;
    margin-top: 8px;
    font-size: 12px;
    font-weight: 700;

    &.is-up {
      color: var(--el-color-success);
    }

    &.is-down {
      color: var(--el-color-danger);
    }
  }

  &__sparkline {
    position: absolute;
    inset: 0;
    z-index: 0;
    width: 100%;
    // 趋势图铺满整个卡片，文字通过更高层级悬浮在折线上方。
    height: 100%;
    overflow: visible;

    &-fill {
      fill: color-mix(in srgb, var(--metric-color) 12%, transparent);
    }

    &-line {
      fill: none;
      stroke: var(--metric-color);
      stroke-width: 2;
      vector-effect: non-scaling-stroke;
    }
  }
}

@media (max-width: 1400px) {
  .gorse-overview-metrics {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}

@media (max-width: 900px) {
  .gorse-overview-metrics {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 560px) {
  .gorse-overview-metrics {
    grid-template-columns: 1fr;
  }
}
</style>
