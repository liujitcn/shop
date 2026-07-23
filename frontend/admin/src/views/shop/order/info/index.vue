<template>
  <div class="table-box">
    <el-alert
      v-if="reportSourceLabel"
      class="order-report-alert"
      type="info"
      :closable="false"
      show-icon
      :title="`当前正在查看 ${reportSourceLabel} 的订单明细`"
    />

    <ProTable
      ref="proTable"
      :key="isDefaultTenant ? 'default-tenant' : 'normal-tenant'"
      row-key="id"
      :columns="columns"
      :request-api="requestOrderTable"
      :init-param="reportInitParam"
    />

    <ProDialog v-model="dialogShipped.visible" :title="dialogShipped.title" width="1080px" @close="handleCloseShippedDialog">
      <div v-loading="dialogShipped.loading">
        <el-card class="shipped-hero-card" shadow="never">
          <div class="shipped-hero">
            <div class="dialog-summary">
              <div class="dialog-summary__label">订单发货摘要</div>
              <p class="dialog-summary__desc">先确认收货地址与物流状态，再填写或查看物流信息。</p>
            </div>
            <div class="shipped-metrics">
              <div class="shipped-metric-card">
                <span class="shipped-metric-card__label">联系人</span>
                <strong class="shipped-metric-card__value">{{ dataShipped.address?.receiver || "-" }}</strong>
              </div>
              <div class="shipped-metric-card">
                <span class="shipped-metric-card__label">联系方式</span>
                <strong class="shipped-metric-card__value">{{ dataShipped.address?.contact || "-" }}</strong>
              </div>
              <div class="shipped-metric-card">
                <span class="shipped-metric-card__label">物流状态</span>
                <strong class="shipped-metric-card__value">{{ dataShipped.logistics ? "已发货" : "待填写" }}</strong>
              </div>
              <div class="shipped-metric-card">
                <span class="shipped-metric-card__label">物流单号</span>
                <strong class="shipped-metric-card__value">{{ dataShipped.logistics?.no || formDataShipped.no || "-" }}</strong>
              </div>
            </div>
          </div>
        </el-card>

        <div class="shipped-detail-grid">
          <el-card v-if="hasShippedAddressSection" class="shipped-section-card" shadow="never">
            <template #header>
              <div class="shipped-section-card__header">
                <span>收货地址</span>
              </div>
            </template>

            <el-descriptions :column="1" border class="shipped-descriptions">
              <el-descriptions-item label="联系人">{{ dataShipped.address?.receiver }}</el-descriptions-item>
              <el-descriptions-item label="联系方式">{{ dataShipped.address?.contact }}</el-descriptions-item>
              <el-descriptions-item label="地区">{{ dataShipped.address?.address?.join(" / ") }}</el-descriptions-item>
              <el-descriptions-item label="详细地址">{{ dataShipped.address?.detail }}</el-descriptions-item>
            </el-descriptions>
          </el-card>

          <el-card v-if="hasShippedLogisticsSection" class="shipped-section-card" shadow="never">
            <template #header>
              <div class="shipped-section-card__header">
                <span>物流信息</span>
              </div>
            </template>

            <el-descriptions :column="2" border class="shipped-descriptions">
              <el-descriptions-item label="物流公司">{{ dataShipped.logistics?.name }}</el-descriptions-item>
              <el-descriptions-item label="物流单号">{{ dataShipped.logistics?.no }}</el-descriptions-item>
              <el-descriptions-item label="联系方式">{{ dataShipped.logistics?.contact }}</el-descriptions-item>
              <el-descriptions-item label="发货时间">{{ dataShipped.logistics?.created_at }}</el-descriptions-item>
            </el-descriptions>

            <el-timeline class="shipped-timeline">
              <el-timeline-item
                v-for="(detail, index) in dataShipped.logistics?.detail"
                :key="index"
                :timestamp="detail.time"
                placement="top"
              >
                {{ detail.text }}
              </el-timeline-item>
            </el-timeline>
          </el-card>
        </div>

        <el-card v-if="dataShipped.goods.length" class="shipped-section-card shipped-section-card--full" shadow="never">
          <template #header>
            <div class="shipped-section-card__header">
              <span>商品清单</span>
              <span class="shipped-section-card__extra">{{ dataShipped.goods.length }} 项</span>
            </div>
          </template>

          <div class="goods-list-table">
            <ProTable
              row-key="sku_code"
              :data="dataShipped.goods"
              :columns="shippedGoodsColumns"
              :pagination="false"
              :tool-button="false"
            >
              <template #spec_item="scope">{{ scope.row.spec_item?.join(" ") }}</template>
            </ProTable>
          </div>
        </el-card>

        <el-card v-if="isShippedEditable" class="shipped-section-card shipped-section-card--full" shadow="never">
          <template #header>
            <div class="shipped-section-card__header">
              <span>填写物流信息</span>
            </div>
          </template>

          <ProForm
            ref="dataFormRefShipped"
            :model="formDataShipped"
            :fields="shippedFormFields"
            :rules="rulesShipped"
            label-width="150px"
          />
        </el-card>
      </div>

      <template #footer>
        <div class="dialog-footer">
          <el-button v-if="isShippedEditable" type="primary" :disabled="dialogShipped.loading" @click="handleShippedSubmitClick">
            确 定
          </el-button>
          <el-button @click="handleCloseShippedDialog">取 消</el-button>
        </div>
      </template>
    </ProDialog>

    <ProDialog v-model="dialogRefund.visible" :title="dialogRefund.title" width="1080px" @close="handleCloseRefundDialog">
      <div v-loading="dialogRefund.loading">
        <el-card class="refund-hero-card" shadow="never">
          <div class="refund-hero">
            <div class="dialog-summary">
              <div class="dialog-summary__label">订单退款摘要</div>
              <p class="dialog-summary__desc">先核对支付与对账信息，再决定是否发起退款或查看退款明细。</p>
            </div>
            <div class="refund-metrics">
              <div class="refund-metric-card">
                <span class="refund-metric-card__label">支付金额</span>
                <strong class="refund-metric-card__value">{{ formatPrice(refundOrderPayMoney) }} 元</strong>
              </div>
              <div class="refund-metric-card">
                <span class="refund-metric-card__label">订单总额</span>
                <strong class="refund-metric-card__value">{{ formatPrice(refundOrderTotalMoney) }} 元</strong>
              </div>
              <div class="refund-metric-card">
                <span class="refund-metric-card__label">支付状态</span>
                <strong class="refund-metric-card__value">
                  <DictLabel v-model="refundOrderTradeStatus" code="order_trade_status" />
                </strong>
              </div>
              <div class="refund-metric-card">
                <span class="refund-metric-card__label">对帐状态</span>
                <strong class="refund-metric-card__value">{{ refundReconciliationStatus }}</strong>
              </div>
            </div>
          </div>
        </el-card>

        <div class="refund-detail-grid">
          <el-card v-if="hasRefundPaymentSection" class="refund-section-card" shadow="never">
            <template #header>
              <div class="refund-section-card__header">
                <span>支付信息</span>
              </div>
            </template>
            <el-descriptions :column="2" border class="refund-descriptions">
              <el-descriptions-item label="三方订单号">{{ dataRefund.payment?.third_order_no }}</el-descriptions-item>
              <el-descriptions-item label="交易类型">{{ dataRefund.payment?.trade_type }}</el-descriptions-item>
              <el-descriptions-item label="支付状态">{{ dataRefund.payment?.trade_state }}</el-descriptions-item>
              <el-descriptions-item label="支付状态描述">{{ dataRefund.payment?.trade_state_desc }}</el-descriptions-item>
              <el-descriptions-item label="支付时间">{{ dataRefund.payment?.success_time }}</el-descriptions-item>
              <el-descriptions-item label="支付金额">
                {{ formatPrice(dataRefund.payment?.amount?.payer_total) }} 元
              </el-descriptions-item>
              <el-descriptions-item label="总金额">{{ formatPrice(dataRefund.payment?.amount?.total) }} 元</el-descriptions-item>
              <el-descriptions-item label="对帐状态" :span="2">
                <DictLabel v-model="dataRefund.payment!.status" code="order_bill_status" />
              </el-descriptions-item>
            </el-descriptions>
          </el-card>
        </div>

        <el-card v-if="isRefundEditable" class="refund-section-card refund-section-card--full" shadow="never">
          <template #header>
            <div class="refund-section-card__header">
              <span>填写退款信息</span>
            </div>
          </template>

          <ProForm
            ref="dataFormRefRefund"
            :model="formDataRefund"
            :fields="refundFormFields"
            :rules="rulesRefund"
            label-width="150px"
          />
        </el-card>

        <el-card v-if="dataRefund.refund.length" class="refund-section-card refund-section-card--full" shadow="never">
          <template #header>
            <div class="refund-section-card__header">
              <span>退款信息</span>
              <span class="refund-section-card__extra">{{ dataRefund.refund.length }} 条</span>
            </div>
          </template>
          <ProTable
            row-key="refund_no"
            :data="dataRefund.refund"
            :columns="refundColumns"
            :pagination="false"
            :tool-button="false"
          />
        </el-card>
      </div>

      <template #footer>
        <div class="dialog-footer">
          <el-button v-if="isRefundEditable" type="primary" :disabled="dialogRefund.loading" @click="handleRefundSubmitClick">
            确 定
          </el-button>
          <el-button @click="handleCloseRefundDialog">取 消</el-button>
        </div>
      </template>
    </ProDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, h, reactive, ref, type VNode, watch } from "vue";
