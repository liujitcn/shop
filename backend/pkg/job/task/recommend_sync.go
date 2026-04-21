package task

import (
	"context"
	"fmt"
	"shop/pkg/gen/models"
	"strconv"

	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	pkgRecommend "shop/pkg/recommend"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/liujitcn/gorm-kit/repo"
)

const recommendSyncDefaultBatchSize = 200

// RecommendSync 推荐系统主数据同步任务。
type RecommendSync struct {
	baseUserRepo  *data.BaseUserRepo
	goodsInfoRepo *data.GoodsInfoRepo
	recommend     *pkgRecommend.Recommend
	ctx           context.Context
}

// NewRecommendSync 创建推荐系统主数据同步任务实例。
func NewRecommendSync(baseUserRepo *data.BaseUserRepo, goodsInfoRepo *data.GoodsInfoRepo, recommendClient *pkgRecommend.Recommend) *RecommendSync {
	return &RecommendSync{
		baseUserRepo:  baseUserRepo,
		goodsInfoRepo: goodsInfoRepo,
		recommend:     recommendClient,
		ctx:           context.Background(),
	}
}

// Exec 执行推荐系统主数据同步。
func (t *RecommendSync) Exec(args map[string]string) ([]string, error) {
	log.Infof("Job RecommendSync Exec %+v", args)

	// 推荐系统未启用时，只记录跳过结果，避免空配置导致任务报错。
	if t.recommend == nil || !t.recommend.Enabled() {
		return []string{"推荐系统未启用，已跳过主数据同步"}, nil
	}

	batchSize, err := parsePositiveIntArg(args["batchSize"], recommendSyncDefaultBatchSize, "batchSize")
	if err != nil {
		return []string{err.Error()}, err
	}

	var syncedUserCount int
	syncedUserCount, err = t.syncBaseUser(batchSize)
	if err != nil {
		return []string{err.Error()}, err
	}
	var syncedGoodsCount int
	syncedGoodsCount, err = t.syncGoodsInfo(batchSize)
	if err != nil {
		return []string{err.Error()}, err
	}

	return []string{fmt.Sprintf("推荐系统主数据同步完成: 用户 %d 条，商品 %d 条，批次 %d 条", syncedUserCount, syncedGoodsCount, batchSize)}, nil
}

// syncBaseUser 分批同步后台用户快照到推荐系统。
func (t *RecommendSync) syncBaseUser(batchSize int) (int, error) {
	query := t.baseUserRepo.Query(t.ctx).BaseUser
	total := 0
	existingUserIds, err := t.recommend.LoadBaseUserIds(t.ctx, batchSize)
	if err != nil {
		return total, fmt.Errorf("加载推荐系统用户索引失败: %w", err)
	}
	staleUserIds := cloneIdSet(existingUserIds)

	for offset := 0; ; offset += batchSize {
		opts := make([]repo.QueryOption, 0, 3)
		opts = append(opts, repo.Order(query.ID.Asc()))
		opts = append(opts, repo.Offset(offset))
		opts = append(opts, repo.Limit(batchSize))

		userList := make([]*models.BaseUser, batchSize)
		userList, err = t.baseUserRepo.List(t.ctx, opts...)
		if err != nil {
			return total, fmt.Errorf("查询推荐系统用户同步批次失败: %w", err)
		}
		// 当前批次没有数据时，说明用户全量遍历已经完成。
		if len(userList) == 0 {
			break
		}

		err = t.recommend.SyncBaseUsers(t.ctx, userList, existingUserIds, staleUserIds)
		if err != nil {
			return total, fmt.Errorf("同步推荐系统用户数据失败: %w", err)
		}
		total += len(userList)
		// 当前批次数量未满时，说明已经到达最后一批用户数据。
		if len(userList) < batchSize {
			break
		}
	}
	err = t.recommend.DeleteBaseUsers(t.ctx, staleUserIds)
	if err != nil {
		return total, fmt.Errorf("清理推荐系统冗余用户数据失败: %w", err)
	}
	return total, nil
}

// syncGoodsInfo 分批同步商品快照到推荐系统。
func (t *RecommendSync) syncGoodsInfo(batchSize int) (int, error) {
	query := t.goodsInfoRepo.Query(t.ctx).GoodsInfo
	total := 0
	existingItemIds, err := t.recommend.LoadGoodsInfoIds(t.ctx, batchSize)
	if err != nil {
		return total, fmt.Errorf("加载推荐系统商品索引失败: %w", err)
	}
	staleItemIds := cloneIdSet(existingItemIds)

	for offset := 0; ; offset += batchSize {
		opts := make([]repo.QueryOption, 0, 3)
		opts = append(opts, repo.Order(query.ID.Asc()))
		opts = append(opts, repo.Offset(offset))
		opts = append(opts, repo.Limit(batchSize))

		goodsList := make([]*models.GoodsInfo, batchSize)
		goodsList, err = t.goodsInfoRepo.List(t.ctx, opts...)
		if err != nil {
			return total, fmt.Errorf("查询推荐系统商品同步批次失败: %w", err)
		}
		// 当前批次没有数据时，说明商品全量遍历已经完成。
		if len(goodsList) == 0 {
			break
		}

		err = t.recommend.SyncGoodsInfos(t.ctx, goodsList, existingItemIds, staleItemIds)
		if err != nil {
			return total, fmt.Errorf("同步推荐系统商品数据失败: %w", err)
		}
		total += len(goodsList)
		// 当前批次数量未满时，说明已经到达最后一批商品数据。
		if len(goodsList) < batchSize {
			break
		}
	}
	err = t.recommend.DeleteGoodsInfos(t.ctx, staleItemIds)
	if err != nil {
		return total, fmt.Errorf("清理推荐系统冗余商品数据失败: %w", err)
	}
	return total, nil
}

// parsePositiveIntArg 解析正整数任务参数，未传时回退默认值。
func parsePositiveIntArg(rawValue string, defaultValue int, argName string) (int, error) {
	// 未传参数时，直接回退到默认值。
	if rawValue == "" {
		return defaultValue, nil
	}

	value, err := strconv.Atoi(rawValue)
	if err != nil {
		return 0, errorsx.InvalidArgument(fmt.Sprintf("%s 必须是正整数", argName))
	}
	// 参数小于等于 0 时，直接返回明确错误，避免任务进入死循环或空批次。
	if value <= 0 {
		return 0, errorsx.InvalidArgument(fmt.Sprintf("%s 必须大于 0", argName))
	}
	return value, nil
}

// cloneIdSet 复制一份编号集合，避免后续清理候选和存在索引互相污染。
func cloneIdSet(source map[string]struct{}) map[string]struct{} {
	cloned := make(map[string]struct{}, len(source))
	for id := range source {
		cloned[id] = struct{}{}
	}
	return cloned
}
