<template>
  <div v-loading="loading" class="table-box gorse-flow-page">
    <FlowToolbar
      :palette-nodes="paletteNodes"
      @palette-drag-start="handlePaletteDragStart"
      @clear="clearFlow"
      @save="saveConfig"
    />

    <FlowCanvas @ready="handleCanvasReady" @drop="handleCanvasDrop" />

    <NodeEditDialog ref="nodeEditDialogRef" v-model="nodeDialogVisible" :node-form="nodeForm" @submit="updateNode" />
  </div>
</template>

<script setup lang="ts">
import LogicFlow, { BezierEdge, BezierEdgeModel, HtmlNode, HtmlNodeModel } from "@logicflow/core";
import "@logicflow/core/dist/index.css";
import dagre from "dagre";
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref } from "vue";
import { ElMessage } from "element-plus";
import FlowCanvas from "./components/FlowCanvas.vue";
import FlowToolbar from "./components/FlowToolbar.vue";
import NodeEditDialog from "./components/NodeEditDialog.vue";
import {
  fixedNodeTypes,
  nodeTypeLabelMap,
  paletteNodes,
  recommenderNameLabelMap,
  recommenderNodeTypes,
  singletonNodeTypes
} from "./constants";
import type { FlowEdgeConfig, FlowNodeConfig, FlowProperties, NodeFormState } from "./types";
import { defRecommendGorseService } from "@/api/admin/recommend_gorse";
import type { ConfigResponse } from "@/rpc/admin/v1/recommend_gorse";

defineOptions({
  name: "RecommendGorseFlow"
});

const loading = ref(false);
const canvasRef = ref<HTMLDivElement>();
const logicFlow = ref<LogicFlow>();
const config = ref<ConfigResponse>();
const nodeDialogVisible = ref(false);
const nodeEditDialogRef = ref<InstanceType<typeof NodeEditDialog>>();
const nodeForm = reactive<NodeFormState>({
  id: "",
  type: "",
  text: "",
  name: "",
  properties: {}
});

const canEditNodeName = computed(() => !fixedNodeTypes.has(nodeForm.type) && !nodeForm.properties.fixedName);

class DashedEdgeModel extends BezierEdgeModel {
  /** 设置兜底推荐连线为虚线。 */
  getEdgeStyle() {
    const style = super.getEdgeStyle();
    style.strokeDasharray = "6 4";
    style.stroke = resolveThemePrimaryColor();
    return style;
  }
}

class FlowNodeModel extends HtmlNodeModel<FlowProperties> {
  /** 设置 HTML 节点尺寸与固定节点删除策略。 */
  setAttributes() {
    this.width = resolveNodeWidth(String(this.type));
    this.height = 50;
    this.text.editable = false;
    // 核心节点由配置固定生成，禁止直接从画布删除。
    if (fixedNodeTypes.has(String(this.type))) {
      this.deletable = false;
    }
  }

  /** 隐藏 LogicFlow 默认文本，统一使用自定义 HTML 内容。 */
  getTextStyle() {
    const style = super.getTextStyle();
    style.display = "none";
    return style;
  }
}

class FlowNode extends HtmlNode {
  /** 渲染自定义推荐编排节点。 */
  setHtml(rootEl: SVGForeignObjectElement) {
    rootEl.innerHTML = "";
    const model = this.props.model;
    const properties = toRecord(model.properties);
    const nodeText = formatNodeText(String(model.type), model.text, properties);
    const card = document.createElement("div");
    card.className = "gorse-flow-node-card";
    card.style.backgroundColor = "transparent";
    card.style.border = "0";
    card.style.boxShadow = "none";
    card.style.boxSizing = "border-box";
    card.style.minWidth = "0";
    card.style.padding = "0";
    card.style.backgroundClip = "border-box";

    const body = document.createElement("div");
    body.className = "gorse-flow-node-body";
    body.style.alignItems = "center";
    body.style.backgroundColor = "var(--admin-page-card-bg)";
    body.style.border = "1px solid var(--el-color-primary)";
    body.style.borderRadius = "4px";
    body.style.boxShadow = "0 0.125rem 0.25rem rgb(0 0 0 / 7.5%)";
    body.style.boxSizing = "border-box";
    body.style.display = "flex";
    body.style.height = "50px";
    body.style.overflow = "hidden";
    body.style.padding = "8px";
    body.style.width = `${resolveNodeWidth(String(model.type))}px`;

    const icon = document.createElement("i");
    icon.className = "material-icons";
    icon.style.color = "var(--el-color-primary)";
    icon.style.fontSize = "20px";
    icon.style.lineHeight = "1";
    icon.style.marginRight = "8px";
    icon.textContent = resolveNodeIcon(String(model.type));

    const title = document.createElement("span");
    title.className = "gorse-flow-node-title";
    title.style.color = "var(--admin-page-text-primary)";
    title.style.fontSize = "14px";
    title.style.fontWeight = "700";
    title.style.overflow = "hidden";
    title.style.textOverflow = "ellipsis";
    title.style.whiteSpace = "nowrap";
    title.textContent = nodeText;

    body.appendChild(icon);
    body.appendChild(title);
    card.appendChild(body);
    rootEl.appendChild(card);
  }
}

