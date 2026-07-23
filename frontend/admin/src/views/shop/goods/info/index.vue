<!-- 商品 -->
<template>
  <div class="main-box">
    <TreeFilter
      label="name"
      title="分类列表"
      :request-api="requestCategoryTreeFilter"
      :show-all="false"
      :default-value="categoryFilterValue"
      @change="changeTreeFilter"
    />

    <div class="table-box">
      <ProTable
        ref="proTable"
        :key="isDefaultTenant ? 'default-tenant' : 'normal-tenant'"
        row-key="id"
        :columns="columns"
        :header-actions="headerActions"
        :request-api="requestGoodsTable"
        :init-param="tableInitParam"
      >
        <template #name="scope">
          <el-link v-if="BUTTONS['goods:info:detail']" type="primary" @click.stop="handleOpenDetail(scope.row)">
            {{ scope.row.name }}
          </el-link>
          <span v-else>{{ scope.row.name }}</span>
        </template>
      </ProTable>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { ElMessage, ElMessageBox } from "element-plus";
import { CirclePlus, Delete, EditPen, List, Tickets } from "@element-plus/icons-vue";
import type { ColumnProps, EnumProps, HeaderActionProps, ProTableInstance } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
import TreeFilter from "@/components/TreeFilter/index.vue";
import { useAuthButtons } from "@/hooks/useAuthButtons";
import { defGoodsInfoService } from "@/api/shop/goods_info";
import { defGoodsCategoryService } from "@/api/shop/goods_category";
import { defTenantStoreService } from "@/api/shop/tenant_store";
import type { GoodsInfo, PageGoodsInfoRequest } from "@/rpc/shop/admin/v1/goods_info";
import type { TreeTenantStoreResponse_Option } from "@/rpc/shop/admin/v1/tenant_store";
import type { TreeOptionResponse_Option } from "@/rpc/common/v1/common";
import { GoodsStatus } from "@/rpc/shop/common/v1/enum";
import { useUserStore } from "@/stores/modules/user";
import { buildPageRequest, normalizeSelectedIds } from "@/utils/proTable";
import { navigateTo } from "@/utils/router";
import {
  buildTenantStoreDisplayMap,
  buildTenantStoreDisplayMapFromOptions,
  DEFAULT_TENANT_CODE,
  formatTenantStoreDisplay,
  parseTenantStoreTreeValue,
  transformTenantStoreTreeOptions,
  type TenantStoreDisplayInfo
} from "@/views/shop/utils/tenant";

defineOptions({
  name: "GoodsInfo",
  inheritAttrs: false
});

/** 商品列表左侧分类筛选树节点。 */
type CategoryFilterNode = {
  id: string;
  name: string;
  children?: CategoryFilterNode[];
};

/** 商品列表搜索参数。 */
type GoodsInfoSearchParams = PageGoodsInfoRequest & {
  /** 默认租户的租户门店树筛选值。 */
  tenant_store_tree_value?: string;
};

const { BUTTONS } = useAuthButtons();
const proTable = ref<ProTableInstance>();
const router = useRouter();
const route = useRoute();
const userStore = useUserStore();
const tenantStoreDisplayMap = ref(new Map<number, TenantStoreDisplayInfo>());
const categoryDisplayMap = ref(new Map<number, string>());

const initParam = reactive({
  tenant_id: undefined as number | undefined,
  tenant_store_id: undefined as number | undefined,
  tenant_store_tree_value: undefined as string | undefined,
  category_id: undefined as number | undefined,
  status: undefined as number | undefined,
  inventory_alert: undefined as number | undefined,
  price_alert: undefined as number | undefined
});

/** 只向表格注入有效的工作台初始筛选值，避免空值覆盖用户当前搜索。 */
const tableInitParam = computed(() => {
  return Object.fromEntries(Object.entries(initParam).filter(([, value]) => value !== undefined));
});
const categoryFilterValue = ref("");

