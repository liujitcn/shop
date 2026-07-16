<template>
  <div v-if="message.blocks?.length" class="assistant-flow-blocks">
    <section v-for="(block, blockIndex) in message.blocks" :key="resolveBlockKey(block, blockIndex)" class="assistant-flow-block">
      <div class="assistant-flow-block__head">
        <div class="assistant-flow-block__title">{{ resolveBlockTitle(block) }}</div>
        <span class="assistant-flow-block__type">{{ block.type }}</span>
      </div>
      <p v-if="block.desc" class="assistant-flow-block__desc">{{ block.desc }}</p>

      <div v-if="resolveMetricItems(block).length" class="assistant-flow-metrics">
        <div v-for="(item, itemIndex) in resolveMetricItems(block)" :key="itemIndex" class="assistant-flow-metric">
          <span>{{ item.label }}</span>
          <strong>{{ item.value }}{{ item.unit }}</strong>
        </div>
      </div>

      <div v-for="section in resolveSections(block)" :key="section.key" class="assistant-flow-section">
        <div class="assistant-flow-section__title">{{ section.title }}</div>
        <template v-if="section.kind === 'list'">
          <div class="assistant-flow-list">
            <article v-for="(item, itemIndex) in section.items" :key="itemIndex" class="assistant-flow-list__item">
              <div class="assistant-flow-list__content">
                <div class="assistant-flow-list__title">{{ resolveItemTitle(item, itemIndex) }}</div>
                <div class="assistant-flow-list__fields">
                  <span v-for="field in resolveItemFields(item)" :key="field.key">{{ field.label }}：{{ field.value }}</span>
                </div>
              </div>
              <el-button
                v-if="item.action?.type"
                size="small"
                type="primary"
                plain
                :disabled="Boolean(item.disabled)"
                @click="emitFlowAction(item.action, resolveActionLabel(item.action, item))"
              >
                {{ resolveActionLabel(item.action, item) }}
              </el-button>
            </article>
          </div>
        </template>
        <template v-else>
          <div class="assistant-flow-fields">
            <div v-for="field in section.fields" :key="field.key" class="assistant-flow-field">
              <span>{{ field.label }}</span>
              <strong>{{ field.value }}</strong>
            </div>
          </div>
        </template>
      </div>

      <div v-if="isShipmentFormBlock(block)" class="assistant-flow-shipment">
        <div class="assistant-flow-shipment__grid">
          <label class="assistant-flow-shipment__field">
            <span>物流公司</span>
            <el-input
              :model-value="resolveShipmentFormState(block, resolveBlockKey(block, blockIndex)).name"
              size="small"
              placeholder="请输入物流公司"
              :disabled="Boolean(block.disabled)"
              @update:model-value="value => updateShipmentFormField(block, resolveBlockKey(block, blockIndex), 'name', value)"
            />
          </label>
          <label class="assistant-flow-shipment__field">
            <span>物流单号</span>
            <el-input
              :model-value="resolveShipmentFormState(block, resolveBlockKey(block, blockIndex)).no"
              size="small"
              placeholder="请输入物流单号"
              :disabled="Boolean(block.disabled)"
              @update:model-value="value => updateShipmentFormField(block, resolveBlockKey(block, blockIndex), 'no', value)"
            />
          </label>
          <label class="assistant-flow-shipment__field">
            <span>联系电话</span>
            <el-input
              :model-value="resolveShipmentFormState(block, resolveBlockKey(block, blockIndex)).contact"
              size="small"
              placeholder="请输入联系电话"
              :disabled="Boolean(block.disabled)"
              @update:model-value="value => updateShipmentFormField(block, resolveBlockKey(block, blockIndex), 'contact', value)"
            />
          </label>
        </div>
        <div class="assistant-flow-actions">
          <el-button
            v-if="resolveShipmentAction(block)"
            size="small"
            type="primary"
            :disabled="isShipmentSubmitDisabled(block, resolveBlockKey(block, blockIndex))"
            @click="emitShipmentAction(block, resolveBlockKey(block, blockIndex))"
          >
            {{ resolveShipmentActionLabel(block) }}
          </el-button>
        </div>
      </div>

      <div v-else-if="resolveActions(block).length" class="assistant-flow-actions">
        <el-button
          v-for="(action, actionIndex) in resolveActions(block)"
          :key="action.action_id || action.type || actionIndex"
          size="small"
          type="primary"
          :disabled="Boolean((action as any).disabled || block.disabled)"
          @click="emitFlowAction(action, resolveActionLabel(action, block))"
        >
          {{ resolveActionLabel(action, block) }}
        </el-button>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { reactive } from "vue";
