package biz

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"shop/pkg/config"
	_const "shop/pkg/const"

	"shop/pkg/biz"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	appv1 "shop/api/gen/go/app/v1"
	commonv1 "shop/api/gen/go/common/v1"
	"shop/pkg/queue"
	"shop/pkg/recommend/dto"
	"shop/pkg/wx"
	"shop/service/app/utils"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/liujitcn/go-utils/id"
	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	_time "github.com/liujitcn/go-utils/time"
	"github.com/liujitcn/go-utils/trans"
	"github.com/liujitcn/gorm-kit/repository"
	wxPayCore "github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/services/refunddomestic"
)

const ORDER_REFUND_REASON string = "order_refund_reason" // 退款原因

// OrderInfoCase 订单业务处理对象
type OrderInfoCase struct {
	*biz.BaseCase
	tx data.Transaction
	*data.OrderInfoRepository
	orderCancelCase    *OrderCancelCase
	orderGoodsCase     *OrderGoodsCase
	orderAddressCase   *OrderAddressCase
	orderLogisticsCase *OrderLogisticsCase
	orderPaymentCase   *OrderPaymentCase
	orderRefundCase    *OrderRefundCase
	goodsInfoCase      *GoodsInfoCase
	goodsSKUCase       *GoodsSKUCase
	userAddressCase    *UserAddressCase
	userCartCase       *UserCartCase
	baseDictItemCase   *BaseDictItemCase
	orderSchedulerCase *OrderSchedulerCase
	payCase            *PayCase
	wxPayCase          *wx.WxPayCase
	mapper             *mapper.CopierMapper[appv1.OrderInfo, models.OrderInfo]
}

