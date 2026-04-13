package biz

import (
	"context"
	"errors"

	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"shop/api/gen/go/app"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repo"
)

// OrderAddressCase 订单收货地址业务处理对象
type OrderAddressCase struct {
	*biz.BaseCase
	*data.OrderAddressRepo
	userAddressRepo *data.UserAddressRepo
	baseAreaCase    *BaseAreaCase
	mapper          *mapper.CopierMapper[app.OrderInfoResponse_Address, models.OrderAddress]
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
		mapper: func() *mapper.CopierMapper[app.OrderInfoResponse_Address, models.OrderAddress] {
			m := mapper.NewCopierMapper[app.OrderInfoResponse_Address, models.OrderAddress]()
			m.AppendConverters(mapper.NewJSONTypeConverter[[]string]().NewConverterPair())
			return m
		}(),
	}
}

// findByOrderId 按订单编号查询订单地址
func (c *OrderAddressCase) findByOrderId(ctx context.Context, orderId int64) (*app.OrderInfoResponse_Address, error) {
	query := c.Query(ctx).OrderAddress
	opts := make([]repo.QueryOption, 0, 1)
	opts = append(opts, repo.Where(query.OrderID.Eq(orderId)))
	orderAddress, err := c.Find(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return c.mapper.ToDTO(orderAddress), nil
}

// createByOrder 按用户地址创建订单地址快照
func (c *OrderAddressCase) createByOrder(ctx context.Context, userId, orderId, addressId int64) error {
	query := c.userAddressRepo.Query(ctx).UserAddress
	opts := make([]repo.QueryOption, 0, 2)
	opts = append(opts, repo.Where(query.ID.Eq(addressId)))
	opts = append(opts, repo.Where(query.UserID.Eq(userId)))
	userAddress, err := c.userAddressRepo.Find(ctx, opts...)
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
