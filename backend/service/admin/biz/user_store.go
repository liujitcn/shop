package biz

import (
	"context"

	"shop/api/gen/go/admin"
	"shop/api/gen/go/common"
	"shop/pkg/biz"
	_const "shop/pkg/const"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repo"
)

// UserStoreCase 门店申请业务实例
type UserStoreCase struct {
	*biz.BaseCase
	tx data.Transaction
	*data.UserStoreRepo
	baseAreaRepo *data.BaseAreaRepo
	baseUserCase *BaseUserCase
	baseRoleCase *BaseRoleCase
	mapper       *mapper.CopierMapper[admin.UserStore, models.UserStore]
}

// NewUserStoreCase 创建门店申请业务实例
func NewUserStoreCase(baseCase *biz.BaseCase, tx data.Transaction, userStoreRepo *data.UserStoreRepo, baseAreaRepo *data.BaseAreaRepo, baseUserCase *BaseUserCase, baseRoleCase *BaseRoleCase) *UserStoreCase {
	userStoreMapper := mapper.NewCopierMapper[admin.UserStore, models.UserStore]()
	userStoreMapper.AppendConverters(mapper.NewJSONTypeConverter[[]string]().NewConverterPair())
	return &UserStoreCase{
		BaseCase:      baseCase,
		tx:            tx,
		UserStoreRepo: userStoreRepo,
		baseAreaRepo:  baseAreaRepo,
		baseUserCase:  baseUserCase,
		baseRoleCase:  baseRoleCase,
		mapper:        userStoreMapper,
	}
}

// PageUserStore 分页查询门店申请
func (c *UserStoreCase) PageUserStore(ctx context.Context, req *admin.PageUserStoreRequest) (*admin.PageUserStoreResponse, error) {
	query := c.Query(ctx).UserStore
	opts := make([]repo.QueryOption, 0, 3)
	opts = append(opts, repo.Order(query.CreatedAt.Desc()))
	if req.GetName() != "" {
		opts = append(opts, repo.Where(query.Name.Like("%"+req.GetName()+"%")))
	}
	if req.Status != nil {
		opts = append(opts, repo.Where(query.Status.Eq(int32(req.GetStatus()))))
	}

	page, count, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}

	userIds := make([]int64, 0, len(page))
	for _, item := range page {
		userIds = append(userIds, item.UserID)
	}

	userMap := make(map[int64]*models.BaseUser)
	if len(userIds) > 0 {
		var userList []*models.BaseUser
		userList, err = c.baseUserCase.ListByIds(ctx, userIds)
		if err != nil {
			return nil, err
		}
		for _, item := range userList {
			userMap[item.ID] = item
		}
	}

	list := make([]*admin.UserStore, 0, len(page))
	for _, item := range page {
		userStore := c.toUserStore(ctx, item)
		if user, ok := userMap[item.UserID]; ok {
			userStore.NickName = user.NickName
			userStore.Phone = user.Phone
		}
		list = append(list, userStore)
	}

	return &admin.PageUserStoreResponse{
		List:  list,
		Total: int32(count),
	}, nil
}

// GetUserStore 获取门店申请
func (c *UserStoreCase) GetUserStore(ctx context.Context, id int64) (*admin.UserStore, error) {
	userStore, err := c.FindById(ctx, id)
	if err != nil {
		return nil, err
	}

	res := c.toUserStore(ctx, userStore)

	var baseUser *models.BaseUser
	baseUser, err = c.baseUserCase.FindById(ctx, userStore.UserID)
	if err != nil {
		return nil, err
	}
	res.NickName = baseUser.NickName
	res.Phone = baseUser.Phone
	return res, nil
}

// AuditUserStore 审核门店申请
func (c *UserStoreCase) AuditUserStore(ctx context.Context, req *admin.AuditUserStoreForm) error {
	userStore, err := c.FindById(ctx, req.GetId())
	if err != nil {
		return err
	}

	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		err = c.UpdateById(ctx, &models.UserStore{
			ID:     req.GetId(),
			Status: int32(req.GetStatus()),
			Remark: req.GetRemark(),
		})
		if err != nil {
			return err
		}

		code := _const.BaseRoleCode_Guest
		if req.GetStatus() == common.UserStoreStatus_APPROVED {
			code = _const.BaseRoleCode_User
		}

		roleQuery := c.baseRoleCase.Query(ctx).BaseRole
		var baseRole *models.BaseRole
		baseRole, err = c.baseRoleCase.Find(ctx, repo.Where(roleQuery.Code.Eq(code)))
		if err != nil {
			return err
		}
		return c.baseUserCase.UpdateById(ctx, &models.BaseUser{
			ID:     userStore.UserID,
			RoleID: baseRole.ID,
		})
	})
}

// toUserStore 转换门店申请响应
func (c *UserStoreCase) toUserStore(ctx context.Context, item *models.UserStore) *admin.UserStore {
	res := c.mapper.ToDTO(item)
	res.Address = c.getAddressListByCode(ctx, item.Address)
	return res
}

// getAddressListByCode 根据区域编号构建地址名称
func (c *UserStoreCase) getAddressListByCode(ctx context.Context, address string) []string {
	ids := _string.ConvertJsonStringToInt64Array(address)
	if len(ids) == 0 {
		return []string{}
	}

	areaList, err := c.baseAreaRepo.ListByIds(ctx, ids)
	if err != nil {
		return []string{}
	}

	areaMap := make(map[int64]string, len(areaList))
	for _, item := range areaList {
		areaMap[item.ID] = item.Name
	}

	res := make([]string, 0, len(ids))
	for _, id := range ids {
		if name, ok := areaMap[id]; ok {
			res = append(res, name)
		}
	}
	return res
}
