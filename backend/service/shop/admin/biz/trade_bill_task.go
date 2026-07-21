package biz

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	shopappv1 "shop/api/gen/go/shop/app/v1"

	shopadminv1 "shop/api/gen/go/shop/admin/v1"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	"shop/pkg/job"
	_const "shop/service/shop/consts"
	"shop/service/shop/workspaceevent"
	"shop/service/shop/wx"
	"shop/service/shop/wx/bill"

	"github.com/go-kratos/kratos/v3/log"
	_string "github.com/liujitcn/go-utils/string"
	_time "github.com/liujitcn/go-utils/time"
	"github.com/liujitcn/go-utils/trans"
	"github.com/liujitcn/gorm-kit/repository"
	"github.com/liujitcn/kratos-kit/oss"
	"gorm.io/gorm"
)

// TradeBill 申请交易账单
type TradeBill struct {
	data             *data.Data
	oss              oss.OSS
	tx               data.Transaction
	wxPayCase        *wx.WxPayCase
	payBillRepo      *data.PayBillRepository
	orderPaymentRepo *data.OrderPaymentRepository
	orderRefundRepo  *data.OrderRefundRepository
	ctx              context.Context
}

var _ job.TaskExec = (*TradeBill)(nil)

// tradeBillCheckResult 表示单类交易账单核对结果。
type tradeBillCheckResult struct {
	name           string
	localCount     int64
	thirdCount     int64
	successCount   int
	failedCount    int
	unmatchedCount int
	localAmount    int64
	thirdAmount    int64
}

// NewTradeBill 创建交易账单任务实例
func NewTradeBill(
	dataStore *data.Data,
	oss oss.OSS,
	tx data.Transaction,
	wxPayCase *wx.WxPayCase,
	payBillRepo *data.PayBillRepository,
	orderPaymentRepo *data.OrderPaymentRepository,
	orderRefundRepo *data.OrderRefundRepository,
) *TradeBill {
	return &TradeBill{
		data:             dataStore,
		oss:              oss,
		tx:               tx,
		wxPayCase:        wxPayCase,
		payBillRepo:      payBillRepo,
		orderPaymentRepo: orderPaymentRepo,
		orderRefundRepo:  orderRefundRepo,
		ctx:              context.Background(),
	}
}

// Exec 执行交易账单下载与核对任务
func (t *TradeBill) Exec(args map[string]string) ([]string, error) {
	log.Info(fmt.Sprintf("Job TradeBill Exec %+v", args))
	v, ok := args["billDate"]
	var now *time.Time
	// 任务参数显式指定账单日期时，优先按指定日期执行。
	if ok && len(v) > 0 {
		now = _time.StringTimeToTime(v)
	}
	// 未指定账单日期时，默认回退到前一天执行对账。
	if now == nil {
		defaultBillDate := time.Now().AddDate(0, 0, -1)
		now = &defaultBillDate
	}

	billDate := _time.TimeToDateString(*now)

	ret := make([]string, 0, 2)
	payment, paymentErr := t.payment(billDate, bill.BILL_TYPE_SUCCESS)
	// 支付账单核对失败时，记录错误信息并继续处理退款账单。
	if paymentErr != nil {
		ret = append(ret, "支付账单核对失败")
	} else {
		ret = append(ret, payment.formatMessage())
	}
	refund, refundErr := t.refund(billDate, bill.BILL_TYPE_REFUND)
	// 退款账单核对失败时，记录错误信息供任务结果统一返回。
	if refundErr != nil {
		ret = append(ret, "退款账单核对失败")
	} else {
		ret = append(ret, refund.formatMessage())
	}
	// 任一账单核对失败时，都需要把失败状态返回给任务调度器。
	return ret, errors.Join(paymentErr, refundErr)
}