// 多分类场景下分类名称会更长，适当放宽列表列宽避免首屏截断过早。
const goodsCategoryColumnMinWidth = 220;

/** 当前登录账号是否默认租户。 */
const isDefaultTenant = computed(() => userStore.userInfo.tenant_code === DEFAULT_TENANT_CODE);

/** 商品状态枚举，补齐系统门店禁用状态的后台展示。 */
const goodsStatusOptions: EnumProps[] = [
  { label: "上架", value: GoodsStatus.PUT_ON, tagType: "success" },
  { label: "下架", value: GoodsStatus.PULL_OFF, tagType: "info" },
  { label: "门店禁用", value: GoodsStatus.DISABLED_BY_STORE, tagType: "warning" }
];

/** 商品表格列配置。 */
const columns = computed<ColumnProps[]>(() => [
  { type: "selection", width: 55 },
  {
    prop: "tenant_store_id",
    label: isDefaultTenant.value ? "租户/门店" : "门店",
    minWidth: isDefaultTenant.value ? 220 : 160,
    showOverflowTooltip: true,
    render: scope => getTenantStoreText(scope.row as GoodsInfo),
    search: isDefaultTenant.value
      ? {
          el: "tree-select",
          key: "tenant_store_tree_value",
          order: 1,
          props: {
            clearable: true,
            filterable: true,
            checkStrictly: true,
            renderAfterExpand: false,
            placeholder: "请选择租户/门店",
            style: { width: "100%" }
          }
        }
      : {
          el: "select",
          key: "tenant_store_id",
          order: 1,
          props: {
            clearable: true,
            filterable: true,
            placeholder: "请选择门店",
            style: { width: "100%" }
          }
        },
    enum: isDefaultTenant.value ? requestTenantStoreTreeOptions : requestTenantStoreOptions
  },
  {
    prop: "picture",
    label: "商品主图",
    minWidth: 150,
    cellType: "image",
    imageProps: {
      previewWidth: 400,
      previewHeight: 400,
      width: 60,
      height: 60
    }
  },
  { prop: "name", label: "商品名称", minWidth: 200, search: { el: "input" } },
  {
    prop: "category_id",
    label: "分类",
    minWidth: goodsCategoryColumnMinWidth,
    showOverflowTooltip: true,
    render: scope => getCategoryText(scope.row as GoodsInfo)
  },
  { prop: "desc", label: "商品描述", minWidth: 200 },
  { prop: "inventory", label: "总库存", minWidth: 100, align: "right" },
  {
    prop: "inventory_alert",
    label: "库存预警",
    minWidth: 120,
    dictCode: "goods_inventory_alert",
    isShow: false,
    search: { el: "select" }
  },
  {
    prop: "price_alert",
    label: "价格异常",
    minWidth: 120,
    dictCode: "goods_price_alert",
    isShow: false,
    search: { el: "select" }
  },
  { prop: "init_sale_num", label: "初始销量", minWidth: 100, align: "right" },
  { prop: "real_sale_num", label: "真实销量", minWidth: 100, align: "right" },
  { prop: "price", label: "价格（元）", minWidth: 110, align: "right", cellType: "money" },
  { prop: "discount_price", label: "折扣价格（元）", minWidth: 130, align: "right", cellType: "money" },
  {
    prop: "status",
    label: "状态",
    minWidth: 100,
    enum: goodsStatusOptions,
    search: { el: "select" },
    tag: true
  },
  { prop: "created_at", label: "创建时间", minWidth: 180 },
  { prop: "updated_at", label: "更新时间", minWidth: 180 },
  {
    prop: "operation",
    label: "操作",
    width: 300,
    fixed: "right",
    cellType: "actions",
    actions: [
      {
        label: "库存",
        type: "primary",
        link: true,
        icon: List,
        hidden: () => !BUTTONS.value["goods:info:sku"],
        onClick: scope => handleOpenSku(scope.row as GoodsInfo)
      },
      {
        label: "属性",
        type: "primary",
        link: true,
        icon: Tickets,
        hidden: () => !BUTTONS.value["goods:info:prop"],
        onClick: scope => handleOpenProp(scope.row as GoodsInfo)
      },
      {
        label: "编辑",
        type: "primary",
        link: true,
        icon: EditPen,
        hidden: () => !BUTTONS.value["goods:info:update"],
        onClick: scope => handleOpenDialog(scope.row as GoodsInfo)
      },
      {
        label: "上架",
        type: "success",
        link: true,
        hidden: scope => !canManualSetStatus(scope.row as GoodsInfo, GoodsStatus.PUT_ON),
        onClick: scope => handleSetStatus(scope.row as GoodsInfo, GoodsStatus.PUT_ON)
      },
      {
        label: "下架",
        type: "warning",
        link: true,
        hidden: scope => !canManualSetStatus(scope.row as GoodsInfo, GoodsStatus.PULL_OFF),
        onClick: scope => handleSetStatus(scope.row as GoodsInfo, GoodsStatus.PULL_OFF)
      },
      {
        label: "删除",
        type: "danger",
        link: true,
        icon: Delete,
        hidden: () => !BUTTONS.value["goods:info:delete"],
        onClick: scope => handleDelete(scope.row as GoodsInfo)
      }
    ]
  }
]);

