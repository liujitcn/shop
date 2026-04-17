package task

import (
	"context"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"shop/api/gen/go/common"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	recommendCache "shop/pkg/recommend/cache"
	recommendContent "shop/pkg/recommend/content"
	recommendEvent "shop/pkg/recommend/event"
	recommendCtr "shop/pkg/recommend/offline/train/ctr"

	"github.com/liujitcn/gorm-kit/repo"
)

const (
	// recommendTrainBatchQuerySize 表示离线训练批量查询时的单批数量。
	recommendTrainBatchQuerySize = 500
)

// RecommendRankerActorKey 表示 ranker 缓存子集合的主体维度。
type RecommendRankerActorKey struct {
	Scene     int32
	ActorType int32
	ActorId   int64
}

// RecommendRankerAggregateStats 表示曝光与正反馈聚合统计。
type RecommendRankerAggregateStats struct {
	ExposureCount int
	PositiveCount int
}

// PositiveRate 返回当前聚合口径下的正样本比例。
func (s RecommendRankerAggregateStats) PositiveRate() float32 {
	// 没有曝光样本时，正样本率统一回退为零。
	if s.ExposureCount <= 0 {
		return 0
	}
	return float32(s.PositiveCount) / float32(s.ExposureCount)
}

// RecommendRankerTrainData 表示 ranker 真实训练与发布所需的数据快照。
type RecommendRankerTrainData struct {
	SampleList    []recommendCtr.Sample
	ActorStatsMap map[RecommendRankerActorKey]RecommendRankerAggregateStats
	RequestList   []*models.RecommendRequest
	RequestItems  []*models.RecommendRequestItem
	ActionList    []*models.RecommendGoodsAction
	GoodsMap      map[int64]*models.GoodsInfo
	PutOnGoodsMap map[int64]*models.GoodsInfo
}

// recommendRankerActorGoodsKey 表示主体场景与商品交叉特征键。
type recommendRankerActorGoodsKey struct {
	Scene     int32
	ActorType int32
	ActorId   int64
	GoodsId   int64
}

// recommendRankerSceneGoodsKey 表示场景与商品交叉特征键。
type recommendRankerSceneGoodsKey struct {
	Scene   int32
	GoodsId int64
}

// recommendRankerTrainingRow 表示 ranker 单条训练样本的中间结构。
type recommendRankerTrainingRow struct {
	ActorKey      RecommendRankerActorKey
	ActorGoodsKey recommendRankerActorGoodsKey
	SceneGoodsKey recommendRankerSceneGoodsKey
	Goods         *models.GoodsInfo
	IsPositive    bool
}

// ParseRecommendTrainTestRatioArg 解析离线训练验证集比例参数。
func ParseRecommendTrainTestRatioArg(value string, defaultValue float32) (float32, error) {
	trimmedValue := strings.TrimSpace(value)
	if trimmedValue == "" {
		return defaultValue, nil
	}
	ratio, err := strconv.ParseFloat(trimmedValue, 32)
	if err != nil {
		return 0, errorsx.InvalidArgument("testRatio 格式错误")
	}
	// 验证集比例必须落在合理开区间内，避免切分出空训练集或空验证集。
	if ratio <= 0 || ratio >= 1 {
		return 0, errorsx.InvalidArgument("testRatio 必须在 0 和 1 之间")
	}
	return float32(ratio), nil
}

// ParseRecommendTrainTrialCountArg 解析自动调参试验次数参数。
func ParseRecommendTrainTrialCountArg(value string, defaultValue int) (int, error) {
	trimmedValue := strings.TrimSpace(value)
	if trimmedValue == "" {
		return defaultValue, nil
	}
	trialCount, err := strconv.Atoi(trimmedValue)
	if err != nil {
		return 0, errorsx.InvalidArgument("trialCount 格式错误")
	}
	// 调参试验次数至少为 1，避免进入非法的零试验状态。
	if trialCount <= 0 {
		return 0, errorsx.InvalidArgument("trialCount 必须大于 0")
	}
	return trialCount, nil
}

