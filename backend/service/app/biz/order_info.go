package biz

import (
	"context"
	"errors"
	"fmt"
	"shop/pkg/configs"
	"strconv"
	"time"

	"shop/pkg/biz"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"shop/api/gen/go/app"
	"shop/api/gen/go/common"
	pkgQueue "shop/pkg/queue"
	"shop/pkg/wx"
	"shop/service/app/utils"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/liujitcn/go-utils/id"
	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	_time "github.com/liujitcn/go-utils/time"
	"github.com/liujitcn/go-utils/trans"
	"github.com/liujitcn/gorm-kit/repo"
	wxPayCore "github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/services/refunddomestic"
)

const orderRefundReason string = "order_refund_reason" // 退款原因

// OrderInfoCase 订单业务处理对象
type OrderInfoCase struct {
	*biz.BaseCase
	tx data.Transaction
	*data.OrderInfoRepo
	orderCancelCase    *OrderCancelCase
	orderGoodsCase     *OrderGoodsCase
	orderAddressCase   *OrderAddressCase
	orderLogisticsCase *OrderLogisticsCase
	orderPaymentCase   *OrderPaymentCase
	orderRefundCase    *OrderRefundCase
	goodsInfoCase      *GoodsInfoCase
	goodsSkuCase       *GoodsSkuCase
	userAddressCase    *UserAddressCase
	userCartCase       *UserCartCase
	baseDictItemCase   *BaseDictItemCase
	orderSchedulerCase *OrderSchedulerCase
	payCase            *PayCase
	wxPayCase          *wx.WxPayCase
	mapper             *mapper.CopierMapper[app.OrderInfo, models.OrderInfo]
}

// NewOrderInfoCase 创建订单业务处理对象
func NewOrderInfoCase(
	baseCase *biz.BaseCase,
	tx data.Transaction,
	orderInfoRepo *data.OrderInfoRepo,
	orderCancelCase *OrderCancelCase,
	orderGoodsCase *OrderGoodsCase,
	orderAddressCase *OrderAddressCase,
	orderLogisticsCase *OrderLogisticsCase,
	orderPaymentCase *OrderPaymentCase,
	orderRefundCase *OrderRefundCase,
	goodsInfoCase *GoodsInfoCase,
	goodsSkuCase *GoodsSkuCase,
	userAddressCase *UserAddressCase,
	userCartCase *UserCartCase,
	baseDictItemCase *BaseDictItemCase,
	orderSchedulerCase *OrderSchedulerCase,
	payCase *PayCase,
	wxPayCase *wx.WxPayCase,
) (*OrderInfoCase, error) {
	c := &OrderInfoCase{
		BaseCase:           baseCase,
		tx:                 tx,
		OrderInfoRepo:      orderInfoRepo,
		orderCancelCase:    orderCancelCase,
		orderGoodsCase:     orderGoodsCase,
		orderAddressCase:   orderAddressCase,
		orderLogisticsCase: orderLogisticsCase,
		orderPaymentCase:   orderPaymentCase,
		orderRefundCase:    orderRefundCase,
		goodsInfoCase:      goodsInfoCase,
		goodsSkuCase:       goodsSkuCase,
		userAddressCase:    userAddressCase,
		userCartCase:       userCartCase,
		baseDictItemCase:   baseDictItemCase,
		orderSchedulerCase: orderSchedulerCase,
		payCase:            payCase,
		wxPayCase:          wxPayCase,
		mapper:             mapper.NewCopierMapper[app.OrderInfo, models.OrderInfo](),
	}

	// 服务启动时恢复全部未支付订单的超时取消任务
	query := c.Query(context.Background()).OrderInfo
	opts := make([]repo.QueryOption, 0, 1)
	opts = append(opts, repo.Where(query.Status.Eq(int32(common.OrderStatus_CREATED))))
	list, err := c.List(context.Background(), opts...)
	if err != nil {
		return nil, err
	}
	payTimeout := configs.ParsePayTimeout()
	for _, item := range list {
		// 计算当前订单距离支付超时还剩余多少秒
		createdAt := item.CreatedAt.Add(payTimeout)
		nowTime := time.Now()
		countdown := createdAt.Sub(nowTime).Seconds()
		// 已超出支付时限的订单立即执行取消。
		if countdown < 0 {
			// 自动取消订单
			err = c.cancelOrder(context.Background(), item.UserID, &app.CancelOrderInfoRequest{
				OrderId: item.ID,
			})
			// 自动取消执行失败时，仅记录日志继续恢复其余任务。
			if err != nil {
				log.Errorf("CancelOrder order %d failed: %v", item.ID, err)
			}
		} else {
			// 添加自动取消定时任务
			c.orderSchedulerCase.AddSchedule(item.ID, time.Duration(countdown)*time.Second, func() {
				err = c.cancelOrder(context.Background(), item.UserID, &app.CancelOrderInfoRequest{
					OrderId: item.ID,
				})
				// 定时任务取消失败时，仅记录日志避免影响调度器运行。
				if err != nil {
					log.Errorf("CancelOrder order %d failed: %v", item.ID, err)
				}
			})
		}
	}

	return c, nil
}

