<!-- 商品属性 -->
<template>
  <div v-loading="loading" class="app-container">
    <!-- 基础信息 -->
    <el-card v-if="formData.order" class="box-card" shadow="never">
      <template #header>
        <div class="card-header">
          <span>订单基本信息</span>
        </div>
      </template>

      <el-descriptions :column="2" border>
        <el-descriptions-item label="订单编号">{{ formData.order.orderNo }}</el-descriptions-item>
        <el-descriptions-item label="用户">{{ formData.order.nickName }}</el-descriptions-item>
        <el-descriptions-item label="支付金额"> {{ formatPrice(formData.order.payMoney) }} 元 </el-descriptions-item>
        <el-descriptions-item label="总金额"> {{ formatPrice(formData.order.totalMoney) }} 元 </el-descriptions-item>
        <el-descriptions-item label="运费"> {{ formatPrice(formData.order.postFee) }} 元 </el-descriptions-item>
        <el-descriptions-item label="商品总数">{{ formData.order.goodsNum }}</el-descriptions-item>
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
        <el-descriptions-item label="备注" :span="2">
          {{ formData.order.remark || "-" }}
        </el-descriptions-item>
        <el-descriptions-item label="创建时间">
          {{ formData.order.createdAt }}
        </el-descriptions-item>
        <el-descriptions-item label="更新时间">
          {{ formData.order.updatedAt }}
        </el-descriptions-item>
      </el-descriptions>
    </el-card>

    <!-- 地址信息 -->
    <el-card v-if="formData.address" class="box-card" shadow="never">
      <template #header>
        <div class="card-header">
          <span>收货地址</span>
        </div>
      </template>

      <el-descriptions :column="1" border>
        <el-descriptions-item label="联系人">{{ formData.address.receiver }}</el-descriptions-item>
        <el-descriptions-item label="联系方式">
          {{ formData.address.contact }}
        </el-descriptions-item>
        <el-descriptions-item label="地区">
          {{ formData.address.address.join(" / ") }}
        </el-descriptions-item>
        <el-descriptions-item label="详细地址">{{ formData.address.detail }}</el-descriptions-item>
      </el-descriptions>
    </el-card>

    <!-- 商品信息 -->
    <el-card v-if="formData.goods.length" class="box-card" shadow="never">
      <template #header>
        <div class="card-header">
          <span>商品清单</span>
        </div>
      </template>

      <ProTable row-key="skuCode" :data="formData.goods" :columns="goodsColumns" :pagination="false" :tool-button="false">
        <template #specItem="scope">{{ scope.row.specItem.join("/") }}</template>
      </ProTable>
    </el-card>

    <!-- 支付信息 -->
    <el-card v-if="formData.payment" class="box-card" shadow="never">
      <template #header>
        <div class="card-header">
          <span>支付信息</span>
        </div>
      </template>

      <el-descriptions :column="2" border>
        <el-descriptions-item label="三方订单号" align="center">
          {{ formData.payment.thirdOrderNo }}
        </el-descriptions-item>
        <el-descriptions-item label="交易类型" align="center">
          {{ formData.payment.tradeType }}
        </el-descriptions-item>
        <el-descriptions-item label="支付状态" align="center">
          {{ formData.payment.tradeStateDesc }}
        </el-descriptions-item>
        <el-descriptions-item label="支付时间" align="center">
          {{ formData.payment.successTime }}
        </el-descriptions-item>
        <el-descriptions-item label="支付金额" align="right">
          {{ formatPrice(formData.payment.amount?.payerTotal) }} 元
        </el-descriptions-item>
        <el-descriptions-item label="总金额" align="right">
          {{ formatPrice(formData.payment.amount?.total) }} 元
        </el-descriptions-item>
        <el-descriptions-item label="对帐状态" align="right">
          <DictLabel v-model="formData.payment.status" code="order_bill_status" />
        </el-descriptions-item>
      </el-descriptions>
    </el-card>

    <!-- 物流信息 -->
    <el-card v-if="formData.logistics" class="box-card" shadow="never">
      <template #header>
        <div class="card-header">
          <span>物流信息</span>
        </div>
      </template>

      <el-descriptions :column="2" border>
        <el-descriptions-item label="物流公司">{{ formData.logistics.name }}</el-descriptions-item>
        <el-descriptions-item label="物流单号">{{ formData.logistics.no }}</el-descriptions-item>
        <el-descriptions-item label="联系方式">
          {{ formData.logistics.contact }}
        </el-descriptions-item>
        <el-descriptions-item label="发货时间">
          {{ formData.logistics.createdAt }}
        </el-descriptions-item>
      </el-descriptions>

      <el-timeline style="margin-top: 20px">
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

    <!-- 退款信息 -->
    <el-card v-if="formData.refund.length" class="box-card" shadow="never">
      <template #header>
        <div class="card-header">
          <span>退款信息</span>
        </div>
      </template>
      <ProTable row-key="refundNo" :data="formData.refund" :columns="refundColumns" :pagination="false" :tool-button="false" />
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref, watch } from "vue";
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

const goodsColumns: ColumnProps[] = [
  { prop: "name", label: "商品名称" },
  { prop: "skuCode", label: "规格编号" },
  { prop: "specItem", label: "规格名称" },
  { prop: "num", label: "数量", align: "right" },
  { prop: "price", label: "单价", align: "right", cellType: "money" },
  { prop: "payPrice", label: "支付价", align: "right", cellType: "money" },
  { prop: "totalPayPrice", label: "总金额", align: "right", cellType: "money" }
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
    prop: "amount.payerRefund",
    label: "退款金额",
    align: "right",
    cellType: "money",
    moneyProps: { value: scope => scope.row.amount?.payerRefund }
  },
  {
    prop: "amount.total",
    label: "原订单金额",
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