// LoadRecommendRankerTrainData 读取 ranker 真实训练所需的数据快照。
func LoadRecommendRankerTrainData(
	ctx context.Context,
	recommendRequestRepo *data.RecommendRequestRepo,
	recommendRequestItemRepo *data.RecommendRequestItemRepo,
	recommendGoodsActionRepo *data.RecommendGoodsActionRepo,
	goodsInfoRepo *data.GoodsInfoRepo,
	startAt time.Time,
) (*RecommendRankerTrainData, error) {
	requestList, err := loadRecommendRankerRequestList(ctx, recommendRequestRepo, startAt)
	if err != nil {
		return nil, err
	}
	requestIds := make([]int64, 0, len(requestList))
	requestIdValues := make([]string, 0, len(requestList))
	for _, item := range requestList {
		// 无效请求不参与后续样本装配。
		if item == nil || item.ID <= 0 || strings.TrimSpace(item.RequestID) == "" {
			continue
		}
		requestIds = append(requestIds, item.ID)
		requestIdValues = append(requestIdValues, item.RequestID)
	}

	requestItemList, err := loadRecommendRankerRequestItems(ctx, recommendRequestItemRepo, requestIds)
	if err != nil {
		return nil, err
	}
	actionList, err := loadRecommendRankerGoodsActions(ctx, recommendGoodsActionRepo, requestIdValues, startAt)
	if err != nil {
		return nil, err
	}

	goodsIds := make([]int64, 0, len(requestItemList))
	for _, item := range requestItemList {
		// 明细商品编号非法时，不纳入训练商品维表加载范围。
		if item == nil || item.GoodsID <= 0 {
			continue
		}
		goodsIds = append(goodsIds, item.GoodsID)
	}
	goodsMap, err := loadRecommendRankerGoodsMap(ctx, goodsInfoRepo, goodsIds)
	if err != nil {
		return nil, err
	}
	putOnGoodsMap, err := loadRecommendRankerPutOnGoodsMap(ctx, goodsInfoRepo)
	if err != nil {
		return nil, err
	}

	sampleList, actorStatsMap := buildRecommendRankerTrainingSamples(requestList, requestItemList, actionList, goodsMap)
	return &RecommendRankerTrainData{
		SampleList:    sampleList,
		ActorStatsMap: actorStatsMap,
		RequestList:   requestList,
		RequestItems:  requestItemList,
		ActionList:    actionList,
		GoodsMap:      goodsMap,
		PutOnGoodsMap: putOnGoodsMap,
	}, nil
}

