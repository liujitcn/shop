<template>
  <el-dialog v-model="dialogVisible" :title="nodeDialogTitle" width="760px" @closed="handleClosed">
    <ProForm
      ref="nodeFormRef"
      class="gorse-flow-node-form"
      :model="nodeFormModel"
      :fields="nodeFormFields"
      label-position="top"
      label-width="auto"
      :gutter="12"
      :col-span="12"
    >
      <template #fallback-recommenders="{ model }">
        <FallbackRecommendersField v-model="model.properties.recommenders" />
      </template>
    </ProForm>

    <NodePreviewPanel ref="nodePreviewPanelRef" :node-form="nodeForm" />

    <template #footer>
      <el-button @click="dialogVisible = false">取消</el-button>
      <el-button type="primary" @click="handleSubmit">保存</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import ProForm from "@/components/ProForm/index.vue";
import type { ProFormField, ProFormInstance } from "@/components/ProForm/interface";
import FallbackRecommendersField from "./FallbackRecommendersField.vue";
import NodePreviewPanel from "./NodePreviewPanel.vue";
import { buildNodeFormFields } from "../nodeFormFields";
import type { NodeFormState } from "../types";
import { fixedNodeTypes, nodeTypeLabelMap } from "../constants";

interface NodeEditDialogProps {
  /** 弹窗显示状态。 */
  modelValue: boolean;
  /** 当前编辑节点表单状态。 */
  nodeForm: NodeFormState;
}

const props = defineProps<NodeEditDialogProps>();
const emit = defineEmits<{
  /** 更新弹窗显示状态。 */
  "update:modelValue": [value: boolean];
  /** 表单校验通过后提交。 */
  submit: [];
}>();

const nodeFormRef = ref<ProFormInstance>();
const nodePreviewPanelRef = ref<InstanceType<typeof NodePreviewPanel>>();

const dialogVisible = computed({
  get: () => props.modelValue,
  set: value => emit("update:modelValue", value)
});
const currentNodeLabel = computed(() => nodeTypeLabelMap[props.nodeForm.type] ?? props.nodeForm.type);
const nodeDialogTitle = computed(() => `编辑${currentNodeLabel.value}`);
const canEditNodeName = computed(() => !fixedNodeTypes.has(props.nodeForm.type) && !props.nodeForm.properties.fixedName);
const nodeFormModel = computed(() => props.nodeForm as unknown as Record<string, any>);
const nodeFormFields = computed<ProFormField[]>(() => buildNodeFormFields(props.nodeForm.type, canEditNodeName.value));

/** 清理表单校验状态。 */
function clearValidate() {
  nodeFormRef.value?.clearValidate();
}

/** 校验表单并通知页面更新画布节点。 */
async function handleSubmit() {
  try {
    await nodeFormRef.value?.validate();
  } catch (error) {
    return;
  }
  emit("submit");
}

/** 弹窗完全关闭后清理子组件状态，避免下次编辑串用旧预览。 */
function handleClosed() {
  nodePreviewPanelRef.value?.resetPreviewState();
}

defineExpose({
  clearValidate
});
</script>

<style scoped lang="scss">
.gorse-flow-node-form {
  margin-top: 16px;
  padding-right: 8px;

  :deep(.el-input-number) {
    width: 100%;
  }
}
</style>
