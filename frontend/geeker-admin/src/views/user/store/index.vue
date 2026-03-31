<template>
  <div class="table-box">
    <ProTable ref="proTable" row-key="id" :columns="columns" :request-api="requestUserStoreTable" />

    <ProDialog v-model="dialog.visible" :title="dialog.title" width="1200px" @close="handleCloseDialog">
      <el-card shadow="never">
        <template #header>
          <div class="card-header">
            <span>门店信息</span>
          </div>
        </template>
        <el-descriptions border :column="2">
          <el-descriptions-item label="门店名称">
            {{ detail.name }}
          </el-descriptions-item>
          <el-descriptions-item label="门店地址">{{ formatAddress(detail.address, detail.detail) }}</el-descriptions-item>
          <el-descriptions-item label="门店照片">
            <div class="demo-image__preview">
              <el-image
                v-for="(img, index) in detail.picture"
                :key="index"
                style="width: 100px; height: 100px"
                :src="img"
                :preview-src-list="detail.picture"
                :zoom-rate="1.2"
                :max-scale="7"
                :min-scale="0.2"
                :initial-index="4"
                fit="cover"
              />
            </div>
          </el-descriptions-item>
          <el-descriptions-item label="营业执照">
            <div class="demo-image__preview">
              <el-image
                v-for="(img, index) in detail.businessLicense"
                :key="index"
                style="width: 100px; height: 100px"
                :src="img"
                :preview-src-list="detail.businessLicense"
                :zoom-rate="1.2"
                :max-scale="7"
                :min-scale="0.2"
                :initial-index="4"
                fit="cover"
              />
            </div>
          </el-descriptions-item>
        </el-descriptions>
      </el-card>

      <el-card shadow="never">
        <template #header>
          <div class="card-header">
            <span>用户信息</span>
          </div>
        </template>
        <el-descriptions border :column="2">
          <el-descriptions-item label="用户名">{{ detail.nickName }}</el-descriptions-item>
          <el-descriptions-item label="手机号">{{ detail.phone }}</el-descriptions-item>
        </el-descriptions>
      </el-card>

      <el-card shadow="never">
        <template #header>
          <div class="card-header">
            <span>审核信息</span>
          </div>
        </template>
        <ProForm ref="proFormRef" :model="formData" :fields="formFields" :rules="rules" label-width="150px" />
      </el-card>
      <template #footer>
        <div class="dialog-footer">
          <el-button type="primary" @click="handleSubmitClick">确 定</el-button>
          <el-button @click="handleCloseDialog">取 消</el-button>
        </div>
      </template>
    </ProDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from "vue";
import { ElMessage } from "element-plus";
import { CircleCheck } from "@element-plus/icons-vue";
import type { ColumnProps, ProTableInstance } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
import ProDialog from "@/components/Dialog/ProDialog.vue";
import ProForm from "@/components/ProForm/index.vue";
import type { ProFormField, ProFormInstance } from "@/components/ProForm/interface";
import { useAuthButtons } from "@/hooks/useAuthButtons";
import { defUserStoreService } from "@/api/admin/user_store";
import type { AuditUserStoreForm, PageUserStoreRequest, UserStore } from "@/rpc/admin/user_store";
import { UserStoreStatus } from "@/rpc/common/enum";
import { buildPageRequest } from "@/utils/proTable";

defineOptions({
  name: "UserStore",
  inheritAttrs: false
});

const { BUTTONS } = useAuthButtons();
const proTable = ref<ProTableInstance>();
const proFormRef = ref<ProFormInstance>();

const dialog = reactive({
  title: "",
  visible: false
});

const detail = ref<UserStore>({
  /** 用户门店ID */
  id: 0,
  /** 门店名称 */
  name: "",
  /** 省市区 */
  address: [],
  /** 详细地址 */
  detail: "",
  /** 门店照片 */
  picture: [],
  /** 营业执照 */
  businessLicense: [],
  /** 状态 */
  status: UserStoreStatus.UNKNOWN_USS,
  /** 备注名 */
  remark: "",
  /** 联系人 */
  nickName: "",
  /** 手机号 */
  phone: ""
});

