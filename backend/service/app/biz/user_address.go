package biz

import (
	"context"

	appv1 "shop/api/gen/go/app/v1"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repository"
)

// UserAddressCase 用户收货地址业务处理对象
type UserAddressCase struct {
	*biz.BaseCase
	tx data.Transaction
	*data.UserAddressRepository
	baseAreaCase *BaseAreaCase
	formMapper   *mapper.CopierMapper[appv1.UserAddressForm, models.UserAddress]
	mapper       *mapper.CopierMapper[appv1.UserAddress, models.UserAddress]
}

// NewUserAddressCase 创建用户收货地址业务处理对象
func NewUserAddressCase(baseCase *biz.BaseCase, tx data.Transaction,
	userAddressRepo *data.UserAddressRepository,
	baseAreaCase *BaseAreaCase,
) *UserAddressCase {
	formMapper := mapper.NewCopierMapper[appv1.UserAddressForm, models.UserAddress]()
	formMapper.AppendConverters(mapper.NewJSONTypeConverter[[]string]().NewConverterPair())
	return &UserAddressCase{
		BaseCase:              baseCase,
		tx:                    tx,
		UserAddressRepository: userAddressRepo,
		baseAreaCase:          baseAreaCase,
		formMapper:            formMapper,
		mapper:                mapper.NewCopierMapper[appv1.UserAddress, models.UserAddress](),
	}
}

// ListUserAddresses 查询用户地址列表
func (c *UserAddressCase) ListUserAddresses(ctx context.Context) (*appv1.ListUserAddressesResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	query := c.Query(ctx).UserAddress
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	opts = append(opts, repository.Where(query.UserID.Eq(authInfo.UserId)))
	var all []*models.UserAddress
	all, err = c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	list := make([]*appv1.UserAddress, 0)
	for _, address := range all {
		item := c.mapper.ToDTO(address)
		item.Address = c.baseAreaCase.getAddressListByCode(ctx, address.Address)
		list = append(list, item)
	}
	return &appv1.ListUserAddressesResponse{
		UserAddresses: list,
	}, nil
}

// GetUserAddress 查询用户地址
func (c *UserAddressCase) GetUserAddress(ctx context.Context, id int64) (*appv1.UserAddressForm, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	query := c.Query(ctx).UserAddress
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.ID.Eq(id)))
	opts = append(opts, repository.Where(query.UserID.Eq(authInfo.UserId)))
	var userAddress *models.UserAddress
	userAddress, err = c.Find(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return c.convertToProto(ctx, userAddress), nil
}

// CreateUserAddress 创建用户地址
func (c *UserAddressCase) CreateUserAddress(ctx context.Context, userAddress *appv1.UserAddressForm) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	address := c.convertToModel(authInfo.UserId, userAddress)
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		// 新地址勾选默认时，先清空当前用户已有默认地址。
		if address.IsDefault {
			// 新地址设为默认时，需要先清空当前用户的其他默认地址。
			if err = c.clearDefaultAddress(ctx, authInfo.UserId, 0); err != nil {
				return err
			}
		}
		err = c.UserAddressRepository.Create(ctx, address)
		if err != nil {
			return err
		}
		return nil
	})
}

// UpdateUserAddress 更新用户地址
func (c *UserAddressCase) UpdateUserAddress(ctx context.Context, userAddress *appv1.UserAddressForm) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	address := c.convertToModel(authInfo.UserId, userAddress)

	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		// 更新地址勾选默认时，先清空当前用户其他默认地址。
		if address.IsDefault {
			// 修改默认地址时，同样需要保证只有一条默认记录。
			if err = c.clearDefaultAddress(ctx, authInfo.UserId, address.ID); err != nil {
				return err
			}
		}
		query := c.Query(ctx).UserAddress
		opts := make([]repository.QueryOption, 0, 2)
		opts = append(opts, repository.Where(query.ID.Eq(address.ID)))
		opts = append(opts, repository.Where(query.UserID.Eq(authInfo.UserId)))
		err = c.UserAddressRepository.Update(ctx, address, opts...)
		if err != nil {
			return err
		}
		return nil
	})
}

// DeleteUserAddress 删除用户地址
func (c *UserAddressCase) DeleteUserAddress(ctx context.Context, id int64) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	query := c.Query(ctx).UserAddress
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.ID.Eq(id)))
	opts = append(opts, repository.Where(query.UserID.Eq(authInfo.UserId)))
	return c.Delete(ctx, opts...)
}

// 将用户地址模型转换为表单响应
func (c *UserAddressCase) convertToProto(ctx context.Context, item *models.UserAddress) *appv1.UserAddressForm {
	res := c.formMapper.ToDTO(item)
	res.AddressName = c.baseAreaCase.getAddressListByCode(ctx, item.Address)
	return res
}

// 将用户地址表单转换为模型
func (c *UserAddressCase) convertToModel(userID int64, item *appv1.UserAddressForm) *models.UserAddress {
	res := c.formMapper.ToEntity(item)
	res.UserID = userID
	return res
}

// clearDefaultAddress 清空指定用户的默认地址，可选择排除当前地址。
func (c *UserAddressCase) clearDefaultAddress(ctx context.Context, userID, excludeID int64) error {
	query := c.Query(ctx).UserAddress
	do := query.WithContext(ctx).Where(query.UserID.Eq(userID), query.IsDefault.Is(true))
	// 排除当前正在操作的地址时，不更新该地址默认状态。
	if excludeID > 0 {
		do = do.Where(query.ID.Neq(excludeID))
	}

	res, err := do.Updates(map[string]interface{}{
		"is_default": false,
	})
	if err != nil {
		return err
	}
	return res.Error
}
