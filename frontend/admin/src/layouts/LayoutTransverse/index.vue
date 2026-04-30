<!-- 横向布局 -->
<template>
  <el-container class="layout">
    <el-header>
      <div class="logo flx-center">
        <img class="logo-img" :src="logoUrl" alt="logo" />
        <span class="logo-text">{{ title }}</span>
      </div>
      <el-menu mode="horizontal" :router="false" :default-active="activeMenu">
        <!-- 不能直接使用 SubMenu 组件，无法触发 el-menu 隐藏省略功能 -->
        <template v-for="subItem in menuList" :key="subItem.path">
          <el-sub-menu v-if="subItem.children?.length" :key="subItem.path" :index="subItem.path + 'el-sub-menu'">
            <template #title>
              <el-icon>
                <component :is="getRouteMetaIcon(subItem.meta)"></component>
              </el-icon>
              <span>{{ getRouteMetaTitle(subItem.meta) }}</span>
            </template>
            <SubMenu :menu-list="subItem.children" />
          </el-sub-menu>
          <el-menu-item v-else :key="subItem.path + 'el-menu-item'" :index="subItem.path" @click="handleClickMenu(subItem)">
            <el-icon>
              <component :is="getRouteMetaIcon(subItem.meta)"></component>
            </el-icon>
            <template #title>
              <span>{{ getRouteMetaTitle(subItem.meta) }}</span>
            </template>
          </el-menu-item>
        </template>
      </el-menu>
      <ToolBarRight />
    </el-header>
    <Main />
  </el-container>
</template>

<script setup lang="ts" name="layoutTransverse">
import { computed } from "vue";
import { useAuthStore } from "@/stores/modules/auth";
import { useConfigStore } from "@/stores/modules/config";
import { useRoute, useRouter } from "vue-router";
import type { RouteItem } from "@/rpc/admin/v1/auth";
import { getRouteMetaIcon, getRouteMetaTitle, isExternalPath } from "@/utils";
import Main from "@/layouts/components/Main/index.vue";
import ToolBarRight from "@/layouts/components/Header/ToolBarRight.vue";
import SubMenu from "@/layouts/components/Menu/SubMenu.vue";

const route = useRoute();
const router = useRouter();
const authStore = useAuthStore();
const configStore = useConfigStore();
const menuList = computed(() => authStore.showMenuListGet);
const activeMenu = computed(() => route.path as string);
const title = computed(() => configStore.display.sysName || import.meta.env.VITE_GLOB_APP_TITLE);
const logoUrl = computed(() => configStore.display.adminLogo);

const handleClickMenu = (subItem: RouteItem) => {
  if (!subItem.path) return;
  if (isExternalPath(subItem.path)) {
    window.open(subItem.path, "_blank", "noopener,noreferrer");
    return;
  }
  router.push(subItem.path);
};
</script>

<style scoped lang="scss">
@use "./index.scss" as *;
</style>
