package biz

import (
	"context"
	"errors"
	"slices"

	"shop/api/gen/go/app"
	"shop/api/gen/go/common"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/go-kratos/kratos/v2/log"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repo"
	"github.com/liujitcn/kratos-kit/sdk"
	"gorm.io/gorm"
)

// UserStoreCase 用户门店入驻业务处理对象
type UserStoreCase struct {
	*biz.BaseCase
	tx data.Transaction
	*data.UserStoreRepo
	baseAreaCase *BaseAreaCase
}

// NewUserStoreCase 创建用户门店入驻业务处理对象
func NewUserStoreCase(baseCase *biz.BaseCase, tx data.Transaction,
	userStoreRepo *data.UserStoreRepo,
	baseAreaCase *BaseAreaCase,
) *UserStoreCase {
	return &UserStoreCase{
		BaseCase:      baseCase,
		tx:            tx,
		UserStoreRepo: userStoreRepo,
		baseAreaCase:  baseAreaCase,
	}
}

// GetUserStore 查询用户门店
func (c *UserStoreCase) GetUserStore(ctx context.Context) (*app.UserStore, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}

	query := c.Query(ctx).UserStore
	var userStore *models.UserStore
	userStore, err = c.Find(ctx,
		repo.Where(query.UserID.Eq(authInfo.UserId)),
	)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &app.UserStore{}, nil
		}
		return nil, err
	}
	return c.convertToProto(ctx, userStore), nil
}

// CreateUserStore 创建用户门店
func (c *UserStoreCase) CreateUserStore(ctx context.Context, form *app.UserStoreForm) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	return c.Create(ctx, c.convertToModel(authInfo.UserId, form))
}

// UpdateUserStore 更新用户门店
func (c *UserStoreCase) UpdateUserStore(ctx context.Context, form *app.UserStoreForm) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}

	query := c.Query(ctx).UserStore
	var oldUserStore *models.UserStore
	oldUserStore, err = c.Find(ctx,
		repo.Where(query.ID.Eq(form.GetId())),
		repo.Where(query.UserID.Eq(authInfo.UserId)),
	)
	if err != nil {
		return err
	}
	// 更新前先保留旧数据，供后续清理被替换的图片使用
	userStore := c.convertToModel(authInfo.UserId, form)
	err = c.Update(ctx,
		userStore,
		repo.Where(query.ID.Eq(form.GetId())),
		repo.Where(query.UserID.Eq(authInfo.UserId)),
	)
	if err != nil {
		return err
	}

	// 更新成功后清理已被替换的旧文件
	c.multiDeleteFileByString(oldUserStore.Picture, form.GetPicture())
	c.multiDeleteFileByString(oldUserStore.BusinessLicense, form.GetBusinessLicense())
	return nil
}

// 将用户门店模型转换为接口响应
func (c *UserStoreCase) convertToProto(ctx context.Context, item *models.UserStore) *app.UserStore {
	return &app.UserStore{
		Id:              item.ID,
		Name:            item.Name,
		Address:         _string.ConvertJsonStringToStringArray(item.Address),
		Detail:          item.Detail,
		Picture:         _string.ConvertJsonStringToStringArray(item.Picture),
		BusinessLicense: _string.ConvertJsonStringToStringArray(item.BusinessLicense),
		AddressName:     c.baseAreaCase.getAddressListByCode(ctx, item.Address),
		Status:          common.UserStoreStatus(item.Status),
		Remark:          item.Remark,
	}
}

// 将用户门店表单转换为模型
func (c *UserStoreCase) convertToModel(userId int64, item *app.UserStoreForm) *models.UserStore {
	res := &models.UserStore{
		ID:              item.GetId(),
		UserID:          userId,
		Name:            item.GetName(),
		Address:         _string.ConvertStringArrayToString(item.GetAddress()),
		Detail:          item.GetDetail(),
		Picture:         _string.ConvertStringArrayToString(item.GetPicture()),
		BusinessLicense: _string.ConvertStringArrayToString(item.GetBusinessLicense()),
		Status:          int32(common.UserStoreStatus_PENDING_REVIEW),
	}
	return res
}

// 删除字符串形式旧文件集合中的冗余文件
func (c *UserStoreCase) multiDeleteFileByString(oldFile string, newFile []string) {
	oldFileList := _string.ConvertJsonStringToStringArray(oldFile)
	oss := sdk.Runtime.GetOSS()
	if oss != nil {
		for _, item := range oldFileList {
			if len(newFile) == 0 || !slices.Contains(newFile, item) {
				if err := oss.DeleteFile(item); err != nil {
					log.Error("multiDeleteFile err:", err.Error())
				}
			}
		}
	}
}
