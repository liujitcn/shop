import defaultSettings from "@/settings";
import { ThemeEnum } from "@/enums/ThemeEnum";
import { generateThemeColors, applyTheme, toggleDarkMode } from "@/utils/theme";
import { defConfigService } from "@/api/base/config";

type SettingsValue = boolean | string;

export const useSettingsStore = defineStore("setting", () => {
  // 基本设置
  const settingsVisible = ref(false);
  // 标签
  const tagsView = useStorage<boolean>("tagsView", defaultSettings.tagsView);
  // 侧边栏 Logo
  const sidebarLogo = useStorage<boolean>("sidebarLogo", defaultSettings.sidebarLogo);
  // 布局
  const layout = useStorage<string>("layout", defaultSettings.layout);
  // 水印
  const watermarkEnabled = useStorage<boolean>(
    "watermarkEnabled",
    defaultSettings.watermarkEnabled
  );

  // 后台配置
  const data = useStorage<Record<string, string>>("data", {});

  // 主题
  const themeColor = useStorage<string>("themeColor", defaultSettings.themeColor);
  const theme = useStorage<string>("theme", defaultSettings.theme);

  // 监听主题变化
  watch(
    [theme, themeColor],
    ([newTheme, newThemeColor]) => {
      toggleDarkMode(newTheme === ThemeEnum.DARK);
      const colors = generateThemeColors(newThemeColor);
      applyTheme(colors);
    },
    { immediate: true }
  );
  // 设置更改函数
  const settingsMap: Record<string, Ref<SettingsValue>> = {
    tagsView,
    sidebarLogo,
    layout,
    watermarkEnabled,
  };

  function changeSetting({ key, value }: { key: string; value: SettingsValue }) {
    const setting = settingsMap[key];
    if (setting) setting.value = value;
  }

  function changeTheme(val: string) {
    theme.value = val;
  }

  function changeThemeColor(color: string) {
    themeColor.value = color;
  }

  function changeLayout(val: string) {
    layout.value = val;
  }

  const loadData = async () => {
    const res = await defConfigService.GetConfig({
      site: 2,
    });
    data.value = {};
    res.data?.forEach((item: { key: string; value: string }) => {
      data.value[item.key] = item.value;
    });
  };

  const getData = (key: string): string | undefined => {
    return data.value[key];
  };

  return {
    settingsVisible,
    tagsView,
    sidebarLogo,
    layout,
    themeColor,
    theme,
    watermarkEnabled,
    changeSetting,
    changeTheme,
    changeThemeColor,
    changeLayout,
    loadData,
    getData,
  };
});
