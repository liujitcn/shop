<template>
  <el-dropdown ref="dropdownRef" trigger="click" placement="bottom-end" popper-class="header-avatar-dropdown">
    <div class="avatar">
      <img :src="avatarSrc" alt="avatar" @error="handleAvatarError" />
    </div>
    <template #dropdown>
      <div class="user-panel">
        <div class="user-panel__summary">
          <div class="user-panel__avatar">
            <img :src="avatarSrc" alt="avatar" @error="handleAvatarError" />
          </div>
          <div class="user-panel__identity">
            <div class="user-panel__name">{{ displayName }}</div>
            <div class="user-panel__meta">{{ profileSummary }}</div>
          </div>
        </div>
        <div class="user-panel__actions">
          <button class="action-btn action-btn--primary" type="button" @click.stop="handleGoToProfile">
            <el-icon><User /></el-icon>
            <span>个人中心</span>
          </button>
          <button class="action-btn action-btn--danger" type="button" @click="logout">
            <el-icon><SwitchButton /></el-icon>
            <span>退出登录</span>
          </button>
        </div>
      </div>
    </template>
  </el-dropdown>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { LOGIN_URL } from "@/config";
import { useRouter } from "vue-router";
import { useUserStore } from "@/stores/modules/user";
import { ElMessageBox, ElMessage } from "element-plus";
import type { DropdownInstance } from "element-plus";
import defaultAvatar from "@/assets/images/avatar.png";

const router = useRouter();
const userStore = useUserStore();
const dropdownRef = ref<DropdownInstance>();
const avatarSrc = ref(defaultAvatar);
const displayName = computed(() => userStore.userInfo.nickName || userStore.userInfo.userName || "未设置");
const roleName = computed(() => userStore.userInfo.roleName || "未分配角色");
const deptName = computed(() => userStore.userInfo.deptName || "未分配部门");
const profileSummary = computed(() => `${roleName.value} / ${deptName.value}`);

/**
 * 同步头部头像展示，优先使用用户头像，为空时回退默认头像。
 *
 * @param avatar 用户头像地址
 */
const syncAvatarSrc = (avatar?: string) => {
  // 用户未上传头像时，统一回退到本地默认头像。
  avatarSrc.value = avatar || defaultAvatar;
};

watch(
  () => userStore.userInfo.avatar,
  avatar => {
    syncAvatarSrc(avatar);
  },
  { immediate: true }
);

/** 退出登录并清理当前登录态。 */
const logout = () => {
  ElMessageBox.confirm("您是否确认退出登录?", "温馨提示", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning"
  }).then(async () => {
    // 1.执行退出登录接口并清理本地状态
    await userStore.logout();

    // 2.重定向到登录页
    router.replace(LOGIN_URL);
    ElMessage.success("退出登录成功！");
  });
};

/** 头像加载失败时兜底显示默认头像，避免出现破图。 */
const handleAvatarError = () => {
  avatarSrc.value = defaultAvatar;
};

/**
 * 跳转到个人中心页。
 * 头像面板统一收敛到个人中心入口，进入后默认展示账号信息模块。
 */
const goToProfile = async () => {
  await router.push("/profile");
};

/** 关闭头像下拉后跳转个人中心，避免自定义浮层吞掉点击行为。 */
const handleGoToProfile = async () => {
  dropdownRef.value?.handleClose();
  await goToProfile();
};
</script>

<style scoped lang="scss">
.avatar {
  width: 40px;
  height: 40px;
  overflow: hidden;
  cursor: pointer;
  border-radius: 50%;
  border: 2px solid rgba(255, 255, 255, 0.85);
  box-shadow: 0 8px 18px rgba(15, 23, 42, 0.12);

  img {
    width: 100%;
    height: 100%;
    display: block;
    object-fit: cover;
  }
}

.user-panel {
  width: 280px;
  padding: 16px;
  background: linear-gradient(180deg, #ffffff 0%, #f7f9fc 100%);
  border: 1px solid rgba(15, 23, 42, 0.08);
  border-radius: 12px;
  box-shadow: 0 16px 40px rgba(15, 23, 42, 0.14);

  &__summary {
    display: flex;
    align-items: center;
    padding-bottom: 14px;
    border-bottom: 1px solid rgba(15, 23, 42, 0.08);
  }

  &__avatar {
    width: 48px;
    height: 48px;
    overflow: hidden;
    flex-shrink: 0;
    border-radius: 50%;
    background: #edf2f7;

    img {
      width: 100%;
      height: 100%;
      display: block;
      object-fit: cover;
    }
  }

  &__identity {
    min-width: 0;
    margin-left: 12px;
  }

  &__name {
    font-size: 16px;
    font-weight: 600;
    line-height: 22px;
    color: #1f2937;
  }

  &__meta {
    margin-top: 3px;
    overflow: hidden;
    font-size: 13px;
    line-height: 18px;
    color: #6b7280;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  &__actions {
    display: flex;
    flex-direction: column;
    margin-top: 14px;
    gap: 10px;
  }
}

.action-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  width: 100%;
  height: 38px;
  cursor: pointer;
  border: none;
  border-radius: 10px;
  transition:
    transform 0.2s ease,
    box-shadow 0.2s ease,
    background-color 0.2s ease;

  // 悬浮时仅做轻微上浮，避免头部浮层交互显得过重。
  &:hover {
    transform: translateY(-1px);
  }

  &--primary {
    color: #1d4ed8;
    background: rgba(37, 99, 235, 0.1);

    &:hover {
      box-shadow: 0 8px 18px rgba(37, 99, 235, 0.16);
      background: rgba(37, 99, 235, 0.14);
    }
  }

  &--danger {
    color: #dc2626;
    background: rgba(220, 38, 38, 0.08);

    &:hover {
      box-shadow: 0 8px 18px rgba(220, 38, 38, 0.14);
      background: rgba(220, 38, 38, 0.12);
    }
  }
}

:deep(.header-avatar-dropdown) {
  padding: 0;
  border: none;
  box-shadow: none;
  background: transparent;
}
</style>
