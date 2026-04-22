package biz

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"shop/api/gen/go/admin"
	appApi "shop/api/gen/go/app"
	"shop/api/gen/go/common"
	"shop/pkg/biz"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	"shop/pkg/wx"

	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	_time "github.com/liujitcn/go-utils/time"
	"github.com/liujitcn/go-utils/trans"
	"github.com/liujitcn/gorm-kit/repo"
	"github.com/wechatpay-apiv3/wechatpay-go/services/refunddomestic"
	"gorm.io/gorm"
)

// OrderInfoCase 订单业务实例
type OrderInfoCase struct {
	*biz.BaseCase
	tx data.Transaction
	*data.OrderInfoRepo
	orderAddressCase   *OrderAddressCase
	orderCancelCase    *OrderCancelCase
	orderGoodsCase     *OrderGoodsCase
	orderLogisticsCase *OrderLogisticsCase
	orderPaymentCase   *OrderPaymentCase
	orderRefundCase    *OrderRefundCase
	baseUserCase       *BaseUserCase
	baseDictItemCase   *BaseDictItemCase
	wxPayCase          *wx.WxPayCase
	mapper             *mapper.CopierMapper[admin.OrderInfo, models.OrderInfo]
}

// NewOrderInfoCase 创建订单业务实例
func NewOrderInfoCase(baseCase *biz.BaseCase, tx data.Transaction, orderAddressCase *OrderAddressCase, orderInfoRepo *data.OrderInfoRepo, orderCancelCase *OrderCancelCase, orderGoodsCase *OrderGoodsCase, orderLogisticsCase *OrderLogisticsCase, orderPaymentCase *OrderPaymentCase, orderRefundCase *OrderRefundCase, baseUserCase *BaseUserCase, baseDictItemCase *BaseDictItemCase, wxPayCase *wx.WxPayCase) *OrderInfoCase {
	return &OrderInfoCase{
		BaseCase:           baseCase,
		tx:                 tx,
		OrderInfoRepo:      orderInfoRepo,
		orderAddressCase:   orderAddressCase,
		orderCancelCase:    orderCancelCase,
		orderGoodsCase:     orderGoodsCase,
		orderLogisticsCase: orderLogisticsCase,
		orderPaymentCase:   orderPaymentCase,
		orderRefundCase:    orderRefundCase,
		baseUserCase:       baseUserCase,
		baseDictItemCase:   baseDictItemCase,
		wxPayCase:          wxPayCase,
		mapper:             mapper.NewCopierMapper[admin.OrderInfo, models.OrderInfo](),
	}
}

// PageOrderInfo 分页查询订单
func (c *OrderInfoCase) PageOrderInfo(ctx context.Context, req *admin.PageOrderInfoRequest) (*admin.PageOrderInfoResponse, error) {
	query := c.Query(ctx).OrderInfo
	opts := make([]repo.QueryOption, 0, 7)
	opts = append(opts, repo.Order(query.CreatedAt.Desc()))
	// 传入用户编号时，按用户过滤订单。
	if req.GetUserId() > 0 {
		opts = append(opts, repo.Where(query.UserID.Eq(req.GetUserId())))
	}
	// 传入订单号关键字时，按订单号模糊匹配。
	if req.GetOrderNo() != "" {
		opts = append(opts, repo.Where(query.OrderNo.Like("%"+req.GetOrderNo()+"%")))
	}
	if req.Status != nil {
		opts = append(opts, repo.Where(query.Status.Eq(int32(req.GetStatus()))))
	}
	if req.PayType != nil {
		opts = append(opts, repo.Where(query.PayType.Eq(int32(req.GetPayType()))))
	}
	if req.PayChannel != nil {
		opts = append(opts, repo.Where(query.PayChannel.Eq(int32(req.GetPayChannel()))))
	}
	// 传入时间范围时，按创建时间区间过滤订单。
	if len(req.GetCreatedAt()) == 2 {
		startTime := _time.StringTimeToTime(req.GetCreatedAt()[0])
		endTime := _time.StringTimeToTime(req.GetCreatedAt()[1])
		// 开始时间可解析时，追加起始时间条件。
		if startTime != nil {
			opts = append(opts, repo.Where(query.CreatedAt.Gte(*startTime)))
		}
		// 结束时间可解析时，追加结束时间条件。
		if endTime != nil {
			endAt := endTime.Add(24 * time.Hour)
			opts = append(opts, repo.Where(query.CreatedAt.Lt(endAt)))
		}
	}

	list, total, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}

	var userMap map[int64]*models.BaseUser
	userMap, err = c.getOrderUserMap(ctx, list)
	if err != nil {
		return nil, err
	}

	resList := make([]*admin.OrderInfo, 0, len(list))
	for _, item := range list {
		orderInfo := c.mapper.ToDTO(item)
		// 命中用户信息时，补齐下单用户昵称。
		if user, ok := userMap[item.UserID]; ok {
			orderInfo.NickName = user.NickName
		}
		resList = append(resList, orderInfo)
	}
	return &admin.PageOrderInfoResponse{List: resList, Total: int32(total)}, nil
}

