<template>
  <div class="app-container">
    <el-tabs v-model="activeTab" tab-position="left" @tab-change="handleTabChange">
      <!-- 基本设置 Tab Pane -->
      <el-tab-pane label="账号信息" name="account">
        <div class="w-full">
          <el-card>
            <!-- 头像和昵称部分 -->
            <div class="relative w-100px h-100px flex-center">
              <el-avatar :src="userProfileForm.avatar" :size="100" />
              <el-button
                type="info"
                class="absolute bottom-0 right-0 cursor-pointer"
                circle
                :icon="Camera"
                size="small"
                @click="triggerFileUpload"
              />
              <input ref="fileInput" type="file" style="display: none" @change="handleFileChange" />
            </div>
            <div class="mt-5">
              {{ userProfileForm.nickName }}
              <el-icon class="align-middle cursor-pointer" @click="handleOpenDialog(DialogType.ACCOUNT)">
                <Edit />
              </el-icon>
            </div>
            <!-- 用户信息描述 -->
            <el-descriptions :column="1" class="mt-10">
              <!-- 用户名 -->
              <el-descriptions-item>
                <template #label>
                  <el-icon class="align-middle"><User /></el-icon>
                  用户名
                </template>
                {{ userProfileForm.userName }}
                <el-icon v-if="userProfileForm.gender === 1" class="align-middle color-blue">
                  <Male />
                </el-icon>
                <el-icon v-else class="align-middle color-pink">
                  <Female />
                </el-icon>
              </el-descriptions-item>
              <el-descriptions-item>
                <template #label>
                  <el-icon class="align-middle"><Phone /></el-icon>
                  手机号码
                </template>
                {{ userProfileForm.phone }}
              </el-descriptions-item>
              <el-descriptions-item>
                <template #label>
                  <SvgIcon icon-class="tree" />
                  部门
                </template>
                {{ userProfileForm.deptName }}
              </el-descriptions-item>
              <el-descriptions-item>
                <template #label>
                  <SvgIcon icon-class="role" />
                  角色
                </template>
                {{ userProfileForm.roleName }}
              </el-descriptions-item>

              <el-descriptions-item>
                <template #label>
                  <el-icon class="align-middle"><Timer /></el-icon>
                  创建日期
                </template>
                {{ userProfileForm.createdAt }}
              </el-descriptions-item>
            </el-descriptions>
          </el-card>
        </div>
      </el-tab-pane>

      <!-- 安全设置  -->
      <el-tab-pane label="安全设置" name="security">
        <el-card>
          <!-- 账户密码 -->
          <el-row>
            <el-col :span="16">
              <div class="font-bold">账户密码</div>
              <div class="text-14px mt-2">
                定期修改密码有助于保护账户安全
                <el-button type="primary" plain size="small" class="ml-5" @click="() => handleOpenDialog(DialogType.PASSWORD)">
                  修改
                </el-button>
              </div>
            </el-col>
          </el-row>
          <!-- 绑定手机 -->
          <div class="mt-5">
            <div class="font-bold">绑定手机</div>
            <div class="text-14px mt-2">
              <span v-if="userProfileForm.phone">已绑定手机号：{{ userProfileForm.phone }}</span>
              <span v-else>未绑定手机</span>
              <el-button
                v-if="userProfileForm.phone"
                type="primary"
                plain
                size="small"
                class="ml-5"
                @click="() => handleOpenDialog(DialogType.MOBILE)"
              >
                更换
              </el-button>
              <el-button v-else type="primary" plain size="small" class="ml-5" @click="() => handleOpenDialog(DialogType.MOBILE)">
                绑定
              </el-button>
            </div>
          </div>
        </el-card>
      </el-tab-pane>
    </el-tabs>

    <!-- 弹窗 -->
    <el-dialog v-model="dialog.visible" :title="dialog.title" :width="500" @closed="handleDialogClose">
      <!-- 账号资料 -->
      <ProForm
        v-if="dialog.type === DialogType.ACCOUNT"
        ref="accountFormRef"
        :model="userProfileForm"
        :fields="accountFormFields"
        label-width="100px"
      />

      <!-- 修改密码 -->
      <ProForm
        v-if="dialog.type === DialogType.PASSWORD"
        ref="passwordFormRef"
        :model="updatePwdForm"
        :fields="passwordFormFields"
        :rules="updatePwdFormRules"
        label-width="100px"
      />
      <!-- 绑定手机 -->
      <ProForm
        v-else-if="dialog.type === DialogType.MOBILE"
        ref="mobileFormRef"
        :model="updatePhoneForm"
        :fields="mobileFormFields"
        :rules="updatePhoneFormRules"
        label-width="100px"
      >
        <template #mobileCodeInput>
          <el-input v-model="updatePhoneForm.code" style="width: 250px">
            <template #append>
              <el-button class="ml-5" :disabled="mobileCountdown > 0" @click="handleSendVerificationCode()">
                {{ mobileCountdown > 0 ? `${mobileCountdown}s后重新发送` : "发送验证码" }}
              </el-button>
            </template>
          </el-input>
        </template>
      </ProForm>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="dialog.visible = false">取消</el-button>
          <el-button type="primary" @click="handleSubmit">确定</el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<script lang="ts" setup>