// payment 核对支付账单
func (t *TradeBill) payment(billDate, billType string) (tradeBillCheckResult, error) {
	result := tradeBillCheckResult{name: "支付账单"}
	payBill, err := t.downloadBill(billDate, billType)
	// 账单下载失败时，当前账单类型无法继续核对。
	if err != nil {
		return result, err
	}
	// 查询全部支付订单
	var paymentList []*models.OrderPayment
	var startTime, endTime time.Time
	startTime, endTime, err = billDateRange(billDate)
	// 账单日期解析失败时，直接终止本次支付账单核对。
	if err != nil {
		return result, err
	}
	query := t.data.Query(t.ctx).OrderPayment
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Where(query.SuccessTime.Gte(startTime)))
	opts = append(opts, repository.Where(query.SuccessTime.Lt(endTime)))
	opts = append(opts, repository.Where(query.TradeState.Eq(shopappv1.PaymentResource_SUCCESS.String())))
	paymentList, err = t.orderPaymentRepo.List(
		t.ctx,
		opts...,
	)
	// 本地支付记录查询失败时，无法继续执行账单比对。
	if err != nil {
		return result, err
	}
	result.localCount = int64(len(paymentList))
	// 转换map
	paymentMap := make(map[string]*models.OrderPayment)
	for _, payment := range paymentList {
		// 本地记录默认视为对账失败，只有渠道账单明确命中且金额状态一致时才改为成功。
		payment.Status = _const.ORDER_BILL_STATUS_CHECK_FAIL
		paymentMap[fmt.Sprintf("%s_%s", payment.TradeNo, payment.ThirdOrderNo)] = payment
	}

	// 获取对账单内容
	var fileByte []byte
	fileByte, err = t.oss.GetFileByte(payBill.FilePath)
	// 账单文件读取失败时，无法继续校验与解析。
	if err != nil {
		return result, err
	}
	err = t.checkHash(fileByte, payBill.HashValue)
	if err != nil {
		return result, err
	}

	reader := csv.NewReader(bytes.NewReader(fileByte))
	reader.Comma = ','       // 设置分隔符
	reader.LazyQuotes = true // 允许非常规引号

	// 跳过标题行
	_, _ = reader.Read()

	for {
		var record []string
		record, err = reader.Read()
		// 读到文件结尾时，结束当前账单解析循环。
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}

		// 去除每个字段的反引号
		for i := range record {
			record[i] = strings.Trim(record[i], "`")
		}
		// 仅处理微信支付账单的标准列数记录。
		switch len(record) {
		case 20:
			// 计算金额
			amount := _string.ConvertYuanStringToFen(record[12])
			payBill.ThirdTotalCount += 1
			payBill.ThirdTotalAmount += amount
			// 交易记录
			key := fmt.Sprintf("%s_%s", record[6], record[5])
			// 记录在数据库不存在，暂时记录日期，后续在做处理
			if v, ok := paymentMap[key]; ok {
				var orderPaymentAmount *shopadminv1.OrderPayment_Amount
				orderPaymentAmount, err = parseOrderPaymentAmount(v.Amount)
				// 金额 JSON 解析失败时，直接终止对账，避免把坏数据按 0 金额继续统计。
				if err != nil {
					return result, err
				}
				// 支付金额和状态一致
				if v.TradeState == record[9] && orderPaymentAmount.GetPayerTotal() == amount {
					v.Status = _const.ORDER_BILL_STATUS_CHECK_SUCCESS
					result.successCount++
				} else {
					v.Status = _const.ORDER_BILL_STATUS_CHECK_FAIL
					result.failedCount++
				}
			} else {
				result.unmatchedCount++
			}
		default:
			continue
		}
	}
	result.failedCount = int(result.localCount) - result.successCount
	err = t.tx.Transaction(t.ctx, func(ctx context.Context) error {
		for _, v := range paymentMap {
			payBill.TotalCount += 1
			orderPaymentAmount, parseErr := parseOrderPaymentAmount(v.Amount)
			// 金额 JSON 解析失败时，终止当前事务，避免错误金额写入账单汇总。
			if parseErr != nil {
				return parseErr
			}
			payBill.TotalAmount += orderPaymentAmount.GetPayerTotal()
			err = t.orderPaymentRepo.UpdateByID(ctx, v)
			if err != nil {
				return err
			}
		}

		// 本地与三方统计完全一致时，标记本次支付账单核对成功。
		if payBill.TotalCount == payBill.ThirdTotalCount && payBill.TotalAmount == payBill.ThirdTotalAmount {
			payBill.Status = 1
		} else {
			payBill.Status = 2
		}
		return t.payBillRepo.UpdateByID(ctx, payBill)
	})
	// 对账结果事务提交失败时，直接返回错误供任务重试。
	if err != nil {
		return result, err
	}
	workspaceevent.Publish(t.ctx, workspaceevent.ReasonPayBillChecked, workspaceevent.AreaRisk)

	result.localAmount = payBill.TotalAmount
	result.thirdAmount = payBill.ThirdTotalAmount
	result.thirdCount = payBill.ThirdTotalCount
	return result, nil
}

