<!-- 商城服务 -->
<template>
  <div class="table-box">
    <ProTable ref="proTable" row-key="id" :columns="columns" :request-api="requestShopServiceTable">
      <template #tableHeader="{ selectedList }">
        <el-button v-hasPerm="['shop:service:create']" type="success" :icon="CirclePlus" @click="handleOpenDialog()">
          新增
        </el-button>
        <el-button
          v-hasPerm="['shop:service:delete']"
          type="danger"
          :icon="Delete"
          :disabled="!selectedList.length"
          @click="handleDelete(selectedList)"
        >
          删除
        </el-button>
      </template>

      <template #status="scope">
        <el-switch
          v-model="scope.row.status"
          inline-prompt
          :active-value="Status.ENABLE"
          :inactive-value="Status.DISABLE"
          active-text="启用"
          inactive-text="禁用"
          :disabled="!BUTTONS['shop:service:status']"
          :before-change="() => handleBeforeSetStatus(scope.row)"
        />
      </template>

      <template #operation="scope">
        <el-button
          v-hasPerm="['shop:service:update']"
          type="primary"
          link
          :icon="EditPen"
          @click="handleOpenDialog(scope.row.id)"
        >
          编辑
        </el-button>
        <el-button v-hasPerm="['shop:service:delete']" type="danger" link :icon="Delete" @click="handleDelete(scope.row)">
          删除
        </el-button>
      </template>
    </ProTable>

    <el-dialog v-model="dialog.visible" :title="dialog.title" width="500px" @close="handleCloseDialog">
      <ProForm ref="proFormRef" :model="formData" :fields="formFields" :rules="rules" label-width="100px" />
      <template #footer>
        <div class="dialog-footer">
          <el-button type="primary" @click="handleSubmit">确定</el-button>
          <el-button @click="handleCloseDialog">取消</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { CirclePlus, Delete, EditPen } from "@element-plus/icons-vue";
import type { ColumnProps, ProTableInstance } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
import ProForm from "@/components/ProForm/index.vue";
import type { ProFormField, ProFormInstance, ProFormOption } from "@/components/ProForm/interface";
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
const proFormRef = ref<ProFormInstance>();

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
  { prop: "label", label: "标签", search: { el: "input" } },
  { prop: "value", label: "值" },
  { prop: "sort", label: "排序", align: "right" },
  { prop: "status", label: "状态", width: 100, dictCode: "status", search: { el: "select" } },
  { prop: "createdAt", label: "创建时间", width: 180 },
  { prop: "updatedAt", label: "更新时间", width: 180 },
  { prop: "operation", label: "操作", width: 150, fixed: "right" }
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
  proFormRef.value?.resetFields();
  proFormRef.value?.clearValidate();
  formData.id = 0;
  formData.label = "";
  formData.value = "";
  formData.sort = 1;
  formData.status = Status.ENABLE;
}

/**
 * 打开商城服务弹窗。
 */
function handleOpenDialog(serviceId?: number) {
  dialog.visible = true;
  if (serviceId) {
    dialog.title = "修改商城服务";
    defShopServiceService.GetShopService({ value: serviceId }).then(data => {
      Object.assign(formData, data);
    });
    return;
  }

  dialog.title = "新增商城服务";
  resetForm();
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
  proFormRef.value?.validate()?.then(valid => {
    if (!valid) return;

    const submitData = JSON.parse(JSON.stringify(formData)) as ShopServiceForm;
    const request = submitData.id
      ? defShopServiceService.UpdateShopService(submitData)
      : defShopServiceService.CreateShopService(submitData);

    request.then(() => {
      ElMessage.success(submitData.id ? "修改成功" : "新增成功");
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
    await ElMessageBox.confirm(`是否确定${text}标签：${serviceName}？`, "提示", {
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
      ? `是否确定删除标签：${serviceList[0].label || serviceList[0].value || `ID:${serviceList[0].id}`}？`
      : `确认删除已选中的 ${serviceList.length} 项商城服务吗？`
    : "确认删除已选中的商城服务吗？";

  ElMessageBox.confirm(confirmMessage, "警告", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning"
  }).then(
    () => {
      defShopServiceService.DeleteShopService({ value: serviceIds }).then(() => {
        ElMessage.success("删除成功");
        refreshTable();
      });
    },
    () => {
      ElMessage.info("已取消删除");
    }
  );
}
</script>
