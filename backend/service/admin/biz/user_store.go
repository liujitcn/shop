package biz

import (
	"context"

	"shop/pkg/queue"

	adminv1 "shop/api/gen/go/admin/v1"
	commonv1 "shop/api/gen/go/common/v1"
	"shop/pkg/biz"
	_const "shop/pkg/const"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
)

// UserStoreCase 门店申请业务实例
type UserStoreCase struct {
	*biz.BaseCase
	tx data.Transaction
	*data.UserStoreRepository
	baseAreaRepo *data.BaseAreaRepository
	baseUserCase *BaseUserCase
	baseRoleCase *BaseRoleCase
	mapper       *mapper.CopierMapper[adminv1.UserStore, models.UserStore]
}

// NewUserStoreCase 创建门店申请业务实例
func NewUserStoreCase(baseCase *biz.BaseCase, tx data.Transaction, userStoreRepo *data.UserStoreRepository, baseAreaRepo *data.BaseAreaRepository, baseUserCase *BaseUserCase, baseRoleCase *BaseRoleCase,
) *UserStoreCase {
	userStoreMapper := mapper.NewCopierMapper[adminv1.UserStore, models.UserStore]()
	userStoreMapper.AppendConverters(mapper.NewJSONTypeConverter[[]string]().NewConverterPair())
	return &UserStoreCase{
		BaseCase:            baseCase,
		tx:                  tx,
		UserStoreRepository: userStoreRepo,
		baseAreaRepo:        baseAreaRepo,
		baseUserCase:        baseUserCase,
		baseRoleCase:        baseRoleCase,
		mapper:              userStoreMapper,
	}
}

// PageUserStores 分页查询门店申请
func (c *UserStoreCase) PageUserStores(ctx context.Context, req *adminv1.PageUserStoresRequest) (*adminv1.PageUserStoresResponse, error) {
	query := c.Query(ctx).UserStore
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	// 传入名称关键字时，按名称模糊匹配门店申请。
	if req.GetName() != "" {
		opts = append(opts, repository.Where(query.Name.Like("%"+req.GetName()+"%")))
	}
	if req.Status != nil {
		opts = append(opts, repository.Where(query.Status.Eq(int32(req.GetStatus()))))
	}

	page, count, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}

	userIDs := make([]int64, 0, len(page))
	for _, item := range page {
		userIDs = append(userIDs, item.UserID)
	}

	userMap := make(map[int64]*models.BaseUser)
	// 页面存在用户编号时，批量回查用户基础信息补齐昵称和手机号。
	if len(userIDs) > 0 {
		var userList []*models.BaseUser
		userList, err = c.baseUserCase.ListByIDs(ctx, userIDs)
		if err != nil {
			return nil, err
		}
		for _, item := range userList {
			userMap[item.ID] = item
		}
	}

	list := make([]*adminv1.UserStore, 0, len(page))
	for _, item := range page {
		userStore := c.toUserStore(ctx, item)
		// 命中用户信息时，补齐申请人的昵称和手机号。
		if user, ok := userMap[item.UserID]; ok {
			userStore.NickName = user.NickName
			userStore.Phone = user.Phone
		}
		list = append(list, userStore)
	}

	return &adminv1.PageUserStoresResponse{
		UserStores: list,
		Total:      int32(count),
	}, nil
}

// GetUserStore 获取门店申请
func (c *UserStoreCase) GetUserStore(ctx context.Context, id int64) (*adminv1.UserStore, error) {
	userStore, err := c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	res := c.toUserStore(ctx, userStore)

	var baseUser *models.BaseUser
	baseUser, err = c.baseUserCase.FindByID(ctx, userStore.UserID)
	if err != nil {
		return nil, err
	}
	res.NickName = baseUser.NickName
	res.Phone = baseUser.Phone
	return res, nil
}

// AuditUserStore 审核门店申请
func (c *UserStoreCase) AuditUserStore(ctx context.Context, req *adminv1.AuditUserStoreRequest) error {
	userStore, err := c.FindByID(ctx, req.GetId())
	if err != nil {
		return err
	}

	err = c.tx.Transaction(ctx, func(ctx context.Context) error {
		err = c.UpdateByID(ctx, &models.UserStore{
			ID:     req.GetId(),
			Status: int32(req.GetStatus()),
			Remark: req.GetRemark(),
		})
		if err != nil {
			return err
		}

		code := _const.BASE_ROLE_CODE_GUEST
		// 审核通过时，将用户角色切换为正式用户角色。
		if req.GetStatus() == commonv1.UserStoreStatus(_const.USER_STORE_STATUS_APPROVED) {
			code = _const.BASE_ROLE_CODE_USER
		}

		query := c.baseRoleCase.Query(ctx).BaseRole
		opts := make([]repository.QueryOption, 0, 1)
		opts = append(opts, repository.Where(query.Code.Eq(code)))
		var baseRole *models.BaseRole
		baseRole, err = c.baseRoleCase.Find(ctx, opts...)
		if err != nil {
			return err
		}
		return c.baseUserCase.UpdateByID(ctx, &models.BaseUser{
			ID:     userStore.UserID,
			RoleID: baseRole.ID,
		})
	})
	if err != nil {
		return err
	}
	// 门店审核会影响用户角色，审核成功后再异步同步最新用户画像到推荐系统。
	queue.DispatchRecommendSyncBaseUser(userStore.UserID)
	return nil
}

// toUserStore 转换门店申请响应
func (c *UserStoreCase) toUserStore(ctx context.Context, item *models.UserStore) *adminv1.UserStore {
	res := c.mapper.ToDTO(item)
	res.Address = c.getAddressListByCode(ctx, item.Address)
	return res
}

// getAddressListByCode 根据区域编号构建地址名称
func (c *UserStoreCase) getAddressListByCode(ctx context.Context, address string) []string {
	ids := _string.ConvertJsonStringToInt64Array(address)
	// 地址编号为空时，直接返回空地址列表。
	if len(ids) == 0 {
		return []string{}
	}

	areaList, err := c.baseAreaRepo.ListByIDs(ctx, ids)
	if err != nil {
		return []string{}
	}

	areaMap := make(map[int64]string, len(areaList))
	for _, item := range areaList {
		areaMap[item.ID] = item.Name
	}

	res := make([]string, 0, len(ids))
	for _, id := range ids {
		// 命中区域名称映射时，按原顺序补齐地址名称。
		if name, ok := areaMap[id]; ok {
			res = append(res, name)
		}
	}
	return res
}