// GetOrderInfo 获取订单
func (c *OrderInfoCase) GetOrderInfo(ctx context.Context, id int64) (*admin.OrderInfoResponse, error) {
	orderInfo, err := c.FindById(ctx, id)
	if err != nil {
		return nil, err
	}

	res := &admin.OrderInfoResponse{
		Order:     c.mapper.ToDTO(orderInfo),
		Countdown: float32((orderInfo.CreatedAt.Add(30 * time.Minute)).Sub(time.Now()).Seconds()),
	}

	var baseUser *models.BaseUser
	baseUser, err = c.baseUserCase.FindById(ctx, orderInfo.UserID)
	// 用户存在时，补齐订单上的下单用户昵称。
	if err == nil {
		res.Order.NickName = baseUser.NickName
	}

	res.Address, err = c.orderAddressCase.FindFromByOrderId(ctx, orderInfo.ID)
	if err != nil {
		return nil, err
	}
	res.Cancel, err = c.orderCancelCase.FindFromByOrderId(ctx, orderInfo.ID)
	if err != nil {
		return nil, err
	}
	res.Goods, err = c.orderGoodsCase.FindFromByOrderId(ctx, orderInfo.ID)
	if err != nil {
		return nil, err
	}
	res.Logistics, err = c.orderLogisticsCase.FindFromByOrderId(ctx, orderInfo.ID)
	if err != nil {
		return nil, err
	}
	res.Payment, err = c.orderPaymentCase.FindFromByOrderId(ctx, orderInfo.ID)
	if err != nil {
		return nil, err
	}
	res.Refund, err = c.orderRefundCase.FindFromByOrderId(ctx, orderInfo.ID)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// GetOrderInfoRefund 获取订单退款信息
func (c *OrderInfoCase) GetOrderInfoRefund(ctx context.Context, id int64) (*admin.OrderInfoRefundResponse, error) {
	orderInfo, err := c.FindById(ctx, id)
	if err != nil {
		return nil, err
	}
	res := &admin.OrderInfoRefundResponse{}
	res.Payment, err = c.orderPaymentCase.FindFromByOrderId(ctx, orderInfo.ID)
	if err != nil {
		return nil, err
	}
	res.Refund, err = c.orderRefundCase.FindFromByOrderId(ctx, orderInfo.ID)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// RefundOrderInfo 退款订单
func (c *OrderInfoCase) RefundOrderInfo(ctx context.Context, req *admin.RefundOrderInfoRequest) error {
	orderInfo, err := c.FindById(ctx, req.GetOrderId())
	if err != nil {
		return err
	}
	// 只有待收货或已完成订单才允许后台发起退款，已退款订单禁止再次退款。
	if !(orderInfo.Status == int32(common.OrderStatus_SHIPPED) || orderInfo.Status == int32(common.OrderStatus_RECEIVED)) {
		return errorsx.StateConflict(
			fmt.Sprintf("订单状态错误：【%s】", common.OrderStatus_name[orderInfo.Status]),
			"order_info",
			common.OrderStatus(orderInfo.Status).String(),
			"SHIPPED|RECEIVED",
		)
	}

	orderRefund := &models.OrderRefund{
		OrderID:  req.GetOrderId(),
		RefundNo: strconv.FormatInt(time.Now().UnixNano(), 10),
		Reason:   int32(req.GetReason()),
	}

	// 微信在线支付订单需要先走微信退款单创建流程。
	if common.OrderPayType(orderInfo.PayType) == common.OrderPayType_ONLINE_PAY && common.OrderPayChannel(orderInfo.PayChannel) == common.OrderPayChannel_WX_PAY {
		reason := common.OrderRefundReason_name[int32(req.GetReason())]
		var refund *refunddomestic.Refund
		refund, err = c.wxPayCase.Refund(refunddomestic.CreateRequest{
			OutTradeNo:  trans.String(orderInfo.OrderNo),
			OutRefundNo: trans.String(orderRefund.RefundNo),
			Reason:      trans.String(reason),
			Amount: &refunddomestic.AmountReq{
				Total:    trans.Int64(orderInfo.PayMoney),
				Refund:   trans.Int64(req.GetRefundMoney()),
				Currency: trans.String("CNY"),
			},
		})
		if err != nil {
			return err
		}
		orderRefund.OrderNo = trans.StringValue(refund.OutTradeNo)
		orderRefund.ThirdOrderNo = trans.StringValue(refund.TransactionId)
		orderRefund.ThirdRefundNo = trans.StringValue(refund.RefundId)
		// 微信返回退款渠道时，补齐退款渠道字段。
		if refund.Channel != nil && refund.Channel.Ptr() != nil {
			orderRefund.Channel = string(*refund.Channel.Ptr())
		}
		orderRefund.UserReceivedAccount = trans.StringValue(refund.UserReceivedAccount)
		orderRefund.CreateTime = trans.TimeValue(refund.CreateTime)
		orderRefund.SuccessTime = trans.TimeValue(refund.SuccessTime)
		// 微信返回退款状态时，补齐退款状态字段。
		if refund.Status != nil && refund.Status.Ptr() != nil {
			orderRefund.RefundState = string(*refund.Status.Ptr())
		}
		// 微信返回资金账户时，补齐资金账户字段。
		if refund.FundsAccount != nil && refund.FundsAccount.Ptr() != nil {
			orderRefund.FundsAccount = string(*refund.FundsAccount.Ptr())
		}
		orderRefund.Amount = _string.ConvertAnyToJsonString(refund.Amount)
		orderRefund.Status = int32(common.OrderBillStatus_NO_CHECK)
	} else {
		now := time.Now()
		orderRefund.CreateTime = now
		orderRefund.SuccessTime = now
		orderRefund.Amount = "{}"
	}

	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		err = c.orderRefundCase.Create(ctx, orderRefund)
		if err != nil {
			return err
		}
		return c.UpdateById(ctx, &models.OrderInfo{
			ID:     orderInfo.ID,
			Status: int32(common.OrderStatus_REFUNDING),
		})
	})
}