// ConfirmOrderInfo 确认订单
func (c *OrderInfoCase) ConfirmOrderInfo(ctx context.Context) (*app.ConfirmOrderInfoResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	member := utils.IsMemberByAuthInfo(authInfo)

	// 查询购物车列表
	query := c.userCartCase.Query(ctx).UserCart
	userCartList := make([]*models.UserCart, 0)
	opts := make([]repo.QueryOption, 0, 2)
	opts = append(opts, repo.Where(query.UserID.Eq(authInfo.UserId)))
	opts = append(opts, repo.Where(query.IsChecked.Is(true)))
	userCartList, err = c.userCartCase.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	createOrderGoods := make([]*app.CreateOrderInfoGoods, 0)
	for _, item := range userCartList {
		createOrderGoods = append(createOrderGoods, &app.CreateOrderInfoGoods{
			GoodsId: item.GoodsID,
			SkuCode: item.SkuCode,
			Num:     item.Num,
			RecommendContext: &app.RecommendContext{
				Scene:     common.RecommendScene(item.Scene),
				RequestId: item.RequestID,
				Position:  item.Position,
			},
		})
	}
	return c.orderBuy(ctx, member, createOrderGoods)
}

// BuyNowOrderInfo 立即购买订单
func (c *OrderInfoCase) BuyNowOrderInfo(ctx context.Context, req *app.BuyNowOrderInfoRequest) (*app.BuyNowOrderInfoResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	member := utils.IsMemberByAuthInfo(authInfo)

	// 将单个商品请求封装成统一的下单明细列表
	createOrderGoods := []*app.CreateOrderInfoGoods{{
		GoodsId:          req.GetGoodsId(),
		SkuCode:          req.GetSkuCode(),
		Num:              req.GetNum(),
		RecommendContext: req.GetRecommendContext(),
	}}
	res, err := c.orderBuy(ctx, member, createOrderGoods)
	if err != nil {
		return nil, err
	}
	return &app.BuyNowOrderInfoResponse{
		Goods:     res.GetGoods(),
		Summary:   res.GetSummary(),
		ClearCart: res.GetClearCart(),
	}, nil
}

// RepurchaseOrderInfo 再次购买订单
func (c *OrderInfoCase) RepurchaseOrderInfo(ctx context.Context, req *app.RepurchaseOrderInfoRequest) (*app.RepurchaseOrderInfoResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	member := utils.IsMemberByAuthInfo(authInfo)
	var orderInfo *models.OrderInfo
	orderInfo, err = c.findByUserIdAndId(ctx, authInfo.UserId, req.GetOrderId())
	if err != nil {
		return nil, err
	}
	// 读取原订单中的商品明细，重新构造成下单请求
	query := c.orderGoodsCase.Query(ctx).OrderGoods
	var oldOrderGoods []*models.OrderGoods
	opts := make([]repo.QueryOption, 0, 1)
	opts = append(opts, repo.Where(query.OrderID.Eq(orderInfo.ID)))
	oldOrderGoods, err = c.orderGoodsCase.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	createOrderGoods := make([]*app.CreateOrderInfoGoods, 0)
	for _, item := range oldOrderGoods {
		createOrderGoods = append(createOrderGoods, &app.CreateOrderInfoGoods{
			GoodsId: item.GoodsID,
			SkuCode: item.SkuCode,
			Num:     item.Num,
			RecommendContext: &app.RecommendContext{
				Scene:     common.RecommendScene(item.Scene),
				RequestId: item.RequestID,
				Position:  item.Position,
			},
		})
	}
	res, err := c.orderBuy(ctx, member, createOrderGoods)
	if err != nil {
		return nil, err
	}
	return &app.RepurchaseOrderInfoResponse{
		Goods:     res.GetGoods(),
		Summary:   res.GetSummary(),
		ClearCart: res.GetClearCart(),
	}, nil
}

