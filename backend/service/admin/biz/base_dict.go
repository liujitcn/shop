package biz

import (
	"context"
	"sort"

	adminv1 "shop/api/gen/go/admin/v1"
	"shop/pkg/biz"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
)

// BaseDictCase 字典业务实例
type BaseDictCase struct {
	*biz.BaseCase
	*data.BaseDictRepository
	baseDictItemCase *BaseDictItemCase
	formMapper       *mapper.CopierMapper[adminv1.BaseDictForm, models.BaseDict]
	mapper           *mapper.CopierMapper[adminv1.BaseDict, models.BaseDict]
}

// NewBaseDictCase 创建字典业务实例
func NewBaseDictCase(baseCase *biz.BaseCase, baseDictRepo *data.BaseDictRepository, baseDictItemCase *BaseDictItemCase) *BaseDictCase {
	return &BaseDictCase{
		BaseCase:           baseCase,
		BaseDictRepository: baseDictRepo,
		baseDictItemCase:   baseDictItemCase,
		formMapper:         mapper.NewCopierMapper[adminv1.BaseDictForm, models.BaseDict](),
		mapper:             mapper.NewCopierMapper[adminv1.BaseDict, models.BaseDict](),
	}
}

// OptionBaseDicts 查询字典下拉选择
func (c *BaseDictCase) OptionBaseDicts(ctx context.Context) (*adminv1.OptionBaseDictsResponse, error) {
	query := c.Query(ctx).BaseDict
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	baseDictList, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	baseDictItemQuery := c.baseDictItemCase.Query(ctx).BaseDictItem
	baseDictItemList := make([]*models.BaseDictItem, 0)
	itemOpts := make([]repository.QueryOption, 0, 2)
	itemOpts = append(itemOpts, repository.Order(baseDictItemQuery.Sort.Asc()))
	itemOpts = append(itemOpts, repository.Order(baseDictItemQuery.CreatedAt.Desc()))
	baseDictItemList, err = c.baseDictItemCase.List(ctx, itemOpts...)
	if err != nil {
		return nil, err
	}

	dictItemMap := make(map[int64][]*models.BaseDictItem)
	for _, item := range baseDictItemList {
		dictItemMap[item.DictID] = append(dictItemMap[item.DictID], item)
	}

	resList := make([]*adminv1.OptionBaseDictsResponse_BaseDict, 0, len(baseDictList))
	for _, dict := range baseDictList {
		items := make([]*adminv1.OptionBaseDictsResponse_BaseDictItem, 0)
		dictItems, ok := dictItemMap[dict.ID]
		// 当前字典存在子项时，按排序字段稳定输出字典项。
		if ok {
			sort.SliceStable(dictItems, func(i, j int) bool {
				return dictItems[i].Sort < dictItems[j].Sort
			})
			for _, dictItem := range dictItems {
				items = append(items, &adminv1.OptionBaseDictsResponse_BaseDictItem{
					Value:   dictItem.Value,
					Label:   dictItem.Label,
					TagType: dictItem.TagType,
				})
			}
		}
		resList = append(resList, &adminv1.OptionBaseDictsResponse_BaseDict{
			Code:  dict.Code,
			Name:  dict.Name,
			Items: items,
		})
	}
	return &adminv1.OptionBaseDictsResponse{BaseDicts: resList}, nil
}

// PageBaseDicts 分页查询字典
func (c *BaseDictCase) PageBaseDicts(ctx context.Context, req *adminv1.PageBaseDictsRequest) (*adminv1.PageBaseDictsResponse, error) {
	query := c.Query(ctx).BaseDict
	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	if req.Status != nil {
		opts = append(opts, repository.Where(query.Status.Eq(int32(req.GetStatus()))))
	}
	// 传入名称关键字时，按名称模糊匹配字典。
	if req.GetName() != "" {
		opts = append(opts, repository.Where(query.Name.Like("%"+req.GetName()+"%")))
	}
	// 传入编码关键字时，按编码模糊匹配字典。
	if req.GetCode() != "" {
		opts = append(opts, repository.Where(query.Code.Like("%"+req.GetCode()+"%")))
	}

	list, total, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}

	resList := make([]*adminv1.BaseDict, 0, len(list))
	for _, item := range list {
		baseDict := c.mapper.ToDTO(item)
		resList = append(resList, baseDict)
	}
	return &adminv1.PageBaseDictsResponse{BaseDicts: resList, Total: int32(total)}, nil
}

// GetBaseDict 获取字典
func (c *BaseDictCase) GetBaseDict(ctx context.Context, id int64) (*adminv1.BaseDictForm, error) {
	baseDict, err := c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	res := c.formMapper.ToDTO(baseDict)
	return res, nil
}

// CreateBaseDict 创建字典
func (c *BaseDictCase) CreateBaseDict(ctx context.Context, req *adminv1.BaseDictForm) error {
	baseDict := c.formMapper.ToEntity(req)
	err := c.Create(ctx, baseDict)
	if err != nil {
		// 命中字典编码唯一索引冲突时，返回稳定的业务冲突错误。
		if errorsx.IsMySQLDuplicateKey(err) {
			return errorsx.UniqueConflict("字典编码重复", "base_dict", "code", "unique_base_dict").WithCause(err)
		}
		return err
	}
	return nil
}

// UpdateBaseDict 更新字典
func (c *BaseDictCase) UpdateBaseDict(ctx context.Context, req *adminv1.BaseDictForm) error {
	baseDict := c.formMapper.ToEntity(req)
	err := c.UpdateByID(ctx, baseDict)
	if err != nil {
		// 命中字典编码唯一索引冲突时，返回稳定的业务冲突错误。
		if errorsx.IsMySQLDuplicateKey(err) {
			return errorsx.UniqueConflict("字典编码重复", "base_dict", "code", "unique_base_dict").WithCause(err)
		}
		return err
	}
	return nil
}

// DeleteBaseDict 删除字典
func (c *BaseDictCase) DeleteBaseDict(ctx context.Context, id string) error {
	ids := _string.ConvertStringToInt64Array(id)
	query := c.baseDictItemCase.Query(ctx).BaseDictItem
	for _, dictID := range ids {
		opts := make([]repository.QueryOption, 0, 1)
		opts = append(opts, repository.Where(query.DictID.Eq(dictID)))
		count, err := c.baseDictItemCase.Count(ctx, opts...)
		if err != nil {
			return errorsx.Internal("删除字典失败").WithCause(err)
		}
		// 字典下仍有子项时，不允许直接删除字典。
		if count > 0 {
			return errorsx.HasChildrenConflict("删除字典失败，下面有属性", "base_dict", "base_dict_item")
		}
	}
	return c.DeleteByIDs(ctx, ids)
}

// SetBaseDictStatus 设置字典状态
func (c *BaseDictCase) SetBaseDictStatus(ctx context.Context, req *adminv1.SetBaseDictStatusRequest) error {
	return c.UpdateByID(ctx, &models.BaseDict{
		ID:     req.GetId(),
		Status: req.GetStatus(),
	})
}
