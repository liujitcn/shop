package biz

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"shop/api/gen/go/common"
	"shop/pkg/biz"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/go-kratos/kratos/v2/transport"
	"github.com/liujitcn/go-utils/id"
	"github.com/liujitcn/gorm-kit/repo"
	"gorm.io/gorm"
)

const recommendAnonymousActorHeaderKey = "X-Recommend-Anonymous-Id"

// RecommendAnonymousActorCase 推荐匿名主体业务处理对象。
type RecommendAnonymousActorCase struct {
	*biz.BaseCase
	tx data.Transaction
	*data.RecommendAnonymousActorRepo
	recommendRequestRepo *data.RecommendRequestRepo
	recommendEventRepo   *data.RecommendEventRepo
}

// NewRecommendAnonymousActorCase 创建推荐匿名主体业务处理对象。
func NewRecommendAnonymousActorCase(
	baseCase *biz.BaseCase,
	tx data.Transaction,
	recommendAnonymousActorRepo *data.RecommendAnonymousActorRepo,
	recommendRequestRepo *data.RecommendRequestRepo,
	recommendEventRepo *data.RecommendEventRepo,
) *RecommendAnonymousActorCase {
	return &RecommendAnonymousActorCase{
		BaseCase:                    baseCase,
		tx:                          tx,
		RecommendAnonymousActorRepo: recommendAnonymousActorRepo,
		recommendRequestRepo:        recommendRequestRepo,
		recommendEventRepo:          recommendEventRepo,
	}
}

// getRecommendAnonymousIdFromHeader 从请求头中解析匿名主体编号。
func (c *RecommendAnonymousActorCase) getRecommendAnonymousIdFromHeader(ctx context.Context) (int64, error) {
	serverTransport, ok := transport.FromServerContext(ctx)
	// 非服务端请求上下文时，不存在可读取的请求头。
	if !ok {
		return 0, nil
	}

	headerValue := strings.TrimSpace(serverTransport.RequestHeader().Get(recommendAnonymousActorHeaderKey))
	// 未传入匿名主体请求头时，返回 0 表示当前请求未使用匿名身份。
	if headerValue == "" {
		return 0, nil
	}

	anonymousId, err := strconv.ParseInt(headerValue, 10, 64)
	if err != nil || anonymousId <= 0 {
		return 0, errorsx.InvalidArgument("匿名推荐主体无效")
	}
	return anonymousId, nil
}

// createRecommendAnonymousActor 创建匿名推荐主体。
func (c *RecommendAnonymousActorCase) createRecommendAnonymousActor(ctx context.Context) (int64, error) {
	anonymousId := id.GenSnowflakeID()
	now := time.Now()
	query := c.Query(ctx).RecommendAnonymousActor
	err := query.WithContext(ctx).
		Omit(query.UserID, query.BindAt).
		Create(&models.RecommendAnonymousActor{
			AnonymousID: anonymousId,
			CreatedAt:   now,
			UpdatedAt:   now,
		})
	if err != nil {
		return 0, errorsx.Internal("创建匿名推荐主体失败").WithCause(err)
	}
	return anonymousId, nil
}

// ensureRecommendAnonymousActor 确保匿名主体记录存在并刷新活跃时间。
func (c *RecommendAnonymousActorCase) ensureRecommendAnonymousActor(ctx context.Context, anonymousId int64) error {
	// 匿名主体编号非法时，无需继续处理匿名会话记录。
	if anonymousId <= 0 {
		return nil
	}

	now := time.Now()
	query := c.Query(ctx).RecommendAnonymousActor
	opts := make([]repo.QueryOption, 0, 1)
	opts = append(opts, repo.Where(query.AnonymousID.Eq(anonymousId)))
	_, err := c.Find(ctx, opts...)
	if err != nil {
		// 记录不存在时，补建匿名主体并写入首次活跃时间。
		if errors.Is(err, gorm.ErrRecordNotFound) {
			createErr := query.WithContext(ctx).
				Omit(query.UserID, query.BindAt).
				Create(&models.RecommendAnonymousActor{
					AnonymousID: anonymousId,
					CreatedAt:   now,
					UpdatedAt:   now,
				})
			if createErr != nil {
				return errorsx.Internal("保存匿名推荐主体失败").WithCause(createErr)
			}
			return nil
		}
		return errorsx.Internal("查询匿名推荐主体失败").WithCause(err)
	}

	updateErr := c.Update(ctx, &models.RecommendAnonymousActor{
		UpdatedAt: now,
	}, opts...)
	if updateErr != nil {
		return errorsx.Internal("更新匿名推荐主体失败").WithCause(updateErr)
	}
	return nil
}

