<!-- 租户门店 -->
<template>
  <div class="table-box">
    <ProTable
      ref="proTable"
      :key="isDefaultTenant ? 'default-tenant' : 'normal-tenant'"
      row-key="id"
      :columns="columns"
      :header-actions="headerActions"
      :request-api="requestTenantStoreTable"
      :init-param="initParam"
    />

    <FormDialog
      v-model="dialog.visible"
      ref="formDialogRef"
      :title="dialog.title"
      width="620px"
      :model="formData"
      :fields="formFields"
      :rules="rules"
      label-width="100px"
      @confirm="handleSubmit"
      @close="handleCloseDialog"
    />

    <ProDialog v-model="auditDialog.visible" :title="auditDialog.title" width="960px" @close="handleCloseAuditDialog">
      <el-descriptions border :column="2">
        <el-descriptions-item label="门店名称">
          {{ auditDetail.name || "--" }}
        </el-descriptions-item>
        <el-descriptions-item v-if="isDefaultTenant" label="所属租户">
          {{ auditTenantName || "--" }}
        </el-descriptions-item>
        <el-descriptions-item label="门店LOGO">
          <el-image
            v-if="auditDetail.logo"
            class="store-preview-image"
            :src="auditDetail.logo"
            :preview-src-list="[auditDetail.logo]"
            fit="cover"
            preview-teleported
          />
          <span v-else>--</span>
        </el-descriptions-item>
        <el-descriptions-item label="门店封面">
          <el-image
            v-if="auditDetail.cover"
            class="store-preview-image store-preview-image--cover"
            :src="auditDetail.cover"
            :preview-src-list="[auditDetail.cover]"
            fit="cover"
            preview-teleported
          />
          <span v-else>--</span>
        </el-descriptions-item>
        <el-descriptions-item label="门店简介">
          {{ auditDetail.intro || "--" }}
        </el-descriptions-item>
        <el-descriptions-item label="门店公告">
          {{ auditDetail.notice || "--" }}
        </el-descriptions-item>
        <el-descriptions-item label="营业执照" :span="2">
          <div v-if="auditDetail.business_license.length" class="store-license-list">
            <el-image
              v-for="(image, index) in auditDetail.business_license"
              :key="`${image}-${index}`"
              class="store-preview-image"
              :src="image"
              :preview-src-list="auditDetail.business_license"
              fit="cover"
              preview-teleported
            />
          </div>
          <span v-else>--</span>
        </el-descriptions-item>
        <el-descriptions-item label="当前备注" :span="2">
          {{ auditDetail.remark || "--" }}
        </el-descriptions-item>
      </el-descriptions>

      <ProForm
        ref="auditFormRef"
        class="store-audit-form"
        :model="auditFormData"
        :fields="auditFields"
        :rules="auditRules"
        label-width="100px"
      />

      <template #footer>
        <div class="dialog-footer">
          <el-button type="primary" @click="handleAuditSubmit">确 定</el-button>
          <el-button @click="handleCloseAuditDialog">取 消</el-button>
        </div>
      </template>
    </ProDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { CircleCheck, CirclePlus, CloseBold, Delete, EditPen } from "@element-plus/icons-vue";
import type { ColumnProps, HeaderActionProps, ProTableInstance } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
import ProDialog from "@/components/Dialog/ProDialog.vue";
import FormDialog from "@/components/Dialog/FormDialog.vue";
import ProForm from "@/components/ProForm/index.vue";
import type { ProFormField, ProFormInstance } from "@/components/ProForm/interface";
import { useAuthButtons } from "@/hooks/useAuthButtons";
import { defTenantStoreService } from "@/api/shop/admin/tenant_store";
import type { AuditTenantStoreRequest, PageTenantStoreRequest, TenantStore, TenantStoreForm } from "@/rpc/shop/admin/v1/tenant_store";
import type { SelectOptionResponse_Option } from "@/rpc/common/v1/common";
import { TenantStoreStatus } from "@/rpc/shop/common/v1/enum";
import { buildPageRequest, normalizeSelectedIds } from "@/utils/proTable";
import { useUserStore } from "@/stores/modules/user";
import { DEFAULT_TENANT_CODE, requestTenantOptions } from "@/views/shop/admin/utils/tenant";

defineOptions({
  name: "TenantStore",
  inheritAttrs: false
});

