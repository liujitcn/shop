<template>
  <div class="table-box">
    <ProTable ref="proTable" row-key="id" :columns="columns" :request-api="requestOrderTable">
      <template #orderNo="scope">
        <el-link v-if="BUTTONS['order:info:detail']" type="primary" @click.stop="handleOpenDetail(scope.row)">
          {{ scope.row.orderNo }}
        </el-link>
        <span v-else>{{ scope.row.orderNo }}</span>
      </template>

      <template #userId="scope">
        <span>{{ scope.row.nickName || formatUser(scope.row.userId) }}</span>
      </template>

      <template #payMoney="scope">
        <el-popover effect="light" trigger="hover" placement="top" width="auto">
          <template #default>
            <div>总价：{{ formatPrice(scope.row.totalMoney) }}</div>
            <div>实付：{{ formatPrice(scope.row.payMoney) }}</div>
            <div>运费：{{ formatPrice(scope.row.postFee) }}</div>
          </template>
          <template #reference>
            {{ formatPrice(scope.row.payMoney) }}
          </template>
        </el-popover>
      </template>

      <template #operation="scope">
        <el-button
          v-if="canOpenShipped(scope.row)"
          v-hasPerm="['order:info:shipped']"
          type="primary"
          link
          :icon="Van"
          @click="handleOpenShippedDialog(scope.row.id, '发货')"
        >
          发货
        </el-button>
        <el-button
          v-if="canViewShipped(scope.row)"
          v-hasPerm="['order:info:shipped']"
          type="primary"
          link
          :icon="View"
          @click="handleOpenShippedDialog(scope.row.id, '发货详情')"
        >
          发货详情
        </el-button>
        <el-button
          v-if="canRefundCod(scope.row)"
          v-hasPerm="['order:info:refund']"
          type="danger"
          link
          :icon="RefreshLeft"
          @click="handleRefund(scope.row)"
        >
          退款
        </el-button>
        <el-button
          v-if="canOpenRefund(scope.row)"
          v-hasPerm="['order:info:refund']"
          type="danger"
          link
          :icon="RefreshLeft"
          @click="handleOpenRefundDialog(scope.row.id, '退款')"
        >
          退款
        </el-button>
        <el-button
          v-if="canViewRefund(scope.row)"
          v-hasPerm="['order:info:refund']"
          type="danger"
          link
          :icon="View"
          @click="handleOpenRefundDialog(scope.row.id, '退款详情')"
        >
          退款详情
        </el-button>
      </template>
    </ProTable>

    <el-dialog v-model="dialogShipped.visible" :title="dialogShipped.title" width="1200px" @close="handleCloseShippedDialog">
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

      <el-form v-else ref="dataFormRefShipped" :model="formDataShipped" :rules="rulesShipped" label-width="150px">
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

    <el-dialog v-model="dialogRefund.visible" :title="dialogRefund.title" width="1200px" @close="handleCloseRefundDialog">
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

      <el-form ref="dataFormRefRefund" :model="formDataRefund" :rules="rulesRefund" label-width="150px">
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
import { computed, reactive, ref } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { RefreshLeft, Van, View } from "@element-plus/icons-vue";
import type { ColumnProps, EnumProps, ProTableInstance } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
import { useAuthButtons } from "@/hooks/useAuthButtons";
import { defOrderService } from "@/api/admin/order";
import { defBaseUserService } from "@/api/admin/base_user";
import type {
  Order,
  OrderRefundResponse,
  OrderShippedResponse,
  PageOrderRequest,
  RefundOrderRequest,
  ShippedOrderRequest
} from "@/rpc/admin/order";
import type { SelectOptionResponse_Option } from "@/rpc/common/common";
import router from "@/routers";
import { OrderPayType, OrderStatus } from "@/rpc/common/enum";
import { buildPageRequest } from "@/utils/proTable";
import { formatPrice } from "@/utils/utils";