import type { AiAssistantAction } from "@/rpc/base/v1/ai_assistant_message";
import type { AssistantFlowBlock, ChatMessageItem } from "../types";

/** 展示字段结构。 */
type DisplayField = {
  /** 原始字段名。 */
  key: string;
  /** 中文标签。 */
  label: string;
  /** 格式化后的展示值。 */
  value: string;
};

/** Flow 卡片分区结构。 */
type DisplaySection =
  | {
      /** 分区键。 */
      key: string;
      /** 分区标题。 */
      title: string;
      /** 分区类型。 */
      kind: "list";
      /** 列表数据。 */
      items: AssistantFlowBlock[];
    }
  | {
      /** 分区键。 */
      key: string;
      /** 分区标题。 */
      title: string;
      /** 分区类型。 */
      kind: "fields";
      /** 字段数据。 */
      fields: DisplayField[];
    };

/** 发货表单本地输入状态。 */
type ShipmentFormState = {
  /** 物流公司名。 */
  name: string;
  /** 物流单号。 */
  no: string;
  /** 联系方式。 */
  contact: string;
};

const props = defineProps<{
  /** 当前助手气泡，包含已解析的结构化卡片。 */
  message: ChatMessageItem;
  /** 当前会话中最新可交互 Flow 消息 ID。 */
  activeFlowMessageId: string;
}>();

const emit = defineEmits<{
  /** 提交流程动作给父级，由页面层创建消息并请求后端。 */
  "flow-action": [action: AiAssistantAction, label?: string];
}>();

/** 可按明细列表展示的卡片字段。 */
const listFieldKeys = new Set(["items", "orders", "comments", "goods", "stores", "bills", "tasks", "hot_tags", "refund"]);
/** 可按字段分组展示的卡片对象。 */
const objectFieldKeys = new Set(["order", "payment", "comment", "store", "config", "form"]);
const ignoredFieldKeys = new Set(["type", "title", "desc", "action", "actions", "disabled"]);
const shipmentFormStateMap = reactive<Record<string, ShipmentFormState>>({});
const fieldLabels: Record<string, string> = {
  label: "名称",
  value: "数值",
  unit: "单位",
  status_label: "状态",
  alert_label: "预警",
  order: "订单信息",
  payment: "支付信息",
  refund: "退款明细",
  order_no: "订单号",
  tenant_store_id: "门店 ID",
  pay_money: "支付金额",
  total_money: "订单金额",
  goods_num: "商品数",
  trade_status_label: "支付状态",
  refund_status_label: "退款状态",
  trade_no: "交易单号",
  third_order_no: "三方订单号",
  trade_type: "交易类型",
  trade_state: "交易状态",
  trade_state_desc: "交易状态说明",
  bank_type: "银行类型",
  success_time: "完成时间",
  order_id: "订单 ID",
  refund_no: "退款编号",
  reason: "退款原因",
  third_refund_no: "三方退款编号",
  channel: "退款渠道",
  create_time: "退款创建时间",
  refund_state: "退款状态",
  goods_name: "商品",
  user_name: "用户",
  goods_score: "评分",
  created_at: "创建时间",
  name: "名称",
  nick_name: "昵称",
  detail: "详情",
  remark: "备注",
  bill_date: "账单日期",
  bill_type: "账单类型",
  total_count: "本地笔数",
  total_amount: "本地金额",
  third_total_count: "三方笔数",
  third_total_amount: "三方金额",
  average_score: "平均评分",
  comment_summary: "评价摘要",
  last_modified: "最后更新",
  total: "总数"
};

