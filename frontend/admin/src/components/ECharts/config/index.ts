import * as echarts from "echarts/core";
import { BarChart, LineChart, LinesChart, PieChart, ScatterChart, RadarChart, GaugeChart, GraphChart } from "echarts/charts";
import {
  TitleComponent,
  TooltipComponent,
  GridComponent,
  DatasetComponent,
  TransformComponent,
  LegendComponent,
  PolarComponent,
  GeoComponent,
  ToolboxComponent,
  DataZoomComponent
} from "echarts/components";
import { LabelLayout, UniversalTransition } from "echarts/features";
import { CanvasRenderer } from "echarts/renderers";
import type {
  BarSeriesOption,
  LineSeriesOption,
  LinesSeriesOption,
  PieSeriesOption,
  ScatterSeriesOption,
  RadarSeriesOption,
  GaugeSeriesOption,
  GraphSeriesOption
} from "echarts/charts";
import type {
  TitleComponentOption,
  TooltipComponentOption,
  GridComponentOption,
  DatasetComponentOption
} from "echarts/components";
import type { ComposeOption } from "echarts/core";
import "echarts-liquidfill";

export type ECOption = ComposeOption<
  | BarSeriesOption
  | LineSeriesOption
  | LinesSeriesOption
  | PieSeriesOption
  | RadarSeriesOption
  | GaugeSeriesOption
  | TitleComponentOption
  | TooltipComponentOption
  | GridComponentOption
  | DatasetComponentOption
  | ScatterSeriesOption
  | GraphSeriesOption
>;

echarts.use([
  TitleComponent,
  TooltipComponent,
  GridComponent,
  DatasetComponent,
  TransformComponent,
  LegendComponent,
  PolarComponent,
  GeoComponent,
  ToolboxComponent,
  DataZoomComponent,
  BarChart,
  LineChart,
  LinesChart,
  PieChart,
  ScatterChart,
  RadarChart,
  GaugeChart,
  GraphChart,
  LabelLayout,
  UniversalTransition,
  CanvasRenderer
]);

export default echarts;
