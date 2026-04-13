<template>
  <div class="profile-security">
    <el-card class="security-card" shadow="never">
      <div class="security-intro">
        <div>
          <h3>安全设置</h3>
          <p>管理登录密码、手机验证和账号完整度。</p>
        </div>
        <el-tag effect="plain" type="success">账号状态正常</el-tag>
      </div>
      <div class="security-list">
        <div class="security-item">
          <div class="security-item__content">
            <strong>登录密码</strong>
            <p>建议定期更新，避免长期使用同一密码。</p>
          </div>
          <el-button type="primary" plain @click="emit('switchTab', 'password')">前往修改</el-button>
        </div>
        <div class="security-item">
          <div class="security-item__content">
            <strong>绑定手机</strong>
            <p>{{ mobileTip }}</p>
          </div>
          <el-button plain @click="openPhoneDialog">{{ profile.phone ? "更换手机号" : "立即绑定" }}</el-button>
        </div>
      </div>
    </el-card>

    <el-card class="status-card" shadow="never">
      <template #header>
        <div class="status-header">
          <div>
            <h3>账号状态</h3>
            <p>查看当前账号验证状态与资料完整度。</p>
          </div>
        </div>
      </template>
      <div class="status-grid">
        <div class="status-item">
          <span>手机验证</span>
          <strong>{{ profile.phone ? "已启用" : "未启用" }}</strong>
        </div>
        <div class="status-item">
          <span>资料完整度</span>
          <strong>{{ profileCompletion }}</strong>
        </div>
      </div>
    </el-card>

    <ProDialog v-model="phoneDialogVisible" title="绑定手机" :width="520" @closed="handleDialogClosed">
      <ProForm ref="phoneFormRef" :model="phoneForm" :fields="phoneFormFields" :rules="phoneFormRules" label-width="96px">
        <template #mobileCodeInput>
          <el-input v-model="phoneForm.code" placeholder="请输入验证码">
            <template #append>
              <el-button :disabled="countdown > 0" @click="handleSendCode">
                {{ countdown > 0 ? `${countdown}s后重试` : "发送验证码" }}
              </el-button>
            </template>
          </el-input>
        </template>
      </ProForm>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="phoneDialogVisible = false">取消</el-button>
          <el-button type="primary" :loading="submitLoading" @click="handleSubmitPhone">保存</el-button>
        </div>
      </template>
    </ProDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, reactive, ref } from "vue";
import { defAuthService } from "@/api/admin/auth";
import ProForm from "@/components/ProForm/index.vue";
import ProDialog from "@/components/Dialog/ProDialog.vue";
import type { ProFormField, ProFormInstance } from "@/components/ProForm/interface";
import type { SendPhoneCodeRequest, UserPhoneForm, UserProfileForm } from "@/rpc/admin/auth";
import { ElMessage } from "element-plus";

/** 安全中心组件属性。 */
interface ProfileSecurityProps {
  /** 当前用户资料。 */
  profile: UserProfileForm;
}

const props = defineProps<ProfileSecurityProps>();

const emit = defineEmits<{
  refreshed: [];
  switchTab: [tab: "account" | "security" | "password"];
}>();

const phoneFormRef = ref<ProFormInstance>();
const phoneDialogVisible = ref(false);
const submitLoading = ref(false);
const countdown = ref(0);
const phoneTimer = ref<number | null>(null);
const phoneForm = reactive<UserPhoneForm>({
  phone: "",
  code: ""
});
const sendPhoneCodeForm = reactive<SendPhoneCodeRequest>({
  phone: ""
});

const phoneFormFields: ProFormField[] = [
  { prop: "phone", label: "手机号码", component: "input", props: { placeholder: "请输入手机号" } },
  { prop: "code", label: "验证码", component: "slot", slotName: "mobileCodeInput" }
];

const phoneFormRules = {
  phone: [
    { required: true, message: "请输入手机号", trigger: "blur" },
    { pattern: /^1[3-9]\d{9}$/, message: "请输入正确的手机号码", trigger: "blur" }
  ],
  code: [{ required: true, message: "请输入验证码", trigger: "blur" }]
};

