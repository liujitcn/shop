package biz

import (
	"context"
	adminv1 "shop/api/gen/go/admin/v1"
	commonv1 "shop/api/gen/go/common/v1"
	"shop/pkg/biz"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	_mapper "github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
)

// BaseMenuCase 菜单业务实例
type BaseMenuCase struct {
	*biz.BaseCase
	tx data.Transaction
	*data.BaseMenuRepository
	casbinRuleCase *CasbinRuleCase
	formMapper     *_mapper.CopierMapper[adminv1.BaseMenuForm, models.BaseMenu]
	mapper         *_mapper.CopierMapper[adminv1.BaseMenu, models.BaseMenu]
	routerMapper   *_mapper.CopierMapper[adminv1.RouteItem, models.BaseMenu]
}

// NewBaseMenuCase 创建菜单业务实例
func NewBaseMenuCase(baseCase *biz.BaseCase, tx data.Transaction, baseMenuRepo *data.BaseMenuRepository, casbinRuleCase *CasbinRuleCase) *BaseMenuCase {
	formMapper := _mapper.NewCopierMapper[adminv1.BaseMenuForm, models.BaseMenu]()
	formMapper.AppendConverters(_mapper.NewJSONTypeConverter[*adminv1.BaseMenuMeta]().NewConverterPair())
	mapper := _mapper.NewCopierMapper[adminv1.BaseMenu, models.BaseMenu]()
	mapper.AppendConverters(_mapper.NewJSONTypeConverter[*adminv1.BaseMenuMeta]().NewConverterPair())
	routerMapper := _mapper.NewCopierMapper[adminv1.RouteItem, models.BaseMenu]()
	routerMapper.AppendConverters(_mapper.NewJSONTypeConverter[*adminv1.RouteMeta]().NewConverterPair())
	return &BaseMenuCase{
		BaseCase:           baseCase,
		tx:                 tx,
		BaseMenuRepository: baseMenuRepo,
		casbinRuleCase:     casbinRuleCase,
		formMapper:         formMapper,
		mapper:             mapper,
		routerMapper:       routerMapper,
	}
}

// TreeBaseMenus 查询菜单树
func (c *BaseMenuCase) TreeBaseMenus(ctx context.Context) (*adminv1.TreeBaseMenusResponse, error) {
	query := c.Query(ctx).BaseMenu
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Order(query.Sort.Asc()))
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return &adminv1.TreeBaseMenusResponse{BaseMenus: c.buildBaseMenuTree(list, 0)}, nil
}

// OptionBaseMenus 查询菜单选项
func (c *BaseMenuCase) OptionBaseMenus(ctx context.Context, req *adminv1.OptionBaseMenusRequest) (*commonv1.TreeOptionResponse, error) {
	query := c.Query(ctx).BaseMenu
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Order(query.Sort.Asc()))
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return &commonv1.TreeOptionResponse{List: c.buildBaseMenuOption(list, req.GetParentId())}, nil
}

// GetBaseMenu 获取菜单
func (c *BaseMenuCase) GetBaseMenu(ctx context.Context, id int64) (*adminv1.BaseMenuForm, error) {
	baseMenu, err := c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return c.formMapper.ToDTO(baseMenu), nil
}

// CreateBaseMenu 创建菜单
func (c *BaseMenuCase) CreateBaseMenu(ctx context.Context, req *adminv1.BaseMenuForm) error {
	baseMenu := c.formMapper.ToEntity(req)
	return c.Create(ctx, baseMenu)
}

// UpdateBaseMenu 更新菜单
func (c *BaseMenuCase) UpdateBaseMenu(ctx context.Context, req *adminv1.BaseMenuForm) error {
	baseMenu := c.formMapper.ToEntity(req)
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		err := c.UpdateByID(ctx, baseMenu)
		if err != nil {
			return err
		}
		return c.casbinRuleCase.RebuildCasbinRuleByMenuID(ctx, baseMenu.ID)
	})
}

// DeleteBaseMenu 删除菜单
func (c *BaseMenuCase) DeleteBaseMenu(ctx context.Context, id string) error {
	ids := _string.ConvertStringToInt64Array(id)
	query := c.Query(ctx).BaseMenu
	for _, menuID := range ids {
		opts := make([]repository.QueryOption, 0, 1)
		opts = append(opts, repository.Where(query.ParentID.Eq(menuID)))
		count, err := c.Count(ctx, opts...)
		if err != nil {
			return err
		}
		// 仍然存在子菜单时，禁止删除当前节点。
		if count > 0 {
			return errorsx.HasChildrenConflict("删除菜单失败，下面有菜单", "base_menu", "base_menu")
		}
	}
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		deleteErr := c.DeleteByIDs(ctx, ids)
		if deleteErr != nil {
			return deleteErr
		}
		return c.casbinRuleCase.DeleteCasbinRuleByMenuIDs(ctx, ids)
	})
}

// SetBaseMenuStatus 设置菜单状态
func (c *BaseMenuCase) SetBaseMenuStatus(ctx context.Context, req *adminv1.SetBaseMenuStatusRequest) error {
	return c.UpdateByID(ctx, &models.BaseMenu{
		ID:     req.GetId(),
		Status: req.GetStatus(),
	})
}

// buildRouteTree 构建菜单路由树
func (c *BaseMenuCase) buildRouteTree(menuList []*models.BaseMenu, parentID int64) []*adminv1.RouteItem {
	list := make([]*adminv1.RouteItem, 0)
	for _, menu := range menuList {
		// 非当前父节点的菜单不参与当前层级路由构建。
		if menu.ParentID != parentID {
			continue
		}

		route := c.routerMapper.ToDTO(menu)

		route.Children = c.buildRouteTree(menuList, menu.ID)
		list = append(list, route)
	}
	return list
}

// buildBaseMenuTree 构建菜单树
func (c *BaseMenuCase) buildBaseMenuTree(menuList []*models.BaseMenu, parentID int64) []*adminv1.BaseMenu {
	res := make([]*adminv1.BaseMenu, 0)
	for _, item := range menuList {
		// 非当前父节点的菜单不参与当前层级树构建。
		if item.ParentID != parentID {
			continue
		}
		menu := c.mapper.ToDTO(item)
		menu.Children = c.buildBaseMenuTree(menuList, item.ID)
		res = append(res, menu)
	}
	return res
}

// buildBaseMenuOption 构建菜单选项树
func (c *BaseMenuCase) buildBaseMenuOption(menuList []*models.BaseMenu, parentID int64) []*commonv1.TreeOptionResponse_Option {
	res := make([]*commonv1.TreeOptionResponse_Option, 0)
	for _, item := range menuList {
		// 非当前父节点的菜单不参与当前层级选项构建。
		if item.ParentID != parentID {
			continue
		}

		label := item.Name
		route := c.routerMapper.ToDTO(item)
		// 路由元信息存在标题时，优先使用前端路由标题作为展示名称。
		if route != nil && route.GetMeta() != nil && route.GetMeta().GetTitle() != "" {
			label = route.GetMeta().GetTitle()
		}

		menu := &commonv1.TreeOptionResponse_Option{
			Label: label,
			Value: item.ID,
		}
		menu.Children = c.buildBaseMenuOption(menuList, item.ID)
		res = append(res, menu)
	}
	return res
}
