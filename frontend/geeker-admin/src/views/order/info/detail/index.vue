<!-- 订单详情 -->
<template>
  <div v-loading="loading" class="app-container order-detail-page">
    <el-card v-if="formData.order" class="detail-hero-card" shadow="never">
      <div class="detail-hero">
        <div class="detail-hero__main">
          <div class="detail-hero__eyebrow">订单详情</div>
          <div class="detail-hero__title-row">
            <h1 class="detail-hero__title">{{ formData.order.orderNo }}</h1>
            <div class="detail-hero__status">
              <DictLabel v-model="formData.order.status" code="order_status" />
            </div>
          </div>
          <p class="detail-hero__desc">用户：{{ formData.order.nickName || "-" }}</p>
          <p v-if="formData.order.remark" class="detail-hero__remark">备注：{{ formData.order.remark }}</p>
        </div>

        <div class="detail-metrics">
          <div class="detail-metric-card">
            <span class="detail-metric-card__label">支付金额</span>
            <strong class="detail-metric-card__value">{{ formatPrice(formData.order.payMoney) }} 元</strong>
          </div>
          <div class="detail-metric-card">
            <span class="detail-metric-card__label">总金额</span>
            <strong class="detail-metric-card__value">{{ formatPrice(formData.order.totalMoney) }} 元</strong>
          </div>
          <div class="detail-metric-card">
            <span class="detail-metric-card__label">商品总数</span>
            <strong class="detail-metric-card__value">{{ formData.order.goodsNum }}</strong>
          </div>
          <div class="detail-metric-card">
            <span class="detail-metric-card__label">运费</span>
            <strong class="detail-metric-card__value">{{ formatPrice(formData.order.postFee) }} 元</strong>
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
          <el-descriptions-item label="订单编号">{{ formData.order.orderNo }}</el-descriptions-item>
          <el-descriptions-item label="用户">{{ formData.order.nickName }}</el-descriptions-item>
          <el-descriptions-item label="支付方式">
            <DictLabel v-model="formData.order.payType" code="order_pay_type" />
          </el-descriptions-item>
          <el-descriptions-item label="支付渠道">
            <DictLabel v-model="formData.order.payChannel" code="order_pay_channel" />
          </el-descriptions-item>
          <el-descriptions-item label="配送时间类型">
            <DictLabel v-model="formData.order.deliveryTime" code="order_delivery_time" />
          </el-descriptions-item>
          <el-descriptions-item label="订单状态">
            <DictLabel v-model="formData.order.status" code="order_status" />
          </el-descriptions-item>
          <el-descriptions-item label="创建时间">{{ formData.order.createdAt }}</el-descriptions-item>
          <el-descriptions-item label="更新时间">{{ formData.order.updatedAt }}</el-descriptions-item>
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
          <el-descriptions-item label="联系人">{{ formData.address.receiver }}</el-descriptions-item>
          <el-descriptions-item label="联系方式">{{ formData.address.contact }}</el-descriptions-item>
          <el-descriptions-item label="地区">{{ formData.address.address.join(" / ") }}</el-descriptions-item>
          <el-descriptions-item label="详细地址">{{ formData.address.detail }}</el-descriptions-item>
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
        <ProTable row-key="skuCode" :data="formData.goods" :columns="goodsColumns" :pagination="false" :tool-button="false">
          <template #specItem="scope">{{ scope.row.specItem.join("/") }}</template>
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
          <el-descriptions-item label="三方订单号">{{ formData.payment.thirdOrderNo }}</el-descriptions-item>
          <el-descriptions-item label="交易类型">{{ formData.payment.tradeType }}</el-descriptions-item>
          <el-descriptions-item label="支付状态">{{ formData.payment.tradeStateDesc }}</el-descriptions-item>
          <el-descriptions-item label="支付时间">{{ formData.payment.successTime }}</el-descriptions-item>
          <el-descriptions-item label="支付金额">{{ formatPrice(formData.payment.amount?.payerTotal) }} 元</el-descriptions-item>
          <el-descriptions-item label="总金额">{{ formatPrice(formData.payment.amount?.total) }} 元</el-descriptions-item>
          <el-descriptions-item label="对帐状态" :span="2">
            <DictLabel v-model="formData.payment.status" code="order_bill_status" />
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
          <el-descriptions-item label="物流公司">{{ formData.logistics.name }}</el-descriptions-item>
          <el-descriptions-item label="物流单号">{{ formData.logistics.no }}</el-descriptions-item>
          <el-descriptions-item label="联系方式">{{ formData.logistics.contact }}</el-descriptions-item>
          <el-descriptions-item label="发货时间">{{ formData.logistics.createdAt }}</el-descriptions-item>
        </el-descriptions>

        <el-timeline class="detail-timeline">
          <el-timeline-item
            v-for="(detail, index) in formData.logistics.detail"
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
      <ProTable row-key="refundNo" :data="formData.refund" :columns="refundColumns" :pagination="false" :tool-button="false" />
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from "vue";
import { useRoute } from "vue-router";
import type { ColumnProps } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
import { type OrderResponse } from "@/rpc/admin/order";
import { defOrderService } from "@/api/admin/order";
import { formatPrice } from "@/utils/utils";

