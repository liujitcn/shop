package task

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"time"

	"shop/api/gen/go/common"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/liujitcn/gorm-kit/repo"
)

type recommendEvalRequestMeta struct {
	scene     int32
	requestId string
}

type recommendEvalRankedGoods struct {
	goodsId   int64
	position  int32
	relevance int32
}

type recommendEvalSceneMetric struct {
	requestCount         int64
	exposureCount        int64
	clickCount           int64
	orderCount           int64
	payCount             int64
	positiveRequestCount int64
	positiveGoodsCount   int64
	evalRequestCount     int64
	precisionSum         float64
	recallSum            float64
	ndcgSum              float64
}

// RecommendEvalReport 推荐离线评估报告任务。
type RecommendEvalReport struct {
	tx                        data.Transaction
	recommendEvalReportRepo   *data.RecommendEvalReportRepo
	recommendModelVersionRepo *data.RecommendModelVersionRepo
	recommendRequestRepo      *data.RecommendRequestRepo
	recommendRequestItemRepo  *data.RecommendRequestItemRepo
	recommendExposureRepo     *data.RecommendExposureRepo
	recommendExposureItemRepo *data.RecommendExposureItemRepo
	recommendGoodsActionRepo  *data.RecommendGoodsActionRepo
	ctx                       context.Context
}

// NewRecommendEvalReport 创建推荐离线评估报告任务实例。
func NewRecommendEvalReport(
	tx data.Transaction,
	recommendEvalReportRepo *data.RecommendEvalReportRepo,
	recommendModelVersionRepo *data.RecommendModelVersionRepo,
	recommendRequestRepo *data.RecommendRequestRepo,
	recommendRequestItemRepo *data.RecommendRequestItemRepo,
	recommendExposureRepo *data.RecommendExposureRepo,
	recommendExposureItemRepo *data.RecommendExposureItemRepo,
	recommendGoodsActionRepo *data.RecommendGoodsActionRepo,
) *RecommendEvalReport {
	return &RecommendEvalReport{
		tx:                        tx,
		recommendEvalReportRepo:   recommendEvalReportRepo,
		recommendModelVersionRepo: recommendModelVersionRepo,
		recommendRequestRepo:      recommendRequestRepo,
		recommendRequestItemRepo:  recommendRequestItemRepo,
		recommendExposureRepo:     recommendExposureRepo,
		recommendExposureItemRepo: recommendExposureItemRepo,
		recommendGoodsActionRepo:  recommendGoodsActionRepo,
		ctx:                       context.Background(),
	}
}