// CountOrderInfo 查询订单数量汇总
func (c *OrderInfoCase) CountOrderInfo(ctx context.Context) (*app.CountOrderInfoResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	query := c.Query(ctx).OrderInfo
	opts := make([]repo.QueryOption, 0, 1)
	opts = append(opts, repo.Where(query.UserID.Eq(authInfo.UserId)))
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	res := make(map[int32]int32)
	for _, item := range list {
		res[item.Status]++
	}
	count := make([]*app.CountOrderInfoResponse_Count, 0)
	for k, v := range res {
		count = append(count, &app.CountOrderInfoResponse_Count{
			Status: common.OrderStatus(k),
			Num:    v,
		})
	}
	return &app.CountOrderInfoResponse{
		Count: count,
	}, nil
}

// PageOrderInfo 查询订单分页列表
func (c *OrderInfoCase) PageOrderInfo(ctx context.Context, req *app.PageOrderInfoRequest) (*app.PageOrderInfoResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	query := c.Query(ctx).OrderInfo
	opts := make([]repo.QueryOption, 0, 4)
	opts = append(opts, repo.Order(query.CreatedAt.Desc()))
	opts = append(opts, repo.Where(query.UserID.Eq(authInfo.UserId)))
	// 指定订单状态时，只返回该状态下的订单记录。
	if req.GetStatus() != common.OrderStatus_UNKNOWN_OS {
		opts = append(opts, repo.Where(query.Status.Eq(int32(req.GetStatus()))))
	}
	var page []*models.OrderInfo
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

	list := make([]*app.OrderInfo, 0)
	for _, item := range page {
		orderInfo := c.convertToProto(item)
		// 命中订单商品映射时，补齐当前订单的商品列表。
		if v, ok := orderGoodsMap[orderInfo.Id]; ok {
			orderInfo.Goods = v
		}
		list = append(list, orderInfo)
	}

	return &app.PageOrderInfoResponse{
		List:  list,
		Total: int32(count),
	}, nil
}

// GetOrderInfoIdByOrderNo 按订单号查询订单编号
func (c *OrderInfoCase) GetOrderInfoIdByOrderNo(ctx context.Context, orderNo string) (int64, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return 0, err
	}

	query := c.Query(ctx).OrderInfo
	var item *models.OrderInfo
	opts := make([]repo.QueryOption, 0, 2)
	opts = append(opts, repo.Where(query.OrderNo.Eq(orderNo)))
	opts = append(opts, repo.Where(query.UserID.Eq(authInfo.UserId)))
	item, err = c.Find(ctx, opts...)
	if err != nil {
		return 0, err
	}
	return item.ID, nil
}

