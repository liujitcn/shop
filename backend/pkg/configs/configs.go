package configs

import (
	"os"
	"path/filepath"
	"shop/api/gen/go/conf"
	"shop/pkg/errorsx"
	recommendCandidate "shop/pkg/recommend/candidate"
	recommendEvent "shop/pkg/recommend/event"
	recommendRank "shop/pkg/recommend/rank"
	"strconv"
	"time"

	bootstrapConf "github.com/liujitcn/kratos-kit/api/gen/go/conf"
	"github.com/liujitcn/kratos-kit/bootstrap"
	"github.com/liujitcn/kratos-kit/sdk"
)

const WrapperConfigKey = "Shop"

var payTimeoutMinutes = 30

const cacheKeyPayTimeout = "payTimeout"

const (
	defaultGoodsScoreViewWeight        = 1.0
	defaultGoodsScoreCollectWeight     = 3.0
	defaultGoodsScoreCartWeight        = 4.0
	defaultGoodsScoreOrderWeight       = 6.0
	defaultGoodsScorePayWeight         = 8.0
	defaultGoodsScorePayGoodsNumWeight = 1.0
	defaultGoodsScorePayAmountWeight   = 0.0001

	defaultPersonalizedRelationWeight             = 0.30
	defaultPersonalizedUserGoodsWeight            = 0.25
	defaultPersonalizedSimilarUserWeight          = 0.25
	defaultPersonalizedProfileWeight              = 0.15
	defaultPersonalizedScenePopularityWeight      = 0.20
	defaultPersonalizedGlobalPopularityWeight     = 0.10
	defaultPersonalizedFreshnessWeight            = 0.10
	defaultPersonalizedExposurePenaltyWeight      = 1.0
	defaultPersonalizedActorExposurePenaltyWeight = 1.0
	defaultPersonalizedRepeatPenaltyWeight        = 1.0

	defaultAnonymousRelationWeight             = 0.45
	defaultAnonymousScenePopularityWeight      = 0.30
	defaultAnonymousGlobalPopularityWeight     = 0.15
	defaultAnonymousFreshnessWeight            = 0.10
	defaultAnonymousExposurePenaltyWeight      = 1.0
	defaultAnonymousActorExposurePenaltyWeight = 1.0

	defaultRecommendEventClickWeight       = 3.0
	defaultRecommendEventViewWeight        = 2.0
	defaultRecommendEventCollectWeight     = 4.0
	defaultRecommendEventAddCartWeight     = 6.0
	defaultRecommendEventOrderCreateWeight = 8.0
	defaultRecommendEventOrderPayWeight    = 10.0

	defaultRecommendRelationClickWeight       = 3.0
	defaultRecommendRelationViewWeight        = 2.0
	defaultRecommendRelationOrderCreateWeight = 8.0
	defaultRecommendRelationOrderPayWeight    = 10.0

	defaultRecommendFreshnessWindowDays   = 30.0
	defaultRecommendHighExposureThreshold = int32(20)
	defaultRecommendNoClickPenalty        = 1.2
	defaultRecommendLowCtrPenalty         = 0.8
	defaultRecommendMediumCtrPenalty      = 0.4
	defaultRecommendLowCtrThreshold       = 0.005
	defaultRecommendMediumCtrThreshold    = 0.01
	defaultRecommendDayDecayFactor        = 0.08
	defaultRecommendAggregateWindowDays   = int32(30)

	defaultRecommendPoolMultiplier            = int32(8)
	defaultRecommendPoolMin                   = int32(80)
	defaultRecommendPoolMax                   = int32(240)
	defaultRecommendMaxPerCategory            = int32(2)
	defaultRecommendAnonymousRecallDays       = int32(30)
	defaultRecommendStatLookbackDays          = int32(30)
	defaultRecommendRecentPayPenaltyDays      = int32(15)
	defaultRecommendActorExposureLookbackDays = int32(7)

	defaultRecommendActorNoClickExposureThreshold = int32(3)
	defaultRecommendActorNoClickPenalty           = 0.6
	defaultRecommendActorLowCtrExposureThreshold  = int32(5)
	defaultRecommendActorLowCtrPenalty            = 0.3
	defaultRecommendActorLowCtrThreshold          = 0.05
)