// NewOrderInfoCase 创建订单业务处理对象
func NewOrderInfoCase(
	baseCase *biz.BaseCase,
	tx data.Transaction,
	orderInfoRepo *data.OrderInfoRepository,
	orderCancelCase *OrderCancelCase,
	orderGoodsCase *OrderGoodsCase,
	orderAddressCase *OrderAddressCase,
	orderLogisticsCase *OrderLogisticsCase,
	orderPaymentCase *OrderPaymentCase,
	orderRefundCase *OrderRefundCase,
	goodsInfoCase *GoodsInfoCase,
	goodsSKUCase *GoodsSKUCase,
	userAddressCase *UserAddressCase,
	userCartCase *UserCartCase,
	baseDictItemCase *BaseDictItemCase,
	orderSchedulerCase *OrderSchedulerCase,
	payCase *PayCase,
	wxPayCase *wx.WxPayCase,
) (*OrderInfoCase, error) {
	c := &OrderInfoCase{
		BaseCase:            baseCase,
		tx:                  tx,
		OrderInfoRepository: orderInfoRepo,
		orderCancelCase:     orderCancelCase,
		orderGoodsCase:      orderGoodsCase,
		orderAddressCase:    orderAddressCase,
		orderLogisticsCase:  orderLogisticsCase,
		orderPaymentCase:    orderPaymentCase,
		orderRefundCase:     orderRefundCase,
		goodsInfoCase:       goodsInfoCase,
		goodsSKUCase:        goodsSKUCase,
		userAddressCase:     userAddressCase,
		userCartCase:        userCartCase,
		baseDictItemCase:    baseDictItemCase,
		orderSchedulerCase:  orderSchedulerCase,
		payCase:             payCase,
		wxPayCase:           wxPayCase,
		mapper:              mapper.NewCopierMapper[appv1.OrderInfo, models.OrderInfo](),
	}

	// 服务启动时恢复全部未支付订单的超时取消任务
	query := c.Query(context.Background()).OrderInfo
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.Status.Eq(_const.ORDER_STATUS_CREATED)))
	list, err := c.List(context.Background(), opts...)
	if err != nil {
		return nil, err
	}
	payTimeout := config.ParsePayTimeout()
	for _, item := range list {
		// 计算当前订单距离支付超时还剩余多少秒
		createdAt := item.CreatedAt.Add(payTimeout)
		nowTime := time.Now()
		countdown := createdAt.Sub(nowTime).Seconds()
		// 已超出支付时限的订单立即执行取消。
		if countdown < 0 {
			// 自动取消订单
			err = c.cancelOrder(context.Background(), item.UserID, &appv1.CancelOrderInfoRequest{
				OrderId: item.ID,
			})
			// 自动取消执行失败时，仅记录日志继续恢复其余任务。
			if err != nil {
				log.Errorf("CancelOrder order %d failed: %v", item.ID, err)
			}
		} else {
			// 添加自动取消定时任务
			c.orderSchedulerCase.AddSchedule(item.ID, time.Duration(countdown)*time.Second, func() {
				err = c.cancelOrder(context.Background(), item.UserID, &appv1.CancelOrderInfoRequest{
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
func (c *OrderInfoCase) ConfirmOrderInfo(ctx context.Context) (*appv1.ConfirmOrderInfoResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	member := utils.IsMemberByAuthInfo(authInfo)

	// 查询购物车列表
	query := c.userCartCase.Query(ctx).UserCart
	userCartList := make([]*models.UserCart, 0)
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.UserID.Eq(authInfo.UserId)))
	opts = append(opts, repository.Where(query.IsChecked.Is(true)))
	userCartList, err = c.userCartCase.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	createOrderGoods := make([]*appv1.CreateOrderInfoGoods, 0)
	for _, item := range userCartList {
		createOrderGoods = append(createOrderGoods, &appv1.CreateOrderInfoGoods{
			GoodsId: item.GoodsID,
			SkuCode: item.SKUCode,
			Num:     item.Num,
			RecommendContext: &appv1.RecommendContext{
				Scene:     commonv1.RecommendScene(item.Scene),
				RequestId: item.RequestID,
				Position:  item.Position,
			},
		})
	}
	return c.orderBuy(ctx, member, createOrderGoods)
}

// BuyNowOrderInfo 立即购买订单
func (c *OrderInfoCase) BuyNowOrderInfo(ctx context.Context, req *appv1.BuyNowOrderInfoRequest) (*appv1.BuyNowOrderInfoResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	member := utils.IsMemberByAuthInfo(authInfo)

	// 将单个商品请求封装成统一的下单明细列表
	createOrderGoods := []*appv1.CreateOrderInfoGoods{{
		GoodsId:          req.GetGoodsId(),
		SkuCode:          req.GetSkuCode(),
		Num:              req.GetNum(),
		RecommendContext: req.GetRecommendContext(),
	}}
	var res *appv1.ConfirmOrderInfoResponse
	res, err = c.orderBuy(ctx, member, createOrderGoods)
	if err != nil {
		return nil, err
	}
	return &appv1.BuyNowOrderInfoResponse{
		Goods:     res.GetGoods(),
		Summary:   res.GetSummary(),
		ClearCart: res.GetClearCart(),
	}, nil
}

// RepurchaseOrderInfo 再次购买订单
func (c *OrderInfoCase) RepurchaseOrderInfo(ctx context.Context, req *appv1.RepurchaseOrderInfoRequest) (*appv1.RepurchaseOrderInfoResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	member := utils.IsMemberByAuthInfo(authInfo)
	var orderInfo *models.OrderInfo
	orderInfo, err = c.findByUserIDAndID(ctx, authInfo.UserId, req.GetOrderId())
	if err != nil {
		return nil, err
	}
	// 读取原订单中的商品明细，重新构造成下单请求
	query := c.orderGoodsCase.Query(ctx).OrderGoods
	var oldOrderGoods []*models.OrderGoods
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.OrderID.Eq(orderInfo.ID)))
	oldOrderGoods, err = c.orderGoodsCase.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	createOrderGoods := make([]*appv1.CreateOrderInfoGoods, 0)
	for _, item := range oldOrderGoods {
		createOrderGoods = append(createOrderGoods, &appv1.CreateOrderInfoGoods{
			GoodsId: item.GoodsID,
			SkuCode: item.SKUCode,
			Num:     item.Num,
			RecommendContext: &appv1.RecommendContext{
				Scene:     commonv1.RecommendScene(item.Scene),
				RequestId: item.RequestID,
				Position:  item.Position,
			},
		})
	}
	var res *appv1.ConfirmOrderInfoResponse
	res, err = c.orderBuy(ctx, member, createOrderGoods)
	if err != nil {
		return nil, err
	}
	return &appv1.RepurchaseOrderInfoResponse{
		Goods:     res.GetGoods(),
		Summary:   res.GetSummary(),
		ClearCart: res.GetClearCart(),
	}, nil
}

// CountOrderInfo 查询订单数量汇总
func (c *OrderInfoCase) CountOrderInfo(ctx context.Context) (*appv1.CountOrderInfoResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	query := c.Query(ctx).OrderInfo
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.UserID.Eq(authInfo.UserId)))
	var list []*models.OrderInfo
	list, err = c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	res := make(map[int32]int32)
	for _, item := range list {
		res[item.Status]++
	}
	count := make([]*appv1.CountOrderInfoResponse_Count, 0)
	for k, v := range res {
		count = append(count, &appv1.CountOrderInfoResponse_Count{
			Status: commonv1.OrderStatus(k),
			Num:    v,
		})
	}
	return &appv1.CountOrderInfoResponse{
		Counts: count,
	}, nil
}

