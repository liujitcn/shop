package biz

import (
	"context"

	"shop/pkg/biz"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	appv1 "shop/api/gen/go/app/v1"

	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
)

// OrderAddressCase 订单收货地址业务处理对象
type OrderAddressCase struct {
	*biz.BaseCase
	*data.OrderAddressRepository
	userAddressRepo *data.UserAddressRepository
	baseAreaCase    *BaseAreaCase
	mapper          *mapper.CopierMapper[appv1.OrderInfoResponse_Address, models.OrderAddress]
}

// NewOrderAddressCase 创建订单收货地址业务处理对象
func NewOrderAddressCase(baseCase *biz.BaseCase, orderAddressRepo *data.OrderAddressRepository,
	userAddressRepo *data.UserAddressRepository,
	baseAreaCase *BaseAreaCase,
) *OrderAddressCase {
	return &OrderAddressCase{
		BaseCase:               baseCase,
		OrderAddressRepository: orderAddressRepo,
		userAddressRepo:        userAddressRepo,
		baseAreaCase:           baseAreaCase,
		mapper: func() *mapper.CopierMapper[appv1.OrderInfoResponse_Address, models.OrderAddress] {
			m := mapper.NewCopierMapper[appv1.OrderInfoResponse_Address, models.OrderAddress]()
			m.AppendConverters(mapper.NewJSONTypeConverter[[]string]().NewConverterPair())
			return m
		}(),
	}
}

// findByOrderID 按订单编号查询订单地址
func (c *OrderAddressCase) findByOrderID(ctx context.Context, orderID int64) (*appv1.OrderInfoResponse_Address, error) {
	query := c.Query(ctx).OrderAddress
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.OrderID.Eq(orderID)))
	orderAddress, err := c.Find(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return c.mapper.ToDTO(orderAddress), nil
}

// createByOrder 按用户地址创建订单地址快照
func (c *OrderAddressCase) createByOrder(ctx context.Context, userID, orderID, addressID int64) error {
	query := c.userAddressRepo.Query(ctx).UserAddress
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.ID.Eq(addressID)))
	opts = append(opts, repository.Where(query.UserID.Eq(userID)))
	userAddress, err := c.userAddressRepo.Find(ctx, opts...)
	if err != nil {
		return errorsx.InvalidArgument("地址错误").WithCause(err)
	}
	// 下单时复制一份地址快照，避免用户后续修改地址影响历史订单展示
	return c.Create(ctx, &models.OrderAddress{
		OrderID:  orderID,
		Receiver: userAddress.Receiver,
		Contact:  userAddress.Contact,
		Address:  _string.ConvertStringArrayToString(c.baseAreaCase.getAddressListByCode(ctx, userAddress.Address)),
		Detail:   userAddress.Detail,
	})
}
