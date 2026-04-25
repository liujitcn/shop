<template>
  <div v-loading="loading" class="remote-flow-page">
    <div class="recflow-palette">
      <button
        v-for="tool in nodeTools"
        :key="tool.type"
        class="recflow-palette__item"
        draggable="true"
        type="button"
        @dragstart="handleToolDragStart(tool.type)"
      >
        <span>{{ tool.icon }}</span>
        <strong>{{ tool.label }}</strong>
      </button>
      <div class="recflow-palette__actions">
        <el-button :loading="loading" @click="loadFlow">刷新</el-button>
        <el-button type="danger" plain :loading="resetting" @click="resetFlow">Clear</el-button>
        <el-button type="primary" :loading="saving" @click="saveFlow">Save</el-button>
      </div>
    </div>

    <section
      ref="boardRef"
      class="recflow-board"
      :style="{ height: `${chartHeight}px` }"
      @dragover.prevent
      @drop.prevent="handleBoardDrop"
    >
      <ECharts
        ref="chartRef"
        :option="chartOption"
        :height="chartHeight"
        :on-click="handleChartClick"
        :on-mouseup="handleChartMouseup"
      />
    </section>

    <section class="remote-flow-debug">
      <el-card class="remote-flow-card" shadow="never">
        <template #header>
          <div class="remote-flow-card__header">
            <strong>Data Export</strong>
            <span>导出远程推荐当前页数据</span>
          </div>
        </template>
        <el-form :model="dataForm" label-width="110px">
          <el-form-item label="数据类型">
            <el-select v-model="dataForm.type" style="width: 240px" @change="handleDataTypeChange">
              <el-option v-for="item in dataTypes" :key="item.value" :label="item.label" :value="item.value" />
            </el-select>
          </el-form-item>
          <el-form-item label="返回数量">
            <el-input-number v-model="dataForm.n" :min="1" :max="500" :step="50" controls-position="right" />
          </el-form-item>
          <el-form-item label="游标">
            <el-input v-model.trim="dataForm.cursor" clearable placeholder="继续导出下一页时填写上次返回的游标" />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" :loading="exportLoading" @click="exportData">导出当前页</el-button>
            <el-button :disabled="!exportNextCursor" :loading="exportLoading" @click="exportNextPage">导出下一页</el-button>
          </el-form-item>
        </el-form>
      </el-card>

      <el-card class="remote-flow-card" shadow="never">
        <template #header>
          <div class="remote-flow-card__header">
            <strong>Data Import</strong>
            <span>写入远程推荐引擎</span>
          </div>
        </template>
        <el-form label-width="110px">
          <el-form-item label="导入类型">
            <el-tag effect="light">{{ selectedDataTypeLabel }}</el-tag>
          </el-form-item>
          <el-form-item label="导入 JSON">
            <el-input v-model="importJson" type="textarea" :rows="11" placeholder="粘贴远程推荐用户或商品 JSON 数组" />
          </el-form-item>
          <el-form-item>
            <el-button type="success" :loading="importLoading" @click="importData">导入到远程推荐</el-button>
          </el-form-item>
        </el-form>
      </el-card>
    </section>

    <FormDialog
      v-model="nodeDialogVisible"
      :title="nodeDialogTitle"
      width="720px"
      :model="nodeFormModel"
      :fields="nodeFormFields"
      label-width="128px"
      confirm-text="应用到画布"
      cancel-text="取消"
      :destroy-on-close="true"
      :close-on-click-modal="false"
      @confirm="applyNodeDialog"
      @closed="resetNodeDialog"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import type { ECElementEvent, EChartsType } from "echarts/core";
import type { ECOption } from "@/components/ECharts/config";
import ECharts from "@/components/ECharts/index.vue";
import FormDialog from "@/components/Dialog/FormDialog.vue";
import type { ProFormField, ProFormOption } from "@/components/ProForm/interface";
import { defRecommendRemoteService } from "@/api/admin/recommend_remote";
import { formatRemoteJson, type RemoteRecord } from "../utils";

defineOptions({
  name: "RemoteFlow"
});