// PageOrderInfo 查询订单分页列表
func (c *OrderInfoCase) PageOrderInfo(ctx context.Context, req *appv1.PageOrderInfoRequest) (*appv1.PageOrderInfoResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	query := c.Query(ctx).OrderInfo
	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	opts = append(opts, repository.Where(query.UserID.Eq(authInfo.UserId)))
	// 指定订单状态时，只返回该状态下的订单记录。
	if req.GetStatus() != commonv1.OrderStatus(_const.ORDER_STATUS_UNKNOWN) {
		opts = append(opts, repository.Where(query.Status.Eq(int32(req.GetStatus()))))
	}
	var page []*models.OrderInfo
	var count int64
	page, count, err = c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}

	orderIDs := make([]int64, 0, len(page))
	refundOrderIDs := make([]int64, 0)
	for _, item := range page {
		orderIDs = append(orderIDs, item.ID)
		// 售后列表展示“已退款”依赖退款成功时间，分页接口需要批量补齐。
		if item.Status == _const.ORDER_STATUS_REFUNDING {
			refundOrderIDs = append(refundOrderIDs, item.ID)
		}
	}

	orderGoodsMap := make(map[int64][]*appv1.OrderGoods)
	orderGoodsMap, err = c.orderGoodsCase.mapByOrderIDs(ctx, orderIDs)
	if err != nil {
		return nil, err
	}
	refundTimeMap := make(map[int64]string)
	refundTimeMap, err = c.orderRefundCase.mapRefundTimeByOrderIDs(ctx, refundOrderIDs)
	if err != nil {
		return nil, err
	}

	list := make([]*appv1.OrderInfo, 0)
	for _, item := range page {
		orderInfo := c.convertToProto(item)
		// 命中订单商品映射时，补齐当前订单的商品列表。
		if v, ok := orderGoodsMap[orderInfo.Id]; ok {
			orderInfo.Goods = v
		}
		// 已退款订单需要返回退款成功时间，前端据此区分处理中和已退款。
		if v, ok := refundTimeMap[orderInfo.Id]; ok {
			orderInfo.RefundTime = v
		}
		list = append(list, orderInfo)
	}

	return &appv1.PageOrderInfoResponse{
		OrderInfos: list,
		Total:      int32(count),
	}, nil
}

// GetOrderInfoIDByOrderNo 按订单号查询订单编号
func (c *OrderInfoCase) GetOrderInfoIDByOrderNo(ctx context.Context, orderNo string) (int64, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return 0, err
	}

	query := c.Query(ctx).OrderInfo
	var item *models.OrderInfo
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.OrderNo.Eq(orderNo)))
	opts = append(opts, repository.Where(query.UserID.Eq(authInfo.UserId)))
	item, err = c.Find(ctx, opts...)
	if err != nil {
		return 0, err
	}
	return item.ID, nil
}