// GetOrderInfoById 根据订单编号查询订单
func (c *OrderInfoCase) GetOrderInfoById(ctx context.Context, id int64) (*app.OrderInfoResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}

	var item *models.OrderInfo
	item, err = c.findByUserIdAndId(ctx, authInfo.UserId, id)
	if err != nil {
		return nil, err
	}
	orderInfo := c.convertToProto(item)
	// 只有待支付订单才需要返回支付剩余时间
	payTimeout := configs.ParsePayTimeout()
	createdAt := item.CreatedAt.Add(payTimeout)
	nowTime := time.Now()
	countdown := createdAt.Sub(nowTime).Seconds()

	// 查询订单商品明细
	orderInfo.Goods, err = c.orderGoodsCase.listByOrderId(ctx, orderInfo.Id)
	if err != nil {
		return nil, err
	}
	// 查询订单收货地址快照
	var address *app.OrderInfoResponse_Address
	address, err = c.orderAddressCase.findByOrderId(ctx, orderInfo.Id)
	if err != nil {
		return nil, err
	}

	res := app.OrderInfoResponse{
		Order:     orderInfo,
		Address:   address,
		Countdown: float32(countdown),
	}

	// 根据订单状态补充额外展示字段
	switch common.OrderStatus(item.Status) {
	case common.OrderStatus_PAID:
		// 在线支付且已支付的订单需要补充支付完成时间。
		if common.OrderPayType(item.PayType) == common.OrderPayType_ONLINE_PAY {
			res.Order.PaymentTime, err = c.orderPaymentCase.findPaymentTimeByOrderId(ctx, orderInfo.Id)
			if err != nil {
				return nil, err
			}
		}
	case common.OrderStatus_SHIPPED, common.OrderStatus_RECEIVED:
		// 已发货订单返回物流信息
		var logistics *app.OrderInfoResponse_Logistics
		logistics, err = c.orderLogisticsCase.findByOrderId(ctx, orderInfo.Id)
		if err != nil {
			return nil, err
		}
		res.Logistics = logistics
	case common.OrderStatus_CANCELED:
		// 已取消订单返回取消时间
		res.Order.CancelTime, err = c.orderCancelCase.findCancelTimeByOrderId(ctx, orderInfo.Id)
		if err != nil {
			return nil, err
		}
	case common.OrderStatus_REFUNDING:
		// 退款订单返回退款时间
		res.Order.RefundTime, err = c.orderRefundCase.findRefundTimeByOrderId(ctx, orderInfo.Id)
		if err != nil {
			return nil, err
		}
	}
	return &res, nil
}

// CreateOrderInfo 创建订单并发起支付准备
func (c *OrderInfoCase) CreateOrderInfo(ctx context.Context, request *app.CreateOrderInfoRequest) (*app.CreateOrderInfoResponse, error) {
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
	orderInfo := &models.OrderInfo{
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
			orderInfo.PayMoney += orderGoods.TotalPayPrice
			orderInfo.TotalMoney += orderGoods.TotalPrice
			orderInfo.GoodsNum += orderGoods.Num
			// 创建订单后立即扣减库存并增加销量
			err = c.goodsInfoCase.addSaleNum(ctx, orderGoods.GoodsID, orderGoods.Num)
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
		orderInfo.PostFee = 0
		err = c.OrderInfoRepo.Create(ctx, orderInfo)
		if err != nil {
			return err
		}

		// 保存订单商品快照
		err = c.orderGoodsCase.createByOrder(ctx, orderInfo.ID, orderGoodsList)
		if err != nil {
			return err
		}
		// 保存订单地址快照
		err = c.orderAddressCase.createByOrder(ctx, authInfo.UserId, orderInfo.ID, request.GetAddressId())
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	c.dispatchRecommendOrderEvent(request.PayType, authInfo.UserId, request.GetGoods(), orderInfo.CreatedAt)
	// 为在线支付订单增加超时自动取消任务
	if orderInfo.Status == int32(common.OrderStatus_CREATED) {
		// 延迟时间使用支付超时配置
		payTimeout := configs.ParsePayTimeout()
		createdAt := orderInfo.CreatedAt.Add(payTimeout)
		nowTime := time.Now()
		countdown := createdAt.Sub(nowTime).Seconds()
		c.orderSchedulerCase.AddSchedule(orderInfo.ID, time.Duration(countdown)*time.Second, func() {
			err = c.cancelOrder(context.Background(), orderInfo.UserID, &app.CancelOrderInfoRequest{
				OrderId: orderInfo.ID,
			})
			// 定时取消执行失败时，仅记录日志避免影响后续调度。
			if err != nil {
				log.Errorf("CancelOrder order %d failed: %v", orderInfo.ID, err)
			}
		})
	}
	return &app.CreateOrderInfoResponse{
		OrderId: orderInfo.ID,
	}, nil
}

// DeleteOrderInfo 删除订单
func (c *OrderInfoCase) DeleteOrderInfo(ctx context.Context, id int64) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}

	var orderInfo *models.OrderInfo
	orderInfo, err = c.findByUserIdAndId(ctx, authInfo.UserId, id)
	if err != nil {
		return err
	}
	// 只有已收货、退款中或已取消订单允许删除。
	if !(orderInfo.Status == int32(common.OrderStatus_RECEIVED) || orderInfo.Status == int32(common.OrderStatus_REFUNDING) || orderInfo.Status == int32(common.OrderStatus_CANCELED)) {
		return errorsx.StateConflict(
			fmt.Sprintf("订单状态错误：【%s】", common.OrderStatus_name[orderInfo.Status]),
			"order_info",
			common.OrderStatus(orderInfo.Status).String(),
			"RECEIVED|REFUNDING|CANCELED",
		)
	}

	orderIds := []int64{id}
	return c.updateByIds(ctx, authInfo.UserId, orderIds, &models.OrderInfo{
		Status: int32(common.OrderStatus_DELETED),
	})
}

