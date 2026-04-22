<template>
  <div class="table-box">
    <ProTable
      ref="proTable"
      title="菜单列表"
      row-key="id"
      :indent="20"
      :columns="columns"
      :header-actions="headerActions"
      :request-api="requestMenuTable"
      :pagination="false"
      :default-expand-all="false"
      :tree-props="{ children: 'children', hasChildren: 'hasChildren' }"
    />

    <FormDialog
      v-model="dialog.visible"
      ref="formDialogRef"
      :title="dialog.title"
      width="1180px"
      :model="formData"
      :fields="formFields"
      :rules="rules"
      label-width="120px"
      :col-span="12"
      @confirm="handleSubmit"
      @close="handleCloseDialog"
    >
      <template #menuIcon>
        <SelectIcon v-model:icon-value="menuIconValue" placeholder="请选择菜单图标" />
      </template>

      <template #apiTransferItem="slotScope">
        <el-popover effect="light" trigger="hover" placement="top" width="auto">
          <template #default>
            <div>操作名称：{{ slotScope.option.operation }}</div>
            <div>请求方式：{{ slotScope.option.method }}</div>
            <div>请求地址：{{ slotScope.option.path }}</div>
          </template>
          <template #reference>{{ slotScope.option.label }}</template>
        </el-popover>
      </template>
    </FormDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, h, reactive, ref, resolveComponent, resolveDynamicComponent } from "vue";
import type { FormRules } from "element-plus";
import ProTable from "@/components/ProTable/index.vue";
import FormDialog from "@/components/Dialog/FormDialog.vue";
import type { ColumnProps, HeaderActionProps, ProTableInstance, RenderScope } from "@/components/ProTable/interface";
import type { ProFormField, ProFormOption } from "@/components/ProForm/interface";
import SelectIcon from "@/components/SelectIcon/index.vue";
import { defBaseMenuService } from "@/api/admin/base_menu";
import { defBaseApiService } from "@/api/admin/base_api";
import { useAuthButtons } from "@/hooks/useAuthButtons";
import type { BaseApi } from "@/rpc/admin/base_api";
import type { BaseMenu, BaseMenuForm, BaseMenuMeta } from "@/rpc/admin/base_menu";
import type { TreeOptionResponse_Option } from "@/rpc/common/common";
import { BaseMenuType, Status } from "@/rpc/common/enum";
import { normalizeSelectedIds } from "@/utils/proTable";

defineOptions({
  name: "BaseMenu",
  inheritAttrs: false
});

/**
 * 菜单表单状态，统一补齐 meta 字段，便于 ProForm 直接双向绑定。
 */
type MenuFormState = Omit<BaseMenuForm, "meta"> & {
  /** 菜单元信息。 */
  meta: BaseMenuMeta;
};

const { BUTTONS } = useAuthButtons();
const proTable = ref<ProTableInstance>();
const formDialogRef = ref<InstanceType<typeof FormDialog>>();
const menuOptions = ref<TreeOptionResponse_Option[]>([]);
const apiList = ref<BaseApi[]>([]);

const dialog = reactive({
  title: "",
  visible: false
});

/** 创建默认菜单元信息。 */
function createDefaultMenuMeta(): BaseMenuMeta {
  return {
    title: "",
    icon: "",
    alwaysShow: false,
    hidden: false,
    keepAlive: false,
    full: false,
    affix: false,
    params: []
  };
}

/** 创建默认菜单表单。 */
function createDefaultMenuForm(): MenuFormState {
  return {
    id: 0,
    parentId: 0,
    type: BaseMenuType.FOLDER,
    path: "",
    name: "",
    component: "",
    redirect: "",
    meta: createDefaultMenuMeta(),
    api: [],
    sort: 1,
    status: Status.ENABLE
  };
}

const formData = reactive<MenuFormState>(createDefaultMenuForm());

/** 统一接管菜单图标字段，规避可选字段类型带来的模板告警。 */
const menuIconValue = computed({
  get: () => formData.meta.icon ?? "",
  set: value => {
    formData.meta.icon = value;
  }
});

const menuTypeOptions: ProFormOption[] = [
  { label: "目录", value: BaseMenuType.FOLDER },
  { label: "菜单", value: BaseMenuType.MENU },
  { label: "按钮", value: BaseMenuType.BUTTON },
  { label: "外链", value: BaseMenuType.EXT_LINK }
];

const statusOptions: ProFormOption[] = [
  { label: "启用", value: Status.ENABLE },
  { label: "禁用", value: Status.DISABLE }
];

