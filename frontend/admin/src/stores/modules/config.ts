import { defineStore } from "pinia";
import { defConfigService } from "@/api/base/config";
import { BaseConfigSite } from "@/rpc/common/enum";
import type { SiteConfigState, SiteDisplayConfig } from "@/stores/interface";
import piniaPersistConfig from "@/stores/helper/persist";
import defaultLogoUrl from "@/assets/images/logo.svg";
import defaultBackgroundUrl from "@/assets/images/login_left.png";

const DEFAULT_SITE_DISPLAY_CONFIG: SiteDisplayConfig = {
  sysName: "Shop Admin",
  icp: "",
  copyright: "2026 © Shop Admin",
  watermark: "Shop Working",
  adminLogo: defaultLogoUrl,
  background: defaultBackgroundUrl
};

/**
 * 合并站点展示配置，忽略空字符串配置，避免后端返回空值覆盖默认值。
 */
function mergeSiteDisplayConfig(
  currentDisplayConfig: SiteDisplayConfig,
  nextDisplayConfig: Partial<SiteDisplayConfig>
): SiteDisplayConfig {
  const mergedDisplayConfig = { ...currentDisplayConfig };

  Object.entries(nextDisplayConfig).forEach(([key, value]) => {
    if (typeof value !== "string") return;
    if (!value.trim()) return;
    mergedDisplayConfig[key as keyof SiteDisplayConfig] = value;
  });

  return mergedDisplayConfig;
}

/**
 * 将服务端配置项转换为站点展示配置字段。
 */
function normalizeSiteDisplayConfig(configMap: Record<string, string>) {
  return {
    sysName: configMap.sysName,
    icp: configMap.icp,
    copyright: configMap.copyright,
    watermark: configMap.watermark,
    adminLogo: configMap.adminLogo,
    background: configMap.background
  } satisfies Partial<SiteDisplayConfig>;
}

export const useConfigStore = defineStore({
  id: "shop-config",
  state: (): SiteConfigState => ({
    display: { ...DEFAULT_SITE_DISPLAY_CONFIG }
  }),
  getters: {},
  actions: {
    /**
     * 设置站点展示配置。
     */
    setDisplayConfig(nextDisplayConfig: Partial<SiteDisplayConfig>) {
      this.display = mergeSiteDisplayConfig(this.display, nextDisplayConfig);
    },
    /**
     * 重置为默认站点展示配置。
     */
    resetDisplayConfig() {
      this.display = { ...DEFAULT_SITE_DISPLAY_CONFIG };
    },
    /**
     * 加载管理端站点配置，并以服务端返回值覆盖本地默认值。
     */
    async loadDisplayConfig() {
      const configResponse = await defConfigService.GetConfig({
        site: BaseConfigSite.ADMIN
      });
      const configMap: Record<string, string> = {};

      configResponse.data?.forEach(configItem => {
        if (!configItem.key) return;
        configMap[configItem.key] = configItem.value ?? "";
      });

      this.setDisplayConfig(normalizeSiteDisplayConfig(configMap));
      return this.display;
    }
  },
  persist: piniaPersistConfig("shop-config")
});
