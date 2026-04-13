package biz

import (
	"context"

	"shop/api/gen/go/admin"
	"shop/api/gen/go/common"
	"shop/pkg/biz"
	_const "shop/pkg/const"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repo"
)

// BaseRoleCase 角色业务实例
type BaseRoleCase struct {
	*biz.BaseCase
	tx data.Transaction
	*data.BaseRoleRepo
	casbinRuleCase *CasbinRuleCase
	formMapper     *mapper.CopierMapper[admin.BaseRoleForm, models.BaseRole]
	mapper         *mapper.CopierMapper[admin.BaseRole, models.BaseRole]
}

// NewBaseRoleCase 创建角色业务实例
func NewBaseRoleCase(baseCase *biz.BaseCase, tx data.Transaction, baseRoleRepo *data.BaseRoleRepo, casbinRuleCase *CasbinRuleCase) *BaseRoleCase {
	return &BaseRoleCase{
		BaseCase:       baseCase,
		tx:             tx,
		BaseRoleRepo:   baseRoleRepo,
		casbinRuleCase: casbinRuleCase,
		formMapper:     mapper.NewCopierMapper[admin.BaseRoleForm, models.BaseRole](),
		mapper:         mapper.NewCopierMapper[admin.BaseRole, models.BaseRole](),
	}
}

// OptionBaseRole 查询角色选项
func (c *BaseRoleCase) OptionBaseRole(ctx context.Context) (*common.SelectOptionResponse, error) {
	query := c.Query(ctx).BaseRole
	opts := make([]repo.QueryOption, 0, 1)
	opts = append(opts, repo.Order(query.CreatedAt.Desc()))
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	options := make([]*common.SelectOptionResponse_Option, 0, len(list))
	for _, item := range list {
		options = append(options, &common.SelectOptionResponse_Option{
			Label: item.Name,
			Value: item.ID,
		})
	}
	return &common.SelectOptionResponse{List: options}, nil
}

// PageBaseRole 分页查询角色
func (c *BaseRoleCase) PageBaseRole(ctx context.Context, req *admin.PageBaseRoleRequest) (*admin.PageBaseRoleResponse, error) {
	query := c.Query(ctx).BaseRole
	opts := make([]repo.QueryOption, 0, 4)
	opts = append(opts, repo.Order(query.CreatedAt.Desc()))
	if req.Status != nil {
		opts = append(opts, repo.Where(query.Status.Eq(int32(req.GetStatus()))))
	}
	// 传入名称关键字时，按名称模糊匹配角色。
	if req.GetName() != "" {
		opts = append(opts, repo.Where(query.Name.Like("%"+req.GetName()+"%")))
	}
	// 传入编码关键字时，按编码模糊匹配角色。
	if req.GetCode() != "" {
		opts = append(opts, repo.Where(query.Code.Like("%"+req.GetCode()+"%")))
	}

	list, total, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}

	resList := make([]*admin.BaseRole, 0, len(list))
	for _, item := range list {
		resList = append(resList, c.toBaseRole(item))
	}
	return &admin.PageBaseRoleResponse{List: resList, Total: int32(total)}, nil
}

// GetBaseRole 获取角色
func (c *BaseRoleCase) GetBaseRole(ctx context.Context, id int64) (*admin.BaseRoleForm, error) {
	baseRole, err := c.FindById(ctx, id)
	if err != nil {
		return nil, err
	}
	res := c.formMapper.ToDTO(baseRole)
	res.Menus = _string.ConvertJsonStringToInt64Array(baseRole.Menus)
	return res, nil
}

// CreateBaseRole 创建角色
func (c *BaseRoleCase) CreateBaseRole(ctx context.Context, req *admin.BaseRoleForm) error {
	baseRole := c.formMapper.ToEntity(req)
	baseRole.Menus = _string.ConvertInt64ArrayToString(req.GetMenus())
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		err := c.Create(ctx, baseRole)
		if err != nil {
			return err
		}
		return c.casbinRuleCase.RebuildCasbinRuleByRole(ctx, baseRole)
	})
}

// UpdateBaseRole 更新角色
func (c *BaseRoleCase) UpdateBaseRole(ctx context.Context, req *admin.BaseRoleForm) error {
	oldBaseRole, err := c.FindById(ctx, req.GetId())
	if err != nil {
		return err
	}
	// 超级管理员角色不允许被修改。
	if oldBaseRole.Code == _const.BaseRoleCode_Super {
		return errorsx.PermissionDenied("更新角色失败，不能操作超级管理员角色")
	}

	baseRole := c.formMapper.ToEntity(req)
	baseRole.Menus = _string.ConvertInt64ArrayToString(req.GetMenus())
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		err = c.UpdateById(ctx, baseRole)
		if err != nil {
			return err
		}
		return c.casbinRuleCase.RebuildCasbinRuleByRole(ctx, baseRole)
	})
}

// DeleteBaseRole 删除角色
func (c *BaseRoleCase) DeleteBaseRole(ctx context.Context, id string) error {
	ids := _string.ConvertStringToInt64Array(id)
	query := c.Query(ctx).BaseRole

	opts := make([]repo.QueryOption, 0, 2)
	opts = append(opts, repo.Where(query.ID.In(ids...)))
	opts = append(opts, repo.Where(query.Code.Eq(_const.BaseRoleCode_Super)))
	count, err := c.Count(ctx, opts...)
	if err != nil {
		return errorsx.Internal("删除角色失败").WithCause(err)
	}
	// 命中超级管理员角色时，禁止继续删除。
	if count > 0 {
		return errorsx.PermissionDenied("删除角色失败，不能操作超级管理员角色")
	}

	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		err = c.DeleteByIds(ctx, ids)
		if err != nil {
			return err
		}
		return c.casbinRuleCase.DeleteCasbinRuleByRoleIds(ctx, ids)
	})
}

// SetBaseRoleStatus 设置角色状态
func (c *BaseRoleCase) SetBaseRoleStatus(ctx context.Context, req *common.SetStatusRequest) error {
	baseRole, err := c.FindById(ctx, req.GetId())
	if err != nil {
		return err
	}
	// 超级管理员角色不允许修改状态。
	if baseRole.Code == _const.BaseRoleCode_Super {
		return errorsx.PermissionDenied("设置状态失败，不能操作超级管理员角色")
	}
	return c.UpdateById(ctx, &models.BaseRole{
		ID:     req.GetId(),
		Status: req.GetStatus(),
	})
}

// SetBaseRoleMenus 设置角色菜单
func (c *BaseRoleCase) SetBaseRoleMenus(ctx context.Context, req *admin.SetMenusRequest) error {
	oldBaseRole, err := c.FindById(ctx, req.GetId())
	if err != nil {
		return err
	}
	// 超级管理员角色不允许调整菜单权限。
	if oldBaseRole.Code == _const.BaseRoleCode_Super {
		return errorsx.PermissionDenied("更新角色失败，不能操作超级管理员角色")
	}

	baseRole := &models.BaseRole{
		ID:    req.GetId(),
		Menus: _string.ConvertInt64ArrayToString(req.GetMenus()),
	}
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		err = c.UpdateById(ctx, baseRole)
		if err != nil {
			return err
		}
		baseRole.Code = oldBaseRole.Code
		return c.casbinRuleCase.RebuildCasbinRuleByRole(ctx, baseRole)
	})
}

// toBaseRole 转换角色响应数据
func (c *BaseRoleCase) toBaseRole(item *models.BaseRole) *admin.BaseRole {
	baseRole := c.mapper.ToDTO(item)
	baseRole.Menus = _string.ConvertJsonStringToInt64Array(item.Menus)
	return baseRole
}