// BuildRecommendRankerDocuments 依据训练好的 AFM 模型生成 actor-scene 级缓存文档。
func BuildRecommendRankerDocuments(model *recommendCtr.AFM, trainData *RecommendRankerTrainData, limit int64) map[RecommendRankerActorKey][]recommendCache.Score {
	result := make(map[RecommendRankerActorKey][]recommendCache.Score)
	// 模型为空、训练快照为空或发布上限非法时，不生成 ranker 文档。
	if model == nil || trainData == nil || limit <= 0 {
		return result
	}
	if len(trainData.ActorStatsMap) == 0 || len(trainData.PutOnGoodsMap) == 0 {
		return result
	}

	requestMap := make(map[int64]*models.RecommendRequest, len(trainData.RequestList))
	for _, item := range trainData.RequestList {
		// actor-scene 键构建失败的请求不参与推理特征聚合。
		if item == nil || item.ID <= 0 || item.ActorType != recommendEvent.ActorTypeUser || item.ActorID <= 0 || item.Scene <= 0 {
			continue
		}
		requestMap[item.ID] = item
	}
	actionSignalMap := buildRecommendRankerActionSignalMap(trainData.ActionList)
	actorGoodsStatsMap := make(map[recommendRankerActorGoodsKey]RecommendRankerAggregateStats)
	sceneGoodsStatsMap := make(map[recommendRankerSceneGoodsKey]RecommendRankerAggregateStats)
	for _, item := range trainData.RequestItems {
		// 明细缺失、主请求缺失或商品不在发布维表中时，不构造历史交叉特征。
		if item == nil || item.GoodsID <= 0 {
			continue
		}
		requestEntity, ok := requestMap[item.RecommendRequestID]
		if !ok || requestEntity == nil {
			continue
		}
		if _, ok = trainData.GoodsMap[item.GoodsID]; !ok {
			continue
		}

		actorGoodsKey := recommendRankerActorGoodsKey{
			Scene:     requestEntity.Scene,
			ActorType: requestEntity.ActorType,
			ActorId:   requestEntity.ActorID,
			GoodsId:   item.GoodsID,
		}
		sceneGoodsKey := recommendRankerSceneGoodsKey{
			Scene:   requestEntity.Scene,
			GoodsId: item.GoodsID,
		}
		actionKey := buildRecommendRankerActionKey(requestEntity.RequestID, item.GoodsID)
		isPositive := actionKey != "" && actionSignalMap[actionKey]
		increaseRecommendRankerActorGoodsStats(actorGoodsStatsMap, actorGoodsKey, isPositive)
		increaseRecommendRankerSceneGoodsStats(sceneGoodsStatsMap, sceneGoodsKey, isPositive)
	}

	activeGoodsList := sortRecommendRankerGoods(trainData.PutOnGoodsMap)
	for actorKey, actorStats := range trainData.ActorStatsMap {
		// 非登录态或非法主体键不发布 ranker 子集合。
		if actorKey.ActorType != recommendEvent.ActorTypeUser || actorKey.ActorId <= 0 || actorKey.Scene <= 0 {
			continue
		}
		sampleList := make([]recommendCtr.Sample, 0, len(activeGoodsList))
		for _, goods := range activeGoodsList {
			// 推理阶段商品为空时，直接跳过异常商品。
			if goods == nil || goods.ID <= 0 {
				continue
			}
			actorGoodsKey := recommendRankerActorGoodsKey{
				Scene:     actorKey.Scene,
				ActorType: actorKey.ActorType,
				ActorId:   actorKey.ActorId,
				GoodsId:   goods.ID,
			}
			sceneGoodsKey := recommendRankerSceneGoodsKey{
				Scene:   actorKey.Scene,
				GoodsId: goods.ID,
			}
			sampleList = append(sampleList, buildRecommendRankerInferenceSample(
				actorKey,
				goods,
				actorStats,
				actorGoodsStatsMap[actorGoodsKey],
				sceneGoodsStatsMap[sceneGoodsKey],
			))
		}
		scoreList := model.PredictBatch(sampleList)
		documentList := make([]recommendCache.Score, 0, len(scoreList))
		for index, item := range scoreList {
			// 预测批次与商品顺序不一致时，直接结束当前主体的文档装配。
			if index >= len(activeGoodsList) {
				break
			}
			goods := activeGoodsList[index]
			if goods == nil || goods.ID <= 0 {
				continue
			}
			documentList = append(documentList, recommendCache.Score{
				Id:        strconv.FormatInt(goods.ID, 10),
				Score:     float64(item),
				Timestamp: maxRecommendRankerTime(goods.UpdatedAt, goods.CreatedAt),
			})
		}
		recommendCache.SortDocuments(documentList)
		// 每个 actor-scene 子集合只保留 TopN 商品。
		if int64(len(documentList)) > limit {
			documentList = documentList[:limit]
		}
		result[actorKey] = documentList
	}
	return result
}

// CountRecommendRankerPositiveSamples 统计 ranker 训练正样本数量。
func CountRecommendRankerPositiveSamples(sampleList []recommendCtr.Sample) int {
	total := 0
	for _, item := range sampleList {
		// 目标标签为正时，累加正样本数量。
		if item.Target > 0 {
			total++
		}
	}
	return total
}

// loadRecommendRankerRequestList 加载指定时间之后的推荐请求主表记录。
func loadRecommendRankerRequestList(ctx context.Context, recommendRequestRepo *data.RecommendRequestRepo, startAt time.Time) ([]*models.RecommendRequest, error) {
	query := recommendRequestRepo.Query(ctx).RecommendRequest
	opts := make([]repo.QueryOption, 0, 3)
	opts = append(opts, repo.Where(query.ActorType.Eq(recommendEvent.ActorTypeUser)))
	opts = append(opts, repo.Where(query.CreatedAt.Gte(startAt)))
	opts = append(opts, repo.Order(query.CreatedAt.Desc()))
	return recommendRequestRepo.List(ctx, opts...)
}