type RemoteConfigRecord = Record<string | number, unknown>;
type FieldValueType = "string" | "textarea" | "number" | "boolean" | "array" | "json";
type NodeToolType =
  | "latest"
  | "collaborative"
  | "non-personalized"
  | "user-to-user"
  | "item-to-item"
  | "external"
  | "ranker"
  | "fallback";

interface FlowNode {
  id: string;
  name: string;
  category: string;
  description: string;
  configPath: Array<string | number>;
  x: number;
  y: number;
  width: number;
  icon: string;
  value: unknown;
}

interface FlowEdge {
  source: string;
  target: string;
  lineStyle?: {
    type?: "solid" | "dashed";
  };
}

interface NodeFieldMeta {
  prop: string;
  sourceKey: string;
  valueType: FieldValueType;
}

interface NodeTool {
  type: NodeToolType;
  label: string;
  icon: string;
}

interface NodePosition {
  x: number;
  y: number;
}

interface DataTypeOption {
  label: string;
  value: "users" | "items";
}

const nodeTools: NodeTool[] = [
  { type: "latest", label: "Latest", icon: "✹" },
  { type: "collaborative", label: "Collaborative", icon: "●" },
  { type: "non-personalized", label: "Non-Personalized", icon: "◔" },
  { type: "user-to-user", label: "User to User", icon: "♟" },
  { type: "item-to-item", label: "Item to Item", icon: "▦" },
  { type: "external", label: "External", icon: "☁" },
  { type: "ranker", label: "Ranker", icon: "≡" },
  { type: "fallback", label: "Fallback", icon: "◴" }
];

const dataTypes: DataTypeOption[] = [
  { label: "用户数据", value: "users" },
  { label: "商品数据", value: "items" }
];

const nodeLabelMap: Record<string, string> = {
  name: "名称",
  type: "类型",
  column: "字段列",
  prompt: "提示词",
  score: "评分表达式",
  filter: "过滤条件",
  recommenders: "推荐器列表",
  cache_expire: "缓存过期时间",
  fit_period: "训练周期",
  fit_epoch: "训练轮数",
  optimize_period: "调参周期",
  optimize_trials: "调参次数",
  query_template: "查询模板",
  document_template: "文档模板",
  early_stopping: "早停配置",
  reranker_api: "重排接口",
  positive_feedback_types: "正反馈类型",
  read_feedback_types: "阅读反馈类型",
  negative_feedback_types: "负反馈类型",
  positive_feedback_ttl: "正反馈有效期",
  item_ttl: "商品有效期",
  enable_replacement: "启用替换",
  positive_replacement_decay: "正反馈替换衰减",
  read_replacement_decay: "阅读替换衰减",
  cache_size: "缓存数量",
  context_size: "上下文数量",
  active_user_ttl: "活跃用户有效期"
};

const nodeDescriptionMap: Record<string, string> = {
  data_source: "数据源节点决定哪些反馈类型进入推荐链路。",
  collaborative: "协同过滤召回节点，负责个性化召回。",
  ranker: "排序节点，配置排序模型以及参与排序的召回器。",
  fallback: "兜底节点，配置排序无结果时的后备推荐器。",
  recommend: "推荐输出节点，展示 recommend 总配置。",
  latest: "最新内容召回节点。"
};

const canvasWidth = 2040;
const canvasMinHeight = 930;
const sourceNodeWidth = 224;
const recallNodeWidth = 404;
const actionNodeWidth = 180;
const recommendNodeWidth = 240;
const nodeHeight = 72;

const loading = ref(false);
const saving = ref(false);
const resetting = ref(false);
const rawConfig = ref<RemoteConfigRecord>({});
const selectedNodeID = ref("");
const nodeDialogVisible = ref(false);
const editingNode = ref<FlowNode>();
const nodeFormModel = ref<Record<string, any>>({});
const nodeFormFields = ref<ProFormField[]>([]);
const nodeFieldMetas = ref<NodeFieldMeta[]>([]);
const draggedToolType = ref<NodeToolType>();
const nodePositions = ref<Record<string, NodePosition>>({});
const boardRef = ref<HTMLElement>();
const chartRef = ref<InstanceType<typeof ECharts>>();
const dataForm = reactive({
  type: "items" as DataTypeOption["value"],
  cursor: "",
  n: 100
});
const exportLoading = ref(false);
const importLoading = ref(false);
const exportRows = ref<RemoteRecord[]>([]);
const exportNextCursor = ref("");
const importJson = ref("[]");