onMounted(async () => {
  await nextTick();
  initLogicFlow();
  await loadConfig();
});

onBeforeUnmount(() => {
  logicFlow.value?.off("node:dbclick", handleNodeDoubleClick);
  logicFlow.value?.off("edge:add,connection:add,edge:exchange-node", handleEdgeChange);
  logicFlow.value?.keyboard.off(["backspace", "delete"]);
  logicFlow.value = undefined;
});

/** 初始化 LogicFlow 画布与事件。 */
function initLogicFlow() {
  if (!canvasRef.value || logicFlow.value) return;

  const lf = new LogicFlow({
    container: canvasRef.value,
    grid: true,
    nodeTextEdit: false,
    edgeTextEdit: false,
    edgeType: "bezier",
    keyboard: { enabled: true },
    style: buildLogicFlowTheme()
  });

  lf.register({ type: "dashed-edge", view: BezierEdge, model: DashedEdgeModel });
  Object.keys(nodeTypeLabelMap).forEach(type => {
    lf.register({ type, view: FlowNode, model: FlowNodeModel });
  });
  lf.on("node:dbclick", handleNodeDoubleClick);
  lf.on("edge:add,connection:add,edge:exchange-node", handleEdgeChange);
  lf.keyboard.on(["backspace", "delete"], handleDeleteSelected);
  logicFlow.value = lf;
}

/** 接收画布组件挂载后的容器元素，用于初始化 LogicFlow。 */
function handleCanvasReady(element: HTMLDivElement) {
  canvasRef.value = element;
  initLogicFlow();
  // 若配置先于画布容器完成加载，则容器就绪后立即补一次渲染。
  if (config.value) renderGraph();
}

/** 生成 LogicFlow 主题，连线、箭头与拖拽辅助线统一使用当前页面主色。 */
function buildLogicFlowTheme(): Partial<LogicFlow.Theme> {
  const primaryColor = resolveThemePrimaryColor();
  return {
    baseEdge: {
      stroke: primaryColor,
      strokeWidth: 1
    },
    bezier: {
      stroke: primaryColor,
      strokeWidth: 1
    },
    anchorLine: {
      stroke: primaryColor,
      strokeWidth: 1
    },
    arrow: {
      offset: 10,
      stroke: primaryColor,
      verticalLength: 5
    },
    edgeOutline: {
      hover: {
        stroke: primaryColor
      }
    },
    edgeAnimation: {
      stroke: primaryColor
    }
  };
}

/** 读取 Element Plus 当前主题主色，避免画布连线与节点边框写死为蓝色。 */
function resolveThemePrimaryColor() {
  const fallbackColor = "#409eff";
  const styleSources = [canvasRef.value, document.documentElement, document.body].filter(Boolean) as HTMLElement[];
  for (const source of styleSources) {
    const primaryColor = getComputedStyle(source).getPropertyValue("--el-color-primary").trim();
    if (primaryColor) return primaryColor;
  }
  return fallbackColor;
}

/** 加载 Gorse 推荐配置并刷新画布。 */
async function loadConfig() {
  loading.value = true;
  try {
    await reloadConfig();
  } finally {
    loading.value = false;
  }
}

/** 重新读取Gorse 推荐配置并刷新画布。 */
async function reloadConfig() {
  const data = await defRecommendGorseService.GetConfig({});
  config.value = normalizeConfigResponse(data);
  renderGraph();
}

/** 根据当前配置重新渲染流程图。 */
function renderGraph() {
  if (!logicFlow.value || !config.value) return;

  const graphData = buildGraphData(config.value);
  logicFlow.value.render(graphData);
  centerGraph();
}

/** 将画布内容适配到可视区域，避免管理端侧栏压缩后左右节点被裁切。 */
function centerGraph() {
  logicFlow.value?.fitView(60, 40);
}

/** 保存当前流程图到Gorse 推荐配置。 */
async function saveConfig() {
  if (!config.value) return;

  const graphData = getGraphData();
  if (!isGraphReady(graphData)) {
    ElMessage.warning("画布尚未加载完成，请稍后再保存");
    return;
  }

  syncGraphToConfig(graphData);
  loading.value = true;
  try {
    await defRecommendGorseService.SaveConfig({ config: config.value });
    // 保存接口返回体可能不是最终完整配置，保存成功后统一重新拉取一次，避免画布被部分响应清空。
    await reloadConfig();
    ElMessage.success("配置保存成功");
  } finally {
    loading.value = false;
  }
}

/** 清空Gorse 推荐配置并重新加载默认配置。 */
async function clearFlow() {
  loading.value = true;
  try {
    await defRecommendGorseService.ResetConfig({});
    await loadConfig();
    ElMessage.success("配置清空成功");
  } finally {
    loading.value = false;
  }
}

/** 处理组件面板拖拽。 */
function handlePaletteDragStart(event: DragEvent, type: string) {
  event.dataTransfer?.setData("type", type);
}

/** 将组件面板节点拖入画布。 */
function handleCanvasDrop(event: DragEvent) {
  const type = event.dataTransfer?.getData("type") ?? "";
  if (!type) return;

  const point = logicFlow.value?.getPointByClient(event.clientX, event.clientY).canvasOverlayPosition;
  addPaletteNode(type, point?.x, point?.y);
}