// loadRecommendRankerRequestItems 按推荐请求主表编号批量加载逐商品明细。
func loadRecommendRankerRequestItems(ctx context.Context, recommendRequestItemRepo *data.RecommendRequestItemRepo, recommendRequestIds []int64) ([]*models.RecommendRequestItem, error) {
	result := make([]*models.RecommendRequestItem, 0)
	if len(recommendRequestIds) == 0 {
		return result, nil
	}
	query := recommendRequestItemRepo.Query(ctx).RecommendRequestItem
	for _, chunk := range chunkRecommendInt64s(recommendRequestIds, recommendTrainBatchQuerySize) {
		opts := make([]repo.QueryOption, 0, 1)
		opts = append(opts, repo.Where(query.RecommendRequestID.In(chunk...)))
		list, err := recommendRequestItemRepo.List(ctx, opts...)
		if err != nil {
			return nil, err
		}
		result = append(result, list...)
	}
	return result, nil
}

// loadRecommendRankerGoodsActions 按推荐请求编号批量加载行为事实。
func loadRecommendRankerGoodsActions(ctx context.Context, recommendGoodsActionRepo *data.RecommendGoodsActionRepo, requestIds []string, startAt time.Time) ([]*models.RecommendGoodsAction, error) {
	result := make([]*models.RecommendGoodsAction, 0)
	if len(requestIds) == 0 {
		return result, nil
	}
	query := recommendGoodsActionRepo.Query(ctx).RecommendGoodsAction
	for _, chunk := range chunkRecommendStrings(requestIds, recommendTrainBatchQuerySize) {
		opts := make([]repo.QueryOption, 0, 3)
		opts = append(opts, repo.Where(query.RequestID.In(chunk...)))
		opts = append(opts, repo.Where(query.CreatedAt.Gte(startAt)))
		opts = append(opts, repo.Order(query.CreatedAt.Desc()))
		list, err := recommendGoodsActionRepo.List(ctx, opts...)
		if err != nil {
			return nil, err
		}
		result = append(result, list...)
	}
	return result, nil
}

// loadRecommendRankerGoodsMap 按商品编号批量加载商品维表。
func loadRecommendRankerGoodsMap(ctx context.Context, goodsInfoRepo *data.GoodsInfoRepo, goodsIds []int64) (map[int64]*models.GoodsInfo, error) {
	result := make(map[int64]*models.GoodsInfo)
	if len(goodsIds) == 0 {
		return result, nil
	}
	query := goodsInfoRepo.Query(ctx).GoodsInfo
	for _, chunk := range chunkRecommendInt64s(goodsIds, recommendTrainBatchQuerySize) {
		opts := make([]repo.QueryOption, 0, 1)
		opts = append(opts, repo.Where(query.ID.In(chunk...)))
		list, err := goodsInfoRepo.List(ctx, opts...)
		if err != nil {
			return nil, err
		}
		for _, item := range list {
			// 非法商品记录不纳入训练维表。
			if item == nil || item.ID <= 0 {
				continue
			}
			result[item.ID] = item
		}
	}
	return result, nil
}

// loadRecommendRankerPutOnGoodsMap 查询当前可发布的上架商品映射。
func loadRecommendRankerPutOnGoodsMap(ctx context.Context, goodsInfoRepo *data.GoodsInfoRepo) (map[int64]*models.GoodsInfo, error) {
	query := goodsInfoRepo.Query(ctx).GoodsInfo
	opts := make([]repo.QueryOption, 0, 2)
	opts = append(opts, repo.Where(query.Status.Eq(int32(common.GoodsStatus_PUT_ON))))
	opts = append(opts, repo.Order(query.CreatedAt.Desc()))
	list, err := goodsInfoRepo.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	result := make(map[int64]*models.GoodsInfo, len(list))
	for _, item := range list {
		// 非法商品不进入 ranker 发布范围。
		if item == nil || item.ID <= 0 {
			continue
		}
		result[item.ID] = item
	}
	return result, nil
}