import { ElButton, ElLink, ElMessage, ElPopover } from "element-plus";
import { RefreshLeft, Van, View } from "@element-plus/icons-vue";
import { useRoute } from "vue-router";
import type { ColumnProps, ProTableInstance, RenderScope } from "@/components/ProTable/interface";
import ProDialog from "@/components/Dialog/ProDialog.vue";
import ProForm from "@/components/ProForm/index.vue";
import ProTable from "@/components/ProTable/index.vue";
import type { ProFormField, ProFormInstance } from "@/components/ProForm/interface";
import { useAuthButtons } from "@/hooks/useAuthButtons";
import { defOrderInfoService } from "@/api/shop/order_info";
import { defBaseUserService } from "@/api/system/base_user";
import { defTenantStoreService } from "@/api/shop/tenant_store";
import type {
  OrderInfo,
  OrderInfoRefundResponse,
  OrderInfoShipmentForm,
  PageOrderInfoRequest,
  RefundOrderInfoRequest,
  ShipOrderInfoRequest
} from "@/rpc/shop/admin/v1/order_info";
import type { SelectOptionResponse_Option } from "@/rpc/common/v1/common";
import router from "@/routers";
import { OrderInfoStatus, OrderPayType, OrderRefundStatus, OrderTradeStatus } from "@/rpc/shop/common/v1/enum";
import { buildPageRequest } from "@/utils/proTable";
import { navigateTo } from "@/utils/router";
import { formatPrice } from "@/utils/utils";
import { useUserStore } from "@/stores/modules/user";
import {
  buildTenantStoreDisplayMap,
  buildTenantStoreDisplayMapFromOptions,
  DEFAULT_TENANT_CODE,
  formatTenantStoreDisplay,
  parseTenantStoreTreeValue,
  transformTenantStoreTreeOptions,
  type TenantStoreDisplayInfo
} from "@/views/shop/utils/tenant";