// Exec 执行推荐离线评估报告任务。
func (t *RecommendEvalReport) Exec(args map[string]string) ([]string, error) {
	log.Infof("Job RecommendEvalReport Exec %+v", args)

	statTime, err := parseStatDateArg(args["statDate"])
	if err != nil {
		return []string{err.Error()}, err
	}

	reportDate := time.Date(statTime.Year(), statTime.Month(), statTime.Day(), 0, 0, 0, 0, statTime.Location())
	startAt := reportDate
	endAt := reportDate.AddDate(0, 0, 1)

	var reportList []*models.RecommendEvalReport
	err = t.tx.Transaction(t.ctx, func(ctx context.Context) error {
		reportQuery := t.recommendEvalReportRepo.Query(ctx).RecommendEvalReport
		// 评估报告按天全量重算，先清掉当天旧数据再回写。
		opts := make([]repo.QueryOption, 0, 1)
		opts = append(opts, repo.Where(reportQuery.ReportDate.Eq(reportDate)))
		err = t.recommendEvalReportRepo.Delete(ctx, opts...)
		if err != nil {
			return err
		}

		sceneMetricMap := make(map[int32]*recommendEvalSceneMetric)
		ensureSceneMetric := func(scene int32) *recommendEvalSceneMetric {
			item, ok := sceneMetricMap[scene]
			// 首次命中的场景需要先初始化统计容器。
			if !ok {
				item = &recommendEvalSceneMetric{}
				sceneMetricMap[scene] = item
			}
			return item
		}

		requestQuery := t.recommendRequestRepo.Query(ctx).RecommendRequest
		opts = make([]repo.QueryOption, 0, 2)
		opts = append(opts, repo.Where(requestQuery.CreatedAt.Gte(startAt)))
		opts = append(opts, repo.Where(requestQuery.CreatedAt.Lt(endAt)))
		var requestList []*models.RecommendRequest
		requestList, err = t.recommendRequestRepo.List(ctx, opts...)
		if err != nil {
			return err
		}

		requestMetaByRecordId := make(map[int64]recommendEvalRequestMeta, len(requestList))
		requestMetaByRequestId := make(map[string]recommendEvalRequestMeta, len(requestList))
		requestRecordIds := make([]int64, 0, len(requestList))
		for _, item := range requestList {
			// 无效场景、主键或请求编号不参与评估。
			if item.Scene <= 0 || item.ID <= 0 || item.RequestID == "" {
				continue
			}
			ensureSceneMetric(item.Scene).requestCount++
			requestMeta := recommendEvalRequestMeta{
				scene:     item.Scene,
				requestId: item.RequestID,
			}
			requestMetaByRecordId[item.ID] = requestMeta
			requestMetaByRequestId[item.RequestID] = requestMeta
			requestRecordIds = append(requestRecordIds, item.ID)
		}

		requestItemsByRequestId := make(map[string][]recommendEvalRankedGoods, len(requestMetaByRequestId))
		// 只有存在推荐请求主表记录时，才继续回查推荐结果明细。
		if len(requestRecordIds) > 0 {
			requestItemQuery := t.recommendRequestItemRepo.Query(ctx).RecommendRequestItem
			opts = make([]repo.QueryOption, 0, 1)
			opts = append(opts, repo.Where(requestItemQuery.RecommendRequestID.In(requestRecordIds...)))
			var requestItemList []*models.RecommendRequestItem
			requestItemList, err = t.recommendRequestItemRepo.List(ctx, opts...)
			if err != nil {
				return err
			}
			for _, item := range requestItemList {
				requestMeta, ok := requestMetaByRecordId[item.RecommendRequestID]
				// 找不到请求主表或商品非法时，无法参与排序评估。
				if !ok || item.GoodsID <= 0 {
					continue
				}
				requestItemsByRequestId[requestMeta.requestId] = append(
					requestItemsByRequestId[requestMeta.requestId],
					recommendEvalRankedGoods{
						goodsId:  item.GoodsID,
						position: item.Position,
					},
				)
			}
		}

		exposureQuery := t.recommendExposureRepo.Query(ctx).RecommendExposure
		opts = make([]repo.QueryOption, 0, 2)
		opts = append(opts, repo.Where(exposureQuery.CreatedAt.Gte(startAt)))
		opts = append(opts, repo.Where(exposureQuery.CreatedAt.Lt(endAt)))
		var exposureList []*models.RecommendExposure
		exposureList, err = t.recommendExposureRepo.List(ctx, opts...)
		if err != nil {
			return err
		}

		exposureSceneMap := make(map[int64]int32, len(exposureList))
		exposureIds := make([]int64, 0, len(exposureList))
		for _, item := range exposureList {
			// 无效场景或曝光主表主键不参与逐商品曝光统计。
			if item.Scene <= 0 || item.ID <= 0 {
				continue
			}
			exposureSceneMap[item.ID] = item.Scene
			exposureIds = append(exposureIds, item.ID)
		}
		// 存在曝光主表时，再读取逐商品曝光明细累计曝光次数。
		if len(exposureIds) > 0 {
			exposureItemQuery := t.recommendExposureItemRepo.Query(ctx).RecommendExposureItem
			opts = make([]repo.QueryOption, 0, 1)
			opts = append(opts, repo.Where(exposureItemQuery.RecommendExposureID.In(exposureIds...)))
			var exposureItemList []*models.RecommendExposureItem
			exposureItemList, err = t.recommendExposureItemRepo.List(ctx, opts...)
			if err != nil {
				return err
			}
			for _, item := range exposureItemList {
				scene, ok := exposureSceneMap[item.RecommendExposureID]
				// 找不到曝光主表或商品非法时，直接跳过该曝光明细。
				if !ok || item.GoodsID <= 0 {
					continue
				}
				ensureSceneMetric(scene).exposureCount++
			}
		}

		actionQuery := t.recommendGoodsActionRepo.Query(ctx).RecommendGoodsAction
		opts = make([]repo.QueryOption, 0, 2)
		opts = append(opts, repo.Where(actionQuery.CreatedAt.Gte(startAt)))
		opts = append(opts, repo.Where(actionQuery.CreatedAt.Lt(endAt)))
		var actionList []*models.RecommendGoodsAction
		actionList, err = t.recommendGoodsActionRepo.List(ctx, opts...)
		if err != nil {
			return err
		}

		positiveGoodsByRequestId := make(map[string]map[int64]int32)
		for _, item := range actionList {
			// 无效场景的行为不参与场景级离线评估。
			if item.Scene > 0 {
				sceneMetric := ensureSceneMetric(item.Scene)
				eventType := common.RecommendGoodsActionType(item.EventType)
				// 场景级基础指标按点击、下单、支付三类事件累计。
				switch eventType {
				case common.RecommendGoodsActionType_CLICK:
					// 点击事件用于 CTR 与后续 CVR 计算。
					sceneMetric.clickCount++
				case common.RecommendGoodsActionType_ORDER_CREATE:
					// 下单事件用于统计推荐带来的下单量。
					sceneMetric.orderCount++
				case common.RecommendGoodsActionType_ORDER_PAY:
					// 支付事件用于统计最终成交量。
					sceneMetric.payCount++
				}
			}

			relevance, ok := recommendEvalActionRelevance(common.RecommendGoodsActionType(item.EventType))
			// 只有正向反馈事件才进入排序评估样本。
			if !ok {
				continue
			}
			// 没有关联请求编号的行为无法回放到具体推荐列表。
			if item.RequestID == "" {
				continue
			}
			requestMeta, ok := requestMetaByRequestId[item.RequestID]
			// 找不到对应推荐请求主表时，不参与离线评估。
			if !ok {
				continue
			}
			// 非法商品无法参与命中率与排序指标计算。
			if item.GoodsID <= 0 {
				continue
			}
			positiveGoodsMap, exists := positiveGoodsByRequestId[item.RequestID]
			// 当前请求首次命中正向反馈时，先初始化反馈集合。
			if !exists {
				positiveGoodsMap = make(map[int64]int32)
				positiveGoodsByRequestId[item.RequestID] = positiveGoodsMap
			}
			currentRelevance := positiveGoodsMap[item.GoodsID]
			// 同一请求商品出现多种行为时，保留更强的正反馈等级。
			if relevance > currentRelevance {
				positiveGoodsMap[item.GoodsID] = relevance
			}
			// 行为回放只使用请求主表的场景口径，避免主表与行为场景不一致。
			ensureSceneMetric(requestMeta.scene)
		}

		for requestId, positiveGoodsMap := range positiveGoodsByRequestId {
			requestMeta := requestMetaByRequestId[requestId]
			sceneMetric := ensureSceneMetric(requestMeta.scene)
			sceneMetric.positiveRequestCount++
			sceneMetric.positiveGoodsCount += int64(len(positiveGoodsMap))

			rankedGoodsList := dedupeRecommendEvalRankedGoods(requestItemsByRequestId[requestId])
			// 没有推荐结果明细时，无法计算 precision、recall 与 NDCG。
			if len(rankedGoodsList) == 0 {
				continue
			}

			hitCount, ndcg := calculateRecommendEvalRankingMetrics(rankedGoodsList, positiveGoodsMap)
			sceneMetric.evalRequestCount++
			sceneMetric.precisionSum += float64(hitCount) / float64(len(rankedGoodsList))
			sceneMetric.recallSum += float64(hitCount) / float64(len(positiveGoodsMap))
			sceneMetric.ndcgSum += ndcg
		}

		sceneList := make([]int32, 0, len(sceneMetricMap))
		for scene := range sceneMetricMap {
			// 未指定场景不生成评估报告，避免污染正式报表。
			if scene <= 0 {
				continue
			}
			sceneList = append(sceneList, scene)
		}
		sort.Slice(sceneList, func(i int, j int) bool {
			return sceneList[i] < sceneList[j]
		})

		strategyByScene := make(map[int32]*models.RecommendModelVersion, len(sceneList))
		// 命中场景后，再查询当前启用的模型版本作为策略标识。
		if len(sceneList) > 0 {
			modelQuery := t.recommendModelVersionRepo.Query(ctx).RecommendModelVersion
			opts = make([]repo.QueryOption, 0, 4)
			opts = append(opts, repo.Where(modelQuery.Scene.In(sceneList...)))
			opts = append(opts, repo.Where(modelQuery.Status.Eq(int32(common.Status_ENABLE))))
			opts = append(opts, repo.Order(modelQuery.Scene.Asc()))
			opts = append(opts, repo.Order(modelQuery.CreatedAt.Desc()))
			var modelVersionList []*models.RecommendModelVersion
			modelVersionList, err = t.recommendModelVersionRepo.List(ctx, opts...)
			if err != nil {
				return err
			}
			for _, item := range modelVersionList {
				_, exists := strategyByScene[item.Scene]
				// 每个场景只取最新一条启用版本作为评估策略名。
				if exists {
					continue
				}
				strategyByScene[item.Scene] = item
			}
		}

		reportList = make([]*models.RecommendEvalReport, 0, len(sceneList))
		for _, scene := range sceneList {
			sceneMetric := sceneMetricMap[scene]
			precisionScore := 0.0
			recallScore := 0.0
			ndcg := 0.0
			// 只有命中有效评估样本时，才计算排序指标均值。
			if sceneMetric.evalRequestCount > 0 {
				precisionScore = sceneMetric.precisionSum / float64(sceneMetric.evalRequestCount)
				recallScore = sceneMetric.recallSum / float64(sceneMetric.evalRequestCount)
				ndcg = sceneMetric.ndcgSum / float64(sceneMetric.evalRequestCount)
			}

			ctr := 0.0
			// CTR 口径固定为点击次数 / 逐商品曝光次数。
			if sceneMetric.exposureCount > 0 {
				ctr = float64(sceneMetric.clickCount) / float64(sceneMetric.exposureCount)
			}

			cvr := 0.0
			// CVR 口径固定为支付次数 / 点击次数。
			if sceneMetric.clickCount > 0 {
				cvr = float64(sceneMetric.payCount) / float64(sceneMetric.clickCount)
			}

			strategyName := buildRecommendEvalStrategyName(strategyByScene[scene])
			extraJSONData := map[string]interface{}{
				"scene_name":              recommendEvalSceneName(scene),
				"ctr_denominator":         "exposure_count",
				"cvr_denominator":         "click_count",
				"positive_request_count":  sceneMetric.positiveRequestCount,
				"positive_goods_count":    sceneMetric.positiveGoodsCount,
				"evaluated_request_count": sceneMetric.evalRequestCount,
			}
			modelVersion := strategyByScene[scene]
			// 当前场景存在启用模型版本时，把版本元信息落到扩展字段，便于回溯。
			if modelVersion != nil {
				extraJSONData["model_name"] = modelVersion.ModelName
				extraJSONData["model_type"] = modelVersion.ModelType
				extraJSONData["version"] = modelVersion.Version
			}
			extraJSONBytes, jsonErr := json.Marshal(extraJSONData)
			if jsonErr != nil {
				return jsonErr
			}

			reportList = append(reportList, &models.RecommendEvalReport{
				ReportDate:     reportDate,
				Scene:          scene,
				StrategyName:   strategyName,
				SampleSize:     sceneMetric.evalRequestCount,
				RequestCount:   sceneMetric.requestCount,
				ExposureCount:  sceneMetric.exposureCount,
				ClickCount:     sceneMetric.clickCount,
				OrderCount:     sceneMetric.orderCount,
				PayCount:       sceneMetric.payCount,
				Ctr:            ctr,
				Cvr:            cvr,
				Ndcg:           ndcg,
				PrecisionScore: precisionScore,
				RecallScore:    recallScore,
				ExtraJSON:      string(extraJSONBytes),
			})
		}

		// 没有任何场景数据时，只保留删除旧报告的动作即可。
		if len(reportList) == 0 {
			return nil
		}
		return t.recommendEvalReportRepo.BatchCreate(ctx, reportList)
	})
	if err != nil {
		return []string{err.Error()}, err
	}
	return []string{fmt.Sprintf("推荐离线评估报告生成完成: %s, 场景数 %d", reportDate.Format(time.DateOnly), len(reportList))}, nil
}