const recommendConfig = computed(() => resolveRecord(rawConfig.value.recommend));
const graphModel = computed(() => buildGraphModel(recommendConfig.value));
const flowNodes = computed(() => graphModel.value.nodes);
const recommenderOptions = computed<ProFormOption[]>(() =>
  flowNodes.value
    .filter(node => node.category !== "入口" && node.category !== "输出")
    .map(node => ({ label: node.name, value: node.id }))
);
const nodeDialogTitle = computed(() => (editingNode.value ? `编辑节点：${editingNode.value.name}` : "编辑节点"));
const chartHeight = computed(() => Math.max(canvasMinHeight, ...flowNodes.value.map(node => node.y + 110)));
const chartOption = computed(() => buildChartOption(flowNodes.value, graphModel.value.edges, selectedNodeID.value));
const selectedDataTypeLabel = computed(() => dataTypes.find(item => item.value === dataForm.type)?.label ?? "商品数据");

function buildGraphModel(recommend: RemoteConfigRecord) {
  const nodes: FlowNode[] = [];
  const edges: FlowEdge[] = [];
  const sourceNode = createNode({
    id: "data_source",
    name: "Data Source",
    category: "入口",
    configPath: ["recommend", "data_source"],
    x: 186,
    y: 490,
    width: sourceNodeWidth,
    icon: "▦",
    value: recommend.data_source
  });
  nodes.push(sourceNode);

  const recallNodes: FlowNode[] = [];
  if (recommend.collaborative !== undefined) {
    recallNodes.push(
      createNode({
        id: "collaborative",
        name: "Collaborative",
        category: "召回",
        configPath: ["recommend", "collaborative"],
        x: 650,
        y: 185,
        width: recallNodeWidth,
        icon: "●",
        value: recommend.collaborative
      })
    );
  }

  resolveRecordList(recommend["item-to-item"]).forEach((item, index) => {
    const name = resolveName(item, `item_to_item_${index + 1}`);
    recallNodes.push(
      createNode({
        id: `item-to-item/${name}`,
        name,
        category: "召回",
        configPath: ["recommend", "item-to-item", index],
        x: 650,
        y: 300 + index * 120,
        width: recallNodeWidth,
        icon: "▦",
        value: item
      })
    );
  });

  resolveRecordList(recommend["user-to-user"]).forEach((item, index) => {
    const name = resolveName(item, `user_to_user_${index + 1}`);
    recallNodes.push(
      createNode({
        id: `user-to-user/${name}`,
        name,
        category: "召回",
        configPath: ["recommend", "user-to-user", index],
        x: 650,
        y: 420 + index * 120,
        width: recallNodeWidth,
        icon: "♟",
        value: item
      })
    );
  });

  resolveRecordList(recommend["non-personalized"]).forEach((item, index) => {
    const name = resolveName(item, `non_personalized_${index + 1}`);
    recallNodes.push(
      createNode({
        id: `non-personalized/${name}`,
        name,
        category: "召回",
        configPath: ["recommend", "non-personalized", index],
        x: 650,
        y: 540 + index * 120,
        width: recallNodeWidth,
        icon: "◔",
        value: item
      })
    );
  });

  resolveRecordList(recommend.external).forEach((item, index) => {
    const name = resolveName(item, `external_${index + 1}`);
    recallNodes.push(
      createNode({
        id: `external/${name}`,
        name,
        category: "召回",
        configPath: ["recommend", "external", index],
        x: 650,
        y: 660 + index * 120,
        width: recallNodeWidth,
        icon: "☁",
        value: item
      })
    );
  });

  recallNodes.push(
    createNode({
      id: "latest",
      name: "Latest",
      category: "召回",
      configPath: ["recommend", "latest"],
      x: 650,
      y: Math.max(850, 185 + recallNodes.length * 120),
      width: recallNodeWidth,
      icon: "✹",
      value: recommend.latest || {}
    })
  );

  normalizeRecallLayout(recallNodes);
  nodes.push(...recallNodes);
  recallNodes.forEach(node => edges.push({ source: sourceNode.id, target: node.id }));

  const rankerNode = createNode({
    id: "ranker",
    name: "Ranker",
    category: "排序",
    configPath: ["recommend", "ranker"],
    x: 1090,
    y: 420,
    width: actionNodeWidth,
    icon: "≡",
    value: recommend.ranker
  });
  nodes.push(rankerNode);
  resolveStringList(resolveRecord(recommend.ranker).recommenders).forEach(recommender => {
    if (nodes.some(node => node.id === recommender)) {
      edges.push({ source: recommender, target: rankerNode.id });
    }
  });

  const fallbackNode = createNode({
    id: "fallback",
    name: "Fallback",
    category: "兜底",
    configPath: ["recommend", "fallback"],
    x: 1090,
    y: 875,
    width: actionNodeWidth,
    icon: "◴",
    value: recommend.fallback
  });
  nodes.push(fallbackNode);
  resolveStringList(resolveRecord(recommend.fallback).recommenders).forEach(recommender => {
    if (nodes.some(node => node.id === recommender)) {
      edges.push({ source: recommender, target: fallbackNode.id, lineStyle: { type: "dashed" } });
    }
  });

  const recommendNode = createNode({
    id: "recommend",
    name: "Recommend",
    category: "输出",
    configPath: ["recommend"],
    x: 1450,
    y: 575,
    width: recommendNodeWidth,
    icon: "⚙",
    value: recommend
  });
  nodes.push(recommendNode);
  edges.push({ source: rankerNode.id, target: recommendNode.id }, { source: fallbackNode.id, target: recommendNode.id });

  return { nodes, edges };
}

