<template>
  <div class="profile-page">
    <section class="profile-shell">
      <aside class="profile-nav card-panel">
        <button
          v-for="tab in profileTabs"
          :key="tab.value"
          type="button"
          class="nav-item"
          :class="{ 'nav-item--active': activeTab === tab.value }"
          @click="handleTabChange(tab.value)"
        >
          <div>
            <strong>{{ tab.label }}</strong>
          </div>
          <el-icon><ArrowRight /></el-icon>
        </button>
      </aside>

      <main class="profile-content">
        <ProfileBase v-if="activeTab === 'account'" :profile="userProfileForm" @refreshed="loadUserProfile" />
        <ProfileSecurity
          v-else-if="activeTab === 'security'"
          :profile="userProfileForm"
          @refreshed="loadUserProfile"
          @switch-tab="handleTabChange"
        />
        <ProfilePassword v-else />
      </main>
    </section>
  </div>
</template>

<script setup lang="ts">
defineOptions({
  name: "Profile",
  inheritAttrs: false
});

import { onMounted, reactive, ref } from "vue";
import { defAuthService } from "@/api/admin/auth";
import type { UserProfileForm } from "@/rpc/admin/auth";
import { useUserStore } from "@/stores/modules/user";
import ProfileBase from "./components/base.vue";
import ProfileSecurity from "./components/security.vue";
import ProfilePassword from "./components/password.vue";
import { ArrowRight } from "@element-plus/icons-vue";

/** 个人中心标签页。 */
type ProfileTab = "account" | "security" | "password";

/** 左侧导航项结构。 */
interface ProfileTabOption {
  /** 标签值。 */
  value: ProfileTab;
  /** 导航标题。 */
  label: string;
  /** 导航描述。 */
  description: string;
}

const userStore = useUserStore();
const activeTab = ref<ProfileTab>("account");
const profileTabs: ProfileTabOption[] = [
  {
    value: "account",
    label: "账号信息",
    description: "维护头像和资料"
  },
  {
    value: "security",
    label: "安全设置",
    description: "管理验证与安全"
  },
  {
    value: "password",
    label: "修改密码",
    description: "更新登录密码"
  }
];
const userProfileForm = reactive<UserProfileForm>({
  userName: "",
  nickName: "",
  avatar: "",
  gender: 3,
  phone: "",
  roleName: "",
  deptName: "",
  createdAt: ""
});

/** 切换当前显示标签，仅更新本地视图状态，不触发路由变化。 */
function handleTabChange(tab: ProfileTab) {
  activeTab.value = tab;
}

/** 拉取当前登录用户的个人中心资料。 */
async function loadUserProfile() {
  const profile = await defAuthService.GetUserProfile({});
  Object.assign(userProfileForm, profile);
  // 个人中心资料更新后，同步刷新头部头像和昵称展示，避免页面内外信息不一致。
  userStore.setUserInfo({
    ...userStore.userInfo,
    userName: profile.userName,
    nickName: profile.nickName,
    phone: profile.phone,
    avatar: profile.avatar,
    roleName: profile.roleName,
    deptName: profile.deptName
  });
  // 无论从哪个入口进入个人中心，默认都展示账号信息模块。
  activeTab.value = "account";
}

onMounted(async () => {
  await loadUserProfile();
});
</script>

<style scoped lang="scss">
.profile-page {
  min-height: calc(100vh - 50px);
  padding: 24px;
  background:
    radial-gradient(circle at top left, rgb(230 239 255 / 88%), transparent 28%),
    radial-gradient(circle at bottom right, rgb(255 236 221 / 72%), transparent 24%),
    linear-gradient(180deg, #f6f9fd 0%, #f3f7fc 100%);
}

.card-panel {
  border: 1px solid #e7eef7;
  border-radius: 26px;
  background: rgb(255 255 255 / 88%);
  box-shadow: 0 22px 48px rgb(34 64 102 / 8%);
  backdrop-filter: blur(10px);
}

.profile-shell {
  display: grid;
  grid-template-columns: 280px minmax(0, 1fr);
  gap: 20px;
  align-items: start;
}

.profile-nav {
  position: sticky;
  top: 24px;
  padding: 14px;
}

.nav-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  padding: 18px 16px;
  color: #31445f;
  text-align: left;
  cursor: pointer;
  background: transparent;
  border: 0;
  border-radius: 18px;
  transition: all 0.2s ease;
}

.nav-item + .nav-item {
  margin-top: 8px;
}

.nav-item strong {
  display: block;
  margin-bottom: 6px;
  font-size: 15px;
}

.nav-item p {
  margin: 0;
  font-size: 12px;
  line-height: 1.6;
  color: #71839d;
}

.nav-item:hover,
.nav-item--active {
  color: #1d4ed8;
  background: linear-gradient(135deg, #edf4ff 0%, #fff3ea 100%);
}

.profile-content {
  min-width: 0;
}

@media screen and (width <= 1080px) {
  .profile-shell {
    grid-template-columns: 1fr;
  }

  .profile-nav {
    position: static;
  }
}

@media screen and (width <= 640px) {
  .profile-page {
    padding: 16px;
  }
}
</style>