// CancelOrderInfo 取消订单并回退库存销量
func (c *OrderInfoCase) CancelOrderInfo(ctx context.Context, req *app.CancelOrderInfoRequest) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	return c.cancelOrder(ctx, authInfo.UserId, req)
}

// RefundOrderInfo 申请订单退款
func (c *OrderInfoCase) RefundOrderInfo(ctx context.Context, req *app.RefundOrderInfoRequest) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	var orderInfo *models.OrderInfo
	orderInfo, err = c.findByUserIdAndId(ctx, authInfo.UserId, req.GetOrderId())
	if err != nil {
		return err
	}

	// 只有已支付订单才能继续申请退款。
	if orderInfo.Status != int32(common.OrderStatus_PAID) {
		return errorsx.StateConflict(
			fmt.Sprintf("订单状态错误：【%s】", common.OrderStatus_name[orderInfo.Status]),
			"order_info",
			common.OrderStatus(orderInfo.Status).String(),
			common.OrderStatus_PAID.String(),
		)
	}

	orderRefund := &models.OrderRefund{
		OrderID:  req.GetOrderId(),
		RefundNo: orderInfo.OrderNo, // 退款单号，和订单号使用一个方便查询退款
		Reason:   int32(req.GetReason()),
	}
	// 只有在线支付订单才会走退款单和微信退款流程
	if common.OrderPayType(orderInfo.PayType) == common.OrderPayType_ONLINE_PAY {
		// 先把退款原因值翻译成字典标签，便于退款记录展示
		reason := strconv.Itoa(int(orderRefund.Reason))
		var label string
		label, err = c.baseDictItemCase.findLabelByCodeAndValue(ctx, orderRefundReason, reason)
		// 字典标签查询成功时，使用标签替换原始原因值。
		if err == nil {
			reason = label
		}
		// 仅微信支付订单需要调用微信退款接口
		if common.OrderPayChannel(orderInfo.PayChannel) == common.OrderPayChannel_WX_PAY {
			var refund *refunddomestic.Refund
			refund, err = c.wxPayCase.Refund(refunddomestic.CreateRequest{
				OutTradeNo:  trans.String(orderInfo.OrderNo),
				OutRefundNo: trans.String(orderRefund.RefundNo),
				Reason:      trans.String(reason),
				Amount: &refunddomestic.AmountReq{
					Total:    trans.Int64(orderInfo.PayMoney),
					Refund:   trans.Int64(orderInfo.PayMoney),
					Currency: trans.String("CNY"),
				},
			})
			// 微信退款创建失败时，需要识别“已全额退款”的可恢复场景。
			if err != nil {
				// 命中微信 API 错误结构时，再判断是否属于幂等退款场景。
				if apiErr, ok := errors.AsType[*wxPayCore.APIError](err); ok {
					// 微信明确返回“订单已全额退款”时，补查退款结果并同步本地状态。
					if apiErr.Code == "INVALID_REQUEST" && apiErr.Message == "订单已全额退款" {
						// 调用查询退款接口
						var refundResource *app.RefundResource
						refundResource, err = c.wxPayCase.QueryByOutRefundNo(orderRefund.RefundNo)
						if err != nil {
							return err
						}
						err = c.payCase.RefundSuccess(ctx, orderInfo, refundResource)
						if err != nil {
							return err
						}
						return errorsx.StateConflict(
							"订单已退款，不能重复退款",
							"order_refund",
							app.RefundResource_SUCCESS.String(),
							"NONE",
						)
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
		return c.updateByIds(ctx, authInfo.UserId, orderIds, &models.OrderInfo{
			Status: int32(common.OrderStatus_REFUNDING),
		})
	})
}

// ReceiveOrderInfo 确认收货
func (c *OrderInfoCase) ReceiveOrderInfo(ctx context.Context, req *app.ReceiveOrderInfoRequest) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}

	var orderInfo *models.OrderInfo
	orderInfo, err = c.findByUserIdAndId(ctx, authInfo.UserId, req.GetOrderId())
	if err != nil {
		return err
	}
	// 只有已发货订单才能确认收货。
	if orderInfo.Status != int32(common.OrderStatus_SHIPPED) {
		return errorsx.StateConflict(
			fmt.Sprintf("订单状态错误：【%s】", common.OrderStatus_name[orderInfo.Status]),
			"order_info",
			common.OrderStatus(orderInfo.Status).String(),
			common.OrderStatus_SHIPPED.String(),
		)
	}

	orderIds := []int64{req.GetOrderId()}
	return c.updateByIds(ctx, authInfo.UserId, orderIds, &models.OrderInfo{
		Status: int32(common.OrderStatus_RECEIVED),
	})
}

