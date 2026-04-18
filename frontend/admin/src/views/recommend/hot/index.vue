<!-- 热门推荐 -->
<template>
  <div class="table-box">
    <ProTable ref="proTable" row-key="id" :columns="columns" :header-actions="headerActions" :request-api="requestShopHotTable" />

    <FormDialog
      v-model="dialog.visible"
      ref="formDialogRef"
      :title="dialog.title"
      width="1000px"
      :model="formData"
      :fields="formFields"
      :rules="rules"
      label-width="150px"
      @confirm="handleSubmit"
      @close="handleCloseDialog"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from "vue";
import { useRouter } from "vue-router";
import { ElMessage, ElMessageBox } from "element-plus";
import { CirclePlus, Delete, EditPen, List } from "@element-plus/icons-vue";
import type { ColumnProps, HeaderActionProps, ProTableInstance } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
import FormDialog from "@/components/Dialog/FormDialog.vue";
import type { ProFormField, ProFormOption } from "@/components/ProForm/interface";
import { useAuthButtons } from "@/hooks/useAuthButtons";
import { defShopHotService } from "@/api/admin/shop_hot";
import type { PageShopHotRequest, ShopHot, ShopHotForm } from "@/rpc/admin/shop_hot";
import { Status } from "@/rpc/common/enum";
import { buildPageRequest, normalizeSelectedIds } from "@/utils/proTable";
import { navigateTo } from "@/utils/router";

defineOptions({
  name: "ShopShopHot",
  inheritAttrs: false
});

const { BUTTONS } = useAuthButtons();
const proTable = ref<ProTableInstance>();
const formDialogRef = ref<InstanceType<typeof FormDialog>>();
const router = useRouter();

const dialog = reactive({
  title: "",
  visible: false
});

const formData = reactive<ShopHotForm>({
  /** 热门推荐ID */
  id: 0,
  /** 商城热门推荐标题 */
  title: "",
  /** 商城热门推荐名称 */
  desc: "",
  /** 轮播图 */
  banner: "",
  /** 图片 */
  picture: [],
  /** 排序 */
  sort: 1,
  /** 状态 */
  status: Status.ENABLE
});

const rules = computed(() => ({
  title: [{ required: true, message: "请输入热门推荐标题", trigger: "blur" }],
  desc: [{ required: true, message: "请输入热门推荐描述", trigger: "blur" }],
  banner: [{ required: true, message: "请上传轮播图", trigger: "blur" }],
  picture: [{ required: true, message: "请上传推荐图片", trigger: "blur" }],
  status: [{ required: true, message: "状态不能为空", trigger: "blur" }]
}));

const statusOptions: ProFormOption[] = [
  { label: "启用", value: Status.ENABLE },
  { label: "禁用", value: Status.DISABLE }
];

/** 热门推荐表单字段配置。 */
const formFields: ProFormField[] = [
  { prop: "title", label: "热门推荐标题", component: "input", props: { placeholder: "请输入热门推荐标题" } },
  { prop: "desc", label: "热门推荐描述", component: "input", props: { placeholder: "请输入热门推荐描述" } },
  { prop: "banner", label: "轮播图", component: "image-upload", props: { uploadType: "hot" } },
  { prop: "picture", label: "推荐图片", component: "images-upload", props: { limit: 2, uploadType: "hot" } },
  {
    prop: "sort",
    label: "排序",
    component: "input-number",
    props: { min: 1, precision: 0, step: 1, controlsPosition: "right", style: { width: "100%" } }
  },
  { prop: "status", label: "状态", component: "radio-group", options: statusOptions }
];

/** 热门推荐表格列配置。 */
const columns: ColumnProps[] = [
  { type: "selection", width: 55 },
  { prop: "title", label: "热门推荐标题", minWidth: 160, search: { el: "input" } },
  { prop: "desc", label: "热门推荐描述", minWidth: 180, search: { el: "input" } },
  { prop: "sort", label: "排序", minWidth: 90, align: "right" },
  {
    prop: "status",
    label: "状态",
    minWidth: 100,
    search: { el: "select" },
    cellType: "status",
    statusProps: {
      activeValue: Status.ENABLE,
      inactiveValue: Status.DISABLE,
      activeText: "启用",
      inactiveText: "禁用",
      disabled: () => !BUTTONS.value["shop:hot:status"],
      beforeChange: scope => handleBeforeSetStatus(scope.row as ShopHot)
    }
  },
  { prop: "createdAt", label: "创建时间", minWidth: 180 },
  { prop: "updatedAt", label: "更新时间", minWidth: 180 },
  {
    prop: "operation",
    label: "操作",
    width: 240,
    fixed: "right",
    cellType: "actions",
    actions: [
      {
        label: "推荐选项",
        type: "primary",
        link: true,
        icon: List,
        hidden: () => !BUTTONS.value["shop:hot:items"],
        onClick: scope => handleOpenShopHotItem(scope.row as ShopHot)
      },
      {
        label: "编辑",
        type: "primary",
        link: true,
        icon: EditPen,
        hidden: () => !BUTTONS.value["shop:hot:update"],
        params: scope => ({ hotId: scope.row.id }),
        onClick: (scope, params) => handleOpenDialog((params?.hotId as number | undefined) ?? (scope.row as ShopHot).id)
      },
      {
        label: "删除",
        type: "danger",
        link: true,
        icon: Delete,
        hidden: () => !BUTTONS.value["shop:hot:delete"],
        onClick: scope => handleDelete(scope.row as ShopHot)
      }
    ]
  }
];

