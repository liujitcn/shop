package biz

import (
	"context"
	"sort"

	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"shop/api/gen/go/app"
	"shop/api/gen/go/common"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repo"
)

// BaseDictCase 字典业务处理对象
type BaseDictCase struct {
	*biz.BaseCase
	*data.BaseDictRepo
	baseDictItemCase *BaseDictItemCase
	dictMapper       *mapper.CopierMapper[app.BaseDictForm, models.BaseDict]
	itemMapper       *mapper.CopierMapper[app.BaseDictForm_DictItem, models.BaseDictItem]
}

// NewBaseDictCase 创建字典业务处理对象
func NewBaseDictCase(baseCase *biz.BaseCase, baseDictRepo *data.BaseDictRepo, baseDictItemCase *BaseDictItemCase) *BaseDictCase {
	return &BaseDictCase{
		BaseCase:         baseCase,
		BaseDictRepo:     baseDictRepo,
		baseDictItemCase: baseDictItemCase,
		dictMapper:       mapper.NewCopierMapper[app.BaseDictForm, models.BaseDict](),
		itemMapper:       mapper.NewCopierMapper[app.BaseDictForm_DictItem, models.BaseDictItem](),
	}
}

// GetBaseDict 查询字典
func (c *BaseDictCase) GetBaseDict(ctx context.Context, code string) (*app.BaseDictForm, error) {
	query := c.Query(ctx).BaseDict
	opts := make([]repo.QueryOption, 0, 3)
	opts = append(opts, repo.Order(query.CreatedAt.Desc()))
	opts = append(opts, repo.Where(query.Code.Eq(code)))
	opts = append(opts, repo.Where(query.Status.Eq(int32(common.Status_ENABLE))))
	baseDict, err := c.Find(ctx, opts...)
	if err != nil {
		return nil, err
	}

	var baseDictItemList []*models.BaseDictItem
	baseDictItemList, err = c.baseDictItemCase.findByDictIds(ctx, []int64{baseDict.ID})
	if err != nil {
		return nil, err
	}

	dictItemMap := make(map[int64][]*models.BaseDictItem)
	for _, item := range baseDictItemList {
		dictItemMap[item.DictID] = append(dictItemMap[item.DictID], item)
	}

	items := make([]*app.BaseDictForm_DictItem, 0)
	// 命中字典项映射时，再按排序规则组装当前字典的子项。
	if dictItems, ok := dictItemMap[baseDict.ID]; ok {
		sort.SliceStable(dictItems, func(i, j int) bool {
			return dictItems[i].Sort < dictItems[j].Sort
		})
		for _, dictItem := range dictItems {
			items = append(items, c.itemMapper.ToDTO(dictItem))
		}
	}

	res := c.dictMapper.ToDTO(baseDict)
	res.Items = items
	return res, nil
}