// GetOrderInfoByID 根据订单编号查询订单
func (c *OrderInfoCase) GetOrderInfoByID(ctx context.Context, id int64) (*appv1.OrderInfoResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}

	var item *models.OrderInfo
	item, err = c.findByUserIDAndID(ctx, authInfo.UserId, id)
	if err != nil {
		return nil, err
	}
	orderInfo := c.convertToProto(item)
	// 只有待支付订单才需要返回支付剩余时间
	payTimeout := config.ParsePayTimeout()
	createdAt := item.CreatedAt.Add(payTimeout)
	nowTime := time.Now()
	countdown := createdAt.Sub(nowTime).Seconds()

	// 查询订单商品明细
	orderInfo.Goods, err = c.orderGoodsCase.listByOrderID(ctx, orderInfo.Id)
	if err != nil {
		return nil, err
	}
	// 查询订单收货地址快照
	var address *appv1.OrderInfoResponse_Address
	address, err = c.orderAddressCase.findByOrderID(ctx, orderInfo.Id)
	if err != nil {
		return nil, err
	}

	res := appv1.OrderInfoResponse{
		Order:     orderInfo,
		Address:   address,
		Countdown: float32(countdown),
	}

	// 根据订单状态补充额外展示字段
	switch commonv1.OrderStatus(item.Status) {
	case commonv1.OrderStatus(_const.ORDER_STATUS_PAID):
		// 在线支付且已支付的订单需要补充支付完成时间。
		if commonv1.OrderPayType(item.PayType) == commonv1.OrderPayType(_const.ORDER_PAY_TYPE_ONLINE_PAY) {
			res.Order.PaymentTime, err = c.orderPaymentCase.findPaymentTimeByOrderID(ctx, orderInfo.Id)
			if err != nil {
				return nil, err
			}
		}
	case commonv1.OrderStatus(_const.ORDER_STATUS_SHIPPED), commonv1.OrderStatus(_const.ORDER_STATUS_WAIT_REVIEW), commonv1.OrderStatus(_const.ORDER_STATUS_COMPLETED):
		// 已发货、待评价、已完成订单返回物流信息。
		var logistics *appv1.OrderInfoResponse_Logistics
		logistics, err = c.orderLogisticsCase.findByOrderID(ctx, orderInfo.Id)
		if err != nil {
			return nil, err
		}
		res.Logistics = logistics
	case commonv1.OrderStatus(_const.ORDER_STATUS_CANCELED):
		// 已取消订单返回取消时间
		res.Order.CancelTime, err = c.orderCancelCase.findCancelTimeByOrderID(ctx, orderInfo.Id)
		if err != nil {
			return nil, err
		}
	case commonv1.OrderStatus(_const.ORDER_STATUS_REFUNDING):
		// 退款订单返回退款时间
		res.Order.RefundTime, err = c.orderRefundCase.findRefundTimeByOrderID(ctx, orderInfo.Id)
		if err != nil {
			return nil, err
		}
	}
	return &res, nil
}