/**
 * 渲染菜单图标单元格，统一兼容 Element Plus 图标和本地 svg 图标。
 */
function renderMenuIconCell(scope: RenderScope<BaseMenu>) {
  const icon = scope.row.meta?.icon;
  const iconName = resolveElementIcon(icon);
  if (iconName) {
    return h(
      resolveComponent("el-icon"),
      { size: 18 },
      {
        default: () => [h(resolveDynamicComponent(iconName) as any)]
      }
    );
  }
  if (icon) return h(resolveComponent("svg-icon"), { iconClass: icon });
  return "--";
}

/**
 * 渲染菜单显示状态标签，减少页面模板中的重复判断。
 */
function renderHiddenCell(scope: RenderScope<BaseMenu>) {
  const isHidden = Boolean(scope.row.meta?.hidden);
  return h(resolveComponent("el-tag"), { type: isHidden ? "info" : "success" }, () => (isHidden ? "隐藏" : "显示"));
}

/** 菜单表格列配置。 */
const columns = computed<ColumnProps[]>(() => [
  { type: "selection", width: 55 },
  {
    prop: "meta.title",
    label: "菜单名称",
    minWidth: 220,
    align: "left",
    search: { el: "input", key: "title" }
  },
  { prop: "type", label: "菜单类型", minWidth: 120, dictCode: "base_menu_type", search: { el: "select" } },
  {
    prop: "meta.icon",
    label: "菜单图标",
    width: 90,
    render: scope => renderMenuIconCell(scope as unknown as RenderScope<BaseMenu>)
  },
  { prop: "path", label: "路由路径/权限标识", minWidth: 260, search: { el: "input" } },
  { prop: "name", label: "路由名称", minWidth: 180, search: { el: "input" } },
  { prop: "component", label: "组件路径", minWidth: 260 },
  { prop: "redirect", label: "重定向地址", minWidth: 220 },
  { prop: "sort", label: "排序", minWidth: 80, align: "right" },
  {
    prop: "status",
    label: "状态",
    width: 100,
    dictCode: "status",
    search: { el: "select" },
    cellType: "status",
    statusProps: {
      activeValue: Status.ENABLE,
      inactiveValue: Status.DISABLE,
      activeText: "启用",
      inactiveText: "禁用",
      disabled: () => !BUTTONS.value["base:menu:status"],
      beforeChange: scope => handleBeforeSetStatus(scope.row as BaseMenu)
    }
  },
  {
    prop: "meta.hidden",
    label: "显示状态",
    width: 100,
    render: scope => renderHiddenCell(scope as unknown as RenderScope<BaseMenu>)
  },
  { prop: "createdAt", label: "创建时间", minWidth: 180 },
  { prop: "updatedAt", label: "更新时间", minWidth: 180 },
  {
    prop: "operation",
    label: "操作",
    width: 220,
    fixed: "right",
    cellType: "actions",
    actions: [
      {
        label: "新增",
        type: "primary",
        link: true,
        icon: CirclePlus,
        hidden: () => !BUTTONS.value["base:menu:create"],
        onClick: scope => handleOpenDialog((scope.row as BaseMenu).id)
      },
      {
        label: "编辑",
        type: "primary",
        link: true,
        icon: EditPen,
        hidden: () => !BUTTONS.value["base:menu:update"],
        onClick: scope => handleOpenDialog(0, (scope.row as BaseMenu).id)
      },
      {
        label: "删除",
        type: "danger",
        link: true,
        icon: Delete,
        hidden: () => !BUTTONS.value["base:menu:delete"],
        onClick: scope => handleDeleteMenu(scope.row as BaseMenu)
      }
    ]
  }
]);

/** 菜单表格顶部按钮配置。 */
const headerActions = computed<HeaderActionProps[]>(() => [
  {
    label: "新增",
    type: "success",
    icon: CirclePlus,
    hidden: () => !BUTTONS.value["base:menu:create"],
    onClick: () => handleOpenDialog()
  },
  {
    label: "删除",
    type: "danger",
    icon: Delete,
    hidden: () => !BUTTONS.value["base:menu:delete"],
    disabled: scope => !scope.selectedList.length,
    onClick: scope => handleDeleteMenu(scope.selectedList as BaseMenu[])
  }
]);

/** API 穿梭框可选数据。 */
const transferData = computed<ProFormOption[]>(() => {
  return apiList.value.map(item => ({
    ...item,
    value: item.operation,
    label: `${item.serviceDesc}/${item.desc}`
  }));
});

