package configs

import (
	"os"
	"path/filepath"
	"shop/api/gen/go/conf"
	"shop/pkg/errorsx"
	"strconv"
	"time"

	bootstrapConf "github.com/liujitcn/kratos-kit/api/gen/go/conf"
	"github.com/liujitcn/kratos-kit/bootstrap"
	"github.com/liujitcn/kratos-kit/sdk"
)

const WrapperConfigKey = "Shop"

var payTimeoutMinutes = 30

const cacheKeyPayTimeout = "payTimeout"

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

func ParseGorse(cfg *conf.ShopConfig) (*conf.Gorse, error) {
	gorse := cfg.GetGorse()
	// 缺少推荐配置时，直接返回配置错误。
	if gorse == nil {
		return nil, errorsx.Internal("推荐配置信息错误")
	}
	return gorse, nil
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
