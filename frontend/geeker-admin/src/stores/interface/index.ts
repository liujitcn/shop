import type { RouteItem, UserInfo } from "@/rpc/admin/auth";
import type { ListBaseDictResponse_BaseDictItem } from "@/rpc/admin/base_dict";

export type LayoutType = "vertical" | "classic" | "transverse" | "columns";

export type AssemblySizeType = "large" | "default" | "small";

/* GlobalState */
export interface GlobalState {
  layout: LayoutType;
  assemblySize: AssemblySizeType;
  maximize: boolean;
  primary: string;
  isDark: boolean;
  isGrey: boolean;
  isWeak: boolean;
  asideInverted: boolean;
  headerInverted: boolean;
  isCollapse: boolean;
  accordion: boolean;
  watermark: boolean;
  breadcrumb: boolean;
  breadcrumbIcon: boolean;
  tabs: boolean;
  tabsIcon: boolean;
  footer: boolean;
}

/* UserState */
export interface UserState {
  token: string;
  refreshToken: string;
  tokenType: string;
  tokenExpiresAt: number;
  userInfo: UserInfo;
}

/* tabsMenuProps */
export interface TabsMenuProps {
  icon: string;
  title: string;
  path: string;
  name: string;
  close: boolean;
  isKeepAlive: boolean;
}

/* TabsState */
export interface TabsState {
  tabsMenuList: TabsMenuProps[];
}

/* AuthState */
export interface AuthState {
  routeName: string;
  authButtonList: {
    [key: string]: string[];
  };
  authMenuList: RouteItem[];
}

/* KeepAliveState */
export interface KeepAliveState {
  keepAliveName: string[];
}

/* DictState */
export interface DictState {
  dictionary: Record<string, ListBaseDictResponse_BaseDictItem[]>;
}
