package biz

import (
	"context"

	"shop/api/gen/go/admin"
	"shop/api/gen/go/common"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repo"
)

// BaseDictItemCase 字典项业务实例
type BaseDictItemCase struct {
	*biz.BaseCase
	baseDictRepo *data.BaseDictRepo
	*data.BaseDictItemRepo
	formMapper *mapper.CopierMapper[admin.BaseDictItemForm, models.BaseDictItem]
	mapper     *mapper.CopierMapper[admin.BaseDictItem, models.BaseDictItem]
}

// NewBaseDictItemCase 创建字典项业务实例
func NewBaseDictItemCase(baseCase *biz.BaseCase, baseDictRepo *data.BaseDictRepo, baseDictItemRepo *data.BaseDictItemRepo) *BaseDictItemCase {
	return &BaseDictItemCase{
		BaseCase:         baseCase,
		baseDictRepo:     baseDictRepo,
		BaseDictItemRepo: baseDictItemRepo,
		formMapper:       mapper.NewCopierMapper[admin.BaseDictItemForm, models.BaseDictItem](),
		mapper:           mapper.NewCopierMapper[admin.BaseDictItem, models.BaseDictItem](),
	}
}

// PageBaseDictItem 分页查询字典项
func (c *BaseDictItemCase) PageBaseDictItem(ctx context.Context, req *admin.PageBaseDictItemRequest) (*admin.PageBaseDictItemResponse, error) {
	query := c.Query(ctx).BaseDictItem
	opts := make([]repo.QueryOption, 0, 5)
	opts = append(opts, repo.Order(query.Sort.Asc()))
	opts = append(opts, repo.Order(query.CreatedAt.Desc()))
	if req.GetDictId() > 0 {
		opts = append(opts, repo.Where(query.DictID.Eq(req.GetDictId())))
	}
	if req.Status != nil {
		opts = append(opts, repo.Where(query.Status.Eq(int32(req.GetStatus()))))
	}
	if req.GetLabel() != "" {
		opts = append(opts, repo.Where(query.Label.Like("%"+req.GetLabel()+"%")))
	}

	list, total, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}

	resList := make([]*admin.BaseDictItem, 0, len(list))
	for _, item := range list {
		baseDictItem := c.mapper.ToDTO(item)
		resList = append(resList, baseDictItem)
	}
	return &admin.PageBaseDictItemResponse{List: resList, Total: int32(total)}, nil
}

// GetBaseDictItem 获取字典项
func (c *BaseDictItemCase) GetBaseDictItem(ctx context.Context, id int64) (*admin.BaseDictItemForm, error) {
	baseDictItem, err := c.FindById(ctx, id)
	if err != nil {
		return nil, err
	}
	res := c.formMapper.ToDTO(baseDictItem)
	return res, nil
}

// CreateBaseDictItem 创建字典项
func (c *BaseDictItemCase) CreateBaseDictItem(ctx context.Context, req *admin.BaseDictItemForm) error {
	baseDictItem := c.formMapper.ToEntity(req)
	return c.Create(ctx, baseDictItem)
}

// UpdateBaseDictItem 更新字典项
func (c *BaseDictItemCase) UpdateBaseDictItem(ctx context.Context, req *admin.BaseDictItemForm) error {
	baseDictItem := c.formMapper.ToEntity(req)
	return c.UpdateById(ctx, baseDictItem)
}

// DeleteBaseDictItem 删除字典项
func (c *BaseDictItemCase) DeleteBaseDictItem(ctx context.Context, id string) error {
	return c.DeleteByIds(ctx, _string.ConvertStringToInt64Array(id))
}

// SetBaseDictItemStatus 设置字典项状态
func (c *BaseDictItemCase) SetBaseDictItemStatus(ctx context.Context, req *common.SetStatusRequest) error {
	return c.UpdateById(ctx, &models.BaseDictItem{
		ID:     req.GetId(),
		Status: req.GetStatus(),
	})
}