function createNode(params: {
  id: string;
  name: string;
  category: string;
  configPath: Array<string | number>;
  x: number;
  y: number;
  width: number;
  icon: string;
  value: unknown;
}): FlowNode {
  const position = nodePositions.value[params.id];
  return {
    id: params.id,
    name: params.name,
    category: params.category,
    description: nodeDescriptionMap[params.id] || `${params.category}节点配置`,
    configPath: params.configPath,
    x: position?.x ?? params.x,
    y: position?.y ?? params.y,
    width: params.width,
    icon: params.icon,
    value: params.value
  };
}

function normalizeRecallLayout(nodes: FlowNode[]) {
  const top = 185;
  const gap = 120;
  nodes.forEach((node, index) => {
    if (nodePositions.value[node.id]) return;
    node.y = top + index * gap;
  });
}

function buildChartOption(nodes: FlowNode[], edges: FlowEdge[], activeID: string): ECOption {
  return {
    animation: false,
    grid: { left: 0, top: 0, right: 0, bottom: 0 },
    tooltip: { show: false },
    xAxis: { show: false, min: 0, max: canvasWidth },
    yAxis: { show: false, min: 0, max: chartHeight.value, inverse: true },
    series: [
      {
        type: "graph",
        coordinateSystem: "cartesian2d",
        layout: "none",
        roam: true,
        draggable: true,
        edgeSymbol: ["none", "arrow"],
        edgeSymbolSize: 14,
        cursor: "pointer",
        data: nodes.map(node => ({
          id: node.id,
          name: node.name,
          value: [node.x, node.y],
          symbol: "roundRect",
          symbolSize: [node.width, nodeHeight],
          itemStyle: {
            color: "#ffffff",
            borderColor: "#1677ff",
            borderWidth: 2,
            borderType: activeID === node.id ? "dashed" : "solid",
            shadowBlur: activeID === node.id ? 12 : 0,
            shadowColor: "rgba(22, 119, 255, 0.24)"
          },
          label: {
            show: true,
            align: "left",
            verticalAlign: "middle",
            distance: 0,
            color: "#20242a",
            fontSize: 20,
            fontWeight: 700,
            formatter: `{icon|${node.icon}}  {name|${node.name}}`,
            rich: {
              icon: { color: "#1677ff", fontSize: 22, fontWeight: 700, width: 24, align: "center", padding: [0, 0, 0, 24] },
              name: { color: "#20242a", fontSize: 20, fontWeight: 700 }
            }
          }
        })),
        links: edges,
        lineStyle: {
          color: "#1677ff",
          width: 2,
          curveness: 0.42
        },
        emphasis: {
          focus: "adjacency",
          lineStyle: { width: 3 }
        }
      }
    ]
  };
}

