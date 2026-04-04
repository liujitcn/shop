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

    <ProTable ref="proTable" row-key="id" :columns="columns" :request-api="requestOrderTable" :init-param="reportInitParam" />

    <ProDialog v-model="dialogShipped.visible" :title="dialogShipped.title" width="1080px" @close="handleCloseShippedDialog">
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
            <el-descriptions-item label="发货时间">{{ dataShipped.logistics?.createdAt }}</el-descriptions-item>
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
            row-key="skuCode"
            :data="dataShipped.goods"
            :columns="shippedGoodsColumns"
            :pagination="false"
            :tool-button="false"
          >
            <template #specItem="scope">{{ scope.row.specItem?.join(" ") }}</template>
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

      <template #footer>
        <div class="dialog-footer">
          <el-button v-if="isShippedEditable" type="primary" @click="handleShippedSubmitClick">确 定</el-button>
          <el-button @click="handleCloseShippedDialog">取 消</el-button>
        </div>
      </template>
    </ProDialog>

    <ProDialog v-model="dialogRefund.visible" :title="dialogRefund.title" width="1080px" @close="handleCloseRefundDialog">
      <el-card class="refund-hero-card" shadow="never">
        <div class="refund-hero">
          <div class="dialog-summary">
            <div class="dialog-summary__label">订单退款摘要</div>
            <p class="dialog-summary__desc">先核对支付与对账信息，再决定是否发起退款或查看退款明细。</p>
          </div>
          <div class="refund-metrics">
            <div class="refund-metric-card">
              <span class="refund-metric-card__label">支付金额</span>
              <strong class="refund-metric-card__value">{{ formatPrice(dataRefund.payment?.amount?.payerTotal) }} 元</strong>
            </div>
            <div class="refund-metric-card">
              <span class="refund-metric-card__label">订单总额</span>
              <strong class="refund-metric-card__value">{{ formatPrice(dataRefund.payment?.amount?.total) }} 元</strong>
            </div>
            <div class="refund-metric-card">
              <span class="refund-metric-card__label">支付状态</span>
              <strong class="refund-metric-card__value">{{
                dataRefund.payment?.tradeStateDesc || dataRefund.payment?.tradeState || "-"
              }}</strong>
            </div>
            <div class="refund-metric-card">
              <span class="refund-metric-card__label">对帐状态</span>
              <strong class="refund-metric-card__value">{{ dataRefund.payment ? "已获取" : "-" }}</strong>
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
            <el-descriptions-item label="三方订单号">{{ dataRefund.payment?.thirdOrderNo }}</el-descriptions-item>
            <el-descriptions-item label="交易类型">{{ dataRefund.payment?.tradeType }}</el-descriptions-item>
            <el-descriptions-item label="支付状态">{{ dataRefund.payment?.tradeState }}</el-descriptions-item>
            <el-descriptions-item label="支付状态描述">{{ dataRefund.payment?.tradeStateDesc }}</el-descriptions-item>
            <el-descriptions-item label="支付时间">{{ dataRefund.payment?.successTime }}</el-descriptions-item>
            <el-descriptions-item label="支付金额">
              {{ formatPrice(dataRefund.payment?.amount?.payerTotal) }} 元
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
          row-key="refundNo"
          :data="dataRefund.refund"
          :columns="refundColumns"
          :pagination="false"
          :tool-button="false"
        />
      </el-card>

      <template #footer>
        <div class="dialog-footer">
          <el-button v-if="isRefundEditable" type="primary" @click="handleRefundSubmitClick">确 定</el-button>
          <el-button @click="handleCloseRefundDialog">取 消</el-button>
        </div>
      </template>
    </ProDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, h, reactive, ref, resolveComponent, type VNode, watch } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { RefreshLeft, Van, View } from "@element-plus/icons-vue";
