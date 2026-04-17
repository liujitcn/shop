package task

import (
	"context"
	"strconv"
	"strings"
	"time"

	"shop/api/gen/go/common"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	recommendCache "shop/pkg/recommend/cache"
	recommendEvent "shop/pkg/recommend/event"
	recommendCf "shop/pkg/recommend/offline/train/cf"
	recommendCtr "shop/pkg/recommend/offline/train/ctr"

	"github.com/liujitcn/gorm-kit/repo"
)

// parseRecommendTrainLookbackDaysArg 解析离线训练回看天数参数。
func parseRecommendTrainLookbackDaysArg(value string, defaultValue int) (int, error) {
	trimmedValue := strings.TrimSpace(value)
	if trimmedValue == "" {
		return defaultValue, nil
	}
	lookbackDays, err := strconv.Atoi(trimmedValue)
	if err != nil {
		return 0, errorsx.InvalidArgument("lookbackDays 格式错误")
	}
	// 回看窗口必须大于零，避免离线训练读到空时间范围。
	if lookbackDays <= 0 {
		return 0, errorsx.InvalidArgument("lookbackDays 必须大于 0")
	}
	return lookbackDays, nil
}

// parseRecommendTrainEpochArg 解析离线训练轮数参数。
func parseRecommendTrainEpochArg(value string, defaultValue int) (int, error) {
	trimmedValue := strings.TrimSpace(value)
	if trimmedValue == "" {
		return defaultValue, nil
	}
	epochCount, err := strconv.Atoi(trimmedValue)
	if err != nil {
		return 0, errorsx.InvalidArgument("epochs 格式错误")
	}
	// 训练轮数必须大于零，避免模型直接跳过拟合。
	if epochCount <= 0 {
		return 0, errorsx.InvalidArgument("epochs 必须大于 0")
	}
	return epochCount, nil
}

// parseRecommendTrainBackendArg 解析离线训练后端参数。
func parseRecommendTrainBackendArg(value string, defaultValue string) (string, error) {
	trimmedValue := strings.TrimSpace(strings.ToLower(value))
	// 未显式传入时，沿用调用方给定的默认后端。
	if trimmedValue == "" {
		return defaultValue, nil
	}
	switch trimmedValue {
	// 仅允许显式选择当前已接入的训练后端，避免任务参数写出无效值。
	case recommendCtr.BackendNative, recommendCtr.BackendGoMLX:
		return trimmedValue, nil
	default:
		return "", errorsx.InvalidArgument("backend 仅支持 native 或 gomlx")
	}
}

// loadRecommendGoodsActionSince 读取指定时间之后的推荐商品行为。
func loadRecommendGoodsActionSince(ctx context.Context, recommendGoodsActionRepo *data.RecommendGoodsActionRepo, startAt time.Time) ([]*models.RecommendGoodsAction, error) {
	query := recommendGoodsActionRepo.Query(ctx).RecommendGoodsAction
	opts := make([]repo.QueryOption, 0, 2)
	opts = append(opts, repo.Where(query.CreatedAt.Gte(startAt)))
	opts = append(opts, repo.Order(query.CreatedAt.Desc()))
	return recommendGoodsActionRepo.List(ctx, opts...)
}

// loadRecommendPutOnGoodsMap 读取当前全部上架商品映射。
func loadRecommendPutOnGoodsMap(ctx context.Context, goodsInfoRepo *data.GoodsInfoRepo) (map[string]*models.GoodsInfo, error) {
	list, err := loadPutOnGoodsList(ctx, goodsInfoRepo)
	if err != nil {
		return nil, err
	}
	result := make(map[string]*models.GoodsInfo, len(list))
	for _, item := range list {
		// 非法商品不进入离线训练发布范围。
		if item == nil || item.ID <= 0 {
			continue
		}
		result[strconv.FormatInt(item.ID, 10)] = item
	}
	return result, nil
}

// buildRecommendCollaborativeInteractions 将真实行为映射为 BPR 隐式正反馈。
func buildRecommendCollaborativeInteractions(actionList []*models.RecommendGoodsAction) []recommendCf.Interaction {
	result := make([]recommendCf.Interaction, 0, len(actionList))
	for _, item := range actionList {
		// 只使用登录态有效商品行为作为协同过滤正反馈。
		if item == nil || item.ActorType != recommendEvent.ActorTypeUser || item.ActorID <= 0 || item.GoodsID <= 0 {
			continue
		}
		weight := resolveRecommendCollaborativeActionWeight(item.EventType)
		if weight <= 0 {
			continue
		}
		result = append(result, recommendCf.Interaction{
			UserId: strconv.FormatInt(item.ActorID, 10),
			ItemId: strconv.FormatInt(item.GoodsID, 10),
			Weight: weight,
		})
	}
	return result
}

// resolveRecommendCollaborativeActionWeight 返回行为在协同过滤里的训练权重。
func resolveRecommendCollaborativeActionWeight(eventType int32) int {
	switch eventType {
	// 浏览和点击只提供弱正反馈，避免对偶然行为过拟合。
	case int32(common.RecommendGoodsActionType_VIEW), int32(common.RecommendGoodsActionType_CLICK):
		return 1
	// 收藏和加购代表更强购买意图，适当提高采样权重。
	case int32(common.RecommendGoodsActionType_COLLECT):
		return 2
	case int32(common.RecommendGoodsActionType_ADD_CART):
		return 3
	// 下单和支付代表最高质量反馈，作为强正样本放大训练占比。
	case int32(common.RecommendGoodsActionType_ORDER_CREATE):
		return 4
	case int32(common.RecommendGoodsActionType_ORDER_PAY):
		return 5
	default:
		return 0
	}
}

// buildRecommendCollaborativeDocuments 依据训练好的 BPR 模型生成用户级推荐结果。
func buildRecommendCollaborativeDocuments(model *recommendCf.Model, putOnGoodsMap map[string]*models.GoodsInfo, limit int) map[int64][]recommendCache.Score {
	result := make(map[int64][]recommendCache.Score)
	if model == nil || limit <= 0 {
		return result
	}
	disallowedItemIdSet := make(map[string]struct{})
	for _, itemId := range model.ItemIds() {
		if _, ok := putOnGoodsMap[itemId]; ok {
			continue
		}
		disallowedItemIdSet[itemId] = struct{}{}
	}
	for _, userId := range model.UserIds() {
		parsedUserId, err := strconv.ParseInt(userId, 10, 64)
		if err != nil || parsedUserId <= 0 {
			continue
		}
		scoredItemList := model.Recommend(userId, limit, disallowedItemIdSet)
		documentList := make([]recommendCache.Score, 0, len(scoredItemList))
		for _, item := range scoredItemList {
			goods, ok := putOnGoodsMap[item.ItemId]
			// 已下架商品或异常商品编号不继续写入缓存。
			if !ok || goods == nil {
				continue
			}
			documentList = append(documentList, recommendCache.Score{
				Id:        item.ItemId,
				Score:     float64(item.Score),
				Timestamp: maxRecommendTime(goods.UpdatedAt, goods.CreatedAt),
			})
		}
		result[parsedUserId] = documentList
	}
	return result
}
