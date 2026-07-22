import type { ProFormField, ProFormOption } from "@/components/ProForm/interface";
import { itemToItemTypeOptions, rankerTypeOptions, userToUserTypeOptions } from "./constants";

/** ProForm 可见性判断接收的节点表单模型。 */
type NodeFormFieldModel = Record<string, any>;

/** 根据节点类型生成 ProForm 字段配置，避免使用大 JSON 文本框编辑节点属性。 */
export function buildNodeFormFields(type: string, canEditNodeName: boolean): ProFormField[] {
  const fields: ProFormField[] = [];
  if (canEditNodeName) {
    fields.push(textField("name", "节点名称", "请输入节点名称"));
  }

  if (type === "data-source") {
    fields.push(
      dynamicListField("properties.positive_feedback_types", "正反馈类型", "如 CLICK"),
      dynamicListField("properties.read_feedback_types", "已读反馈类型", "如 VIEW"),
      dynamicListField("properties.negative_feedback_types", "负反馈类型", "如 DISLIKE"),
      numberField("properties.positive_feedback_ttl", "正反馈有效期", 0),
      numberField("properties.item_ttl", "商品有效期", 0)
    );
  }
  if (type === "recommend") {
    fields.push(
      numberField("properties.cache_size", "推荐缓存数量", 0),
      textField("properties.cache_expire", "推荐缓存过期", "如 120h"),
      numberField("properties.active_user_ttl", "活跃用户有效期", 0),
      numberField("properties.context_size", "上下文数量", 0),
      switchField("properties.replacement.enable_replacement", "启用替换"),
      numberField("properties.replacement.positive_replacement_decay", "正反馈替换衰减", 0, 1, 0.1),
      numberField("properties.replacement.read_replacement_decay", "已读替换衰减", 0, 1, 0.1)
    );
  }
  if (type === "collaborative") {
    fields.push(...buildTrainingFields("properties"));
  }
  if (type === "non-personalized") {
    fields.push(
      textareaField("properties.score", "评分表达式", "请输入评分表达式", 4),
      textareaField("properties.filter", "过滤表达式", "请输入过滤表达式", 3)
    );
  }
  if (type === "user-to-user") {
    fields.push(
      selectField("properties.type", "相似度类型", userToUserTypeOptions),
      textField("properties.column", "相似度字段", "请输入相似度字段")
    );
  }
  if (type === "item-to-item") {
    fields.push(
      selectField("properties.type", "相似度类型", itemToItemTypeOptions),
      textField("properties.column", "相似度字段", "请输入相似度字段"),
      textareaField("properties.prompt", "提示词", "请输入提示词", 3)
    );
  }
  if (type === "external") {
    fields.push(textareaField("properties.script", "外部脚本", "请输入外部推荐脚本", 8, 24));
  }
  if (type === "ranker") {
    fields.push(
      selectField("properties.type", "排序器类型", rankerTypeOptions, 24),
      ...buildTrainingFields("properties"),
      textField("properties.cache_expire", "缓存过期", "如 120h"),
      textareaField(
        "properties.query_template",
        "查询模板",
        "请输入大语言模型查询模板",
        4,
        24,
        model => model.properties?.type === "llm"
      ),
      textareaField(
        "properties.document_template",
        "文档模板",
        "请输入大语言模型文档模板",
        4,
        24,
        model => model.properties?.type === "llm"
      )
    );
  }
  if (type === "fallback") {
    fields.push(slotField("properties.recommenders", "兜底推荐器", "fallback-recommenders", 24));
  }
  return fields;
}

/** 生成训练类节点共用字段。 */
function buildTrainingFields(prefix: string): ProFormField[] {
  return [
    textField(`${prefix}.fit_period`, "训练周期", "如 24h"),
    numberField(`${prefix}.fit_epoch`, "训练轮数", 1),
    textField(`${prefix}.optimize_period`, "调参周期", "如 168h"),
    numberField(`${prefix}.optimize_trials`, "调参次数", 1),
    numberField(`${prefix}.early_stopping.patience`, "早停等待轮数", 0, undefined, 1, 24)
  ];
}

/** 创建文本输入字段配置。 */
function textField(prop: string, label: string, placeholder: string, colSpan = 12): ProFormField {
  return {
    prop,
    label,
    component: "input",
    colSpan,
    props: { placeholder, clearable: true }
  };
}

/** 创建多行文本输入字段配置。 */
function textareaField(
  prop: string,
  label: string,
  placeholder: string,
  rows: number,
  colSpan = 24,
  visible?: (model: NodeFormFieldModel) => boolean
): ProFormField {
  return {
    prop,
    label,
    component: "textarea",
    colSpan,
    props: { placeholder, rows },
    visible
  };
}

/** 创建数值输入字段配置。 */
function numberField(prop: string, label: string, min?: number, max?: number, step = 1, colSpan = 12): ProFormField {
  return {
    prop,
    label,
    component: "input-number",
    colSpan,
    props: {
      min,
      max,
      step,
      controlsPosition: "right"
    }
  };
}

/** 创建开关字段配置。 */
function switchField(prop: string, label: string, colSpan = 12): ProFormField {
  return {
    prop,
    label,
    component: "switch",
    colSpan
  };
}

/** 创建下拉选择字段配置。 */
function selectField(prop: string, label: string, options: ProFormOption[], colSpan = 12): ProFormField {
  return {
    prop,
    label,
    component: "select",
    colSpan,
    options,
    props: {
      clearable: false
    }
  };
}

/** 创建字符串数组字段配置。 */
function dynamicListField(prop: string, label: string, placeholder: string, colSpan = 12): ProFormField {
  return {
    prop,
    label,
    component: "dynamic-list",
    colSpan,
    props: {
      inputProps: { placeholder, clearable: true }
    }
  };
}

/** 创建自定义插槽字段配置。 */
function slotField(prop: string, label: string, slotName: string, colSpan = 12): ProFormField {
  return {
    prop,
    label,
    component: "slot",
    colSpan,
    slotName
  };
}