/** 热门推荐顶部按钮配置。 */
const headerActions: HeaderActionProps[] = [
  {
    label: "新增",
    type: "success",
    icon: CirclePlus,
    hidden: () => !BUTTONS.value["shop:hot:create"],
    onClick: () => handleOpenDialog()
  },
  {
    label: "删除",
    type: "danger",
    icon: Delete,
    hidden: () => !BUTTONS.value["shop:hot:delete"],
    disabled: scope => !scope.selectedList.length,
    onClick: scope => handleDelete(scope.selectedList as ShopHot[])
  }
];

/**
 * 请求热门推荐列表，交给 ProTable 统一处理分页和筛选。
 */
async function requestShopHotTable(params: PageShopHotRequest) {
  const data = await defShopHotService.PageShopHot(buildPageRequest(params));
  return { data };
}

/**
 * 刷新热门推荐表格。
 */
function refreshTable() {
  proTable.value?.getTableList();
}

/**
 * 重置热门推荐表单，清理上次编辑残留。
 */
function resetForm() {
  formDialogRef.value?.resetFields();
  formDialogRef.value?.clearValidate();
  formData.id = 0;
  formData.title = "";
  formData.desc = "";
  formData.banner = "";
  formData.picture = [];
  formData.sort = 1;
  formData.status = Status.ENABLE;
}

/**
 * 打开热门推荐弹窗。
 */
function handleOpenDialog(hotId?: number) {
  resetForm();
  dialog.title = hotId ? "修改热门推荐" : "新增热门推荐";
  dialog.visible = true;
  if (!hotId) return;

  defShopHotService.GetShopHot({ value: hotId }).then(data => {
    Object.assign(formData, data);
  });
}

/**
 * 关闭热门推荐弹窗并恢复默认值。
 */
function handleCloseDialog() {
  dialog.visible = false;
  resetForm();
}

/**
 * 提交热门推荐表单。
 */
function handleSubmit() {
  formDialogRef.value?.validate()?.then(isValid => {
    if (!isValid) return;

    const submitData = JSON.parse(JSON.stringify(formData)) as ShopHotForm;
    const request = submitData.id ? defShopHotService.UpdateShopHot(submitData) : defShopHotService.CreateShopHot(submitData);
    request.then(() => {
      ElMessage.success(submitData.id ? "修改热门推荐成功" : "新增热门推荐成功");
      handleCloseDialog();
      refreshTable();
    });
  });
}

/**
 * 在热门推荐状态切换前先完成确认与接口调用，避免首屏渲染触发误操作。
 */
async function handleBeforeSetStatus(row: ShopHot) {
  const nextStatus = row.status === Status.ENABLE ? Status.DISABLE : Status.ENABLE;
  const text = nextStatus === Status.ENABLE ? "启用" : "禁用";
  const hotName = row.title || `ID:${row.id}`;
  try {
    await ElMessageBox.confirm(`是否确定${text}热门推荐？\n推荐标题：${hotName}`, "提示", {
      confirmButtonText: "确认",
      cancelButtonText: "取消",
      type: "warning"
    });
    await defShopHotService.SetShopHotStatus({ id: row.id, status: nextStatus });
    ElMessage.success(`${text}成功`);
    refreshTable();
    return true;
  } catch {
    return false;
  }
}

/**
 * 删除热门推荐，兼容单行删除与批量删除。
 */
function handleDelete(selected?: number | string | Array<number | string> | ShopHot | ShopHot[]) {
  const hotList = Array.isArray(selected)
    ? (selected.filter(item => typeof item === "object") as ShopHot[])
    : selected && typeof selected === "object"
      ? [selected as ShopHot]
      : [];
  const hotIds = (
    hotList.length ? hotList.map(item => item.id) : normalizeSelectedIds(selected as number | string | Array<number | string>)
  ).join(",");
  if (!hotIds) {
    ElMessage.warning("请勾选删除项");
    return;
  }

  const confirmMessage = hotList.length
    ? hotList.length === 1
      ? `是否确定删除热门推荐？\n推荐标题：${hotList[0].title || `ID:${hotList[0].id}`}`
      : `确认删除已选中的 ${hotList.length} 个热门推荐吗？`
    : "确认删除已选中的热门推荐吗？";

  ElMessageBox.confirm(confirmMessage, "警告", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning"
  }).then(
    () => {
      defShopHotService.DeleteShopHot({ value: hotIds }).then(() => {
        ElMessage.success("删除热门推荐成功");
        refreshTable();
      });
    },
    () => {
      ElMessage.info("已取消删除热门推荐");
    }
  );
}

/**
 * 打开热门推荐选项页面。
 */
function handleOpenShopHotItem(row: ShopHot) {
  navigateTo(router, "/recommend/hot-item", { hotId: row.id, title: `【${row.title}】热门推荐选项` });
}
</script>
