package biz

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	commonv1 "shop/api/gen/go/common/v1"
	_const "shop/pkg/const"
	"shop/pkg/queue"

	"shop/pkg/biz"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/liujitcn/go-utils/id"
	"github.com/liujitcn/gorm-kit/repository"
	"gorm.io/gorm"
)

const RECOMMEND_ANONYMOUS_ACTOR_HEADER_KEY = "X-Recommend-Anonymous-Id"

// RecommendAnonymousActorCase 推荐匿名主体业务处理对象。
type RecommendAnonymousActorCase struct {
	*biz.BaseCase
	tx data.Transaction
	*data.RecommendAnonymousActorRepository
	recommendRequestRepo *data.RecommendRequestRepository
	recommendEventRepo   *data.RecommendEventRepository
}

// NewRecommendAnonymousActorCase 创建推荐匿名主体业务处理对象。
func NewRecommendAnonymousActorCase(
	baseCase *biz.BaseCase,
	tx data.Transaction,
	recommendAnonymousActorRepo *data.RecommendAnonymousActorRepository,
	recommendRequestRepo *data.RecommendRequestRepository,
	recommendEventRepo *data.RecommendEventRepository,
) *RecommendAnonymousActorCase {
	return &RecommendAnonymousActorCase{
		BaseCase:                          baseCase,
		tx:                                tx,
		RecommendAnonymousActorRepository: recommendAnonymousActorRepo,
		recommendRequestRepo:              recommendRequestRepo,
		recommendEventRepo:                recommendEventRepo,
	}
}

// getRecommendAnonymousIDFromHeader 从请求头中解析匿名主体编号。
func (c *RecommendAnonymousActorCase) getRecommendAnonymousIDFromHeader(ctx context.Context) (int64, error) {
	serverTransport, ok := transport.FromServerContext(ctx)
	// 非服务端请求上下文时，不存在可读取的请求头。
	if !ok {
		return 0, nil
	}

	headerValue := strings.TrimSpace(serverTransport.RequestHeader().Get(RECOMMEND_ANONYMOUS_ACTOR_HEADER_KEY))
	// 未传入匿名主体请求头时，返回 0 表示当前请求未使用匿名身份。
	if headerValue == "" {
		return 0, nil
	}

	anonymousID, err := strconv.ParseInt(headerValue, 10, 64)
	if err != nil || anonymousID <= 0 {
		return 0, errorsx.InvalidArgument("匿名推荐主体无效")
	}
	return anonymousID, nil
}

// createRecommendAnonymousActor 创建匿名推荐主体。
func (c *RecommendAnonymousActorCase) createRecommendAnonymousActor(ctx context.Context) (int64, error) {
	anonymousID := id.GenSnowflakeID()
	now := time.Now()
	query := c.Query(ctx).RecommendAnonymousActor
	err := query.WithContext(ctx).
		Omit(query.UserID, query.BindAt).
		Create(&models.RecommendAnonymousActor{
			AnonymousID: anonymousID,
			CreatedAt:   now,
			UpdatedAt:   now,
		})
	if err != nil {
		return 0, errorsx.Internal("创建匿名推荐主体失败").WithCause(err)
	}
	return anonymousID, nil
}

// ensureRecommendAnonymousActor 确保匿名主体记录存在并刷新活跃时间。
func (c *RecommendAnonymousActorCase) ensureRecommendAnonymousActor(ctx context.Context, anonymousID int64) error {
	// 匿名主体编号非法时，无需继续处理匿名会话记录。
	if anonymousID <= 0 {
		return nil
	}

	now := time.Now()
	query := c.Query(ctx).RecommendAnonymousActor
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.AnonymousID.Eq(anonymousID)))
	_, err := c.Find(ctx, opts...)
	if err != nil {
		// 记录不存在时，补建匿名主体并写入首次活跃时间。
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = query.WithContext(ctx).
				Omit(query.UserID, query.BindAt).
				Create(&models.RecommendAnonymousActor{
					AnonymousID: anonymousID,
					CreatedAt:   now,
					UpdatedAt:   now,
				})
			if err != nil {
				return errorsx.Internal("保存匿名推荐主体失败").WithCause(err)
			}
			return nil
		}
		return errorsx.Internal("查询匿名推荐主体失败").WithCause(err)
	}

	err = c.Update(ctx, &models.RecommendAnonymousActor{
		UpdatedAt: now,
	}, opts...)
	if err != nil {
		return errorsx.Internal("更新匿名推荐主体失败").WithCause(err)
	}
	return nil
}