defineOptions({
  name: "OrderInfo",
  inheritAttrs: false
});

const props = defineProps({
  /** 订单状态 */
  status: {
    type: Number,
    default: 0
  }
});

const { BUTTONS } = useAuthButtons();
const route = useRoute();
const userStore = useUserStore();
const proTable = ref<ProTableInstance>();
const dataFormRefShipped = ref<ProFormInstance>();
const dataFormRefRefund = ref<ProFormInstance>();
const userOptions = ref<SelectOptionResponse_Option[]>([]);
const tenantStoreDisplayMap = ref(new Map<number, TenantStoreDisplayInfo>());
// 当前退款订单的金额与支付状态来自订单列表，避免货到付款缺少三方支付记录时摘要显示零值。
const refundOrderPayMoney = ref(0);
const refundOrderTotalMoney = ref(0);
const refundOrderTradeStatus = ref(OrderTradeStatus.UNKNOWN_OTS);
const refundOrderPayType = ref(OrderPayType.UNKNOWN_OPT);

/** 订单列表搜索参数，兼容租户门店树筛选展示值。 */
type OrderInfoSearchParams = PageOrderInfoRequest & {
  /** 默认租户的租户门店树筛选值。 */
  tenant_store_tree_value?: string;
};

/** 当前登录账号是否默认租户。 */
const isDefaultTenant = computed(() => userStore.userInfo.tenant_code === DEFAULT_TENANT_CODE);

const reportInitParam = computed(() => {
  const startDate = String(route.query.startDate ?? "");
  const endDate = String(route.query.endDate ?? "");
  const status = Number(route.query.status ?? 0);
  const trade_status = Number(route.query.tradeStatus ?? 0);
  const refund_status = Number(route.query.refundStatus ?? 0);
  const pay_type = Number(route.query.payType ?? 0);
  const pay_channel = Number(route.query.payChannel ?? 0);
  const tenant_id = Number(route.query.tenantId ?? 0);
  const tenant_store_id = Number(route.query.tenantStoreId ?? 0);
  const initParam: Record<string, unknown> = {};
  if (startDate && endDate) {
    initParam.created_at = [startDate, endDate];
  }
  if (status > 0) {
    initParam.status = status;
  }
  if (trade_status > 0) {
    initParam.trade_status = trade_status;
  }
  if (refund_status > 0) {
    initParam.refund_status = refund_status;
  }
  if (pay_type > 0) {
    initParam.pay_type = pay_type;
  }
  if (pay_channel > 0) {
    initParam.pay_channel = pay_channel;
  }
  if (isDefaultTenant.value && tenant_store_id > 0) {
    initParam.tenant_store_tree_value = `store:${tenant_store_id}`;
  } else if (isDefaultTenant.value && tenant_id > 0) {
    initParam.tenant_store_tree_value = `tenant:${tenant_id}`;
  } else if (!isDefaultTenant.value && tenant_store_id > 0) {
    initParam.tenant_store_id = tenant_store_id;
  }
  return initParam;
});

watch(
  () => [route.query, proTable.value],
  () => {
    if (!proTable.value) return;
    const startDate = String(route.query.startDate ?? "");
    const endDate = String(route.query.endDate ?? "");
    const status = Number(route.query.status ?? 0);
    const trade_status = Number(route.query.tradeStatus ?? 0);
    const refund_status = Number(route.query.refundStatus ?? 0);
    const pay_type = Number(route.query.payType ?? 0);
    const pay_channel = Number(route.query.payChannel ?? 0);
    const tenant_id = Number(route.query.tenantId ?? 0);
    const tenant_store_id = Number(route.query.tenantStoreId ?? 0);
    const created_at = startDate && endDate ? [startDate, endDate] : undefined;
    const tenant_store_tree_value =
      tenant_store_id > 0 ? `store:${tenant_store_id}` : tenant_id > 0 ? `tenant:${tenant_id}` : undefined;
    Object.assign(proTable.value.searchParam, {
      status: status > 0 ? status : undefined,
      trade_status: trade_status > 0 ? trade_status : undefined,
      refund_status: refund_status > 0 ? refund_status : undefined,
      pay_type: pay_type > 0 ? pay_type : undefined,
      pay_channel: pay_channel > 0 ? pay_channel : undefined,
      tenant_store_tree_value: isDefaultTenant.value ? tenant_store_tree_value : undefined,
      tenant_store_id: !isDefaultTenant.value && tenant_store_id > 0 ? tenant_store_id : undefined,
      created_at
    });
    Object.assign(proTable.value.searchInitParam, {
      status: status > 0 ? status : undefined,
      trade_status: trade_status > 0 ? trade_status : undefined,
      refund_status: refund_status > 0 ? refund_status : undefined,
      pay_type: pay_type > 0 ? pay_type : undefined,
      pay_channel: pay_channel > 0 ? pay_channel : undefined,
      tenant_store_tree_value: isDefaultTenant.value ? tenant_store_tree_value : undefined,
      tenant_store_id: !isDefaultTenant.value && tenant_store_id > 0 ? tenant_store_id : undefined,
      created_at
    });
    proTable.value.pageable.page_num = 1;
    proTable.value.search();
  },
  { immediate: true }
);

const reportSourceLabel = computed(() => {
  if (route.query.source === "month-report") {
    const periodLabel = String(route.query.periodLabel ?? "");
    return periodLabel ? `${periodLabel} 月报` : "报表";
  }
  if (route.query.source === "day-report") {
    const periodLabel = String(route.query.periodLabel ?? "");
    return periodLabel ? `${periodLabel} 日报` : "报表";
  }
  const periodLabel = String(route.query.periodLabel ?? "");
  return periodLabel ? `${periodLabel} 报表` : "";
});

