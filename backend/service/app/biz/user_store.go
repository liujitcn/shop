package biz

import (
	"context"
	"errors"
	"slices"

	_const "shop/pkg/const"

	appv1 "shop/api/gen/go/app/v1"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
	"github.com/liujitcn/kratos-kit/sdk"
	"gorm.io/gorm"
)

// UserStoreCase 用户门店入驻业务处理对象
type UserStoreCase struct {
	*biz.BaseCase
	tx data.Transaction
	*data.UserStoreRepository
	baseAreaCase *BaseAreaCase
	formMapper   *mapper.CopierMapper[appv1.UserStoreForm, models.UserStore]
	dtoMapper    *mapper.CopierMapper[appv1.UserStore, models.UserStore]
}

// NewUserStoreCase 创建用户门店入驻业务处理对象
func NewUserStoreCase(baseCase *biz.BaseCase, tx data.Transaction,
	userStoreRepo *data.UserStoreRepository,
	baseAreaCase *BaseAreaCase,
) *UserStoreCase {
	formMapper := mapper.NewCopierMapper[appv1.UserStoreForm, models.UserStore]()
	formMapper.AppendConverters(mapper.NewJSONTypeConverter[[]string]().NewConverterPair())
	dtoMapper := mapper.NewCopierMapper[appv1.UserStore, models.UserStore]()
	dtoMapper.AppendConverters(mapper.NewJSONTypeConverter[[]string]().NewConverterPair())
	return &UserStoreCase{
		BaseCase:            baseCase,
		tx:                  tx,
		UserStoreRepository: userStoreRepo,
		baseAreaCase:        baseAreaCase,
		formMapper:          formMapper,
		dtoMapper:           dtoMapper,
	}
}

// GetUserStore 查询用户门店
func (c *UserStoreCase) GetUserStore(ctx context.Context) (*appv1.UserStore, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}

	query := c.Query(ctx).UserStore
	var userStore *models.UserStore
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.UserID.Eq(authInfo.UserId)))
	userStore, err = c.Find(ctx, opts...)
	if err != nil {
		// 当前用户尚未开店时，返回空门店信息而不是错误。
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &appv1.UserStore{}, nil
		}
		return nil, err
	}
	return c.convertToProto(ctx, userStore), nil
}

// CreateUserStore 创建用户门店
func (c *UserStoreCase) CreateUserStore(ctx context.Context, form *appv1.UserStoreForm) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	return c.Create(ctx, c.convertToModel(authInfo.UserId, form))
}

// UpdateUserStore 更新用户门店
func (c *UserStoreCase) UpdateUserStore(ctx context.Context, form *appv1.UserStoreForm) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}

	query := c.Query(ctx).UserStore
	var oldUserStore *models.UserStore
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.ID.Eq(form.GetId())))
	opts = append(opts, repository.Where(query.UserID.Eq(authInfo.UserId)))
	oldUserStore, err = c.Find(ctx, opts...)
	if err != nil {
		return err
	}
	// 更新前先保留旧数据，供后续清理被替换的图片使用
	userStore := c.convertToModel(authInfo.UserId, form)
	err = c.Update(ctx, userStore, opts...)
	if err != nil {
		return err
	}

	// 更新成功后清理已被替换的旧文件
	c.multiDeleteFileByString(oldUserStore.Picture, form.GetPicture())
	c.multiDeleteFileByString(oldUserStore.BusinessLicense, form.GetBusinessLicense())
	return nil
}

// 将用户门店模型转换为接口响应
func (c *UserStoreCase) convertToProto(ctx context.Context, item *models.UserStore) *appv1.UserStore {
	res := c.dtoMapper.ToDTO(item)
	res.AddressName = c.baseAreaCase.getAddressListByCode(ctx, item.Address)
	return res
}

// 将用户门店表单转换为模型
func (c *UserStoreCase) convertToModel(userID int64, item *appv1.UserStoreForm) *models.UserStore {
	res := c.formMapper.ToEntity(item)
	res.UserID = userID
	res.Status = _const.USER_STORE_STATUS_PENDING_REVIEW
	return res
}

// 删除字符串形式旧文件集合中的冗余文件
func (c *UserStoreCase) multiDeleteFileByString(oldFile string, newFile []string) {
	oldFileList := _string.ConvertJsonStringToStringArray(oldFile)
	oss := sdk.Runtime.GetOSS()
	// OSS 已初始化时，按差异删除已被替换的旧文件。
	if oss != nil {
		for _, item := range oldFileList {
			// 新文件列表未保留该文件时，删除旧文件释放对象存储空间。
			if len(newFile) == 0 || !slices.Contains(newFile, item) {
				// 单个旧文件删除失败时，仅记录日志不影响主流程。
				if err := oss.DeleteFile(item); err != nil {
					log.Errorf("MultiDeleteFile %v", err)
				}
			}
		}
	}
}