// GetOrderInfoShipment 获取订单发货信息
func (c *OrderInfoCase) GetOrderInfoShipment(ctx context.Context, id int64) (*admin.OrderInfoShipmentForm, error) {
	orderInfo, err := c.FindById(ctx, id)
	if err != nil {
		return nil, err
	}

	res := &admin.OrderInfoShipmentForm{}
	res.Address, err = c.orderAddressCase.FindFromByOrderId(ctx, orderInfo.ID)
	if err != nil {
		return nil, err
	}
	res.Goods, err = c.orderGoodsCase.FindFromByOrderId(ctx, orderInfo.ID)
	if err != nil {
		return nil, err
	}
	// 已发货或已收货订单需要补充物流信息。
	if orderInfo.Status == int32(common.OrderStatus_SHIPPED) || orderInfo.Status == int32(common.OrderStatus_RECEIVED) {
		res.Logistics, err = c.orderLogisticsCase.FindFromByOrderId(ctx, orderInfo.ID)
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

// ShipOrderInfo 发货订单
func (c *OrderInfoCase) ShipOrderInfo(ctx context.Context, req *admin.ShipOrderInfoRequest) error {
	orderInfo, err := c.FindById(ctx, req.GetOrderId())
	if err != nil {
		return err
	}
	// 只有已支付订单才能继续发货。
	if orderInfo.Status != int32(common.OrderStatus_PAID) {
		return errorsx.StateConflict(
			fmt.Sprintf("订单状态错误：【%s】", common.OrderStatus_name[orderInfo.Status]),
			"order_info",
			common.OrderStatus(orderInfo.Status).String(),
			common.OrderStatus_PAID.String(),
		)
	}

	// 微信支付订单在发货前需要再次核验支付状态，避免未支付订单被误发货。
	if common.OrderPayType(orderInfo.PayType) == common.OrderPayType_ONLINE_PAY && common.OrderPayChannel(orderInfo.PayChannel) == common.OrderPayChannel_WX_PAY {
		var paymentResource *appApi.PaymentResource
		paymentResource, err = c.wxPayCase.QueryOrderByOutTradeNo(orderInfo.OrderNo)
		if err != nil {
			return err
		}

		// 只有微信侧明确返回支付成功，才允许继续同步支付单并发货。
		if paymentResource.GetTradeState() != appApi.PaymentResource_SUCCESS {
			return errorsx.StateConflict(
				fmt.Sprintf("订单状态错误：【%s】", paymentResource.GetTradeState().String()),
				"order_payment",
				paymentResource.GetTradeState().String(),
				appApi.PaymentResource_SUCCESS.String(),
			)
		}

		query := c.orderPaymentCase.Query(ctx).OrderPayment
		opts := make([]repo.QueryOption, 0, 1)
		opts = append(opts, repo.Where(query.OrderID.Eq(orderInfo.ID)))
		var orderPayment *models.OrderPayment
		orderPayment, err = c.orderPaymentCase.Find(ctx, opts...)
		// 支付记录查询失败时，仅对“未找到”场景回退空对象。
		if err != nil {
			// 支付单不存在时按首次补录处理，其余查询异常直接返回。
			if errors.Is(err, gorm.ErrRecordNotFound) {
				orderPayment = &models.OrderPayment{}
			} else {
				return err
			}
		}

		successTime := paymentResource.GetSuccessTime()
		// 微信未返回成功时间时，使用当前时间兜底，避免后续入库空值。
		if successTime == nil {
			now := time.Now()
			orderPayment.SuccessTime = now
		} else {
			orderPayment.SuccessTime = successTime.AsTime()
		}
		orderPayment.OrderID = orderInfo.ID
		orderPayment.OrderNo = paymentResource.GetOutTradeNo()
		orderPayment.ThirdOrderNo = paymentResource.GetTransactionId()
		orderPayment.TradeType = paymentResource.GetTradeType().String()
		orderPayment.TradeState = paymentResource.GetTradeState().String()
		orderPayment.TradeStateDesc = paymentResource.GetTradeStateDesc()
		orderPayment.BankType = paymentResource.GetBankType()
		orderPayment.Payer = _string.ConvertAnyToJsonString(paymentResource.GetPayer())
		orderPayment.Amount = _string.ConvertAnyToJsonString(paymentResource.GetAmount())
		// 首次发货前补录支付单时走创建，已有记录则直接覆盖同步微信最新状态。
		if orderPayment.ID == 0 {
			err = c.orderPaymentCase.Create(ctx, orderPayment)
		} else {
			err = c.orderPaymentCase.UpdateById(ctx, orderPayment)
		}
		if err != nil {
			return err
		}
	}

	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		err = c.orderLogisticsCase.Create(ctx, &models.OrderLogistics{
			OrderID: orderInfo.ID,
			Name:    req.GetName(),
			No:      req.GetNo(),
			Contact: req.GetContact(),
			Detail:  "[]",
		})
		if err != nil {
			return err
		}
		return c.UpdateById(ctx, &models.OrderInfo{
			ID:     orderInfo.ID,
			Status: int32(common.OrderStatus_SHIPPED),
		})
	})
}

// getOrderUserMap 查询订单用户映射
func (c *OrderInfoCase) getOrderUserMap(ctx context.Context, list []*models.OrderInfo) (map[int64]*models.BaseUser, error) {
	userIds := make([]int64, 0, len(list))
	for _, item := range list {
		userIds = append(userIds, item.UserID)
	}
	userMap := make(map[int64]*models.BaseUser)
	// 订单列表为空时，直接返回空用户映射。
	if len(userIds) == 0 {
		return userMap, nil
	}

	userList, err := c.baseUserCase.ListByIds(ctx, userIds)
	if err != nil {
		return nil, err
	}
	for _, item := range userList {
		userMap[item.ID] = item
	}
	return userMap, nil
}