const dialogShipped = reactive({
  title: "发货详情",
  visible: false,
  loading: false,
  requestId: 0
});

const dataShipped = reactive<OrderInfoShipmentForm>({
  /** 地址信息 */
  address: undefined,
  /** 商品信息 */
  goods: [],
  /** 物流信息 */
  logistics: undefined
});

const formDataShipped = reactive<ShipOrderInfoRequest>({
  /** 订单id */
  order_id: 0,
  /** 物流公司名 */
  name: "",
  /** 物流单号 */
  no: "",
  /** 联系方式 */
  contact: ""
});

const rulesShipped = computed(() => ({
  name: [{ required: true, message: "请输入物流公司名称", trigger: "blur" }],
  no: [{ required: true, message: "请输入物流单号", trigger: "blur" }],
  contact: [{ required: true, message: "请输入联系方式", trigger: "blur" }]
}));

/** 当前发货弹窗是否处于可编辑发货态。 */
const isShippedEditable = computed(() => dialogShipped.title === "发货");

/** 收货地址存在有效内容时才展示地址模块。 */
const hasShippedAddressSection = computed(() => {
  return Boolean(
    dataShipped.address && (dataShipped.address.receiver || dataShipped.address.contact || dataShipped.address.detail)
  );
});

/** 物流信息存在有效内容时才展示物流模块。 */
const hasShippedLogisticsSection = computed(() => {
  return Boolean(
    dataShipped.logistics &&
    (dataShipped.logistics.name || dataShipped.logistics.no || dataShipped.logistics.contact || dataShipped.logistics.created_at)
  );
});

/** 发货表单字段配置。 */
const shippedFormFields = computed<ProFormField[]>(() => [
  {
    prop: "name",
    label: "物流公司名称",
    component: "input",
    props: { placeholder: "请输入物流公司名称" }
  },
  {
    prop: "no",
    label: "物流单号",
    component: "input",
    props: { placeholder: "请输入物流单号" }
  },
  {
    prop: "contact",
    label: "联系方式",
    component: "input",
    props: { placeholder: "请输入联系方式" }
  }
]);

const dialogRefund = reactive({
  title: "退款详情",
  visible: false,
  loading: false,
  requestId: 0
});

const dataRefund = reactive<OrderInfoRefundResponse>({
  /** 支付信息 */
  payment: undefined,
  /** 退款信息 */
  refund: []
});

const formDataRefund = reactive<RefundOrderInfoRequest>({
  /** 订单id */
  order_id: 0,
  /** 退款原因 */
  reason: undefined,
  /** 退款金额 */
  refund_money: 0
});

const rulesRefund = computed(() => ({
  reason: [{ required: true, message: "请输入退款原因", trigger: "blur" }],
  refund_money: [
    { required: true, message: "请输入退款金额", trigger: "blur" },
    {
      validator: (_rule: unknown, value: number, callback: (error?: Error) => void) => {
        if (!value || value <= 0) {
          callback(new Error("退款金额必须大于 0"));
          return;
        }
        if (value > maxRefundMoney.value) {
          callback(new Error(`退款金额不能大于支付金额 ${maxRefundMoney.value.toFixed(2)} 元`));
          return;
        }
        callback();
      },
      trigger: "blur"
    }
  ]
}));

/** 退款表单字段配置。 */
const refundFormFields = computed<ProFormField[]>(() => [
  {
    prop: "reason",
    label: "退款原因",
    component: "dict",
    props: { code: "order_refund_reason" }
  },
  {
    prop: "refund_money",
    label: "退款金额",
    component: "input-number",
    props: {
      min: 0.01,
      max: maxRefundMoney.value,
      precision: 2,
      step: 0.1,
      style: { width: "100%" }
    }
  }
]);

const isRefundEditable = computed(() => dialogRefund.title === "退款");
/** 支付信息存在有效内容时才展示支付模块。 */
const hasRefundPaymentSection = computed(() => {
  return Boolean(
    dataRefund.payment &&
    (dataRefund.payment.third_order_no ||
      dataRefund.payment.trade_type ||
      dataRefund.payment.trade_state_desc ||
      dataRefund.payment.success_time)
  );
});

/** 根据支付方式和三方支付记录显示当前退款订单的对账状态。 */
const refundReconciliationStatus = computed(() => {
  if (hasRefundPaymentSection.value) return "已获取";
  if (refundOrderPayType.value === OrderPayType.CASH_ON_DELIVERY) return "无需对账";
  return "-";
});

const maxRefundMoney = computed(() => {
  const refundedMoney = dataRefund.refund.reduce((total, item) => {
    return item.refund_state === "SUCCESS" ? total + (item.amount?.payer_refund ?? 0) : total;
  }, 0);
  return Number((Math.max(refundOrderPayMoney.value - refundedMoney, 0) / 100).toFixed(2));
});

const shippedGoodsColumns: ColumnProps[] = [
  { prop: "name", label: "商品名称", minWidth: 180 },
  { prop: "sku_code", label: "规格编号", minWidth: 140 },
  { prop: "spec_item", label: "规格名称", minWidth: 160 },
  { prop: "num", label: "数量", align: "right", minWidth: 90 },
  { prop: "price", label: "单价", align: "right", minWidth: 110, cellType: "money" },
  { prop: "pay_price", label: "支付价", align: "right", minWidth: 110, cellType: "money" },
  { prop: "total_pay_price", label: "总金额", align: "right", minWidth: 110, cellType: "money" }
];