/** 菜单表单字段配置。 */
const formFields = computed<ProFormField[]>(() => [
  {
    prop: "parentId",
    label: "上级菜单",
    component: "tree-select",
    options: menuOptions.value as unknown as ProFormOption[],
    props: {
      nodeKey: "value",
      props: { label: "label", children: "children" },
      checkStrictly: true,
      clearable: false,
      filterable: true,
      style: { width: "100%" }
    }
  },
  {
    prop: "type",
    label: "菜单类型",
    component: "radio-group",
    options: menuTypeOptions
  },
  {
    prop: "meta.title",
    label: formData.type === BaseMenuType.BUTTON ? "按钮名称" : "菜单标题",
    component: "input",
    itemProps: {
      required: true
    },
    props: () => ({
      placeholder: formData.type === BaseMenuType.BUTTON ? "请输入按钮名称" : "请输入菜单标题"
    })
  },
  {
    prop: "path",
    label: getPathFieldLabel(),
    component: "input",
    labelTooltip: getPathFieldTooltip(),
    itemProps: {
      required: true
    },
    props: () => ({
      placeholder: getPathFieldPlaceholder()
    })
  },
  {
    prop: "redirect",
    label: "跳转路由",
    component: "input",
    props: { placeholder: "请输入跳转路由" },
    visible: model => model.type === BaseMenuType.FOLDER
  },
  {
    prop: "meta.icon",
    label: "菜单图标",
    component: "slot",
    slotName: "menuIcon",
    itemProps: model => ({
      required: model.type !== BaseMenuType.BUTTON
    }),
    visible: model => model.type !== BaseMenuType.BUTTON
  },
  {
    prop: "name",
    label: "路由名称",
    component: "input",
    labelTooltip: "如果需要开启缓存，需保证页面 defineOptions 中的 name 与此处一致，建议使用驼峰。",
    itemProps: model => ({
      required: model.type === BaseMenuType.MENU
    }),
    props: { placeholder: "请输入路由名称" },
    visible: model => model.type === BaseMenuType.MENU
  },
  {
    prop: "component",
    label: "组件路径",
    component: "input",
    labelTooltip: "组件页面完整路径，相对于 src/views/，例如 base/user/index，缺省后缀 .vue。",
    itemProps: model => ({
      required: model.type === BaseMenuType.MENU
    }),
    props: { placeholder: "base/user/index" },
    visible: model => model.type === BaseMenuType.MENU
  },
  {
    prop: "meta.hidden",
    label: "是否隐藏",
    component: "switch",
    props: {
      inlinePrompt: true,
      activeText: "是",
      inactiveText: "否",
      activeValue: true,
      inactiveValue: false
    },
    visible: model => model.type !== BaseMenuType.BUTTON
  },
  {
    prop: "meta.alwaysShow",
    label: "始终显示",
    component: "switch",
    labelTooltip: "选择“是”，即使目录或菜单下只有一个子节点，也会显示父节点；选择“否”，若只有一个子节点则只显示子节点。",
    props: {
      inlinePrompt: true,
      activeText: "是",
      inactiveText: "否",
      activeValue: true,
      inactiveValue: false
    },
    visible: model => model.type === BaseMenuType.FOLDER || model.type === BaseMenuType.MENU
  },
  {
    prop: "meta.keepAlive",
    label: "缓存页面",
    component: "switch",
    props: {
      inlinePrompt: true,
      activeText: "开",
      inactiveText: "关",
      activeValue: true,
      inactiveValue: false
    },
    visible: model => model.type === BaseMenuType.MENU
  },
  {
    prop: "meta.full",
    label: "全屏模式",
    component: "switch",
    props: {
      inlinePrompt: true,
      activeText: "开",
      inactiveText: "关",
      activeValue: true,
      inactiveValue: false
    },
    visible: model => model.type === BaseMenuType.MENU
  },
  {
    prop: "meta.affix",
    label: "固定标签",
    component: "switch",
    props: {
      inlinePrompt: true,
      activeText: "开",
      inactiveText: "关",
      activeValue: true,
      inactiveValue: false
    },
    visible: model => model.type === BaseMenuType.MENU
  },
  {
    prop: "meta.params",
    label: "路由参数",
    component: "kv-list",
    labelTooltip: "组件页面使用 useRoute().query.参数名 获取路由参数值。",
    props: {
      addText: "添加路由参数",
      keyInputProps: { placeholder: "参数名" },
      valueInputProps: { placeholder: "参数值" }
    },
    itemProps: {
      class: "menu-form__params"
    },
    visible: model => model.type === BaseMenuType.MENU
  },
  {
    prop: "api",
    label: "API 列表",
    component: "transfer",
    slotName: "apiTransferItem",
    options: transferData.value,
    props: {
      filterable: true,
      titles: ["可选 API", "已选 API"]
    },
    visible: model => model.type === BaseMenuType.MENU || model.type === BaseMenuType.BUTTON,
    colSpan: 24
  },
  {
    prop: "sort",
    label: "排序",
    component: "input-number",
    itemProps: {
      required: true
    },
    props: {
      min: 1,
      controlsPosition: "right",
      precision: 0,
      step: 1,
      style: { width: "100%" }
    }
  },
  {
    prop: "status",
    label: "状态",
    component: "radio-group",
    itemProps: {
      required: true
    },
    options: statusOptions
  }
]);