const { BUTTONS } = useAuthButtons();
const userStore = useUserStore();
const proTable = ref<ProTableInstance>();
const formDialogRef = ref<InstanceType<typeof FormDialog>>();
const auditFormRef = ref<ProFormInstance>();
const tenantOptions = ref<SelectOptionResponse_Option[]>([]);

const initParam = reactive<PageTenantStoreRequest>({
  name: "",
  page_num: 1,
  page_size: 10
});

const dialog = reactive({
  title: "",
  visible: false
});

const auditDialog = reactive({
  title: "",
  visible: false
});

const formData = reactive<TenantStoreForm>(createDefaultFormData());
const auditDetail = ref<TenantStore>(createDefaultAuditDetail());
const auditFormData = reactive<AuditTenantStoreRequest>(createDefaultAuditFormData());

/** 当前登录账号是否默认租户。 */
const isDefaultTenant = computed(() => userStore.userInfo.tenant_code === DEFAULT_TENANT_CODE);

/** 审核详情中的租户名称。 */
const auditTenantName = computed(() => tenantOptions.value.find(item => item.value === auditDetail.value.tenant_id)?.label ?? "");

const rules = reactive({
  name: [
    { required: true, message: "请输入门店名称", trigger: "blur" },
    { max: 100, message: "门店名称不能超过 100 个字符", trigger: "blur" }
  ],
  logo: [
    { required: true, message: "请上传门店LOGO", trigger: "change" },
    { max: 1024, message: "门店LOGO不能超过 1024 个字符", trigger: "change" }
  ],
  cover: [
    { required: true, message: "请上传门店封面", trigger: "change" },
    { max: 1024, message: "门店封面不能超过 1024 个字符", trigger: "change" }
  ],
  intro: [
    { required: true, message: "请输入门店简介", trigger: "blur" },
    { max: 500, message: "门店简介不能超过 500 个字符", trigger: "blur" }
  ],
  notice: [{ max: 500, message: "门店公告不能超过 500 个字符", trigger: "blur" }],
  business_license: [{ required: true, message: "请上传营业执照", trigger: "change" }],
  remark: [{ max: 500, message: "备注不能超过 500 个字符", trigger: "blur" }]
});

const auditRules = reactive({
  status: [{ required: true, message: "审核结果不能为空", trigger: "change" }],
  remark: [{ required: true, message: "请输入拒绝原因", trigger: "blur" }]
});

/** 租户门店表单字段配置。 */
const formFields = computed<ProFormField[]>(() => [
  { prop: "name", label: "门店名称", component: "input", colSpan: 24, props: { placeholder: "请输入门店名称" } },
  {
    prop: "logo",
    label: "LOGO",
    component: "image-upload",
    colSpan: 24,
    props: { uploadType: "store", width: "140px", height: "140px" }
  },
  {
    prop: "cover",
    label: "封面",
    component: "image-upload",
    colSpan: 24,
    props: { uploadType: "store", width: "240px", height: "140px" }
  },
  {
    prop: "intro",
    label: "简介",
    component: "textarea",
    colSpan: 24,
    props: { rows: 3, maxlength: 200, showWordLimit: true, resize: "none" }
  },
  {
    prop: "notice",
    label: "公告",
    component: "textarea",
    colSpan: 24,
    props: { rows: 3, maxlength: 200, showWordLimit: true, resize: "none" }
  },
  {
    prop: "business_license",
    label: "营业执照",
    component: "images-upload",
    colSpan: 24,
    props: { uploadType: "store", width: "140px", height: "140px" }
  },
  {
    prop: "remark",
    label: "备注",
    component: "textarea",
    colSpan: 24,
    props: { rows: 3, maxlength: 500, showWordLimit: true, resize: "none" }
  }
]);

/** 租户门店审核表单字段配置。 */
const auditFields = computed<ProFormField[]>(() => [
  { prop: "status", label: "审核结果", component: "dict", props: { code: "tenant_store_status", type: "radio" } },
  {
    prop: "remark",
    label: "拒绝原因",
    component: "textarea",
    props: { rows: 3, maxlength: 500, showWordLimit: true, resize: "none", placeholder: "请输入拒绝原因" },
    visible: model => model.status === TenantStoreStatus.FAILED_REVIEW_TSS
  }
]);