/** 商品顶部按钮配置。 */
const headerActions: HeaderActionProps[] = [
  {
    label: "新增",
    type: "success",
    icon: CirclePlus,
    hidden: () => !BUTTONS.value["goods:info:create"],
    onClick: () => handleOpenDialog()
  },
  {
    label: "删除",
    type: "danger",
    icon: Delete,
    hidden: () => !BUTTONS.value["goods:info:delete"],
    disabled: scope => !scope.selectedList.length,
    onClick: scope => handleDelete(scope.selectedList as GoodsInfo[])
  }
];

/**
 * 递归转换分类树筛选数据，适配 TreeFilter 组件字段。
 */
function transformCategoryFilterNodes(options: TreeOptionResponse_Option[] = []): CategoryFilterNode[] {
  return options.map(option => ({
    id: String(option.value),
    name: option.label,
    children: transformCategoryFilterNodes(option.children ?? [])
  }));
}

/**
 * 构建分类编号到完整分类路径的映射，供商品列表按分类编号还原展示文本。
 */
function buildCategoryDisplayMap(options: TreeOptionResponse_Option[] = [], parentPath = "") {
  const result = new Map<number, string>();
  for (const option of options) {
    const label = option.label || "";
    const fullPath = [parentPath, label].filter(Boolean).join("/");
    result.set(option.value, fullPath);
    for (const [id, path] of buildCategoryDisplayMap(option.children ?? [], fullPath)) {
      result.set(id, path);
    }
  }
  return result;
}

/**
 * 请求分类树筛选数据。
 */
async function requestCategoryTreeFilter() {
  const response = await defGoodsCategoryService.OptionGoodsCategory({});
  categoryDisplayMap.value = buildCategoryDisplayMap(response.list ?? []);
  return {
    data: transformCategoryFilterNodes(response.list ?? [])
  };
}

/**
 * 根据商品分类编号还原完整分类名称，多个分类按商品配置顺序展示。
 */
function getCategoryText(row: GoodsInfo) {
  const categoryIDs = Array.isArray(row.category_id) ? row.category_id : [];
  const categoryNames = categoryIDs
    .map(categoryID => categoryDisplayMap.value.get(categoryID))
    .filter((name): name is string => Boolean(name));
  return categoryNames.join("、") || "--";
}

/**
 * 请求租户门店树筛选数据。
 */
async function requestTenantStoreTreeOptions() {
  const response = await defTenantStoreService.TreeTenantStore({ keyword: "" });
  tenantStoreDisplayMap.value = buildTenantStoreDisplayMap(response.list ?? []);
  return { data: transformTenantStoreTreeOptions(response.list ?? []) };
}