const refundColumns: ColumnProps[] = [
  { prop: "third_order_no", label: "三方支付订单编号", align: "center", minWidth: 180 },
  { prop: "refund_no", label: "退款编号", align: "center", minWidth: 160 },
  { prop: "third_refund_no", label: "三方退款编号", align: "center", minWidth: 180 },
  { prop: "reason", label: "退款原因", minWidth: 160 },
  { prop: "channel", label: "退款渠道", align: "center", minWidth: 120 },
  { prop: "user_received_account", label: "退款入账账户", align: "center", minWidth: 160 },
  { prop: "funds_account", label: "资金账户类型", align: "center", minWidth: 140 },
  {
    prop: "payer_refund",
    label: "退款金额",
    minWidth: 110,
    align: "right",
    cellType: "money",
    moneyProps: { value: scope => scope.row.amount?.payer_refund }
  },
  {
    prop: "refundTotal",
    label: "原订单金额",
    minWidth: 120,
    align: "right",
    cellType: "money",
    moneyProps: { value: scope => scope.row.amount?.total }
  },
  { prop: "refund_state", label: "退款状态", align: "center", minWidth: 120 },
  { prop: "success_time", label: "退款时间", align: "center", minWidth: 180 },
  { prop: "status", label: "对帐状态", align: "center", minWidth: 120, dictCode: "order_bill_status" }
];

/**
 * 渲染订单编号列，带权限时可直接跳转到详情页。
 */
function renderOrderNoCell(scope: RenderScope<OrderInfo>) {
  if (!BUTTONS.value["order:info:detail"]) return scope.row.order_no;
  return h(
    ElLink,
    {
      type: "primary",
      onClick: () => handleOpenDetail(scope.row)
    },
    () => scope.row.order_no
  );
}

/**
 * 渲染用户展示名，优先使用当前行昵称，兜底用户字典映射。
 */
function renderUserCell(scope: RenderScope<OrderInfo>) {
  return scope.row.nick_name || formatUser(scope.row.user_id);
}

/**
 * 渲染支付金额列，并展示总价、实付和运费明细。
 */
function renderPayMoneyCell(scope: RenderScope<OrderInfo>) {
  return h(
    ElPopover,
    {
      effect: "light",
      trigger: "hover",
      placement: "top",
      width: "auto"
    },
    {
      default: () => [
        h("div", null, `总价：${formatPrice(scope.row.total_money)}`),
        h("div", null, `实付：${formatPrice(scope.row.pay_money)}`),
        h("div", null, `运费：${formatPrice(scope.row.post_fee)}`)
      ],
      reference: () => formatPrice(scope.row.pay_money)
    }
  );
}

/**
 * 渲染订单操作列。
 * 发货与退款流程依赖订单状态判断，保留在页面侧集中编排。
 */
function renderOperationCell(scope: RenderScope<OrderInfo>) {
  const row = scope.row;
  const actionNodes: VNode[] = [];

  if (canOpenShipped(row) && BUTTONS.value["order:info:shipped"]) {
    actionNodes.push(
      h(
        ElButton,
        { key: `ship-${row.id}`, type: "primary", link: true, icon: Van, onClick: () => handleOpenShippedDialog(row.id, "发货") },
        () => "发货"
      )
    );
  }

  if (canViewShipped(row) && BUTTONS.value["order:info:shipped"]) {
    actionNodes.push(
      h(
        ElButton,
        {
          key: `ship-detail-${row.id}`,
          type: "primary",
          link: true,
          icon: View,
          onClick: () => handleOpenShippedDialog(row.id, "发货详情")
        },
        () => "发货详情"
      )
    );
  }

  if (canOpenRefund(row) && BUTTONS.value["order:info:refund"]) {
    actionNodes.push(
      h(
        ElButton,
        {
          key: `refund-${row.id}`,
          type: "danger",
          link: true,
          icon: RefreshLeft,
          onClick: () => handleOpenRefundDialog(row, "退款")
        },
        () => "退款"
      )
    );
  }

  if (canViewRefund(row) && BUTTONS.value["order:info:refund"]) {
    actionNodes.push(
      h(
        ElButton,
        {
          key: `refund-detail-${row.id}`,
          type: "danger",
          link: true,
          icon: View,
          onClick: () => handleOpenRefundDialog(row, "退款详情")
        },
        () => "退款详情"
      )
    );
  }

  if (!actionNodes.length) return "--";
  return h("div", { class: "order-operation", key: `order-operation-${row.id}` }, actionNodes);
}