// bindRecommendAnonymousActor 绑定匿名推荐主体到当前用户。
func (c *RecommendAnonymousActorCase) bindRecommendAnonymousActor(ctx context.Context, userId, anonymousId int64) error {
	// 当前未携带匿名主体或用户编号非法时，无需继续绑定。
	if userId <= 0 || anonymousId <= 0 {
		return nil
	}

	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		now := time.Now()
		query := c.Query(ctx).RecommendAnonymousActor
		opts := make([]repo.QueryOption, 0, 1)
		opts = append(opts, repo.Where(query.AnonymousID.Eq(anonymousId)))
		actor, err := c.Find(ctx, opts...)
		if err != nil {
			// 记录不存在时，按“创建并绑定”路径补齐匿名主体。
			if errors.Is(err, gorm.ErrRecordNotFound) {
				createErr := query.WithContext(ctx).Create(&models.RecommendAnonymousActor{
					AnonymousID: anonymousId,
					UserID:      userId,
					CreatedAt:   now,
					UpdatedAt:   now,
					BindAt:      now,
				})
				if createErr != nil {
					return errorsx.Internal("绑定匿名推荐主体失败").WithCause(createErr)
				}
			} else {
				return errorsx.Internal("绑定匿名推荐主体失败").WithCause(err)
			}
		} else {
			// 匿名主体已绑定到其他用户时，不允许再串联到当前账号。
			if actor.UserID > 0 && actor.UserID != userId {
				return errorsx.Conflict("匿名推荐主体已绑定其他用户")
			}

			updateErr := c.Update(ctx, &models.RecommendAnonymousActor{
				UserID:    userId,
				UpdatedAt: now,
				BindAt:    now,
			}, opts...)
			if updateErr != nil {
				return errorsx.Internal("绑定匿名推荐主体失败").WithCause(updateErr)
			}
		}

		// 匿名主体完成绑定后，需要把匿名阶段积累的推荐历史一并迁移到登录用户。
		err = c.rebindRecommendActorHistory(ctx, userId, anonymousId)
		if err != nil {
			return err
		}
		return nil
	})
}

// rebindRecommendActorHistory 将匿名主体下的推荐历史迁移到登录用户。
func (c *RecommendAnonymousActorCase) rebindRecommendActorHistory(ctx context.Context, userId, anonymousId int64) error {
	err := c.rebindRecommendRequestActor(ctx, userId, anonymousId)
	if err != nil {
		return err
	}
	return c.rebindRecommendEventActor(ctx, userId, anonymousId)
}

// rebindRecommendRequestActor 将匿名主体下的推荐请求记录迁移到登录用户。
func (c *RecommendAnonymousActorCase) rebindRecommendRequestActor(ctx context.Context, userId, anonymousId int64) error {
	// 用户编号或匿名主体编号非法时，不存在可迁移的推荐请求记录。
	if userId <= 0 || anonymousId <= 0 {
		return nil
	}

	query := c.recommendRequestRepo.Query(ctx).RecommendRequest
	res, err := query.WithContext(ctx).Where(
		query.ActorType.Eq(int32(common.RecommendActorType_ANONYMOUS)),
		query.ActorID.Eq(anonymousId),
	).Updates(map[string]interface{}{
		"actor_type": int32(common.RecommendActorType_USER),
		"actor_id":   userId,
	})
	if err != nil {
		return errorsx.Internal("迁移匿名推荐请求失败").WithCause(err)
	}
	return res.Error
}

// rebindRecommendEventActor 将匿名主体下的推荐事件记录迁移到登录用户。
func (c *RecommendAnonymousActorCase) rebindRecommendEventActor(ctx context.Context, userId, anonymousId int64) error {
	// 用户编号或匿名主体编号非法时，不存在可迁移的推荐事件记录。
	if userId <= 0 || anonymousId <= 0 {
		return nil
	}

	query := c.recommendEventRepo.Query(ctx).RecommendEvent
	res, err := query.WithContext(ctx).Where(
		query.ActorType.Eq(int32(common.RecommendActorType_ANONYMOUS)),
		query.ActorID.Eq(anonymousId),
	).Updates(map[string]interface{}{
		"actor_type": int32(common.RecommendActorType_USER),
		"actor_id":   userId,
	})
	if err != nil {
		return errorsx.Internal("迁移匿名推荐事件失败").WithCause(err)
	}
	return res.Error
}