/** 租户门店表格列配置。 */
const columns = computed<ColumnProps[]>(() => [
  { type: "selection", width: 55 },
  ...(isDefaultTenant.value
    ? ([
        {
          prop: "tenant_id",
          label: "租户",
          minWidth: 140,
          showOverflowTooltip: true,
          search: { el: "select", key: "tenant_id", props: { filterable: true }, order: 1 },
          enum: requestTenantOptions
        }
      ] satisfies ColumnProps[])
    : []),
  { prop: "name", label: "门店名称", minWidth: 160, search: { el: "input" }, showOverflowTooltip: true },
  {
    prop: "logo",
    label: "LOGO",
    minWidth: 100,
    cellType: "image",
    imageProps: { width: 48, height: 48, previewWidth: 300, previewHeight: 300 }
  },
  {
    prop: "cover",
    label: "封面",
    minWidth: 130,
    cellType: "image",
    imageProps: { width: 72, height: 48, previewWidth: 420, previewHeight: 260 }
  },
  {
    prop: "status",
    label: "审核状态",
    minWidth: 120,
    dictCode: "tenant_store_status",
    search: { el: "select" },
    dictValueType: "number"
  },
  { prop: "remark", label: "备注", minWidth: 160, showOverflowTooltip: true },
  { prop: "created_at", label: "创建时间", minWidth: 180 },
  { prop: "updated_at", label: "更新时间", minWidth: 180 },
  {
    prop: "operation",
    label: "操作",
    width: 230,
    fixed: "right",
    cellType: "actions",
    actions: [
      {
        label: "审核",
        type: "success",
        link: true,
        icon: CircleCheck,
        hidden: () => !BUTTONS.value["shop:store:audit"],
        onClick: scope => handleOpenAuditDialog(scope.row as TenantStore)
      },
      {
        label: "编辑",
        type: "primary",
        link: true,
        icon: EditPen,
        hidden: () => !BUTTONS.value["shop:store:update"],
        onClick: scope => handleOpenDialog((scope.row as TenantStore).id)
      },
      {
        label: "删除",
        type: "danger",
        link: true,
        icon: Delete,
        hidden: () => !BUTTONS.value["shop:store:delete"],
        onClick: scope => handleDelete(scope.row as TenantStore)
      }
    ]
  }
]);

/** 租户门店表格顶部按钮配置。 */
const headerActions: HeaderActionProps[] = [
  {
    label: "新增",
    type: "success",
    icon: CirclePlus,
    hidden: () => !BUTTONS.value["shop:store:create"],
    onClick: () => handleOpenDialog()
  },
  {
    label: "删除",
    type: "danger",
    icon: Delete,
    hidden: () => !BUTTONS.value["shop:store:delete"],
    disabled: scope => !scope.selectedList.length,
    onClick: scope => handleDelete(scope.selectedList as TenantStore[])
  }
];

/** 创建门店表单默认值。 */
function createDefaultFormData(): TenantStoreForm {
  return {
    id: 0,
    name: "",
    logo: "",
    cover: "",
    intro: "",
    notice: "",
    business_license: [],
    status: TenantStoreStatus.UNKNOWN_TSS,
    remark: ""
  };
}

/** 创建租户门店审核详情默认值。 */
function createDefaultAuditDetail(): TenantStore {
  return {
    id: 0,
    tenant_id: 0,
    name: "",
    logo: "",
    cover: "",
    intro: "",
    notice: "",
    business_license: [],
    status: TenantStoreStatus.UNKNOWN_TSS,
    remark: "",
    created_at: "",
    updated_at: ""
  };
}

/** 创建租户门店审核表单默认值。 */
function createDefaultAuditFormData(): AuditTenantStoreRequest {
  return {
    id: 0,
    status: TenantStoreStatus.UNKNOWN_TSS,
    remark: ""
  };
}

/** 规范化枚举筛选值。 */
function normalizeEnumFilter(value: unknown) {
  if (value === undefined || value === null || value === "") return undefined;
  const numberValue = Number(value);
  return Number.isFinite(numberValue) && numberValue > 0 ? numberValue : undefined;
}

/** 加载租户下拉选项。 */
async function loadTenantOptions() {
  if (!isDefaultTenant.value || tenantOptions.value.length) return;
  const response = await requestTenantOptions();
  tenantOptions.value = response.data ?? [];
}

/** 请求租户门店列表，并由 ProTable 统一管理分页和筛选。 */
async function requestTenantStoreTable(params: PageTenantStoreRequest) {
  await loadTenantOptions();
  const requestParams = buildPageRequest(params);
  requestParams.status = normalizeEnumFilter(requestParams.status) as PageTenantStoreRequest["status"];
  if (!isDefaultTenant.value) {
    requestParams.tenant_id = undefined;
  }
  const data = await defTenantStoreService.PageTenantStore(requestParams);
  return { data: { list: data.tenant_stores ?? [], total: data.total } };
}