import { useRoute } from "vue-router";
import type { ColumnProps, ProTableInstance, RenderScope } from "@/components/ProTable/interface";
import ProDialog from "@/components/Dialog/ProDialog.vue";
import ProForm from "@/components/ProForm/index.vue";
import ProTable from "@/components/ProTable/index.vue";
import type { ProFormField, ProFormInstance } from "@/components/ProForm/interface";
import { useAuthButtons } from "@/hooks/useAuthButtons";
import { defOrderInfoService } from "@/api/admin/order_info";
import { defBaseUserService } from "@/api/admin/base_user";
import type {
  OrderInfo,
  OrderInfoRefundResponse,
  OrderInfoShippedResponse,
  PageOrderInfoRequest,
  RefundOrderInfoRequest,
  ShippedOrderInfoRequest
} from "@/rpc/admin/order_info";
import type { SelectOptionResponse_Option } from "@/rpc/common/common";
import router from "@/routers";
import { OrderPayType, OrderStatus } from "@/rpc/common/enum";
import { buildPageRequest } from "@/utils/proTable";
import { navigateTo } from "@/utils/router";
import { formatPrice } from "@/utils/utils";

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
const proTable = ref<ProTableInstance>();
const dataFormRefShipped = ref<ProFormInstance>();
const dataFormRefRefund = ref<ProFormInstance>();
const userOptions = ref<SelectOptionResponse_Option[]>([]);

