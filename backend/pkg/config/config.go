package config

import (
	"os"
	"path/filepath"
	configv1 "shop/api/gen/go/config/v1"
	"shop/pkg/errorsx"
	"strconv"
	"time"

	bootstrapConfigv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	"github.com/liujitcn/kratos-kit/bootstrap"
	"github.com/liujitcn/kratos-kit/sdk"
)

const WRAPPER_CONFIG_KEY = "Shop"

var payTimeoutMinutes = 30

const CACHE_KEY_PAY_TIMEOUT = "payTimeout"

// NewShopConfig 获取商城业务配置。
func NewShopConfig(ctx *bootstrap.Context) *configv1.ShopConfig {
	cfg, ok := ctx.GetCustomConfig(WRAPPER_CONFIG_KEY)
	// 自定义包装配置存在时，优先返回包装中的商城配置。
	if ok {
		wrapperCfg := cfg.(*configv1.ShopConfigWrapper)
		return wrapperCfg.GetShop()
	}
	return &configv1.ShopConfig{}
}

// ParseWxMiniApp 解析微信小程序配置。
func ParseWxMiniApp(cfg *configv1.ShopConfig) (*configv1.WxMiniApp, error) {
	wxMiniApp := cfg.GetWxMiniApp()
	// 缺少微信小程序配置时，直接返回配置错误。
	if wxMiniApp == nil {
		return nil, errorsx.Internal("微信登录配置信息错误")
	}
	appID := wxMiniApp.GetAppid()
	secret := wxMiniApp.GetSecret()
	// 小程序关键字段缺失时，视为配置不可用。
	if appID == "" || secret == "" {
		return nil, errorsx.Internal("微信登录配置信息错误")
	}
	return wxMiniApp, nil
}

// ParseWxPay 解析微信支付配置。
func ParseWxPay(cfg *configv1.ShopConfig) (*configv1.WxPay, error) {
	wxPay := cfg.GetWxPay()
	// 缺少微信支付配置时，直接返回配置错误。
	if wxPay == nil {
		return nil, errorsx.Internal("支付配置信息错误")
	}
	appID := wxPay.GetAppid()
	mchID := wxPay.GetMchId()
	mchCertSn := wxPay.GetMchCertSn()
	mchCertPath := wxPay.GetMchCertPath()
	mchAPIV3Key := wxPay.GetMchApiV3Key()
	// 微信支付关键字段缺失时，视为配置不可用。
	if appID == "" || mchID == "" || mchCertSn == "" || mchCertPath == "" || mchAPIV3Key == "" {
		return nil, errorsx.Internal("支付配置信息错误")
	}
	// 兼容不同工作目录启动（GoLand/命令行）导致的相对路径差异。
	if resolvedPath, ok := resolveFilePath(mchCertPath); ok {
		wxPay.MchCertPath = resolvedPath
	}
	return wxPay, nil
}

// ParseRecommend 解析推荐配置。
func ParseRecommend(cfg *configv1.ShopConfig) (*configv1.Recommend, error) {
	// 商城配置缺失时，回退到空推荐配置，后续由业务侧自动走本地兜底。
	if cfg == nil {
		return &configv1.Recommend{}, nil
	}
	recommend := cfg.GetRecommend()
	// 缺少推荐配置时，回退到空推荐配置，避免因未启用推荐系统导致服务启动失败。
	if recommend == nil {
		return &configv1.Recommend{}, nil
	}
	return recommend, nil
}

// ParseLLM 解析大模型客户端配置。
func ParseLLM(ctx *bootstrap.Context) *bootstrapConfigv1.Client_Llm {
	cfg := ctx.GetConfig()
	// 未配置客户端大模型参数时，返回空配置并由客户端保持关闭状态。
	if cfg == nil || cfg.GetClient() == nil || cfg.GetClient().GetLlm() == nil {
		return &bootstrapConfigv1.Client_Llm{}
	}
	return cfg.GetClient().GetLlm()
}

// ParsePrompt 解析提示词配置。
func ParsePrompt(cfg *configv1.ShopConfig) *configv1.Prompt {
	// 商城配置缺失时，返回空提示词配置，由调用方判断是否可用。
	if cfg == nil {
		return &configv1.Prompt{}
	}
	prompt := cfg.GetPrompt()
	// 缺少提示词配置时，返回空提示词配置，由调用方判断是否可用。
	if prompt == nil {
		return &configv1.Prompt{}
	}
	return prompt
}

// ParsePayTimeout 解析支付超时时间。
func ParsePayTimeout() time.Duration {
	cache := sdk.Runtime.GetCache()
	// 未启用缓存时，回退到默认支付超时时间。
	if cache == nil {
		return time.Duration(payTimeoutMinutes) * time.Minute
	}

	cacheValue, err := cache.Get(CACHE_KEY_PAY_TIMEOUT)
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

// ParseOSS 解析对象存储配置。
func ParseOSS(ctx *bootstrap.Context) (*bootstrapConfigv1.Oss, error) {
	cfg := ctx.GetConfig()
	// 对象存储配置缺失时，直接返回错误。
	if cfg == nil || cfg.GetOss() == nil {
		return nil, errorsx.Internal("对象存储配置缺失")
	}
	return cfg.GetOss(), nil
}

// ParseData 解析数据源配置。
func ParseData(ctx *bootstrap.Context) (*bootstrapConfigv1.Data, error) {
	cfg := ctx.GetConfig()
	// 数据源配置缺失时，直接返回错误。
	if cfg == nil || cfg.GetData() == nil {
		return nil, errorsx.Internal("数据源配置缺失")
	}
	return cfg.GetData(), nil
}

// ParseDatabase 解析数据库配置。
func ParseDatabase(cfg *bootstrapConfigv1.Data) *bootstrapConfigv1.Data_Database {
	return cfg.GetDatabase()
}

// ParseQueue 解析队列配置。
func ParseQueue(cfg *bootstrapConfigv1.Data) *bootstrapConfigv1.Data_Queue {
	return cfg.GetQueue()
}

// ParseRedis 解析 Redis 配置。
func ParseRedis(cfg *bootstrapConfigv1.Data) *bootstrapConfigv1.Data_Redis {
	return cfg.GetRedis()
}

// ParsePprof 解析性能分析配置。
func ParsePprof(ctx *bootstrap.Context) (*bootstrapConfigv1.Pprof, error) {
	cfg := ctx.GetConfig()
	// 性能分析配置缺失时，直接返回错误。
	if cfg == nil || cfg.GetPprof() == nil {
		return nil, errorsx.Internal("性能分析配置缺失")
	}
	return cfg.GetPprof(), nil
}

// ParseAuthnJWT 解析 JWT 认证配置。
func ParseAuthnJWT(ctx *bootstrap.Context) *bootstrapConfigv1.Authentication_Jwt {
	cfg := ctx.GetConfig()
	// 未配置 JWT 时，回退到项目默认认证参数。
	if cfg == nil || cfg.GetAuthn() == nil || cfg.GetAuthn().GetJwt() == nil {
		return &bootstrapConfigv1.Authentication_Jwt{
			Method: "HS256",
			Secret: "shop-base",
		}
	}
	return cfg.GetAuthn().GetJwt()
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
