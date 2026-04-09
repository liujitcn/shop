package biz

import (
	"context"
	"strconv"
	"sync"

	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"shop/api/gen/go/common"

	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/kratos-kit/cache"
)

var tree *common.AppTreeOptionResponse
var codeMap map[string]string
var lock sync.RWMutex

// BaseAreaCase 行政区域业务处理对象
type BaseAreaCase struct {
	*biz.BaseCase
	*data.BaseAreaRepo
	cache  cache.Cache
	mapper *mapper.CopierMapper[common.AppTreeOptionResponse_Option, models.BaseArea]
}

// NewBaseAreaCase 创建行政区域业务处理对象
func NewBaseAreaCase(
	baseCase *biz.BaseCase,
	baseAreaRepo *data.BaseAreaRepo,
) *BaseAreaCase {
	return &BaseAreaCase{
		BaseCase:     baseCase,
		BaseAreaRepo: baseAreaRepo,
		mapper:       mapper.NewCopierMapper[common.AppTreeOptionResponse_Option, models.BaseArea](),
	}
}

// TreeBaseArea 查询行政区域树形列表
func (c *BaseAreaCase) TreeBaseArea(ctx context.Context) (*common.AppTreeOptionResponse, error) {
	lock.RLock()
	defer lock.RUnlock()
	if tree == nil {
		// 首次访问时从数据库加载并缓存，避免重复构树
		list, err := c.List(ctx)
		if err != nil {
			return nil, err
		}
		tree = &common.AppTreeOptionResponse{
			List: c.buildTree(list, 0),
		}
	}
	return tree, nil
}

// 将行政区划编码字符串转成拼接后的地址文本
func (c *BaseAreaCase) getAddressByCode(ctx context.Context, code string) string {
	res := c.getAddressListByCode(ctx, code)
	return _string.ConvertStringArrayToString(res)
}

// 将行政区划编码字符串转成地址名称列表
func (c *BaseAreaCase) getAddressListByCode(ctx context.Context, code string) []string {
	lock.RLock()
	defer lock.RUnlock()
	res := make([]string, 0)
	if codeMap == nil {
		// 懒加载编码映射，减少重复查询
		list, err := c.List(ctx)
		if err != nil {
			return res
		}
		codeMap = make(map[string]string)
		for _, item := range list {
			codeMap[strconv.FormatInt(item.ID, 10)] = item.Name
		}
	}
	codeList := _string.ConvertJsonStringToStringArray(code)
	for _, item := range codeList {
		if v, ok := codeMap[item]; ok {
			res = append(res, v)
		} else {
			res = append(res, item)
		}
	}
	return res
}

// 递归构建行政区域树
func (c *BaseAreaCase) buildTree(list []*models.BaseArea, parentId int64) []*common.AppTreeOptionResponse_Option {
	var res []*common.AppTreeOptionResponse_Option
	for _, item := range list {
		if item.ParentID == parentId {
			option := c.mapper.ToDTO(item)
			option.Value = strconv.FormatInt(item.ID, 10)
			option.Text = item.Name
			option.Children = c.buildTree(list, item.ID)
			res = append(res, option)
		}
	}
	return res
}
