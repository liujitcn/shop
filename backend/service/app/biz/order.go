package biz

import (
	"context"
	"errors"
	"fmt"
	"shop/pkg/configs"
	"strconv"
	"time"

	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"shop/api/gen/go/app"
	"shop/api/gen/go/common"
	"shop/service/app/util"
	"shop/service/app/wx"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/liujitcn/go-utils/id"
	_string "github.com/liujitcn/go-utils/string"
	_time "github.com/liujitcn/go-utils/time"
	"github.com/liujitcn/go-utils/trans"
	"github.com/liujitcn/gorm-kit/repo"
	wxPayCore "github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/services/refunddomestic"
)

const orderRefundReason string = "order_refund_reason" // 退款原因

// OrderCase 订单业务处理对象
type OrderCase struct {
	*biz.BaseCase
	tx data.Transaction
	*data.OrderRepo
	orderCancelCase    *OrderCancelCase
	orderGoodsCase     *OrderGoodsCase
	orderAddressCase   *OrderAddressCase
	orderLogisticsCase *OrderLogisticsCase
	orderPaymentCase   *OrderPaymentCase
	orderRefundCase    *OrderRefundCase
	goodsCase          *GoodsCase
	goodsSkuCase       *GoodsSkuCase
	userAddressCase    *UserAddressCase
	userCartCase       *UserCartCase
	baseDictItemCase   *BaseDictItemCase
	orderSchedulerCase *OrderSchedulerCase
	payCase            *PayCase
	wxPayCase          *wx.WxPayCase
}

// NewOrderCase 创建订单业务处理对象
func NewOrderCase(
	baseCase *biz.BaseCase,
	tx data.Transaction,
	orderRepo *data.OrderRepo,
	orderCancelCase *OrderCancelCase,
	orderGoodsCase *OrderGoodsCase,
	orderAddressCase *OrderAddressCase,
	orderLogisticsCase *OrderLogisticsCase,
	orderPaymentCase *OrderPaymentCase,
	orderRefundCase *OrderRefundCase,
	goodsCase *GoodsCase,
	goodsSkuCase *GoodsSkuCase,
	userAddressCase *UserAddressCase,
	userCartCase *UserCartCase,
	baseDictItemCase *BaseDictItemCase,
	orderSchedulerCase *OrderSchedulerCase,
	payCase *PayCase,
	wxPayCase *wx.WxPayCase,
) (*OrderCase, error) {
	c := &OrderCase{
		BaseCase:           baseCase,
		tx:                 tx,
		OrderRepo:          orderRepo,
		orderCancelCase:    orderCancelCase,
		orderGoodsCase:     orderGoodsCase,
		orderAddressCase:   orderAddressCase,
		orderLogisticsCase: orderLogisticsCase,
		orderPaymentCase:   orderPaymentCase,
		orderRefundCase:    orderRefundCase,
		goodsCase:          goodsCase,
		goodsSkuCase:       goodsSkuCase,
		userAddressCase:    userAddressCase,
		userCartCase:       userCartCase,
		baseDictItemCase:   baseDictItemCase,
		orderSchedulerCase: orderSchedulerCase,
		payCase:            payCase,
		wxPayCase:          wxPayCase,
	}

	// 服务启动时恢复全部未支付订单的超时取消任务
	orderQuery := c.Query(context.Background()).Order
	list, err := c.List(context.Background(),
		repo.Where(orderQuery.Status.Eq(int32(common.OrderStatus_CREATED))),
	)
	if err != nil {
		return nil, err
	}
	payTimeout := configs.ParsePayTimeout()
	for _, item := range list {
		// 计算当前订单距离支付超时还剩余多少秒
		createdAt := item.CreatedAt.Add(payTimeout)
		nowTime := time.Now()
		countdown := createdAt.Sub(nowTime).Seconds()
		if countdown < 0 {
			// 自动取消订单
			err = c.cancelOrder(context.Background(), item.UserID, &app.CancelOrderRequest{
				OrderId: item.ID,
			})
			if err != nil {
				log.Errorf("CancelOrder order %d failed: %v", item.ID, err)
			}
		} else {
			// 添加自动取消定时任务
			c.orderSchedulerCase.AddSchedule(item.ID, time.Duration(countdown)*time.Second, func() {
				err = c.cancelOrder(context.Background(), item.UserID, &app.CancelOrderRequest{
					OrderId: item.ID,
				})
				if err != nil {
					log.Errorf("CancelOrder order %d failed: %v", item.ID, err)
				}
			})
		}
	}

	return c, nil
}