// CreateOrderInfo 创建订单并发起支付准备
func (c *OrderInfoCase) CreateOrderInfo(ctx context.Context, request *appv1.CreateOrderInfoRequest) (*appv1.CreateOrderInfoResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	status := _const.ORDER_STATUS_CREATED
	// 货到付款订单创建后直接进入待发货状态
	if request.PayType == commonv1.OrderPayType(_const.ORDER_PAY_TYPE_CASH_ON_DELIVERY) {
		status = _const.ORDER_STATUS_PAID
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
		member := utils.IsMember(ctx)
		orderGoodsList := make([]*models.OrderGoods, 0)
		for _, item := range request.GetGoods() {
			var orderGoods *models.OrderGoods
			orderGoods, err = c.orderGoodsCase.convertToModel(ctx, member, item)
			if err != nil {
				return err
			}
			orderGoodsList = append(orderGoodsList, orderGoods)
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
			err = c.goodsSKUCase.addSaleNum(ctx, orderGoods.SKUCode, orderGoods.Num)
			if err != nil {
				return err
			}
			// 从购物车下单的商品需要在下单成功后移出购物车
			err = c.userCartCase.deleteByUserIDAndGoodsIDAndSKUCode(ctx, authInfo.UserId, orderGoods.GoodsID, orderGoods.SKUCode)
			if err != nil {
				return err
			}
		}
		// 当前版本统一免运费
		orderInfo.PostFee = 0
		err = c.OrderInfoRepository.Create(ctx, orderInfo)
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
	if orderInfo.Status == _const.ORDER_STATUS_CREATED {
		// 延迟时间使用支付超时配置
		payTimeout := config.ParsePayTimeout()
		createdAt := orderInfo.CreatedAt.Add(payTimeout)
		nowTime := time.Now()
		countdown := createdAt.Sub(nowTime).Seconds()
		c.orderSchedulerCase.AddSchedule(orderInfo.ID, time.Duration(countdown)*time.Second, func() {
			err = c.cancelOrder(context.Background(), orderInfo.UserID, &appv1.CancelOrderInfoRequest{
				OrderId: orderInfo.ID,
			})
			// 定时取消执行失败时，仅记录日志避免影响后续调度。
			if err != nil {
				log.Errorf("CancelOrder order %d failed: %v", orderInfo.ID, err)
			}
		})
	}
	return &appv1.CreateOrderInfoResponse{
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
	orderInfo, err = c.findByUserIDAndID(ctx, authInfo.UserId, id)
	if err != nil {
		return err
	}
	// 只有已完成、退款中或已取消订单允许删除，待评价订单需要先完成评价流程。
	if !(orderInfo.Status == _const.ORDER_STATUS_COMPLETED || orderInfo.Status == _const.ORDER_STATUS_REFUNDING || orderInfo.Status == _const.ORDER_STATUS_CANCELED) {
		return errorsx.StateConflict(
			fmt.Sprintf("订单状态错误：【%s】", commonv1.OrderStatus_name[orderInfo.Status]),
			"order_info",
			commonv1.OrderStatus(orderInfo.Status).String(),
			"COMPLETED|REFUNDING|CANCELED",
		)
	}

	orderIDs := []int64{id}
	return c.updateByIDs(ctx, authInfo.UserId, orderIDs, &models.OrderInfo{
		Status: _const.ORDER_STATUS_DELETED,
	})
}

// CancelOrderInfo 取消订单并回退库存销量
func (c *OrderInfoCase) CancelOrderInfo(ctx context.Context, req *appv1.CancelOrderInfoRequest) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	return c.cancelOrder(ctx, authInfo.UserId, req)
}

// RefundOrderInfo 申请订单退款
func (c *OrderInfoCase) RefundOrderInfo(ctx context.Context, req *appv1.RefundOrderInfoRequest) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	var orderInfo *models.OrderInfo
	orderInfo, err = c.findByUserIDAndID(ctx, authInfo.UserId, req.GetOrderId())
	if err != nil {
		return err
	}

	// 只有已支付订单才能继续申请退款。
	if orderInfo.Status != _const.ORDER_STATUS_PAID {
		return errorsx.StateConflict(
			fmt.Sprintf("订单状态错误：【%s】", commonv1.OrderStatus_name[orderInfo.Status]),
			"order_info",
			commonv1.OrderStatus(orderInfo.Status).String(),
			commonv1.OrderStatus(_const.ORDER_STATUS_PAID).String(),
		)
	}

	orderRefund := &models.OrderRefund{
		OrderID:  req.GetOrderId(),
		RefundNo: orderInfo.OrderNo, // 退款单号，和订单号使用一个方便查询退款
		Reason:   int32(req.GetReason()),
	}
	// 只有在线支付订单才会走退款单和微信退款流程
	if commonv1.OrderPayType(orderInfo.PayType) == commonv1.OrderPayType(_const.ORDER_PAY_TYPE_ONLINE_PAY) {
		// 先把退款原因值翻译成字典标签，便于退款记录展示
		reason := strconv.Itoa(int(orderRefund.Reason))
		var label string
		label, err = c.baseDictItemCase.findLabelByCodeAndValue(ctx, ORDER_REFUND_REASON, reason)
		// 字典标签查询成功时，使用标签替换原始原因值。
		if err == nil {
			reason = label
		}
		// 仅微信支付订单需要调用微信退款接口
		if commonv1.OrderPayChannel(orderInfo.PayChannel) == commonv1.OrderPayChannel(_const.ORDER_PAY_CHANNEL_WX_PAY) {
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
						var refundResource *appv1.RefundResource
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
							appv1.RefundResource_RefundStatus(_const.REFUND_RESOURCE_STATUS_SUCCESS).String(),
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
	orderIDs := []int64{req.GetOrderId()}
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		// 退款成功后保存退款记录
		err = c.orderRefundCase.Create(ctx, orderRefund)
		if err != nil {
			return err
		}
		return c.updateByIDs(ctx, authInfo.UserId, orderIDs, &models.OrderInfo{
			Status: _const.ORDER_STATUS_REFUNDING,
		})
	})
}