const reportInitParam = computed(() => {
  const startDate = String(route.query.startDate ?? "");
  const endDate = String(route.query.endDate ?? "");
  const status = Number(route.query.status ?? 0);
  const payType = Number(route.query.payType ?? 0);
  const payChannel = Number(route.query.payChannel ?? 0);
  const initParam: Record<string, unknown> = {};
  if (startDate && endDate) {
    initParam.createdAt = [startDate, endDate];
  }
  if (status > 0) {
    initParam.status = status;
  }
  if (payType > 0) {
    initParam.payType = payType;
  }
  if (payChannel > 0) {
    initParam.payChannel = payChannel;
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
    const payType = Number(route.query.payType ?? 0);
    const payChannel = Number(route.query.payChannel ?? 0);
    const createdAt = startDate && endDate ? [startDate, endDate] : undefined;
    Object.assign(proTable.value.searchParam, {
      status: status > 0 ? status : undefined,
      payType: payType > 0 ? payType : undefined,
      payChannel: payChannel > 0 ? payChannel : undefined,
      createdAt
    });
    Object.assign(proTable.value.searchInitParam, {
      status: status > 0 ? status : undefined,
      payType: payType > 0 ? payType : undefined,
      payChannel: payChannel > 0 ? payChannel : undefined,
      createdAt
    });
    proTable.value.pageable.pageNum = 1;
    proTable.value.search();
  },
  { immediate: true }
);

const reportSourceLabel = computed(() => {
  if (route.query.source !== "month-report") return "";
  const periodLabel = String(route.query.periodLabel ?? "");
  return periodLabel ? `${periodLabel} 月报` : "报表";
});

const dialogShipped = reactive({
  title: "发货详情",
  visible: false
});

const dataShipped = reactive<OrderInfoShippedResponse>({
  /** 地址信息 */
  address: undefined,
  /** 商品信息 */
  goods: [],
  /** 物流信息 */
  logistics: undefined
});

const formDataShipped = reactive<ShippedOrderInfoRequest>({
  /** 订单id */
  orderId: 0,
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
    (dataShipped.logistics.name || dataShipped.logistics.no || dataShipped.logistics.contact || dataShipped.logistics.createdAt)
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
  visible: false
});

const dataRefund = reactive<OrderInfoRefundResponse>({
  /** 支付信息 */
  payment: undefined,
  /** 退款信息 */
  refund: []
});

const formDataRefund = reactive<RefundOrderInfoRequest>({
  /** 订单id */
  orderId: 0,
  /** 退款原因 */
  reason: undefined,
  /** 退款金额 */
  refundMoney: 0
});

const rulesRefund = computed(() => ({
  reason: [{ required: true, message: "请输入退款原因", trigger: "blur" }],
  refundMoney: [
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
    prop: "refundMoney",
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
    (dataRefund.payment.thirdOrderNo ||
      dataRefund.payment.tradeType ||
      dataRefund.payment.tradeStateDesc ||
      dataRefund.payment.successTime)
  );
});

const maxRefundMoney = computed(() => Number(((dataRefund.payment?.amount?.payerTotal ?? 0) / 100).toFixed(2)));

const shippedGoodsColumns: ColumnProps[] = [
  { prop: "name", label: "商品名称", minWidth: 180 },
  { prop: "skuCode", label: "规格编号", minWidth: 140 },
  { prop: "specItem", label: "规格名称", minWidth: 160 },
  { prop: "num", label: "数量", align: "right", minWidth: 90 },
  { prop: "price", label: "单价", align: "right", minWidth: 110, cellType: "money" },
  { prop: "payPrice", label: "支付价", align: "right", minWidth: 110, cellType: "money" },
  { prop: "totalPayPrice", label: "总金额", align: "right", minWidth: 110, cellType: "money" }
];

const refundColumns: ColumnProps[] = [
  { prop: "thirdOrderNo", label: "三方支付订单编号", align: "center", minWidth: 180 },
  { prop: "refundNo", label: "退款编号", align: "center", minWidth: 160 },
  { prop: "thirdRefundNo", label: "三方退款编号", align: "center", minWidth: 180 },
  { prop: "reason", label: "退款原因", minWidth: 160 },
  { prop: "channel", label: "退款渠道", align: "center", minWidth: 120 },
  { prop: "userReceivedAccount", label: "退款入账账户", align: "center", minWidth: 160 },
  { prop: "fundsAccount", label: "资金账户类型", align: "center", minWidth: 140 },
  {
    prop: "payerRefund",
    label: "退款金额",
    minWidth: 110,
    align: "right",
    cellType: "money",
    moneyProps: { value: scope => scope.row.amount?.payerRefund }
  },
  {
    prop: "refundTotal",
    label: "原订单金额",
    minWidth: 120,
    align: "right",
    cellType: "money",
    moneyProps: { value: scope => scope.row.amount?.total }
  },
  { prop: "refundState", label: "退款状态", align: "center", minWidth: 120 },
  { prop: "successTime", label: "退款时间", align: "center", minWidth: 180 },
  { prop: "status", label: "对帐状态", align: "center", minWidth: 120, dictCode: "order_bill_status" }
];

/**
 * 渲染订单编号列，带权限时可直接跳转到详情页。
 */
function renderOrderNoCell(scope: RenderScope<OrderInfo>) {
  if (!BUTTONS.value["order:info:detail"]) return scope.row.orderNo;
  return h(
    resolveComponent("el-link"),
    {
      type: "primary",
      onClick: () => handleOpenDetail(scope.row)
    },
    () => scope.row.orderNo
  );
}

/**
 * 渲染用户展示名，优先使用当前行昵称，兜底用户字典映射。
 */
function renderUserCell(scope: RenderScope<OrderInfo>) {
  return scope.row.nickName || formatUser(scope.row.userId);
}

/**
 * 渲染支付金额列，并展示总价、实付和运费明细。
 */
function renderPayMoneyCell(scope: RenderScope<OrderInfo>) {
  return h(
    resolveComponent("el-popover"),
    {
      effect: "light",
      trigger: "hover",
      placement: "top",
      width: "auto"
    },
    {
      default: () => [
        h("div", null, `总价：${formatPrice(scope.row.totalMoney)}`),
        h("div", null, `实付：${formatPrice(scope.row.payMoney)}`),
        h("div", null, `运费：${formatPrice(scope.row.postFee)}`)
      ],
      reference: () => formatPrice(scope.row.payMoney)
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
        resolveComponent("el-button"),
        { key: `ship-${row.id}`, type: "primary", link: true, icon: Van, onClick: () => handleOpenShippedDialog(row.id, "发货") },
        () => "发货"
      )
    );
  }

  if (canViewShipped(row) && BUTTONS.value["order:info:shipped"]) {
    actionNodes.push(
      h(
        resolveComponent("el-button"),
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

  if (canRefundCod(row) && BUTTONS.value["order:info:refund"]) {
    actionNodes.push(
      h(
        resolveComponent("el-button"),
        { key: `refund-cod-${row.id}`, type: "danger", link: true, icon: RefreshLeft, onClick: () => handleRefund(row) },
        () => "退款"
      )
    );
  }

  if (canOpenRefund(row) && BUTTONS.value["order:info:refund"]) {
    actionNodes.push(
      h(
        resolveComponent("el-button"),
        {
          key: `refund-${row.id}`,
          type: "danger",
          link: true,
          icon: RefreshLeft,
          onClick: () => handleOpenRefundDialog(row.id, "退款")
        },
        () => "退款"
      )
    );
  }

  if (canViewRefund(row) && BUTTONS.value["order:info:refund"]) {
    actionNodes.push(
      h(
        resolveComponent("el-button"),
        {
          key: `refund-detail-${row.id}`,
          type: "danger",
          link: true,
          icon: View,
          onClick: () => handleOpenRefundDialog(row.id, "退款详情")
        },
        () => "退款详情"
      )
    );
  }

  if (!actionNodes.length) return "--";
  return h("div", { class: "order-operation", key: `order-operation-${row.id}` }, actionNodes);
}

/** 订单表格列配置。 */
const columns: ColumnProps[] = [
  {
    prop: "userId",
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
      }
    }
  },
  {
    prop: "orderNo",
    label: "订单编号",
    minWidth: 190,
    search: { el: "input" },
    render: scope => renderOrderNoCell(scope as unknown as RenderScope<OrderInfo>)
  },
  {
    prop: "payMoney",
    label: "金额（元）",
    align: "right",
    minWidth: 120,
    render: scope => renderPayMoneyCell(scope as unknown as RenderScope<OrderInfo>)
  },
  { prop: "payType", label: "支付方式", minWidth: 110, dictCode: "order_pay_type", search: { el: "select" } },
  { prop: "payChannel", label: "支付渠道", minWidth: 110, dictCode: "order_pay_channel", search: { el: "select" } },
  {
    prop: "status",
    label: "状态",
    minWidth: 110,
    dictCode: "order_status",
    search: props.status ? undefined : { el: "select" }
  },
  {
    prop: "createdAt",
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
  { prop: "goodsNum", label: "商品数", minWidth: 90, align: "right" },
  {
    prop: "operation",
    label: "操作",
    width: 280,
    fixed: "right",
    render: scope => renderOperationCell(scope as unknown as RenderScope<OrderInfo>)
  }
];

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
 * 处理用户远程搜索。
 */
function handleUserSearch(keyword: string) {
  loadUserOptionsByKeyword(keyword);
}

/**
 * 请求订单分页列表，并补齐固定筛选参数。
 */
async function requestOrderTable(params: PageOrderInfoRequest) {
  const data = await defOrderInfoService.PageOrderInfo(
    buildPageRequest({
      ...params,
      userId: Number(params.userId ?? 0),
      status: props.status || params.status,
      createdAt: params.createdAt ?? ["", ""]
    })
  );
  return { data };
}

/**
 * 判断当前订单是否可发货。
 */
function canOpenShipped(row: OrderInfo) {
  return row.status === OrderStatus.PAID;
}

/**
 * 判断当前订单是否可查看发货详情。
 */
function canViewShipped(row: OrderInfo) {
  return row.status === OrderStatus.SHIPPED || row.status === OrderStatus.RECEIVED;
}

/**
 * 判断货到付款订单是否可直接退款。
 */
function canRefundCod(row: OrderInfo) {
  return row.payType === OrderPayType.CASH_ON_DELIVERY && canViewShipped(row);
}

/**
 * 判断在线支付订单是否可发起退款。
 */
function canOpenRefund(row: OrderInfo) {
  return (
    row.payType === OrderPayType.ONLINE_PAY &&
    (row.status === OrderStatus.SHIPPED || row.status === OrderStatus.RECEIVED || row.status === OrderStatus.REFUNDING)
  );
}

/**
 * 判断当前订单是否可查看退款详情。
 */
function canViewRefund(row: OrderInfo) {
  return row.status === OrderStatus.REFUNDING;
}

/**
 * 打开发货弹窗并加载物流数据。
 */
function handleOpenShippedDialog(orderId: number, title: string) {
  resetShippedDialog();
  dialogShipped.visible = true;
  dialogShipped.title = title;
  defOrderInfoService.GetOrderInfoShipped({ value: orderId }).then(data => {
    formDataShipped.orderId = orderId;
    Object.assign(dataShipped, data);
  });
}

/**
 * 关闭发货弹窗并清理表单。
 */
function handleCloseShippedDialog() {
  dialogShipped.visible = false;
  resetShippedDialog();
}

/**
 * 重置发货弹窗数据，避免不同订单之间串值。
 */
function resetShippedDialog() {
  dataFormRefShipped.value?.resetFields();
  dataFormRefShipped.value?.clearValidate();
  formDataShipped.orderId = 0;
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

      defOrderInfoService.ShippedOrderInfo(formDataShipped).then(() => {
        ElMessage.success("订单发货成功");
        handleCloseShippedDialog();
        proTable.value?.getTableList();
      });
    })
    .catch(() => undefined);
}

/**
 * 对货到付款订单发起退款。
 */
function handleRefund(row: OrderInfo) {
  ElMessageBox.prompt(`请输入退款原因\n订单编号：${row.orderNo || `ID:${row.id}`}`, "申请退款", {
    confirmButtonText: "确定",
    cancelButtonText: "取消"
  }).then(
    () => {
      defOrderInfoService.RefundOrderInfo({ orderId: row.id, refundMoney: 0 }).then(() => {
        ElMessage.success("订单退款成功");
        proTable.value?.getTableList();
      });
    },
    () => {
      ElMessage.info("已取消订单退款");
    }
  );
}

/**
 * 打开退款弹窗并加载退款详情。
 */
function handleOpenRefundDialog(orderId: number, title: string) {
  resetRefundDialog();
  dialogRefund.visible = true;
  dialogRefund.title = title;
  defOrderInfoService.GetOrderInfoRefund({ value: orderId }).then(data => {
    formDataRefund.orderId = orderId;
    Object.assign(dataRefund, data);
  });
}

/**
 * 关闭退款弹窗并清理表单。
 */
function handleCloseRefundDialog() {
  dialogRefund.visible = false;
  resetRefundDialog();
}

/**
 * 重置退款弹窗数据，避免切换订单时残留上一条记录。
 */
function resetRefundDialog() {
  dataFormRefRefund.value?.resetFields();
  dataFormRefRefund.value?.clearValidate();
  formDataRefund.orderId = 0;
  formDataRefund.reason = undefined;
  formDataRefund.refundMoney = 0;
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
      submitData.refundMoney = submitData.refundMoney * 100;
      defOrderInfoService.RefundOrderInfo(submitData).then(() => {
        ElMessage.success("订单退款成功");
        handleCloseRefundDialog();
        proTable.value?.getTableList();
      });
    })
    .catch(() => undefined);
}

