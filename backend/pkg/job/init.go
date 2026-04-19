package job

import (
	"shop/pkg/job/task"
	recommendCache "shop/pkg/recommend/cache"
	"shop/pkg/recommend/offline/materialize"

	"github.com/google/wire"
)

// ProviderSet 注册定时任务模块依赖。
var ProviderSet = wire.NewSet(
	NewCronServer,
	task.NewTradeBill,
	task.NewOrderStatDay,
	task.NewGoodsStatDay,
	task.NewRecommendGoodsStatDay,
	task.NewRecommendUserPreferenceRebuild,
	task.NewRecommendGoodsRelationRebuild,
	recommendCache.NewStore,
	materialize.NewMaterializer,
	task.NewRecommendHotMaterialize,
	task.NewRecommendLatestMaterialize,
	task.NewRecommendSimilarItemMaterialize,
	task.NewRecommendSimilarUserMaterialize,
	task.NewRecommendCollaborativeFilteringMaterialize,
	task.NewRecommendContentBasedMaterialize,
	task.NewRecommendRankerMaterialize,
	task.NewRecommendResultMaterialize,
	task.NewRecommendLlmRerankMaterialize,
	task.NewRecommendEvalReport,
	task.NewRecommendVersionPublish,
	task.NewTaskList,
)
