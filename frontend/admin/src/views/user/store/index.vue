<template>
  <div class="app-container">
    <div class="search-bar">
      <el-form ref="queryFormRef" :model="queryParams" :inline="true">
        <el-form-item prop="name" label="门店名称">
          <el-input
            v-model="queryParams.name"
            placeholder="门店名称"
            clearable
            @keyup.enter="handleQuery"
          />
        </el-form-item>
        <el-form-item label="状态" prop="status">
          <Dict v-model="queryParams.status" code="user_store_status" />
        </el-form-item>

        <el-form-item>
          <el-button type="primary" icon="search" @click="handleQuery">搜索</el-button>
          <el-button icon="refresh" @click="handleResetQuery">重置</el-button>
        </el-form-item>
      </el-form>
    </div>

    <el-card shadow="never">
      <el-table v-loading="loading" :data="pageData" highlight-current-row border>
        <el-table-column label="门店名称" prop="name" />
        <el-table-column label="联系人" prop="nickName" />
        <el-table-column label="电话" prop="phone" />
        <el-table-column label="门店地址">
          <template #default="scope">
            {{ scope.row.address.join("-") }} {{ scope.row.detail }}
          </template>
        </el-table-column>
        <el-table-column label="状态" align="center">
          <template #default="scope">
            <DictLabel v-model="scope.row.status" code="user_store_status" />
          </template>
        </el-table-column>
        <el-table-column label="备注" prop="remark" />
        <el-table-column fixed="right" label="操作" width="100">
          <template #default="scope">
            <el-button
              v-hasPerm="['user:store:audit']"
              type="primary"
              size="small"
              link
              icon="info"
              @click="handleOpenDialog(scope.row.id)"
            >
              审核
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <pagination
        v-if="total > 0"
        v-model:total="total"
        v-model:page="queryParams.pageNum"
        v-model:limit="queryParams.pageSize"
        @pagination="handleQuery"
      />
    </el-card>

    <el-dialog
      v-model="dialog.visible"
      :title="dialog.title"
      width="1200px"
      @close="handleCloseDialog"
    >
      <!-- 基础信息 -->
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
          <el-descriptions-item label="门店地址">
            {{ detail.address.join("-") }} {{ detail.detail }}
          </el-descriptions-item>
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

      <!-- 用户信息 -->
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

      <el-form ref="dataFormRef" :model="formData" :rules="rules" label-width="150px">
        <el-card shadow="never">
          <template #header>
            <div class="card-header">
              <span>审核信息</span>
            </div>
          </template>
          <el-form-item label="审核结果" prop="status">
            <Dict v-model="formData.status" type="radio" code="user_store_status" />
          </el-form-item>

          <el-form-item v-if="formData.status == 2" label="拒绝原因" prop="remark">
            <el-input v-model="formData.remark" type="textarea" placeholder="请输入拒绝原因" />
          </el-form-item>
        </el-card>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button type="primary" @click="handleSubmitClick">确 定</el-button>
          <el-button @click="handleCloseDialog">取 消</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { UserStoreStatus } from "@/rpc/common/enum";

defineOptions({
  name: "UserStore",
  inheritAttrs: false,
});

import { defUserStoreService } from "@/api/admin/user_store";
import { UserStore, PageUserStoreRequest, AuditUserStoreForm } from "@/rpc/admin/user_store";
const queryFormRef = ref(ElForm);
const dataFormRef = ref(ElForm);

const loading = ref(false);
const total = ref(0);

const queryParams = reactive<PageUserStoreRequest>({
  /** 门店名称 */
  name: "",
  /** 状态码 */
  status: undefined,
  /** 当前页码 */
  pageNum: 0,
  /** 每一页的行数 */
  pageSize: 10,
});

// 表格数据
const pageData = ref<UserStore[]>();

const dialog = reactive({
  title: "",
  visible: false,
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
  phone: "",
});

const formData = reactive<AuditUserStoreForm>({
  /** 用户门店ID */
  id: 0,
  /** 状态 */
  status: UserStoreStatus.UNKNOWN_USS,
  /** 备注名 */
  remark: "",
});

const rules = reactive({
  status: [{ required: true, message: "审核结果不能为空", trigger: "change" }],
  remark: [{ required: true, message: "拒绝原因", trigger: "blur" }],
});

/** 查询 */
function handleQuery() {
  loading.value = true;
  defUserStoreService
    .PageUserStore(queryParams)
    .then((data) => {
      pageData.value = data.list;
      total.value = data.total;
    })
    .finally(() => {
      loading.value = false;
    });
}
/** 重置查询 */
function handleResetQuery() {
  queryFormRef.value.resetFields();
  queryParams.pageNum = 1;
  handleQuery();
}

// 打开用户门店弹窗
function handleOpenDialog(logId?: number) {
  dialog.visible = true;
  if (logId) {
    dialog.title = "用户门店详情";
    defUserStoreService
      .GetUserStore({
        value: logId,
      })
      .then((data) => {
        detail.value = data;
        formData.id = detail.value.id;
        formData.status = detail.value.status;
      });
  }
}

// 关闭用户门店弹窗
function handleCloseDialog() {
  dialog.visible = false;
  dataFormRef.value.resetFields();
  dataFormRef.value.clearValidate();

  formData.id = 0;
}

// 提交字典表单
function handleSubmitClick() {
  dataFormRef.value.validate((isValid: boolean) => {
    if (isValid) {
      loading.value = true;
      defUserStoreService
        .AuditUserStore(formData)
        .then(() => {
          ElMessage.success("审核成功");
          handleCloseDialog();
          handleQuery();
        })
        .finally(() => (loading.value = false));
    }
  });
}

onMounted(() => {
  handleQuery();
});
</script>
