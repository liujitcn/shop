package biz

import (
	"context"
	"errors"
	"fmt"
	"shop/service/admin/wx"
	"strconv"
	"time"

	"shop/api/gen/go/admin"
	"shop/api/gen/go/common"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	_time "github.com/liujitcn/go-utils/time"
	"github.com/liujitcn/go-utils/trans"
	"github.com/liujitcn/gorm-kit/repo"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/jsapi"
	"github.com/wechatpay-apiv3/wechatpay-go/services/refunddomestic"
	"gorm.io/gorm"
)

// OrderCase 订单业务实例
type OrderCase struct {
	*biz.BaseCase
	tx data.Transaction
	*data.OrderRepo
	orderAddressCase   *OrderAddressCase
	orderCancelCase    *OrderCancelCase
	orderGoodsCase     *OrderGoodsCase
	orderLogisticsCase *OrderLogisticsCase
	orderPaymentCase   *OrderPaymentCase
	orderRefundCase    *OrderRefundCase
	baseUserCase       *BaseUserCase
	baseDictItemCase   *BaseDictItemCase
	wxPayCase          *wx.WxPayCase
	mapper             *mapper.CopierMapper[admin.Order, models.Order]
}

// NewOrderCase 创建订单业务实例
func NewOrderCase(baseCase *biz.BaseCase, tx data.Transaction, orderAddressCase *OrderAddressCase, orderRepo *data.OrderRepo, orderCancelCase *OrderCancelCase, orderGoodsCase *OrderGoodsCase, orderLogisticsCase *OrderLogisticsCase, orderPaymentCase *OrderPaymentCase, orderRefundCase *OrderRefundCase, baseUserCase *BaseUserCase, baseDictItemCase *BaseDictItemCase, wxPayCase *wx.WxPayCase) *OrderCase {
	return &OrderCase{
		BaseCase:           baseCase,
		tx:                 tx,
		OrderRepo:          orderRepo,
		orderAddressCase:   orderAddressCase,
		orderCancelCase:    orderCancelCase,
		orderGoodsCase:     orderGoodsCase,
		orderLogisticsCase: orderLogisticsCase,
		orderPaymentCase:   orderPaymentCase,
		orderRefundCase:    orderRefundCase,
		baseUserCase:       baseUserCase,
		baseDictItemCase:   baseDictItemCase,
		wxPayCase:          wxPayCase,
		mapper:             mapper.NewCopierMapper[admin.Order, models.Order](),
	}
}