/** 订单表格列配置。 */
const columns = computed<ColumnProps[]>(() => [
  {
    prop: "user_id",
    label: "用户",
    minWidth: 180,
    enum: userOptions,
    render: scope => renderUserCell(scope as unknown as RenderScope<OrderInfo>),
    search: {
      el: "select",
      props: {
        filterable: true,
        clearable: true,
        remote: true,
        remoteMethod: handleUserSearch,
        reserveKeyword: false
      },
      order: 2
    }
  },
  {
    prop: "order_no",
    label: "订单编号",
    minWidth: 190,
    search: { el: "input" },
    render: scope => renderOrderNoCell(scope as unknown as RenderScope<OrderInfo>)
  },
  {
    prop: "pay_money",
    label: "金额（元）",
    align: "right",
    minWidth: 120,
    render: scope => renderPayMoneyCell(scope as unknown as RenderScope<OrderInfo>)
  },
  { prop: "pay_type", label: "支付方式", minWidth: 110, dictCode: "order_pay_type", search: { el: "select" } },
  { prop: "pay_channel", label: "支付渠道", minWidth: 110, dictCode: "order_pay_channel", search: { el: "select" } },
  {
    prop: "tenant_store_id",
    label: isDefaultTenant.value ? "租户/门店" : "门店",
    minWidth: isDefaultTenant.value ? 220 : 150,
    showOverflowTooltip: true,
    render: scope => getTenantStoreText(scope.row as OrderInfo),
    search: isDefaultTenant.value
      ? {
          el: "tree-select",
          key: "tenant_store_tree_value",
          order: 1,
          props: {
            clearable: true,
            filterable: true,
            checkStrictly: true,
            renderAfterExpand: false,
            placeholder: "请选择租户/门店",
            style: { width: "100%" }
          }
        }
      : {
          el: "select",
          key: "tenant_store_id",
          order: 1,
          props: {
            clearable: true,
            filterable: true,
            placeholder: "请选择门店",
            style: { width: "100%" }
          }
        },
    enum: isDefaultTenant.value ? requestTenantStoreTreeOptions : requestTenantStoreOptions
  },
  {
    prop: "trade_status",
    label: "支付状态",
    minWidth: 110,
    dictCode: "order_trade_status",
    search: { el: "select" }
  },
  {
    prop: "status",
    label: "履约状态",
    minWidth: 110,
    dictCode: "order_info_status",
    search: props.status ? undefined : { el: "select" }
  },
  {
    prop: "refund_status",
    label: "退款状态",
    minWidth: 120,
    dictCode: "order_refund_status",
    search: { el: "select" }
  },
  {
    prop: "created_at",
    label: "创建时间",
    minWidth: 180,
    search: {
      el: "date-picker",
      props: {
        type: "daterange",
        editable: false,
        class: "!w-[240px]",
        rangeSeparator: "~",
        startPlaceholder: "开始时间",
        endPlaceholder: "截止时间",
        valueFormat: "YYYY-MM-DD"
      }
    }
  },
  { prop: "goods_num", label: "商品数", minWidth: 90, align: "right" },
  {
    prop: "operation",
    label: "操作",
    width: 280,
    fixed: "right",
    render: scope => renderOperationCell(scope as unknown as RenderScope<OrderInfo>)
  }
]);

/**
 * 请求订单分页列表，并补齐固定筛选参数。
 */
async function requestOrderTable(params: PageOrderInfoRequest) {
  const searchParams = params as OrderInfoSearchParams;
  // 默认租户按树节点解析租户或门店，普通租户直接传下拉选择的门店编号。
  const tenantStoreSelection = isDefaultTenant.value
    ? parseTenantStoreTreeValue(searchParams.tenant_store_tree_value)
    : { tenant_store_id: searchParams.tenant_store_id };
  const { tenant_store_tree_value: _tenantStoreTreeValue, tenant_id: _tenantId, tenant_store_id: _tenantStoreId, ...requestParams } = searchParams;
  const data = await defOrderInfoService.PageOrderInfo(
    buildPageRequest({
      ...requestParams,
      tenant_id: tenantStoreSelection.tenant_id,
      tenant_store_id: tenantStoreSelection.tenant_store_id,
      user_id: Number(params.user_id ?? 0),
      status: props.status || params.status,
      created_at: params.created_at ?? ["", ""]
    })
  );
  return { data: { list: data.order_infos ?? [], total: data.total } };
}

/** 刷新订单表格。 */
function refreshTable() {
  proTable.value?.getTableList();
}

/**
 * 按关键字远程加载用户下拉项；空关键字直接清空，避免查询全量用户。
 */
async function loadUserOptionsByKeyword(keyword: string) {
  const trimmedKeyword = keyword.trim();
  if (!trimmedKeyword) {
    userOptions.value.splice(0, userOptions.value.length);
    return userOptions.value;
  }

  const response = await defBaseUserService.OptionBaseUser({ keyword: trimmedKeyword });
  userOptions.value.splice(0, userOptions.value.length, ...(response.list ?? []));
  return userOptions.value;
}

/**
 * 请求租户门店树筛选数据。
 */
async function requestTenantStoreTreeOptions() {
  const response = await defTenantStoreService.TreeTenantStore({ keyword: "" });
  tenantStoreDisplayMap.value = buildTenantStoreDisplayMap(response.list ?? []);
  return { data: transformTenantStoreTreeOptions(response.list ?? []) };
}

/** 请求普通租户的门店下拉筛选数据。 */
async function requestTenantStoreOptions() {
  const response = await defTenantStoreService.OptionTenantStore({ keyword: "" });
  tenantStoreDisplayMap.value = buildTenantStoreDisplayMapFromOptions(response.list ?? []);
  return { data: response.list ?? [] };
}

/**
 * 读取订单列表租户门店展示文本，默认租户显示租户/门店。
 */
function getTenantStoreText(row: OrderInfo) {
  return formatTenantStoreDisplay(row.tenant_store_id, tenantStoreDisplayMap.value);
}

/**
 * 处理用户远程搜索。
 */
function handleUserSearch(keyword: string) {
  loadUserOptionsByKeyword(keyword);
}

/**
 * 判断当前订单是否可发货。
 */
function canOpenShipped(row: OrderInfo) {
  return (
    row.status === OrderInfoStatus.WAIT_SHIPMENT_OIS &&
    [OrderTradeStatus.PAID_OTS, OrderTradeStatus.CASH_ON_DELIVERY_OTS, OrderTradeStatus.PARTIAL_REFUND_OTS].includes(
      row.trade_status
    ) &&
    [OrderRefundStatus.NONE_ORS, OrderRefundStatus.PARTIAL_REFUND_ORS, OrderRefundStatus.CLOSED_OR_FAILED_ORS].includes(
      row.refund_status
    )
  );
}

/**
 * 判断当前订单是否可查看发货详情。
 */
function canViewShipped(row: OrderInfo) {
  return [OrderInfoStatus.SHIPPED_OIS, OrderInfoStatus.WAIT_REVIEW_OIS, OrderInfoStatus.COMPLETED_OIS].includes(row.status);
}