// buildRecommendRankerTrainingSamples 基于真实曝光与行为事实构建 AFM 训练样本。
func buildRecommendRankerTrainingSamples(
	requestList []*models.RecommendRequest,
	requestItemList []*models.RecommendRequestItem,
	actionList []*models.RecommendGoodsAction,
	goodsMap map[int64]*models.GoodsInfo,
) ([]recommendCtr.Sample, map[RecommendRankerActorKey]RecommendRankerAggregateStats) {
	requestMap := make(map[int64]*models.RecommendRequest, len(requestList))
	for _, item := range requestList {
		// 只使用登录态且带场景的推荐请求参与 ranker 训练。
		if item == nil || item.ID <= 0 || item.ActorType != recommendEvent.ActorTypeUser || item.ActorID <= 0 || item.Scene <= 0 {
			continue
		}
		requestMap[item.ID] = item
	}
	actionSignalMap := buildRecommendRankerActionSignalMap(actionList)

	actorStatsMap := make(map[RecommendRankerActorKey]RecommendRankerAggregateStats)
	actorGoodsStatsMap := make(map[recommendRankerActorGoodsKey]RecommendRankerAggregateStats)
	sceneGoodsStatsMap := make(map[recommendRankerSceneGoodsKey]RecommendRankerAggregateStats)
	rowList := make([]recommendRankerTrainingRow, 0, len(requestItemList))
	for _, item := range requestItemList {
		// 逐商品明细为空或商品编号非法时，不生成训练样本。
		if item == nil || item.GoodsID <= 0 {
			continue
		}
		requestEntity, ok := requestMap[item.RecommendRequestID]
		if !ok || requestEntity == nil {
			continue
		}
		goods, ok := goodsMap[item.GoodsID]
		if !ok || goods == nil {
			continue
		}

		actorKey := RecommendRankerActorKey{
			Scene:     requestEntity.Scene,
			ActorType: requestEntity.ActorType,
			ActorId:   requestEntity.ActorID,
		}
		actorGoodsKey := recommendRankerActorGoodsKey{
			Scene:     requestEntity.Scene,
			ActorType: requestEntity.ActorType,
			ActorId:   requestEntity.ActorID,
			GoodsId:   item.GoodsID,
		}
		sceneGoodsKey := recommendRankerSceneGoodsKey{
			Scene:   requestEntity.Scene,
			GoodsId: item.GoodsID,
		}
		actionKey := buildRecommendRankerActionKey(requestEntity.RequestID, item.GoodsID)
		isPositive := actionKey != "" && actionSignalMap[actionKey]

		increaseRecommendRankerStats(actorStatsMap, actorKey, isPositive)
		increaseRecommendRankerActorGoodsStats(actorGoodsStatsMap, actorGoodsKey, isPositive)
		increaseRecommendRankerSceneGoodsStats(sceneGoodsStatsMap, sceneGoodsKey, isPositive)
		rowList = append(rowList, recommendRankerTrainingRow{
			ActorKey:      actorKey,
			ActorGoodsKey: actorGoodsKey,
			SceneGoodsKey: sceneGoodsKey,
			Goods:         goods,
			IsPositive:    isPositive,
		})
	}

	sampleList := make([]recommendCtr.Sample, 0, len(rowList))
	for _, row := range rowList {
		sampleList = append(sampleList, buildRecommendRankerSample(
			row,
			actorStatsMap[row.ActorKey],
			actorGoodsStatsMap[row.ActorGoodsKey],
			sceneGoodsStatsMap[row.SceneGoodsKey],
		))
	}
	return sampleList, actorStatsMap
}