// NewShopConfig 获取商城业务配置。
func NewShopConfig(ctx *bootstrap.Context) *conf.ShopConfig {
	cfg, ok := ctx.GetCustomConfig(WrapperConfigKey)
	// 自定义包装配置存在时，优先返回包装中的商城配置。
	if ok {
		wrapperCfg := cfg.(*conf.ShopConfigWrapper)
		return wrapperCfg.GetShop()
	}
	return &conf.ShopConfig{}
}

// ParseWxMiniApp 解析微信小程序配置。
func ParseWxMiniApp(cfg *conf.ShopConfig) (*conf.WxMiniApp, error) {
	wxMiniApp := cfg.GetWxMiniApp()
	// 缺少微信小程序配置时，直接返回配置错误。
	if wxMiniApp == nil {
		return nil, errorsx.Internal("微信登录配置信息错误")
	}
	appId := wxMiniApp.GetAppid()
	secret := wxMiniApp.GetSecret()
	// 小程序关键字段缺失时，视为配置不可用。
	if appId == "" || secret == "" {
		return nil, errorsx.Internal("微信登录配置信息错误")
	}
	return wxMiniApp, nil
}

// ParseWxPay 解析微信支付配置。
func ParseWxPay(cfg *conf.ShopConfig) (*conf.WxPay, error) {
	wxPay := cfg.GetWxPay()
	// 缺少微信支付配置时，直接返回配置错误。
	if wxPay == nil {
		return nil, errorsx.Internal("支付配置信息错误")
	}
	appId := wxPay.GetAppid()
	mchId := wxPay.GetMchId()
	mchCertSn := wxPay.GetMchCertSn()
	mchCertPath := wxPay.GetMchCertPath()
	mchApiV3Key := wxPay.GetMchAPIv3Key()
	// 微信支付关键字段缺失时，视为配置不可用。
	if appId == "" || mchId == "" || mchCertSn == "" || mchCertPath == "" || mchApiV3Key == "" {
		return nil, errorsx.Internal("支付配置信息错误")
	}
	// 兼容不同工作目录启动（GoLand/命令行）导致的相对路径差异。
	if resolvedPath, ok := resolveFilePath(mchCertPath); ok {
		wxPay.MchCertPath = resolvedPath
	}
	return wxPay, nil
}