const rules = computed<FormRules>(() => ({
  parentId: [{ required: true, message: "请选择上级菜单", trigger: "change" }],
  type: [{ required: true, message: "请选择菜单类型", trigger: "change" }],
  "meta.title": [
    {
      validator: (_rule, value, callback) => {
        if (value) return callback();
        callback(new Error(formData.type === BaseMenuType.BUTTON ? "请输入按钮名称" : "请输入菜单标题"));
      },
      trigger: "blur"
    }
  ],
  "meta.icon": [
    {
      validator: (_rule, value, callback) => {
        if (formData.type === BaseMenuType.BUTTON) return callback();
        if (value) return callback();
        callback(new Error("请选择菜单图标"));
      },
      trigger: "change"
    }
  ],
  path: [
    {
      validator: (_rule, value, callback) => {
        if (value) return callback();
        if (formData.type === BaseMenuType.BUTTON) return callback(new Error("请输入权限标识"));
        if (formData.type === BaseMenuType.EXT_LINK) return callback(new Error("请输入完整外链地址"));
        callback(new Error("请输入路由路径"));
      },
      trigger: "blur"
    }
  ],
  name: [
    {
      validator: (_rule, value, callback) => {
        if (formData.type !== BaseMenuType.MENU) return callback();
        if (value) return callback();
        callback(new Error("请输入路由名称"));
      },
      trigger: "blur"
    }
  ],
  component: [
    {
      validator: (_rule, value, callback) => {
        if (formData.type !== BaseMenuType.MENU) return callback();
        if (value) return callback();
        callback(new Error("请输入组件路径"));
      },
      trigger: "blur"
    }
  ],
  sort: [{ required: true, message: "请输入排序值", trigger: "blur" }],
  status: [{ required: true, message: "请选择状态", trigger: "change" }]
}));

/** 计算当前路径字段文案。 */
function getPathFieldLabel() {
  if (formData.type === BaseMenuType.BUTTON) return "权限标识";
  if (formData.type === BaseMenuType.EXT_LINK) return "外链地址";
  return "路由路径";
}

/** 计算当前路径字段占位文案。 */
function getPathFieldPlaceholder() {
  if (formData.type === BaseMenuType.BUTTON) return "请输入按钮权限标识";
  if (formData.type === BaseMenuType.EXT_LINK) return "请输入完整外链地址";
  if (formData.type === BaseMenuType.FOLDER) return "base";
  return "user";
}

/** 计算当前路径字段提示文案。 */
function getPathFieldTooltip() {
  if (formData.type === BaseMenuType.BUTTON) return "按钮类型菜单使用权限标识，例如 base:user:create。";
  if (formData.type === BaseMenuType.EXT_LINK) return "请输入完整可访问的外部链接地址。";
  return "目录通常以 / 开头，例如 /base；菜单项通常不带 /，例如 user。";
}

/** 判断当前图标是否为 Element Plus 图标。 */
function resolveElementIcon(icon?: string) {
  if (!icon) return "";
  if (icon.startsWith("el-icon-")) return icon.replace("el-icon-", "");
  if (/^[A-Z]/.test(icon)) return icon;
  return "";
}

/**
 * 将菜单接口返回的 API 字段统一转换为穿梭框可识别的 operation 列表。
 */
