package biz

import (
	"context"
	"errors"

	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"shop/api/gen/go/app"

	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repo"
)

// OrderAddressCase 订单收货地址业务处理对象
type OrderAddressCase struct {
	*biz.BaseCase
	*data.OrderAddressRepo
	userAddressRepo *data.UserAddressRepo
	baseAreaCase    *BaseAreaCase
}

// NewOrderAddressCase 创建订单收货地址业务处理对象
func NewOrderAddressCase(baseCase *biz.BaseCase, orderAddressRepo *data.OrderAddressRepo,
	userAddressRepo *data.UserAddressRepo,
	baseAreaCase *BaseAreaCase,
) *OrderAddressCase {
	return &OrderAddressCase{
		BaseCase:         baseCase,
		OrderAddressRepo: orderAddressRepo,
		userAddressRepo:  userAddressRepo,
		baseAreaCase:     baseAreaCase,
	}
}

// findByOrderId 按订单编号查询订单地址
func (c *OrderAddressCase) findByOrderId(ctx context.Context, orderId int64) (*app.OrderResponse_Address, error) {
	query := c.Query(ctx).OrderAddress
	orderAddress, err := c.Find(ctx,
		repo.Where(query.OrderID.Eq(orderId)),
	)
	if err != nil {
		return nil, err
	}
	return &app.OrderResponse_Address{
		Receiver: orderAddress.Receiver,
		Contact:  orderAddress.Contact,
		Address:  _string.ConvertJsonStringToStringArray(orderAddress.Address),
		Detail:   orderAddress.Detail,
	}, nil
}

// createByOrder 按用户地址创建订单地址快照
func (c *OrderAddressCase) createByOrder(ctx context.Context, userId, orderId, addressId int64) error {
	userAddressQuery := c.userAddressRepo.Query(ctx).UserAddress
	userAddress, err := c.userAddressRepo.Find(ctx,
		repo.Where(userAddressQuery.ID.Eq(addressId)),
		repo.Where(userAddressQuery.UserID.Eq(userId)),
	)
	if err != nil {
		return errors.New("地址错误")
	}
	// 下单时复制一份地址快照，避免用户后续修改地址影响历史订单展示
	return c.Create(ctx, &models.OrderAddress{
		OrderID:  orderId,
		Receiver: userAddress.Receiver,
		Contact:  userAddress.Contact,
		Address:  c.baseAreaCase.getAddressByCode(ctx, userAddress.Address),
		Detail:   userAddress.Detail,
	})
}