/** 生成卡片稳定键。 */
function resolveBlockKey(block: AssistantFlowBlock, index: number) {
  return [block.type, block.title, index].filter(Boolean).join(":");
}

/** 生成卡片标题。 */
function resolveBlockTitle(block: AssistantFlowBlock) {
  return block.title || block.desc || resolveTypeLabel(block.type);
}

/** 生成类型兜底名称。 */
function resolveTypeLabel(type?: string) {
  return String(type || "结构化内容").replace(/_/g, " ");
}

/** 返回指标类数据。 */
function resolveMetricItems(block: AssistantFlowBlock) {
  const items = Array.isArray(block.items) ? block.items : [];
  if (!items.every(item => isRecord(item) && ("label" in item || "value" in item))) return [];
  return items.map(item => ({
    label: formatValue(item.label),
    value: formatValue(item.value),
    unit: formatValue(item.unit)
  }));
}

/** 将卡片扩展字段拆成可展示分区。 */
function resolveSections(block: AssistantFlowBlock): DisplaySection[] {
  const sections: DisplaySection[] = [];
  Object.entries(block).forEach(([key, value]) => {
    if (ignoredFieldKeys.has(key)) return;
    if (key === "items" && resolveMetricItems(block).length) return;
    if (Array.isArray(value) && listFieldKeys.has(key)) {
      const items = value.filter(isRecord) as AssistantFlowBlock[];
      if (items.length) sections.push({ key, title: resolveFieldLabel(key), kind: "list", items });
      return;
    }
    if (isRecord(value) && objectFieldKeys.has(key)) {
      const fields = resolveItemFields(value);
      if (fields.length) sections.push({ key, title: resolveFieldLabel(key), kind: "fields", fields });
      return;
    }
    if (!isRecord(value) && !Array.isArray(value) && value !== undefined && value !== "") {
      sections.push({
        key,
        title: resolveFieldLabel(key),
        kind: "fields",
        fields: [{ key, label: resolveFieldLabel(key), value: formatValue(value) }]
      });
    }
  });
  return sections;
}

/** 返回对象列表项标题。 */
function resolveItemTitle(item: AssistantFlowBlock, index: number) {
  return formatValue(item.title || item.name || item.goods_name || item.order_no || item.nick_name || item.id || `#${index + 1}`);
}

/** 返回对象列表项中的重点字段。 */
function resolveItemFields(item: Record<string, any>) {
  return Object.entries(item)
    .filter(([key, value]) => !ignoredFieldKeys.has(key) && key !== "id" && !Array.isArray(value) && !isRecord(value) && value !== undefined && value !== "")
    .slice(0, 8)
    .map(([key, value]) => ({ key, label: resolveFieldLabel(key), value: formatValue(value) }));
}

/** 返回字段中文标签。 */
function resolveFieldLabel(key: string) {
  return fieldLabels[key] || key.replace(/_/g, " ");
}

/** 返回当前块可执行动作。 */
function resolveActions(block: AssistantFlowBlock) {
  const actions = Array.isArray(block.actions) ? block.actions : [];
  return [block.action, ...actions].filter(isFlowAction);
}

/** 判断当前卡片是否为后台发货表单。 */
function isShipmentFormBlock(block: AssistantFlowBlock) {
  return block.type === "shipment_form";
}

/** 返回发货表单的确认动作。 */
function resolveShipmentAction(block: AssistantFlowBlock) {
  return resolveActions(block).find(action => action.type === "confirm_shipment");
}

/** 生成发货表单确认按钮文案。 */
function resolveShipmentActionLabel(block: AssistantFlowBlock) {
  const action = resolveShipmentAction(block);
  return action ? resolveActionLabel(action, block) : "确认发货";
}