// ReceiveOrderInfo 确认收货
func (c *OrderInfoCase) ReceiveOrderInfo(ctx context.Context, req *appv1.ReceiveOrderInfoRequest) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}

	var orderInfo *models.OrderInfo
	orderInfo, err = c.findByUserIDAndID(ctx, authInfo.UserId, req.GetOrderId())
	if err != nil {
		return err
	}
	// 只有已发货订单才能确认收货。
	if orderInfo.Status != _const.ORDER_STATUS_SHIPPED {
		return errorsx.StateConflict(
			fmt.Sprintf("订单状态错误：【%s】", commonv1.OrderStatus_name[orderInfo.Status]),
			"order_info",
			commonv1.OrderStatus(orderInfo.Status).String(),
			commonv1.OrderStatus(_const.ORDER_STATUS_SHIPPED).String(),
		)
	}

	orderIDs := []int64{req.GetOrderId()}
	return c.updateByIDs(ctx, authInfo.UserId, orderIDs, &models.OrderInfo{
		Status: _const.ORDER_STATUS_WAIT_REVIEW,
	})
}

// 将订单模型转换为接口响应
func (c *OrderInfoCase) convertToProto(item *models.OrderInfo) *appv1.OrderInfo {
	res := c.mapper.ToDTO(item)
	res.CreatedAt = _time.TimeToTimeString(item.CreatedAt)
	res.UpdatedAt = _time.TimeToTimeString(item.UpdatedAt)
	return res
}

// 汇总下单商品信息并生成确认单
func (c *OrderInfoCase) orderBuy(ctx context.Context, member bool, createOrderGoods []*appv1.CreateOrderInfoGoods) (*appv1.ConfirmOrderInfoResponse, error) {
	newOrderGoods := make([]*appv1.OrderGoods, 0)
	for _, item := range createOrderGoods {
		model, err := c.orderGoodsCase.convertToModel(ctx, member, item)
		if err != nil {
			return nil, err
		}
		newGoods := c.orderGoodsCase.toOrderGoods(model)
		newOrderGoods = append(newOrderGoods, newGoods)
	}

	var summary appv1.OrderSummary
	for _, orderGoods := range newOrderGoods {
		summary.PayMoney += orderGoods.TotalPayPrice
		summary.TotalMoney += orderGoods.TotalPrice
		summary.GoodsNum += orderGoods.Num
	}
	// 当前版本统一免运费
	summary.PostFee = 0
	return &appv1.ConfirmOrderInfoResponse{
		Goods:   newOrderGoods,
		Summary: &summary,
	}, nil
}