// buildRecommendRankerActionSignalMap 将行为事实规整为请求商品级正反馈索引。
func buildRecommendRankerActionSignalMap(actionList []*models.RecommendGoodsAction) map[string]bool {
	result := make(map[string]bool, len(actionList))
	for _, item := range actionList {
		// 匿名行为、缺失请求编号或商品编号的记录不参与监督标签构建。
		if item == nil || item.ActorType != recommendEvent.ActorTypeUser || item.ActorID <= 0 || strings.TrimSpace(item.RequestID) == "" || item.GoodsID <= 0 {
			continue
		}
		actionKey := buildRecommendRankerActionKey(item.RequestID, item.GoodsID)
		if actionKey == "" {
			continue
		}
		// 当前行为不属于可视为正反馈的动作时，不写入标签索引。
		if !isRecommendRankerPositiveEvent(item.EventType) {
			continue
		}
		result[actionKey] = true
	}
	return result
}

// isRecommendRankerPositiveEvent 判断行为是否可作为 ranker 正样本。
func isRecommendRankerPositiveEvent(eventType int32) bool {
	switch eventType {
	// 点击及更强行为都视为排序阶段的正反馈。
	case int32(common.RecommendGoodsActionType_CLICK),
		int32(common.RecommendGoodsActionType_COLLECT),
		int32(common.RecommendGoodsActionType_ADD_CART),
		int32(common.RecommendGoodsActionType_ORDER_CREATE),
		int32(common.RecommendGoodsActionType_ORDER_PAY):
		return true
	default:
		return false
	}
}

// buildRecommendRankerActionKey 构建请求商品级行为索引键。
func buildRecommendRankerActionKey(requestId string, goodsId int64) string {
	trimmedRequestId := strings.TrimSpace(requestId)
	// 请求编号或商品编号非法时，不生成行为索引键。
	if trimmedRequestId == "" || goodsId <= 0 {
		return ""
	}
	return trimmedRequestId + ":" + strconv.FormatInt(goodsId, 10)
}

// increaseRecommendRankerStats 累加 actor-scene 统计。
func increaseRecommendRankerStats(statsMap map[RecommendRankerActorKey]RecommendRankerAggregateStats, key RecommendRankerActorKey, isPositive bool) {
	current := statsMap[key]
	current.ExposureCount++
	// 当前曝光最终产生了点击或转化时，再累加正样本数。
	if isPositive {
		current.PositiveCount++
	}
	statsMap[key] = current
}

// increaseRecommendRankerActorGoodsStats 累加 actor-scene-goods 统计。
func increaseRecommendRankerActorGoodsStats(statsMap map[recommendRankerActorGoodsKey]RecommendRankerAggregateStats, key recommendRankerActorGoodsKey, isPositive bool) {
	current := statsMap[key]
	current.ExposureCount++
	// 当前商品在该主体场景下命中正反馈时，再累加正样本数。
	if isPositive {
		current.PositiveCount++
	}
	statsMap[key] = current
}

// increaseRecommendRankerSceneGoodsStats 累加 scene-goods 统计。
func increaseRecommendRankerSceneGoodsStats(statsMap map[recommendRankerSceneGoodsKey]RecommendRankerAggregateStats, key recommendRankerSceneGoodsKey, isPositive bool) {
	current := statsMap[key]
	current.ExposureCount++
	// 当前商品在该场景下命中正反馈时，再累加正样本数。
	if isPositive {
		current.PositiveCount++
	}
	statsMap[key] = current
}

// buildRecommendRankerSample 构建单条 AFM 训练样本。
func buildRecommendRankerSample(
	row recommendRankerTrainingRow,
	actorStats RecommendRankerAggregateStats,
	actorGoodsStats RecommendRankerAggregateStats,
	sceneGoodsStats RecommendRankerAggregateStats,
) recommendCtr.Sample {
	sample := buildRecommendRankerInferenceSample(row.ActorKey, row.Goods, actorStats, actorGoodsStats, sceneGoodsStats)
	// 当前曝光产生了点击或转化时，标签置为正样本。
	if row.IsPositive {
		sample.Target = 1
	}
	return sample
}

