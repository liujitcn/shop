package biz

import (
	"context"
	"fmt"

	shopadminv1 "shop/api/gen/go/shop/admin/v1"
	"shop/pkg/biz"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	_const "shop/service/shop/consts"
	systemadminbiz "shop/service/system/admin/biz"

	"github.com/liujitcn/go-utils/mapper"
	_set "github.com/liujitcn/go-utils/set"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
	databaseGorm "github.com/liujitcn/kratos-kit/database/gorm"
)

// TenantStoreCase 租户门店业务实例。
type TenantStoreCase struct {
	*biz.BaseCase
	tx data.Transaction
	*data.TenantStoreRepository
	goodsInfoRepo  *data.GoodsInfoRepository
	baseTenantCase *systemadminbiz.BaseTenantCase
	formMapper     *mapper.CopierMapper[shopadminv1.TenantStoreForm, models.TenantStore]
	mapper         *mapper.CopierMapper[shopadminv1.TenantStore, models.TenantStore]
}

// NewTenantStoreCase 创建租户门店业务实例。
func NewTenantStoreCase(baseCase *biz.BaseCase, tx data.Transaction, tenantStoreRepo *data.TenantStoreRepository, goodsInfoRepo *data.GoodsInfoRepository, baseTenantCase *systemadminbiz.BaseTenantCase) *TenantStoreCase {
	formMapper := mapper.NewCopierMapper[shopadminv1.TenantStoreForm, models.TenantStore]()
	formMapper.AppendConverters(mapper.NewJSONTypeConverter[[]string]().NewConverterPair())
	tenantStoreMapper := mapper.NewCopierMapper[shopadminv1.TenantStore, models.TenantStore]()
	tenantStoreMapper.AppendConverters(mapper.NewJSONTypeConverter[[]string]().NewConverterPair())
	return &TenantStoreCase{
		BaseCase:              baseCase,
		tx:                    tx,
		TenantStoreRepository: tenantStoreRepo,
		goodsInfoRepo:         goodsInfoRepo,
		baseTenantCase:        baseTenantCase,
		formMapper:            formMapper,
		mapper:                tenantStoreMapper,
	}
}

// OptionTenantStore 查询已审核通过的门店下拉选项。
func (c *TenantStoreCase) OptionTenantStore(ctx context.Context, req *shopadminv1.OptionTenantStoreRequest) (*shopadminv1.OptionTenantStoreResponse, error) {
	query := c.Query(ctx).TenantStore
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Where(query.Status.Eq(_const.TENANT_STORE_STATUS_APPROVED)))
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	if req.GetKeyword() != "" {
		opts = append(opts, repository.Where(query.Name.Like("%"+req.GetKeyword()+"%")))
	}

	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	options := make([]*shopadminv1.OptionTenantStoreResponse_Option, 0, len(list))
	for _, item := range list {
		options = append(options, &shopadminv1.OptionTenantStoreResponse_Option{
			Label: item.Name,
			Value: item.ID,
		})
	}
	return &shopadminv1.OptionTenantStoreResponse{List: options}, nil
}

// TreeTenantStore 查询租户门店树形选项。
func (c *TenantStoreCase) TreeTenantStore(ctx context.Context, req *shopadminv1.TreeTenantStoreRequest) (*shopadminv1.TreeTenantStoreResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}

	query := c.Query(ctx).TenantStore
	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Where(query.Status.Eq(_const.TENANT_STORE_STATUS_APPROVED)))
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	if req.GetKeyword() != "" {
		opts = append(opts, repository.Where(query.Name.Like("%"+req.GetKeyword()+"%")))
	}

	var list []*models.TenantStore
	list, err = c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	if authInfo.TenantCode != databaseGorm.DefaultTenantCode {
		return &shopadminv1.TreeTenantStoreResponse{List: c.buildStoreTreeOptions(list)}, nil
	}

	tenantIDSet := _set.NewSet[int64]()
	for _, item := range list {
		tenantIDSet.Add(item.TenantID)
	}
	var tenantList []*models.BaseTenant
	tenantList, err = c.baseTenantCase.ListByIDs(ctx, tenantIDSet.ToSlice())
	if err != nil {
		return nil, err
	}
	tenantNames := make(map[int64]string, len(tenantList))
	for _, item := range tenantList {
		tenantNames[item.ID] = item.Name
	}
	return &shopadminv1.TreeTenantStoreResponse{List: c.buildTenantStoreTreeOptions(list, tenantNames)}, nil
}