// OrderPre 预付订单
func (c *OrderCase) OrderPre(ctx context.Context) (*app.ConfirmOrderResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	member := util.IsMemberByAuthInfo(authInfo)

	// 查询购物车列表
	userCartQuery := c.userCartCase.Query(ctx).UserCart
	userCartList := make([]*models.UserCart, 0)
	userCartList, err = c.userCartCase.List(ctx,
		repo.Where(userCartQuery.UserID.Eq(authInfo.UserId)),
		repo.Where(userCartQuery.IsChecked.Is(true)),
	)
	if err != nil {
		return nil, err
	}
	createOrderGoods := make([]*app.CreateOrderGoods, 0)
	for _, item := range userCartList {
		createOrderGoods = append(createOrderGoods, &app.CreateOrderGoods{
			GoodsId: item.GoodsID,
			SkuCode: item.SkuCode,
			Num:     item.Num,
		})
	}
	return c.orderBuy(ctx, member, createOrderGoods)
}

// OrderBuy 立即购买订单
func (c *OrderCase) OrderBuy(ctx context.Context, req *app.CreateOrderGoods) (*app.ConfirmOrderResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	member := util.IsMemberByAuthInfo(authInfo)

	// 将单个商品请求封装成统一的下单明细列表
	createOrderGoods := []*app.CreateOrderGoods{req}
	return c.orderBuy(ctx, member, createOrderGoods)
}

// OrderRepurchase 再次购买订单
func (c *OrderCase) OrderRepurchase(ctx context.Context, req *app.OrderRepurchaseRequest) (*app.ConfirmOrderResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	member := util.IsMemberByAuthInfo(authInfo)
	var order *models.Order
	order, err = c.findByUserIdAndId(ctx, authInfo.UserId, req.GetOrderId())
	if err != nil {
		return nil, err
	}
	// 读取原订单中的商品明细，重新构造成下单请求
	orderGoodsQuery := c.orderGoodsCase.Query(ctx).OrderGoods
	var oldOrderGoods []*models.OrderGoods
	oldOrderGoods, err = c.orderGoodsCase.List(ctx,
		repo.Where(orderGoodsQuery.OrderID.Eq(order.ID)),
	)
	if err != nil {
		return nil, err
	}
	createOrderGoods := make([]*app.CreateOrderGoods, 0)
	for _, item := range oldOrderGoods {
		createOrderGoods = append(createOrderGoods, &app.CreateOrderGoods{
			GoodsId: item.GoodsID,
			SkuCode: item.SkuCode,
			Num:     item.Num,
		})
	}
	return c.orderBuy(ctx, member, createOrderGoods)
}

// CountOrder 查询订单数量汇总
func (c *OrderCase) CountOrder(ctx context.Context) (*app.CountOrderResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	query := c.Query(ctx).Order
	list, err := c.List(ctx,
		repo.Where(query.UserID.Eq(authInfo.UserId)),
	)
	if err != nil {
		return nil, err
	}
	res := make(map[int32]int32)
	for _, item := range list {
		res[item.Status]++
	}
	count := make([]*app.CountOrderResponse_Count, 0)
	for k, v := range res {
		count = append(count, &app.CountOrderResponse_Count{
			Status: common.OrderStatus(k),
			Num:    v,
		})
	}
	return &app.CountOrderResponse{
		Count: count,
	}, nil
}

// PageOrder 查询订单分页列表
func (c *OrderCase) PageOrder(ctx context.Context, req *app.PageOrderRequest) (*app.PageOrderResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	orderQuery := c.Query(ctx).Order
	opts := make([]repo.QueryOption, 0, 3)
	opts = append(opts, repo.Order(orderQuery.CreatedAt.Desc()))
	opts = append(opts, repo.Where(orderQuery.UserID.Eq(authInfo.UserId)))
	if req.GetStatus() != common.OrderStatus_UNKNOWN_OS {
		opts = append(opts, repo.Where(orderQuery.Status.Eq(int32(req.GetStatus()))))
	}
	var page []*models.Order
	var count int64
	page, count, err = c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}

	orderIds := make([]int64, 0)
	for _, item := range page {
		orderIds = append(orderIds, item.ID)
	}

	orderGoodsMap := make(map[int64][]*app.OrderGoods)
	orderGoodsMap, err = c.orderGoodsCase.mapByOrderIds(ctx, orderIds)
	if err != nil {
		return nil, err
	}

	list := make([]*app.Order, 0)
	for _, item := range page {
		order := c.convertToProto(item)
		if v, ok := orderGoodsMap[order.Id]; ok {
			order.Goods = v
		}
		list = append(list, order)
	}

	return &app.PageOrderResponse{
		List:  list,
		Total: int32(count),
	}, nil
}

