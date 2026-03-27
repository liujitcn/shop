import router from "@/routers/index";
import { LOGIN_URL } from "@/config";
import { RouteRecordRaw } from "vue-router";
import { ElNotification } from "element-plus";
import { useUserStore } from "@/stores/modules/user";
import { useAuthStore } from "@/stores/modules/auth";
import type { RouteItem } from "@/rpc/admin/auth";
import { getRouteMetaFull } from "@/utils";

// 引入 views 文件夹下所有 vue 文件
const modules = import.meta.glob("@/views/**/*.vue");
const pendingComponent = modules["/src/views/migration/pending/index.vue"];

/**
 * 规范化组件路径，兼容不同返回形式。
 */
function normalizeComponentPath(component?: string) {
  if (!component) return "";

  let normalizedComponent = component.trim();
  normalizedComponent = normalizedComponent.replace(/^\/?src\/views\//, "");
  normalizedComponent = normalizedComponent.replace(/^\/?views\//, "");
  normalizedComponent = normalizedComponent.replace(/^\//, "");
  normalizedComponent = normalizedComponent.replace(/\.vue$/, "");

  return normalizedComponent;
}

/**
 * 解析菜单对应的页面组件。
 */
function resolveRouteComponent(component?: string) {
  const normalizedComponent = normalizeComponentPath(component);
  if (!normalizedComponent || normalizedComponent === "Layout") return null;

  const candidates = [
    `/src/views/${normalizedComponent}.vue`,
    `/src/views/${normalizedComponent}/index.vue`,
    `/src/views/${normalizedComponent.replace(/\/index$/, "")}.vue`,
    `/src/views/${normalizedComponent.replace(/\/index$/, "")}/index.vue`
  ];

  const matchedKey = candidates.find(item => modules[item]);
  return matchedKey ? modules[matchedKey] : pendingComponent;
}

/**
 * 将后端路由项转换为前端路由记录。
 */
function createRouteRecord(item: RouteItem) {
  return {
    path: item.path ?? "",
    redirect: item.redirect ?? undefined,
    name: item.name ?? undefined,
    component: item.component ?? undefined,
    meta: item.meta ?? undefined
  } as RouteRecordRaw & {
    component?: string | (() => Promise<unknown>);
  };
}

/**
 * @description 初始化动态路由
 */
export const initDynamicRouter = async () => {
  const userStore = useUserStore();
  const authStore = useAuthStore();

  try {
    // 1.获取菜单列表 && 按钮权限列表
    await authStore.getAuthMenuList();
    await authStore.getAuthButtonList();

    // 2.判断当前用户有没有菜单权限
    if (!authStore.authMenuListGet.length) {
      ElNotification({
        title: "无权限访问",
        message: "当前账号无任何菜单权限，请联系系统管理员！",
        type: "warning",
        duration: 3000
      });
      userStore.clearAuthData();
      router.replace(LOGIN_URL);
      return Promise.reject("No permission");
    }

    // 3.添加动态路由
    authStore.flatMenuListGet.forEach(item => {
      const routeRecord = createRouteRecord(item);

      if (typeof routeRecord.component === "string") {
        const resolvedComponent = resolveRouteComponent(routeRecord.component);
        if (!resolvedComponent) return;
        routeRecord.component = resolvedComponent;
      }
      if (!routeRecord.component) return;
      if (routeRecord.name && router.hasRoute(routeRecord.name)) return;
      if (getRouteMetaFull(item.meta)) {
        router.addRoute(routeRecord);
      } else {
        router.addRoute("layout", routeRecord);
      }
    });
  } catch (error) {
    // 当按钮 || 菜单请求出错时，重定向到登陆页
    userStore.clearAuthData();
    router.replace(LOGIN_URL);
    return Promise.reject(error);
  }
};