// ParseGoodsRecommendConfig 解析商品推荐配置。
func ParseGoodsRecommendConfig(cfg *conf.ShopConfig) *conf.GoodsRecommendConfig {
	parsedConfig := &conf.GoodsRecommendConfig{
		GoodsStatScore: &conf.GoodsStatScoreConfig{
			ViewWeight:        float64Ptr(defaultGoodsScoreViewWeight),
			CollectWeight:     float64Ptr(defaultGoodsScoreCollectWeight),
			CartWeight:        float64Ptr(defaultGoodsScoreCartWeight),
			OrderWeight:       float64Ptr(defaultGoodsScoreOrderWeight),
			PayWeight:         float64Ptr(defaultGoodsScorePayWeight),
			PayGoodsNumWeight: float64Ptr(defaultGoodsScorePayGoodsNumWeight),
			PayAmountWeight:   float64Ptr(defaultGoodsScorePayAmountWeight),
		},
		PersonalizedRank: &conf.GoodsRecommendPersonalizedRankWeightConfig{
			RelationWeight:             float64Ptr(defaultPersonalizedRelationWeight),
			UserGoodsWeight:            float64Ptr(defaultPersonalizedUserGoodsWeight),
			SimilarUserWeight:          float64Ptr(defaultPersonalizedSimilarUserWeight),
			ProfileWeight:              float64Ptr(defaultPersonalizedProfileWeight),
			ScenePopularityWeight:      float64Ptr(defaultPersonalizedScenePopularityWeight),
			GlobalPopularityWeight:     float64Ptr(defaultPersonalizedGlobalPopularityWeight),
			FreshnessWeight:            float64Ptr(defaultPersonalizedFreshnessWeight),
			ExposurePenaltyWeight:      float64Ptr(defaultPersonalizedExposurePenaltyWeight),
			ActorExposurePenaltyWeight: float64Ptr(defaultPersonalizedActorExposurePenaltyWeight),
			RepeatPenaltyWeight:        float64Ptr(defaultPersonalizedRepeatPenaltyWeight),
		},
		AnonymousRank: &conf.GoodsRecommendAnonymousRankWeightConfig{
			RelationWeight:             float64Ptr(defaultAnonymousRelationWeight),
			ScenePopularityWeight:      float64Ptr(defaultAnonymousScenePopularityWeight),
			GlobalPopularityWeight:     float64Ptr(defaultAnonymousGlobalPopularityWeight),
			FreshnessWeight:            float64Ptr(defaultAnonymousFreshnessWeight),
			ExposurePenaltyWeight:      float64Ptr(defaultAnonymousExposurePenaltyWeight),
			ActorExposurePenaltyWeight: float64Ptr(defaultAnonymousActorExposurePenaltyWeight),
		},
		EventWeight: &conf.RecommendEventWeightConfig{
			ClickWeight:       float64Ptr(defaultRecommendEventClickWeight),
			ViewWeight:        float64Ptr(defaultRecommendEventViewWeight),
			CollectWeight:     float64Ptr(defaultRecommendEventCollectWeight),
			AddCartWeight:     float64Ptr(defaultRecommendEventAddCartWeight),
			OrderCreateWeight: float64Ptr(defaultRecommendEventOrderCreateWeight),
			OrderPayWeight:    float64Ptr(defaultRecommendEventOrderPayWeight),
		},
		RelationWeight: &conf.RecommendRelationWeightConfig{
			ClickWeight:       float64Ptr(defaultRecommendRelationClickWeight),
			ViewWeight:        float64Ptr(defaultRecommendRelationViewWeight),
			OrderCreateWeight: float64Ptr(defaultRecommendRelationOrderCreateWeight),
			OrderPayWeight:    float64Ptr(defaultRecommendRelationOrderPayWeight),
		},
		Rank: &conf.RecommendRankConfig{
			FreshnessWindowDays:   float64Ptr(defaultRecommendFreshnessWindowDays),
			HighExposureThreshold: int32Ptr(defaultRecommendHighExposureThreshold),
			NoClickPenalty:        float64Ptr(defaultRecommendNoClickPenalty),
			LowCtrPenalty:         float64Ptr(defaultRecommendLowCtrPenalty),
			MediumCtrPenalty:      float64Ptr(defaultRecommendMediumCtrPenalty),
			LowCtrThreshold:       float64Ptr(defaultRecommendLowCtrThreshold),
			MediumCtrThreshold:    float64Ptr(defaultRecommendMediumCtrThreshold),
			DayDecayFactor:        float64Ptr(defaultRecommendDayDecayFactor),
		},
		AggregateWindowDays: int32Ptr(defaultRecommendAggregateWindowDays),
		Recall: &conf.RecommendRecallConfig{
			PoolMultiplier:            int32Ptr(defaultRecommendPoolMultiplier),
			PoolMin:                   int32Ptr(defaultRecommendPoolMin),
			PoolMax:                   int32Ptr(defaultRecommendPoolMax),
			MaxPerCategory:            int32Ptr(defaultRecommendMaxPerCategory),
			AnonymousRecallDays:       int32Ptr(defaultRecommendAnonymousRecallDays),
			StatLookbackDays:          int32Ptr(defaultRecommendStatLookbackDays),
			RecentPayPenaltyDays:      int32Ptr(defaultRecommendRecentPayPenaltyDays),
			ActorExposureLookbackDays: int32Ptr(defaultRecommendActorExposureLookbackDays),
		},
		ActorExposurePenalty: &conf.RecommendActorExposurePenaltyConfig{
			NoClickExposureThreshold: int32Ptr(defaultRecommendActorNoClickExposureThreshold),
			NoClickPenalty:           float64Ptr(defaultRecommendActorNoClickPenalty),
			LowCtrExposureThreshold:  int32Ptr(defaultRecommendActorLowCtrExposureThreshold),
			LowCtrPenalty:            float64Ptr(defaultRecommendActorLowCtrPenalty),
			LowCtrThreshold:          float64Ptr(defaultRecommendActorLowCtrThreshold),
		},
	}
	// 未配置商品推荐配置时，直接回退默认值。
	if cfg == nil || cfg.GetRecommend() == nil {
		applyRecommendRuntimeConfig(parsedConfig)
		return parsedConfig
	}

	sourceConfig := cfg.GetRecommend()
	mergeGoodsStatScoreConfig(parsedConfig.GetGoodsStatScore(), sourceConfig.GetGoodsStatScore())
	mergePersonalizedRankWeightConfig(parsedConfig.GetPersonalizedRank(), sourceConfig.GetPersonalizedRank())
	mergeAnonymousRankWeightConfig(parsedConfig.GetAnonymousRank(), sourceConfig.GetAnonymousRank())
	mergeRecommendEventWeightConfig(parsedConfig.GetEventWeight(), sourceConfig.GetEventWeight())
	mergeRecommendRelationWeightConfig(parsedConfig.GetRelationWeight(), sourceConfig.GetRelationWeight())
	mergeRecommendRankConfig(parsedConfig.GetRank(), sourceConfig.GetRank())
	mergeRecommendRecallConfig(parsedConfig.GetRecall(), sourceConfig.GetRecall())
	mergeRecommendActorExposurePenaltyConfig(parsedConfig.GetActorExposurePenalty(), sourceConfig.GetActorExposurePenalty())
	if sourceConfig.AggregateWindowDays != nil {
		parsedConfig.AggregateWindowDays = sourceConfig.AggregateWindowDays
	}
	applyRecommendRuntimeConfig(parsedConfig)
	return parsedConfig
}