/** 双击组件面板时快速添加节点。 */
function addPaletteNode(type: string, x = 460, y = 260) {
  if (!logicFlow.value) return;
  if (singletonNodeTypes.has(type) && hasNodeType(type)) {
    ElMessage.warning(`${nodeTypeLabelMap[type]}节点已存在`);
    return;
  }

  const node = buildDefaultNode(type, x, y);
  logicFlow.value.addNode(node);
  addDefaultEdges(node);
}

/** 双击节点后打开编辑弹窗。 */
function handleNodeDoubleClick({ data }: { data: LogicFlow.NodeData }) {
  const properties = toRecord(data.properties);
  // 只读节点用于展示固定推荐来源，不允许编辑属性。
  if (properties.readOnly) return;

  const text = readTextValue(data.text) || formatNodeText(data.type, data.text, properties);
  const nextProperties = cloneValue(properties);
  if (data.type === "ranker" || data.type === "fallback") {
    nextProperties.recommenders = mergeOrderedRecommenders(
      readStringList(nextProperties.recommenders),
      getConnectedRecommenders(data.id)
    );
  }
  nodeForm.id = data.id;
  nodeForm.type = data.type;
  nodeForm.text = text;
  nodeForm.name = String(nextProperties.name ?? text);
  nodeForm.properties = normalizeNodeFormProperties(data.type, nextProperties);
  nodeDialogVisible.value = true;
  nextTick(() => {
    nodeEditDialogRef.value?.clearValidate();
  });
}

/** 按当前 ProForm 表单内容更新画布节点。 */
function updateNode() {
  const lf = logicFlow.value;
  if (!lf) return;

  const properties = cloneValue(nodeForm.properties);

  let text = nodeForm.text;
  if (canEditNodeName.value) {
    const name = nodeForm.name.trim();
    // 可命名推荐器需要将展示名称同步到属性 name，保存配置时会以该字段生成推荐器路径。
    if (name) {
      properties.name = name;
      text = name;
    }
  }

  lf.updateText(nodeForm.id, text);
  lf.setProperties(nodeForm.id, properties);
  nodeDialogVisible.value = false;
}

/** 补齐节点表单所需的嵌套对象和默认字段，确保 ProForm 可直接双向绑定。 */
function normalizeNodeFormProperties(type: string, properties: FlowProperties) {
  const nextProperties = cloneValue(properties);
  if (type === "data-source") {
    nextProperties.positive_feedback_types = readStringList(nextProperties.positive_feedback_types);
    nextProperties.read_feedback_types = readStringList(nextProperties.read_feedback_types);
    nextProperties.negative_feedback_types = readStringList(nextProperties.negative_feedback_types);
    nextProperties.positive_feedback_ttl = toNumberValue(nextProperties.positive_feedback_ttl, 0);
    nextProperties.item_ttl = toNumberValue(nextProperties.item_ttl, 0);
  }
  if (type === "recommend") {
    nextProperties.cache_size = toNumberValue(nextProperties.cache_size, 0);
    nextProperties.active_user_ttl = toNumberValue(nextProperties.active_user_ttl, 0);
    nextProperties.context_size = toNumberValue(nextProperties.context_size, 0);
    nextProperties.replacement = {
      enable_replacement: false,
      positive_replacement_decay: 0,
      read_replacement_decay: 0,
      ...toRecord(nextProperties.replacement)
    };
  }
  if (type === "collaborative" || type === "ranker") {
    nextProperties.fit_epoch = toNumberValue(nextProperties.fit_epoch, 1);
    nextProperties.optimize_trials = toNumberValue(nextProperties.optimize_trials, 1);
    nextProperties.early_stopping = {
      patience: 0,
      ...toRecord(nextProperties.early_stopping)
    };
  }
  if (type === "ranker" || type === "fallback") {
    nextProperties.recommenders = readStringList(nextProperties.recommenders);
  }
  return nextProperties;
}

/** 处理新增或变更连线后的业务限制。 */
function handleEdgeChange({ data }: { data: LogicFlow.EdgeData }) {
  const lf = logicFlow.value;
  if (!lf) return;

  const sourceNode = lf.getNodeModelById(data.sourceNodeId);
  const targetNode = lf.getNodeModelById(data.targetNodeId);
  if (!sourceNode || !targetNode) return;

  if (String(targetNode.type) === "recommend" && recommenderNodeTypes.has(String(sourceNode.type))) {
    const hasRanker = getGraphData().nodes.some(node => node.type === "ranker");
    // 启用排序器后，候选推荐器必须先连到排序器，不能直接连到最终推荐节点。
    if (hasRanker) {
      lf.deleteEdge(data.id);
      ElMessage.warning("已启用排序器：候选推荐器必须连接到排序器，不能直接连接到推荐结果。");
      return;
    }

    const directEdges = getGraphData().edges.filter(edge => {
      if (edge.id === data.id) return false;
      const edgeSourceNode = lf.getNodeModelById(edge.sourceNodeId);
      return edge.targetNodeId === "recommend" && edgeSourceNode && recommenderNodeTypes.has(String(edgeSourceNode.type));
    });
    // 未启用排序器时，只允许一个候选推荐器直接连接到最终推荐节点。
    if (directEdges.length >= 1) {
      lf.deleteEdge(data.id);
      ElMessage.warning("未启用排序器：只允许一个候选推荐器直接连接到推荐结果。");
      return;
    }
  }

  if (data.targetNodeId === "fallback" && data.type !== "dashed-edge") {
    lf.changeEdgeType(data.id, "dashed-edge");
  } else if (data.targetNodeId !== "fallback" && data.type === "dashed-edge") {
    lf.changeEdgeType(data.id, "bezier");
  }
}