/** 判断动作是否可作为按钮渲染。 */
function isFlowAction(action?: AiAssistantAction): action is AiAssistantAction {
  return Boolean(action?.type);
}

/** 触发流程动作。 */
function emitFlowAction(action: AiAssistantAction, label?: string) {
  if ((action as any).disabled || !isActionEnabled(action)) return;
  emit("flow-action", action, label);
}

/** 触发发货动作前把本地物流表单写入 action payload。 */
function emitShipmentAction(block: AssistantFlowBlock, blockKey: string) {
  const action = resolveShipmentAction(block);
  if (!action || isShipmentSubmitDisabled(block, blockKey)) return;
  const form = resolveShipmentFormState(block, blockKey);
  const nextAction: AiAssistantAction = {
    ...action,
    payload_json: JSON.stringify({
      ...readActionPayload(action),
      name: form.name,
      no: form.no,
      contact: form.contact
    })
  };
  emitFlowAction(nextAction, resolveActionLabel(action, block));
}

/** 返回发货表单本地状态，首次渲染时尝试使用已有物流信息回填。 */
function resolveShipmentFormState(block: AssistantFlowBlock, blockKey: string) {
  if (!shipmentFormStateMap[blockKey]) {
    const logistics = resolveShipmentLogistics(block);
    shipmentFormStateMap[blockKey] = {
      name: stringValue(logistics.name),
      no: stringValue(logistics.no),
      contact: stringValue(logistics.contact)
    };
  }
  return shipmentFormStateMap[blockKey];
}

/** 更新发货表单单字段。 */
function updateShipmentFormField(block: AssistantFlowBlock, blockKey: string, field: keyof ShipmentFormState, value: string | number) {
  resolveShipmentFormState(block, blockKey)[field] = String(value ?? "");
}

/** 判断发货表单当前是否可提交。 */
function isShipmentSubmitDisabled(block: AssistantFlowBlock, blockKey: string) {
  const action = resolveShipmentAction(block);
  const form = resolveShipmentFormState(block, blockKey);
  return Boolean(block.disabled || !action || (action as any).disabled || !isActionEnabled(action) || !form.name || !form.no);
}

/** 从发货卡片中读取已有物流数据。 */
function resolveShipmentLogistics(block: AssistantFlowBlock) {
  const order = isRecord(block.order) ? block.order : {};
  const logistics = isRecord(block.logistics) ? block.logistics : isRecord(order.logistics) ? order.logistics : {};
  return logistics;
}

/** 生成动作按钮文案。 */
function resolveActionLabel(action: AiAssistantAction, source?: Record<string, any>) {
  const actionLabelMap: Record<string, string> = {
    view_shipment_detail: "查看发货",
    confirm_shipment: "确认发货",
    view_comment_detail: "查看评价",
    confirm_comment_review: Number(readActionPayload(action).status ?? 0) === 2 ? "审核通过" : "审核不通过",
    view_goods_detail: "查看商品",
    confirm_goods_status: "确认变更",
    view_refund_detail: "查看退款",
    view_store_detail: "查看门店",
    confirm_store_audit: Number(readActionPayload(action).status ?? 0) === 2 ? "审核通过" : "审核拒绝"
  };
  return actionLabelMap[action.type] || formatValue(source?.title || source?.name || source?.order_no || "继续");
}

/** 读取 action 负载，按钮文案需要根据状态做轻量区分。 */
function readActionPayload(action: AiAssistantAction) {
  try {
    return JSON.parse(action.payload_json || "{}") as Record<string, unknown>;
  } catch {
    return {};
  }
}

/** 判断当前动作是否仍属于最新 Flow 消息。 */
function isActionEnabled(action?: AiAssistantAction) {
  if (!action?.type || !props.activeFlowMessageId) return false;
  return action.source_message_id === props.activeFlowMessageId && String(action.flow_version || "") === props.activeFlowMessageId;
}

/** 判断普通对象。 */
function isRecord(value: unknown): value is Record<string, any> {
  return Boolean(value) && typeof value === "object" && !Array.isArray(value);
}