function normalizeMenuApiSelection(api?: unknown[]) {
  if (!Array.isArray(api)) return [];

  const apiOperationSet = new Set(apiList.value.map(item => item.operation));
  const apiIdMap = new Map(apiList.value.map(item => [String(item.id), item.operation]));

  return api
    .map(item => {
      if (typeof item === "string") {
        if (apiOperationSet.has(item)) return item;
        return apiIdMap.get(item) ?? "";
      }

      if (typeof item === "number") {
        return apiIdMap.get(String(item)) ?? "";
      }

      if (item && typeof item === "object") {
        const currentItem = item as Record<string, unknown>;
        if (typeof currentItem.operation === "string") return currentItem.operation;
        if (currentItem.id !== undefined) return apiIdMap.get(String(currentItem.id)) ?? "";
      }

      return "";
    })
    .filter((item, index, currentList) => item && currentList.indexOf(item) === index);
}

/** 将服务端菜单表单补齐为前端可编辑结构。 */
function normalizeMenuForm(data?: Partial<BaseMenuForm>): MenuFormState {
  const defaultForm = createDefaultMenuForm();
  const normalizedMeta = {
    ...createDefaultMenuMeta(),
    ...(data?.meta ?? {}),
    params: data?.meta?.params ?? []
  };

  return {
    ...defaultForm,
    ...data,
    parentId: data?.parentId ?? 0,
    type: data?.type ?? BaseMenuType.FOLDER,
    status: data?.status ?? Status.ENABLE,
    api: normalizeMenuApiSelection(data?.api),
    sort: data?.sort ?? 1,
    meta: normalizedMeta
  };
}

/** 重置当前表单数据和校验状态，避免新增与编辑切换时残留旧值。 */
function resetForm(data?: Partial<BaseMenuForm>) {
  formDialogRef.value?.resetFields();
  formDialogRef.value?.clearValidate();
  Object.assign(formData, normalizeMenuForm(data));
}

/** 构建带顶级菜单节点的菜单树选项。 */
function buildMenuOptions(options: TreeOptionResponse_Option[] = []) {
  return [
    {
      value: 0,
      label: "顶级菜单",
      disabled: false,
      children: options
    }
  ];
}

/** 根据菜单类型清理无效字段，避免提交脏数据。 */
function buildSubmitPayload(): BaseMenuForm {
  const payload = normalizeMenuForm(formData);
  payload.meta.params = (payload.meta.params ?? []).filter(item => item.key || item.value);

  if (payload.type === BaseMenuType.BUTTON) {
    payload.name = "";
    payload.component = "";
    payload.redirect = "";
    payload.meta.icon = "";
    payload.meta.alwaysShow = false;
    payload.meta.hidden = true;
    payload.meta.keepAlive = false;
    payload.meta.full = false;
    payload.meta.affix = false;
    payload.meta.params = [];
  }

  if (payload.type === BaseMenuType.FOLDER) {
    payload.name = "";
    payload.component = "Layout";
    payload.api = [];
    payload.meta.keepAlive = false;
    payload.meta.full = false;
    payload.meta.affix = false;
    payload.meta.params = [];
  }

  if (payload.type === BaseMenuType.EXT_LINK) {
    payload.name = "";
    payload.component = "";
    payload.redirect = "";
    payload.api = [];
    payload.meta.keepAlive = false;
    payload.meta.full = false;
    payload.meta.affix = false;
    payload.meta.params = [];
  }

  return payload;
}

/** 加载菜单树选项和 API 列表，确保弹窗打开时相关数据已可用。 */
async function loadDialogResources() {
  const [menuData, apiData] = await Promise.all([defBaseMenuService.OptionBaseMenu({}), defBaseApiService.ListBaseApi({})]);
  menuOptions.value = buildMenuOptions(menuData.list ?? []);
  apiList.value = apiData.list ?? [];
}

/** 根据关键字递归过滤菜单树，保留匹配节点及其父级。 */
function filterMenuTree(menuList: BaseMenu[], keywordMap: Record<string, string>) {
  const titleKeyword = (keywordMap.title ?? "").trim().toLowerCase();
  const nameKeyword = (keywordMap.name ?? "").trim().toLowerCase();
  const pathKeyword = (keywordMap.path ?? "").trim().toLowerCase();

  return menuList.reduce<BaseMenu[]>((result, item) => {
    const children = filterMenuTree(item.children ?? [], keywordMap);
    const title = item.meta?.title?.toLowerCase() ?? "";
    const name = item.name?.toLowerCase() ?? "";
    const path = item.path?.toLowerCase() ?? "";
    const matched =
      (!titleKeyword || title.includes(titleKeyword)) &&
      (!nameKeyword || name.includes(nameKeyword)) &&
      (!pathKeyword || path.includes(pathKeyword));

    if (!matched && !children.length) return result;

    result.push({
      ...item,
      children
    });
    return result;
  }, []);
}