function handleToolDragStart(type: NodeToolType) {
  draggedToolType.value = type;
}

function handleBoardDrop(event: DragEvent) {
  if (!draggedToolType.value) return;
  addNodeByTool(draggedToolType.value, convertClientPointToChart(event.clientX, event.clientY));
  draggedToolType.value = undefined;
}

function handleChartMouseup(event: ECElementEvent) {
  if (event.dataType !== "node") return;
  const nodeID = String((event.data as { id?: string })?.id || event.name || "");
  const nativeEvent = event.event?.event as MouseEvent | undefined;
  if (!nodeID || !nativeEvent) return;
  nodePositions.value = {
    ...nodePositions.value,
    [nodeID]: convertClientPointToChart(nativeEvent.clientX, nativeEvent.clientY)
  };
}

function convertClientPointToChart(clientX: number, clientY: number): NodePosition {
  const chart = chartRef.value?.getInstance() as EChartsType | undefined;
  const rect = boardRef.value?.getBoundingClientRect();
  if (!chart || !rect) return { x: 900, y: 450 };
  const point = chart.convertFromPixel({ gridIndex: 0 }, [clientX - rect.left, clientY - rect.top]) as number[];
  return {
    x: Math.max(80, Math.min(canvasWidth - 120, Math.round(point[0] || 900))),
    y: Math.max(40, Math.round(point[1] || 450))
  };
}

function addNodeByTool(type: NodeToolType, position: NodePosition) {
  const recommend = ensureRecommendConfig();
  let nodeID = "";
  if (type === "collaborative") {
    recommend.collaborative = recommend.collaborative || buildDefaultNodeConfig(type, "collaborative");
    nodeID = "collaborative";
  } else if (type === "ranker") {
    recommend.ranker = recommend.ranker || buildDefaultNodeConfig(type, "ranker");
    nodeID = "ranker";
  } else if (type === "fallback") {
    recommend.fallback = recommend.fallback || buildDefaultNodeConfig(type, "fallback");
    nodeID = "fallback";
  } else if (type === "latest") {
    recommend.latest = recommend.latest || {};
    nodeID = "latest";
  } else {
    const listKey = resolveToolListKey(type);
    const list = ensureRecordList(recommend, listKey);
    const name = buildUniqueNodeName(type, list.length + 1);
    list.push(buildDefaultNodeConfig(type, name));
    nodeID = `${listKey}/${name}`;
  }

  rawConfig.value = cloneValue(rawConfig.value);
  nodePositions.value = { ...nodePositions.value, [nodeID]: position };
  selectedNodeID.value = nodeID;
}

function resolveToolListKey(type: NodeToolType) {
  if (type === "item-to-item") return "item-to-item";
  if (type === "user-to-user") return "user-to-user";
  if (type === "external") return "external";
  return "non-personalized";
}

function buildUniqueNodeName(type: NodeToolType, index: number) {
  const prefixMap: Record<NodeToolType, string> = {
    latest: "latest",
    collaborative: "collaborative",
    "non-personalized": "hot",
    "user-to-user": "similar_users",
    "item-to-item": "goods_relation",
    external: "external",
    ranker: "ranker",
    fallback: "fallback"
  };
  return `${prefixMap[type]}_${index}`;
}

function buildDefaultNodeConfig(type: NodeToolType, name: string): RemoteConfigRecord {
  if (type === "item-to-item") return { name, type: "users", column: "", prompt: "" };
  if (type === "user-to-user") return { name, type: "items", column: "" };
  if (type === "external") return { name, endpoint: "", timeout: "5s" };
  if (type === "ranker") return { type: "fm", recommenders: [], cache_expire: "120h", fit_period: "24h", fit_epoch: 40 };
  if (type === "fallback") return { recommenders: [] };
  if (type === "collaborative") return { type: "mf", fit_period: "24h", fit_epoch: 120 };
  return { name, score: "", filter: "" };
}

