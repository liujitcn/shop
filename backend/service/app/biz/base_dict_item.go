package biz

import (
	"context"

	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"shop/api/gen/go/common"

	"github.com/liujitcn/gorm-kit/repo"
)

// BaseDictItemCase 字典项业务处理对象
type BaseDictItemCase struct {
	*biz.BaseCase
	*data.BaseDictItemRepo
	baseDictRepo *data.BaseDictRepo
}

// NewBaseDictItemCase 创建字典项业务处理对象
func NewBaseDictItemCase(baseCase *biz.BaseCase, baseDictRepo *data.BaseDictRepo, baseDictItemRepo *data.BaseDictItemRepo) *BaseDictItemCase {
	return &BaseDictItemCase{
		BaseCase:         baseCase,
		baseDictRepo:     baseDictRepo,
		BaseDictItemRepo: baseDictItemRepo,
	}
}

// 按字典编号列表查询启用中的字典项
func (c *BaseDictItemCase) findByDictIds(ctx context.Context, dictIds []int64) ([]*models.BaseDictItem, error) {
	query := c.Query(ctx).BaseDictItem
	opts := make([]repo.QueryOption, 0, 4)
	opts = append(opts, repo.Order(query.Sort.Asc()))
	opts = append(opts, repo.Order(query.UpdatedAt.Desc()))
	opts = append(opts, repo.Where(query.DictID.In(dictIds...)))
	opts = append(opts, repo.Where(query.Status.Eq(int32(common.Status_ENABLE))))
	return c.List(ctx, opts...)
}

// 按字典编码和值查询标签
func (c *BaseDictItemCase) findLabelByCodeAndValue(ctx context.Context, code, value string) (string, error) {
	baseDictItemQuery := c.Query(ctx).BaseDictItem
	baseDictQuery := c.baseDictRepo.Query(ctx).BaseDict
	// 通过字典表和字典项表联查，直接返回展示标签
	query := baseDictItemQuery.WithContext(ctx).Select(baseDictItemQuery.Label).Join(baseDictQuery, baseDictItemQuery.DictID.EqCol(baseDictQuery.ID))
	query = query.Where(baseDictQuery.Code.Eq(code))
	query = query.Where(baseDictItemQuery.Value.Eq(value))

	var label string
	err := query.Scan(&label)
	return label, err
}