/** 处理删除快捷键，保持与 Gorse RecFlow 一致的固定边保护和 Ranker 删除回连行为。 */
function handleDeleteSelected(event?: KeyboardEvent) {
  event?.preventDefault();
  const lf = logicFlow.value;
  if (!lf) return;

  const { nodes, edges } = lf.getSelectElements(true);
  const deletedNodeIds = new Set<string>();
  const deletedRankerIds = new Set<string>();
  const graphData = getGraphData();
  const rankerIncomingNodeIds = graphData.edges.filter(edge => edge.targetNodeId === "ranker").map(edge => edge.sourceNodeId);

  nodes.forEach(node => {
    const nodeModel = lf.getNodeModelById(node.id);
    // 固定核心节点由配置生成，不允许通过键盘删除。
    if (!nodeModel || nodeModel.deletable === false) return;

    if (String(nodeModel.type) === "ranker") {
      deletedRankerIds.add(node.id);
    }
    lf.deleteNode(node.id);
    deletedNodeIds.add(node.id);
  });

  edges.forEach(edge => {
    const sourceNodeModel = lf.getNodeModelById(edge.sourceNodeId);
    const targetNodeModel = lf.getNodeModelById(edge.targetNodeId);
    // Data Source 到候选推荐器的边由节点存在性维护，不允许单独删边导致画布与配置不一致。
    if (sourceNodeModel && String(sourceNodeModel.type) === "data-source" && !deletedNodeIds.has(edge.sourceNodeId)) return;
    // 指向 Recommend 的核心结果边需要保留，避免误删后保存出不可用链路。
    if (targetNodeModel && String(targetNodeModel.type) === "recommend") return;
    lf.deleteEdge(edge.id);
  });

  if (deletedRankerIds.size === 0) return;

  const existingRecommendEdges = getGraphData().edges.filter(edge => {
    const sourceNode = lf.getNodeModelById(edge.sourceNodeId);
    return edge.targetNodeId === "recommend" && sourceNode && recommenderNodeTypes.has(String(sourceNode.type));
  });
  // 删除 Ranker 后，沿用 Gorse RecFlow 行为：挑选原先进入 Ranker 的第一个候选推荐器直连 Recommend。
  if (existingRecommendEdges.length > 0) return;

  const fallbackCandidate = rankerIncomingNodeIds.find(sourceNodeId => !deletedNodeIds.has(sourceNodeId));
  if (fallbackCandidate) {
    lf.addEdge(createEdge(fallbackCandidate, "recommend"));
  }
}

/** 根据配置生成流程图数据。 */
function buildGraphData(sourceConfig: ConfigResponse): LogicFlow.GraphConfigData {
  const recommend = toRecord(sourceConfig.recommend);
  const nodes: FlowNodeConfig[] = [];
  const edges: FlowEdgeConfig[] = [];
  const sourceNodeMap: Record<string, string> = {};
  const rankerConfig = readConfigValue(recommend, "ranker");
  const collaborativeConfig = readConfigValue(recommend, "collaborative");
  const ranker = toRecord(rankerConfig);
  const collaborative = toRecord(collaborativeConfig);
  const fallback = readRecord(recommend, "fallback");
  const rankerEnabled = Boolean(rankerConfig) && ranker.type !== "none";
  const collaborativeEnabled = Boolean(collaborativeConfig) && collaborative.type !== "none";
  const fallbackRecommenders = readStringList(fallback.recommenders);
  const fallbackEnabled = fallbackRecommenders.length > 0;

  nodes.push(
    createNode("data-source", nodeTypeLabelMap["data-source"], {
      fixedName: true,
      ...readRecord(recommend, ["data_source", "dataSource"])
    })
  );
  if (rankerEnabled) nodes.push(createNode("ranker", nodeTypeLabelMap.ranker, { fixedName: true, ...ranker }));
  if (fallbackEnabled) nodes.push(createNode("fallback", nodeTypeLabelMap.fallback, { fixedName: true, ...fallback }));
  nodes.push(
    createNode("recommend", nodeTypeLabelMap.recommend, {
      fixedName: true,
      cache_size: readConfigValue(recommend, ["cache_size", "cacheSize"]),
      cache_expire: readConfigValue(recommend, ["cache_expire", "cacheExpire"]),
      active_user_ttl: readConfigValue(recommend, ["active_user_ttl", "activeUserTtl"]),
      context_size: readConfigValue(recommend, ["context_size", "contextSize"]),
      replacement: readRecord(recommend, "replacement")
    })
  );

  if (rankerEnabled) edges.push(createEdge("ranker", "recommend"));
  if (fallbackEnabled) edges.push(createEdge("fallback", "recommend"));

  nodes.push(createNode("latest", nodeTypeLabelMap.latest, { readOnly: true }));
  sourceNodeMap.latest = "latest";
  edges.push(createEdge("data-source", "latest"));

  if (collaborativeEnabled) {
    nodes.push(createNode("collaborative", nodeTypeLabelMap.collaborative, { fixedName: true, ...collaborative }));
    sourceNodeMap.collaborative = "collaborative";
    edges.push(createEdge("data-source", "collaborative"));
  }

  appendRecommendNodes(nodes, edges, sourceNodeMap, "non-personalized", readRecordList(recommend, "non-personalized"));
  appendRecommendNodes(nodes, edges, sourceNodeMap, "user-to-user", readRecordList(recommend, "user-to-user"));
  appendRecommendNodes(nodes, edges, sourceNodeMap, "item-to-item", readRecordList(recommend, "item-to-item"));
  appendRecommendNodes(nodes, edges, sourceNodeMap, "external", readRecordList(recommend, "external"));

  const rankerRecommenders = readStringList(ranker.recommenders);
  if (rankerEnabled) {
    rankerRecommenders.forEach(recommender => addCandidateEdge(edges, sourceNodeMap[recommender], "ranker"));
  } else {
    const directNodeId =
      rankerRecommenders.map(recommender => sourceNodeMap[recommender]).find(Boolean) ?? Object.values(sourceNodeMap)[0];
    addCandidateEdge(edges, directNodeId, "recommend");
  }

  if (fallbackEnabled) {
    fallbackRecommenders.forEach(recommender => addCandidateEdge(edges, sourceNodeMap[recommender], "fallback", "dashed-edge"));
  }

  return layoutGraph(nodes, edges);
}

