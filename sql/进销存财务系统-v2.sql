-- 进销存财务系统表结构 v2
-- 设计原则：业务事实使用结构化字段；仅展示、审计、原始请求和规则快照使用 JSON。
-- MySQL 8.0+；金额单位为分，数量按 SKU 基本单位记账；deleted_at 对应 Go int64。
SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

CREATE TABLE IF NOT EXISTS scm_supplier (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  tenant_id BIGINT NOT NULL COMMENT '租户ID',
  code VARCHAR(32) NOT NULL COMMENT '供应商编码',
  name VARCHAR(100) NOT NULL COMMENT '供应商名称',
  contact VARCHAR(50) COMMENT '联系人',
  phone VARCHAR(32) COMMENT '联系电话',
  address VARCHAR(255) COMMENT '联系地址',
  status TINYINT NOT NULL DEFAULT 1 COMMENT '状态：1启用，2停用',
  deleted_at BIGINT NOT NULL DEFAULT 0 COMMENT '删除时间戳，0表示未删除',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  UNIQUE KEY uk_supplier_tenant_code (tenant_id, code),
  KEY idx_supplier_tenant_status (tenant_id, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='供应商主数据';

CREATE TABLE IF NOT EXISTS scm_warehouse (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  tenant_id BIGINT NOT NULL COMMENT '租户ID',
  code VARCHAR(32) NOT NULL COMMENT '仓库编码',
  name VARCHAR(100) NOT NULL COMMENT '仓库名称',
  warehouse_type TINYINT NOT NULL DEFAULT 1 COMMENT '仓库类型',
  address VARCHAR(255) COMMENT '仓库地址',
  manager_id BIGINT COMMENT '负责人用户ID',
  allow_negative TINYINT NOT NULL DEFAULT 0 COMMENT '是否允许负库存',
  status TINYINT NOT NULL DEFAULT 1 COMMENT '状态',
  deleted_at BIGINT NOT NULL DEFAULT 0 COMMENT '删除时间戳，0表示未删除',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  UNIQUE KEY uk_warehouse_tenant_code (tenant_id, code),
  KEY idx_warehouse_tenant_status (tenant_id, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='仓库主数据';

CREATE TABLE IF NOT EXISTS scm_warehouse_store (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  tenant_id BIGINT NOT NULL COMMENT '租户ID',
  warehouse_id BIGINT NOT NULL COMMENT '仓库ID',
  tenant_store_id BIGINT NOT NULL COMMENT '服务门店ID',
  priority INT NOT NULL DEFAULT 0 COMMENT '履约优先级',
  status TINYINT NOT NULL DEFAULT 1 COMMENT '状态',
  deleted_at BIGINT NOT NULL DEFAULT 0 COMMENT '删除时间戳，0表示未删除',
  UNIQUE KEY uk_warehouse_store (warehouse_id, tenant_store_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='仓库服务门店关系';

CREATE TABLE IF NOT EXISTS scm_location (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  tenant_id BIGINT NOT NULL COMMENT '租户ID',
  warehouse_id BIGINT NOT NULL COMMENT '仓库ID',
  code VARCHAR(32) NOT NULL COMMENT '库位编码',
  name VARCHAR(100) NOT NULL COMMENT '库位名称',
  location_type TINYINT NOT NULL DEFAULT 1 COMMENT '库位类型',
  sort INT NOT NULL DEFAULT 0 COMMENT '排序号',
  status TINYINT NOT NULL DEFAULT 1 COMMENT '状态',
  deleted_at BIGINT NOT NULL DEFAULT 0 COMMENT '删除时间戳，0表示未删除',
  UNIQUE KEY uk_location_warehouse_code (warehouse_id, code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='仓库库位';

CREATE TABLE IF NOT EXISTS scm_purchase_order (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  tenant_id BIGINT NOT NULL COMMENT '租户ID',
  order_no VARCHAR(32) NOT NULL COMMENT '采购订单号',
  supplier_id BIGINT NOT NULL COMMENT '供应商ID',
  warehouse_id BIGINT NOT NULL COMMENT '收货仓库ID',
  total_money BIGINT NOT NULL DEFAULT 0 COMMENT '订单金额，单位分',
  received_money BIGINT NOT NULL DEFAULT 0 COMMENT '已收货金额，单位分',
  expected_at DATETIME COMMENT '预计到货时间',
  status TINYINT NOT NULL DEFAULT 1 COMMENT '单据状态',
  remark VARCHAR(255) COMMENT '备注',
  created_by BIGINT COMMENT '创建人ID',
  approved_by BIGINT COMMENT '审核人ID',
  approved_at DATETIME COMMENT '审核时间',
  deleted_at BIGINT NOT NULL DEFAULT 0 COMMENT '删除时间戳，0表示未删除',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  UNIQUE KEY uk_purchase_order_tenant_no (tenant_id, order_no),
  KEY idx_purchase_order_supplier (tenant_id, supplier_id, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='采购订单';

CREATE TABLE IF NOT EXISTS scm_purchase_order_item (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  purchase_order_id BIGINT NOT NULL COMMENT '采购订单ID',
  line_no INT NOT NULL COMMENT '明细行号',
  sku_id BIGINT NOT NULL COMMENT 'SKU ID',
  sku_code VARCHAR(64) NOT NULL COMMENT 'SKU编码快照',
  sku_snapshot JSON COMMENT '商品名称、规格等展示快照',
  unit_id BIGINT COMMENT '业务单位ID',
  unit_ratio_num BIGINT NOT NULL DEFAULT 1 COMMENT '换算率分子',
  unit_ratio_den BIGINT NOT NULL DEFAULT 1 COMMENT '换算率分母',
  order_num BIGINT NOT NULL COMMENT '采购数量',
  received_num BIGINT NOT NULL DEFAULT 0 COMMENT '已收货数量',
  price BIGINT NOT NULL DEFAULT 0 COMMENT '采购单价，单位分',
  money BIGINT NOT NULL DEFAULT 0 COMMENT '明细金额，单位分',
  UNIQUE KEY uk_purchase_order_item_line (purchase_order_id, line_no),
  KEY idx_purchase_order_item_sku (sku_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='采购订单明细';

CREATE TABLE IF NOT EXISTS scm_purchase_receipt (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  tenant_id BIGINT NOT NULL COMMENT '租户ID',
  receipt_no VARCHAR(32) NOT NULL COMMENT '收货单号',
  purchase_order_id BIGINT COMMENT '采购订单ID',
  supplier_id BIGINT NOT NULL COMMENT '供应商ID',
  warehouse_id BIGINT NOT NULL COMMENT '收货仓库ID',
  total_num BIGINT NOT NULL DEFAULT 0 COMMENT '收货总数量',
  total_money BIGINT NOT NULL DEFAULT 0 COMMENT '收货金额，单位分',
  source_snapshot JSON COMMENT '外部送货单等展示快照',
  status TINYINT NOT NULL DEFAULT 1 COMMENT '单据状态',
  received_by BIGINT COMMENT '收货人ID',
  received_at DATETIME COMMENT '收货时间',
  deleted_at BIGINT NOT NULL DEFAULT 0 COMMENT '删除时间戳，0表示未删除',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  UNIQUE KEY uk_purchase_receipt_tenant_no (tenant_id, receipt_no),
  KEY idx_purchase_receipt_order (purchase_order_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='采购收货单';

CREATE TABLE IF NOT EXISTS scm_purchase_receipt_item (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  receipt_id BIGINT NOT NULL COMMENT '收货单ID',
  purchase_order_item_id BIGINT COMMENT '采购订单明细ID',
  line_no INT NOT NULL COMMENT '明细行号',
  sku_id BIGINT NOT NULL COMMENT 'SKU ID',
  sku_code VARCHAR(64) NOT NULL COMMENT 'SKU编码快照',
  sku_snapshot JSON COMMENT '商品展示快照',
  num BIGINT NOT NULL COMMENT '收货数量',
  price BIGINT NOT NULL DEFAULT 0 COMMENT '采购单价，单位分',
  money BIGINT NOT NULL DEFAULT 0 COMMENT '明细金额，单位分',
  UNIQUE KEY uk_purchase_receipt_item_line (receipt_id, line_no),
  KEY idx_purchase_receipt_item_sku (sku_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='采购收货明细';

CREATE TABLE IF NOT EXISTS scm_sales_fulfillment (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  tenant_id BIGINT NOT NULL COMMENT '租户ID',
  tenant_store_id BIGINT NOT NULL COMMENT '履约门店ID',
  order_info_id BIGINT NOT NULL COMMENT '商城门店订单ID',
  fulfillment_no VARCHAR(32) NOT NULL COMMENT '履约单号',
  warehouse_id BIGINT NOT NULL COMMENT '履约仓库ID',
  logistics_company VARCHAR(64) COMMENT '物流公司',
  logistics_no VARCHAR(64) COMMENT '物流单号',
  status TINYINT NOT NULL DEFAULT 1 COMMENT '履约状态',
  shipped_at DATETIME COMMENT '发货时间',
  received_at DATETIME COMMENT '签收时间',
  deleted_at BIGINT NOT NULL DEFAULT 0 COMMENT '删除时间戳，0表示未删除',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  UNIQUE KEY uk_fulfillment_tenant_no (tenant_id, fulfillment_no),
  KEY idx_fulfillment_order (order_info_id),
  KEY idx_fulfillment_warehouse (warehouse_id, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='销售履约单';

CREATE TABLE IF NOT EXISTS scm_sales_fulfillment_item (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  fulfillment_id BIGINT NOT NULL COMMENT '履约单ID',
  order_goods_id BIGINT NOT NULL COMMENT '商城订单商品ID',
  line_no INT NOT NULL COMMENT '明细行号',
  sku_id BIGINT NOT NULL COMMENT 'SKU ID',
  sku_code VARCHAR(64) NOT NULL COMMENT 'SKU编码快照',
  sku_snapshot JSON COMMENT '商品展示快照',
  num BIGINT NOT NULL COMMENT '本次发货数量',
  stock_bill_goods_id BIGINT COMMENT '生成的库存明细ID',
  UNIQUE KEY uk_fulfillment_item_line (fulfillment_id, line_no),
  KEY idx_fulfillment_item_order_goods (order_goods_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='销售履约明细';

CREATE TABLE IF NOT EXISTS stock_account (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  tenant_id BIGINT NOT NULL COMMENT '租户ID',
  warehouse_id BIGINT NOT NULL COMMENT '仓库ID',
  sku_id BIGINT NOT NULL COMMENT 'SKU ID',
  on_hand_num BIGINT NOT NULL DEFAULT 0 COMMENT '现存数量',
  qualified_num BIGINT NOT NULL DEFAULT 0 COMMENT '合格数量',
  reserved_num BIGINT NOT NULL DEFAULT 0 COMMENT '预占数量',
  frozen_num BIGINT NOT NULL DEFAULT 0 COMMENT '冻结数量',
  available_num BIGINT NOT NULL DEFAULT 0 COMMENT '可售数量',
  cost_amount BIGINT NOT NULL DEFAULT 0 COMMENT '库存总成本，单位分',
  cost_price BIGINT NOT NULL DEFAULT 0 COMMENT '移动平均成本，单位分',
  cost_status TINYINT NOT NULL DEFAULT 1 COMMENT '成本状态',
  version BIGINT NOT NULL DEFAULT 0 COMMENT '并发版本号',
  UNIQUE KEY uk_stock_account (tenant_id, warehouse_id, sku_id),
  KEY idx_stock_account_sku (tenant_id, sku_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='仓库SKU库存账户';

CREATE TABLE IF NOT EXISTS stock_quant (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  tenant_id BIGINT NOT NULL COMMENT '租户ID',
  warehouse_id BIGINT NOT NULL COMMENT '仓库ID',
  location_id BIGINT NOT NULL DEFAULT 0 COMMENT '库位ID，0表示未启用库位',
  sku_id BIGINT NOT NULL COMMENT 'SKU ID',
  batch_id BIGINT NOT NULL DEFAULT 0 COMMENT '批次ID，0表示无批次',
  stock_status TINYINT NOT NULL DEFAULT 2 COMMENT '库存状态',
  on_hand_num BIGINT NOT NULL DEFAULT 0 COMMENT '现存数量',
  reserved_num BIGINT NOT NULL DEFAULT 0 COMMENT '已分配预占数量',
  version BIGINT NOT NULL DEFAULT 0 COMMENT '并发版本号',
  UNIQUE KEY uk_stock_quant (warehouse_id, location_id, sku_id, batch_id, stock_status),
  KEY idx_stock_quant_sku (tenant_id, sku_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='库存实物明细';

CREATE TABLE IF NOT EXISTS stock_batch (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  tenant_id BIGINT NOT NULL COMMENT '租户ID',
  sku_id BIGINT NOT NULL COMMENT 'SKU ID',
  batch_no VARCHAR(64) NOT NULL COMMENT '批次号',
  supplier_batch_no VARCHAR(64) COMMENT '供应商批号',
  production_date DATE COMMENT '生产日期',
  expiry_date DATE COMMENT '失效日期',
  status TINYINT NOT NULL DEFAULT 1 COMMENT '状态',
  UNIQUE KEY uk_stock_batch (tenant_id, sku_id, batch_no)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='库存批次档案';

CREATE TABLE IF NOT EXISTS stock_serial (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  tenant_id BIGINT NOT NULL COMMENT '租户ID',
  sku_id BIGINT NOT NULL COMMENT 'SKU ID',
  serial_no VARCHAR(128) NOT NULL COMMENT '序列号',
  warehouse_id BIGINT COMMENT '当前仓库ID',
  location_id BIGINT COMMENT '当前库位ID',
  batch_id BIGINT COMMENT '批次ID',
  stock_status TINYINT NOT NULL DEFAULT 2 COMMENT '库存状态',
  lifecycle_status TINYINT NOT NULL DEFAULT 1 COMMENT '生命周期状态',
  version BIGINT NOT NULL DEFAULT 0 COMMENT '并发版本号',
  UNIQUE KEY uk_stock_serial (tenant_id, sku_id, serial_no),
  KEY idx_stock_serial_location (warehouse_id, location_id, sku_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='库存序列号当前位置';

CREATE TABLE IF NOT EXISTS stock_reservation (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  tenant_id BIGINT NOT NULL COMMENT '租户ID',
  tenant_store_id BIGINT NOT NULL COMMENT '履约门店ID',
  warehouse_id BIGINT NOT NULL COMMENT '预占仓库ID',
  sku_id BIGINT NOT NULL COMMENT 'SKU ID',
  source_type TINYINT NOT NULL COMMENT '来源类型',
  source_id BIGINT NOT NULL COMMENT '来源订单商品ID',
  reserved_num BIGINT NOT NULL COMMENT '原始预占数量',
  committed_num BIGINT NOT NULL DEFAULT 0 COMMENT '已出库数量',
  released_num BIGINT NOT NULL DEFAULT 0 COMMENT '已释放数量',
  remaining_num BIGINT NOT NULL COMMENT '剩余预占数量',
  status TINYINT NOT NULL DEFAULT 1 COMMENT '状态',
  reserved_at DATETIME NOT NULL COMMENT '预占时间',
  UNIQUE KEY uk_stock_reservation (source_type, source_id, warehouse_id),
  KEY idx_stock_reservation_account (tenant_id, warehouse_id, sku_id, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='库存预占';

CREATE TABLE IF NOT EXISTS stock_bill (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  tenant_id BIGINT NOT NULL COMMENT '租户ID',
  bill_no VARCHAR(32) NOT NULL COMMENT '库存单号',
  warehouse_id BIGINT NOT NULL COMMENT '仓库ID',
  tenant_store_id BIGINT COMMENT '业务门店ID',
  direction TINYINT NOT NULL COMMENT '方向：1入库，2出库',
  biz_type TINYINT NOT NULL COMMENT '业务类型',
  source_type TINYINT COMMENT '来源单据类型',
  source_id BIGINT COMMENT '来源单据ID',
  source_no VARCHAR(32) COMMENT '来源单号快照',
  total_num BIGINT NOT NULL DEFAULT 0 COMMENT '总数量',
  total_cost_amount BIGINT NOT NULL DEFAULT 0 COMMENT '库存成本金额，单位分',
  status TINYINT NOT NULL DEFAULT 1 COMMENT '状态',
  reversal_of_id BIGINT COMMENT '红冲原库存单ID',
  confirmed_by BIGINT COMMENT '确认人ID',
  confirmed_at DATETIME COMMENT '确认时间',
  deleted_at BIGINT NOT NULL DEFAULT 0 COMMENT '删除时间戳，0表示未删除',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  UNIQUE KEY uk_stock_bill (tenant_id, bill_no),
  UNIQUE KEY uk_stock_bill_source (tenant_id, source_type, source_id),
  KEY idx_stock_bill_warehouse (tenant_id, warehouse_id, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='库存出入库单';

CREATE TABLE IF NOT EXISTS stock_bill_item (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  bill_id BIGINT NOT NULL COMMENT '库存单ID',
  line_no INT NOT NULL COMMENT '明细行号',
  source_line_id BIGINT COMMENT '来源明细ID',
  sku_id BIGINT NOT NULL COMMENT 'SKU ID',
  sku_code VARCHAR(64) NOT NULL COMMENT 'SKU编码快照',
  sku_snapshot JSON COMMENT '商品展示快照',
  base_num BIGINT NOT NULL COMMENT '基本单位数量',
  business_price BIGINT NOT NULL DEFAULT 0 COMMENT '业务参考单价，单位分',
  cost_amount BIGINT NOT NULL DEFAULT 0 COMMENT '本次库存成本，单位分',
  UNIQUE KEY uk_stock_bill_item_line (bill_id, line_no),
  KEY idx_stock_bill_item_sku (sku_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='库存出入库明细';

CREATE TABLE IF NOT EXISTS stock_bill_item_trace (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  bill_item_id BIGINT NOT NULL COMMENT '库存明细ID',
  location_id BIGINT NOT NULL DEFAULT 0 COMMENT '库位ID',
  batch_id BIGINT NOT NULL DEFAULT 0 COMMENT '批次ID，0表示无批次',
  serial_id BIGINT NOT NULL DEFAULT 0 COMMENT '序列号ID，0表示无序列号',
  stock_status TINYINT NOT NULL COMMENT '库存状态',
  base_num BIGINT NOT NULL COMMENT '追溯数量',
  UNIQUE KEY uk_stock_bill_trace (bill_item_id, location_id, batch_id, serial_id, stock_status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='库存明细批次序列号分配';

CREATE TABLE IF NOT EXISTS stock_ledger (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  tenant_id BIGINT NOT NULL COMMENT '租户ID',
  warehouse_id BIGINT NOT NULL COMMENT '仓库ID',
  sku_id BIGINT NOT NULL COMMENT 'SKU ID',
  stock_bill_id BIGINT COMMENT '库存单ID',
  source_type TINYINT NOT NULL COMMENT '来源类型',
  source_id BIGINT NOT NULL COMMENT '来源ID',
  change_num BIGINT NOT NULL COMMENT '数量变化，入库为正',
  before_on_hand_num BIGINT NOT NULL COMMENT '变动前现存量',
  after_on_hand_num BIGINT NOT NULL COMMENT '变动后现存量',
  change_cost_amount BIGINT NOT NULL COMMENT '成本变化金额',
  before_cost_amount BIGINT NOT NULL COMMENT '变动前成本',
  after_cost_amount BIGINT NOT NULL COMMENT '变动后成本',
  occurred_at DATETIME NOT NULL COMMENT '业务发生时间',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '写入时间',
  UNIQUE KEY uk_stock_ledger_source (tenant_id, warehouse_id, sku_id, source_type, source_id),
  KEY idx_stock_ledger_account (tenant_id, warehouse_id, sku_id, occurred_at, id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='不可变库存台账';

CREATE TABLE IF NOT EXISTS stock_transfer (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  tenant_id BIGINT NOT NULL COMMENT '租户ID',
  transfer_no VARCHAR(32) NOT NULL COMMENT '调拨单号',
  source_warehouse_id BIGINT NOT NULL COMMENT '源仓库ID',
  target_warehouse_id BIGINT NOT NULL COMMENT '目标仓库ID',
  expected_arrival_at DATETIME COMMENT '预计到达时间',
  status TINYINT NOT NULL DEFAULT 1 COMMENT '调拨状态',
  deleted_at BIGINT NOT NULL DEFAULT 0 COMMENT '删除时间戳，0表示未删除',
  created_by BIGINT COMMENT '创建人ID',
  approved_by BIGINT COMMENT '审核人ID',
  dispatched_at DATETIME COMMENT '调出时间',
  completed_at DATETIME COMMENT '完成时间',
  UNIQUE KEY uk_stock_transfer (tenant_id, transfer_no),
  KEY idx_stock_transfer_status (tenant_id, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='库存调拨单';

CREATE TABLE IF NOT EXISTS stock_transfer_item (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  transfer_id BIGINT NOT NULL COMMENT '调拨单ID',
  line_no INT NOT NULL COMMENT '明细行号',
  sku_id BIGINT NOT NULL COMMENT 'SKU ID',
  sku_code VARCHAR(64) NOT NULL COMMENT 'SKU编码快照',
  sku_snapshot JSON COMMENT '商品展示快照',
  apply_num BIGINT NOT NULL COMMENT '申请数量',
  dispatched_num BIGINT NOT NULL DEFAULT 0 COMMENT '已调出数量',
  received_num BIGINT NOT NULL DEFAULT 0 COMMENT '已收货数量',
  difference_num BIGINT NOT NULL DEFAULT 0 COMMENT '差异数量',
  UNIQUE KEY uk_stock_transfer_item_line (transfer_id, line_no),
  KEY idx_stock_transfer_item_sku (sku_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='库存调拨明细';

CREATE TABLE IF NOT EXISTS stock_transfer_transit (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  transfer_item_id BIGINT NOT NULL COMMENT '调拨明细ID',
  batch_id BIGINT NOT NULL DEFAULT 0 COMMENT '批次ID，0表示无批次',
  serial_id BIGINT NOT NULL DEFAULT 0 COMMENT '序列号ID，0表示无序列号',
  stock_status TINYINT NOT NULL COMMENT '库存状态',
  dispatched_num BIGINT NOT NULL COMMENT '调出数量',
  received_num BIGINT NOT NULL DEFAULT 0 COMMENT '接收数量',
  returned_num BIGINT NOT NULL DEFAULT 0 COMMENT '退回数量',
  difference_num BIGINT NOT NULL DEFAULT 0 COMMENT '差异数量',
  remaining_num BIGINT NOT NULL COMMENT '在途数量',
  cost_amount BIGINT NOT NULL DEFAULT 0 COMMENT '在途成本金额',
  UNIQUE KEY uk_stock_transfer_transit (transfer_item_id, batch_id, serial_id, stock_status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='调拨在途库存';

CREATE TABLE IF NOT EXISTS fin_payable (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  tenant_id BIGINT NOT NULL COMMENT '租户ID',
  payable_no VARCHAR(32) NOT NULL COMMENT '应付单号',
  supplier_id BIGINT NOT NULL COMMENT '供应商ID',
  source_type TINYINT COMMENT '来源类型',
  source_id BIGINT COMMENT '来源单据ID',
  money BIGINT NOT NULL COMMENT '应付金额，单位分',
  allocated_money BIGINT NOT NULL DEFAULT 0 COMMENT '已核销金额，单位分',
  open_money BIGINT NOT NULL COMMENT '未核销金额，单位分',
  due_date DATE COMMENT '到期日',
  status TINYINT NOT NULL DEFAULT 1 COMMENT '状态',
  deleted_at BIGINT NOT NULL DEFAULT 0 COMMENT '删除时间戳，0表示未删除',
  UNIQUE KEY uk_fin_payable (tenant_id, payable_no),
  UNIQUE KEY uk_fin_payable_source (tenant_id, source_type, source_id),
  KEY idx_fin_payable_supplier (tenant_id, supplier_id, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='供应商应付单';

CREATE TABLE IF NOT EXISTS fin_payment (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  tenant_id BIGINT NOT NULL COMMENT '租户ID',
  payment_no VARCHAR(32) NOT NULL COMMENT '付款单号',
  supplier_id BIGINT COMMENT '供应商ID',
  money BIGINT NOT NULL COMMENT '付款金额，单位分',
  fund_account_id BIGINT NOT NULL COMMENT '资金账户ID',
  money_nature TINYINT NOT NULL DEFAULT 1 COMMENT '款项性质：付款或预付款',
  pay_at DATETIME NOT NULL COMMENT '付款时间',
  source_snapshot JSON COMMENT '付款凭证展示快照',
  status TINYINT NOT NULL DEFAULT 1 COMMENT '状态',
  reversal_of_id BIGINT COMMENT '红冲原付款单ID',
  deleted_at BIGINT NOT NULL DEFAULT 0 COMMENT '删除时间戳，0表示未删除',
  UNIQUE KEY uk_fin_payment (tenant_id, payment_no),
  KEY idx_fin_payment_supplier (tenant_id, supplier_id, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='供应商付款单';

CREATE TABLE IF NOT EXISTS fin_payment_allocation (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  payment_id BIGINT NOT NULL COMMENT '付款单ID',
  payable_id BIGINT NOT NULL COMMENT '应付单ID',
  allocated_money BIGINT NOT NULL COMMENT '本次核销金额，单位分',
  UNIQUE KEY uk_fin_payment_allocation (payment_id, payable_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='付款核销应付关系';

CREATE TABLE IF NOT EXISTS fin_fund_account (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  tenant_id BIGINT NOT NULL COMMENT '租户ID',
  accounting_entity_id BIGINT NOT NULL COMMENT '账簿主体ID',
  account_no VARCHAR(32) NOT NULL COMMENT '资金账户编码',
  name VARCHAR(100) NOT NULL COMMENT '资金账户名称',
  account_type TINYINT NOT NULL COMMENT '账户类型',
  currency_code CHAR(3) NOT NULL DEFAULT 'CNY' COMMENT '币种',
  opening_balance BIGINT NOT NULL DEFAULT 0 COMMENT '期初余额，单位分',
  current_balance BIGINT NOT NULL DEFAULT 0 COMMENT '当前余额缓存，单位分',
  allow_negative TINYINT NOT NULL DEFAULT 0 COMMENT '是否允许透支',
  status TINYINT NOT NULL DEFAULT 1 COMMENT '状态',
  deleted_at BIGINT NOT NULL DEFAULT 0 COMMENT '删除时间戳，0表示未删除',
  UNIQUE KEY uk_fin_fund_account (tenant_id, account_no),
  KEY idx_fin_fund_account_entity (accounting_entity_id, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='资金账户';

CREATE TABLE IF NOT EXISTS fin_fund_flow (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  tenant_id BIGINT NOT NULL COMMENT '租户ID',
  fund_account_id BIGINT NOT NULL COMMENT '资金账户ID',
  direction TINYINT NOT NULL COMMENT '方向：1收入，2支出',
  biz_type TINYINT NOT NULL COMMENT '资金业务类型',
  source_type TINYINT NOT NULL COMMENT '来源类型',
  source_id BIGINT NOT NULL COMMENT '来源ID',
  change_money BIGINT NOT NULL COMMENT '变动金额，单位分',
  before_balance BIGINT NOT NULL COMMENT '变动前余额，单位分',
  after_balance BIGINT NOT NULL COMMENT '变动后余额，单位分',
  reversal_of_id BIGINT COMMENT '红冲原流水ID',
  occurred_at DATETIME NOT NULL COMMENT '业务发生时间',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '写入时间',
  UNIQUE KEY uk_fin_fund_flow (fund_account_id, biz_type, source_type, source_id),
  KEY idx_fin_fund_flow_account (fund_account_id, occurred_at, id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='不可变资金流水';

CREATE TABLE IF NOT EXISTS fin_voucher (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  accounting_entity_id BIGINT NOT NULL COMMENT '账簿主体ID',
  source_tenant_id BIGINT NOT NULL COMMENT '来源业务租户ID',
  voucher_no VARCHAR(32) NOT NULL COMMENT '业务凭证号',
  period CHAR(7) NOT NULL COMMENT '业务期间，格式YYYY-MM',
  event_type TINYINT NOT NULL COMMENT '业务事件类型',
  source_type TINYINT NOT NULL COMMENT '来源类型',
  source_id BIGINT NOT NULL COMMENT '来源ID',
  debit_money BIGINT NOT NULL DEFAULT 0 COMMENT '借方合计，单位分',
  credit_money BIGINT NOT NULL DEFAULT 0 COMMENT '贷方合计，单位分',
  summary VARCHAR(255) COMMENT '凭证摘要',
  entry_snapshot JSON COMMENT '导出时的展示快照',
  occurred_at DATETIME NOT NULL COMMENT '业务发生时间',
  status TINYINT NOT NULL DEFAULT 1 COMMENT '凭证状态',
  reversal_of_id BIGINT COMMENT '红冲原凭证ID',
  UNIQUE KEY uk_fin_voucher_no (accounting_entity_id, voucher_no),
  UNIQUE KEY uk_fin_voucher_source (accounting_entity_id, event_type, source_type, source_id),
  KEY idx_fin_voucher_period (accounting_entity_id, period, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='业务凭证';

CREATE TABLE IF NOT EXISTS fin_voucher_entry (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  voucher_id BIGINT NOT NULL COMMENT '凭证ID',
  line_no INT NOT NULL COMMENT '分录行号',
  subject_key VARCHAR(50) NOT NULL COMMENT '系统科目标识',
  subject_code VARCHAR(20) COMMENT '外部科目编码快照',
  subject_name VARCHAR(100) COMMENT '外部科目名称快照',
  direction TINYINT NOT NULL COMMENT '方向：1借，2贷',
  money BIGINT NOT NULL COMMENT '分录金额，单位分',
  tenant_store_id BIGINT COMMENT '门店维度',
  supplier_id BIGINT COMMENT '供应商维度',
  merchant_tenant_id BIGINT COMMENT '商户维度',
  UNIQUE KEY uk_fin_voucher_entry (voucher_id, line_no)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='业务凭证分录';

CREATE TABLE IF NOT EXISTS fin_accounting_period (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  accounting_entity_id BIGINT NOT NULL COMMENT '账簿主体ID',
  period CHAR(7) NOT NULL COMMENT '业务期间，格式YYYY-MM',
  start_date DATE NOT NULL COMMENT '期间开始日期',
  end_date DATE NOT NULL COMMENT '期间结束日期',
  status TINYINT NOT NULL DEFAULT 1 COMMENT '期间状态',
  closed_by BIGINT COMMENT '结账人ID',
  closed_at DATETIME COMMENT '结账时间',
  reopened_by BIGINT COMMENT '反结账人ID',
  reopened_at DATETIME COMMENT '反结账时间',
  UNIQUE KEY uk_fin_period (accounting_entity_id, period)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='业务会计期间';

CREATE TABLE IF NOT EXISTS fin_invoice (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  tenant_id BIGINT NOT NULL COMMENT '租户ID',
  invoice_type TINYINT NOT NULL COMMENT '发票类型：采购或销售',
  invoice_no VARCHAR(64) NOT NULL COMMENT '发票号码',
  business_date DATE COMMENT '开票日期',
  party_type TINYINT NOT NULL COMMENT '往来主体类型',
  party_id BIGINT NOT NULL COMMENT '往来主体ID',
  amount BIGINT NOT NULL DEFAULT 0 COMMENT '不含税金额，单位分',
  tax_amount BIGINT NOT NULL DEFAULT 0 COMMENT '税额，单位分',
  total_amount BIGINT NOT NULL DEFAULT 0 COMMENT '价税合计，单位分',
  attachment_url VARCHAR(1024) COMMENT '附件地址',
  status TINYINT NOT NULL DEFAULT 1 COMMENT '状态',
  deleted_at BIGINT NOT NULL DEFAULT 0 COMMENT '删除时间戳，0表示未删除',
  UNIQUE KEY uk_fin_invoice (tenant_id, invoice_no),
  KEY idx_fin_invoice_party (tenant_id, party_type, party_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='采购销售发票登记';

CREATE TABLE IF NOT EXISTS sys_outbox_message (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  tenant_id BIGINT NOT NULL COMMENT '租户ID',
  biz_type TINYINT NOT NULL COMMENT '消息业务类型',
  source_type TINYINT NOT NULL COMMENT '来源类型',
  source_id BIGINT NOT NULL COMMENT '来源ID',
  payload JSON NOT NULL COMMENT '事务外任务数据，供程序处理',
  status TINYINT NOT NULL DEFAULT 1 COMMENT '处理状态',
  retry_count INT NOT NULL DEFAULT 0 COMMENT '重试次数',
  next_retry_at DATETIME COMMENT '下次重试时间',
  last_error VARCHAR(512) COMMENT '最近错误信息',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  UNIQUE KEY uk_outbox_source (biz_type, source_type, source_id),
  KEY idx_outbox_status (status, next_retry_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='事务消息表';

CREATE TABLE IF NOT EXISTS biz_document_state_event (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  tenant_id BIGINT NOT NULL COMMENT '租户ID',
  domain VARCHAR(16) NOT NULL COMMENT '业务域',
  document_type VARCHAR(32) NOT NULL COMMENT '单据类型',
  document_id BIGINT NOT NULL COMMENT '单据ID',
  action VARCHAR(32) NOT NULL COMMENT '操作动作',
  before_status TINYINT COMMENT '变更前状态',
  after_status TINYINT COMMENT '变更后状态',
  operator_id BIGINT COMMENT '操作人ID',
  reason VARCHAR(255) COMMENT '操作原因',
  snapshot JSON COMMENT '操作前后展示快照',
  occurred_at DATETIME NOT NULL COMMENT '业务发生时间',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '写入时间',
  KEY idx_document_event (tenant_id, document_type, document_id, occurred_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='单据状态审计事件，不允许删除';

SET FOREIGN_KEY_CHECKS = 1;