// 将订单模型转换为接口响应
func (c *OrderInfoCase) convertToProto(item *models.OrderInfo) *app.OrderInfo {
	res := c.mapper.ToDTO(item)
	res.CreatedAt = _time.TimeToTimeString(item.CreatedAt)
	res.UpdatedAt = _time.TimeToTimeString(item.UpdatedAt)
	return res
}

// 汇总下单商品信息并生成确认单
func (c *OrderInfoCase) orderBuy(ctx context.Context, member bool, createOrderGoods []*app.CreateOrderInfoGoods) (*app.ConfirmOrderInfoResponse, error) {
	newOrderGoods := make([]*app.OrderGoods, 0)
	for _, item := range createOrderGoods {
		newGoods, err := c.orderGoodsCase.convertToProtoByCreateOrderInfoGoods(ctx, member, item)
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
	return &app.ConfirmOrderInfoResponse{
		Goods:   newOrderGoods,
		Summary: &summary,
	}, nil
}

// cancelOrder 内部执行订单取消并回退库存销量
func (c *OrderInfoCase) cancelOrder(ctx context.Context, userId int64, req *app.CancelOrderInfoRequest) error {
	orderInfo, err := c.findByUserIdAndId(ctx, userId, req.GetOrderId())
	if err != nil {
		return err
	}
	// 只有待支付订单才能继续取消。
	if orderInfo.Status != int32(common.OrderStatus_CREATED) {
		return errorsx.StateConflict(
			fmt.Sprintf("订单状态错误：【%s】", common.OrderStatus_name[orderInfo.Status]),
			"order_info",
			common.OrderStatus(orderInfo.Status).String(),
			common.OrderStatus_CREATED.String(),
		)
	}

	// 微信在线支付订单在取消前先补查一次真实支付状态，避免回调延迟时误取消已支付订单
	if common.OrderPayType(orderInfo.PayType) == common.OrderPayType_ONLINE_PAY &&
		common.OrderPayChannel(orderInfo.PayChannel) == common.OrderPayChannel_WX_PAY {
		var paymentResource *app.PaymentResource
		paymentResource, err = c.wxPayCase.QueryOrderByOutTradeNo(orderInfo.OrderNo)
		if err != nil {
			return err
		}
		// 微信侧已确认支付成功时，先同步本地状态再拒绝取消。
		if paymentResource != nil && paymentResource.GetTradeState().String() == "SUCCESS" {
			// 查询到已支付后，直接复用支付服务中的成功处理逻辑补齐本地状态
			err = c.payCase.PaySuccess(ctx, orderInfo, paymentResource)
			if err != nil {
				return err
			}
			return errorsx.StateConflict(
				"订单已支付，无法取消",
				"order_info",
				common.OrderStatus_PAID.String(),
				common.OrderStatus_CREATED.String(),
			)
		}
	}
	orderIds := []int64{req.GetOrderId()}
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		query := c.orderGoodsCase.Query(ctx).OrderGoods
		var orderGoodsList []*models.OrderGoods
		opts := make([]repo.QueryOption, 0, 1)
		opts = append(opts, repo.Where(query.OrderID.In(orderIds...)))
		orderGoodsList, err = c.orderGoodsCase.List(ctx, opts...)
		for _, orderGoods := range orderGoodsList {
			// 订单取消后恢复库存并回退销量
			err = c.goodsInfoCase.subSaleNum(ctx, orderGoods.GoodsID, orderGoods.Num)
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
		return c.updateByIds(ctx, userId, orderIds, &models.OrderInfo{
			Status: int32(common.OrderStatus_CANCELED),
		})
	})
}