defineOptions({
  name: "Order",
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
const proTable = ref<ProTableInstance>();
const dataFormRefShipped = ref();
const dataFormRefRefund = ref();
const userOptions = ref<SelectOptionResponse_Option[]>([]);

const dialogShipped = reactive({
  title: "发货详情",
  visible: false
});

const dataShipped = reactive<OrderShippedResponse>({
  /** 地址信息 */
  address: undefined,
  /** 商品信息 */
  goods: [],
  /** 物流信息 */
  logistics: undefined
});

const formDataShipped = reactive<ShippedOrderRequest>({
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

const dialogRefund = reactive({
  title: "退款详情",
  visible: false
});

const dataRefund = reactive<OrderRefundResponse>({
  /** 支付信息 */
  payment: undefined,
  /** 退款信息 */
  refund: []
});

const formDataRefund = reactive<RefundOrderRequest>({
  /** 订单id */
  orderId: 0,
  /** 退款原因 */
  reason: undefined,
  /** 退款金额 */
  refundMoney: 0
});

const rulesRefund = computed(() => ({
  reason: [{ required: true, message: "请输入退款原因", trigger: "blur" }],
  refundMoney: [{ required: true, message: "请输入退款金额", trigger: "blur" }]
}));

/** 订单表格列配置。 */
const columns: ColumnProps[] = [
  {
    prop: "userId",
    label: "用户",
    width: 180,
    enum: requestUserEnum,
    search: {
      el: "select",
      props: {
        filterable: true,
        clearable: true
      }
    }
  },
  { prop: "orderNo", label: "订单编号", minWidth: 190, search: { el: "input" } },
  { prop: "payMoney", label: "金额（元）", align: "right", width: 120 },
  { prop: "payType", label: "支付方式", width: 110, dictCode: "order_pay_type", search: { el: "select" } },
  { prop: "payChannel", label: "支付渠道", width: 110, dictCode: "order_pay_channel", search: { el: "select" } },
  {
    prop: "status",
    label: "状态",
    width: 110,
    dictCode: "order_status",
    search: props.status ? undefined : { el: "select" }
  },
  {
    prop: "createdAt",
    label: "创建时间",
    width: 180,
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
  { prop: "goodsNum", label: "商品数", width: 90, align: "right" },
  { prop: "operation", label: "操作", width: 280, fixed: "right" }
];

/**
 * 加载用户下拉项，供搜索和回显共用。
 */
async function loadUserOptions() {
  if (userOptions.value.length) return userOptions.value;
  const response = await defBaseUserService.OptionBaseUser({ keyword: "" });
  userOptions.value = response.list ?? [];
  return userOptions.value;
}

/**
 * 将用户下拉项转换为 ProTable 搜索枚举。
 */
async function requestUserEnum() {
  const options = await loadUserOptions();
  const data: EnumProps[] = options.map(item => ({
    label: item.label,
    value: item.value
  }));
  return { data };
}

/**
 * 请求订单分页列表，并补齐固定筛选参数。
 */
async function requestOrderTable(params: PageOrderRequest) {
  const data = await defOrderService.PageOrder(
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
function canOpenShipped(row: Order) {
  return row.status === OrderStatus.PAID;
}

/**
 * 判断当前订单是否可查看发货详情。
 */
function canViewShipped(row: Order) {
  return row.status === OrderStatus.SHIPPED || row.status === OrderStatus.RECEIVED;
}

/**
 * 判断货到付款订单是否可直接退款。
 */
function canRefundCod(row: Order) {
  return row.payType === OrderPayType.CASH_ON_DELIVERY && canViewShipped(row);
}

/**
 * 判断在线支付订单是否可发起退款。
 */
function canOpenRefund(row: Order) {
  return (
    row.payType === OrderPayType.ONLINE_PAY &&
    (row.status === OrderStatus.SHIPPED || row.status === OrderStatus.RECEIVED || row.status === OrderStatus.REFUNDING)
  );
}

/**
 * 判断当前订单是否可查看退款详情。
 */
function canViewRefund(row: Order) {
  return row.status === OrderStatus.REFUNDING;
}

/**
 * 打开发货弹窗并加载物流数据。
 */
function handleOpenShippedDialog(orderId: number, title: string) {
  dialogShipped.visible = true;
  dialogShipped.title = title;
  defOrderService.GetOrderShipped({ value: orderId }).then(data => {
    formDataShipped.orderId = orderId;
    Object.assign(dataShipped, data);
  });
}

/**
 * 关闭发货弹窗并清理表单。
 */
function handleCloseShippedDialog() {
  dialogShipped.visible = false;
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
  dataFormRefShipped.value?.validate((isValid: boolean) => {
    if (!isValid) return;

    defOrderService.ShippedOrder(formDataShipped).then(() => {
      ElMessage.success("发货成功");
      handleCloseShippedDialog();
      proTable.value?.getTableList();
    });
  });
}

/**
 * 对货到付款订单发起退款。
 */
function handleRefund(row: Order) {
  ElMessageBox.prompt(`货到付款订单【${row.orderNo}】的退款原因`, "申请退款", {
    confirmButtonText: "确定",
    cancelButtonText: "取消"
  }).then(
    () => {
      defOrderService.RefundOrder({ orderId: row.id, refundMoney: 0 }).then(() => {
        ElMessage.success("退款成功");
        proTable.value?.getTableList();
      });
    },
    () => {
      ElMessage.info("已取消退款");
    }
  );
}

/**
 * 打开退款弹窗并加载退款详情。
 */
function handleOpenRefundDialog(orderId: number, title: string) {
  dialogRefund.visible = true;
  dialogRefund.title = title;
  defOrderService.GetOrderRefund({ value: orderId }).then(data => {
    formDataRefund.orderId = orderId;
    Object.assign(dataRefund, data);
  });
}

/**
 * 关闭退款弹窗并清理表单。
 */
function handleCloseRefundDialog() {
  dialogRefund.visible = false;
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
  dataFormRefRefund.value?.validate((isValid: boolean) => {
    if (!isValid) return;

    const submitData = JSON.parse(JSON.stringify(formDataRefund)) as RefundOrderRequest;
    submitData.refundMoney = submitData.refundMoney * 100;
    defOrderService.RefundOrder(submitData).then(() => {
      ElMessage.success("退款成功");
      handleCloseRefundDialog();
      proTable.value?.getTableList();
    });
  });
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
function handleOpenDetail(row: Order) {
  router.push({
    path: `/order/detail/${row.id}`,
    query: { title: `【${row.orderNo}】订单详情` }
  });
}

loadUserOptions();
</script>