// downloadBill 下载并初始化对账单记录
func (t *TradeBill) downloadBill(billDate, billType string) (*models.PayBill, error) {
	// 获取当前定时账单日期
	query := t.data.Query(t.ctx).PayBill
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.BillDate.Eq(billDate)))
	opts = append(opts, repository.Where(query.BillType.Eq(billType)))
	first, err := t.payBillRepo.Find(
		t.ctx,
		opts...,
	)
	// 已存在的账单查询出现非“未找到”错误时，直接返回。
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	// 账单记录不存在时，先向微信申请并落本地账单文件。
	if errors.Is(err, gorm.ErrRecordNotFound) || first == nil {
		// 申请账单
		var tradeBill *bill.TradeBillResponse
		tradeBill, err = t.wxPayCase.TradeBill(bill.TradeBillRequest{
			BillDate: &billDate,
			BillType: &billType,
		})
		if err != nil {
			return nil, err
		}

		// 下载账单
		var billByte []byte
		billByte, err = t.wxPayCase.DownloadBill(trans.StringValue(tradeBill.DownloadURL))
		if err != nil {
			return nil, err
		}

		// 校验Hash
		hashValue := trans.StringValue(tradeBill.HashValue)
		err = t.checkHash(billByte, hashValue)
		if err != nil {
			return nil, err
		}

		var path string
		path, err = t.oss.UploadByByte(fmt.Sprintf("%s.csv", billType), fmt.Sprintf("bill/file/%s", strings.ReplaceAll(billDate, "-", "/")), billByte)
		if err != nil {
			return nil, err
		}

		first = &models.PayBill{
			BillDate:  billDate,
			BillType:  billType,
			FilePath:  path,
			HashType:  trans.StringValue(tradeBill.HashType),
			HashValue: hashValue,
		}
		err = t.payBillRepo.Create(t.ctx, first)
		if err != nil {
			return nil, err
		}
	} else {
		// 重新计算
		first.TotalCount = 0
		first.TotalAmount = 0
		first.ThirdTotalCount = 0
		first.ThirdTotalAmount = 0
		first.Status = 0
	}
	return first, nil
}

// checkHash 校验账单文件哈希值
func (t *TradeBill) checkHash(fileBytes []byte, hashValue string) error {
	hash := sha1.New()
	hash.Write(fileBytes)
	hashSum := hash.Sum(nil) // 返回 [20]byte 的切片
	// 将哈希值转换为十六进制字符串（与常见工具格式一致）
	hashHex := fmt.Sprintf("%x", hashSum)
	// 账单内容与期望哈希不一致时，视为文件校验失败。
	if hashHex != hashValue {
		return errorsx.Internal("校验账单文件失败")
	}
	return nil
}

// refund 核对退款账单
func (t *TradeBill) refund(billDate, billType string) (tradeBillCheckResult, error) {
	result := tradeBillCheckResult{name: "退款账单"}
	payBill, err := t.downloadBill(billDate, billType)
	// 账单下载失败时，当前账单类型无法继续核对。
	if err != nil {
		return result, err
	}
	// 查询全部退款订单
	var refundList []*models.OrderRefund
	var startTime, endTime time.Time
	startTime, endTime, err = billDateRange(billDate)
	// 账单日期解析失败时，直接终止本次退款账单核对。
	if err != nil {
		return result, err
	}
	query := t.data.Query(t.ctx).OrderRefund
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Where(query.SuccessTime.Gte(startTime)))
	opts = append(opts, repository.Where(query.SuccessTime.Lt(endTime)))
	opts = append(opts, repository.Where(query.RefundState.Eq(shopappv1.RefundResource_SUCCESS.String())))
	refundList, err = t.orderRefundRepo.List(
		t.ctx,
		opts...,
	)
	// 本地退款记录查询失败时，无法继续执行账单比对。
	if err != nil {
		return result, err
	}
	result.localCount = int64(len(refundList))
	// 转换map
	refundMap := make(map[string]*models.OrderRefund)
	for _, refund := range refundList {
		// 本地记录默认视为对账失败，只有渠道账单明确命中且金额状态一致时才改为成功。
		refund.Status = _const.ORDER_BILL_STATUS_CHECK_FAIL
		refundMap[fmt.Sprintf("%s_%s_%s_%s", refund.TradeNo, refund.ThirdOrderNo, refund.RefundNo, refund.ThirdRefundNo)] = refund
	}

	// 获取对账单内容
	var fileByte []byte
	fileByte, err = t.oss.GetFileByte(payBill.FilePath)
	// 账单文件读取失败时，无法继续校验与解析。
	if err != nil {
		return result, err
	}
	err = t.checkHash(fileByte, payBill.HashValue)
	if err != nil {
		return result, err
	}

	reader := csv.NewReader(bytes.NewReader(fileByte))
	reader.Comma = ','       // 设置分隔符
	reader.LazyQuotes = true // 允许非常规引号

	// 跳过标题行
	_, _ = reader.Read()

	for {
		var record []string
		record, err = reader.Read()
		// 读到文件结尾时，结束当前账单解析循环。
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}

		// 去除每个字段的反引号
		for i := range record {
			record[i] = strings.Trim(record[i], "`")
		}
		// 仅处理微信退款账单的标准列数记录。
		switch len(record) {
		case 29:
			// 计算金额
			amount := _string.ConvertYuanStringToFen(record[18])
			payBill.ThirdTotalCount += 1
			payBill.ThirdTotalAmount += amount
			// 交易记录
			key := fmt.Sprintf("%s_%s_%s_%s", record[6], record[5], record[17], record[16])
			// 记录在数据库不存在，暂时记录日期，后续在做处理
			if v, ok := refundMap[key]; ok {
				var orderRefundAmount *shopadminv1.OrderRefund_Amount
				orderRefundAmount, err = parseOrderRefundAmount(v.Amount)
				// 金额 JSON 解析失败时，直接终止对账，避免把坏数据按 0 金额继续统计。
				if err != nil {
					return result, err
				}
				// 支付金额和状态一致
				if v.RefundState == record[21] && orderRefundAmount.GetPayerRefund() == amount {
					v.Status = _const.ORDER_BILL_STATUS_CHECK_SUCCESS
					result.successCount++
				} else {
					v.Status = _const.ORDER_BILL_STATUS_CHECK_FAIL
					result.failedCount++
				}
			} else {
				result.unmatchedCount++
			}
		default:
			continue
		}
	}
	result.failedCount = int(result.localCount) - result.successCount
	err = t.tx.Transaction(t.ctx, func(ctx context.Context) error {
		for _, v := range refundMap {
			payBill.TotalCount += 1
			orderRefundAmount, parseErr := parseOrderRefundAmount(v.Amount)
			// 金额 JSON 解析失败时，终止当前事务，避免错误金额写入账单汇总。
			if parseErr != nil {
				return parseErr
			}
			payBill.TotalAmount += orderRefundAmount.GetPayerRefund()
			err = t.orderRefundRepo.UpdateByID(ctx, v)
			if err != nil {
				return err
			}
		}

		// 本地与三方统计完全一致时，标记本次退款账单核对成功。
		if payBill.TotalCount == payBill.ThirdTotalCount && payBill.TotalAmount == payBill.ThirdTotalAmount {
			payBill.Status = 1
		} else {
			payBill.Status = 2
		}
		return t.payBillRepo.UpdateByID(ctx, payBill)
	})
	// 对账结果事务提交失败时，直接返回错误供任务重试。
	if err != nil {
		return result, err
	}
	workspaceevent.Publish(t.ctx, workspaceevent.ReasonPayBillChecked, workspaceevent.AreaRisk)
	result.localAmount = payBill.TotalAmount
	result.thirdAmount = payBill.ThirdTotalAmount
	result.thirdCount = payBill.ThirdTotalCount
	return result, nil
}

