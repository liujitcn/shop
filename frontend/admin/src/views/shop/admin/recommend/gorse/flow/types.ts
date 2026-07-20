import type LogicFlow from "@logicflow/core";

/** 流程节点属性，保留 Gorse 配置中的动态字段。 */
export type FlowProperties = Record<string, unknown>;

/** 节点表单状态。 */
export interface NodeFormState {
  /** 节点编号。 */
  id: string;
  /** 节点类型。 */
  type: string;
  /** 节点展示文本。 */
  text: string;
  /** 可编辑节点名称。 */
  name: string;
  /** 节点属性表单数据。 */
  properties: FlowProperties;
}

/** 组件面板节点配置。 */
export interface PaletteNode {
  /** 节点类型。 */
  type: string;
  /** 展示图标。 */
  icon: string;
  /** 展示名称。 */
  label: string;
}

/** 推荐流程图节点。 */
export type FlowNodeConfig = LogicFlow.NodeConfig<FlowProperties>;

/** 推荐流程图连线。 */
export type FlowEdgeConfig = LogicFlow.EdgeConfig<FlowProperties>;