// buildRecommendRankerInferenceSample 构建静态 actor-scene-goods 推理样本。
func buildRecommendRankerInferenceSample(
	actorKey RecommendRankerActorKey,
	goods *models.GoodsInfo,
	actorStats RecommendRankerAggregateStats,
	actorGoodsStats RecommendRankerAggregateStats,
	sceneGoodsStats RecommendRankerAggregateStats,
) recommendCtr.Sample {
	return recommendCtr.Sample{
		UserId: strconv.FormatInt(actorKey.ActorId, 10),
		ItemId: strconv.FormatInt(goods.ID, 10),
		UserLabels: []recommendCtr.Label{
			{Name: "actor_type:user"},
			{Name: "actor_scene_exposure_count", Value: normalizeRecommendRankerCount(actorStats.ExposureCount)},
			{Name: "actor_scene_positive_rate", Value: actorStats.PositiveRate()},
		},
		ItemLabels: []recommendCtr.Label{
			{Name: "goods_category:" + strconv.FormatInt(goods.CategoryID, 10)},
			{Name: "goods_price_bucket:" + strconv.Itoa(bucketRecommendRankerPrice(goods.Price))},
			{Name: "goods_sale_bucket:" + strconv.Itoa(bucketRecommendRankerSale(goods.RealSaleNum+goods.InitSaleNum))},
			{Name: "goods_price", Value: normalizeRecommendRankerValue(goods.Price)},
			{Name: "goods_sale_num", Value: normalizeRecommendRankerValue(goods.RealSaleNum + goods.InitSaleNum)},
			{Name: "goods_age_days", Value: normalizeRecommendRankerAge(goods.CreatedAt, goods.UpdatedAt)},
		},
		ContextLabels: []recommendCtr.Label{
			{Name: "scene:" + strconv.FormatInt(int64(actorKey.Scene), 10)},
			{Name: "actor_goods_exposure_count", Value: normalizeRecommendRankerCount(actorGoodsStats.ExposureCount)},
			{Name: "actor_goods_positive_rate", Value: actorGoodsStats.PositiveRate()},
			{Name: "scene_goods_exposure_count", Value: normalizeRecommendRankerCount(sceneGoodsStats.ExposureCount)},
			{Name: "scene_goods_positive_rate", Value: sceneGoodsStats.PositiveRate()},
		},
		Embeddings: []recommendCtr.Embedding{
			{
				Name:  recommendContent.DefaultEmbeddingName,
				Value: buildRecommendRankerContentVector(goods),
			},
		},
	}
}

// buildRecommendRankerContentVector 构建 ranker 训练与推理共享的商品内容向量。
func buildRecommendRankerContentVector(goods *models.GoodsInfo) []float32 {
	if goods == nil || goods.ID <= 0 {
		return []float32{}
	}
	return recommendContent.BuildDocumentVector(recommendContent.Document{
		Id:         goods.ID,
		CategoryId: goods.CategoryID,
		Name:       goods.Name,
		Desc:       goods.Desc,
		Detail:     goods.Detail,
		Price:      goods.Price,
		SaleNum:    goods.RealSaleNum + goods.InitSaleNum,
		CreatedAt:  goods.CreatedAt,
		UpdatedAt:  goods.UpdatedAt,
	}, recommendContent.DefaultDimension).Values
}

// sortRecommendRankerGoods 将可发布商品按商品编号升序排列，保证预测输入顺序稳定。
func sortRecommendRankerGoods(goodsMap map[int64]*models.GoodsInfo) []*models.GoodsInfo {
	result := make([]*models.GoodsInfo, 0, len(goodsMap))
	for _, item := range goodsMap {
		// 非法商品不进入排序阶段的稳定输入序列。
		if item == nil || item.ID <= 0 {
			continue
		}
		result = append(result, item)
	}
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})
	return result
}

// normalizeRecommendRankerCount 归一化曝光类计数特征。
func normalizeRecommendRankerCount(value int) float32 {
	if value <= 0 {
		return 0
	}
	return float32(math.Log1p(float64(value)))
}

// normalizeRecommendRankerValue 归一化价格、销量等数值特征。
func normalizeRecommendRankerValue(value int64) float32 {
	if value <= 0 {
		return 0
	}
	return float32(math.Log1p(float64(value)))
}

