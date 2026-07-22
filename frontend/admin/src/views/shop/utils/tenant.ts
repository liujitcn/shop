import type { EnumProps } from "@/components/ProTable/interface";
import type { OptionTenantStoreResponse_Option, TreeTenantStoreResponse_Option } from "@/rpc/shop/admin/v1/tenant_store";

export { DEFAULT_TENANT_CODE, requestTenantOptions } from "@/utils/tenant";

/** 租户门店树筛选解析结果。 */
export type TenantStoreTreeSelection = {
  /** 租户ID，选中一级租户节点时传入。 */
  tenant_id?: number;
  /** 门店ID，选中二级门店节点时传入。 */
  tenant_store_id?: number;
};

/** 商品、订单、评论列表门店反查展示信息。 */
export type TenantStoreDisplayInfo = {
  /** 所属租户名称。 */
  tenantName: string;
  /** 门店名称。 */
  storeName: string;
};

/** 递归转换租户门店树筛选选项，适配 ProTable 搜索枚举结构。 */
export function transformTenantStoreTreeOptions(options: TreeTenantStoreResponse_Option[] = []): EnumProps[] {
  return options.map(option => ({
    label: option.label,
    value: option.value,
    children: transformTenantStoreTreeOptions(option.children ?? [])
  }));
}

/** 从租户门店树构建门店展示映射，列表可按门店编号反查租户和门店名称。 */
export function buildTenantStoreDisplayMap(options: TreeTenantStoreResponse_Option[] = []) {
  const displayMap = new Map<number, TenantStoreDisplayInfo>();
  options.forEach(option => {
    if (option.type === "store") {
      displayMap.set(option.id, {
        tenantName: "",
        storeName: option.label
      });
      return;
    }

    const tenantName = option.label;
    option.children?.forEach(storeOption => {
      if (storeOption.type !== "store") return;
      displayMap.set(storeOption.id, {
        tenantName,
        storeName: storeOption.label
      });
    });
  });
  return displayMap;
}

/** 从门店下拉选项构建门店展示映射。 */
export function buildTenantStoreDisplayMapFromOptions(options: OptionTenantStoreResponse_Option[] = []) {
  const displayMap = new Map<number, TenantStoreDisplayInfo>();
  options.forEach(option => {
    displayMap.set(option.value, {
      tenantName: "",
      storeName: option.label
    });
  });
  return displayMap;
}

/** 格式化租户门店展示文案，默认租户显示“租户/门店”，普通租户显示门店。 */
export function formatTenantStoreDisplay(tenantStoreId: number | undefined, displayMap: Map<number, TenantStoreDisplayInfo>) {
  if (!tenantStoreId) return "-";
  const displayInfo = displayMap.get(tenantStoreId);
  if (!displayInfo) return "-";
  return [displayInfo.tenantName, displayInfo.storeName].filter(Boolean).join("/") || "-";
}

/** 解析租户门店树选中值，一级租户查租户数据，二级门店查门店数据。 */
export function parseTenantStoreTreeValue(value?: string): TenantStoreTreeSelection {
  if (!value) return {};
  const [type, rawId] = value.split(":");
  const id = Number(rawId);
  if (!Number.isFinite(id) || id <= 0) return {};
  if (type === "tenant") return { tenant_id: id };
  if (type === "store") return { tenant_store_id: id };
  return {};
}
