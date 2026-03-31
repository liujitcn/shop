<template>
  <el-card class="base-card" shadow="never">
    <template #header>
      <div class="panel-header">
        <div>
          <h3>账号信息</h3>
          <p>维护头像和基本资料。</p>
        </div>
        <el-button type="primary" plain @click="openAccountDialog">修改基本资料</el-button>
      </div>
    </template>

    <div class="base-layout">
      <div class="avatar-panel">
        <div class="avatar-shell">
          <el-avatar :src="profile.avatar" :size="116" />
          <el-button class="avatar-trigger" circle type="primary" :icon="Camera" @click="triggerFileUpload" />
          <input ref="fileInputRef" type="file" class="hidden-input" accept="image/*" @change="handleFileChange" />
        </div>
        <div class="avatar-copy">
          <strong>{{ profile.nickName || profile.userName || "未设置昵称" }}</strong>
          <span>{{ profile.roleName || "未分配角色" }}</span>
          <p>点击右下角可更换头像。</p>
        </div>
      </div>

      <div class="detail-grid">
        <div class="detail-item">
          <span class="detail-label">登录账号</span>
          <span class="detail-value">{{ profile.userName || "--" }}</span>
        </div>
        <div class="detail-item">
          <span class="detail-label">昵称</span>
          <span class="detail-value">{{ profile.nickName || "--" }}</span>
        </div>
        <div class="detail-item">
          <span class="detail-label">性别</span>
          <span class="detail-value">{{ genderText }}</span>
        </div>
        <div class="detail-item">
          <span class="detail-label">绑定手机</span>
          <span class="detail-value">{{ profile.phone || "未绑定" }}</span>
        </div>
        <div class="detail-item">
          <span class="detail-label">所属角色</span>
          <span class="detail-value">{{ profile.roleName || "--" }}</span>
        </div>
        <div class="detail-item">
          <span class="detail-label">所属部门</span>
          <span class="detail-value">{{ profile.deptName || "--" }}</span>
        </div>
        <div class="detail-item detail-item--wide">
          <span class="detail-label">创建时间</span>
          <span class="detail-value">{{ profile.createdAt || "--" }}</span>
        </div>
      </div>
    </div>

    <ProDialog v-model="accountDialogVisible" title="修改基本资料" :width="520" @closed="handleDialogClosed">
      <ProForm ref="accountFormRef" :model="accountForm" :fields="accountFormFields" label-width="96px" />
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="accountDialogVisible = false">取消</el-button>
          <el-button type="primary" :loading="submitLoading" @click="handleSubmitAccount">保存</el-button>
        </div>
      </template>
    </ProDialog>
  </el-card>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from "vue";
import { defAuthService } from "@/api/admin/auth";
import { defFileService } from "@/api/base/file";
import ProForm from "@/components/ProForm/index.vue";
import ProDialog from "@/components/Dialog/ProDialog.vue";
import type { ProFormField, ProFormInstance } from "@/components/ProForm/interface";
import type { UserProfileForm } from "@/rpc/admin/auth";
import { ElMessage } from "element-plus";
import { Camera } from "@element-plus/icons-vue";

/** 个人中心基础资料组件属性。 */
interface ProfileBaseProps {
  /** 当前用户资料。 */
  profile: UserProfileForm;
}

const props = defineProps<ProfileBaseProps>();

const emit = defineEmits<{
  refreshed: [];
}>();

const fileInputRef = ref<HTMLInputElement | null>(null);
const accountFormRef = ref<ProFormInstance>();
const accountDialogVisible = ref(false);
const submitLoading = ref(false);
const accountForm = reactive<Pick<UserProfileForm, "nickName" | "gender">>({
  nickName: "",
  gender: 3
});

const accountFormFields: ProFormField[] = [
  { prop: "nickName", label: "昵称", component: "input", props: { placeholder: "请输入昵称" } },
  { prop: "gender", label: "性别", component: "dict", props: { code: "base_user_gender" } }
];

/** 根据资料中的性别值输出展示文案。 */
const genderText = computed(() => {
  if (props.profile.gender === 1) return "男";
  if (props.profile.gender === 2) return "女";
  return "保密";
});