// mergeGoodsStatScoreConfig 合并商品热度分权重配置。
func mergeGoodsStatScoreConfig(target, source *conf.GoodsStatScoreConfig) {
	// 来源为空时，不需要覆盖默认商品热度分权重。
	if target == nil || source == nil {
		return
	}
	// 显式配置了浏览权重时，覆盖默认值；允许配置为 0。
	if source.ViewWeight != nil {
		target.ViewWeight = source.ViewWeight
	}
	// 显式配置了收藏权重时，覆盖默认值；允许配置为 0。
	if source.CollectWeight != nil {
		target.CollectWeight = source.CollectWeight
	}
	// 显式配置了加购权重时，覆盖默认值；允许配置为 0。
	if source.CartWeight != nil {
		target.CartWeight = source.CartWeight
	}
	// 显式配置了下单权重时，覆盖默认值；允许配置为 0。
	if source.OrderWeight != nil {
		target.OrderWeight = source.OrderWeight
	}
	// 显式配置了支付权重时，覆盖默认值；允许配置为 0。
	if source.PayWeight != nil {
		target.PayWeight = source.PayWeight
	}
	// 显式配置了支付件数权重时，覆盖默认值；允许配置为 0。
	if source.PayGoodsNumWeight != nil {
		target.PayGoodsNumWeight = source.PayGoodsNumWeight
	}
	// 显式配置了支付金额权重时，覆盖默认值；允许配置为 0。
	if source.PayAmountWeight != nil {
		target.PayAmountWeight = source.PayAmountWeight
	}
}

