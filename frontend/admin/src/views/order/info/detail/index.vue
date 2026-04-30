<!-- 订单详情 -->
<template>
  <div v-loading="loading" class="app-container">
    <el-card v-if="formData.order" class="detail-hero-card" shadow="never">
      <div class="detail-hero">
        <!-- 顶部摘要仅保留核心指标，避免与下方基础信息区重复展示订单标题和编号。 -->
        <div class="detail-metrics">
          <div class="detail-metric-card">
            <span class="detail-metric-card__label">支付金额</span>
            <strong class="detail-metric-card__value">{{ formatPrice(formData.order.pay_money) }} 元</strong>
          </div>
          <div class="detail-metric-card">
            <span class="detail-metric-card__label">总金额</span>
            <strong class="detail-metric-card__value">{{ formatPrice(formData.order.total_money) }} 元</strong>
          </div>
          <div class="detail-metric-card">
            <span class="detail-metric-card__label">商品总数</span>
            <strong class="detail-metric-card__value">{{ formData.order.goods_num }}</strong>
          </div>
          <div class="detail-metric-card">
            <span class="detail-metric-card__label">运费</span>
            <strong class="detail-metric-card__value">{{ formatPrice(formData.order.post_fee) }} 元</strong>
          </div>
        </div>
      </div>
    </el-card>

    <div class="detail-grid">
      <el-card v-if="formData.order" class="detail-section-card" shadow="never">
        <template #header>
          <div class="detail-section-card__header">
            <span>订单基本信息</span>
          </div>
        </template>

        <el-descriptions :column="2" border class="detail-descriptions">
          <el-descriptions-item label="订单编号">
            <div class="order-no-field">
              <span>{{ formData.order.order_no }}</span>
              <el-button link type="primary" @click="handleCopyOrderNo(formData.order.order_no)">复制</el-button>
            </div>
          </el-descriptions-item>
          <el-descriptions-item label="用户">{{ formData.order.nick_name }}</el-descriptions-item>
          <el-descriptions-item label="支付方式">
            <DictLabel v-model="formData.order.pay_type" code="order_pay_type" />
          </el-descriptions-item>
          <el-descriptions-item label="支付渠道">
            <DictLabel v-model="formData.order.pay_channel" code="order_pay_channel" />
          </el-descriptions-item>
          <el-descriptions-item label="配送时间类型">
            <DictLabel v-model="formData.order.delivery_time" code="order_delivery_time" />
          </el-descriptions-item>
          <el-descriptions-item label="订单状态">
            <DictLabel v-model="formData.order.status" code="order_status" />
          </el-descriptions-item>
          <el-descriptions-item label="创建时间">{{ formData.order.created_at }}</el-descriptions-item>
          <el-descriptions-item label="更新时间">{{ formData.order.updated_at }}</el-descriptions-item>
          <el-descriptions-item label="备注" :span="2">{{ formData.order.remark || "-" }}</el-descriptions-item>
        </el-descriptions>
      </el-card>

      <el-card v-if="hasAddressSection" class="detail-section-card" shadow="never">
        <template #header>
          <div class="detail-section-card__header">
            <span>收货地址</span>
          </div>
        </template>

        <el-descriptions :column="1" border class="detail-descriptions">
          <el-descriptions-item label="联系人">{{ formData.address?.receiver }}</el-descriptions-item>
          <el-descriptions-item label="联系方式">{{ formData.address?.contact }}</el-descriptions-item>
          <el-descriptions-item label="地区">{{ formData.address?.address?.join(" / ") }}</el-descriptions-item>
          <el-descriptions-item label="详细地址">{{ formData.address?.detail }}</el-descriptions-item>
        </el-descriptions>
      </el-card>
    </div>

    <el-card v-if="formData.goods.length" class="detail-section-card detail-section-card--full" shadow="never">
      <template #header>
        <div class="detail-section-card__header">
          <span>商品清单</span>
          <span class="detail-section-card__extra">{{ formData.goods.length }} 项</span>
        </div>
      </template>

      <div class="goods-list-table">
        <ProTable row-key="sku_code" :data="formData.goods" :columns="goodsColumns" :pagination="false" :tool-button="false">
          <template #spec_item="scope">{{ scope.row.spec_item.join("/") }}</template>
        </ProTable>
      </div>
    </el-card>

    <div class="detail-grid">
      <el-card v-if="hasPaymentSection" class="detail-section-card" shadow="never">
        <template #header>
          <div class="detail-section-card__header">
            <span>支付信息</span>
          </div>
        </template>

        <el-descriptions :column="2" border class="detail-descriptions">
          <el-descriptions-item label="三方订单号">{{ formData.payment?.third_order_no }}</el-descriptions-item>
          <el-descriptions-item label="交易类型">{{ formData.payment?.trade_type }}</el-descriptions-item>
          <el-descriptions-item label="支付状态">{{ formData.payment?.trade_state_desc }}</el-descriptions-item>
          <el-descriptions-item label="支付时间">{{ formData.payment?.success_time }}</el-descriptions-item>
          <el-descriptions-item label="支付金额">
            {{ formatPrice(formData.payment?.amount?.payer_total) }} 元
          </el-descriptions-item>
          <el-descriptions-item label="总金额">{{ formatPrice(formData.payment?.amount?.total) }} 元</el-descriptions-item>
          <el-descriptions-item label="对帐状态" :span="2">
            <DictLabel :model-value="formData.payment?.status" code="order_bill_status" />
          </el-descriptions-item>
        </el-descriptions>
      </el-card>

      <el-card v-if="hasLogisticsSection" class="detail-section-card" shadow="never">
        <template #header>
          <div class="detail-section-card__header">
            <span>物流信息</span>
          </div>
        </template>

        <el-descriptions :column="2" border class="detail-descriptions">
          <el-descriptions-item label="物流公司">{{ formData.logistics?.name }}</el-descriptions-item>
          <el-descriptions-item label="物流单号">{{ formData.logistics?.no }}</el-descriptions-item>
          <el-descriptions-item label="联系方式">{{ formData.logistics?.contact }}</el-descriptions-item>
          <el-descriptions-item label="发货时间">{{ formData.logistics?.created_at }}</el-descriptions-item>
        </el-descriptions>

        <el-timeline class="detail-timeline">
          <el-timeline-item
            v-for="(detail, index) in formData.logistics?.detail ?? []"
            :key="index"
            :timestamp="detail.time"
            placement="top"
          >
            {{ detail.text }}
          </el-timeline-item>
        </el-timeline>
      </el-card>
    </div>

    <el-card v-if="formData.refund.length" class="detail-section-card detail-section-card--full" shadow="never">
      <template #header>
        <div class="detail-section-card__header">
          <span>退款信息</span>
          <span class="detail-section-card__extra">{{ formData.refund.length }} 条</span>
        </div>
      </template>
      <ProTable row-key="refund_no" :data="formData.refund" :columns="refundColumns" :pagination="false" :tool-button="false" />
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { computed, onActivated, reactive, ref, watch } from "vue";
import { useRoute } from "vue-router";
import type { ColumnProps } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
import { type OrderInfoResponse } from "@/rpc/admin/v1/order_info";
import { defOrderInfoService } from "@/api/admin/order_info";
import { useTabsStore } from "@/stores/modules/tabs";
import { formatPrice } from "@/utils/utils";

