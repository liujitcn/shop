<template>
  <div class="app-container">
    <div class="search-bar">
      <el-form ref="queryFormRef" :model="queryParams" :inline="true">
        <el-form-item prop="billDate" label="账单日期">
          <el-input
            v-model="queryParams.billDate"
            placeholder="账单日期"
            clearable
            @keyup.enter="handleQuery"
          />
        </el-form-item>

        <el-form-item>
          <el-button type="primary" icon="search" @click="handleQuery">搜索</el-button>
          <el-button icon="refresh" @click="handleResetQuery">重置</el-button>
        </el-form-item>
      </el-form>
    </div>

    <el-card shadow="never">
      <el-table v-loading="loading" :data="pageData" highlight-current-row border>
        <el-table-column label="账单日期" prop="billDate" align="center" />
        <el-table-column label="账单类型" prop="billType" align="center" />
        <el-table-column
          label="文件路径"
          prop="filePath"
          align="center"
          width="300"
          show-overflow-tooltip
        >
          <template #default="scope">
            <el-link
              v-if="useUserStore().hasPerm('pay:bill:download')"
              type="primary"
              @click="handleDownload(scope.row)"
            >
              {{ scope.row.filePath }}
            </el-link>
            <span v-else>{{ scope.row.filePath }}</span>
          </template>
        </el-table-column>
        <el-table-column label="总笔数" prop="totalCount" align="right" />
        <el-table-column label="总金额（元）" align="right">
          <template #default="scope">
            {{ formatPrice(scope.row.totalAmount) }}
          </template>
        </el-table-column>
        <el-table-column label="对账文件总笔数" prop="thirdTotalCount" align="right" />
        <el-table-column label="对账文件总金额（元）" align="right">
          <template #default="scope">
            {{ formatPrice(scope.row.thirdTotalAmount) }}
          </template>
        </el-table-column>
        <el-table-column label="对帐状态" align="center">
          <template #default="scope">
            <DictLabel v-model="scope.row.status" code="pay_bill_status" />
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
  </div>
</template>

<script setup lang="ts">
import { formatPrice } from "@/utils/utils";

defineOptions({
  name: "PayBill",
  inheritAttrs: false,
});

import { defPayBillService } from "@/api/admin/pay_bill";
import { defFileService } from "@/api/base/file";
import { PayBill, PagePayBillRequest } from "@/rpc/admin/pay_bill";
import { useUserStore } from "@/store";
const queryFormRef = ref();

const loading = ref(false);
const total = ref(0);

const queryParams = reactive<PagePayBillRequest>({
  /** 账单日期 */
  billDate: "",
  /** 当前页码 */
  pageNum: 0,
  /** 每一页的行数 */
  pageSize: 10,
});

// 日志表格数据
const pageData = ref<PayBill[]>();

/** 查询 */
function handleQuery() {
  loading.value = true;
  defPayBillService
    .PagePayBill(queryParams)
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

function handleDownload(row: PayBill) {
  defFileService.DownloadFile(row.filePath, `${row.hashValue}.csv`);
}

onMounted(() => {
  handleQuery();
});
</script>
