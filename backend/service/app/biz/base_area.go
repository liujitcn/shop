package biz

import (
	"context"
	"strconv"
	"sync"

	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	commonv1 "shop/api/gen/go/common/v1"

	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/kratos-kit/cache"
)

var tree *commonv1.AppTreeOptionResponse
var codeMap map[string]string
var lock sync.RWMutex

// BaseAreaCase 行政区域业务处理对象
type BaseAreaCase struct {
	*biz.BaseCase
	*data.BaseAreaRepository
	cache  cache.Cache
	mapper *mapper.CopierMapper[commonv1.AppTreeOptionResponse_Option, models.BaseArea]
}

// NewBaseAreaCase 创建行政区域业务处理对象
func NewBaseAreaCase(
	baseCase *biz.BaseCase,
	baseAreaRepo *data.BaseAreaRepository,
) *BaseAreaCase {
	return &BaseAreaCase{
		BaseCase:           baseCase,
		BaseAreaRepository: baseAreaRepo,
		mapper:             mapper.NewCopierMapper[commonv1.AppTreeOptionResponse_Option, models.BaseArea](),
	}
}

// TreeBaseAreas 查询行政区域树形列表
func (c *BaseAreaCase) TreeBaseAreas(ctx context.Context) (*commonv1.AppTreeOptionResponse, error) {
	lock.RLock()
	defer lock.RUnlock()
	// 树缓存尚未初始化时，从数据库加载并构建整棵区域树。
	if tree == nil {
		// 首次访问时从数据库加载并缓存，避免重复构树
		list, err := c.List(ctx)
		if err != nil {
			return nil, err
		}
		tree = &commonv1.AppTreeOptionResponse{
			List: c.buildTree(list, 0),
		}
	}
	return tree, nil
}

// 将行政区划编码字符串转成地址名称列表
func (c *BaseAreaCase) getAddressListByCode(ctx context.Context, code string) []string {
	lock.RLock()
	defer lock.RUnlock()
	res := make([]string, 0)
	// 编码映射尚未初始化时，先懒加载全部区域编码。
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
		// 命中编码映射时，返回对应的区域名称。
		if v, ok := codeMap[item]; ok {
			res = append(res, v)
		} else {
			res = append(res, item)
		}
	}
	return res
}

// 递归构建行政区域树
func (c *BaseAreaCase) buildTree(list []*models.BaseArea, parentID int64) []*commonv1.AppTreeOptionResponse_Option {
	var res []*commonv1.AppTreeOptionResponse_Option
	for _, item := range list {
		// 仅把当前父节点下的区域挂到本层结果里。
		if item.ParentID == parentID {
			option := c.mapper.ToDTO(item)
			option.Value = strconv.FormatInt(item.ID, 10)
			option.Text = item.Name
			option.Children = c.buildTree(list, item.ID)
			res = append(res, option)
		}
	}
	return res
}