/** 将推荐器数组追加为候选来源节点。 */
function appendRecommendNodes(
  nodes: FlowNodeConfig[],
  edges: FlowEdgeConfig[],
  sourceNodeMap: Record<string, string>,
  type: string,
  items: FlowProperties[]
) {
  items.forEach(item => {
    const name = String(item.name ?? "").trim();
    if (!name) return;

    const id = buildNodeId(type, name);
    nodes.push(createNode(type, name, item, id));
    sourceNodeMap[`${type}/${name}`] = id;
    edges.push(createEdge("data-source", id));
  });
}

/** 将图配置转换回Gorse 推荐配置。 */
function syncGraphToConfig(data: LogicFlow.GraphData) {
  if (!config.value) return;

  const currentConfig = normalizeConfigRecord(config.value);
  const currentRecommend = toRecord(currentConfig.recommend);
  const newRecommend: Record<string, unknown> = {
    ...currentRecommend,
    data_source: {},
    collaborative: null,
    "non-personalized": [],
    "user-to-user": [],
    "item-to-item": [],
    external: [],
    ranker: { ...toRecord(currentRecommend.ranker), recommenders: [] },
    fallback: { ...toRecord(currentRecommend.fallback), recommenders: [] }
  };

  let hasCollaborativeNode = false;
  data.nodes.forEach(node => {
    const properties = cleanupNodeProperties(node.properties);
    if (node.type === "data-source") newRecommend.data_source = properties;
    if (node.type === "collaborative") {
      hasCollaborativeNode = true;
      newRecommend.collaborative = properties;
    }
    if (node.type === "non-personalized") pushRecommendConfig(newRecommend, "non-personalized", properties);
    if (node.type === "user-to-user") pushRecommendConfig(newRecommend, "user-to-user", properties);
    if (node.type === "item-to-item") pushRecommendConfig(newRecommend, "item-to-item", properties);
    if (node.type === "external") pushRecommendConfig(newRecommend, "external", properties);
    if (node.type === "recommend") {
      ["cache_size", "cache_expire", "active_user_ttl", "context_size", "replacement"].forEach(key => {
        // 推荐服务节点仅同步推荐服务自身字段，避免把布局字段写入配置。
        if (properties[key] !== undefined) newRecommend[key] = properties[key];
      });
    }
    if (node.type === "ranker") newRecommend.ranker = { ...toRecord(newRecommend.ranker), ...properties, recommenders: [] };
    if (node.type === "fallback") newRecommend.fallback = { ...toRecord(newRecommend.fallback), ...properties, recommenders: [] };
  });

  const connections = collectTargetConnections(data);
  const hasRankerNode = data.nodes.some(node => node.type === "ranker");
  if (!hasRankerNode) {
    newRecommend.ranker = {
      ...toRecord(newRecommend.ranker),
      type: "none",
      recommenders: uniqueStrings(connections.recommend).slice(0, 1)
    };
  } else {
    syncOrderedRecommenders(newRecommend, "ranker", connections.ranker, data.nodes);
  }

  syncOrderedRecommenders(newRecommend, "fallback", connections.fallback, data.nodes);
  if (!hasCollaborativeNode) newRecommend.collaborative = { ...toRecord(currentRecommend.collaborative), type: "none" };

  currentConfig.recommend = newRecommend;
  config.value = currentConfig as unknown as ConfigResponse;
}

/** 判断画布核心节点是否已经完成渲染，避免空图直接保存覆盖Gorse 配置。 */
function isGraphReady(data: LogicFlow.GraphData) {
  const nodeTypes = new Set(data.nodes.map(node => node.type));
  return nodeTypes.has("data-source") && nodeTypes.has("recommend");
}