defineOptions({
  name: "OrderDetail",
  inheritAttrs: false
});

const route = useRoute();
const loading = ref(false);

const orderId = ref(route.params.orderId as unknown as number);
const formData = reactive<OrderResponse>({
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

/** 收货地址存在有效内容时才展示地址模块。 */
const hasAddressSection = computed(() => {
  return Boolean(formData.address && (formData.address.receiver || formData.address.contact || formData.address.detail));
});

/** 支付信息存在有效内容时才展示支付模块。 */
const hasPaymentSection = computed(() => {
  return Boolean(
    formData.payment &&
    (formData.payment.thirdOrderNo ||
      formData.payment.tradeType ||
      formData.payment.tradeStateDesc ||
      formData.payment.successTime)
  );
});

/** 物流信息存在有效内容时才展示物流模块。 */
const hasLogisticsSection = computed(() => {
  return Boolean(
    formData.logistics &&
    (formData.logistics.name || formData.logistics.no || formData.logistics.contact || formData.logistics.createdAt)
  );
});

/** 订单商品明细表格列配置。 */
const goodsColumns: ColumnProps[] = [
  { prop: "name", label: "商品名称", minWidth: 180 },
  { prop: "skuCode", label: "规格编号", minWidth: 140 },
  { prop: "specItem", label: "规格名称", minWidth: 160 },
  { prop: "num", label: "数量", align: "right", minWidth: 90 },
  { prop: "price", label: "单价", align: "right", minWidth: 110, cellType: "money" },
  { prop: "payPrice", label: "支付价", align: "right", minWidth: 110, cellType: "money" },
  { prop: "totalPayPrice", label: "总金额", align: "right", minWidth: 110, cellType: "money" }
];

/** 订单退款明细表格列配置。 */
const refundColumns: ColumnProps[] = [
  { prop: "thirdOrderNo", label: "三方支付订单编号", align: "center", minWidth: 180 },
  { prop: "refundNo", label: "退款编号", align: "center", minWidth: 160 },
  { prop: "thirdRefundNo", label: "三方退款编号", align: "center", minWidth: 180 },
  { prop: "reason", label: "退款原因", minWidth: 160 },
  { prop: "channel", label: "退款渠道", align: "center", minWidth: 120 },
  { prop: "userReceivedAccount", label: "退款入账账户", align: "center", minWidth: 160 },
  { prop: "fundsAccount", label: "资金账户类型", align: "center", minWidth: 140 },
  {
    prop: "amount.payerRefund",
    label: "退款金额",
    minWidth: 110,
    align: "right",
    cellType: "money",
    moneyProps: { value: scope => scope.row.amount?.payerRefund }
  },
  {
    prop: "amount.total",
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

// 监听路由参数变化
watch(
  () => [route.params.orderId],
  ([newOrderId]) => {
    orderId.value = newOrderId as unknown as number;
    if (orderId.value) {
      handleQuery();
    }
  }
);

// 查询
function handleQuery() {
  loading.value = true;
  defOrderService
    .GetOrder({
      value: orderId.value
    })
    .then(data => {
      Object.assign(formData, data);
    })
    .finally(() => {
      loading.value = false;
    });
}

onMounted(() => {
  handleQuery();
});
</script>

<style scoped lang="scss">
.order-detail-page {
  padding-bottom: 24px;
  background:
    radial-gradient(circle at top left, rgb(240 247 255 / 95%), transparent 34%),
    linear-gradient(180deg, #f6f8fb 0%, #f3f5f7 100%);
}

.detail-hero-card,
.detail-section-card {
  border: 1px solid #e5eaf1;
  border-radius: 24px;
  box-shadow: 0 18px 40px rgb(15 23 42 / 6%);
}

.detail-hero-card {
  margin-bottom: 18px;
  overflow: hidden;
  background: linear-gradient(135deg, rgb(255 255 255 / 98%) 0%, rgb(241 247 255 / 95%) 100%);
}

.detail-hero {
  display: grid;
  grid-template-columns: minmax(0, 1.2fr) minmax(320px, 0.8fr);
  gap: 24px;
  align-items: stretch;
}

.detail-hero__main {
  display: flex;
  flex-direction: column;
  gap: 14px;
  justify-content: center;
  min-width: 0;
}

.detail-hero__eyebrow {
  font-size: 13px;
  font-weight: 600;
  letter-spacing: 0.08em;
  color: #6a7890;
  text-transform: uppercase;
}

.detail-hero__title-row {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  align-items: center;
}

.detail-hero__title {
  margin: 0;
  font-size: 30px;
  font-weight: 700;
  line-height: 1.2;
  color: #1f2937;
  word-break: break-all;
}

.detail-hero__status {
  display: inline-flex;
  align-items: center;
  padding: 8px 14px;
  border: 1px solid #d9e4f1;
  border-radius: 999px;
  background: rgb(255 255 255 / 82%);
  font-size: 14px;
  font-weight: 600;
  color: #334155;
}

.detail-hero__desc,
.detail-hero__remark {
  margin: 0;
  font-size: 15px;
  line-height: 1.8;
  color: #526071;
}

.detail-metrics {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
}

.detail-metric-card {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 18px;
  border: 1px solid #e5eaf1;
  border-radius: 18px;
  background: rgb(255 255 255 / 84%);
}

.detail-metric-card__label {
  font-size: 13px;
  color: #718096;
}

.detail-metric-card__value {
  font-size: 22px;
  font-weight: 700;
  color: #0f172a;
}

.detail-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 18px;
  margin-bottom: 18px;
}

.detail-section-card {
  overflow: hidden;
}

.detail-section-card--full {
  margin-bottom: 18px;
}

.detail-section-card__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  font-size: 16px;
  font-weight: 600;
  color: #1f2937;
}

.detail-section-card__extra {
  font-size: 13px;
  font-weight: 500;
  color: #7c8aa0;
}

.detail-descriptions :deep(.el-descriptions__label) {
  width: 120px;
  font-weight: 600;
}

.detail-timeline {
  margin-top: 20px;
  padding: 18px 18px 0;
  border-radius: 18px;
  background: #f8fafc;
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
  .detail-hero,
  .detail-grid {
    grid-template-columns: 1fr;
  }
}

@media (width <= 768px) {
  .order-detail-page {
    padding-bottom: 12px;
  }

  .detail-hero__title {
    font-size: 24px;
  }

  .detail-metrics {
    grid-template-columns: 1fr;
  }
}
</style>
