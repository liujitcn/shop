package biz

import (
	"context"
	"sort"

	"shop/api/gen/go/admin"
	"shop/api/gen/go/common"
	"shop/pkg/biz"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repo"
)

// BaseDictCase 字典业务实例
type BaseDictCase struct {
	*biz.BaseCase
	*data.BaseDictRepo
	baseDictItemCase *BaseDictItemCase
	formMapper       *mapper.CopierMapper[admin.BaseDictForm, models.BaseDict]
	mapper           *mapper.CopierMapper[admin.BaseDict, models.BaseDict]
}

// NewBaseDictCase 创建字典业务实例
func NewBaseDictCase(baseCase *biz.BaseCase, baseDictRepo *data.BaseDictRepo, baseDictItemCase *BaseDictItemCase) *BaseDictCase {
	return &BaseDictCase{
		BaseCase:         baseCase,
		BaseDictRepo:     baseDictRepo,
		baseDictItemCase: baseDictItemCase,
		formMapper:       mapper.NewCopierMapper[admin.BaseDictForm, models.BaseDict](),
		mapper:           mapper.NewCopierMapper[admin.BaseDict, models.BaseDict](),
	}
}

// OptionBaseDict 查询字典下拉选择
func (c *BaseDictCase) OptionBaseDict(ctx context.Context) (*admin.OptionBaseDictResponse, error) {
	query := c.Query(ctx).BaseDict
	opts := make([]repo.QueryOption, 0, 1)
	opts = append(opts, repo.Order(query.CreatedAt.Desc()))
	baseDictList, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	baseDictItemQuery := c.baseDictItemCase.Query(ctx).BaseDictItem
	baseDictItemList := make([]*models.BaseDictItem, 0)
	itemOpts := make([]repo.QueryOption, 0, 2)
	itemOpts = append(itemOpts, repo.Order(baseDictItemQuery.Sort.Asc()))
	itemOpts = append(itemOpts, repo.Order(baseDictItemQuery.CreatedAt.Desc()))
	baseDictItemList, err = c.baseDictItemCase.List(ctx, itemOpts...)
	if err != nil {
		return nil, err
	}

	dictItemMap := make(map[int64][]*models.BaseDictItem)
	for _, item := range baseDictItemList {
		dictItemMap[item.DictID] = append(dictItemMap[item.DictID], item)
	}

	resList := make([]*admin.OptionBaseDictResponse_BaseDict, 0, len(baseDictList))
	for _, dict := range baseDictList {
		items := make([]*admin.OptionBaseDictResponse_BaseDictItem, 0)
		dictItems, ok := dictItemMap[dict.ID]
		// 当前字典存在子项时，按排序字段稳定输出字典项。
		if ok {
			sort.SliceStable(dictItems, func(i, j int) bool {
				return dictItems[i].Sort < dictItems[j].Sort
			})
			for _, dictItem := range dictItems {
				items = append(items, &admin.OptionBaseDictResponse_BaseDictItem{
					Value:   dictItem.Value,
					Label:   dictItem.Label,
					TagType: dictItem.TagType,
				})
			}
		}
		resList = append(resList, &admin.OptionBaseDictResponse_BaseDict{
			Code:  dict.Code,
			Name:  dict.Name,
			Items: items,
		})
	}
	return &admin.OptionBaseDictResponse{List: resList}, nil
}

// PageBaseDict 分页查询字典
func (c *BaseDictCase) PageBaseDict(ctx context.Context, req *admin.PageBaseDictRequest) (*admin.PageBaseDictResponse, error) {
	query := c.Query(ctx).BaseDict
	opts := make([]repo.QueryOption, 0, 4)
	opts = append(opts, repo.Order(query.CreatedAt.Desc()))
	if req.Status != nil {
		opts = append(opts, repo.Where(query.Status.Eq(int32(req.GetStatus()))))
	}
	// 传入名称关键字时，按名称模糊匹配字典。
	if req.GetName() != "" {
		opts = append(opts, repo.Where(query.Name.Like("%"+req.GetName()+"%")))
	}
	// 传入编码关键字时，按编码模糊匹配字典。
	if req.GetCode() != "" {
		opts = append(opts, repo.Where(query.Code.Like("%"+req.GetCode()+"%")))
	}

	list, total, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}

	resList := make([]*admin.BaseDict, 0, len(list))
	for _, item := range list {
		baseDict := c.mapper.ToDTO(item)
		resList = append(resList, baseDict)
	}
	return &admin.PageBaseDictResponse{List: resList, Total: int32(total)}, nil
}

// GetBaseDict 获取字典
func (c *BaseDictCase) GetBaseDict(ctx context.Context, id int64) (*admin.BaseDictForm, error) {
	baseDict, err := c.FindById(ctx, id)
	if err != nil {
		return nil, err
	}
	res := c.formMapper.ToDTO(baseDict)
	return res, nil
}

// CreateBaseDict 创建字典
func (c *BaseDictCase) CreateBaseDict(ctx context.Context, req *admin.BaseDictForm) error {
	baseDict := c.formMapper.ToEntity(req)
	return c.Create(ctx, baseDict)
}

// UpdateBaseDict 更新字典
func (c *BaseDictCase) UpdateBaseDict(ctx context.Context, req *admin.BaseDictForm) error {
	baseDict := c.formMapper.ToEntity(req)
	return c.UpdateById(ctx, baseDict)
}

// DeleteBaseDict 删除字典
func (c *BaseDictCase) DeleteBaseDict(ctx context.Context, id string) error {
	ids := _string.ConvertStringToInt64Array(id)
	query := c.baseDictItemCase.Query(ctx).BaseDictItem
	for _, dictId := range ids {
		opts := make([]repo.QueryOption, 0, 1)
		opts = append(opts, repo.Where(query.DictID.Eq(dictId)))
		count, err := c.baseDictItemCase.Count(ctx, opts...)
		if err != nil {
			return errorsx.Internal("删除字典失败").WithCause(err)
		}
		// 字典下仍有子项时，不允许直接删除字典。
		if count > 0 {
			return errorsx.HasChildrenConflict("删除字典失败，下面有属性", "base_dict", "base_dict_item")
		}
	}
	return c.DeleteByIds(ctx, ids)
}

// SetBaseDictStatus 设置字典状态
func (c *BaseDictCase) SetBaseDictStatus(ctx context.Context, req *common.SetStatusRequest) error {
	return c.UpdateById(ctx, &models.BaseDict{
		ID:     req.GetId(),
		Status: req.GetStatus(),
	})
}