/** 刷新租户门店表格。 */
function refreshTable() {
  proTable.value?.getTableList();
}

/** 重置租户门店表单。 */
function resetForm() {
  formDialogRef.value?.resetFields();
  formDialogRef.value?.clearValidate();
  Object.assign(formData, createDefaultFormData());
}

/** 重置租户门店审核弹窗数据。 */
function resetAuditDialog() {
  auditFormRef.value?.resetFields();
  auditFormRef.value?.clearValidate();
  auditDetail.value = createDefaultAuditDetail();
  Object.assign(auditFormData, createDefaultAuditFormData());
}

/** 打开租户门店弹窗。 */
async function handleOpenDialog(storeId?: number) {
  dialog.visible = true;
  if (storeId) {
    dialog.title = "修改门店";
    const data = await defTenantStoreService.GetTenantStore({ id: storeId });
    Object.assign(formData, createDefaultFormData(), data);
    return;
  }

  dialog.title = "新增门店";
  resetForm();
}

/** 打开租户门店审核弹窗。 */
function handleOpenAuditDialog(row: TenantStore) {
  resetAuditDialog();
  auditDialog.title = "租户门店审核";
  auditDialog.visible = true;
  auditDetail.value = {
    ...createDefaultAuditDetail(),
    ...row,
    business_license: Array.isArray(row.business_license) ? row.business_license : []
  };
  auditFormData.id = row.id;
  auditFormData.status = row.status;
  auditFormData.remark = row.remark;
}

/** 提交租户门店表单。 */
function handleSubmit() {
  formDialogRef.value?.validate()?.then(valid => {
    if (!valid) return;
    const request = formData.id
      ? defTenantStoreService.UpdateTenantStore({ id: formData.id, tenant_store: formData })
      : defTenantStoreService.CreateTenantStore({ tenant_store: formData });
    request.then(() => {
      ElMessage.success(formData.id ? "修改门店成功" : "新增门店成功");
      handleCloseDialog();
      refreshTable();
    });
  });
}

/** 关闭租户门店弹窗并恢复默认表单值。 */
function handleCloseDialog() {
  dialog.visible = false;
  resetForm();
}

/** 关闭租户门店审核弹窗并恢复默认表单值。 */
function handleCloseAuditDialog() {
  auditDialog.visible = false;
  resetAuditDialog();
}

/** 提交租户门店审核结果。 */
function handleAuditSubmit() {
  auditFormRef.value?.validate()?.then(valid => {
    if (!valid) return;

    defTenantStoreService.AuditTenantStore(auditFormData).then(() => {
      ElMessage.success("租户门店审核成功");
      handleCloseAuditDialog();
      refreshTable();
    });
  });
}

/** 删除租户门店，兼容单项删除与批量删除。 */
function handleDelete(selected?: number | string | Array<number | string> | TenantStore | TenantStore[]) {
  const storeList = Array.isArray(selected)
    ? (selected.filter(item => typeof item === "object") as TenantStore[])
    : selected && typeof selected === "object"
      ? [selected as TenantStore]
      : [];
  const storeIds = (
    storeList.length ? storeList.map(item => item.id) : normalizeSelectedIds(selected as number | string | Array<number | string>)
  ).join(",");
  if (!storeIds) {
    ElMessage.warning("请勾选删除项");
    return;
  }

  const confirmMessage = storeList.length
    ? storeList.length === 1
      ? `是否确定删除门店：${storeList[0].name}？`
      : `确认删除已选中的 ${storeList.length} 个门店吗？`
    : "确认删除已选中的门店吗？";

  ElMessageBox.confirm(confirmMessage, "警告", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning",
    icon: CloseBold
  }).then(
    () => {
      defTenantStoreService.DeleteTenantStore({ ids: storeIds }).then(() => {
        ElMessage.success("删除门店成功");
        refreshTable();
      });
    },
    () => {
      ElMessage.info("已取消删除门店");
    }
  );
}
</script>

<style scoped lang="scss">
.store-license-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.store-preview-image {
  width: 88px;
  height: 88px;
  border-radius: 4px;
}

.store-preview-image--cover {
  width: 132px;
}

.store-audit-form {
  margin-top: 20px;
}
</style>