// PageOrder 分页查询订单
func (c *OrderCase) PageOrder(ctx context.Context, req *admin.PageOrderRequest) (*admin.PageOrderResponse, error) {
	query := c.Query(ctx).Order
	opts := make([]repo.QueryOption, 0, 7)
	opts = append(opts, repo.Order(query.UpdatedAt.Desc()))
	if req.GetUserId() > 0 {
		opts = append(opts, repo.Where(query.UserID.Eq(req.GetUserId())))
	}
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
	if len(req.GetCreatedAt()) == 2 {
		startTime := _time.StringTimeToTime(req.GetCreatedAt()[0])
		endTime := _time.StringTimeToTime(req.GetCreatedAt()[1])
		if startTime != nil {
			opts = append(opts, repo.Where(query.CreatedAt.Gte(*startTime)))
		}
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

	resList := make([]*admin.Order, 0, len(list))
	for _, item := range list {
		order := c.mapper.ToDTO(item)
		if user, ok := userMap[item.UserID]; ok {
			order.NickName = user.NickName
		}
		resList = append(resList, order)
	}
	return &admin.PageOrderResponse{List: resList, Total: int32(total)}, nil
}

// GetOrder 获取订单
func (c *OrderCase) GetOrder(ctx context.Context, id int64) (*admin.OrderResponse, error) {
	order, err := c.FindById(ctx, id)
	if err != nil {
		return nil, err
	}

	res := &admin.OrderResponse{
		Order:     c.mapper.ToDTO(order),
		Countdown: float32((order.CreatedAt.Add(30 * time.Minute)).Sub(time.Now()).Seconds()),
	}

	var baseUser *models.BaseUser
	baseUser, err = c.baseUserCase.FindById(ctx, order.UserID)
	if err == nil {
		res.Order.NickName = baseUser.NickName
	}

	res.Address, err = c.orderAddressCase.FindFromByOrderId(ctx, order.ID)
	if err != nil {
		return nil, err
	}
	res.Cancel, err = c.orderCancelCase.FindFromByOrderId(ctx, order.ID)
	if err != nil {
		return nil, err
	}
	res.Goods, err = c.orderGoodsCase.FindFromByOrderId(ctx, order.ID)
	if err != nil {
		return nil, err
	}
	res.Logistics, err = c.orderLogisticsCase.FindFromByOrderId(ctx, order.ID)
	if err != nil {
		return nil, err
	}
	res.Payment, err = c.orderPaymentCase.FindFromByOrderId(ctx, order.ID)
	if err != nil {
		return nil, err
	}
	res.Refund, err = c.orderRefundCase.FindFromByOrderId(ctx, order.ID)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// GetOrderRefund 获取订单退款信息
func (c *OrderCase) GetOrderRefund(ctx context.Context, id int64) (*admin.OrderRefundResponse, error) {
	order, err := c.FindById(ctx, id)
	if err != nil {
		return nil, err
	}
	res := &admin.OrderRefundResponse{}
	res.Payment, err = c.orderPaymentCase.FindFromByOrderId(ctx, order.ID)
	if err != nil {
		return nil, err
	}
	res.Refund, err = c.orderRefundCase.FindFromByOrderId(ctx, order.ID)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// RefundOrder 退款订单
func (c *OrderCase) RefundOrder(ctx context.Context, req *admin.RefundOrderRequest) error {
	order, err := c.FindById(ctx, req.GetOrderId())
	if err != nil {
		return err
	}
	if !(order.Status == int32(common.OrderStatus_SHIPPED) || order.Status == int32(common.OrderStatus_RECEIVED) || order.Status == int32(common.OrderStatus_REFUNDING)) {
		return fmt.Errorf("订单状态错误：【%s】", common.OrderStatus_name[order.Status])
	}

	orderRefund := &models.OrderRefund{
		OrderID:  req.GetOrderId(),
		RefundNo: strconv.FormatInt(time.Now().UnixNano(), 10),
		Reason:   int32(req.GetReason()),
	}

	if common.OrderPayType(order.PayType) == common.OrderPayType_ONLINE_PAY && common.OrderPayChannel(order.PayChannel) == common.OrderPayChannel_WX_PAY {
		reason := common.OrderRefundReason_name[int32(req.GetReason())]
		var refund *refunddomestic.Refund
		refund, err = c.wxPayCase.Refund(refunddomestic.CreateRequest{
			OutTradeNo:  trans.String(order.OrderNo),
			OutRefundNo: trans.String(orderRefund.RefundNo),
			Reason:      trans.String(reason),
			Amount: &refunddomestic.AmountReq{
				Total:    trans.Int64(order.PayMoney),
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
		if refund.Channel != nil && refund.Channel.Ptr() != nil {
			orderRefund.Channel = string(*refund.Channel.Ptr())
		}
		orderRefund.UserReceivedAccount = trans.StringValue(refund.UserReceivedAccount)
		orderRefund.CreateTime = trans.TimeValue(refund.CreateTime)
		orderRefund.SuccessTime = trans.TimeValue(refund.SuccessTime)
		if refund.Status != nil && refund.Status.Ptr() != nil {
			orderRefund.RefundState = string(*refund.Status.Ptr())
		}
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
		return c.UpdateById(ctx, &models.Order{
			ID:     order.ID,
			Status: int32(common.OrderStatus_REFUNDING),
		})
	})
}

// GetOrderShipped 获取订单发货信息
func (c *OrderCase) GetOrderShipped(ctx context.Context, id int64) (*admin.OrderShippedResponse, error) {
	order, err := c.FindById(ctx, id)
	if err != nil {
		return nil, err
	}

	res := &admin.OrderShippedResponse{}
	res.Address, err = c.orderAddressCase.FindFromByOrderId(ctx, order.ID)
	if err != nil {
		return nil, err
	}
	res.Goods, err = c.orderGoodsCase.FindFromByOrderId(ctx, order.ID)
	if err != nil {
		return nil, err
	}
	if order.Status == int32(common.OrderStatus_SHIPPED) || order.Status == int32(common.OrderStatus_RECEIVED) {
		res.Logistics, err = c.orderLogisticsCase.FindFromByOrderId(ctx, order.ID)
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

// ShippedOrder 发货订单
func (c *OrderCase) ShippedOrder(ctx context.Context, req *admin.ShippedOrderRequest) error {
	order, err := c.FindById(ctx, req.GetOrderId())
	if err != nil {
		return err
	}
	if order.Status != int32(common.OrderStatus_PAID) {
		return fmt.Errorf("订单状态错误：【%s】", common.OrderStatus_name[order.Status])
	}

	if common.OrderPayType(order.PayType) == common.OrderPayType_ONLINE_PAY && common.OrderPayChannel(order.PayChannel) == common.OrderPayChannel_WX_PAY {
		var transaction *payments.Transaction
		transaction, err = c.wxPayCase.QueryOrderByOutTradeNo(jsapi.QueryOrderByOutTradeNoRequest{
			OutTradeNo: trans.String(order.OrderNo),
		})
		if err != nil {
			return err
		}

		tradeState := trans.StringValue(transaction.TradeState)
		if tradeState != "SUCCESS" {
			return fmt.Errorf("订单状态错误：【%s】", tradeState)
		}

		paymentQuery := c.orderPaymentCase.Query(ctx).OrderPayment
		var orderPayment *models.OrderPayment
		orderPayment, err = c.orderPaymentCase.Find(ctx, repo.Where(paymentQuery.OrderID.Eq(order.ID)))
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				orderPayment = &models.OrderPayment{}
			} else {
				return err
			}
		}

		successTime := _time.StringDateToTime(transaction.SuccessTime)
		if successTime == nil {
			now := time.Now()
			successTime = &now
		}
		orderPayment.OrderID = order.ID
		orderPayment.OrderNo = trans.StringValue(transaction.OutTradeNo)
		orderPayment.ThirdOrderNo = trans.StringValue(transaction.TransactionId)
		orderPayment.TradeType = trans.StringValue(transaction.TradeType)
		orderPayment.TradeState = tradeState
		orderPayment.TradeStateDesc = trans.StringValue(transaction.TradeStateDesc)
		orderPayment.BankType = trans.StringValue(transaction.BankType)
		orderPayment.SuccessTime = *successTime
		orderPayment.Payer = _string.ConvertAnyToJsonString(transaction.Payer)
		orderPayment.Amount = _string.ConvertAnyToJsonString(transaction.Amount)
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
			OrderID: order.ID,
			Name:    req.GetName(),
			No:      req.GetNo(),
			Contact: req.GetContact(),
			Detail:  "[]",
		})
		if err != nil {
			return err
		}
		return c.UpdateById(ctx, &models.Order{
			ID:     order.ID,
			Status: int32(common.OrderStatus_SHIPPED),
		})
	})
}

// getOrderUserMap 查询订单用户映射
func (c *OrderCase) getOrderUserMap(ctx context.Context, list []*models.Order) (map[int64]*models.BaseUser, error) {
	userIds := make([]int64, 0, len(list))
	for _, item := range list {
		userIds = append(userIds, item.UserID)
	}
	userMap := make(map[int64]*models.BaseUser)
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