function handleChartClick(event: ECElementEvent) {
  if (event.dataType !== "node") return;
  const nodeID = String((event.data as { id?: string })?.id || event.name || "");
  const node = flowNodes.value.find(item => item.id === nodeID);
  if (!node) return;
  selectedNodeID.value = node.id;
  openNodeDialog(node);
}

function openNodeDialog(node: FlowNode) {
  editingNode.value = node;
  const value = getValueByPath(rawConfig.value, node.configPath) ?? {};
  const { model, fields, metas } = buildNodeForm(value);
  nodeFormModel.value = model;
  nodeFormFields.value = fields;
  nodeFieldMetas.value = metas;
  nodeDialogVisible.value = true;
}

function buildNodeForm(value: unknown) {
  const record = resolveRecord(value);
  const model: Record<string, any> = {};
  const fields: ProFormField[] = [];
  const metas: NodeFieldMeta[] = [];
  const keys = Object.keys(record);

  if (!keys.length) {
    model.__json = JSON.stringify(value ?? {}, null, 2);
    fields.push({
      prop: "__json",
      label: "节点配置",
      component: "textarea",
      props: { rows: 16, placeholder: "请输入 JSON 配置" }
    });
    metas.push({ prop: "__json", sourceKey: "", valueType: "json" });
    return { model, fields, metas };
  }

  keys.forEach(key => {
    const valueType = resolveFieldValueType(record[key]);
    const prop = encodeFieldProp(key);
    model[prop] = formatFieldValue(record[key], valueType);
    fields.push(buildProFormField(prop, key, valueType));
    metas.push({ prop, sourceKey: key, valueType });
  });

  return { model, fields, metas };
}

function buildProFormField(prop: string, key: string, valueType: FieldValueType): ProFormField {
  const label = resolveFieldLabel(key);
  if (valueType === "boolean") {
    return { prop, label, component: "switch" };
  }
  if (valueType === "number") {
    return {
      prop,
      label,
      component: "input-number",
      props: { controlsPosition: "right", style: { width: "100%" } }
    };
  }
  if (valueType === "array") {
    return {
      prop,
      label,
      component: "select",
      options: key === "recommenders" ? recommenderOptions.value : [],
      props: { multiple: true, filterable: true, allowCreate: true, defaultFirstOption: true, style: { width: "100%" } }
    };
  }
  if (valueType === "textarea" || valueType === "json") {
    return { prop, label, component: "textarea", props: { rows: 5 } };
  }
  return { prop, label, component: "input", props: { clearable: true } };
}

function resolveFieldValueType(value: unknown): FieldValueType {
  if (typeof value === "boolean") return "boolean";
  if (typeof value === "number") return "number";
  if (Array.isArray(value)) return "array";
  if (typeof value === "object" && value !== null) return "json";
  if (typeof value === "string" && value.length > 80) return "textarea";
  return "string";
}

function formatFieldValue(value: unknown, valueType: FieldValueType) {
  if (valueType === "json") return JSON.stringify(value ?? {}, null, 2);
  return cloneValue(value);
}

function applyNodeDialog() {
  const node = editingNode.value;
  if (!node) return;

  const jsonMeta = nodeFieldMetas.value.find(meta => meta.valueType === "json" && meta.prop === "__json");
  let nextValue: unknown = {};
  if (jsonMeta) {
    try {
      nextValue = JSON.parse(nodeFormModel.value.__json || "{}");
    } catch {
      ElMessage.error("节点配置 JSON 格式不正确");
      return;
    }
  } else {
    nextValue = nodeFieldMetas.value.reduce<RemoteConfigRecord>((record, meta) => {
      record[meta.sourceKey] = parseFormValue(nodeFormModel.value[meta.prop], meta.valueType);
      return record;
    }, {});
  }

  setValueByPath(rawConfig.value, node.configPath, nextValue);
  rawConfig.value = cloneValue(rawConfig.value);
  nodeDialogVisible.value = false;
  ElMessage.success("已应用到画布，点击 Save 后生效");
}

function parseFormValue(value: unknown, valueType: FieldValueType) {
  if (valueType !== "json") return cloneValue(value);
  try {
    return JSON.parse(String(value || "{}"));
  } catch {
    return value;
  }
}