defineOptions({
  name: "OrderDetail",
  inheritAttrs: false
});

const route = useRoute();
const tabsStore = useTabsStore();
const loading = ref(false);

const orderId = ref(0);
const orderDetailRequestId = ref(0);
const formData = reactive<OrderInfoResponse>({
  /** 订单信息 */
  order: undefined,
  /** 支付倒计时 */
  countdown: 0,
  /** 地址信息 */
  address: undefined,
  /** 取消信息 */
  cancel: undefined,
  /** 商品信息 */
  goods: [],
  /** 物流信息 */
  logistics: undefined,
  /** 支付信息 */
  payment: undefined,
  /** 退款信息 */
  refund: []
});

/** 重置订单详情数据，避免切换订单时短暂显示旧内容。 */
function resetOrderDetailForm() {
  Object.assign(formData, {
    order: undefined,
    countdown: 0,
    address: undefined,
    cancel: undefined,
    goods: [],
    logistics: undefined,
    payment: undefined,
    refund: []
  });
}

/** 从路由中同步订单ID，统一兼容 params 字符串场景。 */
function syncOrderIdFromRoute() {
  orderId.value = Number(route.params.orderId ?? 0);
  return orderId.value;
}

/** 订单详情工作区标题固定为“订单详情”，避免依赖查询参数回填标题。 */
const workspaceTitle = "订单详情";

/** 收货地址存在有效内容时才展示地址模块。 */
const hasAddressSection = computed(() => {
  return Boolean(formData.address && (formData.address.receiver || formData.address.contact || formData.address.detail));
});

/** 支付信息存在有效内容时才展示支付模块。 */
const hasPaymentSection = computed(() => {
  return Boolean(
    formData.payment &&
    (formData.payment.third_order_no ||
      formData.payment.trade_type ||
      formData.payment.trade_state_desc ||
      formData.payment.success_time)
  );
});

/** 物流信息存在有效内容时才展示物流模块。 */
const hasLogisticsSection = computed(() => {
  return Boolean(
    formData.logistics &&
    (formData.logistics.name || formData.logistics.no || formData.logistics.contact || formData.logistics.created_at)
  );
});

/** 订单商品明细表格列配置。 */
const goodsColumns: ColumnProps[] = [
  { prop: "name", label: "商品名称", minWidth: 180 },
  { prop: "sku_code", label: "规格编号", minWidth: 140 },
  { prop: "spec_item", label: "规格名称", minWidth: 160 },
  { prop: "num", label: "数量", align: "right", minWidth: 90 },
  { prop: "price", label: "单价", align: "right", minWidth: 110, cellType: "money" },
  { prop: "pay_price", label: "支付价", align: "right", minWidth: 110, cellType: "money" },
  { prop: "total_pay_price", label: "总金额", align: "right", minWidth: 110, cellType: "money" }
];

