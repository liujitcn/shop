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

	resList := make([]*admin.OrderInfo, 0, len(list))
	for _, item := range list {
		orderInfo := c.mapper.ToDTO(item)
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
	if !(orderInfo.Status == int32(common.OrderStatus_SHIPPED) || orderInfo.Status == int32(common.OrderStatus_RECEIVED) || orderInfo.Status == int32(common.OrderStatus_REFUNDING)) {
		return fmt.Errorf("订单状态错误：【%s】", common.OrderStatus_name[orderInfo.Status])
	}

	orderRefund := &models.OrderRefund{
		OrderID:  req.GetOrderId(),
		RefundNo: strconv.FormatInt(time.Now().UnixNano(), 10),
		Reason:   int32(req.GetReason()),
	}

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
		return c.UpdateById(ctx, &models.OrderInfo{
			ID:     orderInfo.ID,
			Status: int32(common.OrderStatus_REFUNDING),
		})
	})
}

// GetOrderInfoShipped 获取订单发货信息
func (c *OrderInfoCase) GetOrderInfoShipped(ctx context.Context, id int64) (*admin.OrderInfoShippedResponse, error) {
	orderInfo, err := c.FindById(ctx, id)
	if err != nil {
		return nil, err
	}

	res := &admin.OrderInfoShippedResponse{}
	res.Address, err = c.orderAddressCase.FindFromByOrderId(ctx, orderInfo.ID)
	if err != nil {
		return nil, err
	}
	res.Goods, err = c.orderGoodsCase.FindFromByOrderId(ctx, orderInfo.ID)
	if err != nil {
		return nil, err
	}
	if orderInfo.Status == int32(common.OrderStatus_SHIPPED) || orderInfo.Status == int32(common.OrderStatus_RECEIVED) {
		res.Logistics, err = c.orderLogisticsCase.FindFromByOrderId(ctx, orderInfo.ID)
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

// ShippedOrderInfo 发货订单
func (c *OrderInfoCase) ShippedOrderInfo(ctx context.Context, req *admin.ShippedOrderInfoRequest) error {
	orderInfo, err := c.FindById(ctx, req.GetOrderId())
	if err != nil {
		return err
	}
	if orderInfo.Status != int32(common.OrderStatus_PAID) {
		return fmt.Errorf("订单状态错误：【%s】", common.OrderStatus_name[orderInfo.Status])
	}

	if common.OrderPayType(orderInfo.PayType) == common.OrderPayType_ONLINE_PAY && common.OrderPayChannel(orderInfo.PayChannel) == common.OrderPayChannel_WX_PAY {
		var transaction *payments.Transaction
		transaction, err = c.wxPayCase.QueryOrderByOutTradeNo(jsapi.QueryOrderByOutTradeNoRequest{
			OutTradeNo: trans.String(orderInfo.OrderNo),
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
		orderPayment, err = c.orderPaymentCase.Find(ctx, repo.Where(paymentQuery.OrderID.Eq(orderInfo.ID)))
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
		orderPayment.OrderID = orderInfo.ID
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
