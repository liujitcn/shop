<!-- 分栏布局 -->
<template>
  <el-container class="layout">
    <div class="aside-split">
      <div class="logo flx-center">
        <img class="logo-img" :src="logoUrl" alt="logo" />
      </div>
      <el-scrollbar>
        <div class="split-list">
          <div
            v-for="item in menuList"
            :key="item.path"
            class="split-item"
            :class="{ 'split-active': splitActive === item.path || `/${splitActive.split('/')[1]}` === item.path }"
            @click="changeSubMenu(item)"
          >
            <el-icon>
              <component :is="getRouteMetaIcon(item.meta)"></component>
            </el-icon>
            <span class="title">{{ getRouteMetaTitle(item.meta) }}</span>
          </div>
        </div>
      </el-scrollbar>
    </div>
    <el-aside :class="{ 'not-aside': !subMenuList.length }" :style="{ width: isCollapse ? '65px' : '210px' }">
      <div class="logo flx-center">
        <span v-show="subMenuList.length" class="logo-text">{{ isCollapse ? collapseTitle : title }}</span>
      </div>
      <el-scrollbar>
        <el-menu
          :router="false"
          :default-active="activeMenu"
          :collapse="isCollapse"
          :unique-opened="accordion"
          :collapse-transition="false"
        >
          <SubMenu :menu-list="subMenuList" />
        </el-menu>
      </el-scrollbar>
    </el-aside>
    <el-container>
      <el-header>
        <ToolBarLeft />
        <ToolBarRight />
      </el-header>
      <Main />
    </el-container>
  </el-container>
</template>

<script setup lang="ts" name="layoutColumns">
import { ref, computed, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { useAuthStore } from "@/stores/modules/auth";
import { useConfigStore } from "@/stores/modules/config";
import { useGlobalStore } from "@/stores/modules/global";
import type { RouteItem } from "@/rpc/admin/v1/auth";
import { getRouteMetaIcon, getRouteMetaTitle, isExternalPath } from "@/utils";
import Main from "@/layouts/components/Main/index.vue";
import ToolBarLeft from "@/layouts/components/Header/ToolBarLeft.vue";
import ToolBarRight from "@/layouts/components/Header/ToolBarRight.vue";
import SubMenu from "@/layouts/components/Menu/SubMenu.vue";

/** 具备必填路径字段的菜单项。 */
interface MenuRouteItem extends RouteItem {
  path: string;
}

const route = useRoute();
const router = useRouter();
const authStore = useAuthStore();
const configStore = useConfigStore();
const globalStore = useGlobalStore();
const accordion = computed(() => globalStore.accordion);
const isCollapse = computed(() => globalStore.isCollapse);
const menuList = computed<MenuRouteItem[]>(() => {
  return authStore.showMenuListGet.filter((item): item is MenuRouteItem => Boolean(item.path));
});
const activeMenu = computed(() => route.path as string);
const title = computed(() => configStore.display.sysName || import.meta.env.VITE_GLOB_APP_TITLE);
const collapseTitle = computed(() => title.value.slice(0, 1).toUpperCase() || "S");
const logoUrl = computed(() => configStore.display.adminLogo);

const subMenuList = ref<RouteItem[]>([]);
const splitActive = ref("");
watch(
  () => [menuList, route],
  () => {
    // 当前菜单没有数据直接 return
    if (!menuList.value.length) return;
    splitActive.value = route.path;
    const menuItem = menuList.value.filter((item: RouteItem) => {
      return route.path === item.path || `/${route.path.split("/")[1]}` === item.path;
    });
    if (menuItem[0].children?.length) return (subMenuList.value = menuItem[0].children);
    subMenuList.value = [];
  },
  {
    deep: true,
    immediate: true
  }
);

/** 切换分栏布局的二级菜单。 */
const changeSubMenu = (item: RouteItem) => {
  splitActive.value = item.path ?? "";
  if (item.children?.length) return (subMenuList.value = item.children);
  subMenuList.value = [];
  if (!item.path) return;
  if (isExternalPath(item.path)) {
    window.open(item.path, "_blank", "noopener,noreferrer");
    return;
  }
  router.push(item.path);
};
</script>

<style scoped lang="scss">
@use "./index.scss" as *;
</style>
