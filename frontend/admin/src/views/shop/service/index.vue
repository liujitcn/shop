<!-- 商城服务 -->
<template>
  <div class="table-box">
    <ProTable
      ref="proTable"
      row-key="id"
      :columns="columns"
      :header-actions="headerActions"
      :request-api="requestShopServiceTable"
    />

    <FormDialog
      v-model="dialog.visible"
      ref="formDialogRef"
      :title="dialog.title"
      width="500px"
      :model="formData"
      :fields="formFields"
      :rules="rules"
      label-width="100px"
      @confirm="handleSubmit"
      @close="handleCloseDialog"
    />
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { CirclePlus, Delete, EditPen } from "@element-plus/icons-vue";
import type { ColumnProps, HeaderActionProps, ProTableInstance } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
import FormDialog from "@/components/Dialog/FormDialog.vue";
import type { ProFormField, ProFormOption } from "@/components/ProForm/interface";
import { useAuthButtons } from "@/hooks/useAuthButtons";
import { defShopServiceService } from "@/api/admin/shop_service";
import type { PageShopServiceRequest, ShopService, ShopServiceForm } from "@/rpc/admin/shop_service";
import { Status } from "@/rpc/common/enum";
import { buildPageRequest, normalizeSelectedIds } from "@/utils/proTable";

defineOptions({
  name: "ShopService",
  inheritAttrs: false
});

const { BUTTONS } = useAuthButtons();
const proTable = ref<ProTableInstance>();
const formDialogRef = ref<InstanceType<typeof FormDialog>>();

const dialog = reactive({
  title: "",
  visible: false
});

const formData = reactive<ShopServiceForm>({
  /** 商城服务ID */
  id: 0,
  /** 标签 */
  label: "",
  /** 值 */
  value: "",
  /** 排序 */
  sort: 1,
  /** 状态 */
  status: Status.ENABLE
});

const rules = reactive({
  label: [{ required: true, message: "标签不能为空", trigger: "blur" }],
  value: [{ required: true, message: "值不能为空", trigger: "blur" }]
});

const statusOptions: ProFormOption[] = [
  { label: "启用", value: Status.ENABLE },
  { label: "禁用", value: Status.DISABLE }
];

/** 商城服务表单字段配置。 */
const formFields: ProFormField[] = [
  { prop: "label", label: "标签", component: "input", props: { placeholder: "请输入标签" } },
  { prop: "value", label: "值", component: "textarea", props: { placeholder: "请输入值" } },
  {
    prop: "sort",
    label: "排序",
    component: "input-number",
    props: { min: 1, precision: 0, step: 1, controlsPosition: "right", style: { width: "100%" } }
  },
  { prop: "status", label: "状态", component: "radio-group", options: statusOptions }
];

/** 商城服务表格列配置。 */
const columns: ColumnProps[] = [
  { type: "selection", width: 55 },
  { prop: "label", label: "标签", minWidth: 120, search: { el: "input" } },
  { prop: "value", label: "值", minWidth: 200 },
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
      disabled: () => !BUTTONS.value["shop:service:status"],
      beforeChange: scope => handleBeforeSetStatus(scope.row as ShopService)
    }
  },
  { prop: "createdAt", label: "创建时间", minWidth: 180 },
  { prop: "updatedAt", label: "更新时间", minWidth: 180 },
  {
    prop: "operation",
    label: "操作",
    width: 150,
    fixed: "right",
    cellType: "actions",
    actions: [
      {
        label: "编辑",
        type: "primary",
        link: true,
        icon: EditPen,
        hidden: () => !BUTTONS.value["shop:service:update"],
        params: scope => ({ serviceId: scope.row.id }),
        onClick: (scope, params) => handleOpenDialog((params?.serviceId as number | undefined) ?? (scope.row as ShopService).id)
      },
      {
        label: "删除",
        type: "danger",
        link: true,
        icon: Delete,
        hidden: () => !BUTTONS.value["shop:service:delete"],
        onClick: scope => handleDelete(scope.row as ShopService)
      }
    ]
  }
];

/** 商城服务顶部按钮配置。 */
const headerActions: HeaderActionProps[] = [
  {
    label: "新增",
    type: "success",
    icon: CirclePlus,
    hidden: () => !BUTTONS.value["shop:service:create"],
    onClick: () => handleOpenDialog()
  },
  {
    label: "删除",
    type: "danger",
    icon: Delete,
    hidden: () => !BUTTONS.value["shop:service:delete"],
    disabled: scope => !scope.selectedList.length,
    onClick: scope => handleDelete(scope.selectedList as ShopService[])
  }
];

