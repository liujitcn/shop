<template>
  <div class="table-box">
    <ProTable ref="proTable" row-key="hashValue" :columns="columns" :request-api="requestPayBillTable" :init-param="initParam">
      <template #filePath="scope">
        <el-link v-if="BUTTONS['pay:bill:download']" type="primary" @click="handleDownload(scope.row)">
          {{ scope.row.filePath }}
        </el-link>
        <span v-else>{{ scope.row.filePath }}</span>
      </template>
    </ProTable>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { useRoute } from "vue-router";
import type { ColumnProps, ProTableInstance } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
import { useAuthButtons } from "@/hooks/useAuthButtons";
import { defPayBillService } from "@/api/admin/pay_bill";
import { defFileService } from "@/api/base/file";
import type { PayBill, PagePayBillRequest } from "@/rpc/admin/pay_bill";
import { buildPageRequest } from "@/utils/proTable";

defineOptions({
  name: "PayBill",
  inheritAttrs: false
});

const { BUTTONS } = useAuthButtons();
const route = useRoute();
const proTable = ref<ProTableInstance>();

const initParam = computed<PagePayBillRequest>(() => {
  const status = Number(route.query.status ?? 0);
  return {
    billDate: "",
    status: status > 0 ? status : undefined,
    pageNum: 1,
    pageSize: 10
  };
});

watch(
  () => [route.query.status, proTable.value],
  () => {
    if (!proTable.value) return;
    const status = Number(route.query.status ?? 0);
    Object.assign(proTable.value.searchParam, {
      status: status > 0 ? status : undefined
    });
    Object.assign(proTable.value.searchInitParam, {
      status: status > 0 ? status : undefined
    });
    proTable.value.pageable.pageNum = 1;
    proTable.value.search();
  },
  { immediate: true }
);

/** 支付对账单表格列配置。 */
const columns: ColumnProps[] = [
  { prop: "billDate", label: "账单日期", minWidth: 120, search: { el: "input" } },
  { prop: "billType", label: "账单类型", minWidth: 120 },
  { prop: "filePath", label: "文件路径", minWidth: 300 },
  { prop: "totalCount", label: "总笔数", minWidth: 100, align: "right" },
  { prop: "totalAmount", label: "总金额（元）", minWidth: 130, align: "right", cellType: "money" },
  { prop: "thirdTotalCount", label: "对账文件总笔数", align: "right", minWidth: 150 },
  { prop: "thirdTotalAmount", label: "对账文件总金额（元）", align: "right", minWidth: 180, cellType: "money" },
  { prop: "status", label: "对账状态", minWidth: 120, dictCode: "pay_bill_status", search: { el: "select" } }
];

/**
 * 请求支付对账单列表，统一交给 ProTable 处理分页。
 */
async function requestPayBillTable(params: PagePayBillRequest) {
  const data = await defPayBillService.PagePayBill(buildPageRequest(params));
  return { data };
}

/**
 * 下载当前对账文件。
 */
function handleDownload(row: PayBill) {
  defFileService.DownloadFile(row.filePath, `${row.hashValue}.csv`);
}
</script>