/** 根据当前绑定状态输出手机号说明文案。 */
const mobileTip = computed(() => {
  return props.profile.phone ? `已绑定：${props.profile.phone}` : "当前未绑定手机。";
});

/** 根据关键资料估算当前资料完成度。 */
const profileCompletion = computed(() => {
  const fieldList = [props.profile.nickName, props.profile.phone, props.profile.roleName, props.profile.deptName];
  const completedCount = fieldList.filter(item => Boolean(item)).length;
  return `${Math.round((completedCount / fieldList.length) * 100)}%`;
});

/** 打开手机号绑定弹窗，并回填当前手机号。 */
function openPhoneDialog() {
  phoneForm.phone = props.profile.phone;
  phoneForm.code = "";
  phoneDialogVisible.value = true;
}

/** 发送手机验证码并启动倒计时。 */
async function handleSendCode() {
  if (!phoneForm.phone) {
    ElMessage.error("请输入手机号");
    return;
  }
  if (!/^1[3-9]\d{9}$/.test(phoneForm.phone)) {
    ElMessage.error("手机号格式不正确");
    return;
  }

  sendPhoneCodeForm.phone = phoneForm.phone;
  await defAuthService.SendPhoneCode(sendPhoneCodeForm);
  ElMessage.success("验证码已发送");
  startCountdown();
}

/** 提交绑定手机号请求。 */
async function handleSubmitPhone() {
  if (!(await phoneFormRef.value?.validate())) return;

  submitLoading.value = true;
  try {
    await defAuthService.UpdateUserPhone(phoneForm);
    ElMessage.success("手机号更新成功");
    phoneDialogVisible.value = false;
    emit("refreshed");
  } finally {
    submitLoading.value = false;
  }
}

/** 启动验证码倒计时，并在重复发送前进行限制。 */
function startCountdown() {
  clearCountdown();
  countdown.value = 60;
  phoneTimer.value = window.setInterval(() => {
    if (countdown.value <= 1) {
      clearCountdown();
      return;
    }
    countdown.value -= 1;
  }, 1000);
}

/** 清理验证码倒计时。 */
function clearCountdown() {
  if (phoneTimer.value !== null) {
    window.clearInterval(phoneTimer.value);
    phoneTimer.value = null;
  }
  countdown.value = 0;
}

/** 弹窗关闭后重置临时表单状态。 */
function handleDialogClosed() {
  phoneFormRef.value?.resetFields();
  phoneFormRef.value?.clearValidate();
  phoneForm.phone = props.profile.phone;
  phoneForm.code = "";
}

onBeforeUnmount(() => {
  clearCountdown();
});
</script>

<style scoped lang="scss">
.profile-security {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.security-card,
.status-card {
  border: 1px solid #ebeef5;
  border-radius: 12px;
}

:deep(.security-card .el-card__body),
:deep(.status-card .el-card__body) {
  padding: 20px;
}

:deep(.status-card .el-card__header) {
  padding: 18px 20px;
  border-bottom: 1px solid #f0f2f5;
}

.security-intro,
.status-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.security-intro h3,
.status-header h3 {
  margin: 0;
  font-size: 18px;
  color: #303133;
}

.security-intro p,
.status-header p {
  margin: 6px 0 0;
  font-size: 13px;
  color: #909399;
}

.security-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-top: 16px;
}

.security-item,
.status-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
  padding: 16px;
  background: #fff;
  border: 1px solid #f0f2f5;
  border-radius: 10px;
}

.security-item__content {
  min-width: 0;
}

.security-item strong,
.status-item strong {
  display: block;
  margin-bottom: 6px;
  font-size: 15px;
  color: #303133;
}

.security-item p,
.status-item span {
  margin: 0;
  font-size: 13px;
  line-height: 1.6;
  color: #909399;
}

.status-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
}

@media screen and (width <= 768px) {
  .security-intro,
  .status-header,
  .security-item,
  .status-item {
    flex-direction: column;
    align-items: flex-start;
  }

  .status-grid {
    grid-template-columns: 1fr;
  }
}
</style>