// mergePersonalizedRankWeightConfig 合并登录态排序权重配置。
func mergePersonalizedRankWeightConfig(target, source *conf.GoodsRecommendPersonalizedRankWeightConfig) {
	// 来源为空时，不需要覆盖默认登录态排序权重。
	if target == nil || source == nil {
		return
	}
	if source.RelationWeight != nil {
		target.RelationWeight = source.RelationWeight
	}
	if source.UserGoodsWeight != nil {
		target.UserGoodsWeight = source.UserGoodsWeight
	}
	if source.SimilarUserWeight != nil {
		target.SimilarUserWeight = source.SimilarUserWeight
	}
	if source.ProfileWeight != nil {
		target.ProfileWeight = source.ProfileWeight
	}
	if source.ScenePopularityWeight != nil {
		target.ScenePopularityWeight = source.ScenePopularityWeight
	}
	if source.GlobalPopularityWeight != nil {
		target.GlobalPopularityWeight = source.GlobalPopularityWeight
	}
	if source.FreshnessWeight != nil {
		target.FreshnessWeight = source.FreshnessWeight
	}
	if source.ExposurePenaltyWeight != nil {
		target.ExposurePenaltyWeight = source.ExposurePenaltyWeight
	}
	if source.ActorExposurePenaltyWeight != nil {
		target.ActorExposurePenaltyWeight = source.ActorExposurePenaltyWeight
	}
	if source.RepeatPenaltyWeight != nil {
		target.RepeatPenaltyWeight = source.RepeatPenaltyWeight
	}
}

// mergeAnonymousRankWeightConfig 合并匿名态排序权重配置。
func mergeAnonymousRankWeightConfig(target, source *conf.GoodsRecommendAnonymousRankWeightConfig) {
	// 来源为空时，不需要覆盖默认匿名态排序权重。
	if target == nil || source == nil {
		return
	}
	if source.RelationWeight != nil {
		target.RelationWeight = source.RelationWeight
	}
	if source.ScenePopularityWeight != nil {
		target.ScenePopularityWeight = source.ScenePopularityWeight
	}
	if source.GlobalPopularityWeight != nil {
		target.GlobalPopularityWeight = source.GlobalPopularityWeight
	}
	if source.FreshnessWeight != nil {
		target.FreshnessWeight = source.FreshnessWeight
	}
	if source.ExposurePenaltyWeight != nil {
		target.ExposurePenaltyWeight = source.ExposurePenaltyWeight
	}
	if source.ActorExposurePenaltyWeight != nil {
		target.ActorExposurePenaltyWeight = source.ActorExposurePenaltyWeight
	}
}

// mergeRecommendEventWeightConfig 合并用户偏好行为权重配置。
func mergeRecommendEventWeightConfig(target, source *conf.RecommendEventWeightConfig) {
	// 来源为空时，不需要覆盖默认用户偏好行为权重。
	if target == nil || source == nil {
		return
	}
	if source.ClickWeight != nil {
		target.ClickWeight = source.ClickWeight
	}
	if source.ViewWeight != nil {
		target.ViewWeight = source.ViewWeight
	}
	if source.CollectWeight != nil {
		target.CollectWeight = source.CollectWeight
	}
	if source.AddCartWeight != nil {
		target.AddCartWeight = source.AddCartWeight
	}
	if source.OrderCreateWeight != nil {
		target.OrderCreateWeight = source.OrderCreateWeight
	}
	if source.OrderPayWeight != nil {
		target.OrderPayWeight = source.OrderPayWeight
	}
}

// mergeRecommendRelationWeightConfig 合并商品关联行为权重配置。
func mergeRecommendRelationWeightConfig(target, source *conf.RecommendRelationWeightConfig) {
	// 来源为空时，不需要覆盖默认商品关联行为权重。
	if target == nil || source == nil {
		return
	}
	if source.ClickWeight != nil {
		target.ClickWeight = source.ClickWeight
	}
	if source.ViewWeight != nil {
		target.ViewWeight = source.ViewWeight
	}
	if source.OrderCreateWeight != nil {
		target.OrderCreateWeight = source.OrderCreateWeight
	}
	if source.OrderPayWeight != nil {
		target.OrderPayWeight = source.OrderPayWeight
	}
}