/**
 * 请求商城服务列表，分页与搜索参数交给 ProTable 统一管理。
 */
async function requestShopServiceTable(params: PageShopServiceRequest) {
  const data = await defShopServiceService.PageShopService(buildPageRequest(params));
  return { data };
}

/**
 * 刷新当前表格数据。
 */
function refreshTable() {
  proTable.value?.getTableList();
}

/**
 * 重置商城服务表单，避免编辑残留污染下次新增。
 */
function resetForm() {
  formDialogRef.value?.resetFields();
  formDialogRef.value?.clearValidate();
  formData.id = 0;
  formData.label = "";
  formData.value = "";
  formData.sort = 1;
  formData.status = Status.ENABLE;
}

/**
 * 打开商城服务弹窗，编辑前先清理旧表单避免闪烁旧数据。
 */
function handleOpenDialog(serviceId?: number) {
  resetForm();
  dialog.title = serviceId ? "修改商城服务" : "新增商城服务";
  dialog.visible = true;
  if (!serviceId) return;

  defShopServiceService.GetShopService({ value: serviceId }).then(data => {
    Object.assign(formData, data);
  });
}

/**
 * 关闭商城服务弹窗并清理表单。
 */
function handleCloseDialog() {
  dialog.visible = false;
  resetForm();
}

/**
 * 提交商城服务表单。
 */
function handleSubmit() {
  formDialogRef.value?.validate()?.then(valid => {
    if (!valid) return;

    // 提交前复制一份表单数据，避免请求期间继续编辑导致引用污染。
    const submitData = JSON.parse(JSON.stringify(formData)) as ShopServiceForm;
    const request = submitData.id
      ? defShopServiceService.UpdateShopService(submitData)
      : defShopServiceService.CreateShopService(submitData);

    request.then(() => {
      ElMessage.success(submitData.id ? "修改商城服务成功" : "新增商城服务成功");
      handleCloseDialog();
      refreshTable();
    });
  });
}

/**
 * 在商城服务状态切换前先完成确认与接口调用，避免首屏渲染触发误操作。
 */
async function handleBeforeSetStatus(row: ShopService) {
  const nextStatus = row.status === Status.ENABLE ? Status.DISABLE : Status.ENABLE;
  const text = nextStatus === Status.ENABLE ? "启用" : "禁用";
  const serviceName = row.label || row.value || `ID:${row.id}`;
  try {
    await ElMessageBox.confirm(`是否确定${text}商城服务？\n服务名称：${serviceName}`, "提示", {
      confirmButtonText: "确认",
      cancelButtonText: "取消",
      type: "warning"
    });
    await defShopServiceService.SetShopServiceStatus({ id: row.id, status: nextStatus });
    ElMessage.success(`${text}成功`);
    refreshTable();
    return true;
  } catch {
    return false;
  }
}

/**
 * 删除商城服务，兼容单行删除与批量删除。
 */
function handleDelete(selected?: number | string | Array<number | string> | ShopService | ShopService[]) {
  const serviceList = Array.isArray(selected)
    ? (selected.filter(item => typeof item === "object") as ShopService[])
    : selected && typeof selected === "object"
      ? [selected as ShopService]
      : [];
  const serviceIds = (
    serviceList.length
      ? serviceList.map(item => item.id)
      : normalizeSelectedIds(selected as number | string | Array<number | string>)
  ).join(",");
  if (!serviceIds) {
    ElMessage.warning("请勾选删除项");
    return;
  }

  const confirmMessage = serviceList.length
    ? serviceList.length === 1
      ? `是否确定删除商城服务？\n服务名称：${serviceList[0].label || serviceList[0].value || `ID:${serviceList[0].id}`}`
      : `确认删除已选中的 ${serviceList.length} 项商城服务吗？`
    : "确认删除已选中的商城服务吗？";

  ElMessageBox.confirm(confirmMessage, "警告", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning"
  }).then(
    () => {
      defShopServiceService.DeleteShopService({ value: serviceIds }).then(() => {
        ElMessage.success("删除商城服务成功");
        refreshTable();
      });
    },
    () => {
      ElMessage.info("已取消删除商城服务");
    }
  );
}
</script>
