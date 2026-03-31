<template>
  <el-input v-if="field.component === 'input'" v-model.trim="fieldValue" v-bind="fieldProps" />

  <el-input
    v-else-if="field.component === 'password'"
    v-model.trim="fieldValue"
    type="password"
    show-password
    v-bind="fieldProps"
  />

  <el-input v-else-if="field.component === 'textarea'" v-model.trim="fieldValue" type="textarea" v-bind="fieldProps" />

  <el-input-number v-else-if="field.component === 'input-number'" v-model="fieldValue" v-bind="fieldProps" />

  <el-switch v-else-if="field.component === 'switch'" v-model="fieldValue" v-bind="fieldProps" />

  <el-select v-else-if="field.component === 'select'" v-model="fieldValue" v-bind="fieldProps">
    <el-option
      v-for="option in fieldOptions"
      :key="String(option.value)"
      :label="option.label"
      :value="option.value"
      :disabled="option.disabled"
    />
  </el-select>

  <el-radio-group v-else-if="field.component === 'radio-group'" v-model="fieldValue" v-bind="fieldProps">
    <el-radio v-for="option in fieldOptions" :key="String(option.value)" :label="option.value" :disabled="option.disabled">
      {{ option.label }}
    </el-radio>
  </el-radio-group>

  <el-checkbox-group v-else-if="field.component === 'checkbox-group'" v-model="fieldValue" v-bind="fieldProps">
    <el-checkbox v-for="option in fieldOptions" :key="String(option.value)" :label="option.value" :disabled="option.disabled">
      {{ option.label }}
    </el-checkbox>
  </el-checkbox-group>

  <el-tree-select v-else-if="field.component === 'tree-select'" v-model="fieldValue" :data="fieldOptions" v-bind="fieldProps" />

  <el-date-picker v-else-if="field.component === 'date-picker'" v-model="fieldValue" v-bind="fieldProps" />

  <CronExpression v-else-if="field.component === 'cron-expression'" v-model="fieldValue" v-bind="fieldProps" />

  <Dict v-else-if="field.component === 'dict'" v-model="fieldValue" v-bind="fieldProps" />

  <el-transfer
    v-else-if="field.component === 'transfer'"
    v-model="multipleFileValue"
    :data="fieldOptions"
    :props="{ key: 'value', label: 'label', disabled: 'disabled' }"
    v-bind="fieldProps"
  >
    <template v-if="field.slotName" #default="slotScope">
      <slot :name="field.slotName" :field="field" :model="model" v-bind="slotScope" />
    </template>
  </el-transfer>

  <UploadImg v-else-if="field.component === 'image-upload'" v-model:image-url="fieldValue" v-bind="fieldProps" />

  <UploadImgs v-else-if="field.component === 'images-upload'" v-model:file-list="multipleImageFileValue" v-bind="fieldProps" />

  <UploadFile v-else-if="field.component === 'file-upload'" v-model:file-info="fieldValue" v-bind="fieldProps" />

  <UploadFiles v-else-if="field.component === 'files-upload'" v-model:file-list="multipleFileValue" v-bind="fieldProps" />

  <WangEditor v-else-if="field.component === 'rich-text'" v-model:value="fieldValue" v-bind="fieldProps" />

  <DynamicList v-else-if="field.component === 'dynamic-list'" v-model="multipleStringValue" v-bind="fieldProps" />

  <KvList v-else-if="field.component === 'kv-list'" v-model="multipleKvValue" v-bind="fieldProps" />

  <slot v-else :name="field.slotName ?? field.prop" :field="field" :model="model" />
</template>

<script setup lang="ts" name="ProFormItem">
import { computed } from "vue";
import type { UploadUserFile } from "element-plus";
import type { ProFormField, ProFormOption } from "@/components/ProForm/interface";
import CronExpression from "@/components/CronExpression/index.vue";
import Dict from "@/components/Dict/index.vue";
import WangEditor from "@/components/WangEditor/index.vue";
import DynamicList from "@/components/ProForm/components/DynamicList.vue";
import KvList from "@/components/ProForm/components/KvList.vue";
import UploadFile from "@/components/Upload/File.vue";
import UploadFiles from "@/components/Upload/Files.vue";
import UploadImg from "@/components/Upload/Img.vue";
import UploadImgs from "@/components/Upload/Imgs.vue";

interface ProFormItemProps {
  field: ProFormField;
  model: Record<string, any>;
}

const props = defineProps<ProFormItemProps>();

/** 解析字段组件参数，支持静态对象和函数。 */
const fieldProps = computed(() => {
  if (!props.field.props) return {};
  return typeof props.field.props === "function" ? props.field.props(props.model) : props.field.props;
});

/** 解析字段选项参数，支持静态数组和函数。 */
const fieldOptions = computed<ProFormOption[]>(() => {
  if (!props.field.options) return [];
  return typeof props.field.options === "function" ? props.field.options(props.model) : props.field.options;
});

/** 根据点路径读取字段值。 */
function getFieldValue() {
  return props.field.prop.split(".").reduce<any>((current, key) => current?.[key], props.model);
}

/** 根据点路径写入字段值。 */
function setFieldValue(value: unknown) {
  const pathList = props.field.prop.split(".");
  const lastKey = pathList.pop();
  if (!lastKey) return;

  const target = pathList.reduce<Record<string, any>>((current, key) => {
    if (!current[key] || typeof current[key] !== "object") {
      current[key] = {};
    }
    return current[key];
  }, props.model);

  target[lastKey] = value;
}

/** 统一接管字段值的双向绑定。 */
const fieldValue = computed({
  get: () => getFieldValue(),
  set: value => setFieldValue(value)
});

/** 统一处理数组型上传字段。 */
const multipleFileValue = computed({
  get: () => (Array.isArray(fieldValue.value) ? fieldValue.value : []),
  set: value => setFieldValue(value)
});

/**
 * 将多图上传字段兼容为 UploadImgs 所需的 UploadUserFile[]，
 * 并在回写时根据原始数据结构恢复为 string[] 或 UploadUserFile[]。
 */
const multipleImageFileValue = computed<UploadUserFile[]>({
  get: () => {
    if (!Array.isArray(fieldValue.value)) return [];
    return fieldValue.value.map((item: string | UploadUserFile) => {
      if (typeof item === "string") {
        return {
          name: item.split("/").pop() ?? "image",
          url: item
        };
      }
      return item;
    });
  },
  set: value => {
    const shouldStoreObjectList =
      Array.isArray(fieldValue.value) && fieldValue.value.some(item => typeof item === "object" && item !== null);
    if (shouldStoreObjectList) {
      setFieldValue(value);
      return;
    }
    setFieldValue(value.map(item => item.url ?? "").filter(Boolean));
  }
});

/** 统一处理字符串数组字段。 */
const multipleStringValue = computed({
  get: () => (Array.isArray(fieldValue.value) ? fieldValue.value : []),
  set: value => setFieldValue(value)
});

/** 统一处理键值对数组字段。 */
const multipleKvValue = computed({
  get: () => (Array.isArray(fieldValue.value) ? fieldValue.value : []),
  set: value => setFieldValue(value)
});
</script>
