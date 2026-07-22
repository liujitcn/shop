-- 进销存财务系统核心表结构
-- MySQL 8.0+；金额单位为分，数量按 SKU 基本单位记账。
SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

CREATE TABLE IF NOT EXISTS scm_supplier (
   id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  tenant_id BIGINT NOT NULL COMMENT '租户ID',
  code VARCHAR(32) NOT NULL COMMENT '业务编码',
  name VARCHAR(100) NOT NULL COMMENT '名称',
  contact VARCHAR(50) COMMENT '联系人',
  phone VARCHAR(32) COMMENT '联系电话',
  status TINYINT NOT NULL DEFAULT 1 COMMENT '状态',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  deleted_at BIGINT NOT NULL DEFAULT 0 COMMENT '删除时间戳，0表示未删除',
  UNIQUE KEY uk_tenant_code(tenant_id,code),
  KEY idx_tenant_name(tenant_id,name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='供应商';
CREATE TABLE IF NOT EXISTS scm_warehouse (
   id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  tenant_id BIGINT NOT NULL COMMENT '租户ID',
  code VARCHAR(32) NOT NULL COMMENT '业务编码',
  name VARCHAR(100) NOT NULL COMMENT '名称',
  warehouse_type TINYINT NOT NULL DEFAULT 1,
  address VARCHAR(255),
  manager_id BIGINT,
  allow_negative TINYINT NOT NULL DEFAULT 0,
  status TINYINT NOT NULL DEFAULT 1 COMMENT '状态',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  deleted_at BIGINT NOT NULL DEFAULT 0 COMMENT '删除时间戳，0表示未删除',
  UNIQUE KEY uk_tenant_code(tenant_id,code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='仓库';
CREATE TABLE IF NOT EXISTS scm_warehouse_store (
   id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  tenant_id BIGINT NOT NULL COMMENT '租户ID',
  warehouse_id BIGINT NOT NULL COMMENT '仓库ID',
  tenant_store_id BIGINT NOT NULL,
  priority INT NOT NULL DEFAULT 0,
  status TINYINT NOT NULL DEFAULT 1 COMMENT '状态',
  UNIQUE KEY uk_warehouse_store(warehouse_id,tenant_store_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='仓库服务门店';
CREATE TABLE IF NOT EXISTS scm_location (
   id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  tenant_id BIGINT NOT NULL COMMENT '租户ID',
  warehouse_id BIGINT NOT NULL COMMENT '仓库ID',
  code VARCHAR(32) NOT NULL COMMENT '业务编码',
  name VARCHAR(100) NOT NULL COMMENT '名称',
  location_type TINYINT NOT NULL DEFAULT 1,
  sort INT NOT NULL DEFAULT 0,
  status TINYINT NOT NULL DEFAULT 1 COMMENT '状态',
  UNIQUE KEY uk_warehouse_code(warehouse_id,code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='库位';
CREATE TABLE IF NOT EXISTS scm_goods_setting (
   id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  tenant_id BIGINT NOT NULL COMMENT '租户ID',
  sku_id BIGINT NOT NULL COMMENT 'SKU ID',
  default_warehouse_id BIGINT,
  default_location_id BIGINT,
  inventory_mode TINYINT NOT NULL DEFAULT 1,
  batch_enabled TINYINT NOT NULL DEFAULT 0,
  expiry_enabled TINYINT NOT NULL DEFAULT 0,
  serial_enabled TINYINT NOT NULL DEFAULT 0,
  shelf_life_days INT,
  outbound_strategy TINYINT NOT NULL DEFAULT 1,
  min_stock_num BIGINT NOT NULL DEFAULT 0,
  max_stock_num BIGINT NOT NULL DEFAULT 0,
  UNIQUE KEY uk_tenant_sku(tenant_id,sku_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='SKU仓储属性';
CREATE TABLE IF NOT EXISTS scm_purchase_order (
   id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  tenant_id BIGINT NOT NULL COMMENT '租户ID',
  order_no VARCHAR(32) NOT NULL,
  supplier_id BIGINT NOT NULL,
  warehouse_id BIGINT NOT NULL COMMENT '仓库ID',
  total_money BIGINT NOT NULL DEFAULT 0 COMMENT '金额，单位分',
  received_money BIGINT NOT NULL DEFAULT 0,
  expected_at DATETIME,
  status TINYINT NOT NULL DEFAULT 1 COMMENT '状态',
  created_by BIGINT COMMENT '创建人ID',
  approved_by BIGINT COMMENT '审核人ID',
  approved_at DATETIME,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  UNIQUE KEY uk_tenant_no(tenant_id,order_no),
  KEY idx_supplier(supplier_id,status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='采购订单';
CREATE TABLE IF NOT EXISTS scm_purchase_order_goods (
   id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  purchase_order_id BIGINT NOT NULL,
  sku_id BIGINT NOT NULL COMMENT 'SKU ID',
  sku_code VARCHAR(64),
  goods_name VARCHAR(255),
  unit_id BIGINT,
  unit_ratio_num BIGINT NOT NULL DEFAULT 1,
  unit_ratio_den BIGINT NOT NULL DEFAULT 1,
  order_num BIGINT NOT NULL,
  received_num BIGINT NOT NULL DEFAULT 0,
  price BIGINT NOT NULL DEFAULT 0,
  money BIGINT NOT NULL DEFAULT 0,
  KEY idx_order(purchase_order_id),
  KEY idx_sku(sku_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='采购订单明细';
CREATE TABLE IF NOT EXISTS scm_purchase_receipt (
   id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  tenant_id BIGINT NOT NULL COMMENT '租户ID',
  receipt_no VARCHAR(32) NOT NULL,
  purchase_order_id BIGINT,
  supplier_id BIGINT NOT NULL,
  warehouse_id BIGINT NOT NULL COMMENT '仓库ID',
  total_num BIGINT NOT NULL DEFAULT 0 COMMENT '总数量',
  total_money BIGINT NOT NULL DEFAULT 0 COMMENT '金额，单位分',
  status TINYINT NOT NULL DEFAULT 1 COMMENT '状态',
  received_by BIGINT,
  received_at DATETIME,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  UNIQUE KEY uk_tenant_no(tenant_id,receipt_no)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='采购收货入库';
CREATE TABLE IF NOT EXISTS scm_purchase_receipt_goods (
   id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  receipt_id BIGINT NOT NULL,
  purchase_order_goods_id BIGINT,
  sku_id BIGINT NOT NULL COMMENT 'SKU ID',
  sku_code VARCHAR(64),
  goods_name VARCHAR(255),
  num BIGINT NOT NULL,
  price BIGINT NOT NULL DEFAULT 0,
  money BIGINT NOT NULL DEFAULT 0,
  accepted_num BIGINT NOT NULL DEFAULT 0,
  KEY idx_receipt(receipt_id));
CREATE TABLE IF NOT EXISTS scm_purchase_return (
   id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  tenant_id BIGINT NOT NULL COMMENT '租户ID',
  return_no VARCHAR(32) NOT NULL,
  supplier_id BIGINT NOT NULL,
  warehouse_id BIGINT NOT NULL COMMENT '仓库ID',
  total_num BIGINT NOT NULL DEFAULT 0 COMMENT '总数量',
  total_money BIGINT NOT NULL DEFAULT 0 COMMENT '金额，单位分',
  status TINYINT NOT NULL DEFAULT 1 COMMENT '状态',
  approved_by BIGINT COMMENT '审核人ID',
  approved_at DATETIME,
  UNIQUE KEY uk_tenant_no(tenant_id,return_no)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='采购退货';
CREATE TABLE IF NOT EXISTS scm_purchase_return_goods (
   id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  return_id BIGINT NOT NULL,
  receipt_goods_id BIGINT,
  sku_id BIGINT NOT NULL COMMENT 'SKU ID',
  return_num BIGINT NOT NULL,
  original_price BIGINT NOT NULL DEFAULT 0,
  supplier_credit_money BIGINT NOT NULL DEFAULT 0,
  inventory_cost_money BIGINT NOT NULL DEFAULT 0,
  KEY idx_return(return_id));
CREATE TABLE IF NOT EXISTS stock_account (
   id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  tenant_id BIGINT NOT NULL COMMENT '租户ID',
  warehouse_id BIGINT NOT NULL COMMENT '仓库ID',
  goods_id BIGINT NOT NULL DEFAULT 0,
  sku_id BIGINT NOT NULL COMMENT 'SKU ID',
  on_hand_num BIGINT NOT NULL DEFAULT 0,
  qualified_num BIGINT NOT NULL DEFAULT 0,
  reserved_num BIGINT NOT NULL DEFAULT 0,
  frozen_num BIGINT NOT NULL DEFAULT 0,
  available_num BIGINT NOT NULL DEFAULT 0,
  cost_amount BIGINT NOT NULL DEFAULT 0,
  cost_price BIGINT NOT NULL DEFAULT 0,
  cost_status TINYINT NOT NULL DEFAULT 1,
  stock_version BIGINT NOT NULL DEFAULT 0,
  UNIQUE KEY uk_tenant_warehouse_sku(tenant_id,warehouse_id,sku_id),
  KEY idx_sku(sku_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='仓库SKU库存账户';
CREATE TABLE IF NOT EXISTS stock_batch (
   id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  tenant_id BIGINT NOT NULL COMMENT '租户ID',
  sku_id BIGINT NOT NULL COMMENT 'SKU ID',
  batch_no VARCHAR(64) NOT NULL,
  supplier_batch_no VARCHAR(64),
  production_date DATE,
  expiry_date DATE,
  quality_status TINYINT NOT NULL DEFAULT 1,
  source_type TINYINT COMMENT '来源单据类型',
  source_id BIGINT COMMENT '来源单据ID',
  status TINYINT NOT NULL DEFAULT 1 COMMENT '状态',
  UNIQUE KEY uk_tenant_sku_batch(tenant_id,sku_id,batch_no));
CREATE TABLE IF NOT EXISTS stock_quant (
   id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  tenant_id BIGINT NOT NULL COMMENT '租户ID',
  warehouse_id BIGINT NOT NULL COMMENT '仓库ID',
  location_id BIGINT NOT NULL DEFAULT 0,
  goods_id BIGINT NOT NULL DEFAULT 0,
  sku_id BIGINT NOT NULL COMMENT 'SKU ID',
  batch_id BIGINT NOT NULL DEFAULT 0,
  stock_status TINYINT NOT NULL DEFAULT 2,
  on_hand_num BIGINT NOT NULL DEFAULT 0,
  reserved_num BIGINT NOT NULL DEFAULT 0,
  version BIGINT NOT NULL DEFAULT 0,
  UNIQUE KEY uk_quant(warehouse_id,location_id,sku_id,batch_id,stock_status),
  KEY idx_sku(tenant_id,sku_id));
CREATE TABLE IF NOT EXISTS stock_serial (
   id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  tenant_id BIGINT NOT NULL COMMENT '租户ID',
  sku_id BIGINT NOT NULL COMMENT 'SKU ID',
  serial_no VARCHAR(128) NOT NULL,
  batch_id BIGINT,
  warehouse_id BIGINT,
  location_id BIGINT,
  stock_status TINYINT NOT NULL DEFAULT 2,
  lifecycle_status TINYINT NOT NULL DEFAULT 1,
  source_type TINYINT COMMENT '来源单据类型',
  source_id BIGINT COMMENT '来源单据ID',
  version BIGINT NOT NULL DEFAULT 0,
  UNIQUE KEY uk_tenant_sku_serial(tenant_id,sku_id,serial_no));
CREATE TABLE IF NOT EXISTS stock_serial_event (
   id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  tenant_id BIGINT NOT NULL COMMENT '租户ID',
  serial_id BIGINT NOT NULL,
  event_type TINYINT NOT NULL,
  source_type TINYINT COMMENT '来源单据类型',
  source_id BIGINT COMMENT '来源单据ID',
  before_warehouse_id BIGINT,
  after_warehouse_id BIGINT,
  before_location_id BIGINT,
  after_location_id BIGINT,
  before_status TINYINT,
  after_status TINYINT,
  occurred_at DATETIME NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  KEY idx_serial(serial_id,occurred_at));
CREATE TABLE IF NOT EXISTS stock_reservation (
   id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  tenant_id BIGINT NOT NULL COMMENT '租户ID',
  tenant_store_id BIGINT NOT NULL,
  warehouse_id BIGINT NOT NULL COMMENT '仓库ID',
  sku_id BIGINT NOT NULL COMMENT 'SKU ID',
  source_type TINYINT NOT NULL COMMENT '来源单据类型',
  source_id BIGINT NOT NULL COMMENT '来源单据ID',
  reserved_num BIGINT NOT NULL,
  committed_num BIGINT NOT NULL DEFAULT 0,
  released_num BIGINT NOT NULL DEFAULT 0,
  remaining_num BIGINT NOT NULL,
  status TINYINT NOT NULL DEFAULT 1 COMMENT '状态',
  reserved_at DATETIME NOT NULL,
  UNIQUE KEY uk_source_warehouse(source_type,source_id,warehouse_id));
CREATE TABLE IF NOT EXISTS stock_reservation_event (
   id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  reservation_id BIGINT NOT NULL,
  event_type TINYINT NOT NULL,
  source_type TINYINT COMMENT '来源单据类型',
  source_id BIGINT COMMENT '来源单据ID',
  change_num BIGINT NOT NULL,
  before_remaining_num BIGINT NOT NULL,
  after_remaining_num BIGINT NOT NULL,
  occurred_at DATETIME NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  UNIQUE KEY uk_event(reservation_id,event_type,source_type,source_id));
CREATE TABLE IF NOT EXISTS stock_bill (
   id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  tenant_id BIGINT NOT NULL COMMENT '租户ID',
  bill_no VARCHAR(32) NOT NULL,
  warehouse_id BIGINT NOT NULL COMMENT '仓库ID',
  tenant_store_id BIGINT,
  direction TINYINT NOT NULL,
  biz_type TINYINT NOT NULL,
  source_type TINYINT COMMENT '来源单据类型',
  source_id BIGINT COMMENT '来源单据ID',
  source_no VARCHAR(32),
  reversal_of_id BIGINT,
  total_num BIGINT NOT NULL DEFAULT 0 COMMENT '总数量',
  total_cost_money BIGINT NOT NULL DEFAULT 0,
  status TINYINT NOT NULL DEFAULT 1 COMMENT '状态',
  confirmed_by BIGINT,
  confirmed_at DATETIME,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  UNIQUE KEY uk_tenant_no(tenant_id,bill_no),
  KEY idx_source(source_type,source_id));
CREATE TABLE IF NOT EXISTS stock_bill_goods (
   id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  bill_id BIGINT NOT NULL,
  source_line_id BIGINT,
  goods_id BIGINT,
  sku_id BIGINT NOT NULL COMMENT 'SKU ID',
  sku_code VARCHAR(64),
  goods_name VARCHAR(255),
  num BIGINT NOT NULL,
  base_num BIGINT NOT NULL,
  unit_id BIGINT,
  unit_ratio_num BIGINT NOT NULL DEFAULT 1,
  unit_ratio_den BIGINT NOT NULL DEFAULT 1,
  business_price BIGINT NOT NULL DEFAULT 0,
  cost_money BIGINT NOT NULL DEFAULT 0,
  unit_cost BIGINT NOT NULL DEFAULT 0,
  UNIQUE KEY uk_bill_line_sku(bill_id,source_line_id,sku_id));
CREATE TABLE IF NOT EXISTS stock_bill_goods_allocation (
   id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  bill_goods_id BIGINT NOT NULL,
  location_id BIGINT NOT NULL DEFAULT 0,
  batch_id BIGINT,
  serial_id BIGINT,
  stock_status TINYINT NOT NULL DEFAULT 2,
  base_num BIGINT NOT NULL,
  KEY idx_bill_goods(bill_goods_id));
CREATE TABLE IF NOT EXISTS stock_ledger (
   id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  tenant_id BIGINT NOT NULL COMMENT '租户ID',
  warehouse_id BIGINT NOT NULL COMMENT '仓库ID',
  tenant_store_id BIGINT,
  goods_id BIGINT,
  sku_id BIGINT NOT NULL COMMENT 'SKU ID',
  bill_id BIGINT,
  bill_goods_id BIGINT,
  source_type TINYINT NOT NULL COMMENT '来源单据类型',
  source_id BIGINT NOT NULL COMMENT '来源单据ID',
  direction TINYINT NOT NULL,
  biz_type TINYINT NOT NULL,
  change_num BIGINT NOT NULL,
  before_on_hand_num BIGINT NOT NULL,
  after_on_hand_num BIGINT NOT NULL,
  change_cost_amount BIGINT NOT NULL,
  before_cost_amount BIGINT NOT NULL,
  after_cost_amount BIGINT NOT NULL,
  occurred_at DATETIME NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  UNIQUE KEY uk_source(tenant_id,warehouse_id,sku_id,source_type,source_id),
  KEY idx_account(tenant_id,warehouse_id,sku_id,occurred_at,id));
CREATE TABLE IF NOT EXISTS stock_transfer (
   id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  tenant_id BIGINT NOT NULL COMMENT '租户ID',
  transfer_no VARCHAR(32) NOT NULL,
  source_warehouse_id BIGINT NOT NULL,
  target_warehouse_id BIGINT NOT NULL,
  expected_arrival_at DATETIME,
  status TINYINT NOT NULL DEFAULT 1 COMMENT '状态',
  submitted_by BIGINT,
  approved_by BIGINT COMMENT '审核人ID',
  dispatched_at DATETIME,
  completed_at DATETIME,
  UNIQUE KEY uk_tenant_no(tenant_id,transfer_no));
CREATE TABLE IF NOT EXISTS stock_transfer_goods (
   id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  transfer_id BIGINT NOT NULL,
  sku_id BIGINT NOT NULL COMMENT 'SKU ID',
  apply_num BIGINT NOT NULL,
  dispatched_num BIGINT NOT NULL DEFAULT 0,
  received_num BIGINT NOT NULL DEFAULT 0,
  returned_num BIGINT NOT NULL DEFAULT 0,
  difference_num BIGINT NOT NULL DEFAULT 0);
CREATE TABLE IF NOT EXISTS stock_transfer_transit (
   id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  transfer_goods_id BIGINT NOT NULL,
  batch_id BIGINT,
  serial_id BIGINT,
  stock_status TINYINT NOT NULL,
  dispatched_num BIGINT NOT NULL,
  received_num BIGINT NOT NULL DEFAULT 0,
  returned_num BIGINT NOT NULL DEFAULT 0,
  difference_num BIGINT NOT NULL DEFAULT 0,
  remaining_num BIGINT NOT NULL,
  transit_cost_amount BIGINT NOT NULL DEFAULT 0,
  UNIQUE KEY uk_transit(transfer_goods_id,batch_id,serial_id,stock_status));
CREATE TABLE IF NOT EXISTS stock_check (
   id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  tenant_id BIGINT NOT NULL COMMENT '租户ID',
  check_no VARCHAR(32) NOT NULL,
  warehouse_id BIGINT NOT NULL COMMENT '仓库ID',
  scope_type TINYINT NOT NULL,
  blind_check TINYINT NOT NULL DEFAULT 0,
  snapshot_at DATETIME NOT NULL,
  profit_num BIGINT NOT NULL DEFAULT 0,
  loss_num BIGINT NOT NULL DEFAULT 0,
  status TINYINT NOT NULL DEFAULT 1 COMMENT '状态',
  UNIQUE KEY uk_tenant_no(tenant_id,check_no));
CREATE TABLE IF NOT EXISTS stock_check_goods (
   id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  check_id BIGINT NOT NULL,
  sku_id BIGINT NOT NULL COMMENT 'SKU ID',
  location_id BIGINT,
  batch_id BIGINT,
  serial_id BIGINT,
  stock_status TINYINT NOT NULL,
  snapshot_stock_version BIGINT NOT NULL,
  book_on_hand_num BIGINT NOT NULL,
  real_num BIGINT,
  diff_num BIGINT NOT NULL DEFAULT 0,
  line_status TINYINT NOT NULL DEFAULT 1);
CREATE TABLE IF NOT EXISTS stock_adjustment (
   id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  tenant_id BIGINT NOT NULL COMMENT '租户ID',
  adjustment_no VARCHAR(32) NOT NULL,
  warehouse_id BIGINT NOT NULL COMMENT '仓库ID',
  adjustment_type TINYINT NOT NULL,
  reason TINYINT NOT NULL,
  total_num BIGINT NOT NULL DEFAULT 0 COMMENT '总数量',
  stock_bill_id BIGINT,
  status TINYINT NOT NULL DEFAULT 1 COMMENT '状态',
  UNIQUE KEY uk_tenant_no(tenant_id,adjustment_no));
CREATE TABLE IF NOT EXISTS stock_cost_adjustment (
   id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  tenant_id BIGINT NOT NULL COMMENT '租户ID',
  adjustment_no VARCHAR(32) NOT NULL,
  warehouse_id BIGINT NOT NULL COMMENT '仓库ID',
  reason VARCHAR(255),
  period VARCHAR(7) NOT NULL,
  total_money BIGINT NOT NULL DEFAULT 0 COMMENT '金额，单位分',
  reversal_of_id BIGINT,
  status TINYINT NOT NULL DEFAULT 1 COMMENT '状态',
  UNIQUE KEY uk_tenant_no(tenant_id,adjustment_no));
CREATE TABLE IF NOT EXISTS stock_cost_adjustment_goods (
   id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  adjustment_id BIGINT NOT NULL,
  sku_id BIGINT NOT NULL COMMENT 'SKU ID',
  before_cost_amount BIGINT NOT NULL,
  change_amount BIGINT NOT NULL,
  after_cost_amount BIGINT NOT NULL);
CREATE TABLE IF NOT EXISTS fin_payable (
   id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  tenant_id BIGINT NOT NULL COMMENT '租户ID',
  payable_no VARCHAR(32) NOT NULL,
  supplier_id BIGINT NOT NULL,
  source_type TINYINT COMMENT '来源单据类型',
  source_id BIGINT COMMENT '来源单据ID',
  money BIGINT NOT NULL,
  allocated_money BIGINT NOT NULL DEFAULT 0,
  open_money BIGINT NOT NULL,
  due_date DATE,
  status TINYINT NOT NULL DEFAULT 1 COMMENT '状态',
  UNIQUE KEY uk_tenant_no(tenant_id,payable_no));
CREATE TABLE IF NOT EXISTS fin_payment (
   id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  tenant_id BIGINT NOT NULL COMMENT '租户ID',
  payment_no VARCHAR(32) NOT NULL,
  supplier_id BIGINT,
  source_type TINYINT COMMENT '来源单据类型',
  source_id BIGINT COMMENT '来源单据ID',
  money BIGINT NOT NULL,
  fund_account_id BIGINT NOT NULL,
  money_nature TINYINT NOT NULL DEFAULT 1,
  pay_at DATETIME NOT NULL,
  reversal_of_id BIGINT,
  status TINYINT NOT NULL DEFAULT 1 COMMENT '状态',
  UNIQUE KEY uk_tenant_no(tenant_id,payment_no));
CREATE TABLE IF NOT EXISTS fin_payable_allocation (
   id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  payment_id BIGINT NOT NULL,
  payable_id BIGINT NOT NULL,
  allocated_money BIGINT NOT NULL,
  UNIQUE KEY uk_payment_payable(payment_id,payable_id));
CREATE TABLE IF NOT EXISTS fin_receivable (
   id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  tenant_id BIGINT NOT NULL COMMENT '租户ID',
  receivable_no VARCHAR(32) NOT NULL,
  customer_type TINYINT NOT NULL,
  customer_id BIGINT NOT NULL,
  source_type TINYINT COMMENT '来源单据类型',
  source_id BIGINT COMMENT '来源单据ID',
  money BIGINT NOT NULL,
  allocated_money BIGINT NOT NULL DEFAULT 0,
  open_money BIGINT NOT NULL,
  due_date DATE,
  status TINYINT NOT NULL DEFAULT 1 COMMENT '状态',
  UNIQUE KEY uk_tenant_no(tenant_id,receivable_no));
CREATE TABLE IF NOT EXISTS fin_receipt (
   id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  tenant_id BIGINT NOT NULL COMMENT '租户ID',
  receipt_no VARCHAR(32) NOT NULL,
  customer_type TINYINT NOT NULL,
  customer_id BIGINT NOT NULL,
  source_type TINYINT COMMENT '来源单据类型',
  source_id BIGINT COMMENT '来源单据ID',
  money BIGINT NOT NULL,
  fund_account_id BIGINT NOT NULL,
  money_nature TINYINT NOT NULL DEFAULT 1,
  receive_at DATETIME NOT NULL,
  status TINYINT NOT NULL DEFAULT 1 COMMENT '状态',
  UNIQUE KEY uk_tenant_no(tenant_id,receipt_no));
CREATE TABLE IF NOT EXISTS fin_receivable_allocation (
   id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  receipt_id BIGINT NOT NULL,
  receivable_id BIGINT NOT NULL,
  allocated_money BIGINT NOT NULL,
  UNIQUE KEY uk_receipt_receivable(receipt_id,receivable_id));
CREATE TABLE IF NOT EXISTS fin_fund_account (
   id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  tenant_id BIGINT NOT NULL COMMENT '租户ID',
  accounting_entity_id BIGINT NOT NULL,
  account_no VARCHAR(32) NOT NULL,
  name VARCHAR(100) NOT NULL COMMENT '名称',
  account_type TINYINT NOT NULL,
  currency_code CHAR(3) NOT NULL DEFAULT 'CNY',
  opening_balance BIGINT NOT NULL DEFAULT 0,
  current_balance BIGINT NOT NULL DEFAULT 0,
  allow_negative TINYINT NOT NULL DEFAULT 0,
  status TINYINT NOT NULL DEFAULT 1 COMMENT '状态',
  UNIQUE KEY uk_tenant_account(tenant_id,account_no));
CREATE TABLE IF NOT EXISTS fin_fund_flow (
   id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  tenant_id BIGINT NOT NULL COMMENT '租户ID',
  fund_account_id BIGINT NOT NULL,
  direction TINYINT NOT NULL,
  biz_type TINYINT NOT NULL,
  source_type TINYINT NOT NULL COMMENT '来源单据类型',
  source_id BIGINT NOT NULL COMMENT '来源单据ID',
  change_money BIGINT NOT NULL,
  before_balance BIGINT NOT NULL,
  after_balance BIGINT NOT NULL,
  reversal_of_id BIGINT,
  occurred_at DATETIME NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  UNIQUE KEY uk_source(fund_account_id,biz_type,source_type,source_id));
CREATE TABLE IF NOT EXISTS fin_accounting_period (
   id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  accounting_entity_id BIGINT NOT NULL,
  period CHAR(7) NOT NULL,
  start_date DATE NOT NULL,
  end_date DATE NOT NULL,
  status TINYINT NOT NULL DEFAULT 1 COMMENT '状态',
  closed_by BIGINT,
  closed_at DATETIME,
  reopened_by BIGINT,
  reopened_at DATETIME,
  UNIQUE KEY uk_entity_period(accounting_entity_id,period));
CREATE TABLE IF NOT EXISTS fin_voucher (
   id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  tenant_id BIGINT NOT NULL COMMENT '租户ID',
  voucher_no VARCHAR(32) NOT NULL,
  accounting_entity_id BIGINT NOT NULL,
  source_tenant_id BIGINT NOT NULL,
  period CHAR(7) NOT NULL,
  event_type TINYINT NOT NULL,
  source_type TINYINT NOT NULL COMMENT '来源单据类型',
  source_id BIGINT NOT NULL COMMENT '来源单据ID',
  debit_money BIGINT NOT NULL DEFAULT 0,
  credit_money BIGINT NOT NULL DEFAULT 0,
  summary VARCHAR(255),
  occurred_at DATETIME NOT NULL,
  status TINYINT NOT NULL DEFAULT 1 COMMENT '状态',
  reversal_of_id BIGINT,
  UNIQUE KEY uk_event(accounting_entity_id,event_type,source_type,source_id));
CREATE TABLE IF NOT EXISTS fin_voucher_entry (
   id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  voucher_id BIGINT NOT NULL,
  line_no INT NOT NULL,
  subject_key VARCHAR(50) NOT NULL,
  subject_code VARCHAR(20),
  subject_name VARCHAR(100),
  direction TINYINT NOT NULL,
  money BIGINT NOT NULL,
  tenant_store_id BIGINT,
  supplier_id BIGINT,
  merchant_tenant_id BIGINT,
  UNIQUE KEY uk_voucher_line(voucher_id,line_no));
CREATE TABLE IF NOT EXISTS fin_invoice (
   id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  tenant_id BIGINT NOT NULL COMMENT '租户ID',
  invoice_type TINYINT NOT NULL,
  invoice_no VARCHAR(64) NOT NULL,
  business_date DATE,
  party_type TINYINT NOT NULL,
  party_id BIGINT NOT NULL,
  amount BIGINT NOT NULL DEFAULT 0,
  tax_amount BIGINT NOT NULL DEFAULT 0,
  total_amount BIGINT NOT NULL DEFAULT 0,
  attachment_url VARCHAR(1024),
  reversal_of_id BIGINT,
  status TINYINT NOT NULL DEFAULT 1 COMMENT '状态',
  UNIQUE KEY uk_tenant_invoice(tenant_id,invoice_no));
CREATE TABLE IF NOT EXISTS sys_outbox_message (
   id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  tenant_id BIGINT NOT NULL COMMENT '租户ID',
  biz_type TINYINT NOT NULL,
  source_type TINYINT NOT NULL COMMENT '来源单据类型',
  source_id BIGINT NOT NULL COMMENT '来源单据ID',
  payload JSON NOT NULL,
  status TINYINT NOT NULL DEFAULT 1 COMMENT '状态',
  retry_count INT NOT NULL DEFAULT 0,
  next_retry_at DATETIME,
  last_error VARCHAR(512),
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  UNIQUE KEY uk_message(biz_type,source_type,source_id));
CREATE TABLE IF NOT EXISTS biz_document_state_event (
   id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  tenant_id BIGINT NOT NULL COMMENT '租户ID',
  domain VARCHAR(16) NOT NULL,
  document_type VARCHAR(32) NOT NULL,
  document_id BIGINT NOT NULL,
  action VARCHAR(32) NOT NULL,
  before_status TINYINT,
  after_status TINYINT,
  operator_id BIGINT,
  reason VARCHAR(255),
  occurred_at DATETIME NOT NULL,
  KEY idx_document(tenant_id,document_type,document_id));
SET FOREIGN_KEY_CHECKS = 1;

-- 统一补充项目约定的 int64 软删除字段（0 表示未删除）。
ALTER TABLE scm_supplier ADD COLUMN IF NOT EXISTS deleted_at BIGINT NOT NULL DEFAULT 0 COMMENT '删除时间戳，0表示未删除';
ALTER TABLE scm_warehouse ADD COLUMN IF NOT EXISTS deleted_at BIGINT NOT NULL DEFAULT 0 COMMENT '删除时间戳，0表示未删除';
ALTER TABLE scm_warehouse_store ADD COLUMN IF NOT EXISTS deleted_at BIGINT NOT NULL DEFAULT 0 COMMENT '删除时间戳，0表示未删除';
ALTER TABLE scm_location ADD COLUMN IF NOT EXISTS deleted_at BIGINT NOT NULL DEFAULT 0 COMMENT '删除时间戳，0表示未删除';
ALTER TABLE scm_goods_setting ADD COLUMN IF NOT EXISTS deleted_at BIGINT NOT NULL DEFAULT 0 COMMENT '删除时间戳，0表示未删除';
ALTER TABLE scm_purchase_order ADD COLUMN IF NOT EXISTS deleted_at BIGINT NOT NULL DEFAULT 0 COMMENT '删除时间戳，0表示未删除';
ALTER TABLE scm_purchase_order_goods ADD COLUMN IF NOT EXISTS deleted_at BIGINT NOT NULL DEFAULT 0 COMMENT '删除时间戳，0表示未删除';
ALTER TABLE scm_purchase_receipt ADD COLUMN IF NOT EXISTS deleted_at BIGINT NOT NULL DEFAULT 0 COMMENT '删除时间戳，0表示未删除';
ALTER TABLE scm_purchase_receipt_goods ADD COLUMN IF NOT EXISTS deleted_at BIGINT NOT NULL DEFAULT 0 COMMENT '删除时间戳，0表示未删除';
ALTER TABLE scm_purchase_return ADD COLUMN IF NOT EXISTS deleted_at BIGINT NOT NULL DEFAULT 0 COMMENT '删除时间戳，0表示未删除';
ALTER TABLE scm_purchase_return_goods ADD COLUMN IF NOT EXISTS deleted_at BIGINT NOT NULL DEFAULT 0 COMMENT '删除时间戳，0表示未删除';
ALTER TABLE stock_account ADD COLUMN IF NOT EXISTS deleted_at BIGINT NOT NULL DEFAULT 0 COMMENT '删除时间戳，0表示未删除';
ALTER TABLE stock_batch ADD COLUMN IF NOT EXISTS deleted_at BIGINT NOT NULL DEFAULT 0 COMMENT '删除时间戳，0表示未删除';
ALTER TABLE stock_quant ADD COLUMN IF NOT EXISTS deleted_at BIGINT NOT NULL DEFAULT 0 COMMENT '删除时间戳，0表示未删除';
ALTER TABLE stock_serial ADD COLUMN IF NOT EXISTS deleted_at BIGINT NOT NULL DEFAULT 0 COMMENT '删除时间戳，0表示未删除';
ALTER TABLE stock_serial_event ADD COLUMN IF NOT EXISTS deleted_at BIGINT NOT NULL DEFAULT 0 COMMENT '删除时间戳，0表示未删除';
ALTER TABLE stock_reservation ADD COLUMN IF NOT EXISTS deleted_at BIGINT NOT NULL DEFAULT 0 COMMENT '删除时间戳，0表示未删除';
ALTER TABLE stock_reservation_event ADD COLUMN IF NOT EXISTS deleted_at BIGINT NOT NULL DEFAULT 0 COMMENT '删除时间戳，0表示未删除';
ALTER TABLE stock_bill ADD COLUMN IF NOT EXISTS deleted_at BIGINT NOT NULL DEFAULT 0 COMMENT '删除时间戳，0表示未删除';
ALTER TABLE stock_bill_goods ADD COLUMN IF NOT EXISTS deleted_at BIGINT NOT NULL DEFAULT 0 COMMENT '删除时间戳，0表示未删除';
ALTER TABLE stock_bill_goods_allocation ADD COLUMN IF NOT EXISTS deleted_at BIGINT NOT NULL DEFAULT 0 COMMENT '删除时间戳，0表示未删除';
ALTER TABLE stock_ledger ADD COLUMN IF NOT EXISTS deleted_at BIGINT NOT NULL DEFAULT 0 COMMENT '删除时间戳，0表示未删除';
ALTER TABLE stock_transfer ADD COLUMN IF NOT EXISTS deleted_at BIGINT NOT NULL DEFAULT 0 COMMENT '删除时间戳，0表示未删除';
ALTER TABLE stock_transfer_goods ADD COLUMN IF NOT EXISTS deleted_at BIGINT NOT NULL DEFAULT 0 COMMENT '删除时间戳，0表示未删除';
ALTER TABLE stock_transfer_transit ADD COLUMN IF NOT EXISTS deleted_at BIGINT NOT NULL DEFAULT 0 COMMENT '删除时间戳，0表示未删除';
ALTER TABLE stock_check ADD COLUMN IF NOT EXISTS deleted_at BIGINT NOT NULL DEFAULT 0 COMMENT '删除时间戳，0表示未删除';
ALTER TABLE stock_check_goods ADD COLUMN IF NOT EXISTS deleted_at BIGINT NOT NULL DEFAULT 0 COMMENT '删除时间戳，0表示未删除';
ALTER TABLE stock_adjustment ADD COLUMN IF NOT EXISTS deleted_at BIGINT NOT NULL DEFAULT 0 COMMENT '删除时间戳，0表示未删除';
ALTER TABLE stock_cost_adjustment ADD COLUMN IF NOT EXISTS deleted_at BIGINT NOT NULL DEFAULT 0 COMMENT '删除时间戳，0表示未删除';
ALTER TABLE stock_cost_adjustment_goods ADD COLUMN IF NOT EXISTS deleted_at BIGINT NOT NULL DEFAULT 0 COMMENT '删除时间戳，0表示未删除';
ALTER TABLE fin_payable ADD COLUMN IF NOT EXISTS deleted_at BIGINT NOT NULL DEFAULT 0 COMMENT '删除时间戳，0表示未删除';
ALTER TABLE fin_payment ADD COLUMN IF NOT EXISTS deleted_at BIGINT NOT NULL DEFAULT 0 COMMENT '删除时间戳，0表示未删除';
ALTER TABLE fin_payable_allocation ADD COLUMN IF NOT EXISTS deleted_at BIGINT NOT NULL DEFAULT 0 COMMENT '删除时间戳，0表示未删除';
ALTER TABLE fin_receivable ADD COLUMN IF NOT EXISTS deleted_at BIGINT NOT NULL DEFAULT 0 COMMENT '删除时间戳，0表示未删除';
ALTER TABLE fin_receipt ADD COLUMN IF NOT EXISTS deleted_at BIGINT NOT NULL DEFAULT 0 COMMENT '删除时间戳，0表示未删除';
ALTER TABLE fin_receivable_allocation ADD COLUMN IF NOT EXISTS deleted_at BIGINT NOT NULL DEFAULT 0 COMMENT '删除时间戳，0表示未删除';
ALTER TABLE fin_fund_account ADD COLUMN IF NOT EXISTS deleted_at BIGINT NOT NULL DEFAULT 0 COMMENT '删除时间戳，0表示未删除';
ALTER TABLE fin_fund_flow ADD COLUMN IF NOT EXISTS deleted_at BIGINT NOT NULL DEFAULT 0 COMMENT '删除时间戳，0表示未删除';
ALTER TABLE fin_accounting_period ADD COLUMN IF NOT EXISTS deleted_at BIGINT NOT NULL DEFAULT 0 COMMENT '删除时间戳，0表示未删除';
ALTER TABLE fin_voucher ADD COLUMN IF NOT EXISTS deleted_at BIGINT NOT NULL DEFAULT 0 COMMENT '删除时间戳，0表示未删除';
ALTER TABLE fin_voucher_entry ADD COLUMN IF NOT EXISTS deleted_at BIGINT NOT NULL DEFAULT 0 COMMENT '删除时间戳，0表示未删除';
ALTER TABLE fin_invoice ADD COLUMN IF NOT EXISTS deleted_at BIGINT NOT NULL DEFAULT 0 COMMENT '删除时间戳，0表示未删除';
ALTER TABLE sys_outbox_message ADD COLUMN IF NOT EXISTS deleted_at BIGINT NOT NULL DEFAULT 0 COMMENT '删除时间戳，0表示未删除';
ALTER TABLE biz_document_state_event ADD COLUMN IF NOT EXISTS deleted_at BIGINT NOT NULL DEFAULT 0 COMMENT '删除时间戳，0表示未删除';

-- 表级注释（字段注释已在字段定义或上面的软删除字段中声明）。
ALTER TABLE scm_supplier COMMENT='供应商主数据';
ALTER TABLE scm_warehouse COMMENT='仓库主数据';
ALTER TABLE scm_warehouse_store COMMENT='仓库与门店服务关系';
ALTER TABLE scm_location COMMENT='仓库库位';
ALTER TABLE scm_goods_setting COMMENT='SKU仓储管理属性';
ALTER TABLE scm_purchase_order COMMENT='采购订单';
ALTER TABLE scm_purchase_order_goods COMMENT='采购订单明细';
ALTER TABLE scm_purchase_receipt COMMENT='采购收货单';
ALTER TABLE scm_purchase_receipt_goods COMMENT='采购收货明细';
ALTER TABLE scm_purchase_return COMMENT='采购退货单';
ALTER TABLE scm_purchase_return_goods COMMENT='采购退货明细';
ALTER TABLE stock_account COMMENT='仓库SKU库存账户';
ALTER TABLE stock_batch COMMENT='库存批次档案';
ALTER TABLE stock_quant COMMENT='库存实物明细';
ALTER TABLE stock_serial COMMENT='库存序列号';
ALTER TABLE stock_serial_event COMMENT='序列号生命周期事件';
ALTER TABLE stock_reservation COMMENT='库存预占余额';
ALTER TABLE stock_reservation_event COMMENT='库存预占事件';
ALTER TABLE stock_bill COMMENT='库存出入库单';
ALTER TABLE stock_bill_goods COMMENT='库存出入库明细';
ALTER TABLE stock_bill_goods_allocation COMMENT='库存明细追溯分配';
ALTER TABLE stock_ledger COMMENT='库存成本数量台账';
ALTER TABLE stock_transfer COMMENT='库存调拨单';
ALTER TABLE stock_transfer_goods COMMENT='库存调拨明细';
ALTER TABLE stock_transfer_transit COMMENT='调拨在途库存';
ALTER TABLE stock_check COMMENT='库存盘点单';
ALTER TABLE stock_check_goods COMMENT='库存盘点明细';
ALTER TABLE stock_adjustment COMMENT='库存报损报溢单';
ALTER TABLE stock_cost_adjustment COMMENT='库存成本调整单';
ALTER TABLE stock_cost_adjustment_goods COMMENT='库存成本调整明细';
ALTER TABLE fin_payable COMMENT='供应商应付单';
ALTER TABLE fin_payment COMMENT='供应商付款单';
ALTER TABLE fin_payable_allocation COMMENT='付款核销应付';
ALTER TABLE fin_receivable COMMENT='客户应收单';
ALTER TABLE fin_receipt COMMENT='客户收款单';
ALTER TABLE fin_receivable_allocation COMMENT='收款核销应收';
ALTER TABLE fin_fund_account COMMENT='资金账户';
ALTER TABLE fin_fund_flow COMMENT='不可变资金流水';
ALTER TABLE fin_accounting_period COMMENT='业务会计期间';
ALTER TABLE fin_voucher COMMENT='业务凭证';
ALTER TABLE fin_voucher_entry COMMENT='业务凭证分录';
ALTER TABLE fin_invoice COMMENT='采购销售发票登记';
ALTER TABLE sys_outbox_message COMMENT='事务消息 outbox';
ALTER TABLE biz_document_state_event COMMENT='业务单据状态审计事件';

-- 明细及关联表补充幂等唯一索引。
ALTER TABLE scm_purchase_order_goods ADD UNIQUE KEY IF NOT EXISTS uk_order_sku_line(purchase_order_id,id,sku_id);
ALTER TABLE scm_purchase_receipt_goods ADD UNIQUE KEY IF NOT EXISTS uk_receipt_line(receipt_id,id);
ALTER TABLE scm_purchase_return_goods ADD UNIQUE KEY IF NOT EXISTS uk_return_line(return_id,id);
ALTER TABLE stock_transfer_goods ADD UNIQUE KEY IF NOT EXISTS uk_transfer_sku_line(transfer_id,id,sku_id);
ALTER TABLE stock_check_goods ADD UNIQUE KEY IF NOT EXISTS uk_check_line(check_id,id,sku_id,location_id,batch_id,serial_id);
ALTER TABLE stock_adjustment ADD UNIQUE KEY IF NOT EXISTS uk_adjustment_bill(tenant_id,adjustment_no);
ALTER TABLE stock_cost_adjustment_goods ADD UNIQUE KEY IF NOT EXISTS uk_cost_adjustment_sku(adjustment_id,sku_id);