/** 订单退款明细表格列配置。 */
const refundColumns: ColumnProps[] = [
  { prop: "third_order_no", label: "三方支付订单编号", align: "center", minWidth: 180 },
  { prop: "refund_no", label: "退款编号", align: "center", minWidth: 160 },
  { prop: "third_refund_no", label: "三方退款编号", align: "center", minWidth: 180 },
  { prop: "reason", label: "退款原因", minWidth: 160 },
  { prop: "channel", label: "退款渠道", align: "center", minWidth: 120 },
  { prop: "user_received_account", label: "退款入账账户", align: "center", minWidth: 160 },
  { prop: "funds_account", label: "资金账户类型", align: "center", minWidth: 140 },
  {
    prop: "amount.payer_refund",
    label: "退款金额",
    minWidth: 110,
    align: "right",
    cellType: "money",
    moneyProps: { value: scope => scope.row.amount?.payer_refund }
  },
  {
    prop: "amount.total",
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

// 监听路由参数变化
watch(
  () => route.params.orderId,
  () => {
    const currentOrderId = syncOrderIdFromRoute();
    syncWorkspaceTitle();
    if (!currentOrderId) {
      resetOrderDetailForm();
      return;
    }
    handleQuery(currentOrderId);
  },
  { immediate: true }
);

/** 同步当前页签和浏览器标题，确保列表点击进入详情时无需刷新即可生效。 */
function syncWorkspaceTitle() {
  tabsStore.setTabsTitle(workspaceTitle);
  document.title = `${workspaceTitle} - ${import.meta.env.VITE_GLOB_APP_TITLE}`;
}

// 查询
function handleQuery(targetOrderId: number = orderId.value) {
  if (!targetOrderId) return;
  const requestId = ++orderDetailRequestId.value;
  loading.value = true;
  defOrderInfoService
    .GetOrderInfo({
      id: targetOrderId
    })
    .then(data => {
      if (requestId !== orderDetailRequestId.value) return;
      resetOrderDetailForm();
      Object.assign(formData, data);
    })
    .finally(() => {
      if (requestId !== orderDetailRequestId.value) return;
      loading.value = false;
    });
}

/**
 * 复制订单编号，便于客服或运营快速粘贴查询。
 */
async function handleCopyOrderNo(order_no: string) {
  if (!order_no) {
    ElMessage.warning("订单编号为空，无法复制");
    return;
  }

  try {
    await navigator.clipboard.writeText(order_no);
    ElMessage.success("订单编号已复制");
  } catch {
    ElMessage.error("复制失败，请手动复制");
  }
}

onActivated(() => {
  syncWorkspaceTitle();
  const currentOrderId = syncOrderIdFromRoute();
  if (!currentOrderId || loading.value) return;
  handleQuery(currentOrderId);
});
</script>

<style scoped lang="scss">
.detail-hero-card,
.detail-section-card {
  border: 1px solid var(--admin-page-card-border);
  border-radius: var(--admin-page-radius);
  background: var(--admin-page-card-bg);
  box-shadow: var(--admin-page-shadow);
}

.detail-hero-card {
  margin-bottom: 18px;
}

:deep(.detail-hero-card .el-card__body),
:deep(.detail-section-card .el-card__body) {
  padding: 16px;
}

.detail-hero {
  display: block;
}

.detail-metrics {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}

.detail-metric-card {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 14px;
  border: 1px solid var(--admin-page-card-border-soft);
  border-radius: var(--admin-page-radius);
  background: var(--admin-page-card-bg-soft);
}

.detail-metric-card__label {
  font-size: 13px;
  color: var(--admin-page-text-secondary);
}

.detail-metric-card__value {
  font-size: 20px;
  font-weight: 700;
  color: var(--admin-page-text-primary);
}

.detail-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
  margin-bottom: 16px;
}

.detail-section-card {
  overflow: hidden;
}

.detail-section-card--full {
  margin-bottom: 16px;
}

.detail-section-card__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  font-size: 16px;
  font-weight: 600;
  color: var(--admin-page-text-primary);
}

.detail-section-card__extra {
  font-size: 13px;
  font-weight: 500;
  color: var(--admin-page-text-placeholder);
}

.detail-descriptions :deep(.el-descriptions__label) {
  width: 110px;
  font-weight: 600;
}

.detail-descriptions :deep(.el-descriptions__cell) {
  padding: 10px 14px;
}

.order-no-field {
  display: inline-flex;
  gap: 8px;
  align-items: center;
  word-break: break-all;
}

.detail-timeline {
  margin-top: 20px;
  padding: 18px 18px 0;
  border-radius: var(--admin-page-radius);
  background: var(--admin-page-card-bg-soft);
}

/* 商品清单按实际行数撑开，避免继承通用表格的固定最小高度。 */
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
  .detail-grid {
    grid-template-columns: 1fr;
  }
}

@media (width <= 768px) {
  .detail-metrics {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (width <= 520px) {
  .detail-metrics {
    grid-template-columns: 1fr;
  }
}
</style>