const formData = reactive<AuditUserStoreForm>({
  /** 用户门店ID */
  id: 0,
  /** 状态 */
  status: UserStoreStatus.UNKNOWN_USS,
  /** 备注名 */
  remark: ""
});

const rules = reactive({
  status: [{ required: true, message: "审核结果不能为空", trigger: "change" }],
  remark: [{ required: true, message: "拒绝原因", trigger: "blur" }]
});

/** 门店审核表单字段配置。 */
const formFields = computed<ProFormField[]>(() => [
  { prop: "status", label: "审核结果", component: "dict", props: { code: "user_store_status", type: "radio" } },
  {
    prop: "remark",
    label: "拒绝原因",
    component: "textarea",
    props: { placeholder: "请输入拒绝原因" },
    visible: model => model.status == 2
  }
]);

/** 用户门店表格列配置。 */
const columns: ColumnProps[] = [
  { prop: "name", label: "门店名称", minWidth: 140, search: { el: "input" } },
  { prop: "nickName", label: "联系人", minWidth: 100 },
  { prop: "phone", label: "电话", minWidth: 130 },
  {
    prop: "address",
    label: "门店地址",
    minWidth: 220,
    render: scope => formatAddress((scope.row as UserStore).address, (scope.row as UserStore).detail)
  },
  { prop: "status", label: "状态", minWidth: 120, dictCode: "user_store_status", search: { el: "select" } },
  { prop: "remark", label: "备注", minWidth: 160 },
  {
    prop: "operation",
    label: "操作",
    width: 100,
    fixed: "right",
    cellType: "actions",
    actions: [
      {
        label: "审核",
        type: "primary",
        link: true,
        icon: CircleCheck,
        hidden: () => !BUTTONS.value["user:store:audit"],
        params: scope => ({ storeId: scope.row.id }),
        onClick: (scope, params) => handleOpenDialog((params?.storeId as number | undefined) ?? (scope.row as UserStore).id)
      }
    ]
  }
];

/**
 * 请求用户门店列表，并由 ProTable 统一维护分页与查询参数。
 */
async function requestUserStoreTable(params: PageUserStoreRequest) {
  const data = await defUserStoreService.PageUserStore(buildPageRequest(params));
  return { data };
}

/**
 * 刷新用户门店表格。
 */
function refreshTable() {
  proTable.value?.getTableList();
}

/**
 * 格式化门店地址，兼容地址数组为空或未返回的情况。
 */
function formatAddress(address?: string[], detailAddress?: string) {
  const addressText = Array.isArray(address) ? address.filter(Boolean).join("-") : "";
  return [addressText, detailAddress].filter(Boolean).join(" ") || "--";
}

/**
 * 打开用户门店详情弹窗。
 */
function handleOpenDialog(storeId?: number) {
  resetDialogData();
  dialog.title = "用户门店详情";
  dialog.visible = true;
  if (!storeId) return;

  defUserStoreService.GetUserStore({ value: storeId }).then(data => {
    detail.value = data;
    formData.id = data.id;
    formData.status = data.status;
    formData.remark = data.remark;
  });
}

/**
 * 关闭用户门店弹窗并清理审核表单。
 */
function handleCloseDialog() {
  dialog.visible = false;
  resetDialogData();
}

/**
 * 重置门店详情与审核表单，避免切换弹窗时残留旧数据。
 */
function resetDialogData() {
  proFormRef.value?.resetFields();
  proFormRef.value?.clearValidate();
  detail.value = {
    id: 0,
    name: "",
    address: [],
    detail: "",
    picture: [],
    businessLicense: [],
    status: UserStoreStatus.UNKNOWN_USS,
    remark: "",
    nickName: "",
    phone: ""
  };
  formData.id = 0;
  formData.status = UserStoreStatus.UNKNOWN_USS;
  formData.remark = "";
}

/**
 * 提交门店审核结果。
 */
function handleSubmitClick() {
  proFormRef.value?.validate()?.then(isValid => {
    if (!isValid) return;

    defUserStoreService.AuditUserStore(formData).then(() => {
      ElMessage.success("门店审核成功");
      handleCloseDialog();
      refreshTable();
    });
  });
}
</script>