/** 请求普通租户的门店下拉筛选数据。 */
async function requestTenantStoreOptions() {
  const response = await defTenantStoreService.OptionTenantStore({ keyword: "" });
  tenantStoreDisplayMap.value = buildTenantStoreDisplayMapFromOptions(response.list ?? []);
  return { data: response.list ?? [] };
}

/**
 * 读取商品列表租户门店展示文本，默认租户显示租户/门店。
 */
function getTenantStoreText(row: GoodsInfo) {
  return formatTenantStoreDisplay(row.tenant_store_id, tenantStoreDisplayMap.value);
}

/**
 * 切换分类树筛选时同步更新表格查询参数。
 */
function changeTreeFilter(value: string) {
  categoryFilterValue.value = value ?? "";
  initParam.category_id = value ? Number(value) : undefined;
  if (proTable.value) {
    proTable.value.pageable.page_num = 1;
    proTable.value.search();
  }
}

/**
 * 请求商品分页列表，并统一处理分页参数。
 */
async function requestGoodsTable(params: PageGoodsInfoRequest) {
  const searchParams = params as GoodsInfoSearchParams;
  // 默认租户按树节点解析租户或门店，普通租户直接传下拉选择的门店编号。
  const tenantStoreSelection = isDefaultTenant.value
    ? parseTenantStoreTreeValue(searchParams.tenant_store_tree_value ?? initParam.tenant_store_tree_value)
    : { tenant_store_id: searchParams.tenant_store_id };
  const {
    tenant_store_tree_value: _tenantStoreTreeValue,
    tenant_id: _tenantId,
    tenant_store_id: _tenantStoreId,
    ...requestParams
  } = searchParams;
  const data = await defGoodsInfoService.PageGoodsInfo(
    buildPageRequest({
      ...requestParams,
      tenant_id: tenantStoreSelection.tenant_id ?? initParam.tenant_id,
      tenant_store_id: tenantStoreSelection.tenant_store_id ?? initParam.tenant_store_id,
      category_id: initParam.category_id,
      // 路由状态只作为首屏默认值，用户搜索选择后优先使用搜索表单值。
      status: searchParams.status ?? initParam.status,
      inventory_alert: searchParams.inventory_alert ?? initParam.inventory_alert,
      price_alert: searchParams.price_alert ?? initParam.price_alert
    })
  );
  return { data: { list: data.goods_infos ?? [], total: data.total } };
}

/** 刷新商品表格。 */
function refreshTable() {
  proTable.value?.getTableList();
}

/** 同步工作台跳转携带的商品列表筛选参数。 */
function syncWorkspaceQuery() {
  const categoryId = Number(route.query.categoryId ?? 0);
  const status = Number(route.query.status ?? 0);
  const inventoryAlert = Number(route.query.inventory_alert ?? 0);
  const priceAlert = Number(route.query.price_alert ?? 0);

  initParam.category_id = categoryId > 0 ? categoryId : undefined;
  initParam.status = status > 0 ? status : undefined;
  initParam.inventory_alert = inventoryAlert > 0 ? inventoryAlert : undefined;
  initParam.price_alert = priceAlert > 0 ? priceAlert : undefined;
  categoryFilterValue.value = initParam.category_id ? String(initParam.category_id) : "";

  if (proTable.value) {
    Object.assign(proTable.value.searchParam, {
      category_id: initParam.category_id,
      tenant_id: initParam.tenant_id,
      tenant_store_id: initParam.tenant_store_id,
      tenant_store_tree_value: initParam.tenant_store_tree_value,
      status: initParam.status,
      inventory_alert: initParam.inventory_alert,
      price_alert: initParam.price_alert
    });
    Object.assign(proTable.value.searchInitParam, {
      category_id: initParam.category_id,
      tenant_id: initParam.tenant_id,
      tenant_store_id: initParam.tenant_store_id,
      tenant_store_tree_value: initParam.tenant_store_tree_value,
      status: initParam.status,
      inventory_alert: initParam.inventory_alert,
      price_alert: initParam.price_alert
    });
  }
}