// 按订单编号和用户编号查询订单
func (c *OrderInfoCase) findByUserIdAndId(ctx context.Context, userId, orderId int64) (*models.OrderInfo, error) {
	query := c.Query(ctx).OrderInfo
	opts := make([]repo.QueryOption, 0, 2)
	opts = append(opts, repo.Where(query.ID.Eq(orderId)))
	opts = append(opts, repo.Where(query.UserID.Eq(userId)))
	return c.Find(ctx, opts...)
}

// 按订单编号批量更新当前用户的订单
func (c *OrderInfoCase) updateByIds(ctx context.Context, userId int64, ids []int64, entity *models.OrderInfo) error {
	// 没有待更新订单时，直接返回避免执行空 SQL。
	if len(ids) == 0 {
		return nil
	}
	query := c.Query(ctx).OrderInfo
	opts := make([]repo.QueryOption, 0, 2)
	opts = append(opts, repo.Where(query.ID.In(ids...)))
	opts = append(opts, repo.Where(query.UserID.Eq(userId)))
	return c.Update(ctx, entity, opts...)
}

// dispatchRecommendOrderEvent 根据已落库订单事实回写推荐下单事件。
func (c *OrderInfoCase) dispatchRecommendOrderEvent(payType common.OrderPayType, userId int64, goodsList []*app.CreateOrderInfoGoods, eventTime time.Time) {
	// 主体编号非法或订单商品为空时，无法构建可归因的推荐下单事件。
	if userId <= 0 || len(goodsList) == 0 {
		return
	}

	eventTypeList := make([]common.RecommendEventType, 0, 2)
	eventTypeList = append(eventTypeList, common.RecommendEventType_ORDER_CREATE)
	// 货到付款订单在创建时同步回写支付行为。
	if payType == common.OrderPayType_CASH_ON_DELIVERY {
		eventTypeList = append(eventTypeList, common.RecommendEventType_ORDER_PAY)
	}

	for _, eventType := range eventTypeList {
		for _, goodsItem := range goodsList {
			// 非法商品项直接跳过，避免把脏数据写入推荐链路。
			if goodsItem == nil || goodsItem.GetGoodsId() <= 0 {
				continue
			}
			recommendContext := goodsItem.GetRecommendContext()
			// 下单项未携带推荐上下文时，回退到空上下文，避免后续空指针。
			if recommendContext == nil {
				recommendContext = &app.RecommendContext{}
			}

			orderEventReport := &app.RecommendEventReportRequest{
				EventType: eventType,
				RecommendContext: &app.RecommendEventContext{
					Scene:     recommendContext.GetScene(),
					RequestId: recommendContext.GetRequestId(),
				},
				Items: []*app.RecommendEventItem{
					{
						GoodsId:  goodsItem.GetGoodsId(),
						GoodsNum: goodsItem.GetNum(),
						Position: recommendContext.GetPosition(),
					},
				},
			}
			// 订单创建事务提交成功后，再按落库事实回写推荐下单事件。
			pkgQueue.DispatchRecommendEvent(&app.RecommendActor{
				ActorType: common.RecommendActorType_USER,
				ActorId:   userId,
			}, orderEventReport, eventTime)
		}
	}
}
