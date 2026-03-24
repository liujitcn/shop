package biz

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"shop/api/gen/go/admin"
	"shop/api/gen/go/common"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/go-kratos/kratos/v2/log"
	_string "github.com/liujitcn/go-utils/string"
	_time "github.com/liujitcn/go-utils/time"
	"github.com/liujitcn/go-utils/trans"
	"github.com/liujitcn/gorm-kit/repo"
)

// BaseMenuCase 菜单业务实例
type BaseMenuCase struct {
	*biz.BaseCase
	tx data.Transaction
	*data.BaseMenuRepo
	casbinRuleCase *CasbinRuleCase
}

// NewBaseMenuCase 创建菜单业务实例
func NewBaseMenuCase(baseCase *biz.BaseCase, tx data.Transaction, baseMenuRepo *data.BaseMenuRepo, casbinRuleCase *CasbinRuleCase) *BaseMenuCase {
	return &BaseMenuCase{
		BaseCase:       baseCase,
		tx:             tx,
		BaseMenuRepo:   baseMenuRepo,
		casbinRuleCase: casbinRuleCase,
	}
}

// TreeBaseMenu 查询菜单树
func (c *BaseMenuCase) TreeBaseMenu(ctx context.Context) (*admin.TreeBaseMenuResponse, error) {
	list, err := c.List(ctx)
	if err != nil {
		return nil, err
	}
	return &admin.TreeBaseMenuResponse{List: c.buildBaseMenuTree(list, 0)}, nil
}

// OptionBaseMenu 查询菜单选项
func (c *BaseMenuCase) OptionBaseMenu(ctx context.Context, req *admin.OptionBaseMenuRequest) (*common.TreeOptionResponse, error) {
	list, err := c.List(ctx)
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
	return c.toBaseMenuForm(baseMenu), nil
}

// CreateBaseMenu 创建菜单
func (c *BaseMenuCase) CreateBaseMenu(ctx context.Context, req *admin.BaseMenuForm) error {
	return c.Create(ctx, c.toBaseMenuModel(req))
}

// UpdateBaseMenu 更新菜单
func (c *BaseMenuCase) UpdateBaseMenu(ctx context.Context, req *admin.BaseMenuForm) error {
	baseMenu := c.toBaseMenuModel(req)
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		var err error
		err = c.UpdateById(ctx, baseMenu)
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
		count, err := c.Count(ctx, repo.Where(query.ParentID.Eq(menuId)))
		if err != nil {
			return err
		}
		if count > 0 {
			return errors.New("删除菜单失败,下面有菜单")
		}
	}
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		var err error
		err = c.DeleteByIds(ctx, ids)
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
		if menu.ParentID != parentId {
			continue
		}

		route := &admin.RouteItem{
			Path:      &menu.Path,
			Redirect:  &menu.Redirect,
			Name:      &menu.Name,
			Component: &menu.Component,
		}

		var meta admin.RouteMeta
		if strings.TrimSpace(menu.Meta) != "" {
			if err := json.Unmarshal([]byte(menu.Meta), &meta); err != nil {
				log.Errorf("解析菜单路由元信息失败[%s]", err.Error())
				continue
			}
			route.Meta = &meta
		}

		route.Children = c.buildRouteTree(menuList, menu.ID)
		list = append(list, route)
	}
	return list
}

// buildBaseMenuTree 构建菜单树
func (c *BaseMenuCase) buildBaseMenuTree(menuList []*models.BaseMenu, parentId int64) []*admin.BaseMenu {
	res := make([]*admin.BaseMenu, 0)
	for _, item := range menuList {
		if item.ParentID != parentId {
			continue
		}

		var meta admin.BaseMenuMeta
		if err := json.Unmarshal([]byte(item.Meta), &meta); err != nil {
			log.Errorf("解析菜单元信息失败[%s]", err.Error())
			continue
		}

		menu := &admin.BaseMenu{
			Id:        item.ID,
			ParentId:  item.ParentID,
			Type:      common.BaseMenuType(item.Type),
			Path:      item.Path,
			Name:      item.Name,
			Component: item.Component,
			Redirect:  item.Redirect,
			Meta:      &meta,
			Sort:      item.Sort,
			Status:    common.Status(item.Status),
			CreatedAt: _time.TimeToTimeString(item.CreatedAt),
			UpdatedAt: _time.TimeToTimeString(item.UpdatedAt),
		}
		menu.Children = c.buildBaseMenuTree(menuList, item.ID)
		res = append(res, menu)
	}
	return res
}

// buildBaseMenuOption 构建菜单选项树
func (c *BaseMenuCase) buildBaseMenuOption(menuList []*models.BaseMenu, parentId int64) []*common.TreeOptionResponse_Option {
	res := make([]*common.TreeOptionResponse_Option, 0)
	for _, item := range menuList {
		if item.ParentID != parentId {
			continue
		}

		var meta admin.RouteMeta
		if err := json.Unmarshal([]byte(item.Meta), &meta); err != nil {
			log.Errorf("解析菜单选项元信息失败[%s]", err.Error())
			continue
		}

		label := item.Name
		if meta.Title != nil && *meta.Title != "" {
			label = *meta.Title
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

// toBaseMenuForm 转换菜单表单数据
func (c *BaseMenuCase) toBaseMenuForm(item *models.BaseMenu) *admin.BaseMenuForm {
	var meta admin.BaseMenuMeta
	if err := json.Unmarshal([]byte(item.Meta), &meta); err != nil {
		log.Errorf("解析菜单表单元信息失败[%s]", err.Error())
	}
	return &admin.BaseMenuForm{
		Id:        item.ID,
		ParentId:  trans.Int64(item.ParentID),
		Type:      trans.Enum(common.BaseMenuType(item.Type)),
		Path:      item.Path,
		Name:      item.Name,
		Component: item.Component,
		Redirect:  item.Redirect,
		Meta:      &meta,
		Apis:      _string.ConvertJsonStringToStringArray(item.Apis),
		Sort:      item.Sort,
		Status:    trans.Enum(common.Status(item.Status)),
	}
}

// toBaseMenuModel 转换菜单模型数据
func (c *BaseMenuCase) toBaseMenuModel(item *admin.BaseMenuForm) *models.BaseMenu {
	metaBytes, err := json.Marshal(item.GetMeta())
	if err != nil {
		log.Errorf("序列化菜单元信息失败[%s]", err.Error())
	}
	return &models.BaseMenu{
		ID:        item.GetId(),
		ParentID:  item.GetParentId(),
		Type:      int32(item.GetType()),
		Path:      item.GetPath(),
		Name:      item.GetName(),
		Component: item.GetComponent(),
		Redirect:  item.GetRedirect(),
		Meta:      string(metaBytes),
		Apis:      _string.ConvertStringArrayToString(item.GetApis()),
		Sort:      item.GetSort(),
		Status:    int32(item.GetStatus()),
	}
}