// normalizeRecommendRankerAge 归一化商品年龄特征。
func normalizeRecommendRankerAge(createdAt time.Time, updatedAt time.Time) float32 {
	baseTime := maxRecommendRankerTime(updatedAt, createdAt)
	// 商品时间为空时，不附加年龄特征。
	if baseTime.IsZero() {
		return 0
	}
	ageHours := time.Since(baseTime).Hours()
	if ageHours < 0 {
		return 0
	}
	return float32(math.Log1p(ageHours / 24))
}

// bucketRecommendRankerPrice 归整价格分桶。
func bucketRecommendRankerPrice(price int64) int {
	switch {
	// 免费或异常价格统一归到最低桶。
	case price <= 0:
		return 0
	case price < 2000:
		return 1
	case price < 5000:
		return 2
	case price < 10000:
		return 3
	default:
		return 4
	}
}

// bucketRecommendRankerSale 归整销量分桶。
func bucketRecommendRankerSale(saleNum int64) int {
	switch {
	// 没有销量历史时，统一归到冷启动桶。
	case saleNum <= 0:
		return 0
	case saleNum < 10:
		return 1
	case saleNum < 50:
		return 2
	case saleNum < 200:
		return 3
	default:
		return 4
	}
}

// maxRecommendRankerTime 返回两个时间中的较晚者。
func maxRecommendRankerTime(left time.Time, right time.Time) time.Time {
	switch {
	// 左侧时间为空时，直接回退到右侧。
	case left.IsZero():
		return right
	// 右侧时间为空时，直接保留左侧。
	case right.IsZero():
		return left
	// 两个时间都有效时，返回较新的一个。
	case left.After(right):
		return left
	default:
		return right
	}
}

// chunkRecommendInt64s 将 int64 列表拆成固定大小的批次。
func chunkRecommendInt64s(values []int64, chunkSize int) [][]int64 {
	dedupedValues := dedupeRecommendInt64s(values)
	if len(dedupedValues) == 0 {
		return [][]int64{}
	}
	if chunkSize <= 0 {
		chunkSize = len(dedupedValues)
	}
	result := make([][]int64, 0, (len(dedupedValues)+chunkSize-1)/chunkSize)
	for start := 0; start < len(dedupedValues); start += chunkSize {
		end := start + chunkSize
		if end > len(dedupedValues) {
			end = len(dedupedValues)
		}
		result = append(result, dedupedValues[start:end])
	}
	return result
}

// chunkRecommendStrings 将字符串列表拆成固定大小的批次。
func chunkRecommendStrings(values []string, chunkSize int) [][]string {
	dedupedValues := dedupeRecommendStrings(values)
	if len(dedupedValues) == 0 {
		return [][]string{}
	}
	if chunkSize <= 0 {
		chunkSize = len(dedupedValues)
	}
	result := make([][]string, 0, (len(dedupedValues)+chunkSize-1)/chunkSize)
	for start := 0; start < len(dedupedValues); start += chunkSize {
		end := start + chunkSize
		if end > len(dedupedValues) {
			end = len(dedupedValues)
		}
		result = append(result, dedupedValues[start:end])
	}
	return result
}

// dedupeRecommendInt64s 对 int64 列表做稳定去重。
func dedupeRecommendInt64s(values []int64) []int64 {
	result := make([]int64, 0, len(values))
	valueSet := make(map[int64]struct{}, len(values))
	for _, item := range values {
		// 非法编号不进入批量查询。
		if item <= 0 {
			continue
		}
		if _, ok := valueSet[item]; ok {
			continue
		}
		valueSet[item] = struct{}{}
		result = append(result, item)
	}
	return result
}

// dedupeRecommendStrings 对字符串列表做稳定去重。
func dedupeRecommendStrings(values []string) []string {
	result := make([]string, 0, len(values))
	valueSet := make(map[string]struct{}, len(values))
	for _, item := range values {
		trimmedValue := strings.TrimSpace(item)
		// 空字符串不进入批量查询。
		if trimmedValue == "" {
			continue
		}
		if _, ok := valueSet[trimmedValue]; ok {
			continue
		}
		valueSet[trimmedValue] = struct{}{}
		result = append(result, trimmedValue)
	}
	return result
}