// PageTenantStore 分页查询租户门店。
func (c *TenantStoreCase) PageTenantStore(ctx context.Context, req *shopadminv1.PageTenantStoreRequest) (*shopadminv1.PageTenantStoreResponse, error) {
	query := c.Query(ctx).TenantStore
	opts := make([]repository.QueryOption, 0, 5)
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	if req.GetName() != "" {
		opts = append(opts, repository.Where(query.Name.Like("%"+req.GetName()+"%")))
	}
	if req.GetTenantId() > 0 {
		opts = append(opts, repository.Where(query.TenantID.Eq(req.GetTenantId())))
	}
	if req.Status != nil {
		opts = append(opts, repository.Where(query.Status.Eq(int32(req.GetStatus()))))
	}

	list, total, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}

	resList := make([]*shopadminv1.TenantStore, 0, len(list))
	for _, item := range list {
		resList = append(resList, c.mapper.ToDTO(item))
	}
	return &shopadminv1.PageTenantStoreResponse{TenantStores: resList, Total: int32(total)}, nil
}

// GetTenantStore 获取租户门店。
func (c *TenantStoreCase) GetTenantStore(ctx context.Context, id int64) (*shopadminv1.TenantStoreForm, error) {
	tenantStore, err := c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return c.formMapper.ToDTO(tenantStore), nil
}

// CreateTenantStore 创建租户门店。
func (c *TenantStoreCase) CreateTenantStore(ctx context.Context, req *shopadminv1.TenantStoreForm) error {
	tenantStore := c.formMapper.ToEntity(req)
	tenantStore.Status = _const.TENANT_STORE_STATUS_PENDING_REVIEW
	tenantStore.Remark = ""
	err := c.Create(ctx, tenantStore)
	if err != nil {
		return err
	}
	return nil
}

// UpdateTenantStore 更新租户门店。
func (c *TenantStoreCase) UpdateTenantStore(ctx context.Context, req *shopadminv1.TenantStoreForm) error {
	if req.GetId() <= 0 {
		return errorsx.InvalidArgument("门店参数不合法")
	}
	oldTenantStore, err := c.FindByID(ctx, req.GetId())
	if err != nil {
		return err
	}

	tenantStore := c.formMapper.ToEntity(req)
	tenantStore.TenantID = oldTenantStore.TenantID
	tenantStore.Status = _const.TENANT_STORE_STATUS_PENDING_REVIEW
	tenantStore.Remark = ""
	err = c.tx.Transaction(ctx, func(ctx context.Context) error {
		err = c.UpdateByID(ctx, tenantStore)
		if err != nil {
			return err
		}

		query := c.goodsInfoRepo.Query(ctx).GoodsInfo
		opts := make([]repository.QueryOption, 0, 2)
		opts = append(opts, repository.Where(query.TenantStoreID.Eq(oldTenantStore.ID)))
		opts = append(opts, repository.Where(query.Status.Eq(_const.GOODS_STATUS_PUT_ON)))
		return c.goodsInfoRepo.Update(ctx, &models.GoodsInfo{Status: _const.GOODS_STATUS_DISABLED_BY_STORE}, opts...)
	})
	return err
}

// DeleteTenantStore 删除租户门店。
func (c *TenantStoreCase) DeleteTenantStore(ctx context.Context, id string) error {
	ids := _string.ConvertStringToInt64Array(id)
	var count int64
	count, err := c.getGoodsCountByTenantStoreIDs(ctx, ids)
	if err != nil {
		return err
	}
	if count > 0 {
		return errorsx.HasChildrenConflict("删除门店失败，下面有商品", "tenant_store", "goods_info")
	}
	return c.DeleteByIDs(ctx, ids)
}

