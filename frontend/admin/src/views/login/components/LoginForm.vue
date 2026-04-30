<template>
  <el-form ref="loginFormRef" :model="loginForm" :rules="loginRules" size="large">
    <el-form-item prop="user_name">
      <el-input v-model="loginForm.user_name" placeholder="请输入用户名">
        <template #prefix>
          <el-icon class="el-input__icon">
            <user />
          </el-icon>
        </template>
      </el-input>
    </el-form-item>
    <el-form-item prop="password">
      <el-input v-model="loginForm.password" type="password" placeholder="请输入密码" show-password autocomplete="new-password">
        <template #prefix>
          <el-icon class="el-input__icon">
            <lock />
          </el-icon>
        </template>
      </el-input>
    </el-form-item>
    <el-form-item prop="captcha_code">
      <div class="captcha-row">
        <el-input v-model="loginForm.captcha_code" placeholder="请输入验证码" @keyup.enter="handleLogin(loginFormRef)">
          <template #prefix>
            <el-icon class="el-input__icon">
              <Key />
            </el-icon>
          </template>
        </el-input>
        <img v-if="captcha_base64" class="captcha-image" :src="captcha_base64" alt="验证码" @click="getCaptcha" />
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
import { ref, reactive, onMounted } from "vue";
import { useRoute, useRouter } from "vue-router";
import { HOME_URL } from "@/config";
import { getTimeState } from "@/utils";
import { defLoginService } from "@/api/base/login";
import { ElNotification } from "element-plus";
import type { LoginRequest } from "@/rpc/base/v1/login";
import { useUserStore } from "@/stores/modules/user";
import { useDictStore } from "@/stores/modules/dict";
import { useTabsStore } from "@/stores/modules/tabs";
import { useKeepAliveStore } from "@/stores/modules/keepAlive";
import { initDynamicRouter } from "@/routers/modules/dynamicRouter";
import { isUnmatchedRoute, navigateTo } from "@/utils/router";
import { CircleClose, UserFilled } from "@element-plus/icons-vue";
import type { ElForm } from "element-plus";

const router = useRouter();
const route = useRoute();
const userStore = useUserStore();
const dictStore = useDictStore();
const tabsStore = useTabsStore();
const keepAliveStore = useKeepAliveStore();

type FormInstance = InstanceType<typeof ElForm>;
const loginFormRef = ref<FormInstance>();
const captcha_base64 = ref("");
const loginRules = reactive({
  user_name: [{ required: true, message: "请输入用户名", trigger: "blur" }],
  password: [{ required: true, message: "请输入密码", trigger: "blur" }],
  captcha_code: [{ required: true, message: "请输入验证码", trigger: "blur" }]
});

const loading = ref(false);
const loginForm = reactive<LoginRequest>({
  user_name: "",
  password: "",
  captcha_code: "",
  captcha_id: ""
});

/** 获取登录后的首个可访问路由 */
const getFirstAccessibleRoutePath = () => {
  // 首页仅在真正完成动态路由注册后才视为可达，避免把全局 404 占位路由误判为首页已加载。
  if (!isUnmatchedRoute(router, HOME_URL)) return HOME_URL;

  const systemRouteSet = new Set(["/", "/layout", "/login", "/403", "/404", "/500"]);
  const firstRoute = router.getRoutes().find(item => {
    if (!item.path || systemRouteSet.has(item.path) || item.path.includes(":pathMatch")) return false;
    return item.meta?.hidden !== true;
  });
  return firstRoute?.path ?? HOME_URL;
};

/** 获取登录成功后的回跳地址，优先使用登录前记录的完整页面地址。 */
const getLoginRedirectPath = () => {
  const redirect = route.query.redirect;
  if (typeof redirect === "string" && redirect && redirect !== HOME_URL) {
    return redirect;
  }
  return getFirstAccessibleRoutePath();
};

/** 获取验证码 */
const getCaptcha = async () => {
  const data = await defLoginService.Captcha({});
  loginForm.captcha_id = data.captcha_id;
  loginForm.captcha_code = "";
  captcha_base64.value = data.captcha_base64;
};

/** 登录 */
const handleLogin = (formEl: FormInstance | undefined) => {
  // 登录请求执行期间直接拦截重复提交，避免回车或连续点击导致重复登录。
  if (!formEl || loading.value) return;
  formEl.validate(async valid => {
    if (!valid) return;
    loading.value = true;
    try {
      // 1.执行登录接口
      await userStore.login({ ...loginForm });

      // 2.获取用户信息
      await userStore.getUserInfo();

      // 3.预加载字典缓存，避免页面首次渲染时字典值为空
      await dictStore.loadDictionaries();

      // 4.添加动态路由
      await initDynamicRouter();

      // 5.清空 tabs、keepAlive 数据
      tabsStore.setTabs([]);
      keepAliveStore.setKeepAliveName([]);

      // 6.优先跳回登录失效前页面，没有记录时再进入首个可访问页面。
      // 统一走动态路由感知跳转，避免首次登录后目标页面尚未完成挂载时直接进入 404。
      await navigateTo(router, getLoginRedirectPath());
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