function resetNodeDialog() {
  editingNode.value = undefined;
  nodeFormModel.value = {};
  nodeFormFields.value = [];
  nodeFieldMetas.value = [];
}

async function loadFlow() {
  loading.value = true;
  try {
    const config = await defRecommendRemoteService.GetFlowConfig({});
    rawConfig.value = cloneValue(config.config ?? {}) as RemoteConfigRecord;
    selectedNodeID.value = "";
    nodeDialogVisible.value = false;
    nodePositions.value = {};
  } catch (error) {
    ElMessage.error("加载推荐编排失败");
    throw error;
  } finally {
    loading.value = false;
  }
}

async function saveFlow() {
  saving.value = true;
  try {
    await defRecommendRemoteService.SaveFlowConfig({ json: formatRemoteJson(JSON.stringify(rawConfig.value)) });
    ElMessage.success("推荐编排保存成功");
    await loadFlow();
  } catch (error) {
    ElMessage.error("保存推荐编排失败");
    throw error;
  } finally {
    saving.value = false;
  }
}

async function resetFlow() {
  await ElMessageBox.confirm("是否确定重置远程推荐编排？重置后将恢复远程默认配置。", "警告", {
    confirmButtonText: "确认",
    cancelButtonText: "取消",
    type: "warning"
  });

  resetting.value = true;
  try {
    await defRecommendRemoteService.ResetFlowConfig({});
    ElMessage.success("推荐编排已重置");
    await loadFlow();
  } catch (error) {
    ElMessage.error("重置推荐编排失败");
    throw error;
  } finally {
    resetting.value = false;
  }
}

function handleDataTypeChange() {
  dataForm.cursor = "";
  exportRows.value = [];
  exportNextCursor.value = "";
}

async function exportData() {
  exportLoading.value = true;
  try {
    const data = await defRecommendRemoteService.ExportData({
      type: dataForm.type,
      cursor: dataForm.cursor,
      n: dataForm.n
    });
    exportRows.value = data.list.map(item => (item.raw ?? item) as RemoteRecord);
    exportNextCursor.value = data.cursor;
  } catch (error) {
    ElMessage.error("导出远程推荐数据失败");
    throw error;
  } finally {
    exportLoading.value = false;
  }
}

async function exportNextPage() {
  if (!exportNextCursor.value) {
    ElMessage.warning("暂无下一页数据");
    return;
  }
  dataForm.cursor = exportNextCursor.value;
  await exportData();
}

async function importData() {
  const body = importJson.value.trim();
  if (!body) {
    ElMessage.warning("请先填写导入 JSON");
    return;
  }
  try {
    JSON.parse(body);
  } catch {
    ElMessage.error("导入 JSON 格式不正确");
    return;
  }

  await ElMessageBox.confirm(`是否确定导入${selectedDataTypeLabel.value}到远程推荐？该操作会直接写入远程推荐引擎。`, "警告", {
    confirmButtonText: "确认",
    cancelButtonText: "取消",
    type: "warning"
  });

  importLoading.value = true;
  try {
    await defRecommendRemoteService.ImportData({
      type: dataForm.type,
      json: body
    });
    ElMessage.success("导入远程推荐数据成功");
    await exportData();
  } catch (error) {
    ElMessage.error("导入远程推荐数据失败");
    throw error;
  } finally {
    importLoading.value = false;
  }
}

function ensureRecommendConfig() {
  const config = resolveRecord(rawConfig.value);
  if (!config.recommend || typeof config.recommend !== "object") {
    config.recommend = {};
  }
  return config.recommend as RemoteConfigRecord;
}

function ensureRecordList(record: RemoteConfigRecord, key: string) {
  if (!Array.isArray(record[key])) {
    record[key] = [];
  }
  return record[key] as RemoteConfigRecord[];
}

function resolveRecord(value: unknown): RemoteConfigRecord {
  return value && typeof value === "object" && !Array.isArray(value) ? (value as RemoteConfigRecord) : {};
}

function resolveRecordList(value: unknown): RemoteConfigRecord[] {
  return Array.isArray(value) ? value.map(resolveRecord) : [];
}