// AuditTenantStore 审核租户门店。
func (c *TenantStoreCase) AuditTenantStore(ctx context.Context, req *shopadminv1.AuditTenantStoreRequest) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	if authInfo.TenantCode != databaseGorm.DefaultTenantCode {
		return errorsx.PermissionDenied("无权审核租户门店")
	}
	_, err = c.FindByID(ctx, req.GetId())
	if err != nil {
		return err
	}

	status := int32(req.GetStatus())
	// 租户门店审核只允许设置最终审核结果，待审核状态由提交流程创建。
	if status != _const.TENANT_STORE_STATUS_APPROVED && status != _const.TENANT_STORE_STATUS_FAILED_REVIEW {
		return errorsx.InvalidArgument("审核状态仅支持通过或不通过")
	}
	if status == _const.TENANT_STORE_STATUS_FAILED_REVIEW && req.GetRemark() == "" {
		return errorsx.InvalidArgument("审核不通过时请填写审核备注")
	}

	goodsStatus := _const.GOODS_STATUS_DISABLED_BY_STORE
	currentGoodsStatus := _const.GOODS_STATUS_PUT_ON
	if status == _const.TENANT_STORE_STATUS_APPROVED {
		goodsStatus = _const.GOODS_STATUS_PUT_ON
		currentGoodsStatus = _const.GOODS_STATUS_DISABLED_BY_STORE
	}
	err = c.tx.Transaction(ctx, func(ctx context.Context) error {
		err = c.UpdateByID(ctx, &models.TenantStore{
			ID:     req.GetId(),
			Status: status,
			Remark: req.GetRemark(),
		})
		if err != nil {
			return err
		}

		query := c.goodsInfoRepo.Query(ctx).GoodsInfo
		opts := make([]repository.QueryOption, 0, 2)
		opts = append(opts, repository.Where(query.TenantStoreID.Eq(req.GetId())))
		opts = append(opts, repository.Where(query.Status.Eq(currentGoodsStatus)))
		return c.goodsInfoRepo.Update(ctx, &models.GoodsInfo{Status: goodsStatus}, opts...)
	})
	return err
}

// GetTenantStoreMapByIDs 按门店id批量查询门店。
func (c *TenantStoreCase) GetTenantStoreMapByIDs(ctx context.Context, ids []int64) (map[int64]*models.TenantStore, error) {
	if len(ids) == 0 {
		return map[int64]*models.TenantStore{}, nil
	}
	list, err := c.ListByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	res := make(map[int64]*models.TenantStore, len(list))
	for _, item := range list {
		res[item.ID] = item
	}
	return res, nil
}

// buildStoreTreeOptions 构建普通租户可见的门店树节点。
func (c *TenantStoreCase) buildStoreTreeOptions(list []*models.TenantStore) []*shopadminv1.TreeTenantStoreResponse_Option {
	options := make([]*shopadminv1.TreeTenantStoreResponse_Option, 0, len(list))
	for _, item := range list {
		options = append(options, &shopadminv1.TreeTenantStoreResponse_Option{
			Value:    fmt.Sprintf("store:%d", item.ID),
			Label:    item.Name,
			Type:     "store",
			Id:       item.ID,
			TenantId: item.TenantID,
		})
	}
	return options
}

// buildTenantStoreTreeOptions 构建默认租户可见的租户-门店树节点。
func (c *TenantStoreCase) buildTenantStoreTreeOptions(list []*models.TenantStore, tenantNames map[int64]string) []*shopadminv1.TreeTenantStoreResponse_Option {
	options := make([]*shopadminv1.TreeTenantStoreResponse_Option, 0)
	tenantOptionMap := make(map[int64]*shopadminv1.TreeTenantStoreResponse_Option)
	for _, item := range list {
		tenantOption := tenantOptionMap[item.TenantID]
		if tenantOption == nil {
			tenantOption = &shopadminv1.TreeTenantStoreResponse_Option{
				Value:    fmt.Sprintf("tenant:%d", item.TenantID),
				Label:    tenantNames[item.TenantID],
				Type:     "tenant",
				Id:       item.TenantID,
				TenantId: item.TenantID,
				Children: []*shopadminv1.TreeTenantStoreResponse_Option{},
			}
			tenantOptionMap[item.TenantID] = tenantOption
			options = append(options, tenantOption)
		}
		tenantOption.Children = append(tenantOption.Children, &shopadminv1.TreeTenantStoreResponse_Option{
			Value:    fmt.Sprintf("store:%d", item.ID),
			Label:    item.Name,
			Type:     "store",
			Id:       item.ID,
			TenantId: item.TenantID,
		})
	}
	return options
}

// getGoodsCountByTenantStoreIDs 查询门店下绑定的商品数量。
func (c *TenantStoreCase) getGoodsCountByTenantStoreIDs(ctx context.Context, ids []int64) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	query := c.goodsInfoRepo.Query(ctx).GoodsInfo
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.TenantStoreID.In(ids...)))
	return c.goodsInfoRepo.Count(ctx, opts...)
}
