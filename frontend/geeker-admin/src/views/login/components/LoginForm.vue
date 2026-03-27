<template>
  <el-form ref="loginFormRef" :model="loginForm" :rules="loginRules" size="large">
    <el-form-item prop="userName">
      <el-input v-model="loginForm.userName" placeholder="请输入用户名">
        <template #prefix>
          <el-icon class="el-input__icon">
            <user />
          </el-icon>
        </template>
      </el-input>
    </el-form-item>
    <el-form-item prop="password">
      <el-input v-model="loginForm.password" type="password" placeholder="密码：123456" show-password autocomplete="new-password">
        <template #prefix>
          <el-icon class="el-input__icon">
            <lock />
          </el-icon>
        </template>
      </el-input>
    </el-form-item>
    <el-form-item prop="captchaCode">
      <div class="captcha-row">
        <el-input v-model="loginForm.captchaCode" placeholder="请输入验证码" @keyup.enter="handleLogin(loginFormRef)">
          <template #prefix>
            <el-icon class="el-input__icon">
              <Key />
            </el-icon>
          </template>
        </el-input>
        <img v-if="captchaBase64" class="captcha-image" :src="captchaBase64" alt="验证码" @click="getCaptcha" />
      </div>
    </el-form-item>
  </el-form>
  <div class="login-btn">
    <el-button :icon="CircleClose" round size="large" @click="resetForm(loginFormRef)"> 重置 </el-button>
    <el-button :icon="UserFilled" round size="large" type="primary" :loading="loading" @click="handleLogin(loginFormRef)">
      登录
    </el-button>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, onBeforeUnmount } from "vue";
import { useRouter } from "vue-router";
import { HOME_URL } from "@/config";
import { getTimeState } from "@/utils";
import { defLoginService } from "@/api/base/login";
import { ElNotification } from "element-plus";
import type { LoginRequest } from "@/rpc/base/login";
import { useUserStore } from "@/stores/modules/user";
import { useAuthStore } from "@/stores/modules/auth";
import { useTabsStore } from "@/stores/modules/tabs";
import { useKeepAliveStore } from "@/stores/modules/keepAlive";
import { initDynamicRouter } from "@/routers/modules/dynamicRouter";
import { CircleClose, UserFilled } from "@element-plus/icons-vue";
import type { ElForm } from "element-plus";

const router = useRouter();
const userStore = useUserStore();
const authStore = useAuthStore();
const tabsStore = useTabsStore();
const keepAliveStore = useKeepAliveStore();

type FormInstance = InstanceType<typeof ElForm>;
const loginFormRef = ref<FormInstance>();
const captchaBase64 = ref("");
const loginRules = reactive({
  userName: [{ required: true, message: "请输入用户名", trigger: "blur" }],
  password: [{ required: true, message: "请输入密码", trigger: "blur" }],
  captchaCode: [{ required: true, message: "请输入验证码", trigger: "blur" }]
});

const loading = ref(false);
const loginForm = reactive<LoginRequest>({
  userName: "",
  password: "",
  captchaCode: "",
  captchaId: ""
});

/** 获取登录后的首个可访问路由 */
const getFirstAccessibleRoutePath = () => {
  const firstRoute = authStore.flatMenuListGet.find(item => item.path && item.component && !item.meta?.hidden);
  return firstRoute?.path ?? HOME_URL;
};

/** 获取验证码 */
const getCaptcha = async () => {
  const data = await defLoginService.Captcha({});
  loginForm.captchaId = data.captchaId;
  loginForm.captchaCode = "";
  captchaBase64.value = data.captchaBase64;
};

/** 登录 */
const handleLogin = (formEl: FormInstance | undefined) => {
  if (!formEl) return;
  formEl.validate(async valid => {
    if (!valid) return;
    loading.value = true;
    try {
      // 1.执行登录接口
      await userStore.login({ ...loginForm });

      // 2.获取用户信息
      await userStore.getUserInfo();

      // 3.添加动态路由
      await initDynamicRouter();

      // 4.清空 tabs、keepAlive 数据
      tabsStore.setTabs([]);
      keepAliveStore.setKeepAliveName([]);

      // 5.跳转到首个可访问页面
      router.push(getFirstAccessibleRoutePath());
      ElNotification({
        title: getTimeState(),
        message: "欢迎登录管理后台",
        type: "success",
        duration: 3000
      });
    } catch (_error) {
      await getCaptcha();
    } finally {
      loading.value = false;
    }
  });
};

/** 重置表单 */
const resetForm = (formEl: FormInstance | undefined) => {
  if (!formEl) return;
  formEl.resetFields();
  getCaptcha();
};

onMounted(() => {
  getCaptcha();
  // 监听 enter 事件（调用登录）
  document.onkeydown = (e: KeyboardEvent) => {
    if (e.code === "Enter" || e.code === "enter" || e.code === "NumpadEnter") {
      if (loading.value) return;
      handleLogin(loginFormRef.value);
    }
  };
});

onBeforeUnmount(() => {
  document.onkeydown = null;
});
</script>

<style scoped lang="scss">
@use "../index.scss" as *;

.captcha-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 120px;
  gap: 12px;
  width: 100%;
}

.captcha-image {
  width: 120px;
  height: 40px;
  cursor: pointer;
  border-radius: 6px;
  object-fit: cover;
}
</style>
