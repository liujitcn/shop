import type { FormRules } from "element-plus";
import type { ProFormOption } from "@/components/ProForm/interface";
import type { CodeGenLeftTreeConfig, CodeGenTableForm } from "@/rpc/admin/v1/code_gen_table";

/** 代码生成表配置页面类型选项。 */
export const codeGenPageTypeOptions: ProFormOption[] = [
  { label: "普通表格", value: "normal" },
  { label: "树形表格", value: "tree" },
  { label: "左树右表", value: "left_tree" }
];

/** 代码生成表配置状态选项。 */
export const codeGenStatusOptions: ProFormOption[] = [
  { label: "草稿", value: 0 },
  { label: "已生成", value: 1 },
  { label: "停用", value: 2 }
];

/** 左树数据源类型选项。 */
export const codeGenSourceTypeOptions: ProFormOption[] = [
  { label: "静态数据", value: "static" },
  { label: "字典", value: "dict" },
  { label: "数据表", value: "table" }
];

/** 代码生成表配置校验规则。 */
export const codeGenTableRules: FormRules = {
  name: [{ required: true, message: "请选择业务表", trigger: "change" }],
  business_name: [{ required: true, message: "业务名不能为空", trigger: "blur" }],
  entity_name: [{ required: true, message: "实体名不能为空", trigger: "blur" }],
  module_path: [{ required: true, message: "模块路径不能为空", trigger: "blur" }],
  parent_menu_id: [{ required: true, message: "请选择父级菜单", trigger: "change" }],
  page_type: [{ required: true, message: "请选择页面类型", trigger: "change" }],
  parent_column: [{ required: true, message: "请选择父节点字段", trigger: "change" }],
  tree_label_column: [{ required: true, message: "请选择树显示字段", trigger: "change" }],
  "left_tree_config.source_type": [{ required: true, message: "请选择左树来源", trigger: "change" }],
  "left_tree_config.source_value": [{ required: true, message: "请配置左树来源", trigger: ["blur", "change"] }],
  "left_tree_config.filter_column": [{ required: true, message: "请选择筛选字段", trigger: "change" }],
  "left_tree_config.parent_column": [{ required: true, message: "请配置左树父字段", trigger: ["blur", "change"] }],
  "left_tree_config.label_column": [{ required: true, message: "请配置左树显示字段", trigger: ["blur", "change"] }],
  "left_tree_config.value_column": [{ required: true, message: "请配置左树值字段", trigger: ["blur", "change"] }]
};

/** 创建默认左树右表页面配置。 */
export function createDefaultCodeGenLeftTreeConfig(): CodeGenLeftTreeConfig {
  return {
    source_type: "",
    source_value: "",
    filter_column: "",
    parent_column: "",
    label_column: "",
    value_column: ""
  };
}

/** 创建代码生成表配置默认表单。 */
export function createDefaultCodeGenTableForm(): CodeGenTableForm {
  return {
    id: 0,
    name: "",
    comment: "",
    business_name: "",
    entity_name: "",
    module_path: "",
    permission_prefix: "",
    page_type: "normal",
    parent_column: "",
    tree_label_column: "",
    left_tree_config: createDefaultCodeGenLeftTreeConfig(),
    gen_backend: true,
    gen_frontend: true,
    gen_sql: true,
    parent_menu_id: 0,
    status: 0,
    remark: ""
  };
}