/** 将任意值格式化成短文本。 */
function formatValue(value: unknown) {
  if (value === true) return "是";
  if (value === false) return "否";
  if (value === null || value === undefined) return "";
  if (typeof value === "object") return JSON.stringify(value);
  return String(value);
}

/** 将字段值转成表单字符串。 */
function stringValue(value: unknown) {
  if (value === null || value === undefined) return "";
  return String(value);
}
</script>

<style scoped lang="scss">
.assistant-flow-blocks {
  display: flex;
  flex-direction: column;
  gap: 12px;
  width: 100%;
  margin-top: 10px;
}
.assistant-flow-block {
  box-sizing: border-box;
  width: 100%;
  padding: 14px;
  color: var(--admin-page-text-primary);
  background: var(--admin-page-card-bg);
  border: 1px solid var(--admin-page-divider-strong);
  border-radius: var(--admin-page-radius);
}
.assistant-flow-block__head {
  display: flex;
  gap: 10px;
  align-items: center;
  justify-content: space-between;
}
.assistant-flow-block__title {
  min-width: 0;
  overflow: hidden;
  font-size: 14px;
  font-weight: 700;
  line-height: 22px;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.assistant-flow-block__type {
  flex: 0 0 auto;
  padding: 2px 7px;
  font-size: 12px;
  line-height: 18px;
  color: var(--admin-page-text-secondary);
  background: var(--el-fill-color-light);
  border-radius: var(--admin-page-radius);
}
.assistant-flow-block__desc {
  margin: 6px 0 0;
  font-size: 13px;
  line-height: 22px;
  color: var(--admin-page-text-secondary);
}
.assistant-flow-metrics {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(118px, 1fr));
  gap: 8px;
  margin-top: 12px;
}
.assistant-flow-metric,
.assistant-flow-field {
  min-width: 0;
  padding: 8px 10px;
  background: var(--el-fill-color-lighter);
  border-radius: var(--admin-page-radius);
}
.assistant-flow-metric span,
.assistant-flow-field span {
  display: block;
  overflow: hidden;
  font-size: 12px;
  line-height: 18px;
  color: var(--admin-page-text-secondary);
  text-overflow: ellipsis;
  white-space: nowrap;
}
.assistant-flow-metric strong,
.assistant-flow-field strong {
  display: block;
  margin-top: 2px;
  overflow-wrap: anywhere;
  font-size: 14px;
  line-height: 22px;
}
.assistant-flow-section {
  margin-top: 12px;
}
.assistant-flow-section__title {
  margin-bottom: 8px;
  font-size: 12px;
  font-weight: 700;
  line-height: 18px;
  color: var(--admin-page-text-secondary);
}
.assistant-flow-fields {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(148px, 1fr));
  gap: 8px;
}
.assistant-flow-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.assistant-flow-list__item {
  display: flex;
  gap: 12px;
  align-items: center;
  justify-content: space-between;
  min-width: 0;
  padding: 10px;
  background: var(--el-fill-color-lighter);
  border-radius: var(--admin-page-radius);
}
.assistant-flow-list__content {
  min-width: 0;
}
.assistant-flow-list__title {
  overflow: hidden;
  font-size: 13px;
  font-weight: 700;
  line-height: 22px;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.assistant-flow-list__fields {
  display: flex;
  flex-wrap: wrap;
  gap: 4px 10px;
  margin-top: 3px;
  font-size: 12px;
  line-height: 18px;
  color: var(--admin-page-text-secondary);
}
.assistant-flow-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 12px;
}
.assistant-flow-shipment {
  margin-top: 12px;
}
.assistant-flow-shipment__grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
}
.assistant-flow-shipment__field {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
  font-size: 12px;
  font-weight: 600;
  line-height: 18px;
  color: var(--admin-page-text-secondary);
}
@media screen and (width <= 768px) {
  .assistant-flow-list__item {
    align-items: stretch;
    flex-direction: column;
  }
  .assistant-flow-shipment__grid {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