// recommendEvalActionRelevance 返回离线评估使用的正反馈等级。
func recommendEvalActionRelevance(eventType common.RecommendGoodsActionType) (int32, bool) {
	// 离线评估按行为强弱划分相关性等级，越靠近成交权重越高。
	switch eventType {
	case common.RecommendGoodsActionType_CLICK:
		// 点击说明推荐结果被用户初步接受。
		return 1, true
	case common.RecommendGoodsActionType_COLLECT, common.RecommendGoodsActionType_ADD_CART:
		// 收藏与加购说明用户存在更强购买意图。
		return 2, true
	case common.RecommendGoodsActionType_ORDER_CREATE:
		// 下单说明推荐结果已转化为明确订单意图。
		return 3, true
	case common.RecommendGoodsActionType_ORDER_PAY:
		// 支付是最强的成交正反馈。
		return 4, true
	default:
		// 其余事件当前不作为离线评估正样本。
		return 0, false
	}
}

// dedupeRecommendEvalRankedGoods 按推荐位序号去重同一请求下的商品列表。
func dedupeRecommendEvalRankedGoods(list []recommendEvalRankedGoods) []recommendEvalRankedGoods {
	// 空列表无需排序和去重，直接返回即可。
	if len(list) == 0 {
		return nil
	}
	sort.Slice(list, func(i int, j int) bool {
		// 推荐位序号相同时，再按商品编号稳定排序，避免去重结果抖动。
		if list[i].position == list[j].position {
			return list[i].goodsId < list[j].goodsId
		}
		return list[i].position < list[j].position
	})

	result := make([]recommendEvalRankedGoods, 0, len(list))
	seenGoodsMap := make(map[int64]struct{}, len(list))
	for _, item := range list {
		_, exists := seenGoodsMap[item.goodsId]
		// 同一请求里重复出现的商品只保留排名最前的一次。
		if exists {
			continue
		}
		seenGoodsMap[item.goodsId] = struct{}{}
		result = append(result, item)
	}
	return result
}