/** 将Gorse 配置响应统一归一为 Proto json_name 字段，避免保存时同一字段出现横线与下划线两份。 */
function normalizeConfigResponse(sourceConfig: ConfigResponse): ConfigResponse {
  return normalizeConfigRecord(sourceConfig) as unknown as ConfigResponse;
}

/** 克隆配置并归一推荐器数组字段名称。 */
function normalizeConfigRecord(sourceConfig: ConfigResponse) {
  const currentConfig = cloneValue(sourceConfig) as unknown as Record<string, unknown>;
  const recommend = toRecord(currentConfig.recommend);
  normalizeRecommendArrayKey(recommend, "non-personalized", ["non_personalized", "nonPersonalized"]);
  normalizeRecommendArrayKey(recommend, "item-to-item", ["item_to_item", "itemToItem"]);
  normalizeRecommendArrayKey(recommend, "user-to-user", ["user_to_user", "userToUser"]);
  currentConfig.recommend = recommend;
  return currentConfig;
}

/** 将接口传输中可能出现的字段别名合并到 proto json_name 字段，并删除别名字段。 */
function normalizeRecommendArrayKey(record: FlowProperties, jsonName: string, aliasNames: string[]) {
  const mergedList = readRecordList(record, jsonName);
  aliasNames.forEach(aliasName => {
    readRecordList(record, aliasName).forEach(item => mergedList.push(item));
    delete record[aliasName];
  });
  record[jsonName] = uniqueRecommendRecords(mergedList);
}

/** 按推荐器名称去重，名称缺失时退回使用完整 JSON 内容去重。 */
function uniqueRecommendRecords(items: FlowProperties[]) {
  const seenKeys = new Set<string>();
  const result: FlowProperties[] = [];
  items.forEach(item => {
    const key = String(item.name ?? JSON.stringify(item));
    if (seenKeys.has(key)) return;
    seenKeys.add(key);
    result.push(item);
  });
  return result;
}

/** 收集指向 ranker、fallback、recommend 的推荐器连接。 */
function collectTargetConnections(data: LogicFlow.GraphData) {
  const nodeMap = new Map(data.nodes.map(node => [node.id, node]));
  const connections = {
    ranker: [] as string[],
    fallback: [] as string[],
    recommend: [] as string[]
  };

  data.edges.forEach(edge => {
    const sourceNode = nodeMap.get(edge.sourceNodeId);
    const targetNode = nodeMap.get(edge.targetNodeId);
    if (!sourceNode || !targetNode) return;

    const recommender = getRecommenderName(sourceNode);
    if (!recommender) return;
    if (targetNode.type === "ranker") connections.ranker.push(recommender);
    if (targetNode.type === "fallback") connections.fallback.push(recommender);
    if (targetNode.type === "recommend") connections.recommend.push(recommender);
  });
  return connections;
}

/** 按节点保存顺序同步推荐器列表。 */
function syncOrderedRecommenders(
  newRecommend: Record<string, unknown>,
  type: "ranker" | "fallback",
  connected: string[],
  nodes: LogicFlow.NodeData[]
) {
  const node = nodes.find(item => item.type === type);
  const properties = toRecord(node?.properties);
  const previousOrder = readStringList(properties.recommenders);
  const ordered = previousOrder.filter(item => connected.includes(item));
  connected.forEach(item => {
    // 新增连线追加到末尾，保留用户在表单中维护的已有顺序。
    if (!ordered.includes(item)) ordered.push(item);
  });
  newRecommend[type] = { ...toRecord(newRecommend[type]), recommenders: uniqueStrings(ordered) };
}

/** 根据类型构造默认节点。 */
function buildDefaultNode(type: string, x: number, y: number): FlowNodeConfig {
  const name = resolveUniqueNodeName(type);
  const properties = buildDefaultProperties(type, name);
  const fixedText = ["latest", "collaborative", "ranker", "fallback"].includes(type);
  return {
    id: buildNodeId(type, name),
    type,
    x,
    y,
    text: fixedText ? nodeTypeLabelMap[type] : name,
    properties
  };
}

/** 根据节点类型生成默认属性。 */
function buildDefaultProperties(type: string, name: string): FlowProperties {
  if (type === "latest") return { readOnly: true };
  if (type === "collaborative")
    return {
      fixedName: true,
      type: "mf",
      fit_period: "60m",
      fit_epoch: 10,
      optimize_period: "60m",
      optimize_trials: 10,
      early_stopping: { patience: 10 }
    };
  if (type === "ranker")
    return {
      fixedName: true,
      type: "fm",
      query_template: "",
      document_template: "",
      cache_expire: "120h",
      recommenders: [
        "latest",
        "collaborative",
        "non-personalized/most_starred_weekly",
        "item-to-item/neighbors",
        "user-to-user/neighbors"
      ],
      fit_period: "60m",
      fit_epoch: 100,
      optimize_period: "360m",
      optimize_trials: 10
    };
  if (type === "fallback") return { fixedName: true, recommenders: [] };
  if (type === "non-personalized") return { name, score: "", filter: "" };
  if (type === "user-to-user") return { name, type: "items" };
  if (type === "item-to-item") return { name, type: "embedding", column: "item.Labels.embedding" };
  if (type === "external") return { name, script: "" };
  return { name };
}

