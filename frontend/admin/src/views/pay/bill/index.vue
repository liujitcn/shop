<template>
  <div class="table-box">
    <ProTable ref="proTable" row-key="hash_value" :columns="columns" :request-api="requestPayBillTable" :init-param="initParam">
      <template #file_path="scope">
        <el-link v-if="BUTTONS['pay:bill:download']" type="primary" @click="handleDownload(scope.row)">
          {{ scope.row.file_path }}
        </el-link>
        <span v-else>{{ scope.row.file_path }}</span>
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
import type { PayBill, PagePayBillsRequest } from "@/rpc/admin/v1/pay_bill";
import { buildPageRequest } from "@/utils/proTable";

defineOptions({
  name: "PayBill",
  inheritAttrs: false
});

const { BUTTONS } = useAuthButtons();
const route = useRoute();
const proTable = ref<ProTableInstance>();

const initParam = computed<PagePayBillsRequest>(() => {
  const status = Number(route.query.status ?? 0);
  return {
    bill_date: "",
    status: status > 0 ? status : undefined,
    page_num: 1,
    page_size: 10
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
  { prop: "bill_date", label: "账单日期", minWidth: 120, search: { el: "input" } },
  { prop: "bill_type", label: "账单类型", minWidth: 120 },
  { prop: "file_path", label: "文件路径", minWidth: 300 },
  { prop: "total_count", label: "总笔数", minWidth: 100, align: "right" },
  { prop: "total_amount", label: "总金额（元）", minWidth: 130, align: "right", cellType: "money" },
  { prop: "third_total_count", label: "对账文件总笔数", align: "right", minWidth: 150 },
  { prop: "third_total_amount", label: "对账文件总金额（元）", align: "right", minWidth: 180, cellType: "money" },
  { prop: "status", label: "对账状态", minWidth: 120, dictCode: "pay_bill_status", search: { el: "select" } }
];

/**
 * 请求支付对账单列表，统一交给 ProTable 处理分页。
 */
async function requestPayBillTable(params: PagePayBillsRequest) {
  const data = await defPayBillService.PagePayBills(buildPageRequest(params));
  const compatData = data as typeof data & { payBills?: typeof data.pay_bills; list?: typeof data.pay_bills };
  // ProTable 固定消费 list，优先使用新 snake_case 字段并兼容历史响应。
  const list = compatData.pay_bills ?? compatData.payBills ?? compatData.list ?? [];
  return { data: { ...data, list } };
}

/**
 * 下载当前对账文件。
 */
function handleDownload(row: PayBill) {
  defFileService.DownloadFile(row.file_path, `${row.hash_value}.csv`);
}
</script>