// bindRecommendAnonymousActor 绑定匿名推荐主体到当前用户。
func (c *RecommendAnonymousActorCase) bindRecommendAnonymousActor(ctx context.Context, userID, anonymousID int64) error {
	// 当前未携带匿名主体或用户编号非法时，无需继续绑定。
	if userID <= 0 || anonymousID <= 0 {
		return nil
	}

	anonymousEventList, err := c.listRecommendEventsByActor(ctx, commonv1.RecommendActorType(_const.RECOMMEND_ACTOR_TYPE_ANONYMOUS), anonymousID)
	if err != nil {
		return err
	}

	needReplayToRecommend := true
	err = c.tx.Transaction(ctx, func(ctx context.Context) error {
		now := time.Now()
		query := c.Query(ctx).RecommendAnonymousActor
		opts := make([]repository.QueryOption, 0, 1)
		opts = append(opts, repository.Where(query.AnonymousID.Eq(anonymousID)))
		var actor *models.RecommendAnonymousActor
		actor, err = c.Find(ctx, opts...)
		if err != nil {
			// 记录不存在时，按“创建并绑定”路径补齐匿名主体。
			if errors.Is(err, gorm.ErrRecordNotFound) {
				err = query.WithContext(ctx).Create(&models.RecommendAnonymousActor{
					AnonymousID: anonymousID,
					UserID:      userID,
					CreatedAt:   now,
					UpdatedAt:   now,
					BindAt:      now,
				})
				if err != nil {
					return errorsx.Internal("绑定匿名推荐主体失败").WithCause(err)
				}
			} else {
				return errorsx.Internal("绑定匿名推荐主体失败").WithCause(err)
			}
		} else {
			// 匿名主体已绑定到其他用户时，不允许再串联到当前账号。
			if actor.UserID > 0 && actor.UserID != userID {
				return errorsx.Conflict("匿名推荐主体已绑定其他用户")
			}
			// 匿名主体已经绑定到当前用户时，只刷新绑定时间，不再重复迁移历史。
			if actor.UserID == userID {
				needReplayToRecommend = false
				err = c.Update(ctx, &models.RecommendAnonymousActor{
					UpdatedAt: now,
					BindAt:    now,
				}, opts...)
				if err != nil {
					return errorsx.Internal("绑定匿名推荐主体失败").WithCause(err)
				}
				return nil
			}

			err = c.Update(ctx, &models.RecommendAnonymousActor{
				UserID:    userID,
				UpdatedAt: now,
				BindAt:    now,
			}, opts...)
			if err != nil {
				return errorsx.Internal("绑定匿名推荐主体失败").WithCause(err)
			}
		}

		// 匿名主体完成绑定后，需要把匿名阶段积累的推荐历史一并迁移到登录用户。
		err = c.rebindRecommendRequestActor(ctx, userID, anonymousID)
		if err != nil {
			return err
		}
		err = c.rebindRecommendEventActor(ctx, userID, anonymousID)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	// 匿名历史首次迁移成功后，再异步把匿名阶段行为重放到登录用户的推荐系统主体下。
	if needReplayToRecommend {
		err = c.syncRecommendActorHistoryToRecommend(userID, anonymousEventList)
		if err != nil {
			log.Errorf("syncRecommendActorHistoryToRecommend %v", err)
		}
	}
	return nil
}

// listRecommendEventsByActor 查询指定推荐主体的历史事件列表。
func (c *RecommendAnonymousActorCase) listRecommendEventsByActor(
	ctx context.Context,
	actorType commonv1.RecommendActorType,
	actorID int64,
) ([]*models.RecommendEvent, error) {
	// 推荐主体编号非法时，不存在可迁移的历史事件。
	if actorID <= 0 {
		return []*models.RecommendEvent{}, nil
	}

	query := c.recommendEventRepo.Query(ctx).RecommendEvent
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Order(query.EventAt.Desc()))
	opts = append(opts, repository.Where(query.ActorType.Eq(int32(actorType))))
	opts = append(opts, repository.Where(query.ActorID.Eq(actorID)))
	list, err := c.recommendEventRepo.List(ctx, opts...)
	if err != nil {
		return nil, errorsx.Internal("查询匿名推荐事件失败").WithCause(err)
	}
	return list, nil
}

// syncRecommendActorHistoryToRecommend 异步回放匿名阶段历史到登录用户。
func (c *RecommendAnonymousActorCase) syncRecommendActorHistoryToRecommend(
	userID int64,
	eventList []*models.RecommendEvent,
) error {
	// 用户编号非法或历史事件为空时，无需继续回放历史。
	if userID <= 0 || len(eventList) == 0 {
		return nil
	}

	replayEventList := make([]*models.RecommendEvent, 0, len(eventList))
	for _, item := range eventList {
		// 匿名历史写入推荐系统前，先改写成登录用户主体，避免匿名身份继续向下游投递。
		if item == nil {
			continue
		}
		replayEvent := *item
		replayEvent.ActorType = _const.RECOMMEND_ACTOR_TYPE_USER
		replayEvent.ActorID = userID
		replayEventList = append(replayEventList, &replayEvent)
	}
	if len(replayEventList) == 0 {
		return nil
	}

	queue.DispatchRecommendEventList(replayEventList)

	return nil
}

// rebindRecommendRequestActor 将匿名主体下的推荐请求记录迁移到登录用户。
func (c *RecommendAnonymousActorCase) rebindRecommendRequestActor(ctx context.Context, userID, anonymousID int64) error {
	// 用户编号或匿名主体编号非法时，不存在可迁移的推荐请求记录。
	if userID <= 0 || anonymousID <= 0 {
		return nil
	}

	query := c.recommendRequestRepo.Query(ctx).RecommendRequest
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.ActorType.Eq(_const.RECOMMEND_ACTOR_TYPE_ANONYMOUS)))
	opts = append(opts, repository.Where(query.ActorID.Eq(anonymousID)))
	err := c.recommendRequestRepo.Update(ctx, &models.RecommendRequest{
		ActorType: _const.RECOMMEND_ACTOR_TYPE_USER,
		ActorID:   userID,
	}, opts...)
	if err != nil {
		return errorsx.Internal("迁移匿名推荐请求失败").WithCause(err)
	}
	return nil
}

// rebindRecommendEventActor 将匿名主体下的推荐事件记录迁移到登录用户。
func (c *RecommendAnonymousActorCase) rebindRecommendEventActor(ctx context.Context, userID, anonymousID int64) error {
	// 用户编号或匿名主体编号非法时，不存在可迁移的推荐事件记录。
	if userID <= 0 || anonymousID <= 0 {
		return nil
	}

	query := c.recommendEventRepo.Query(ctx).RecommendEvent
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.ActorType.Eq(_const.RECOMMEND_ACTOR_TYPE_ANONYMOUS)))
	opts = append(opts, repository.Where(query.ActorID.Eq(anonymousID)))
	err := c.recommendEventRepo.Update(ctx, &models.RecommendEvent{
		ActorType: _const.RECOMMEND_ACTOR_TYPE_USER,
		ActorID:   userID,
	}, opts...)
	if err != nil {
		return errorsx.Internal("迁移匿名推荐事件失败").WithCause(err)
	}
	return nil
}