/** 为新增节点补充默认连线。 */
function addDefaultEdges(node: FlowNodeConfig) {
  const lf = logicFlow.value;
  if (!lf || !node.id) return;

  if (recommenderNodeTypes.has(node.type)) lf.addEdge(createEdge("data-source", node.id));
  if (node.type === "ranker") {
    lf.addEdge(createEdge(node.id, "recommend"));
    getGraphData().edges.forEach(edge => {
      const sourceNode = getGraphData().nodes.find(item => item.id === edge.sourceNodeId);
      const targetNode = getGraphData().nodes.find(item => item.id === edge.targetNodeId);
      // 新增排序器后，沿用 Gorse RecFlow 行为：候选推荐器原本直连 Recommend 的边自动改连 Ranker。
      if (sourceNode && targetNode?.type === "recommend" && recommenderNodeTypes.has(sourceNode.type)) {
        lf.deleteEdge(edge.id);
        lf.addEdge(createEdge(edge.sourceNodeId, String(node.id)));
      }
    });
  }
  if (node.type === "fallback") lf.addEdge(createEdge(node.id, "recommend"));
}

/** 创建流程图节点。 */
function createNode(type: string, text: string, properties: FlowProperties, id = type): FlowNodeConfig {
  return { id, type, x: 0, y: 0, text, properties: cloneValue(properties) };
}

/** 创建流程图连线。 */
function createEdge(sourceNodeId: string, targetNodeId: string, type = "bezier"): FlowEdgeConfig {
  return { sourceNodeId, targetNodeId, type };
}

/** 添加候选推荐器连线，并避免重复边。 */
function addCandidateEdge(edges: FlowEdgeConfig[], sourceNodeId: string | undefined, targetNodeId: string, type = "bezier") {
  if (!sourceNodeId) return;
  const exists = edges.some(edge => edge.sourceNodeId === sourceNodeId && edge.targetNodeId === targetNodeId);
  if (!exists) edges.push(createEdge(sourceNodeId, targetNodeId, type));
}

/** 使用 Gorse RecFlow 一致的 dagre 左右布局。 */
function layoutGraph(nodes: FlowNodeConfig[], edges: FlowEdgeConfig[]): LogicFlow.GraphConfigData {
  const graph = new dagre.graphlib.Graph();
  graph.setGraph({ rankdir: "LR", nodesep: 20, ranksep: 100 });
  graph.setDefaultEdgeLabel(() => ({}));

  nodes.forEach(node => {
    graph.setNode(String(node.id), { width: resolveNodeWidth(node.type), height: 60 });
  });
  edges.forEach(edge => {
    graph.setEdge(edge.sourceNodeId, edge.targetNodeId);
  });
  dagre.layout(graph);

  const layoutedNodes = nodes.map(node => {
    const position = graph.node(String(node.id));
    return {
      ...node,
      x: position.x,
      y: position.y
    };
  });
  return { nodes: layoutedNodes, edges };
}

/** 从画布读取标准图数据。 */
function getGraphData(): LogicFlow.GraphData {
  const data = logicFlow.value?.getGraphData() as LogicFlow.GraphData | undefined;
  return data ?? { nodes: [], edges: [] };
}

/** 判断画布中是否存在指定类型节点。 */
function hasNodeType(type: string) {
  return getGraphData().nodes.some(node => node.type === type);
}

/** 获取连接到目标节点的推荐器名称。 */
function getConnectedRecommenders(nodeId: string) {
  const data = getGraphData();
  const nodeMap = new Map(data.nodes.map(node => [node.id, node]));
  const recommenders: string[] = [];
  data.edges.forEach(edge => {
    if (edge.targetNodeId !== nodeId) return;
    const sourceNode = nodeMap.get(edge.sourceNodeId);
    const recommender = sourceNode ? getRecommenderName(sourceNode) : "";
    if (recommender) recommenders.push(recommender);
  });
  return uniqueStrings(recommenders);
}

/** 合并已有推荐器顺序与当前连线，保留表单上移下移后的排序结果。 */
function mergeOrderedRecommenders(previousOrder: string[], connected: string[]) {
  const connectedSet = new Set(connected);
  const ordered = previousOrder.filter(item => connectedSet.has(item));
  connected.forEach(item => {
    // 新连线追加到末尾，断开的推荐器则不再保留。
    if (!ordered.includes(item)) ordered.push(item);
  });
  return uniqueStrings(ordered);
}

/** 根据推荐器节点获取 Gorse 推荐器路径。 */
function getRecommenderName(node: LogicFlow.NodeData) {
  const properties = toRecord(node.properties);
  if (node.type === "latest") return "latest";
  if (node.type === "collaborative") return "collaborative";
  if (node.type === "external") return properties.name ? `external/${String(properties.name)}` : "";
  if (["non-personalized", "user-to-user", "item-to-item"].includes(node.type)) {
    return properties.name ? `${node.type}/${String(properties.name)}` : "";
  }
  return "";
}