function resolveStringList(value: unknown): string[] {
  return Array.isArray(value) ? value.map(item => String(item)).filter(Boolean) : [];
}

function resolveName(record: RemoteConfigRecord, fallback: string) {
  return typeof record.name === "string" && record.name ? record.name : fallback;
}

function resolveFieldLabel(key: string) {
  return nodeLabelMap[key] || key;
}

function encodeFieldProp(key: string) {
  return `field_${key.replace(/[^a-zA-Z0-9_]/g, "_")}`;
}

function getValueByPath(source: RemoteConfigRecord, path: Array<string | number>) {
  return path.reduce<unknown>((value, key) => resolveRecord(value)[key], source);
}

function setValueByPath(source: RemoteConfigRecord, path: Array<string | number>, value: unknown) {
  let current: RemoteConfigRecord = source;
  path.slice(0, -1).forEach(key => {
    current = resolveRecord(current[key]);
  });
  current[path[path.length - 1]] = value;
}

function cloneValue<T>(value: T): T {
  return JSON.parse(JSON.stringify(value ?? {}));
}

onMounted(() => {
  loadFlow();
});
</script>

<style scoped lang="scss">
.remote-flow-page {
  display: flex;
  flex-direction: column;
  min-height: calc(100vh - 112px);
  background: var(--admin-page-card-bg);
}

.recflow-palette {
  position: sticky;
  top: 0;
  z-index: 3;
  display: flex;
  gap: 12px;
  align-items: center;
  padding: 18px 8px 36px;
  overflow-x: auto;
  border: 0;
  border-radius: 0;
  background: linear-gradient(180deg, rgb(255 255 255 / 96%), rgb(248 250 252 / 96%));
  box-shadow: 0 8px 24px rgb(15 23 42 / 8%);

  &__item {
    display: inline-flex;
    flex: 0 0 auto;
    gap: 9px;
    align-items: center;
    height: 54px;
    padding: 0 13px;
    border: 1px solid #d8e0eb;
    border-radius: 6px;
    background: #ffffff;
    color: #4b5563;
    cursor: grab;
    box-shadow: 0 1px 2px rgb(15 23 42 / 4%);

    &:active {
      cursor: grabbing;
    }

    span {
      color: #1677ff;
      font-size: 16px;
      line-height: 1;
    }

    strong {
      font-size: 20px;
      font-weight: 500;
      white-space: nowrap;
    }
  }

  &__actions {
    display: flex;
    flex: 1 0 auto;
    gap: 8px;
    justify-content: flex-end;
    padding-left: 24px;
  }
}

.recflow-board {
  position: relative;
  min-height: 900px;
  margin-bottom: 16px;
  overflow: auto;
  border: 0;
  border-radius: 0;
  background-color: #ffffff;
  background-image: radial-gradient(#a8abb2 1.4px, transparent 1.4px);
  background-position: 0 0;
  background-size: 16px 16px;
  box-shadow: var(--admin-page-shadow);
}

.remote-flow-debug {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
  margin-bottom: 16px;
}

.remote-flow-card {
  margin-bottom: 16px;
  border-color: var(--admin-page-card-border);
  background: var(--admin-page-card-bg);
  box-shadow: var(--admin-page-shadow);
}

.remote-flow-card__header {
  display: flex;
  gap: 8px;
  align-items: baseline;
  justify-content: space-between;

  strong {
    color: var(--admin-page-text-primary);
  }

  span {
    color: var(--admin-page-text-secondary);
    font-size: 13px;
  }
}

:deep(.remote-flow-form-json .el-textarea__inner),
:deep(.el-textarea__inner) {
  font-family: var(
    --admin-page-font-mono,
    ui-monospace,
    SFMono-Regular,
    Menlo,
    Monaco,
    Consolas,
    "Liberation Mono",
    "Courier New",
    monospace
  );
}

@media (max-width: 760px) {
  .recflow-palette {
    align-items: flex-start;
    flex-wrap: wrap;

    &__actions {
      justify-content: flex-start;
      padding-left: 0;
    }
  }

  .remote-flow-debug {
    grid-template-columns: 1fr;
  }
}
</style>
