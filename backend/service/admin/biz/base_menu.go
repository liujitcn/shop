package biz

import (
	"context"
	"errors"
	"shop/api/gen/go/admin"
	"shop/api/gen/go/common"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	_mapper "github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repo"
)

// BaseMenuCase 菜单业务实例
type BaseMenuCase struct {
	*biz.BaseCase
	tx data.Transaction
	*data.BaseMenuRepo
	casbinRuleCase *CasbinRuleCase
	formMapper     *_mapper.CopierMapper[admin.BaseMenuForm, models.BaseMenu]
	mapper         *_mapper.CopierMapper[admin.BaseMenu, models.BaseMenu]
	routerMapper   *_mapper.CopierMapper[admin.RouteItem, models.BaseMenu]
}

// NewBaseMenuCase 创建菜单业务实例
func NewBaseMenuCase(baseCase *biz.BaseCase, tx data.Transaction, baseMenuRepo *data.BaseMenuRepo, casbinRuleCase *CasbinRuleCase) *BaseMenuCase {
	formMapper := _mapper.NewCopierMapper[admin.BaseMenuForm, models.BaseMenu]()
	formMapper.AppendConverters(_mapper.NewJSONTypeConverter[*admin.BaseMenuMeta]().NewConverterPair())
	mapper := _mapper.NewCopierMapper[admin.BaseMenu, models.BaseMenu]()
	mapper.AppendConverters(_mapper.NewJSONTypeConverter[*admin.BaseMenuMeta]().NewConverterPair())
	routerMapper := _mapper.NewCopierMapper[admin.RouteItem, models.BaseMenu]()
	routerMapper.AppendConverters(_mapper.NewJSONTypeConverter[*admin.RouteMeta]().NewConverterPair())
	return &BaseMenuCase{
		BaseCase:       baseCase,
		tx:             tx,
		BaseMenuRepo:   baseMenuRepo,
		casbinRuleCase: casbinRuleCase,
		formMapper:     formMapper,
		mapper:         mapper,
		routerMapper:   routerMapper,
	}
}

// TreeBaseMenu 查询菜单树
func (c *BaseMenuCase) TreeBaseMenu(ctx context.Context) (*admin.TreeBaseMenuResponse, error) {
	query := c.Query(ctx).BaseMenu
	opts := make([]repo.QueryOption, 0, 2)
	opts = append(opts, repo.Order(query.Sort.Asc()))
	opts = append(opts, repo.Order(query.CreatedAt.Desc()))
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return &admin.TreeBaseMenuResponse{List: c.buildBaseMenuTree(list, 0)}, nil
}

// OptionBaseMenu 查询菜单选项
func (c *BaseMenuCase) OptionBaseMenu(ctx context.Context, req *admin.OptionBaseMenuRequest) (*common.TreeOptionResponse, error) {
	query := c.Query(ctx).BaseMenu
	opts := make([]repo.QueryOption, 0, 2)
	opts = append(opts, repo.Order(query.Sort.Asc()))
	opts = append(opts, repo.Order(query.CreatedAt.Desc()))
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return &common.TreeOptionResponse{List: c.buildBaseMenuOption(list, req.GetParentId())}, nil
}

// GetBaseMenu 获取菜单
func (c *BaseMenuCase) GetBaseMenu(ctx context.Context, id int64) (*admin.BaseMenuForm, error) {
	baseMenu, err := c.FindById(ctx, id)
	if err != nil {
		return nil, err
	}
	return c.formMapper.ToDTO(baseMenu), nil
}

// CreateBaseMenu 创建菜单
func (c *BaseMenuCase) CreateBaseMenu(ctx context.Context, req *admin.BaseMenuForm) error {
	baseMenu := c.formMapper.ToEntity(req)
	return c.Create(ctx, baseMenu)
}

// UpdateBaseMenu 更新菜单
func (c *BaseMenuCase) UpdateBaseMenu(ctx context.Context, req *admin.BaseMenuForm) error {
	baseMenu := c.formMapper.ToEntity(req)
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		err := c.UpdateById(ctx, baseMenu)
		if err != nil {
			return err
		}
		return c.casbinRuleCase.RebuildCasbinRuleByMenuId(ctx, baseMenu.ID)
	})
}

// DeleteBaseMenu 删除菜单
func (c *BaseMenuCase) DeleteBaseMenu(ctx context.Context, id string) error {
	ids := _string.ConvertStringToInt64Array(id)
	query := c.Query(ctx).BaseMenu
	for _, menuId := range ids {
		opts := make([]repo.QueryOption, 0, 1)
		opts = append(opts, repo.Where(query.ParentID.Eq(menuId)))
		count, err := c.Count(ctx, opts...)
		if err != nil {
			return err
		}
		// 仍然存在子菜单时，禁止删除当前节点。
		if count > 0 {
			return errors.New("删除菜单失败,下面有菜单")
		}
	}
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		err := c.DeleteByIds(ctx, ids)
		if err != nil {
			return err
		}
		return c.casbinRuleCase.DeleteCasbinRuleByMenuIds(ctx, ids)
	})
}

// SetBaseMenuStatus 设置菜单状态
func (c *BaseMenuCase) SetBaseMenuStatus(ctx context.Context, req *common.SetStatusRequest) error {
	return c.UpdateById(ctx, &models.BaseMenu{
		ID:     req.GetId(),
		Status: req.GetStatus(),
	})
}

// buildRouteTree 构建菜单路由树
func (c *BaseMenuCase) buildRouteTree(menuList []*models.BaseMenu, parentId int64) []*admin.RouteItem {
	list := make([]*admin.RouteItem, 0)
	for _, menu := range menuList {
		// 非当前父节点的菜单不参与当前层级路由构建。
		if menu.ParentID != parentId {
			continue
		}

		route := c.routerMapper.ToDTO(menu)

		route.Children = c.buildRouteTree(menuList, menu.ID)
		list = append(list, route)
	}
	return list
}

// buildBaseMenuTree 构建菜单树
func (c *BaseMenuCase) buildBaseMenuTree(menuList []*models.BaseMenu, parentId int64) []*admin.BaseMenu {
	res := make([]*admin.BaseMenu, 0)
	for _, item := range menuList {
		// 非当前父节点的菜单不参与当前层级树构建。
		if item.ParentID != parentId {
			continue
		}
		menu := c.mapper.ToDTO(item)
		menu.Children = c.buildBaseMenuTree(menuList, item.ID)
		res = append(res, menu)
	}
	return res
}

// buildBaseMenuOption 构建菜单选项树
func (c *BaseMenuCase) buildBaseMenuOption(menuList []*models.BaseMenu, parentId int64) []*common.TreeOptionResponse_Option {
	res := make([]*common.TreeOptionResponse_Option, 0)
	for _, item := range menuList {
		// 非当前父节点的菜单不参与当前层级选项构建。
		if item.ParentID != parentId {
			continue
		}

		label := item.Name
		route := c.routerMapper.ToDTO(item)
		// 路由元信息存在标题时，优先使用前端路由标题作为展示名称。
		if route != nil && route.GetMeta() != nil && route.GetMeta().GetTitle() != "" {
			label = route.GetMeta().GetTitle()
		}

		menu := &common.TreeOptionResponse_Option{
			Label: label,
			Value: item.ID,
		}
		menu.Children = c.buildBaseMenuOption(menuList, item.ID)
		res = append(res, menu)
	}
	return res
}