// calculateRecommendEvalRankingMetrics 计算单个请求的命中数与 NDCG。
func calculateRecommendEvalRankingMetrics(list []recommendEvalRankedGoods, positiveGoodsMap map[int64]int32) (int64, float64) {
	hitCount := int64(0)
	dcg := 0.0
	idealRelevanceList := make([]int32, 0, len(positiveGoodsMap))
	for _, relevance := range positiveGoodsMap {
		idealRelevanceList = append(idealRelevanceList, relevance)
	}
	sort.Slice(idealRelevanceList, func(i int, j int) bool {
		return idealRelevanceList[i] > idealRelevanceList[j]
	})

	for index, item := range list {
		relevance := positiveGoodsMap[item.goodsId]
		// 当前推荐位没有命中正反馈商品时，不参与 DCG 累计。
		if relevance <= 0 {
			continue
		}
		hitCount++
		dcg += recommendEvalGain(relevance) / math.Log2(float64(index+2))
	}

	idcg := 0.0
	limit := len(list)
	// 理想排序长度不能超过正反馈商品数量。
	if len(idealRelevanceList) < limit {
		limit = len(idealRelevanceList)
	}
	for index := 0; index < limit; index++ {
		idcg += recommendEvalGain(idealRelevanceList[index]) / math.Log2(float64(index+2))
	}
	// 没有理想增益时，当前请求的 NDCG 直接视为 0。
	if idcg == 0 {
		return hitCount, 0
	}
	return hitCount, dcg / idcg
}

// recommendEvalGain 将相关性等级转换为 NDCG 的增益值。
func recommendEvalGain(relevance int32) float64 {
	return math.Pow(2, float64(relevance)) - 1
}

// buildRecommendEvalStrategyName 生成评估报告使用的策略名称。
func buildRecommendEvalStrategyName(modelVersion *models.RecommendModelVersion) string {
	// 场景还没有版本台账时，统一落默认策略名。
	if modelVersion == nil {
		return "default"
	}
	// 版本号为空时，直接退化为模型名称。
	if modelVersion.Version == "" {
		return modelVersion.ModelName
	}
	return fmt.Sprintf("%s:%s", modelVersion.ModelName, modelVersion.Version)
}

// recommendEvalSceneName 返回推荐场景名称。
func recommendEvalSceneName(scene int32) string {
	sceneName, ok := common.RecommendScene_name[scene]
	// 未知场景直接回退为数值字符串，避免序列化空值。
	if !ok {
		return fmt.Sprintf("%d", scene)
	}
	return sceneName
}