// formatMessage 格式化交易账单核对结果。
func (r tradeBillCheckResult) formatMessage() string {
	return fmt.Sprintf(
		"%s核对完成: 本地 %d 条，三方 %d 条，匹配成功 %d 条，匹配失败 %d 条，三方未匹配 %d 条，本地金额 %d 分，三方金额 %d 分，结果 %s",
		r.name,
		r.localCount,
		r.thirdCount,
		r.successCount,
		r.failedCount,
		r.unmatchedCount,
		r.localAmount,
		r.thirdAmount,
		r.resultName(),
	)
}

// resultName 返回交易账单本轮核对结果名称。
func (r tradeBillCheckResult) resultName() string {
	if r.localCount == r.thirdCount && r.localAmount == r.thirdAmount && r.failedCount == 0 && r.unmatchedCount == 0 {
		return "无误差"
	}
	return "有误差"
}

// billDateRange 计算账单日期对应的时间范围
func billDateRange(billDate string) (time.Time, time.Time, error) {
	startTime, err := time.ParseInLocation("2006-01-02", billDate, time.Local)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	return startTime, startTime.AddDate(0, 0, 1), nil
}

// parseOrderPaymentAmount 解析支付金额 JSON。
func parseOrderPaymentAmount(rawAmount string) (*shopadminv1.OrderPayment_Amount, error) {
	var orderPaymentAmount shopadminv1.OrderPayment_Amount
	err := json.Unmarshal([]byte(rawAmount), &orderPaymentAmount)
	if err != nil {
		return nil, errorsx.Internal("解析支付账单金额失败").WithCause(err)
	}
	return &orderPaymentAmount, nil
}

// parseOrderRefundAmount 解析退款金额 JSON。
func parseOrderRefundAmount(rawAmount string) (*shopadminv1.OrderRefund_Amount, error) {
	var orderRefundAmount shopadminv1.OrderRefund_Amount
	err := json.Unmarshal([]byte(rawAmount), &orderRefundAmount)
	if err != nil {
		return nil, errorsx.Internal("解析退款账单金额失败").WithCause(err)
	}
	return &orderRefundAmount, nil
}