/**
 * 判断在线支付订单是否可发起退款。
 */
function canOpenRefund(row: OrderInfo) {
  return (
    [
      OrderInfoStatus.WAIT_SHIPMENT_OIS,
      OrderInfoStatus.SHIPPED_OIS,
      OrderInfoStatus.WAIT_REVIEW_OIS,
      OrderInfoStatus.COMPLETED_OIS
    ].includes(row.status) &&
    [OrderTradeStatus.PAID_OTS, OrderTradeStatus.CASH_ON_DELIVERY_OTS, OrderTradeStatus.PARTIAL_REFUND_OTS].includes(
      row.trade_status
    ) &&
    [OrderRefundStatus.NONE_ORS, OrderRefundStatus.PARTIAL_REFUND_ORS, OrderRefundStatus.CLOSED_OR_FAILED_ORS].includes(
      row.refund_status
    )
  );
}

/**
 * 判断当前订单是否可查看退款详情。
 */
function canViewRefund(row: OrderInfo) {
  return row.refund_status !== OrderRefundStatus.NONE_ORS && row.refund_status !== OrderRefundStatus.UNKNOWN_ORS;
}

/**
 * 打开发货弹窗并加载物流数据。
 */
function handleOpenShippedDialog(order_id: number, title: string) {
  resetShippedDialog();
  dialogShipped.visible = true;
  dialogShipped.title = title;
  dialogShipped.loading = true;
  formDataShipped.order_id = order_id;
  const requestId = ++dialogShipped.requestId;
  defOrderInfoService
    .GetOrderInfoShipment({ id: order_id })
    .then(data => {
      if (requestId !== dialogShipped.requestId || !dialogShipped.visible) return;
      Object.assign(dataShipped, data);
    })
    .catch(() => {
      if (requestId !== dialogShipped.requestId) return;
      ElMessage.error(`${title}加载失败`);
      dialogShipped.visible = false;
    })
    .finally(() => {
      if (requestId !== dialogShipped.requestId) return;
      dialogShipped.loading = false;
    });
}

/**
 * 关闭发货弹窗并清理表单。
 */
function handleCloseShippedDialog() {
  dialogShipped.requestId += 1;
  dialogShipped.loading = false;
  dialogShipped.visible = false;
  resetShippedDialog();
}

/**
 * 重置发货弹窗数据，避免不同订单之间串值。
 */
function resetShippedDialog() {
  dataFormRefShipped.value?.resetFields();
  dataFormRefShipped.value?.clearValidate();
  formDataShipped.order_id = 0;
  formDataShipped.name = "";
  formDataShipped.no = "";
  formDataShipped.contact = "";
  dataShipped.logistics = undefined;
  dataShipped.goods = [];
  dataShipped.address = undefined;
}

/**
 * 提交订单发货信息。
 */
function handleShippedSubmitClick() {
  dataFormRefShipped.value
    ?.validate?.()
    ?.then(isValid => {
      if (!isValid) return;

      defOrderInfoService.ShipOrderInfo(formDataShipped).then(() => {
        ElMessage.success("订单发货成功");
        handleCloseShippedDialog();
        refreshTable();
      });
    })
    .catch(() => undefined);
}

/**
 * 打开退款弹窗并加载退款详情。
 */
function handleOpenRefundDialog(row: OrderInfo, title: string) {
  resetRefundDialog();
  dialogRefund.visible = true;
  dialogRefund.title = title;
  dialogRefund.loading = true;
  formDataRefund.order_id = row.id;
  refundOrderPayMoney.value = row.pay_money;
  refundOrderTotalMoney.value = row.total_money;
  refundOrderTradeStatus.value = row.trade_status;
  refundOrderPayType.value = row.pay_type;
  const requestId = ++dialogRefund.requestId;
  defOrderInfoService
    .GetOrderInfoRefund({ id: row.id })
    .then(data => {
      if (requestId !== dialogRefund.requestId || !dialogRefund.visible) return;
      Object.assign(dataRefund, data);
      // 退款弹窗默认填入当前订单剩余可退金额，避免控件最小值与订单金额不一致。
      formDataRefund.refund_money = maxRefundMoney.value;
    })
    .catch(() => {
      if (requestId !== dialogRefund.requestId) return;
      ElMessage.error(`${title}加载失败`);
      dialogRefund.visible = false;
    })
    .finally(() => {
      if (requestId !== dialogRefund.requestId) return;
      dialogRefund.loading = false;
    });
}

/**
 * 关闭退款弹窗并清理表单。
 */
function handleCloseRefundDialog() {
  dialogRefund.requestId += 1;
  dialogRefund.loading = false;
  dialogRefund.visible = false;
  resetRefundDialog();
}

/**
 * 重置退款弹窗数据，避免切换订单时残留上一条记录。
 */
function resetRefundDialog() {
  dataFormRefRefund.value?.resetFields();
  dataFormRefRefund.value?.clearValidate();
  formDataRefund.order_id = 0;
  formDataRefund.reason = undefined;
  formDataRefund.refund_money = 0;
  refundOrderPayMoney.value = 0;
  refundOrderTotalMoney.value = 0;
  refundOrderTradeStatus.value = OrderTradeStatus.UNKNOWN_OTS;
  refundOrderPayType.value = OrderPayType.UNKNOWN_OPT;
  dataRefund.payment = undefined;
  dataRefund.refund = [];
}

/**
 * 提交退款申请。
 */