// GetOrderIdByOrderNo 按订单号查询订单编号
func (c *OrderCase) GetOrderIdByOrderNo(ctx context.Context, orderNo string) (int64, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return 0, err
	}

	query := c.Query(ctx).Order
	var item *models.Order
	item, err = c.Find(ctx,
		repo.Where(query.OrderNo.Eq(orderNo)),
		repo.Where(query.UserID.Eq(authInfo.UserId)),
	)
	if err != nil {
		return 0, err
	}
	return item.ID, nil
}

// GetOrderById 根据订单编号查询订单
func (c *OrderCase) GetOrderById(ctx context.Context, id int64) (*app.OrderResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}

	var item *models.Order
	item, err = c.findByUserIdAndId(ctx, authInfo.UserId, id)
	if err != nil {
		return nil, err
	}
	order := c.convertToProto(item)
	// 只有待支付订单才需要返回支付剩余时间
	payTimeout := configs.ParsePayTimeout()
	createdAt := item.CreatedAt.Add(payTimeout)
	nowTime := time.Now()
	countdown := createdAt.Sub(nowTime).Seconds()

	// 查询订单商品明细
	order.Goods, err = c.orderGoodsCase.listByOrderId(ctx, order.Id)
	if err != nil {
		return nil, err
	}
	// 查询订单收货地址快照
	var address *app.OrderResponse_Address
	address, err = c.orderAddressCase.findByOrderId(ctx, order.Id)
	if err != nil {
		return nil, err
	}

	res := app.OrderResponse{
		Order:     order,
		Address:   address,
		Countdown: float32(countdown),
	}

	// 根据订单状态补充额外展示字段
	switch common.OrderStatus(item.Status) {
	case common.OrderStatus_PAID:
		// 待支付订单返回支付倒计时
		if common.OrderPayType(item.PayType) == common.OrderPayType_ONLINE_PAY {
			res.Order.PaymentTime, err = c.orderPaymentCase.findPaymentTimeByOrderId(ctx, order.Id)
			if err != nil {
				return nil, err
			}
		}
	case common.OrderStatus_SHIPPED, common.OrderStatus_RECEIVED:
		// 已发货订单返回物流信息
		var logistics *app.OrderResponse_Logistics
		logistics, err = c.orderLogisticsCase.findByOrderId(ctx, order.Id)
		if err != nil {
			return nil, err
		}
		res.Logistics = logistics
	case common.OrderStatus_CANCELED:
		// 已取消订单返回取消时间
		res.Order.CancelTime, err = c.orderCancelCase.findCancelTimeByOrderId(ctx, order.Id)
		if err != nil {
			return nil, err
		}
	case common.OrderStatus_REFUNDING:
		// 退款订单返回退款时间
		res.Order.RefundTime, err = c.orderRefundCase.findRefundTimeByOrderId(ctx, order.Id)
		if err != nil {
			return nil, err
		}
	}
	return &res, nil
}

