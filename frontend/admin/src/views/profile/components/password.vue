<template>
  <div class="profile-password">
    <el-card class="password-card" shadow="never">
      <template #header>
        <div class="panel-header">
          <div>
            <h3>修改密码</h3>
            <p>更新当前登录密码。</p>
          </div>
          <el-tag type="warning" effect="dark">建议高强度</el-tag>
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
          <div class="password-footer">
            <el-button @click="resetPasswordForm">重置</el-button>
            <el-button type="primary" :loading="submitLoading" @click="handleSubmitPassword">更新密码</el-button>
          </div>
        </div>
        <div class="password-tips">
          <div class="tip-card">
            <span class="tip-badge">01</span>
            <strong>避免重复密码</strong>
            <p>不要复用旧密码。</p>
          </div>
          <div class="tip-card">
            <span class="tip-badge">02</span>
            <strong>控制密码强度</strong>
            <p>尽量使用强密码。</p>
          </div>
          <div class="tip-card">
            <span class="tip-badge">03</span>
            <strong>及时通知团队</strong>
            <p>变更后及时同步。</p>
          </div>
        </div>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from "vue";
import { defAuthService } from "@/api/admin/auth";
import ProForm from "@/components/ProForm/index.vue";
import type { ProFormField, ProFormInstance } from "@/components/ProForm/interface";
import type { UpdatePwdForm } from "@/rpc/admin/auth";
import { ElMessage } from "element-plus";

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
  newPwd: [{ required: true, message: "请输入新密码", trigger: "blur" }],
  confirmPwd: [{ required: true, message: "请再次输入新密码", trigger: "blur" }]
};

/** 提交修改密码请求，并校验两次输入的一致性。 */
async function handleSubmitPassword() {
  if (!(await passwordFormRef.value?.validate())) return;
  if (passwordForm.newPwd !== passwordForm.confirmPwd) {
    ElMessage.error("两次输入的密码不一致");
    return;
  }

  submitLoading.value = true;
  try {
    await defAuthService.UpdateUserPwd(passwordForm);
    ElMessage.success("密码修改成功");
    resetPasswordForm();
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
</script>

<style scoped lang="scss">
.password-card {
  border: 1px solid #e7eef7;
  border-radius: 24px;
  box-shadow: 0 18px 40px rgb(34 64 102 / 8%);
}

:deep(.password-card .el-card__header) {
  padding: 22px 24px 0;
  border-bottom: 0;
}

:deep(.password-card .el-card__body) {
  padding: 20px 24px 24px;
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
  color: #1f3251;
}

.panel-header p {
  margin: 8px 0 0;
  font-size: 13px;
  line-height: 1.7;
  color: #70819b;
}

.password-layout {
  display: grid;
  grid-template-columns: minmax(0, 1.2fr) minmax(260px, 0.9fr);
  gap: 20px;
}

.password-form-wrap,
.tip-card {
  padding: 20px;
  background: #f8fbff;
  border: 1px solid #ebf1f8;
  border-radius: 18px;
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
  gap: 14px;
}

.tip-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  margin-bottom: 12px;
  font-size: 12px;
  font-weight: 700;
  color: #1d4ed8;
  background: #dce9ff;
  border-radius: 50%;
}

.tip-card strong {
  display: block;
  margin-bottom: 8px;
  font-size: 16px;
  color: #243754;
}

.tip-card p {
  margin: 0;
  font-size: 13px;
  line-height: 1.7;
  color: #70819b;
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
