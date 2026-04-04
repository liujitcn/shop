<template>
  <div class="profile-page">
    <div class="profile-header">
      <div>
        <h2>个人中心</h2>
        <p>统一管理账号资料、安全设置和登录密码。</p>
      </div>
    </div>

    <section class="profile-shell">
      <aside class="profile-nav">
        <div class="profile-nav__header">
          <strong>{{ userProfileForm.nickName || userProfileForm.userName || "未设置昵称" }}</strong>
          <span>{{ userProfileForm.roleName || "未分配角色" }}</span>
        </div>
        <button
          v-for="tab in profileTabs"
          :key="tab.value"
          type="button"
          class="nav-item"
          :class="{ 'nav-item--active': activeTab === tab.value }"
          @click="handleTabChange(tab.value)"
        >
          <div class="nav-item__content">
            <strong>{{ tab.label }}</strong>
            <span>{{ tab.description }}</span>
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
  padding: 20px;
}

.profile-header {
  margin-bottom: 16px;
}

.profile-header h2 {
  margin: 0;
  font-size: 22px;
  font-weight: 600;
  color: #303133;
}

.profile-header p {
  margin: 6px 0 0;
  font-size: 13px;
  color: #909399;
}

.profile-shell {
  display: grid;
  grid-template-columns: 260px minmax(0, 1fr);
  gap: 16px;
  align-items: start;
}

.profile-nav {
  position: sticky;
  top: 20px;
  padding: 16px;
  background: #fff;
  border: 1px solid #ebeef5;
  border-radius: 12px;
  box-shadow: 0 2px 12px rgb(0 0 0 / 4%);
}

.profile-nav__header {
  padding-bottom: 14px;
  margin-bottom: 14px;
  border-bottom: 1px solid #f0f2f5;
}

.profile-nav__header strong {
  display: block;
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

.profile-nav__header span {
  display: block;
  margin-top: 6px;
  font-size: 13px;
  color: #909399;
}

.nav-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  padding: 14px 12px;
  color: #606266;
  text-align: left;
  cursor: pointer;
  background: #fff;
  border: 1px solid transparent;
  border-radius: 10px;
  transition: all 0.2s ease;
}

.nav-item + .nav-item {
  margin-top: 8px;
}

.nav-item__content {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.nav-item strong {
  display: block;
  font-size: 14px;
  font-weight: 600;
}

.nav-item span {
  font-size: 12px;
  color: #909399;
}

.nav-item:hover,
.nav-item--active {
  color: #409eff;
  background: #f5f9ff;
  border-color: #d9ecff;
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