// CreateOrder 创建订单并发起支付准备
func (c *OrderCase) CreateOrder(ctx context.Context, request *app.CreateOrderRequest) (*app.CreateOrderResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	status := int32(common.OrderStatus_CREATED)
	// 货到付款订单创建后直接进入待发货状态
	if request.PayType == common.OrderPayType_CASH_ON_DELIVERY {
		status = int32(common.OrderStatus_PAID)
	}

	// 先构建订单基础信息，再在事务中统一落库
	order := &models.Order{
		OrderNo:      strconv.FormatInt(id.GenSnowflakeID(), 10),
		UserID:       authInfo.UserId,
		PayType:      int32(request.PayType),
		PayChannel:   int32(request.PayChannel),
		DeliveryTime: int32(request.DeliveryTime),
		Status:       status,
		Remark:       request.Remark,
	}
	err = c.tx.Transaction(ctx, func(ctx context.Context) error {
		var orderGoodsList []*models.OrderGoods
		orderGoodsList, err = c.orderGoodsCase.convertToModelList(ctx, request.GetGoods())
		if err != nil {
			return err
		}
		for _, orderGoods := range orderGoodsList {
			order.PayMoney += orderGoods.TotalPayPrice
			order.TotalMoney += orderGoods.TotalPrice
			order.GoodsNum += orderGoods.Num
			// 创建订单后立即扣减库存并增加销量
			err = c.goodsCase.addSaleNum(ctx, orderGoods.GoodsID, orderGoods.Num)
			if err != nil {
				return err
			}
			err = c.goodsSkuCase.addSaleNum(ctx, orderGoods.SkuCode, orderGoods.Num)
			if err != nil {
				return err
			}
			// 从购物车下单的商品需要在下单成功后移出购物车
			err = c.userCartCase.deleteByUserIdAndGoodsIdAndSkuCode(ctx, authInfo.UserId, orderGoods.GoodsID, orderGoods.SkuCode)
			if err != nil {
				return err
			}
		}
		// 当前版本统一免运费
		order.PostFee = 0
		err = c.OrderRepo.Create(ctx, order)
		if err != nil {
			return err
		}

		// 保存订单商品快照
		err = c.orderGoodsCase.createByOrder(ctx, order.ID, orderGoodsList)
		if err != nil {
			return err
		}
		// 保存订单地址快照
		err = c.orderAddressCase.createByOrder(ctx, authInfo.UserId, order.ID, request.GetAddressId())
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	// 为在线支付订单增加超时自动取消任务
	if order.Status == int32(common.OrderStatus_CREATED) {
		// 延迟时间使用支付超时配置
		payTimeout := configs.ParsePayTimeout()
		createdAt := order.CreatedAt.Add(payTimeout)
		nowTime := time.Now()
		countdown := createdAt.Sub(nowTime).Seconds()
		c.orderSchedulerCase.AddSchedule(order.ID, time.Duration(countdown)*time.Second, func() {
			err = c.cancelOrder(context.Background(), order.UserID, &app.CancelOrderRequest{
				OrderId: order.ID,
			})
			if err != nil {
				log.Errorf("CancelOrder order %d failed: %v", order.ID, err)
			}
		})
	}
	return &app.CreateOrderResponse{
		OrderId: order.ID,
	}, nil
}

// DeleteOrder 删除订单
func (c *OrderCase) DeleteOrder(ctx context.Context, id int64) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}

	var order *models.Order
	order, err = c.findByUserIdAndId(ctx, authInfo.UserId, id)
	if err != nil {
		return err
	}
	if !(order.Status == int32(common.OrderStatus_RECEIVED) || order.Status == int32(common.OrderStatus_REFUNDING) || order.Status == int32(common.OrderStatus_CANCELED)) {
		return fmt.Errorf("订单状态错误：【%s】", common.OrderStatus_name[order.Status])
	}

	orderIds := []int64{id}
	return c.updateByIds(ctx, authInfo.UserId, orderIds, &models.Order{
		Status: int32(common.OrderStatus_DELETED),
	})
}

// CancelOrder 取消订单并回退库存销量
func (c *OrderCase) CancelOrder(ctx context.Context, req *app.CancelOrderRequest) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	return c.cancelOrder(ctx, authInfo.UserId, req)
}

