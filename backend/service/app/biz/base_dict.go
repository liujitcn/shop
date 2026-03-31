package biz

import (
	"context"
	"sort"
	"strings"

	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"shop/api/gen/go/app"
	"shop/api/gen/go/common"

	"github.com/liujitcn/gorm-kit/repo"
)

// BaseDictCase 字典业务处理对象
type BaseDictCase struct {
	*biz.BaseCase
	*data.BaseDictRepo
	baseDictItemCase *BaseDictItemCase
}

// NewBaseDictCase 创建字典业务处理对象
func NewBaseDictCase(baseCase *biz.BaseCase, baseDictRepo *data.BaseDictRepo, baseDictItemCase *BaseDictItemCase) *BaseDictCase {
	return &BaseDictCase{
		BaseCase:         baseCase,
		BaseDictRepo:     baseDictRepo,
		baseDictItemCase: baseDictItemCase,
	}
}

// ListBaseDict 查询字典列表
func (c *BaseDictCase) ListBaseDict(ctx context.Context, codes string) (*app.ListBaseDictResponse, error) {
	query := c.Query(ctx).BaseDict
	opts := make([]repo.QueryOption, 0, 3)
	opts = append(opts, repo.Order(query.UpdatedAt.Desc()))
	opts = append(opts, repo.Where(query.Code.In(strings.Split(codes, ",")...)))
	opts = append(opts, repo.Where(query.Status.Eq(int32(common.Status_ENABLE))))
	baseDictList, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	dictIds := make([]int64, 0, len(baseDictList))
	for _, dict := range baseDictList {
		dictIds = append(dictIds, dict.ID)
	}

	var baseDictItemList []*models.BaseDictItem
	baseDictItemList, err = c.baseDictItemCase.findByDictIds(ctx, dictIds)
	if err != nil {
		return nil, err
	}

	dictItemMap := make(map[int64][]*models.BaseDictItem)
	for _, item := range baseDictItemList {
		dictItemMap[item.DictID] = append(dictItemMap[item.DictID], item)
	}

	list := make([]*app.ListBaseDictResponse_Dict, 0, len(baseDictList))
	for _, dict := range baseDictList {
		items := make([]*app.ListBaseDictResponse_DictItem, 0)
		if dictItems, ok := dictItemMap[dict.ID]; ok {
			sort.SliceStable(dictItems, func(i, j int) bool {
				return dictItems[i].Sort < dictItems[j].Sort
			})
			for _, dictItem := range dictItems {
				items = append(items, &app.ListBaseDictResponse_DictItem{
					Value: dictItem.Value,
					Label: dictItem.Label,
				})
			}
		}

		list = append(list, &app.ListBaseDictResponse_Dict{
			Code:  dict.Code,
			Name:  dict.Name,
			Items: items,
		})
	}

	return &app.ListBaseDictResponse{
		List: list,
	}, nil
}
