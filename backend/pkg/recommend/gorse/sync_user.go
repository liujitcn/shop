package gorse

import (
	"context"
	"errors"
	"strconv"

	"shop/pkg/gen/models"

	client "github.com/gorse-io/gorse-go"
	_set "github.com/liujitcn/go-utils/set"
)

// UserSyncReceiver 表示用户主数据同步接收器。
type UserSyncReceiver struct {
	recommend *Recommend
}

// NewUserSyncReceiver 创建用户主数据同步接收器。
func NewUserSyncReceiver(recommend *Recommend) *UserSyncReceiver {
	return &UserSyncReceiver{recommend: recommend}
}

// Enabled 判断当前用户主数据同步接收器是否可用。
func (r *UserSyncReceiver) Enabled() bool {
	return r.recommend.Enabled()
}

// LoadIDs 加载推荐系统中已存在的用户主体编号集合。
func (r *UserSyncReceiver) LoadIDs(ctx context.Context, pageSize int) (_set.Set[string], error) {
	// 客户端未启用时，直接返回空用户集合。
	if !r.Enabled() {
		return _set.NewThreadUnsafeSet[string](), nil
	}
	// 分页大小非法时，回退到默认分页大小，避免Gorse 接口收到无效参数。
	if pageSize <= 0 {
		pageSize = 100
	}

	userIDSet := _set.NewThreadUnsafeSetWithSize[string](pageSize)
	cursor := ""
	for {
		iterator, err := r.recommend.gorseClient.GetUsers(ctx, pageSize, cursor)
		if err != nil {
			return nil, err
		}
		for _, item := range iterator.Users {
			// Gorse 返回空用户编号时，直接跳过当前无效数据。
			if item.UserId == "" {
				continue
			}
			userIDSet.Add(item.UserId)
		}
		// 当前页没有更多游标或下一页游标未发生变化时，说明Gorse集合已经遍历完成。
		if iterator.Cursor == "" || iterator.Cursor == cursor {
			break
		}
		cursor = iterator.Cursor
	}
	return userIDSet, nil
}

// SyncList 同步一批后台用户快照到推荐系统。
func (r *UserSyncReceiver) SyncList(ctx context.Context, userList []*models.BaseUser, existingUserIDs _set.Set[string], staleUserIDs _set.Set[string]) error {
	// 客户端未启用时，直接跳过当前用户同步批次。
	if !r.Enabled() {
		return nil
	}
	// 未传Gorse索引时，回退到单条 upsert 逻辑保证兼容性。
	if existingUserIDs == nil {
		for _, user := range userList {
			// 无效用户快照不参与当前用户同步批次。
			if user == nil || user.ID <= 0 {
				continue
			}
			syncErr := r.sync(ctx, user)
			if syncErr != nil {
				return syncErr
			}
		}
		return nil
	}

	insertUsers := make([]client.User, 0, len(userList))
	insertUserList := make([]*models.BaseUser, 0, len(userList))
	for _, user := range userList {
		// 无效用户快照不参与当前用户同步批次。
		if user == nil || user.ID <= 0 {
			continue
		}
		recommendUserID := strconv.FormatInt(user.ID, 10)
		// 当前用户在本地仍然存在时，先从Gorse待删除集合中移除，避免后续误删有效主体。
		if staleUserIDs != nil {
			staleUserIDs.Remove(recommendUserID)
		}
		recommendUser, userPatch := r.buildPayload(user)
		// Gorse已经存在时，直接走单条更新，避免重复插入失败后再回退。
		if existingUserIDs.ContainsOne(recommendUser.UserId) {
			_, updateErr := r.recommend.gorseClient.UpdateUser(ctx, recommendUser.UserId, userPatch)
			if updateErr != nil {
				return updateErr
			}
			continue
		}
		insertUsers = append(insertUsers, recommendUser)
		insertUserList = append(insertUserList, user)
	}
	// 当前批次没有新增用户时，说明本轮只命中了更新数据。
	if len(insertUsers) == 0 {
		return nil
	}

	_, err := r.recommend.gorseClient.InsertUsers(ctx, insertUsers)
	// 批量插入失败时，回退到单条 upsert，避免因为索引陈旧或Gorse部分冲突导致整批失败。
	if err != nil {
		var fallbackErr error
		for _, user := range insertUserList {
			syncErr := r.sync(ctx, user)
			if syncErr != nil {
				fallbackErr = errors.Join(fallbackErr, syncErr)
			}
		}
		if fallbackErr != nil {
			return errors.Join(err, fallbackErr)
		}
		return nil
	}

	for _, item := range insertUsers {
		existingUserIDs.Add(item.UserId)
	}
	return nil
}

// DeleteIDs 删除推荐系统中多余的用户主体。
func (r *UserSyncReceiver) DeleteIDs(ctx context.Context, staleUserIDs _set.Set[string]) error {
	// 客户端未启用或没有待删除用户时，直接跳过当前清理批次。
	if !r.Enabled() || staleUserIDs == nil || staleUserIDs.IsEmpty() {
		return nil
	}
	var deleteErr error
	for userID := range staleUserIDs.Iter() {
		// 待删除编号为空时，直接跳过当前无效主体。
		if userID == "" {
			continue
		}
		// 推荐系统接口会在删除用户主体时一并级联删除该用户下的反馈数据。
		_, err := r.recommend.gorseClient.DeleteUser(ctx, userID)
		if err != nil {
			deleteErr = errors.Join(deleteErr, err)
		}
	}
	return deleteErr
}

// sync 将单个用户快照同步到推荐系统。
func (r *UserSyncReceiver) sync(ctx context.Context, user *models.BaseUser) error {
	// 客户端未启用或用户快照无效时，无需继续同步。
	if !r.Enabled() || user == nil || user.ID <= 0 {
		return nil
	}
	recommendUser, userPatch := r.buildPayload(user)
	_, err := r.recommend.gorseClient.InsertUser(ctx, recommendUser)
	if err == nil {
		return nil
	}

	_, updateErr := r.recommend.gorseClient.UpdateUser(ctx, recommendUser.UserId, userPatch)
	if updateErr == nil {
		return nil
	}
	return errors.Join(err, updateErr)
}

// buildPayload 构建推荐系统用户写入载荷。
func (r *UserSyncReceiver) buildPayload(user *models.BaseUser) (client.User, client.UserPatch) {
	comment := user.NickName
	// 用户昵称为空时，回退到用户名作为注释信息。
	if comment == "" {
		comment = user.UserName
	}

	labels := map[string]interface{}{
		"role_id": user.RoleID,
		"dept_id": user.DeptID,
		"gender":  user.Gender,
		"status":  user.Status,
	}
	return client.User{
			UserId:  strconv.FormatInt(user.ID, 10),
			Labels:  labels,
			Comment: comment,
		}, client.UserPatch{
			Labels:  labels,
			Comment: &comment,
		}
}