// cancelOrder 内部执行订单取消并回退库存销量
func (c *OrderInfoCase) cancelOrder(ctx context.Context, userID int64, req *appv1.CancelOrderInfoRequest) error {
	orderInfo, err := c.findByUserIDAndID(ctx, userID, req.GetOrderId())
	if err != nil {
		return err
	}
	// 只有待支付订单才能继续取消。
	if orderInfo.Status != _const.ORDER_STATUS_CREATED {
		return errorsx.StateConflict(
			fmt.Sprintf("订单状态错误：【%s】", commonv1.OrderStatus_name[orderInfo.Status]),
			"order_info",
			commonv1.OrderStatus(orderInfo.Status).String(),
			commonv1.OrderStatus(_const.ORDER_STATUS_CREATED).String(),
		)
	}

	// 微信在线支付订单在取消前先补查一次真实支付状态，避免回调延迟时误取消已支付订单
	if commonv1.OrderPayType(orderInfo.PayType) == commonv1.OrderPayType(_const.ORDER_PAY_TYPE_ONLINE_PAY) &&
		commonv1.OrderPayChannel(orderInfo.PayChannel) == commonv1.OrderPayChannel(_const.ORDER_PAY_CHANNEL_WX_PAY) {
		var paymentResource *appv1.PaymentResource
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
				commonv1.OrderStatus(_const.ORDER_STATUS_PAID).String(),
				commonv1.OrderStatus(_const.ORDER_STATUS_CREATED).String(),
			)
		}
	}
	orderIDs := []int64{req.GetOrderId()}
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		query := c.orderGoodsCase.Query(ctx).OrderGoods
		var orderGoodsList []*models.OrderGoods
		opts := make([]repository.QueryOption, 0, 1)
		opts = append(opts, repository.Where(query.OrderID.In(orderIDs...)))
		orderGoodsList, err = c.orderGoodsCase.List(ctx, opts...)
		for _, orderGoods := range orderGoodsList {
			// 订单取消后恢复库存并回退销量
			err = c.goodsInfoCase.subSaleNum(ctx, orderGoods.GoodsID, orderGoods.Num)
			if err != nil {
				return err
			}
			err = c.goodsSKUCase.subSaleNum(ctx, orderGoods.SKUCode, orderGoods.Num)
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
		return c.updateByIDs(ctx, userID, orderIDs, &models.OrderInfo{
			Status: _const.ORDER_STATUS_CANCELED,
		})
	})
}

// 按订单编号和用户编号查询订单
func (c *OrderInfoCase) findByUserIDAndID(ctx context.Context, userID, orderID int64) (*models.OrderInfo, error) {
	query := c.Query(ctx).OrderInfo
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.ID.Eq(orderID)))
	opts = append(opts, repository.Where(query.UserID.Eq(userID)))
	return c.Find(ctx, opts...)
}

// 按订单编号批量更新当前用户的订单
func (c *OrderInfoCase) updateByIDs(ctx context.Context, userID int64, ids []int64, entity *models.OrderInfo) error {
	// 没有待更新订单时，直接返回避免执行空 SQL。
	if len(ids) == 0 {
		return nil
	}
	query := c.Query(ctx).OrderInfo
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.ID.In(ids...)))
	opts = append(opts, repository.Where(query.UserID.Eq(userID)))
	return c.Update(ctx, entity, opts...)
}

// dispatchRecommendOrderEvent 根据已落库订单事实回写推荐下单事件。
func (c *OrderInfoCase) dispatchRecommendOrderEvent(payType commonv1.OrderPayType, userID int64, goodsList []*appv1.CreateOrderInfoGoods, eventTime time.Time) {
	// 主体编号非法或订单商品为空时，无法构建可归因的推荐下单事件。
	if userID <= 0 || len(goodsList) == 0 {
		return
	}

	eventTypeList := make([]commonv1.RecommendEventType, 0, 2)
	eventTypeList = append(eventTypeList, commonv1.RecommendEventType(_const.RECOMMEND_EVENT_TYPE_ORDER_CREATE))
	// 货到付款订单在创建时同步回写支付行为。
	if payType == commonv1.OrderPayType(_const.ORDER_PAY_TYPE_CASH_ON_DELIVERY) {
		eventTypeList = append(eventTypeList, commonv1.RecommendEventType(_const.RECOMMEND_EVENT_TYPE_ORDER_PAY))
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
				recommendContext = &appv1.RecommendContext{}
			}

			orderEventReport := &appv1.RecommendEventReportRequest{
				EventType: eventType,
				RecommendContext: &appv1.RecommendEventContext{
					Scene:     recommendContext.GetScene(),
					RequestId: recommendContext.GetRequestId(),
				},
				Items: []*appv1.RecommendEventItem{
					{
						GoodsId:  goodsItem.GetGoodsId(),
						GoodsNum: goodsItem.GetNum(),
						Position: recommendContext.GetPosition(),
					},
				},
			}
			// 订单创建事务提交成功后，再按落库事实回写推荐下单事件。
			queue.DispatchRecommendEvent(&dto.RecommendActor{
				ActorType: commonv1.RecommendActorType(_const.RECOMMEND_ACTOR_TYPE_USER),
				ActorID:   userID,
			}, orderEventReport, eventTime)
		}
	}
}
