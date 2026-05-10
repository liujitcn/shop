<template>
  <div class="agent-confirm-card">
    <div class="agent-confirm-title">
      <el-icon><Warning /></el-icon>
      <span>{{ title }}</span>
    </div>
    <div class="agent-confirm-state" :class="`is-${state}`">
      <span class="agent-confirm-state__dot"></span>
      <span>{{ stateText }}</span>
    </div>
    <div class="agent-confirm-lines">
      <div v-for="line in lines" :key="line">{{ line }}</div>
    </div>
    <div v-if="formFields.length" class="agent-confirm-form">
      <div v-for="field in formFields" :key="field.prop" class="agent-confirm-form__item">
        <div class="agent-confirm-form__label">{{ field.label }}</div>
        <el-input
          v-model="localForm[field.prop]"
          size="small"
          :disabled="isActionLocked || disabled"
          :placeholder="field.placeholder"
        />
      </div>
    </div>
    <div class="agent-confirm-actions">
      <el-button size="small" plain :disabled="isActionLocked || disabled" @click="handleAction('reject')">拒绝</el-button>
      <el-button size="small" type="warning" :disabled="isActionLocked || disabled" @click="handleAction('confirm')">
        确认
      </el-button>
    </div>
  </div>
</template>

<script setup lang="ts" name="ConfirmCard">
import { computed, reactive, watch } from "vue";
import { Warning } from "@element-plus/icons-vue";

type ConfirmCardState = "pending" | "processing" | "confirmed" | "rejected";
type ConfirmCardAction = "confirm" | "reject";
type ConfirmCardFormField = {
  prop: string;
  label: string;
  placeholder: string;
  required?: boolean;
};

const props = defineProps<{
  /** 确认卡标题。 */
  title: string;
  /** 确认卡内容行。 */
  lines: string[];
  /** 当前确认卡表单字段。 */
  formFields?: ConfirmCardFormField[];
  /** 当前确认卡表单值。 */
  formValues?: Record<string, string>;
  /** 当前确认卡交互状态。 */
  state?: ConfirmCardState;
  /** 外部发送中的禁用态。 */
  disabled?: boolean;
}>();

const emit = defineEmits<{
  /** 触发确认卡动作。 */
  action: [payload: { action: ConfirmCardAction; formValues: Record<string, string> }];
}>();

const stateTextMap: Record<ConfirmCardState, string> = {
  pending: "等待处理",
  processing: "提交中",
  confirmed: "已确认",
  rejected: "已拒绝"
};

const state = computed<ConfirmCardState>(() => props.state ?? "pending");

const stateText = computed(() => stateTextMap[state.value]);

const isActionLocked = computed(() => state.value === "processing" || state.value === "confirmed" || state.value === "rejected");
const formFields = computed(() => props.formFields ?? []);
const localForm = reactive<Record<string, string>>({});

watch(
  () => [props.formFields, props.formValues],
  () => {
    formFields.value.forEach(field => {
      localForm[field.prop] = props.formValues?.[field.prop] ?? "";
    });
  },
  { immediate: true, deep: true }
);

/** 将确认卡动作回传给上层消息流。 */
function handleAction(action: ConfirmCardAction) {
  if (isActionLocked.value || props.disabled) return;
  emit("action", {
    action,
    formValues: { ...localForm }
  });
}
</script>

<style scoped lang="scss">
.agent-confirm-card {
  min-width: 320px;
  padding: 16px;
  background: var(--el-color-warning-light-9);
  border: 1px solid var(--el-color-warning-light-5);
  border-radius: 14px;
}

.agent-confirm-title {
  display: flex;
  gap: 6px;
  align-items: center;
  font-size: 14px;
  font-weight: 700;
  color: var(--el-color-warning-dark-2);
}

.agent-confirm-state {
  display: inline-flex;
  gap: 6px;
  align-items: center;
  margin-top: 10px;
  padding: 4px 10px;
  font-size: 12px;
  line-height: 18px;
  color: var(--admin-page-text-secondary);
  background: rgb(255 255 255 / 72%);
  border-radius: 999px;
}

.agent-confirm-state.is-pending {
  color: var(--el-color-warning-dark-2);
}

.agent-confirm-state.is-processing {
  color: var(--el-color-primary);
}

.agent-confirm-state.is-confirmed {
  color: var(--el-color-success);
}

.agent-confirm-state.is-rejected {
  color: var(--el-color-danger);
}

.agent-confirm-state__dot {
  width: 8px;
  height: 8px;
  background: currentcolor;
  border-radius: 50%;
}

.agent-confirm-lines {
  margin-top: 12px;
  font-size: 13px;
  line-height: 24px;
  color: var(--admin-page-text-secondary);
}

.agent-confirm-form {
  display: grid;
  gap: 10px;
  margin-top: 14px;
}

.agent-confirm-form__item {
  display: grid;
  gap: 6px;
}

.agent-confirm-form__label {
  font-size: 12px;
  color: var(--admin-page-text-secondary);
}

.agent-confirm-actions {
  display: flex;
  gap: 8px;
  margin-top: 14px;
}
</style>
