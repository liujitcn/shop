import type { FormItemRule } from "element-plus";

export type ProFormComponentType =
  | "input"
  | "password"
  | "textarea"
  | "input-number"
  | "switch"
  | "select"
  | "dict"
  | "radio-group"
  | "checkbox-group"
  | "tree-select"
  | "date-picker"
  | "cron-expression"
  | "transfer"
  | "image-upload"
  | "images-upload"
  | "file-upload"
  | "files-upload"
  | "rich-text"
  | "dynamic-list"
  | "kv-list"
  | "slot";

export interface ProFormOption {
  label: string;
  value: string | number | boolean;
  disabled?: boolean;
  children?: ProFormOption[];
}

export interface ProFormField {
  prop: string;
  label: string;
  component: ProFormComponentType;
  props?: Record<string, any> | ((model: Record<string, any>) => Record<string, any>);
  itemProps?: Record<string, any> | ((model: Record<string, any>) => Record<string, any>);
  options?: ProFormOption[] | ((model: Record<string, any>) => ProFormOption[]);
  colSpan?: number;
  slotName?: string;
  labelTooltip?: string;
  rules?: FormItemRule[];
  visible?: (model: Record<string, any>) => boolean;
}

export interface ProFormInstance {
  validate: () => Promise<boolean | undefined> | undefined;
  resetFields: () => void;
  clearValidate: (props?: string | string[]) => void;
}