// RefundOrder 申请订单退款
func (c *OrderCase) RefundOrder(ctx context.Context, req *app.RefundOrderRequest) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	var order *models.Order
	order, err = c.findByUserIdAndId(ctx, authInfo.UserId, req.GetOrderId())
	if err != nil {
		return err
	}

	if order.Status != int32(common.OrderStatus_PAID) {
		return fmt.Errorf("订单状态错误：【%s】", common.OrderStatus_name[order.Status])
	}

	orderRefund := &models.OrderRefund{
		OrderID:  req.GetOrderId(),
		RefundNo: order.OrderNo, // 退款单号，和订单号使用一个方便查询退款
		Reason:   int32(req.GetReason()),
	}
	// 只有在线支付订单才会走退款单和微信退款流程
	if common.OrderPayType(order.PayType) == common.OrderPayType_ONLINE_PAY {
		// 先把退款原因值翻译成字典标签，便于退款记录展示
		reason := strconv.Itoa(int(orderRefund.Reason))
		var label string
		label, err = c.baseDictItemCase.findLabelByCodeAndValue(ctx, orderRefundReason, reason)
		if err == nil {
			reason = label
		}
		// 仅微信支付订单需要调用微信退款接口
		if common.OrderPayChannel(order.PayChannel) == common.OrderPayChannel_WX_PAY {
			var refund *refunddomestic.Refund
			refund, err = c.wxPayCase.Refund(refunddomestic.CreateRequest{
				OutTradeNo:  trans.String(order.OrderNo),
				OutRefundNo: trans.String(orderRefund.RefundNo),
				Reason:      trans.String(reason),
				Amount: &refunddomestic.AmountReq{
					Total:    trans.Int64(order.PayMoney),
					Refund:   trans.Int64(order.PayMoney),
					Currency: trans.String("CNY"),
				},
			})
			if err != nil {
				if apiErr, ok := errors.AsType[*wxPayCore.APIError](err); ok {
					// 订单已支付
					if apiErr.Code == "INVALID_REQUEST" && apiErr.Message == "订单已全额退款" {
						// 调用查询退款接口
						var refundResource *app.RefundResource
						refundResource, err = c.wxPayCase.QueryByOutRefundNo(orderRefund.RefundNo)
						if err != nil {
							return err
						}
						err = c.payCase.RefundSuccess(ctx, order, refundResource)
						if err != nil {
							return err
						}
						return errors.New("订单已退款，不能重复退款")
					}
				}
				return err
			}
			orderRefund.OrderNo = trans.StringValue(refund.OutTradeNo)
			orderRefund.ThirdOrderNo = trans.StringValue(refund.TransactionId)
			orderRefund.ThirdRefundNo = trans.StringValue(refund.RefundId)
			orderRefund.Channel = string(*refund.Channel.Ptr())
			orderRefund.UserReceivedAccount = trans.StringValue(refund.UserReceivedAccount)
			orderRefund.CreateTime = trans.TimeValue(refund.CreateTime)
			orderRefund.SuccessTime = trans.TimeValue(refund.SuccessTime)
			orderRefund.RefundState = string(*refund.Status.Ptr())
			orderRefund.FundsAccount = string(*refund.FundsAccount)
			orderRefund.Amount = _string.ConvertAnyToJsonString(refund.Amount)
		}
	} else {
		t := time.Now()
		orderRefund.CreateTime = t
		orderRefund.SuccessTime = t
		orderRefund.Amount = "{}"
	}
	orderIds := []int64{req.GetOrderId()}
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		// 退款成功后保存退款记录
		err = c.orderRefundCase.Create(ctx, orderRefund)
		if err != nil {
			return err
		}
		return c.updateByIds(ctx, authInfo.UserId, orderIds, &models.Order{
			Status: int32(common.OrderStatus_REFUNDING),
		})
	})
}

// ReceiveOrder 确认收货
func (c *OrderCase) ReceiveOrder(ctx context.Context, req *app.ReceiveOrderRequest) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}

	var order *models.Order
	order, err = c.findByUserIdAndId(ctx, authInfo.UserId, req.GetOrderId())
	if err != nil {
		return err
	}
	if order.Status != int32(common.OrderStatus_SHIPPED) {
		return fmt.Errorf("订单状态错误：【%s】", common.OrderStatus_name[order.Status])
	}

	orderIds := []int64{req.GetOrderId()}
	return c.updateByIds(ctx, authInfo.UserId, orderIds, &models.Order{
		Status: int32(common.OrderStatus_RECEIVED),
	})
}

// 将订单模型转换为接口响应
func (c *OrderCase) convertToProto(item *models.Order) *app.Order {
	res := &app.Order{
		Id:           item.ID,
		OrderNo:      item.OrderNo,
		PayMoney:     item.PayMoney,
		TotalMoney:   item.TotalMoney,
		PostFee:      item.PostFee,
		GoodsNum:     item.GoodsNum,
		PayType:      common.OrderPayType(item.PayType),
		PayChannel:   common.OrderPayChannel(item.PayChannel),
		DeliveryTime: common.OrderDeliveryTime(item.DeliveryTime),
		Status:       common.OrderStatus(item.Status),
		Remark:       item.Remark,
		CreatedAt:    _time.TimeToTimeString(item.CreatedAt),
		UpdatedAt:    _time.TimeToTimeString(item.UpdatedAt),
	}
	return res
}