/** 请求菜单表格数据，并按搜索条件过滤树形结构。 */
async function requestMenuTable(params: Record<string, string>) {
  const data = await defBaseMenuService.TreeBaseMenu({});
  const keywordMap = {
    title: params.title ?? "",
    name: params.name ?? "",
    path: params.path ?? ""
  };

  return {
    data: filterMenuTree(data.list ?? [], keywordMap)
  };
}

/** 刷新菜单表格。 */
function refreshTable() {
  proTable.value?.getTableList();
}

/**
 * 打开菜单弹窗。
 * parentId 为新增时的父节点 ID，menuId 为编辑时的菜单 ID。
 */
async function handleOpenDialog(parentId = 0, menuId?: number) {
  await loadDialogResources();
  resetForm(menuId ? undefined : { parentId });
  dialog.visible = true;

  if (menuId) {
    dialog.title = "修改菜单";
    const data = await defBaseMenuService.GetBaseMenu({ value: menuId });
    resetForm(data);
    return;
  }

  dialog.title = "新增菜单";
}

/** 提交菜单表单，并在成功后关闭弹窗、刷新表格。 */
async function handleSubmit() {
  const valid = await formDialogRef.value?.validate();
  if (!valid) return;

  const payload = buildSubmitPayload();
  if (payload.id > 0) {
    await defBaseMenuService.UpdateBaseMenu(payload);
    ElMessage.success("菜单更新成功");
  } else {
    await defBaseMenuService.CreateBaseMenu(payload);
    ElMessage.success("菜单创建成功");
  }

  handleCloseDialog();
  refreshTable();
}

/** 关闭菜单弹窗并显式重置表单与校验状态。 */
function handleCloseDialog() {
  dialog.visible = false;
  resetForm();
}

/**
 * 在菜单状态切换前先完成确认与接口调用，避免首屏渲染触发误操作。
 */
async function handleBeforeSetStatus(row: BaseMenu) {
  const nextStatus = row.status === Status.ENABLE ? Status.DISABLE : Status.ENABLE;
  const text = nextStatus === Status.ENABLE ? "启用" : "禁用";
  const menuName = row.meta?.title || row.name || row.path || `ID:${row.id}`;
  try {
    await ElMessageBox.confirm(`是否确定${text}菜单？\n菜单名称：${menuName}`, "提示", {
      confirmButtonText: "确认",
      cancelButtonText: "取消",
      type: "warning"
    });
    await defBaseMenuService.SetBaseMenuStatus({ id: row.id, status: nextStatus });
    ElMessage.success(`${text}成功`);
    refreshTable();
    return true;
  } catch {
    return false;
  }
}

/**
 * 删除菜单，兼容单条删除与批量删除。
 */
function handleDeleteMenu(selected?: number | string | Array<number | string> | BaseMenu | BaseMenu[]) {
  const menuList = Array.isArray(selected)
    ? (selected.filter(item => typeof item === "object") as BaseMenu[])
    : selected && typeof selected === "object"
      ? [selected as BaseMenu]
      : [];
  const menuIds = (
    menuList.length ? menuList.map(item => item.id) : normalizeSelectedIds(selected as number | string | Array<number | string>)
  ).join(",");
  if (!menuIds) {
    ElMessage.warning("请勾选删除项");
    return;
  }

  const singleMenuName = menuList[0]?.meta?.title || menuList[0]?.name || menuList[0]?.path || `ID:${menuList[0]?.id ?? ""}`;
  const confirmMessage = menuList.length
    ? menuList.length === 1
      ? `是否确定删除菜单？\n菜单名称：${singleMenuName}`
      : `确认删除已选中的 ${menuList.length} 个菜单项吗？`
    : "确认删除已选中的菜单项吗？";

  ElMessageBox.confirm(confirmMessage, "警告", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning"
  }).then(
    () => {
      defBaseMenuService.DeleteBaseMenu({ value: menuIds }).then(() => {
        ElMessage.success("删除菜单成功");
        refreshTable();
      });
    },
    () => {
      ElMessage.info("已取消删除菜单");
    }
  );
}
</script>

<style scoped lang="scss">
.table-box {
  height: 100%;
}

:deep(.menu-form__params .el-form-item__content) {
  align-items: flex-start;
}
</style>