defineOptions({
  name: "Profile",
  inheritAttrs: false
});
import { nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from "vue";
import { defAuthService } from "@/api/admin/auth";
import { UpdatePhoneForm, UpdatePwdForm, UserProfileForm } from "@/rpc/admin/auth";
import ProForm from "@/components/ProForm/index.vue";
import type { ProFormField, ProFormInstance } from "@/components/ProForm/interface";
import { ElMessage } from "element-plus";
import { Camera, Edit, Female, Male, Phone, Timer, User } from "@element-plus/icons-vue";
import { defFileService } from "@/api/base/file";
import { useRoute, useRouter } from "vue-router";

enum DialogType {
  ACCOUNT = "account",
  PASSWORD = "password",
  MOBILE = "phone"
}

type ProfileTab = "account" | "security";

const dialog = reactive({
  visible: false,
  title: "",
  type: "" as DialogType // 修改账号资料,修改密码、绑定手机、绑定邮箱
});
const route = useRoute();
const router = useRouter();
const activeTab = ref<ProfileTab>("account");

const accountFormRef = ref<ProFormInstance>();
const passwordFormRef = ref<ProFormInstance>();
const mobileFormRef = ref<ProFormInstance>();

const userProfileForm = reactive<UserProfileForm>({
  /** 用户名 */
  userName: "",
  /** 昵称 */
  nickName: "",
  /** 头像URL */
  avatar: "",
  /** 性别 */
  gender: 3,
  /** 手机号 */
  phone: "",
  /** 角色名 */
  roleName: "",
  /** 部门名 */
  deptName: "",
  /** 创建时间 */
  createdAt: ""
});
const updatePwdForm = reactive<UpdatePwdForm>({
  /** 原密码 */
  oldPwd: "",
  /** 新密码 */
  newPwd: "",
  /** 确认密码 */
  confirmPwd: ""
});
const updatePhoneForm = reactive<UpdatePhoneForm>({
  /** 手机号 */
  phone: "",
  /** 验证码 */
  code: ""
});

const mobileCountdown = ref(0);
const mobileTimer = ref<NodeJS.Timeout | null>(null);

// 修改密码校验规则
const updatePwdFormRules = {
  oldPwd: [{ required: true, message: "请输入原密码", trigger: "blur" }],
  newPwd: [{ required: true, message: "请输入新密码", trigger: "blur" }],
  confirmPwd: [{ required: true, message: "请再次输入新密码", trigger: "blur" }]
};

// 手机号校验规则
const updatePhoneFormRules = {
  phone: [
    { required: true, message: "请输入手机号", trigger: "blur" },
    {
      pattern: /^1[3|4|5|6|7|8|9][0-9]\d{8}$/,
      message: "请输入正确的手机号码",
      trigger: "blur"
    }
  ],
  code: [{ required: true, message: "请输入验证码", trigger: "blur" }]
};

const accountFormFields: ProFormField[] = [
  { prop: "nickName", label: "昵称", component: "input" },
  { prop: "gender", label: "性别", component: "dict", props: { code: "base_user_gender" } }
];

const passwordFormFields: ProFormField[] = [
  { prop: "oldPwd", label: "原密码", component: "password" },
  { prop: "newPwd", label: "新密码", component: "password" },
  { prop: "confirmPwd", label: "确认密码", component: "password" }
];

const mobileFormFields: ProFormField[] = [
  { prop: "phone", label: "手机号码", component: "input", props: { style: { width: "250px" } } },
  { prop: "code", label: "验证码", component: "slot", slotName: "mobileCodeInput" }
];

const tabDialogMap: Record<string, { tab: ProfileTab; dialog?: DialogType }> = {
  account: { tab: "account", dialog: DialogType.ACCOUNT },
  password: { tab: "security", dialog: DialogType.PASSWORD },
  phone: { tab: "security", dialog: DialogType.MOBILE },
  security: { tab: "security" }
};

const syncRouteQuery = (tab: ProfileTab, dialogType?: DialogType | "") => {
  const nextQuery: Record<string, string> = { ...route.query } as Record<string, string>;
  nextQuery.tab = tab;
  if (dialogType) {
    nextQuery.dialog = dialogType;
  } else {
    delete nextQuery.dialog;
  }
  router.replace({ path: route.path, query: nextQuery });
};

const applyRouteState = async () => {
  const tabQuery = Array.isArray(route.query.tab) ? route.query.tab[0] : route.query.tab;
  const dialogQuery = Array.isArray(route.query.dialog) ? route.query.dialog[0] : route.query.dialog;
  const matchedState = tabDialogMap[dialogQuery || ""] ?? tabDialogMap[tabQuery || ""];

  activeTab.value = matchedState?.tab ?? (tabQuery === "security" ? "security" : "account");

  if (!matchedState?.dialog) return;

  await nextTick();
  handleOpenDialog(matchedState.dialog);
};

const handleDialogClose = () => {
  dialog.type = "" as DialogType;
  syncRouteQuery(activeTab.value);
};

/**
 * 打开弹窗
 * @param type 弹窗类型 ACCOUNT: 账号资料 PASSWORD: 修改密码 MOBILE: 绑定手机 EMAIL: 绑定邮箱
 */
const handleOpenDialog = (type: DialogType) => {
  dialog.type = type;
  dialog.visible = true;
  switch (type) {
    case DialogType.ACCOUNT:
      dialog.title = "账号资料";
      break;
    case DialogType.PASSWORD:
      dialog.title = "修改密码";
      break;
    case DialogType.MOBILE:
      dialog.title = "绑定手机";
      break;
  }
  nextTick(() => {
    accountFormRef.value?.clearValidate();
    passwordFormRef.value?.clearValidate();
    mobileFormRef.value?.clearValidate();
  });
  syncRouteQuery(activeTab.value, type);
};

/**
 *  发送验证码
 */
const handleSendVerificationCode = async () => {
  if (!updatePhoneForm.phone) {
    ElMessage.error("请输入手机号");
    return;
  }
  // 验证手机号格式
  const reg = /^1[3-9]\d{9}$/;
  if (!reg.test(updatePhoneForm.phone)) {
    ElMessage.error("手机号格式不正确");
    return;
  }
  await defAuthService.SendUpdatePhoneCode({
    phone: updatePhoneForm.phone
  });

  mobileCountdown.value = 60;
  mobileTimer.value = setInterval(() => {
    if (mobileCountdown.value > 0) {
      mobileCountdown.value -= 1;
    } else {
      clearInterval(mobileTimer.value!);
    }
  }, 1000);
};

/**
 * 提交表单
 */
const handleSubmit = async () => {
  switch (dialog.type) {
    case DialogType.ACCOUNT:
      defAuthService.UpdateUserProfile(userProfileForm).then(() => {
        ElMessage.success("账号资料修改成功");
        dialog.visible = false;
        loadUserProfile();
      });
      break;
    case DialogType.PASSWORD:
      if (!(await passwordFormRef.value?.validate())) return;
      if (updatePwdForm.newPwd !== updatePwdForm.confirmPwd) {
        ElMessage.error("两次输入的密码不一致");
        return;
      }
      defAuthService.UpdateUserPwd(updatePwdForm).then(() => {
        ElMessage.success("密码修改成功");
        dialog.visible = false;
      });
      break;
    case DialogType.MOBILE:
      if (!(await mobileFormRef.value?.validate())) return;
      defAuthService.UpdateUserPhone(updatePhoneForm).then(() => {
        ElMessage.success("手机号修改成功");
        dialog.visible = false;
        loadUserProfile();
      });
      break;
  }
};

const fileInput = ref<HTMLInputElement | null>(null);

const triggerFileUpload = () => {
  fileInput.value?.click();
};

const handleFileChange = async (event: Event) => {
  const target = event.target as HTMLInputElement;
  const file = target.files ? target.files[0] : null;
  if (file) {
    try {
      const data = await defFileService.UploadFile(file, "avatar");
      // 更新用户头像
      userProfileForm.avatar = data.url;
      // 更新用户信息
      await defAuthService.UpdateUserProfile(userProfileForm);
    } catch (error) {
      ElMessage.error("头像上传失败");
    }
  }
};

/** 加载用户信息 */
const loadUserProfile = async () => {
  const data = await defAuthService.GetUserProfile({});
  Object.assign(userProfileForm, data);
};

const handleTabChange = (name: string | number) => {
  activeTab.value = name === "security" ? "security" : "account";
  dialog.visible = false;
  syncRouteQuery(activeTab.value);
};

watch(
  () => [route.query.tab, route.query.dialog],
  () => {
    applyRouteState();
  }
);

onMounted(async () => {
  if (mobileTimer.value) {
    clearInterval(mobileTimer.value);
  }
  await loadUserProfile();
  await applyRouteState();
});

onBeforeUnmount(() => {
  if (mobileTimer.value) {
    clearInterval(mobileTimer.value);
  }
});
</script>

<style lang="scss" scoped>
/** 关闭tag标签  */
.app-container {
  /* 50px = navbar = 50px */
  height: calc(100vh - 50px);
  background: var(--el-fill-color-blank);
}

/** 开启tag标签  */
.hasTagsView {
  .app-container {
    /* 84px = navbar + tags-view = 50px + 34px */
    height: calc(100vh - 84px);
  }
}
</style>