// mergeRecommendRankConfig 合并推荐排序参数配置。
func mergeRecommendRankConfig(target, source *conf.RecommendRankConfig) {
	// 来源为空时，不需要覆盖默认推荐排序参数。
	if target == nil || source == nil {
		return
	}
	if source.FreshnessWindowDays != nil {
		target.FreshnessWindowDays = source.FreshnessWindowDays
	}
	if source.HighExposureThreshold != nil {
		target.HighExposureThreshold = source.HighExposureThreshold
	}
	if source.NoClickPenalty != nil {
		target.NoClickPenalty = source.NoClickPenalty
	}
	if source.LowCtrPenalty != nil {
		target.LowCtrPenalty = source.LowCtrPenalty
	}
	if source.MediumCtrPenalty != nil {
		target.MediumCtrPenalty = source.MediumCtrPenalty
	}
	if source.LowCtrThreshold != nil {
		target.LowCtrThreshold = source.LowCtrThreshold
	}
	if source.MediumCtrThreshold != nil {
		target.MediumCtrThreshold = source.MediumCtrThreshold
	}
	if source.DayDecayFactor != nil {
		target.DayDecayFactor = source.DayDecayFactor
	}
}

// mergeRecommendRecallConfig 合并推荐召回参数配置。
func mergeRecommendRecallConfig(target, source *conf.RecommendRecallConfig) {
	// 来源为空时，不需要覆盖默认召回参数。
	if target == nil || source == nil {
		return
	}
	if source.PoolMultiplier != nil {
		target.PoolMultiplier = source.PoolMultiplier
	}
	if source.PoolMin != nil {
		target.PoolMin = source.PoolMin
	}
	if source.PoolMax != nil {
		target.PoolMax = source.PoolMax
	}
	if source.MaxPerCategory != nil {
		target.MaxPerCategory = source.MaxPerCategory
	}
	if source.AnonymousRecallDays != nil {
		target.AnonymousRecallDays = source.AnonymousRecallDays
	}
	if source.StatLookbackDays != nil {
		target.StatLookbackDays = source.StatLookbackDays
	}
	if source.RecentPayPenaltyDays != nil {
		target.RecentPayPenaltyDays = source.RecentPayPenaltyDays
	}
	if source.ActorExposureLookbackDays != nil {
		target.ActorExposureLookbackDays = source.ActorExposureLookbackDays
	}
}

// mergeRecommendActorExposurePenaltyConfig 合并主体曝光惩罚配置。
func mergeRecommendActorExposurePenaltyConfig(target, source *conf.RecommendActorExposurePenaltyConfig) {
	// 来源为空时，不需要覆盖默认主体曝光惩罚配置。
	if target == nil || source == nil {
		return
	}
	if source.NoClickExposureThreshold != nil {
		target.NoClickExposureThreshold = source.NoClickExposureThreshold
	}
	if source.NoClickPenalty != nil {
		target.NoClickPenalty = source.NoClickPenalty
	}
	if source.LowCtrExposureThreshold != nil {
		target.LowCtrExposureThreshold = source.LowCtrExposureThreshold
	}
	if source.LowCtrPenalty != nil {
		target.LowCtrPenalty = source.LowCtrPenalty
	}
	if source.LowCtrThreshold != nil {
		target.LowCtrThreshold = source.LowCtrThreshold
	}
}

// applyRecommendRuntimeConfig 同步推荐运行时配置。
func applyRecommendRuntimeConfig(cfg *conf.GoodsRecommendConfig) {
	// 运行时推荐配置缺失时，不做任何同步。
	if cfg == nil {
		return
	}
	recommendCandidate.ApplyRecommendConfig(cfg)
	recommendEvent.ApplyRecommendConfig(cfg)
	recommendRank.ApplyRecommendConfig(cfg)
}

// float64Ptr 返回 float64 指针，便于构造 optional 配置字段。
func float64Ptr(value float64) *float64 {
	return &value
}

// int32Ptr 返回 int32 指针，便于构造 optional 配置字段。
func int32Ptr(value int32) *int32 {
	return &value
}