watch(
  () => props.profile,
  profile => {
    accountForm.nickName = profile.nickName;
    accountForm.gender = profile.gender;
  },
  { immediate: true, deep: true }
);

/** 触发头像文件选择。 */
function triggerFileUpload() {
  fileInputRef.value?.click();
}

/** 打开基本资料编辑弹窗，并回填当前资料。 */
function openAccountDialog() {
  accountForm.nickName = props.profile.nickName;
  accountForm.gender = props.profile.gender;
  accountDialogVisible.value = true;
}

/** 处理头像文件上传并同步到个人资料。 */
async function handleFileChange(event: Event) {
  const target = event.target as HTMLInputElement;
  const file = target.files?.[0];
  if (!file) return;

  try {
    const uploadResult = await defFileService.UploadFile(file, "avatar");
    await defAuthService.UpdateUserProfile({
      ...props.profile,
      avatar: uploadResult.url
    });
    ElMessage.success("头像更新成功");
    emit("refreshed");
  } catch (_error) {
    ElMessage.error("头像上传失败");
  } finally {
    target.value = "";
  }
}

/** 提交基本资料更新。 */
async function handleSubmitAccount() {
  if (!(await accountFormRef.value?.validate())) return;

  submitLoading.value = true;
  try {
    await defAuthService.UpdateUserProfile({
      ...props.profile,
      nickName: accountForm.nickName,
      gender: accountForm.gender
    });
    ElMessage.success("基本资料已更新");
    accountDialogVisible.value = false;
    emit("refreshed");
  } finally {
    submitLoading.value = false;
  }
}

/** 弹窗关闭后清理校验状态，并恢复表单值。 */
function handleDialogClosed() {
  accountFormRef.value?.clearValidate();
  accountForm.nickName = props.profile.nickName;
  accountForm.gender = props.profile.gender;
}
</script>

<style scoped lang="scss">
.base-card {
  border: 1px solid #e7eef7;
  border-radius: 24px;
  box-shadow: 0 18px 40px rgb(34 64 102 / 8%);
}

:deep(.base-card .el-card__header) {
  padding: 22px 24px 0;
  border-bottom: 0;
}

:deep(.base-card .el-card__body) {
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

.base-layout {
  display: grid;
  grid-template-columns: minmax(260px, 320px) minmax(0, 1fr);
  gap: 20px;
  align-items: start;
}

.avatar-panel {
  padding: 24px;
  text-align: center;
  background: linear-gradient(180deg, #f7fbff 0%, #fff6ef 100%);
  border: 1px solid #e8eef7;
  border-radius: 22px;
}

.avatar-shell {
  position: relative;
  display: inline-flex;
}

.avatar-trigger {
  position: absolute;
  right: 6px;
  bottom: 4px;
}

.hidden-input {
  display: none;
}

.avatar-copy {
  margin-top: 18px;
}

.avatar-copy strong {
  display: block;
  font-size: 20px;
  color: #243754;
}

.avatar-copy span {
  display: block;
  margin-top: 8px;
  font-size: 14px;
  color: #5f7390;
}

.avatar-copy p {
  margin: 12px 0 0;
  font-size: 13px;
  line-height: 1.7;
  color: #7a8ba6;
}

.detail-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
}

.detail-item {
  padding: 18px;
  background: #f8fbff;
  border: 1px solid #ebf1f8;
  border-radius: 18px;
}

.detail-item--wide {
  grid-column: 1 / -1;
}

.detail-label {
  display: block;
  margin-bottom: 10px;
  font-size: 12px;
  color: #7a8ba6;
}

.detail-value {
  font-size: 15px;
  font-weight: 600;
  color: #243754;
  word-break: break-all;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
}

@media screen and (width <= 992px) {
  .base-layout,
  .detail-grid {
    grid-template-columns: 1fr;
  }

  .detail-item--wide {
    grid-column: auto;
  }
}

@media screen and (width <= 640px) {
  .panel-header {
    flex-direction: column;
    align-items: flex-start;
  }
}
</style>
