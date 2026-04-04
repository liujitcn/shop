<template>
  <div class="profile-password">
    <el-card class="password-card" shadow="never">
      <template #header>
        <div class="panel-header">
          <div>
            <h3>修改密码</h3>
            <p>更新当前登录密码，保持账号安全。</p>
          </div>
          <el-tag type="warning" effect="plain">建议高强度</el-tag>
        </div>
      </template>
      <div class="password-layout">
        <div class="password-form-wrap">
          <ProForm
            ref="passwordFormRef"
            :model="passwordForm"
            :fields="passwordFormFields"
            :rules="passwordFormRules"
            label-width="96px"
          />
          <PasswordStrength :password="passwordForm.newPwd" class="password-form-wrap__strength" />
          <div class="password-footer">
            <el-button @click="resetPasswordForm">重置</el-button>
            <el-button type="primary" :loading="submitLoading" @click="handleSubmitPassword">更新密码</el-button>
          </div>
        </div>
        <div class="password-tips">
          <div class="tip-card">
            <span class="tip-badge">01</span>
            <strong>避免重复密码</strong>
            <p>不要重复使用旧密码或其他系统已使用密码。</p>
          </div>
          <div class="tip-card">
            <span class="tip-badge">02</span>
            <strong>控制密码强度</strong>
            <p>建议使用大小写字母、数字和符号组合。</p>
          </div>
          <div class="tip-card">
            <span class="tip-badge">03</span>
            <strong>及时更新记录</strong>
            <p>如有账号交接，请同步更新保管记录。</p>
          </div>
        </div>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from "vue";
import { useRouter } from "vue-router";
import { defAuthService } from "@/api/admin/auth";
import PasswordStrength from "@/components/PasswordStrength/index.vue";
import ProForm from "@/components/ProForm/index.vue";
import { LOGIN_URL } from "@/config";
import type { ProFormField, ProFormInstance } from "@/components/ProForm/interface";
import type { UpdatePwdForm } from "@/rpc/admin/auth";
import { useUserStore } from "@/stores/modules/user";
import { ElMessage } from "element-plus";
import { PASSWORD_STRENGTH_ERROR_MESSAGE, getPasswordStrength, validatePasswordStrengthValue } from "@/utils/passwordStrength";

const router = useRouter();
const userStore = useUserStore();
const passwordFormRef = ref<ProFormInstance>();
const submitLoading = ref(false);
const passwordForm = reactive<UpdatePwdForm>({
  oldPwd: "",
  newPwd: "",
  confirmPwd: ""
});

const passwordFormFields: ProFormField[] = [
  { prop: "oldPwd", label: "原密码", component: "password", props: { placeholder: "请输入原密码" } },
  { prop: "newPwd", label: "新密码", component: "password", props: { placeholder: "请输入新密码" } },
  { prop: "confirmPwd", label: "确认密码", component: "password", props: { placeholder: "请再次输入新密码" } }
];

const passwordFormRules = {
  oldPwd: [{ required: true, message: "请输入原密码", trigger: "blur" }],
  newPwd: [
    { required: true, message: "请输入新密码", trigger: "blur" },
    { validator: validatePasswordStrength, trigger: "blur" }
  ],
  confirmPwd: [{ required: true, message: "请再次输入新密码", trigger: "blur" }]
};

/** 统一计算当前新密码强度，供展示和提交校验复用。 */
const passwordStrength = computed(() => getPasswordStrength(passwordForm.newPwd));

/** 提交修改密码请求，并校验两次输入的一致性。 */
async function handleSubmitPassword() {
  if (!(await passwordFormRef.value?.validate())) return;
  if (passwordForm.newPwd !== passwordForm.confirmPwd) {
    ElMessage.error("两次输入的密码不一致");
    return;
  }
  if (!passwordStrength.value.isValid) {
    ElMessage.error("新密码强度不足，请提升到最高强度");
    return;
  }

  submitLoading.value = true;
  try {
    await defAuthService.UpdateUserPwd(passwordForm);
    ElMessage.success("密码修改成功，请重新登录");
    resetPasswordForm();
    await forceRelogin();
  } finally {
    submitLoading.value = false;
  }
}

/** 重置密码表单内容与校验状态。 */
function resetPasswordForm() {
  passwordFormRef.value?.resetFields();
  passwordFormRef.value?.clearValidate();
  passwordForm.oldPwd = "";
  passwordForm.newPwd = "";
  passwordForm.confirmPwd = "";
}

/** 修改密码成功后强制重新登录，避免旧登录态继续使用。 */
async function forceRelogin() {
  try {
    await userStore.logout();
  } catch (_error) {
    // 若后端已提前让旧令牌失效，前端仍需保证本地登录态被清空。
    userStore.clearAuthData();
  }
  await router.replace(LOGIN_URL);
}

/**
 * 校验新密码强度，必须达到最高强度才允许提交。
 *
 * @param _rule 表单规则对象
 * @param value 当前输入的新密码
 * @param callback 校验回调
 */
function validatePasswordStrength(_rule: unknown, value: string, callback: (error?: Error) => void) {
  if (!value) {
    callback();
    return;
  }
  const result = validatePasswordStrengthValue(value);
  if (!result.valid) {
    callback(new Error(result.message || PASSWORD_STRENGTH_ERROR_MESSAGE));
    return;
  }
  callback();
}
</script>

<style scoped lang="scss">
.password-card {
  border: 1px solid #ebeef5;
  border-radius: 12px;
}

:deep(.password-card .el-card__header) {
  padding: 18px 20px;
  border-bottom: 1px solid #f0f2f5;
}

:deep(.password-card .el-card__body) {
  padding: 20px;
}

.panel-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.panel-header h3 {
  margin: 0;
  font-size: 18px;
  color: #303133;
}

.panel-header p {
  margin: 6px 0 0;
  font-size: 13px;
  color: #909399;
}

.password-layout {
  display: grid;
  grid-template-columns: minmax(0, 1.2fr) minmax(260px, 0.9fr);
  gap: 16px;
}

.password-form-wrap,
.tip-card {
  padding: 18px;
  background: #fff;
  border: 1px solid #f0f2f5;
  border-radius: 10px;
}

.password-form-wrap__strength {
  margin-top: 16px;
}

.password-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 24px;
}

.password-tips {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.tip-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  margin-bottom: 10px;
  font-size: 12px;
  font-weight: 700;
  color: #409eff;
  background: #ecf5ff;
  border-radius: 8px;
}

.tip-card strong {
  display: block;
  margin-bottom: 8px;
  font-size: 15px;
  color: #303133;
}

.tip-card p {
  margin: 0;
  font-size: 13px;
  line-height: 1.6;
  color: #909399;
}

@media screen and (width <= 960px) {
  .password-layout {
    grid-template-columns: 1fr;
  }
}

@media screen and (width <= 640px) {
  .panel-header {
    flex-direction: column;
    align-items: flex-start;
  }
}
</style>