// resolveFilePath 解析配置中的证书文件路径。
func resolveFilePath(path string) (string, bool) {
	// 绝对路径存在时直接返回原路径。
	if filepath.IsAbs(path) {
		// 绝对路径对应文件存在时，直接返回原路径。
		if _, statErr := os.Stat(path); statErr == nil {
			return path, true
		}
		return path, false
	}

	candidates := []string{
		path,
		filepath.Join("server", path),
		filepath.Join("..", path),
		filepath.Join("..", "..", path),
		filepath.Join("..", "..", "..", path),
		filepath.Join("..", "server", path),
		filepath.Join(filepath.Dir(os.Args[0]), "..", path),
		filepath.Join(filepath.Dir(os.Args[0]), "..", "..", path),
	}

	for _, p := range candidates {
		cleaned := filepath.Clean(p)
		// 命中可用文件后，立即返回标准化路径。
		if _, statErr := os.Stat(cleaned); statErr == nil {
			return cleaned, true
		}
	}
	return path, false
}

// ParsePayTimeout 解析支付超时时间。
func ParsePayTimeout() time.Duration {
	cache := sdk.Runtime.GetCache()
	// 未启用缓存时，回退到默认支付超时时间。
	if cache == nil {
		return time.Duration(payTimeoutMinutes) * time.Minute
	}

	cacheValue, err := cache.Get(cacheKeyPayTimeout)
	if err != nil {
		return time.Duration(payTimeoutMinutes) * time.Minute
	}
	var parsedPayTimeoutMinutes int
	parsedPayTimeoutMinutes, err = strconv.Atoi(cacheValue)
	if err != nil {
		return time.Duration(payTimeoutMinutes) * time.Minute
	}
	payTimeoutMinutes = parsedPayTimeoutMinutes
	return time.Duration(payTimeoutMinutes) * time.Minute
}

// ParseOss 解析对象存储配置。
func ParseOss(ctx *bootstrap.Context) (*bootstrapConf.OSS, error) {
	cfg := ctx.GetConfig()
	// 对象存储配置缺失时，直接返回错误。
	if cfg == nil || cfg.GetOss() == nil {
		return nil, errorsx.Internal("对象存储配置缺失")
	}
	return cfg.GetOss(), nil
}

// ParseData 解析数据源配置。
func ParseData(ctx *bootstrap.Context) (*bootstrapConf.Data, error) {
	cfg := ctx.GetConfig()
	// 数据源配置缺失时，直接返回错误。
	if cfg == nil || cfg.GetData() == nil {
		return nil, errorsx.Internal("数据源配置缺失")
	}
	return cfg.GetData(), nil
}

// ParseDatabase 解析数据库配置。
func ParseDatabase(cfg *bootstrapConf.Data) *bootstrapConf.Data_Database {
	return cfg.GetDatabase()
}

// ParseQueue 解析队列配置。
func ParseQueue(cfg *bootstrapConf.Data) *bootstrapConf.Data_Queue {
	return cfg.GetQueue()
}

// ParseRedis 解析 Redis 配置。
func ParseRedis(cfg *bootstrapConf.Data) *bootstrapConf.Data_Redis {
	return cfg.GetRedis()
}

// ParsePprof 解析性能分析配置。
func ParsePprof(ctx *bootstrap.Context) (*bootstrapConf.Pprof, error) {
	cfg := ctx.GetConfig()
	// 性能分析配置缺失时，直接返回错误。
	if cfg == nil || cfg.GetPprof() == nil {
		return nil, errorsx.Internal("性能分析配置缺失")
	}
	return cfg.GetPprof(), nil
}

// ParseAuthnJwt 解析 JWT 认证配置。
func ParseAuthnJwt(ctx *bootstrap.Context) *bootstrapConf.Authentication_Jwt {
	cfg := ctx.GetConfig()
	// 未配置 JWT 时，回退到项目默认认证参数。
	if cfg == nil || cfg.GetAuthn() == nil || cfg.GetAuthn().GetJwt() == nil {
		return &bootstrapConf.Authentication_Jwt{
			Method: "HS256",
			Secret: "shop-base",
		}
	}
	return cfg.GetAuthn().GetJwt()
}
