<template>
  <div class="app-container">
    <div class="search-bar">
      <el-form ref="queryFormRef" :model="queryParams" :inline="true">
        <el-form-item label="用户" prop="userId">
          <el-select v-model="queryParams.userId" placeholder="请选择">
            <el-option
              v-for="item in userOptions"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </el-select>
        </el-form-item>
        <el-form-item prop="orderNo" label="订单编号">
          <el-input
            v-model="queryParams.orderNo"
            placeholder="订单编号"
            clearable
            @keyup.enter="handleQuery"
          />
        </el-form-item>
        <el-form-item label="支付方式" prop="OrderPayType">
          <Dict v-model="queryParams.payType" code="order_pay_type" />
        </el-form-item>
        <el-form-item label="支付渠道" prop="payChannel">
          <Dict v-model="queryParams.payChannel" code="order_pay_channel" />
        </el-form-item>
        <el-form-item v-if="!props.status" label="状态" prop="status">
          <Dict v-model="queryParams.status" code="order_status" />
        </el-form-item>
        <el-form-item prop="requestTime" label="操作时间">
          <el-date-picker
            v-model="queryParams.createdAt"
            :editable="false"
            class="!w-[240px]"
            type="daterange"
            range-separator="~"
            start-placeholder="开始时间"
            end-placeholder="截止时间"
            value-format="YYYY-MM-DD"
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
        <el-table-column label="订单编号">
          <template #default="scope">
            <el-link
              v-if="useUserStore().hasPerm('order:info:detail')"
              type="primary"
              @click.stop="handleOpenDetail(scope.row)"
            >
              {{ scope.row.orderNo }}
            </el-link>
            <span v-else>{{ scope.row.orderNo }}</span>
          </template>
        </el-table-column>
        <el-table-column label="用户">
          <template #default="scope">
            <span>{{ formatUser(scope.row.userId) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="金额（元）" align="right">
          <template #default="scope">
            <el-popover effect="light" trigger="hover" placement="top" width="auto">
              <template #default>
                <div>总价:{{ formatPrice(scope.row.totalMoney) }}</div>
                <div>实付: {{ formatPrice(scope.row.payMoney) }}</div>
                <div>运费: {{ formatPrice(scope.row.postFee) }}</div>
              </template>
              <template #reference>
                {{ formatPrice(scope.row.payMoney) }}
              </template>
            </el-popover>
          </template>
        </el-table-column>

        <el-table-column label="支付方式" align="center">
          <template #default="scope">
            <DictLabel v-model="scope.row.OrderPayType" code="order_pay_type" />
          </template>
        </el-table-column>

        <el-table-column label="支付渠道" align="center">
          <template #default="scope">
            <DictLabel v-model="scope.row.payChannel" code="order_pay_channel" />
          </template>
        </el-table-column>

        <el-table-column label="状态" align="center">
          <template #default="scope">
            <DictLabel v-model="scope.row.status" code="order_status" />
          </template>
        </el-table-column>

        <el-table-column prop="goodsNum" label="商品数" align="right" />
        <el-table-column prop="createdAt" label="创建时间" align="center" />
        <el-table-column fixed="right" label="操作" width="200" align="center">
          <template #default="scope">
            <el-button
              v-if="scope.row.statusCode === OrderStatus.PAID"
              v-hasPerm="['order:info:shipped']"
              type="primary"
              size="small"
              link
              icon="info"
              @click="handleOpenShippedDialog(scope.row.id, '发货')"
            >
              发货
            </el-button>
            <el-button
              v-if="
                scope.row.statusCode === OrderStatus.SHIPPED ||
                scope.row.statusCode === OrderStatus.RECEIVED
              "
              v-hasPerm="['order:info:shipped']"
              type="primary"
              size="small"
              link
              icon="info"
              @click="handleOpenShippedDialog(scope.row.id, '发货详情')"
            >
              发货详情
            </el-button>

            <el-button
              v-if="
                scope.row.OrderPayTypeCode === OrderPayType.CASH_ON_DELIVERY &&
                (scope.row.statusCode === OrderStatus.SHIPPED ||
                  scope.row.statusCode === OrderStatus.RECEIVED)
              "
              v-hasPerm="['order:info:refund']"
              type="danger"
              size="small"
              link
              icon="info"
              @click="handleRefund(scope.row)"
            >
              退款
            </el-button>

            <el-button
              v-if="
                scope.row.OrderPayTypeCode === OrderPayType.ONLINE_PAY &&
                (scope.row.statusCode === OrderStatus.SHIPPED ||
                  scope.row.statusCode === OrderStatus.RECEIVED ||
                  scope.row.statusCode === OrderStatus.REFUNDING)
              "
              v-hasPerm="['order:info:refund']"
              type="danger"
              size="small"
              link
              icon="info"
              @click="handleOpenRefundDialog(scope.row.id, '退款')"
            >
              退款
            </el-button>
            <el-button
              v-if="scope.row.statusCode === OrderStatus.REFUNDING"
              v-hasPerm="['order:info:refund']"
              type="danger"
              size="small"
              link
              icon="info"
              @click="handleOpenRefundDialog(scope.row.id, '退款详情')"
            >
              退款详情
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

    <!--发货弹窗-->
    <el-dialog
      v-model="dialogShipped.visible"
      :title="dialogShipped.title"
      width="1200px"
      @close="handleCloseShippedDialog"
    >
      <!-- 地址信息 -->
      <el-card class="box-card" shadow="never">
        <template #header>
          <div class="card-header">
            <span>收货地址</span>
          </div>
        </template>

        <el-descriptions :column="1" border>
          <el-descriptions-item label="联系人">
            {{ dataShipped.address?.receiver }}
          </el-descriptions-item>
          <el-descriptions-item label="联系方式">
            {{ dataShipped.address?.contact }}
          </el-descriptions-item>
          <el-descriptions-item label="地区">
            {{ dataShipped.address?.address?.join(" / ") }}
          </el-descriptions-item>
          <el-descriptions-item label="详细地址">
            {{ dataShipped.address?.detail }}
          </el-descriptions-item>
        </el-descriptions>
      </el-card>

      <!-- 商品信息 -->
      <el-card class="box-card" shadow="never">
        <template #header>
          <div class="card-header">
            <span>商品清单</span>
          </div>
        </template>

        <el-table :data="dataShipped.goods" border stripe>
          <el-table-column prop="name" label="商品名称" />
          <el-table-column prop="skuCode" label="规格编号" />
          <el-table-column label="规格名称">
            <template #default="{ row }">{{ row.specItem?.join(" ") }}</template>
          </el-table-column>
          <el-table-column prop="num" label="数量" />
          <el-table-column label="单价">
            <template #default="{ row }">{{ formatPrice(row.price) }}</template>
          </el-table-column>
          <el-table-column label="支付价">
            <template #default="{ row }">{{ formatPrice(row.payPrice) }}</template>
          </el-table-column>
          <el-table-column label="总金额">
            <template #default="{ row }">{{ formatPrice(row.totalPayPrice) }}</template>
          </el-table-column>
        </el-table>
      </el-card>

      <!-- 物流信息 -->
      <el-card v-if="dataShipped.logistics" class="box-card" shadow="never">
        <template #header>
          <div class="card-header">
            <span>物流信息</span>
          </div>
        </template>

        <el-descriptions :column="2" border>
          <el-descriptions-item label="物流公司">
            {{ dataShipped.logistics.name }}
          </el-descriptions-item>
          <el-descriptions-item label="物流单号">
            {{ dataShipped.logistics.no }}
          </el-descriptions-item>
          <el-descriptions-item label="联系方式">
            {{ dataShipped.logistics.contact }}
          </el-descriptions-item>
          <el-descriptions-item label="发货时间">
            {{ dataShipped.logistics.createdAt }}
          </el-descriptions-item>
        </el-descriptions>

        <el-timeline style="margin-top: 20px">
          <el-timeline-item
            v-for="(detail, index) in dataShipped.logistics.detail"
            :key="index"
            :timestamp="detail.time"
            placement="top"
          >
            {{ detail.text }}
          </el-timeline-item>
        </el-timeline>
      </el-card>

      <el-form
        v-else
        ref="dataFormRefShipped"
        :model="formDataShipped"
        :rules="rulesShipped"
        label-width="150px"
      >
        <el-card shadow="never">
          <template #header>
            <div class="card-header">
              <span>物流信息</span>
            </div>
          </template>
          <el-form-item label="物流公司名称" prop="name">
            <el-input v-model="formDataShipped.name" placeholder="请输入物流公司名称" />
          </el-form-item>

          <el-form-item label="物流单号" prop="no">
            <el-input v-model="formDataShipped.no" placeholder="请输入物流单号" />
          </el-form-item>

          <el-form-item label="联系方式" prop="contact">
            <el-input v-model="formDataShipped.contact" placeholder="请输入联系方式" />
          </el-form-item>
        </el-card>
      </el-form>

      <template #footer>
        <div class="dialog-footer">
          <el-button type="primary" @click="handleShippedSubmitClick">确 定</el-button>
          <el-button @click="handleCloseShippedDialog">取 消</el-button>
        </div>
      </template>
    </el-dialog>

    <!--退款弹窗-->
    <el-dialog
      v-model="dialogRefund.visible"
      :title="dialogRefund.title"
      width="1200px"
      @close="handleCloseRefundDialog"
    >
      <!-- 支付信息 -->
      <el-card v-if="dataRefund.payment" class="box-card" shadow="never">
        <template #header>
          <div class="card-header">
            <span>支付信息</span>
          </div>
        </template>
        <el-descriptions :column="2" border>
          <el-descriptions-item label="三方订单号" align="center">
            {{ dataRefund.payment.thirdOrderNo }}
          </el-descriptions-item>
          <el-descriptions-item label="交易类型" align="center">
            {{ dataRefund.payment.tradeType }}
          </el-descriptions-item>
          <el-descriptions-item label="支付状态" align="center">
            {{ dataRefund.payment.tradeState }}
          </el-descriptions-item>
          <el-descriptions-item label="支付状态描述" align="center">
            {{ dataRefund.payment.tradeStateDesc }}
          </el-descriptions-item>
          <el-descriptions-item label="支付时间" align="center">
            {{ dataRefund.payment.successTime }}
          </el-descriptions-item>
          <el-descriptions-item label="支付金额" align="right">
            {{ formatPrice(dataRefund.payment.amount?.payerTotal) }} 元
          </el-descriptions-item>
          <el-descriptions-item label="总金额" align="right">
            {{ formatPrice(dataRefund.payment.amount?.total) }} 元
          </el-descriptions-item>
          <el-descriptions-item label="对帐状态" align="right">
            <DictLabel v-model="dataRefund.payment.status" code="order_bill_status" />
          </el-descriptions-item>
        </el-descriptions>
      </el-card>
      <!-- 退款信息 -->
      <el-card v-if="dataRefund.refund.length" class="box-card" shadow="never">
        <template #header>
          <div class="card-header">
            <span>退款信息</span>
          </div>
        </template>
        <el-table :data="dataRefund.refund" border stripe>
          <el-table-column prop="thirdOrderNo" label="三方支付订单编号" align="center" />
          <el-table-column prop="refundNo" label="退款编号" align="center" />
          <el-table-column prop="thirdRefundNo" label="三方退款编号" align="center" />
          <el-table-column prop="reason" label="退款原因" />
          <el-table-column prop="channel" label="退款渠道" align="center" />
          <el-table-column prop="userReceivedAccount" label="退款入账账户" align="center" />
          <el-table-column prop="fundsAccount" label="资金账户类型" align="center" />
          <el-table-column label="退款金额" align="right">
            <template #default="{ row }">{{ formatPrice(row.amount?.payerRefund) }}</template>
          </el-table-column>
          <el-table-column label="原订单金额" align="right">
            <template #default="{ row }">{{ formatPrice(row.amount?.total) }}</template>
          </el-table-column>
          <el-table-column prop="refundState" label="退款状态" align="center" />
          <el-table-column prop="successTime" label="退款时间" align="center" />
          <el-table-column prop="status" label="对帐状态" align="center">
            <template #default="scope">
              <DictLabel v-model="scope.row.status" code="order_bill_status" />
            </template>
          </el-table-column>
        </el-table>
      </el-card>
      <el-form
        ref="dataFormRefRefund"
        :model="formDataRefund"
        :rules="rulesRefund"
        label-width="150px"
      >
        <el-card shadow="never">
          <el-form-item label="退款原因" prop="reason">
            <Dict v-model="formDataRefund.reason" code="order_refund_reason" />
          </el-form-item>

          <el-form-item label="退款金额" prop="refundMoney">
            <el-input-number v-model="formDataRefund.refundMoney" :precision="2" :step="0.1" />
          </el-form-item>
        </el-card>
      </el-form>

      <template #footer>
        <div class="dialog-footer">
          <el-button type="primary" @click="handleRefundSubmitClick">确 定</el-button>
          <el-button @click="handleCloseRefundDialog">取 消</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { useUserStore } from "@/store";

defineOptions({
  name: "Order",
  inheritAttrs: false,
});

const props = defineProps({
  /**
   * 订单状态
   */
  status: {
    type: Number,
    default: 0,
  },
});

import { defOrderService } from "@/api/admin/order";
import {
  Order,
  OrderRefundResponse,
  OrderShippedResponse,
  PageOrderRequest,
  RefundOrderRequest,
  ShippedOrderRequest,
} from "@/rpc/admin/order";
import router from "@/router";
import { formatPrice } from "@/utils/utils";
import { SelectOptionResponse_Option } from "@/rpc/common/common";
import { defBaseUserService } from "@/api/admin/base_user";
import { OrderPayType, OrderStatus } from "@/rpc/common/enum";
const queryFormRef = ref();

const loading = ref(false);
const total = ref(0);

const queryParams = reactive<PageOrderRequest>({
  /** 订单编号 */
  orderNo: "",
  /** 用户id */
  userId: 0,
  /** 订单状态 */
  status: props.status,
  /** 支付方式，1为在线支付，2为货到付款 */
  payType: undefined,
  /** 支付渠道：支付渠道，1支付宝、2微信--支付方式为在线支付时，传值，为货到付款时，不传值 */
  payChannel: undefined,
  /** 创建时间 */
  createdAt: ["", ""],
  /** 当前页码 */
  pageNum: 0,
  /** 每一页的行数 */
  pageSize: 10,
});

// 订单表格数据
const pageData = ref<Order[]>();
// 用户下拉数据源
const userOptions = ref<SelectOptionResponse_Option[]>();

const dialogShipped = reactive({
  title: "发货详情",
  visible: false,
});
const dataShipped = reactive<OrderShippedResponse>({
  /** 地址信息 */
  address: undefined,
  /** 商品信息 */
  goods: [],
  /** 物流信息 */
  logistics: undefined,
});
const dataFormRefShipped = ref(ElForm);
const formDataShipped = reactive<ShippedOrderRequest>({
  /** 订单id */
  orderId: 0,
  /** 物流公司名 */
  name: "",
  /** 物流单号 */
  no: "",
  /** 联系方式 */
  contact: "",
});
const rulesShipped = computed(() => {
  const rules: Partial<Record<string, any>> = {
    name: [{ required: true, message: "请输入物流公司名称", trigger: "blur" }],
    no: [{ required: true, message: "请输入物流单号", trigger: "blur" }],
    contact: [{ required: true, message: "请输入联系方式", trigger: "blur" }],
  };
  return rules;
});

/**
 * 打开弹窗
 *
 * @param orderId
 * @param title
 */
function handleOpenShippedDialog(orderId: number, title: string) {
  dialogShipped.visible = true;
  dialogShipped.title = title;
  defOrderService
    .GetOrderShipped({
      value: orderId,
    })
    .then((data) => {
      formDataShipped.orderId = orderId;
      Object.assign(dataShipped, data);
    });
}

// 关闭弹窗
function handleCloseShippedDialog() {
  dialogShipped.visible = false;
  if (dataFormRefShipped.value) {
    dataFormRefShipped.value.resetFields();
    dataFormRefShipped.value.clearValidate();
  }
  formDataShipped.orderId = 0;
  dataShipped.logistics = undefined;
  dataShipped.goods = [];
  dataShipped.address = undefined;
}

// 提交
function handleShippedSubmitClick() {
  dataFormRefShipped.value.validate((isValid: boolean) => {
    if (isValid) {
      loading.value = true;
      defOrderService
        .ShippedOrder(formDataShipped)
        .then(() => {
          ElMessage.success("发货成功");
          handleCloseShippedDialog();
          handleQuery();
        })
        .finally(() => (loading.value = false));
    }
  });
}

const dialogRefund = reactive({
  title: "退款详情",
  visible: false,
});
const dataRefund = reactive<OrderRefundResponse>({
  /** 支付信息 */
  payment: undefined,
  /** 退款信息 */
  refund: [],
});

const dataFormRefRefund = ref(ElForm);
const formDataRefund = reactive<RefundOrderRequest>({
  /** 订单id */
  orderId: 0,
  /** 退款原因 */
  reason: undefined,
  /** 退款金额 */
  refundMoney: 0,
});
const rulesRefund = computed(() => {
  const rules: Partial<Record<string, any>> = {
    reason: [{ required: true, message: "请输入退款原因", trigger: "blur" }],
    refundMoney: [{ required: true, message: "请输入退款金额", trigger: "blur" }],
  };
  return rules;
});

/**
 * 打开弹窗
 *
 * @param row
 */
function handleRefund(row: Order) {
  ElMessageBox.prompt("货到付款订单【" + row.orderNo + "】的退款原因", "申请退款", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
  }).then(
    () => {
      defOrderService
        .RefundOrder({
          /** 订单id */
          orderId: row.id,
          refundMoney: 0,
        })
        .then(() => {
          ElMessage.success("退款成功");
          handleQuery();
        });
    },
    () => {
      ElMessage.info("已取消退款");
    }
  );
}

/**
 * 打开弹窗
 *
 * @param orderId
 * @param title
 */
function handleOpenRefundDialog(orderId: number, title: string) {
  dialogRefund.visible = true;
  dialogRefund.title = title;
  defOrderService
    .GetOrderRefund({
      value: orderId,
    })
    .then((data) => {
      formDataRefund.orderId = orderId;
      Object.assign(dataRefund, data);
    });
}

// 关闭弹窗
function handleCloseRefundDialog() {
  dialogRefund.visible = false;
  if (dataFormRefRefund.value) {
    dataFormRefRefund.value.resetFields();
    dataFormRefRefund.value.clearValidate();
  }

  formDataRefund.orderId = 0;
  dataRefund.payment = undefined;
  dataRefund.refund = [];
}

// 提交
function handleRefundSubmitClick() {
  dataFormRefRefund.value.validate((isValid: boolean) => {
    if (isValid) {
      loading.value = true;
      const submitsData = JSON.parse(JSON.stringify(formDataRefund));
      submitsData.refundMoney = submitsData.refundMoney * 100;
      defOrderService
        .RefundOrder(submitsData)
        .then(() => {
          ElMessage.success("退款成功");
          handleCloseRefundDialog();
          handleQuery();
        })
        .finally(() => (loading.value = false));
    }
  });
}

function formatUser(userId: number) {
  const entry = userOptions.value?.find((item: any) => item.value == userId);
  return entry ? entry.label : "-";
}

/** 查询 */
function handleQuery() {
  loading.value = true;
  defOrderService
    .PageOrder(queryParams)
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
  queryParams.createdAt = ["", ""];
  handleQuery();
}
// 打开详情
function handleOpenDetail(row: Order) {
  router.push({
    path: `/order/detail/${row.id}`,
    query: { title: "【" + row.orderNo + "】订单详情" },
  });
}

onMounted(() => {
  // 加载用户下拉数据源
  defBaseUserService
    .OptionBaseUser({
      keyword: "",
    })
    .then((res) => {
      userOptions.value = res.list;
    });
  handleQuery();
});
</script>
<style scoped></style>