function handleRefundSubmitClick() {
  dataFormRefRefund.value
    ?.validate?.()
    ?.then(isValid => {
      if (!isValid) return;

      const submitData = JSON.parse(JSON.stringify(formDataRefund)) as RefundOrderInfoRequest;
      submitData.refund_money = Math.round(submitData.refund_money * 100);
      defOrderInfoService.RefundOrderInfo(submitData).then(() => {
        ElMessage.success("退款申请已提交");
        handleCloseRefundDialog();
        refreshTable();
      });
    })
    .catch(() => undefined);
}

/**
 * 根据用户 ID 回显昵称，兜底旧数据场景。
 */
function formatUser(user_id: number) {
  const entry = userOptions.value.find(item => Number(item.value) === user_id);
  return entry ? entry.label : "-";
}

/**
 * 打开订单详情页。
 */
function handleOpenDetail(row: OrderInfo) {
  // 订单详情页标题固定为“订单详情”，跳转时不再额外携带标题查询参数。
  navigateTo(router, `/order/detail/${row.id}`);
}
</script>

<style scoped lang="scss">
.shipped-hero-card,
.shipped-section-card {
  background: var(--admin-page-card-bg);
  border: 1px solid var(--admin-page-card-border);
  border-radius: var(--admin-page-radius);
  box-shadow: var(--admin-page-shadow);
}
.shipped-hero-card {
  margin-bottom: 18px;
}
:deep(.shipped-hero-card .el-card__body),
:deep(.shipped-section-card .el-card__body),
:deep(.refund-hero-card .el-card__body),
:deep(.refund-section-card .el-card__body) {
  padding: 16px;
}
.shipped-hero {
  display: block;
}
.dialog-summary {
  margin-bottom: 14px;
}
.dialog-summary__label {
  font-size: 12px;
  font-weight: 600;
  color: var(--admin-page-text-secondary);
}
.dialog-summary__desc {
  margin: 6px 0 0;
  font-size: 14px;
  color: var(--admin-page-text-primary);
}
.shipped-metrics {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}
.shipped-metric-card {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 14px;
  background: var(--admin-page-card-bg-soft);
  border: 1px solid var(--admin-page-card-border-soft);
  border-radius: var(--admin-page-radius);
}
.shipped-metric-card__label {
  font-size: 13px;
  color: var(--admin-page-text-secondary);
}
.shipped-metric-card__value {
  font-size: 18px;
  font-weight: 700;
  color: var(--admin-page-text-primary);
  word-break: break-all;
}
.shipped-detail-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
  margin-bottom: 16px;
}
.shipped-section-card {
  overflow: hidden;
}
.shipped-section-card--full {
  margin-bottom: 16px;
}
.shipped-section-card__header {
  display: flex;
  gap: 12px;
  align-items: center;
  justify-content: space-between;
  font-size: 16px;
  font-weight: 600;
  color: var(--admin-page-text-primary);
}
.shipped-section-card__extra {
  font-size: 13px;
  font-weight: 500;
  color: var(--admin-page-text-placeholder);
}
.shipped-descriptions :deep(.el-descriptions__label) {
  width: 110px;
  font-weight: 600;
}
.shipped-descriptions :deep(.el-descriptions__cell),
.refund-descriptions :deep(.el-descriptions__cell) {
  padding: 10px 14px;
}
.shipped-timeline {
  padding: 18px 18px 0;
  margin-top: 20px;
  background: var(--admin-page-card-bg-soft);
  border-radius: var(--admin-page-radius);
}
.refund-hero-card,
.refund-section-card {
  background: var(--admin-page-card-bg);
  border: 1px solid var(--admin-page-card-border);
  border-radius: var(--admin-page-radius);
  box-shadow: var(--admin-page-shadow);
}
.refund-hero-card {
  margin-bottom: 18px;
}
.refund-hero {
  display: block;
}
.refund-metrics {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}
.refund-metric-card {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 14px;
  background: var(--admin-page-card-bg-soft);
  border: 1px solid var(--admin-page-card-border-soft);
  border-radius: var(--admin-page-radius);
}
.refund-metric-card__label {
  font-size: 13px;
  color: var(--admin-page-text-secondary);
}
.refund-metric-card__value {
  font-size: 18px;
  font-weight: 700;
  color: var(--admin-page-text-primary);
  word-break: break-all;
}
.refund-detail-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
  margin-bottom: 16px;
}
.refund-section-card {
  overflow: hidden;
}
.refund-section-card--full {
  margin-bottom: 16px;
}
.refund-section-card__header {
  display: flex;
  gap: 12px;
  align-items: center;
  justify-content: space-between;
  font-size: 16px;
  font-weight: 600;
  color: var(--admin-page-text-primary);
}
.refund-section-card__extra {
  font-size: 13px;
  font-weight: 500;
  color: var(--admin-page-text-placeholder);
}
.refund-descriptions :deep(.el-descriptions__label) {
  width: 110px;
  font-weight: 600;
}
.order-report-alert {
  margin-bottom: 16px;
}

/* 发货相关弹窗中的商品清单按实际行数撑开，避免继承通用表格的固定最小高度。 */
.goods-list-table {
  :deep(.table-main) {
    height: auto;
    min-height: auto;
  }
  :deep(.el-table) {
    flex: initial;
  }
  :deep(.el-table__inner-wrapper),
  :deep(.el-table__body-wrapper),
  :deep(.el-scrollbar),
  :deep(.el-scrollbar__wrap),
  :deep(.el-scrollbar__view) {
    height: auto;
  }
}

@media (width <= 992px) {
  .shipped-detail-grid,
  .refund-detail-grid {
    grid-template-columns: 1fr;
  }
}

@media (width <= 768px) {
  .shipped-metrics,
  .refund-metrics {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (width <= 520px) {
  .shipped-metrics,
  .refund-metrics {
    grid-template-columns: 1fr;
  }
}
</style>