/**
 * 根据用户 ID 回显昵称，兜底旧数据场景。
 */
function formatUser(userId: number) {
  const entry = userOptions.value.find(item => Number(item.value) === userId);
  return entry ? entry.label : "-";
}

/**
 * 打开订单详情页。
 */
function handleOpenDetail(row: OrderInfo) {
  navigateTo(router, `/order/detail/${row.id}`, { title: `【${row.orderNo}】订单详情` });
}
</script>

<style scoped lang="scss">
.shipped-hero-card,
.shipped-section-card {
  border: 1px solid var(--admin-page-card-border);
  border-radius: 16px;
  background: var(--admin-page-card-bg);
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
  border: 1px solid var(--admin-page-card-border-soft);
  border-radius: 12px;
  background: var(--admin-page-card-bg-soft);
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
  align-items: center;
  justify-content: space-between;
  gap: 12px;
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
  margin-top: 20px;
  padding: 18px 18px 0;
  border-radius: 12px;
  background: var(--admin-page-card-bg-soft);
}

.refund-hero-card,
.refund-section-card {
  border: 1px solid var(--admin-page-card-border);
  border-radius: 16px;
  background: var(--admin-page-card-bg);
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
  border: 1px solid var(--admin-page-card-border-soft);
  border-radius: 12px;
  background: var(--admin-page-card-bg-soft);
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
  align-items: center;
  justify-content: space-between;
  gap: 12px;
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