/** 清理节点属性中的布局或只读辅助字段。 */
function cleanupNodeProperties(properties: unknown) {
  const nextProperties = cloneValue(toRecord(properties));
  delete nextProperties.fixedName;
  delete nextProperties.readOnly;
  delete nextProperties.width;
  delete nextProperties.height;
  return nextProperties;
}

/** 推入推荐器配置数组。 */
function pushRecommendConfig(recommend: Record<string, unknown>, key: string, properties: FlowProperties) {
  const list = Array.isArray(recommend[key]) ? (recommend[key] as FlowProperties[]) : [];
  list.push(properties);
  recommend[key] = list;
}

/** 按多个可能字段名读取配置值，兼容 Gorse 原始 JSON、Proto 字段名与 lowerCamelCase 响应。 */
function readConfigValue(source: Record<string, unknown>, key: string | string[]) {
  const keys = Array.isArray(key) ? key : [key];
  for (const item of keys) {
    if (Object.prototype.hasOwnProperty.call(source, item)) {
      return source[item];
    }
  }
  return undefined;
}

/** 读取对象配置。 */
function readRecord(source: Record<string, unknown>, key: string | string[]) {
  return toRecord(readConfigValue(source, key));
}

/** 读取对象数组配置。 */
function readRecordList(source: Record<string, unknown>, key: string | string[]) {
  const value = readConfigValue(source, key);
  if (Array.isArray(value)) return value.map(record => toRecord(record));
  return [];
}

/** 将未知值转换为普通对象。 */
function toRecord(value: unknown): FlowProperties {
  return typeof value === "object" && value !== null && !Array.isArray(value) ? (value as FlowProperties) : {};
}

/** 将表单初始值转换为数字，无法转换时使用默认值。 */
function toNumberValue(value: unknown, defaultValue: number) {
  const numberValue = Number(value);
  return Number.isFinite(numberValue) ? numberValue : defaultValue;
}

/** 读取字符串数组。 */
function readStringList(value: unknown) {
  return Array.isArray(value) ? value.map(item => String(item).trim()).filter(Boolean) : [];
}

/** 字符串数组去重。 */
function uniqueStrings(items: string[]) {
  return Array.from(new Set(items.filter(Boolean)));
}

/** 深拷贝 JSON 兼容值。 */
function cloneValue<T>(value: T): T {
  return JSON.parse(JSON.stringify(value ?? {})) as T;
}

/** 读取 LogicFlow 文本值。 */
function readTextValue(text: LogicFlow.TextConfig | string | undefined) {
  if (!text) return "";
  return typeof text === "string" ? text : String(text.value ?? "");
}

/** 格式化节点展示文本。 */
function formatNodeText(type: string, text: LogicFlow.TextConfig | string | undefined, properties: FlowProperties) {
  if (properties.name) return resolveRecommenderDisplayName(String(properties.name));
  const textValue = readTextValue(text);
  return nodeTypeLabelMap[type] || textValue || type;
}

/** 将常见英文推荐器名称转换为中文展示名称，保留未知自定义名称原样显示。 */
function resolveRecommenderDisplayName(name: string) {
  return recommenderNameLabelMap[name] || name;
}

/** 计算节点宽度。 */
function resolveNodeWidth(type: string) {
  if (type === "data-source") return 150;
  if (type === "recommend") return 160;
  if (type === "ranker" || type === "fallback") return 120;
  return 270;
}

/** 计算节点图标。 */
function resolveNodeIcon(type: string) {
  const matched = paletteNodes.find(item => item.type === type);
  if (matched) return matched.icon;
  if (type === "data-source") return "storage";
  return "settings";
}

/** 生成节点 ID。 */
function buildNodeId(type: string, name: string) {
  const safeName = name.trim().replace(/[^a-zA-Z0-9_-]+/g, "-") || Date.now().toString();
  return `${type}-${safeName}`;
}

/** 生成不重复的默认节点名称。 */
function resolveUniqueNodeName(type: string) {
  const baseName = resolveDefaultNodeNamePrefix(type);
  const existingNames = new Set(
    getGraphData()
      .nodes.filter(node => node.type === type)
      .map(node => String(toRecord(node.properties).name ?? readTextValue(node.text)))
      .filter(Boolean)
  );
  let index = existingNames.size + 1;
  let name = `${baseName}_${index}`;
  while (existingNames.has(name)) {
    index += 1;
    name = `${baseName}_${index}`;
  }
  return name;
}

/** 获取与 Gorse RecFlow 一致的新建节点名称前缀。 */
function resolveDefaultNodeNamePrefix(type: string) {
  if (type === "non-personalized") return "new_non_personalized";
  if (type === "user-to-user") return "new_user_to_user";
  if (type === "item-to-item") return "new_item_to_item";
  if (type === "external") return "new_external";
  return type;
}
</script>

<style scoped lang="scss">
@import url("https://fonts.googleapis.com/icon?family=Material+Icons");

:deep(.material-icons) {
  direction: ltr;
  display: inline-block;
  font-family: "Material Icons";
  font-feature-settings: "liga";
  font-size: 24px;
  font-style: normal;
  font-weight: normal;
  letter-spacing: normal;
  line-height: 1;
  text-transform: none;
  white-space: nowrap;
  word-wrap: normal;
}

.gorse-flow-page {
  gap: 12px;
}
</style>