watch(
  () => route.query,
  () => {
    syncWorkspaceQuery();
    if (proTable.value) {
      proTable.value.pageable.page_num = 1;
      proTable.value.search();
    }
  },
  { immediate: true }
);

watch(
  () => proTable.value,
  value => {
    if (!value) return;
    syncWorkspaceQuery();
  },
  { immediate: true }
);

/**
 * 打开商品编辑页。
 */
function handleOpenDialog(row?: GoodsInfo) {
  if (row?.id) {
    // 编辑页标题固定为“商品编辑”，跳转时不再额外携带商品名称。
    navigateTo(router, "/goods/info/edit", { goodsId: row.id });
    return;
  }

  navigateTo(router, "/goods/info/edit");
}

/** 判断当前商品是否允许通过人工入口切换为指定状态。 */
function canManualSetStatus(row: GoodsInfo, status: GoodsStatus) {
  if (!BUTTONS.value["goods:info:status"]) return false;
  if (row.status === GoodsStatus.DISABLED_BY_STORE) return false;
  return row.status !== status;
}

/** 人工切换商品上下架状态，门店禁用状态只由审核流程维护。 */
async function handleSetStatus(row: GoodsInfo, status: GoodsStatus) {
  const text = status === GoodsStatus.PUT_ON ? "上架" : "下架";
  const goodsName = row.name || `ID:${row.id}`;
  try {
    await ElMessageBox.confirm(`是否确定${text}商品？\n商品名称：${goodsName}`, "提示", {
      confirmButtonText: "确认",
      cancelButtonText: "取消",
      type: "warning"
    });
    await defGoodsInfoService.SetGoodsInfoStatus({ id: row.id, status });
    ElMessage.success(`${text}成功`);
    refreshTable();
  } catch {
    // 用户取消确认时不需要额外提示。
  }
}

/**
 * 删除商品，兼容单条删除与批量删除。
 */
function handleDelete(selected?: number | string | Array<number | string> | GoodsInfo | GoodsInfo[] | { [key: string]: any }[]) {
  const goodsInfoList = Array.isArray(selected)
    ? (selected.filter(item => typeof item === "object") as GoodsInfo[])
    : selected && typeof selected === "object"
      ? [selected as GoodsInfo]
      : [];
  const goodsIds = (
    goodsInfoList.length
      ? goodsInfoList.map(item => item.id)
      : normalizeSelectedIds(selected as number | string | Array<number | string>)
  ).join(",");
  if (!goodsIds) {
    ElMessage.warning("请勾选删除项");
    return;
  }

  const confirmMessage = goodsInfoList.length
    ? goodsInfoList.length === 1
      ? `是否确定删除商品？\n商品名称：${goodsInfoList[0].name || `ID:${goodsInfoList[0].id}`}`
      : `确认删除已选中的 ${goodsInfoList.length} 个商品吗？`
    : "确认删除已选中的商品吗？";

  ElMessageBox.confirm(confirmMessage, "警告", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning"
  }).then(
    () => {
      defGoodsInfoService.DeleteGoodsInfo({ ids: goodsIds }).then(() => {
        ElMessage.success("删除商品成功");
        proTable.value?.search();
      });
    },
    () => {
      ElMessage.info("已取消删除商品");
    }
  );
}

/**
 * 打开商品属性页。
 */
function handleOpenProp(row: GoodsInfo) {
  navigateTo(router, "/goods/prop", { goodsId: row.id, title: `【${row.name}】商品属性` });
}

/**
 * 打开商品规格页。
 */
function handleOpenSku(row: GoodsInfo) {
  navigateTo(router, "/goods/sku", { goodsId: row.id, title: `【${row.name}】商品规格` });
}

/**
 * 打开商品详情页。
 */
function handleOpenDetail(row: GoodsInfo) {
  // 商品详情页与订单详情统一改为路径参数传递商品ID，避免继续使用查询参数。
  navigateTo(router, `/goods/info/detail/${row.id}`);
}
</script>