// 汇总下单商品信息并生成确认单
func (c *OrderCase) orderBuy(ctx context.Context, member bool, createOrderGoods []*app.CreateOrderGoods) (*app.ConfirmOrderResponse, error) {
	newOrderGoods := make([]*app.OrderGoods, 0)
	for _, item := range createOrderGoods {
		newGoods, err := c.orderGoodsCase.convertToProtoByCreateOrderGoods(ctx, member, item)
		if err != nil {
			return nil, err
		}
		newOrderGoods = append(newOrderGoods, newGoods)
	}

	var summary app.OrderSummary
	for _, orderGoods := range newOrderGoods {
		summary.PayMoney += orderGoods.TotalPayPrice
		summary.TotalMoney += orderGoods.TotalPrice
		summary.GoodsNum += orderGoods.Num
	}
	// 当前版本统一免运费
	summary.PostFee = 0
	return &app.ConfirmOrderResponse{
		Goods:   newOrderGoods,
		Summary: &summary,
	}, nil
}

// cancelOrder 内部执行订单取消并回退库存销量
func (c *OrderCase) cancelOrder(ctx context.Context, userId int64, req *app.CancelOrderRequest) error {
	order, err := c.findByUserIdAndId(ctx, userId, req.GetOrderId())
	if err != nil {
		return err
	}
	if order.Status != int32(common.OrderStatus_CREATED) {
		return fmt.Errorf("订单状态错误：【%s】", common.OrderStatus_name[order.Status])
	}

	// 微信在线支付订单在取消前先补查一次真实支付状态，避免回调延迟时误取消已支付订单
	if common.OrderPayType(order.PayType) == common.OrderPayType_ONLINE_PAY &&
		common.OrderPayChannel(order.PayChannel) == common.OrderPayChannel_WX_PAY {
		var paymentResource *app.PaymentResource
		paymentResource, err = c.wxPayCase.QueryOrderByOutTradeNo(order.OrderNo)
		if err != nil {
			return err
		}
		if paymentResource != nil && paymentResource.GetTradeState().String() == "SUCCESS" {
			// 查询到已支付后，直接复用支付服务中的成功处理逻辑补齐本地状态
			err = c.payCase.PaySuccess(ctx, order, paymentResource)
			if err != nil {
				return err
			}
			return errors.New("订单已支付，无法取消")
		}
	}
	orderIds := []int64{req.GetOrderId()}
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		orderGoodsQuery := c.orderGoodsCase.Query(ctx).OrderGoods
		var orderGoodsList []*models.OrderGoods
		orderGoodsList, err = c.orderGoodsCase.List(ctx,
			repo.Where(orderGoodsQuery.OrderID.In(orderIds...)),
		)
		for _, orderGoods := range orderGoodsList {
			// 订单取消后恢复库存并回退销量
			err = c.goodsCase.subSaleNum(ctx, orderGoods.GoodsID, orderGoods.Num)
			if err != nil {
				return err
			}
			err = c.goodsSkuCase.subSaleNum(ctx, orderGoods.SkuCode, orderGoods.Num)
			if err != nil {
				return err
			}
		}
		// 保存订单取消记录，便于订单详情展示取消时间
		err = c.orderCancelCase.Create(ctx, &models.OrderCancel{
			OrderID: req.GetOrderId(),
			Reason:  int32(req.GetReason()),
		})
		if err != nil {
			return err
		}
		return c.updateByIds(ctx, userId, orderIds, &models.Order{
			Status: int32(common.OrderStatus_CANCELED),
		})
	})
}

// 按订单编号和用户编号查询订单
func (c *OrderCase) findByUserIdAndId(ctx context.Context, userId, orderId int64) (*models.Order, error) {
	query := c.Query(ctx).Order
	return c.Find(ctx,
		repo.Where(query.ID.Eq(orderId)),
		repo.Where(query.UserID.Eq(userId)),
	)
}

// 按订单编号批量更新当前用户的订单
func (c *OrderCase) updateByIds(ctx context.Context, userId int64, ids []int64, entity *models.Order) error {
	if len(ids) == 0 {
		return nil
	}
	query := c.Query(ctx).Order
	return c.Update(ctx, entity,
		repo.Where(query.ID.In(ids...)),
		repo.Where(query.UserID.Eq(userId)),
	)
}
