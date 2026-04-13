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
	"github.com/liujitcn/kratos-kit/sdk"
)

const baseConfigCachePrefix = "config:"

// BaseConfigCase 配置业务实例
type BaseConfigCase struct {
	*biz.BaseCase
	*data.BaseConfigRepo
	formMapper *mapper.CopierMapper[admin.BaseConfigForm, models.BaseConfig]
	mapper     *mapper.CopierMapper[admin.BaseConfig, models.BaseConfig]
}

// NewBaseConfigCase 创建配置业务实例
func NewBaseConfigCase(baseCase *biz.BaseCase, baseConfigRepo *data.BaseConfigRepo) *BaseConfigCase {
	return &BaseConfigCase{
		BaseCase:       baseCase,
		BaseConfigRepo: baseConfigRepo,
		formMapper:     mapper.NewCopierMapper[admin.BaseConfigForm, models.BaseConfig](),
		mapper:         mapper.NewCopierMapper[admin.BaseConfig, models.BaseConfig](),
	}
}

// RefreshBaseConfig 刷新配置缓存
func (c *BaseConfigCase) RefreshBaseConfig(ctx context.Context) error {
	query := c.Query(ctx).BaseConfig
	opts := make([]repo.QueryOption, 0, 1)
	opts = append(opts, repo.Where(query.Site.Eq(int32(common.BaseConfigSite_SYSTEM))))
	list, err := c.List(ctx, opts...)
	if err != nil {
		return err
	}

	for _, item := range list {
		err = c.syncBaseConfigCache(item)
		if err != nil {
			return err
		}
	}
	return nil
}

// PageBaseConfig 分页查询配置
func (c *BaseConfigCase) PageBaseConfig(ctx context.Context, req *admin.PageBaseConfigRequest) (*admin.PageBaseConfigResponse, error) {
	query := c.Query(ctx).BaseConfig
	opts := make([]repo.QueryOption, 0, 6)
	opts = append(opts, repo.Order(query.CreatedAt.Desc()))
	if req.Site != nil {
		opts = append(opts, repo.Where(query.Site.Eq(int32(req.GetSite()))))
	}
	// 传入名称关键字时，按配置名称模糊匹配。
	if req.GetName() != "" {
		opts = append(opts, repo.Where(query.Name.Like("%"+req.GetName()+"%")))
	}
	if req.Type != nil {
		opts = append(opts, repo.Where(query.Type.Eq(int32(req.GetType()))))
	}
	// 传入键关键字时，按配置键模糊匹配。
	if req.GetKey() != "" {
		opts = append(opts, repo.Where(query.Key.Like("%"+req.GetKey()+"%")))
	}
	if req.Status != nil {
		opts = append(opts, repo.Where(query.Status.Eq(int32(req.GetStatus()))))
	}

	list, total, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}

	resList := make([]*admin.BaseConfig, 0, len(list))
	for _, item := range list {
		baseConfig := c.mapper.ToDTO(item)
		resList = append(resList, baseConfig)
	}

	return &admin.PageBaseConfigResponse{
		List:  resList,
		Total: int32(total),
	}, nil
}

// GetBaseConfig 根据主键查询配置
func (c *BaseConfigCase) GetBaseConfig(ctx context.Context, id int64) (*admin.BaseConfigForm, error) {
	baseConfig, err := c.FindById(ctx, id)
	if err != nil {
		return nil, err
	}
	res := c.formMapper.ToDTO(baseConfig)
	return res, nil
}

// CreateBaseConfig 创建配置
func (c *BaseConfigCase) CreateBaseConfig(ctx context.Context, req *admin.BaseConfigForm) error {
	entity := c.formMapper.ToEntity(req)
	err := c.Create(ctx, entity)
	if err != nil {
		return err
	}
	err = c.syncBaseConfigCache(entity)
	if err != nil {
		return err
	}
	return nil
}

// UpdateBaseConfig 更新配置
func (c *BaseConfigCase) UpdateBaseConfig(ctx context.Context, req *admin.BaseConfigForm) error {
	oldConfig, err := c.FindById(ctx, req.GetId())
	if err != nil {
		return err
	}

	entity := c.formMapper.ToEntity(req)
	err = c.UpdateById(ctx, entity)
	if err != nil {
		return err
	}

	// 配置键发生变化时，需要先清理旧缓存键。
	if oldConfig.Key != entity.Key {
		err = c.clearBaseConfigCache(oldConfig.Key)
		if err != nil {
			return err
		}
	}

	err = c.syncBaseConfigCache(entity)
	if err != nil {
		return err
	}
	return nil
}

// DeleteBaseConfig 删除配置
func (c *BaseConfigCase) DeleteBaseConfig(ctx context.Context, id string) error {
	ids := _string.ConvertStringToInt64Array(id)
	list, err := c.ListByIds(ctx, ids)
	if err != nil {
		return err
	}

	err = c.DeleteByIds(ctx, ids)
	if err != nil {
		return err
	}

	for _, item := range list {
		err = c.clearBaseConfigCache(item.Key)
		if err != nil {
			return err
		}
	}
	return nil
}

// SetBaseConfigStatus 设置配置状态
func (c *BaseConfigCase) SetBaseConfigStatus(ctx context.Context, req *common.SetStatusRequest) error {
	err := c.UpdateById(ctx, &models.BaseConfig{
		ID:     req.GetId(),
		Status: req.GetStatus(),
	})
	if err != nil {
		return err
	}

	var baseConfig *models.BaseConfig
	baseConfig, err = c.FindById(ctx, req.GetId())
	if err != nil {
		return err
	}

	// 启用状态需要同步配置缓存，禁用状态则清理缓存。
	if baseConfig.Status == int32(common.Status_ENABLE) {
		err = c.syncBaseConfigCache(baseConfig)
		if err != nil {
			return err
		}
	} else {
		err = c.clearBaseConfigCache(baseConfig.Key)
		if err != nil {
			return err
		}
	}

	return nil
}

// syncBaseConfigCache 同步单个配置缓存
func (c *BaseConfigCase) syncBaseConfigCache(item *models.BaseConfig) error {
	return sdk.Runtime.GetCache().Set(c.makeBaseConfigCacheKey(item.Key), item.Value, -1)
}

// clearBaseConfigCache 删除单个配置缓存
func (c *BaseConfigCase) clearBaseConfigCache(key string) error {
	return sdk.Runtime.GetCache().Del(c.makeBaseConfigCacheKey(key))
}

// makeBaseConfigCacheKey 生成配置缓存键
func (c *BaseConfigCase) makeBaseConfigCacheKey(key string) string {
	return baseConfigCachePrefix + key
}
